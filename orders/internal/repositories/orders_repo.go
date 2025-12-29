package repositories

import (
	"context"
	"database/sql"
	"time"

	models "sd_hw4/orders/internal/models"
	"sd_hw4/pkg/db"

	"github.com/google/uuid"
)

type OrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// Create создает новый заказ
func (r *OrderRepository) Create(ctx context.Context, order *models.Order) error {
	order.ID = uuid.New()
	order.CreatedAt = time.Now()

	query := `
        INSERT INTO orders (id, user_id, price, description, status, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `

	_, err := db.Exec(ctx, query,
		order.ID,
		order.UserID,
		order.Price,
		order.Description,
		order.Status,
		order.CreatedAt,
		order.UpdatedAt,
	)

	return err
}

// GetByID возвращает заказ по ID
func (r *OrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Order, error) {
	query := `
        SELECT id, user_id, price, description, status, created_at, updated_at
        FROM orders
        WHERE id = $1
    `

	var order models.Order
	err := db.QueryRow(ctx, query, id).Scan(
		&order.ID,
		&order.UserID,
		&order.Price,
		&order.Description,
		&order.Status,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &order, nil
}

// GetByUserID возвращает заказы пользователя
func (r *OrderRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]models.Order, error) {
	query := `
        SELECT id, user_id, price, description, status, created_at, updated_at
        FROM orders
        WHERE user_id = $1
        ORDER BY created_at DESC
    `

	rows, err := db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.Price,
			&order.Description,
			&order.Status,
			&order.CreatedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	return orders, nil
}

// UpdateStatus обновляет статус заказа
func (r *OrderRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `
        UPDATE orders
        SET status = $1, updated_at = $2
        WHERE id = $3
    `

	_, err := db.Exec(ctx, query, status, time.Now(), id)
	return err
}

// GetByStatus возвращает заказы по статусу
func (r *OrderRepository) GetByStatus(ctx context.Context, status string, limit, offset int) ([]models.Order, error) {
	query := `
        SELECT id, user_id, price, description, status, created_at, updated_at
        FROM orders
        WHERE status = $1
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3
    `

	rows, err := db.Query(ctx, query, status, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.Price,
			&order.Description,
			&order.Status,
			&order.CreatedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	return orders, nil
}

// GetByUserAndStatus возвращает заказы пользователя по статусу
func (r *OrderRepository) GetByUserAndStatus(ctx context.Context, userID uuid.UUID, status string) ([]models.Order, error) {
	query := `
        SELECT id, user_id, price, description, status, created_at, updated_at
        FROM orders
        WHERE user_id = $1 AND status = $2
        ORDER BY created_at DESC
    `

	rows, err := db.Query(ctx, query, userID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.Price,
			&order.Description,
			&order.Status,
			&order.CreatedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	return orders, nil
}

// Delete удаляет заказ
func (r *OrderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM orders WHERE id = $1`
	_, err := db.Exec(ctx, query, id)
	return err
}
