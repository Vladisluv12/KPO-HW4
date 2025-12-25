package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type OutboxStatus string

const (
	StatusPending OutboxStatus = "pending"
	StatusSent    OutboxStatus = "sent"
	StatusFailed  OutboxStatus = "failed"
)

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

type OutboxRepository struct {
	db *sql.DB
}

func NewOutboxRepository(db *sql.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

func (r *OutboxRepository) Create(ctx context.Context, msg *OutboxMessage) error {
	msg.ID = uuid.New()
	msg.CreatedAt = time.Now()

	query := `INSERT INTO outbox_messages 
		(id, message_id, exchange, routing_key, payload, headers, status, created_at, retry_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.db.ExecContext(ctx, query,
		msg.ID, msg.MessageID, msg.Exchange, msg.RoutingKey,
		msg.Payload, msg.Headers, msg.Status, msg.CreatedAt, msg.RetryCount)
	return err
}

func (r *OutboxRepository) GetPending(ctx context.Context, limit int) ([]OutboxMessage, error) {
	query := `SELECT id, message_id, exchange, routing_key, payload, headers, status, created_at, sent_at, error, retry_count
		FROM outbox_messages WHERE status = $1 ORDER BY created_at ASC LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, StatusPending, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []OutboxMessage
	for rows.Next() {
		var msg OutboxMessage
		err := rows.Scan(&msg.ID, &msg.MessageID, &msg.Exchange, &msg.RoutingKey,
			&msg.Payload, &msg.Headers, &msg.Status, &msg.CreatedAt, &msg.SentAt, &msg.Error, &msg.RetryCount)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

func (r *OutboxRepository) MarkAsSent(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE outbox_messages SET status = $1, sent_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, StatusSent, time.Now(), id)
	return err
}

func (r *OutboxRepository) MarkAsFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	query := `UPDATE outbox_messages SET status = $1, error = $2, retry_count = retry_count + 1 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, StatusFailed, errMsg, id)
	return err
}

func (r *OutboxRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM outbox_messages WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
