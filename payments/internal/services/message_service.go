// services/message_service.go
package services

import (
	"context"
	"encoding/json"

	"sd_hw4/payments/internal/repositories"

	amqp "github.com/rabbitmq/amqp091-go"
)

type MessageService interface {
	SaveOutboxMessage(ctx context.Context, msg *repositories.OutboxMessage) error
	SendOutboxMessage(ctx context.Context, msg repositories.OutboxMessage) error
	GetUnprocessedMessages(ctx context.Context, queue string, limit int) ([]repositories.InboxMessage, error)
	MarkMessageProcessed(ctx context.Context, messageID string) error
	SaveInboxMessage(ctx context.Context, messageID, queue string, payload json.RawMessage) error
}

type messageService struct {
	inboxRepo  *repositories.InboxRepo
	outboxRepo *repositories.OutboxRepository
	rabbitConn *amqp.Connection
}

func NewMessageService(inboxRepo *repositories.InboxRepo, outboxRepo *repositories.OutboxRepository, rabbitConn *amqp.Connection) MessageService {
	return &messageService{
		inboxRepo:  inboxRepo,
		outboxRepo: outboxRepo,
		rabbitConn: rabbitConn,
	}
}

func (s *messageService) SaveOutboxMessage(ctx context.Context, msg *repositories.OutboxMessage) error {
	return s.outboxRepo.Create(ctx, msg)
}

func (s *messageService) SendOutboxMessage(ctx context.Context, msg repositories.OutboxMessage) error {
	ch, err := s.rabbitConn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	err = ch.Publish(
		msg.Exchange,
		msg.RoutingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        msg.Payload,
			MessageId:   msg.MessageID,
		},
	)
	return err
}

func (s *messageService) GetUnprocessedMessages(ctx context.Context, queue string, limit int) ([]repositories.InboxMessage, error) {
	return s.inboxRepo.GetUnprocessed(ctx, queue, limit)
}

func (s *messageService) MarkMessageProcessed(ctx context.Context, messageID string) error {
	return s.inboxRepo.MarkProcessed(ctx, messageID)
}

func (s *messageService) SaveInboxMessage(ctx context.Context, messageID, queue string, payload json.RawMessage) error {
	return s.inboxRepo.Save(ctx, messageID, queue, payload)
}
