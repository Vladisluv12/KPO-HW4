package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	OrderStatusNew      OrderStatus = "new"
	OrderStatusFinished OrderStatus = "finished"
	OrderStatusCanceled OrderStatus = "canceled"
)

type Order struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Price       float64
	Description string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type PaymentRequest struct {
	OrderID     uuid.UUID
	UserID      uuid.UUID
	Price       float64
	Description string
}

type PaymentResult struct {
	OrderID uuid.UUID
	Success bool
	Reason  string
}

type OutboxMessage struct {
	ID         uuid.UUID       `db:"id"`
	MessageID  string          `db:"message_id"`
	Exchange   string          `db:"exchange"`
	RoutingKey string          `db:"routing_key"`
	Payload    json.RawMessage `db:"payload"`
	Headers    json.RawMessage `db:"headers"`
	Status     OutboxStatus    `db:"status"`
	CreatedAt  time.Time       `db:"created_at"`
	SentAt     *time.Time      `db:"sent_at"`
	Error      *string         `db:"error"`
	RetryCount int             `db:"retry_count"`
}

type OutboxStatus string

const (
	StatusPending OutboxStatus = "pending"
	StatusSent    OutboxStatus = "sent"
	StatusFailed  OutboxStatus = "failed"
)

type InboxMessage struct {
	ID          uuid.UUID       `db:"id"`
	MessageID   string          `db:"message_id"`
	Queue       string          `db:"queue"`
	Payload     json.RawMessage `db:"payload"`
	Processed   bool            `db:"processed"`
	ProcessedAt *time.Time      `db:"processed_at"`
	CreatedAt   time.Time       `db:"created_at"`
}
