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

	orders "sd_hw4/orders/internal/gen"
	handlers "sd_hw4/orders/internal/handlers"
	"sd_hw4/orders/internal/repositories"
	services "sd_hw4/orders/internal/service"
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
	logger.Println(cfg)

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
	factory := messaging.NewFactory()
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

	// Конфигурация publisher для запросов оплаты
	paymentRequestPub := queueManager.GetOrCreatePublisher(
		"payment_request",
		factory.PaymentRequestPublisher(),
	)

	// Инициализация репозиториев
	orderRepo := repositories.NewOrderRepository(db.DB)
	inboxRepo := repositories.NewInboxRepo(db.DB)
	outboxRepo := repositories.NewOutboxRepository(db.DB)

	// Инициализация сервисов
	orderService := services.NewOrderService(orderRepo, outboxRepo, paymentRequestPub)
	outboxService := services.NewOutboxService(outboxRepo, paymentRequestPub, 10)
	inboxService := services.NewInboxService(inboxRepo, orderService, 10, "orders")

	// Инициализация обработчика сообщений
	consumerHandler := handlers.NewConsumerHandler(inboxService)

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

	// Инициализация хендлеров
	orderHandler := handlers.NewOrderHandler(orderService)

	// Регистрация маршрутов из OpenAPI
	orders.RegisterHandlers(e, orderHandler)

	// Контекст для фоновых задач
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запуск фоновых процессов
	go outboxService.StartProcessor(ctx)
	go inboxService.StartProcessor(ctx)

	// Запуск консьюмера RabbitMQ
	go func() {
		if err := consumerHandler.StartConsumer(ctx, mqConn); err != nil {
			log.Printf("Consumer stopped with error: %v", err)
		}
	}()
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
