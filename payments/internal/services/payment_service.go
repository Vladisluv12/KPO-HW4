package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"sd_hw4/payments/internal/repositories"

	"github.com/google/uuid"
)

type PaymentRequest struct {
	OrderID   string  `json:"order_id"`
	UserID    string  `json:"user_id"`
	Amount    float64 `json:"amount"`
	Timestamp string  `json:"timestamp"`
}

type PaymentResult struct {
	OrderID   string `json:"order_id"`
	UserID    string `json:"user_id"`
	Status    string `json:"status"` // "success" или "failed"
	Reason    string `json:"reason,omitempty"`
	Timestamp string `json:"timestamp"`
}

type PaymentService interface {
	ProcessPayment(ctx context.Context, request PaymentRequest) (*PaymentResult, error)
}

type paymentService struct {
	billService    BillService
	messageService MessageService
}

func NewPaymentService(billService BillService, messageService MessageService) PaymentService {
	return &paymentService{
		billService:    billService,
		messageService: messageService,
	}
}

func (s *paymentService) ProcessPayment(ctx context.Context, request PaymentRequest) (*PaymentResult, error) {
	result := &PaymentResult{
		OrderID:   request.OrderID,
		UserID:    request.UserID,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Получаем счет пользователя
	bills, err := s.billService.GetBillsByUserID(ctx, request.UserID)
	if err != nil {
		result.Status = "failed"
		result.Reason = "error fetching bills"
		return result, err
	}

	if len(bills) == 0 {
		result.Status = "failed"
		result.Reason = "no bill found for user"
		return result, nil
	}

	// Используем первый активный счет
	var activeBill *repositories.Bill
	for _, bill := range bills {
		if bill.Status == "active" {
			activeBill = bill
			break
		}
	}

	if activeBill == nil {
		result.Status = "failed"
		result.Reason = "no active bill found"
		return result, nil
	}

	// Проверяем достаточно ли средств
	if activeBill.Balance < request.Amount {
		result.Status = "failed"
		result.Reason = "insufficient funds"
		return result, nil
	}

	// Списание средств
	activeBill.Balance -= request.Amount
	err = s.billService.UpdateBill(ctx, activeBill)
	if err != nil {
		result.Status = "failed"
		result.Reason = "failed to update balance"
		return result, err
	}

	result.Status = "success"

	// Сохраняем задачу на отправку результата в outbox
	payload, _ := json.Marshal(result)
	outboxMsg := &repositories.OutboxMessage{
		MessageID:  uuid.New().String(),
		Exchange:   "",
		RoutingKey: "payments.result",
		Payload:    payload,
		Status:     repositories.StatusPending,
		RetryCount: 0,
	}

	err = s.messageService.SaveOutboxMessage(ctx, outboxMsg)
	if err != nil {
		// Логируем ошибку, но не возвращаем ее, так как платеж уже выполнен
		fmt.Printf("Failed to save outbox message: %v\n", err)
	}

	return result, nil
}
