package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/streadway/amqp"

	"sd_hw4/payments/internal/repositories"
)

type PaymentService struct {
	billRepo    *repositories.BillRepository
	outboxRepo  *repositories.OutboxRepository
	amqpChannel *amqp.Channel
}

func NewPaymentService(
	billRepo *repositories.BillRepository,
	outboxRepo *repositories.OutboxRepository,
	amqpChannel *amqp.Channel,
) *PaymentService {
	return &PaymentService{
		billRepo:    billRepo,
		outboxRepo:  outboxRepo,
		amqpChannel: amqpChannel,
	}
}

type CreateBillRequest struct {
	UserID string `json:"user_id"`
}

type CreateBillResponse struct {
	BillID string `json:"bill_id"`
	UserID string `json:"user_id"`
	Status string `json:"status"`
}

type AddBalanceRequest struct {
	Amount float64 `json:"amount"`
}

type AddBalanceResponse struct {
	BillID  string  `json:"bill_id"`
	UserID  string  `json:"user_id"`
	Balance float64 `json:"balance"`
}

type BalanceResponse struct {
	BillID  string  `json:"bill_id"`
	UserID  string  `json:"user_id"`
	Balance float64 `json:"balance"`
}

func (s *PaymentService) CreateBill(ctx context.Context, userID string) (*CreateBillResponse, error) {
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id format: %w", err)
	}

	bill := &repositories.Bill{
		UserID:   parsedUserID,
		Balance:  0.0,
		Currency: "RUB",
		Status:   repositories.BillStatusActive,
	}

	if err := s.billRepo.Create(ctx, bill); err != nil {
		return nil, fmt.Errorf("failed to create bill: %w", err)
	}

	if err := s.publishBillCreatedEvent(ctx, bill); err != nil {
		return nil, fmt.Errorf("failed to publish event: %w", err)
	}

	return &CreateBillResponse{
		BillID: bill.ID.String(),
		UserID: bill.UserID.String(),
		Status: string(bill.Status),
	}, nil
}

func (s *PaymentService) AddBalance(ctx context.Context, billID string, userID string, amount float64) (*AddBalanceResponse, error) {

	parsedBillID, err := uuid.Parse(billID)
	if err != nil {
		return nil, fmt.Errorf("invalid bill_id format: %w", err)
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id format: %w", err)
	}

	bill, err := s.billRepo.GetByID(ctx, parsedBillID)
	if err != nil {
		return nil, fmt.Errorf("bill not found: %w", err)
	}

	if bill.UserID != parsedUserID {
		return nil, fmt.Errorf("bill does not belong to user")
	}

	if bill.Status != repositories.BillStatusActive {
		return nil, fmt.Errorf("bill is not active")
	}

	if (bill.Balance + amount) < 0 {
		return nil, fmt.Errorf("insufficient funds")
	}

	bill.Balance += amount

	if err := s.billRepo.Update(ctx, bill); err != nil {
		return nil, fmt.Errorf("failed to update bill: %w", err)
	}

	if err := s.publishBalanceAddedEvent(ctx, bill, amount); err != nil {
		return nil, fmt.Errorf("failed to publish event: %w", err)
	}

	return &AddBalanceResponse{
		BillID:  bill.ID.String(),
		UserID:  bill.UserID.String(),
		Balance: bill.Balance,
	}, nil
}

func (s *PaymentService) GetBalance(ctx context.Context, billID string, userID string) (*BalanceResponse, error) {
	parsedBillID, err := uuid.Parse(billID)
	if err != nil {
		return nil, fmt.Errorf("invalid bill_id format: %w", err)
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id format: %w", err)
	}

	bill, err := s.billRepo.GetByID(ctx, parsedBillID)
	if err != nil {
		return nil, fmt.Errorf("bill not found: %w", err)
	}

	if bill.UserID != parsedUserID {
		return nil, fmt.Errorf("bill does not belong to user")
	}

	return &BalanceResponse{
		BillID:  bill.ID.String(),
		UserID:  bill.UserID.String(),
		Balance: bill.Balance,
	}, nil
}

func (s *PaymentService) publishBillCreatedEvent(ctx context.Context, bill *repositories.Bill) error {
	eventPayload := map[string]interface{}{
		"bill_id":  bill.ID.String(),
		"user_id":  bill.UserID.String(),
		"status":   bill.Status,
		"currency": bill.Currency,
	}

	payload, err := json.Marshal(eventPayload)
	if err != nil {
		return err
	}

	msg := &repositories.OutboxMessage{
		MessageID:  uuid.New().String(),
		Exchange:   "payments",
		RoutingKey: "bill.created",
		Payload:    payload,
		Status:     repositories.StatusPending,
	}

	return s.outboxRepo.Create(ctx, msg)
}

func (s *PaymentService) publishBalanceAddedEvent(ctx context.Context, bill *repositories.Bill, amount float64) error {
	eventPayload := map[string]interface{}{
		"bill_id": bill.ID.String(),
		"user_id": bill.UserID.String(),
		"amount":  amount,
		"balance": bill.Balance,
	}

	payload, err := json.Marshal(eventPayload)
	if err != nil {
		return err
	}

	msg := &repositories.OutboxMessage{
		MessageID:  uuid.New().String(),
		Exchange:   "payments",
		RoutingKey: "bill.balance.added",
		Payload:    payload,
		Status:     repositories.StatusPending,
	}

	return s.outboxRepo.Create(ctx, msg)
}
