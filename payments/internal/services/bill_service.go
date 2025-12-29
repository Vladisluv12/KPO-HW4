package services

import (
	"context"
	"fmt"

	"sd_hw4/payments/internal/repositories"

	"github.com/google/uuid"
)

type BillService interface {
	CreateBill(ctx context.Context, userID string) (*repositories.Bill, error)
	GetBill(ctx context.Context, billID string) (*repositories.Bill, error)
	GetBillsByUserID(ctx context.Context, userID string) ([]*repositories.Bill, error)
	UpdateBalance(ctx context.Context, billID, userID string, amount float64) (*repositories.Bill, error)
	UpdateBill(ctx context.Context, bill *repositories.Bill) error
}

type billService struct {
	billRepo repositories.BillRepository
}

func NewBillService(billRepo *repositories.BillRepository) BillService {
	return &billService{billRepo: *billRepo}
}

func (s *billService) CreateBill(ctx context.Context, userID string) (*repositories.Bill, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	bill := &repositories.Bill{
		UserID:   userUUID,
		Balance:  0,
		Currency: "RUB",
		Status:   repositories.BillStatusActive,
	}

	err = s.billRepo.Create(ctx, bill)
	if err != nil {
		return nil, err
	}

	return bill, nil
}

func (s *billService) GetBill(ctx context.Context, billID string) (*repositories.Bill, error) {
	billUUID, err := uuid.Parse(billID)
	if err != nil {
		return nil, err
	}

	return s.billRepo.GetByID(ctx, billUUID)
}

func (s *billService) GetBillsByUserID(ctx context.Context, userID string) ([]*repositories.Bill, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	return s.billRepo.GetByUserID(ctx, userUUID)
}

func (s *billService) UpdateBalance(ctx context.Context, billID, userID string, amount float64) (*repositories.Bill, error) {
	billUUID, err := uuid.Parse(billID)
	if err != nil {
		return nil, err
	}

	bill, err := s.billRepo.GetByID(ctx, billUUID)
	if err != nil {
		return nil, err
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil || bill.UserID != userUUID {
		return nil, fmt.Errorf("bill does not belong to user")
	}

	bill.Balance += amount
	err = s.billRepo.Update(ctx, bill)
	if err != nil {
		return nil, err
	}

	return bill, nil
}

func (s *billService) UpdateBill(ctx context.Context, bill *repositories.Bill) error {
	return s.billRepo.Update(ctx, bill)
}
