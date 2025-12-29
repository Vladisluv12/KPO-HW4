package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	models "sd_hw4/orders/internal/models"
	"sd_hw4/orders/internal/repositories"
	"sd_hw4/pkg/messaging"

	"github.com/google/uuid"
)

type OrderService struct {
	orderRepo  repositories.OrderRepository
	outboxRepo repositories.OutboxRepository
	publisher  *messaging.Publisher
}

func NewOrderService(orderRepo *repositories.OrderRepository, outboxRepo *repositories.OutboxRepository, publisher *messaging.Publisher) *OrderService {
	return &OrderService{
		orderRepo:  *orderRepo,
		outboxRepo: *outboxRepo,
		publisher:  publisher,
	}
}

// CreateOrder создает новый заказ и сообщение для оплаты
func (s *OrderService) CreateOrder(ctx context.Context, userID uuid.UUID, amount float64, description string) (*models.Order, error) {
	// Создаем заказ
	order := &models.Order{
		ID:          uuid.New(),
		UserID:      userID,
		Price:       amount,
		Description: description,
		Status:      string(models.OrderStatusNew),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Сохраняем заказ в БД
	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to save order: %w", err)
	}

	// Создаем сообщение для оплаты
	paymentRequest := models.PaymentRequest{
		OrderID:     order.ID,
		UserID:      userID,
		Price:       amount,
		Description: description,
	}

	payload, err := json.Marshal(paymentRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payment request: %w", err)
	}

	// Сохраняем в outbox
	outboxMsg := &models.OutboxMessage{
		ID:         uuid.New(),
		MessageID:  uuid.New().String(),
		Exchange:   "payments",
		RoutingKey: "payment.request",
		Payload:    payload,
		Status:     models.StatusPending,
		CreatedAt:  time.Now(),
		RetryCount: 0,
	}

	if err := s.outboxRepo.Create(ctx, outboxMsg); err != nil {
		return nil, fmt.Errorf("failed to save outbox message: %w", err)
	}

	return order, nil
}

// GetOrdersByUser возвращает заказы пользователя
func (s *OrderService) GetOrdersByUser(ctx context.Context, userID uuid.UUID) ([]models.Order, error) {
	return s.orderRepo.GetByUserID(ctx, userID)
}

// GetOrderByID возвращает заказ по ID
func (s *OrderService) GetOrderByID(ctx context.Context, orderID uuid.UUID) (*models.Order, error) {
	return s.orderRepo.GetByID(ctx, orderID)
}

// UpdateOrderStatus обновляет статус заказа
func (s *OrderService) UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, status models.OrderStatus) error {
	return s.orderRepo.UpdateStatus(ctx, orderID, string(status))
}

// ProcessPaymentResult обрабатывает результат оплаты
func (s *OrderService) ProcessPaymentResult(ctx context.Context, paymentResult models.PaymentResult) error {
	var status models.OrderStatus
	if paymentResult.Success {
		status = models.OrderStatusFinished
	} else {
		status = models.OrderStatusCanceled
	}

	return s.UpdateOrderStatus(ctx, paymentResult.OrderID, status)
}
