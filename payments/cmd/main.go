// main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	payments "sd_hw4/payments/internal/gen"
	handlers "sd_hw4/payments/internal/handler"
	"sd_hw4/payments/internal/repositories"
	"sd_hw4/payments/internal/services"
	"sd_hw4/pkg/config"
	"sd_hw4/pkg/db"
	"sd_hw4/pkg/messaging"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

func main() {
	// Инициализация логгера
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)

	// Загрузка конфигурации
	cfg := config.Load()

	// Инициализация базы данных
	logger.Println("Connecting to database...")
	if err := db.Connect(cfg.DatabaseURL); err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	logger.Println("Database connected successfully")
	// Выполнение миграций
	if err := db.Migrate(cfg.MigrationsDir); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Инициализация RabbitMQ
	mqConfig := messaging.ConnectionConfig{
		URL:            cfg.RabbitMQURL,
		MaxReconnects:  10,
		ReconnectDelay: 5 * time.Second,
	}

	mqConn := messaging.NewConnection(mqConfig)
	if err := mqConn.Connect(); err != nil {
		logger.Fatal("Failed to connect to RabbitMQ:", err)
	}
	defer mqConn.Close()

	// Инициализация менеджера очередей
	queueManager := messaging.NewQueueManager(mqConn)

	// Инициализация репозиториев
	billRepo := repositories.NewBillRepository(db.DB)
	inboxRepo := repositories.NewInboxRepo(db.DB)
	outboxRepo := repositories.NewOutboxRepository(db.DB)

	// Инициализация сервисов
	billService := services.NewBillService(billRepo)
	messageService := services.NewMessageService(inboxRepo, outboxRepo, mqConn.Conn)
	paymentService := services.NewPaymentService(billService, messageService)

	orderConsumer := handlers.NewOrderConsumerHandler(
		queueManager,
		logger,
		"orders.payment_requests",
		"payments.payment_results",
	)

	// Запуск обработчика входящих сообщений
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Настраиваем очереди
	if err := orderConsumer.SetupQueue(ctx); err != nil {
		logger.Fatal("Failed to setup queues:", err)
	}

	// Запускаем consumer для orders
	if err := orderConsumer.StartConsumer(ctx, mqConn, messageService); err != nil {
		logger.Fatal("Failed to start consumer:", err)
	}

	// Инициализация и запуск payment processor
	paymentProcessor := services.NewPaymentProcessor(
		paymentService,
		messageService,
	)
	go paymentProcessor.ProcessMessages(ctx)

	// Инициализация и запуск message sender
	messageSender := services.NewMessageSender(messageService)
	go messageSender.Start(ctx)

	// Создание Echo сервера
	e := echo.New()

	e.HideBanner = true

	// Middleware
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:     true,
		LogStatus:  true,
		LogMethod:  true,
		LogLatency: true,
		LogError:   true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			logger.WithFields(logrus.Fields{
				"uri":     v.URI,
				"method":  v.Method,
				"status":  v.Status,
				"latency": v.Latency,
				"error":   v.Error,
			}).Info("HTTP request")
			return nil
		},
	}))

	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
	}))

	handler := handlers.NewHandler(billService, paymentService)
	payments.RegisterHandlers(e, handler)

	// Health check endpoint
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	if err := e.Start(":" + cfg.ServerPort); err != nil && err != http.ErrServerClosed {
		logger.Fatal("Failed to start HTTP server:", err)
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Ожидание сигнала завершения
	<-quit
	logger.Info("Shutting down server...")

	// Graceful shutdown с таймаутом
	ctx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Остановка HTTP сервера
	if err := e.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown error:", err)
	}

	// Остановка обработчиков сообщений
	cancel()
	queueManager.StopAllConsumers()

	// Откат миграций при необходимости
	db.Rollback(cfg.MigrationsDir)

	time.Sleep(2 * time.Second)
	logger.Info("Server stopped gracefully")
}
