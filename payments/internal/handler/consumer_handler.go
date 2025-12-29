// handlers/order_consumer_handler.go
package handlers

import (
	"context"
	"encoding/json"
	"time"

	"sd_hw4/payments/internal/services"
	"sd_hw4/pkg/messaging"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

// OrderConsumerHandler обрабатывает сообщения о заказах
type OrderConsumerHandler struct {
	queueManager *messaging.QueueManager
	logger       *logrus.Logger
	orderQueue   string
	paymentQueue string
}

// NewOrderConsumerHandler создает новый обработчик
func NewOrderConsumerHandler(
	queueManager *messaging.QueueManager,
	logger *logrus.Logger,
	orderQueue string,
	paymentQueue string,
) *OrderConsumerHandler {
	return &OrderConsumerHandler{
		queueManager: queueManager,
		logger:       logger,
		orderQueue:   orderQueue,
		paymentQueue: paymentQueue,
	}
}

// SetupQueue настраивает очереди orders и payments
func (h *OrderConsumerHandler) SetupQueue(ctx context.Context) error {
	h.logger.Info("Setting up RabbitMQ queues")

	// Настраиваем очередь для приема заказов (orders)
	orderPublisher := h.queueManager.GetOrCreatePublisher("orders", messaging.PublisherConfig{
		Exchange:   "orders",
		RoutingKey: h.orderQueue,
		Mandatory:  false,
		Immediate:  false,
	})

	// Создаем exchange и очередь для orders
	if err := orderPublisher.DeclareExchange("orders", "direct", true); err != nil {
		return err
	}
	if err := orderPublisher.DeclareQueue(h.orderQueue, true, false); err != nil {
		return err
	}
	if err := orderPublisher.BindQueue(h.orderQueue, h.orderQueue); err != nil {
		return err
	}

	// Настраиваем очередь для отправки результатов платежей (payments)
	paymentPublisher := h.queueManager.GetOrCreatePublisher("payments", messaging.PublisherConfig{
		Exchange:   "payments",
		RoutingKey: h.paymentQueue,
		Mandatory:  false,
		Immediate:  false,
	})

	// Создаем exchange и очередь для payments
	if err := paymentPublisher.DeclareExchange("payments", "direct", true); err != nil {
		return err
	}
	if err := paymentPublisher.DeclareQueue(h.paymentQueue, true, false); err != nil {
		return err
	}
	if err := paymentPublisher.BindQueue(h.paymentQueue, h.paymentQueue); err != nil {
		return err
	}

	h.logger.WithFields(logrus.Fields{
		"order_queue":   h.orderQueue,
		"payment_queue": h.paymentQueue,
	}).Info("Queues setup completed")

	return nil
}

// StartConsumer запускает потребителя для очереди orders
func (h *OrderConsumerHandler) StartConsumer(ctx context.Context, conn *messaging.Connection, messageService services.MessageService) error {
	h.logger.WithField("queue", h.orderQueue).Info("Starting order consumer")

	consumer := messaging.NewConsumer(
		conn,
		messaging.ConsumerConfig{
			QueueName:     h.orderQueue,
			ConsumerTag:   "payments-service",
			AutoAck:       false, // Важно для transactional inbox
			Exclusive:     false,
			NoLocal:       false,
			NoWait:        false,
			PrefetchCount: 10,
		},
		h.HandleOrderRequest(messageService),
	)

	h.queueManager.RegisterConsumer("order_consumer", consumer)

	// Запускаем в отдельной горутине
	go func() {
		if err := consumer.Start(ctx); err != nil {
			h.logger.WithError(err).Error("Order consumer stopped")
		}
	}()

	return nil
}

// HandleOrderRequest возвращает обработчик для сообщений о заказах
func (h *OrderConsumerHandler) HandleOrderRequest(messageService services.MessageService) messaging.MessageHandler {
	return func(ctx context.Context, delivery amqp.Delivery) error {
		h.logger.WithField("message_id", delivery.MessageId).Debug("Processing order request")

		// 1. Валидируем сообщение
		if delivery.MessageId == "" {
			h.logger.Warn("Received message without ID, rejecting")
			delivery.Nack(false, false) // не requeue
			return nil
		}

		// 3. Сохраняем в inbox (Transactional Inbox - часть 1)
		if err := messageService.SaveInboxMessage(
			ctx,
			delivery.MessageId,
			h.orderQueue,
			delivery.Body,
		); err != nil {
			h.logger.WithError(err).Error("Failed to save message to inbox")
			delivery.Nack(false, true) // requeue - transient ошибка
			return err
		}

		// 4. Ack в RabbitMQ
		if err := delivery.Ack(false); err != nil {
			h.logger.WithError(err).Error("Failed to ack message")
			return err
		}

		return nil
	}
}

// SendPaymentResult отправляет результат платежа в очередь payments
func (h *OrderConsumerHandler) SendPaymentResult(ctx context.Context, result services.PaymentResult) error {
	publisher := h.queueManager.GetOrCreatePublisher("payments", messaging.PublisherConfig{
		Exchange:   "payments",
		RoutingKey: h.paymentQueue,
		Mandatory:  false,
		Immediate:  false,
	})

	payload, err := json.Marshal(result)
	if err != nil {
		h.logger.WithError(err).Error("Failed to marshal payment result")
		return err
	}

	// Добавляем timestamp, если нет
	if result.Timestamp == "" {
		result.Timestamp = time.Now().Format(time.RFC3339)
		payload, _ = json.Marshal(result)
	}

	if err := publisher.Publish(ctx, payload); err != nil {
		h.logger.WithError(err).WithField("order_id", result.OrderID).Error("Failed to send payment result")
		return err
	}

	h.logger.WithFields(logrus.Fields{
		"order_id": result.OrderID,
		"status":   result.Status,
		"queue":    h.paymentQueue,
	}).Info("Payment result sent")

	return nil
}
