package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type BillStatus string

const (
	BillStatusActive   BillStatus = "active"
	BillStatusClosed   BillStatus = "closed"
	BillStatusSuspended BillStatus = "suspended"
)

type Bill struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Balance   float64
	Currency  string
	Status    BillStatus
	CreatedAt time.Time
	UpdatedAt *time.Time
	ClosedAt  *time.Time
}

type BillRepository struct {
	db *sql.DB
}

func NewBillRepository(db *sql.DB) *BillRepository {
	return &BillRepository{db: db}
}

func (r *BillRepository) Create(ctx context.Context, bill *Bill) error {
	bill.ID = uuid.New()
	bill.CreatedAt = time.Now()

	query := `INSERT INTO bills (id, user_id, balance, currency, status, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.db.ExecContext(ctx, query, bill.ID, bill.UserID, bill.Balance, bill.Currency, bill.Status, bill.CreatedAt)
	return err
}

func (r *BillRepository) GetByID(ctx context.Context, id uuid.UUID) (*Bill, error) {
	bill := &Bill{}
	query := `SELECT id, user_id, balance, currency, status, created_at, updated_at, closed_at
			 FROM bills WHERE id = $1`

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&bill.ID, &bill.UserID, &bill.Balance, &bill.Currency, &bill.Status, &bill.CreatedAt, &bill.UpdatedAt, &bill.ClosedAt,
	)
	return bill, err
}

func (r *BillRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*Bill, error) {
	query := `SELECT id, user_id, balance, currency, status, created_at, updated_at, closed_at
			 FROM bills WHERE user_id = $1`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bills []*Bill
	for rows.Next() {
		bill := &Bill{}
		err := rows.Scan(&bill.ID, &bill.UserID, &bill.Balance, &bill.Currency, &bill.Status, &bill.CreatedAt, &bill.UpdatedAt, &bill.ClosedAt)
		if err != nil {
			return nil, err
		}
		bills = append(bills, bill)
	}
	return bills, rows.Err()
}

func (r *BillRepository) Update(ctx context.Context, bill *Bill) error {
	now := time.Now()
	bill.UpdatedAt = &now

	query := `UPDATE bills SET balance = $1, status = $2, updated_at = $3 WHERE id = $4`
	_, err := r.db.ExecContext(ctx, query, bill.Balance, bill.Status, bill.UpdatedAt, bill.ID)
	return err
}

func (r *BillRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM bills WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}