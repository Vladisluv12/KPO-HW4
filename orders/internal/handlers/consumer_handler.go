package handlers

import (
	"context"
	"log"

	services "sd_hw4/orders/internal/service"
	"sd_hw4/pkg/messaging"

	amqp "github.com/rabbitmq/amqp091-go"
)

type ConsumerHandler struct {
	inboxService *services.InboxService
}

func NewConsumerHandler(inboxService *services.InboxService) *ConsumerHandler {
	return &ConsumerHandler{
		inboxService: inboxService,
	}
}

// HandlePaymentResult обрабатывает сообщения о результате оплаты
func (h *ConsumerHandler) HandlePaymentResult(ctx context.Context, delivery amqp.Delivery) error {
	log.Printf("Received payment result message: %s", delivery.MessageId)

	// Сохраняем сообщение в inbox
	err := h.inboxService.SaveInboxMessage(
		ctx,
		delivery.MessageId,
		"orders", // queue name для orders service
		delivery.Body,
	)
	if err != nil {
		log.Printf("Failed to save inbox message: %v", err)
		return err
	}

	log.Printf("Payment result message saved to inbox: %s", delivery.MessageId)
	return nil
}

// StartConsumer запускает консьюмер для сообщений о результате оплаты
func (h *ConsumerHandler) StartConsumer(ctx context.Context, conn *messaging.Connection) error {
	config := messaging.ConsumerConfig{
		QueueName:     "payments.payment_results",
		ConsumerTag:   "orders-service",
		AutoAck:       false,
		Exclusive:     false,
		NoLocal:       false,
		NoWait:        false,
		PrefetchCount: 10,
	}

	consumer := messaging.NewConsumer(conn, config, h.HandlePaymentResult)

	// Создаем очередь и биндинг
	if err := h.setupQueue(conn); err != nil {
		return err
	}

	return consumer.Start(ctx)
}

func (h *ConsumerHandler) setupQueue(conn *messaging.Connection) error {
	ch, err := conn.Channel()
	if err != nil {
		return err
	}

	// Объявляем exchange для orders
	err = ch.ExchangeDeclare(
		"orders",
		"direct",
		true,  // durable
		false, // autoDelete
		false, // internal
		false, // noWait
		nil,
	)
	if err != nil {
		return err
	}

	// Объявляем очередь для результатов оплаты
	_, err = ch.QueueDeclare(
		"payments.payment_results",
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,
	)
	if err != nil {
		return err
	}

	// Биндим очередь к exchange
	err = ch.QueueBind(
		"payments.payment_results",
		"payment.result",
		"payments",
		false,
		nil,
	)
	return err
}
