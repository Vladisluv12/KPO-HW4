package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	models "sd_hw4/orders/internal/models"
	"time"

	"github.com/google/uuid"
)

type InboxRepo struct {
	db *sql.DB
}

func NewInboxRepo(db *sql.DB) *InboxRepo {
	return &InboxRepo{db: db}
}

func (r *InboxRepo) Save(ctx context.Context, messageID, queue string, payload json.RawMessage) error {
	query := `
		INSERT INTO inbox_messages (id, message_id, queue, payload, processed, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (message_id) DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, query, uuid.New(), messageID, queue, payload, false, time.Now())
	return err
}

func (r *InboxRepo) GetUnprocessed(ctx context.Context, queue string, limit int) ([]models.InboxMessage, error) {
	query := `
		SELECT id, message_id, queue, payload, processed, processed_at, created_at
		FROM inbox_messages
		WHERE queue = $1 AND processed = false
		ORDER BY created_at ASC
		LIMIT $2
	`
	rows, err := r.db.QueryContext(ctx, query, queue, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.InboxMessage
	for rows.Next() {
		var msg models.InboxMessage
		if err := rows.Scan(&msg.ID, &msg.MessageID, &msg.Queue, &msg.Payload, &msg.Processed, &msg.ProcessedAt, &msg.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

func (r *InboxRepo) MarkProcessed(ctx context.Context, messageID string) error {
	query := `
		UPDATE inbox_messages
		SET processed = true, processed_at = $1
		WHERE message_id = $2
	`
	_, err := r.db.ExecContext(ctx, query, time.Now(), messageID)
	return err
}
