package handlers

import (
	"net/http"

	payments "sd_hw4/payments/internal/gen"
	"sd_hw4/payments/internal/services"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	billService    services.BillService
	paymentService services.PaymentService
}

func NewHandler(billService services.BillService, paymentService services.PaymentService) *Handler {
	return &Handler{
		billService:    billService,
		paymentService: paymentService,
	}
}

func (h *Handler) PostCreateUserId(c echo.Context, userID string) error {
	if _, err := uuid.Parse(userID); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user_id format"})
	}

	bill, err := h.billService.CreateBill(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"bill_id": bill.ID.String(),
		"user_id": userID,
		"status":  "new",
	})
}

func (h *Handler) PostAddBillId(c echo.Context, billID string, params payments.PostAddBillIdParams) error {
	userID := c.QueryParam("user_id")

	if _, err := uuid.Parse(billID); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid bill_id format"})
	}
	if _, err := uuid.Parse(userID); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user_id format"})
	}

	var request struct {
		Amount float64 `json:"amount"`
	}
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	bill, err := h.billService.UpdateBalance(c.Request().Context(), billID, userID, request.Amount)
	if err != nil {
		if err.Error() == "bill does not belong to user" {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "bill not found for user"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"bill_id": bill.ID.String(),
		"balance": bill.Balance,
		"user_id": userID,
	})
}

func (h *Handler) GetBalanceBillId(c echo.Context, billID string, params payments.GetBalanceBillIdParams) error {
	userID := params.UserId

	if _, err := uuid.Parse(billID); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid bill_id format"})
	}
	if _, err := uuid.Parse(userID); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user_id format"})
	}

	bill, err := h.billService.GetBill(c.Request().Context(), billID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "bill not found"})
	}

	userUUID, _ := uuid.Parse(userID)
	if bill.UserID != userUUID {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "bill not found for user"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"bill_id": bill.ID.String(),
		"balance": bill.Balance,
		"user_id": userID,
	})
}
