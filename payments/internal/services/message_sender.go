package services

import (
	"context"
	"log"
	"time"
)

type MessageSender struct {
	messageService MessageService
}

func NewMessageSender(messageService MessageService) *MessageSender {
	return &MessageSender{
		messageService: messageService,
	}
}

func (s *MessageSender) Start(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.processOutboxMessages(ctx)
		}
	}
}

func (s *MessageSender) processOutboxMessages(ctx context.Context) {
	// Получаем pending сообщения из outbox
	messages, err := s.messageService.(*messageService).outboxRepo.GetPending(ctx, 10)
	if err != nil {
		log.Printf("Error getting pending outbox messages: %v", err)
		return
	}

	for _, msg := range messages {
		if err := s.messageService.SendOutboxMessage(ctx, msg); err != nil {
			log.Printf("Error sending outbox message %s: %v", msg.MessageID, err)

			// Помечаем как failed
			if err := s.messageService.(*messageService).outboxRepo.MarkAsFailed(ctx, msg.ID, err.Error()); err != nil {
				log.Printf("Error marking message as failed: %v", err)
			}
		} else {
			// Помечаем как отправленное
			if err := s.messageService.(*messageService).outboxRepo.MarkAsSent(ctx, msg.ID); err != nil {
				log.Printf("Error marking message as sent: %v", err)
			}
			log.Printf("Message sent successfully: %s", msg.MessageID)
		}
	}
}
