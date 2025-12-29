package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	models "sd_hw4/orders/internal/models"
	"sd_hw4/orders/internal/repositories"
)

type InboxService struct {
	inboxRepo repositories.InboxRepo
	orderSvc  *OrderService
	batchSize int
	queueName string
}

func NewInboxService(
	inboxRepo *repositories.InboxRepo,
	orderSvc *OrderService,
	batchSize int,
	queueName string,
) *InboxService {
	return &InboxService{
		inboxRepo: *inboxRepo,
		orderSvc:  orderSvc,
		batchSize: batchSize,
		queueName: queueName,
	}
}

// StartProcessor запускает фоновый процессор inbox
func (s *InboxService) StartProcessor(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Inbox processor stopped")
			return
		case <-ticker.C:
			s.processMessages(ctx)
		}
	}
}

// processMessages обрабатывает необработанные сообщения
func (s *InboxService) processMessages(ctx context.Context) {
	messages, err := s.inboxRepo.GetUnprocessed(ctx, s.queueName, s.batchSize)
	if err != nil {
		log.Printf("Failed to get unprocessed inbox messages: %v", err)
		return
	}

	for _, msg := range messages {
		if err := s.processMessage(ctx, msg); err != nil {
			log.Printf("Failed to process inbox message %s: %v", msg.MessageID, err)
			continue
		}

		// Помечаем сообщение как обработанное
		if err := s.inboxRepo.MarkProcessed(ctx, msg.MessageID); err != nil {
			log.Printf("Failed to mark message as processed: %v", err)
		}
	}
}

// processMessage обрабатывает одно сообщение
func (s *InboxService) processMessage(ctx context.Context, msg models.InboxMessage) error {
	var paymentResult models.PaymentResult
	if err := json.Unmarshal(msg.Payload, &paymentResult); err != nil {
		return fmt.Errorf("failed to unmarshal payment result: %w", err)
	}

	// Обрабатываем результат оплаты через сервис заказов
	return s.orderSvc.ProcessPaymentResult(ctx, paymentResult)
}

// SaveInboxMessage сохраняет входящее сообщение
func (s *InboxService) SaveInboxMessage(ctx context.Context, messageID, queue string, payload []byte) error {
	return s.inboxRepo.Save(ctx, messageID, queue, payload)
}
