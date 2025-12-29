package services

import (
	"context"
	"fmt"
	"log"
	"time"

	models "sd_hw4/orders/internal/models"
	"sd_hw4/orders/internal/repositories"
	"sd_hw4/pkg/messaging"
)

type OutboxService struct {
	outboxRepo repositories.OutboxRepository
	publisher  *messaging.Publisher
	batchSize  int
}

func NewOutboxService(
	outboxRepo *repositories.OutboxRepository,
	publisher *messaging.Publisher,
	batchSize int,
) *OutboxService {
	return &OutboxService{
		outboxRepo: *outboxRepo,
		publisher:  publisher,
		batchSize:  batchSize,
	}
}

// StartProcessor запускает фоновый процессор outbox
func (s *OutboxService) StartProcessor(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Outbox processor stopped")
			return
		case <-ticker.C:
			s.processPendingMessages(ctx)
		}
	}
}

// processPendingMessages обрабатывает ожидающие сообщения
func (s *OutboxService) processPendingMessages(ctx context.Context) {
	messages, err := s.outboxRepo.GetPending(ctx, s.batchSize)
	if err != nil {
		log.Printf("Failed to get pending outbox messages: %v", err)
		return
	}

	for _, msg := range messages {
		if err := s.sendMessage(ctx, msg); err != nil {
			log.Printf("Failed to send outbox message %s: %v", msg.ID, err)
			continue
		}
	}
}

// sendMessage отправляет сообщение через RabbitMQ
func (s *OutboxService) sendMessage(ctx context.Context, msg models.OutboxMessage) error {
	// Отправляем сообщение
	err := s.publisher.PublishRaw(ctx, msg.Payload, nil)
	if err != nil {
		// Помечаем как неудачное
		if markErr := s.outboxRepo.MarkAsFailed(ctx, msg.ID, err.Error()); markErr != nil {
			log.Printf("Failed to mark message as failed: %v", markErr)
		}
		return fmt.Errorf("failed to publish message: %w", err)
	}

	// Помечаем как отправленное
	if err := s.outboxRepo.MarkAsSent(ctx, msg.ID); err != nil {
		return fmt.Errorf("failed to mark message as sent: %w", err)
	}

	return nil
}
