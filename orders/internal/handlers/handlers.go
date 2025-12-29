package handlers

import (
	"net/http"

	models "sd_hw4/orders/internal/models"
	services "sd_hw4/orders/internal/service"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// OrderHandler обрабатывает HTTP запросы для заказов
type OrderHandler struct {
	orderService *services.OrderService
}

func NewOrderHandler(orderService *services.OrderService) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
	}
}

// CreateOrder создает новый заказ
func (h *OrderHandler) CreateOrder(c echo.Context) error {
	userIDParam := c.Param("user_id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid user ID")
	}

	var req struct {
		Amount      float64 `json:"amount"`
		Description string  `json:"description"`
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if req.Amount <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Amount must be positive")
	}

	// Используем сервис для создания заказа
	order, err := h.orderService.CreateOrder(c.Request().Context(), userID, req.Amount, req.Description)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create order")
	}

	// Преобразуем в DTO для ответа
	response := models.Order{
		ID:          order.ID,
		UserID:      order.UserID,
		Price:       order.Price,
		Description: order.Description,
		Status:      string(models.OrderStatus(order.Status)),
	}

	return c.JSON(http.StatusCreated, response)
}

// GetUserOrders возвращает заказы пользователя
func (h *OrderHandler) GetUserOrders(c echo.Context) error {
	userIDParam := c.Param("user_id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid user ID")
	}

	orders, err := h.orderService.GetOrdersByUser(c.Request().Context(), userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get orders")
	}

	// Преобразуем в DTO для ответа
	response := make([]models.Order, len(orders))
	for i, order := range orders {
		response[i] = models.Order{
			ID:          order.ID,
			UserID:      order.UserID,
			Price:       order.Price,
			Description: order.Description,
			Status:      string(models.OrderStatus(order.Status)),
		}
	}

	return c.JSON(http.StatusOK, response)
}

// GetOrderStatus возвращает статус заказа
func (h *OrderHandler) GetOrderStatus(c echo.Context) error {
	orderIDParam := c.Param("order_id")
	orderID, err := uuid.Parse(orderIDParam)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid order ID")
	}

	order, err := h.orderService.GetOrderByID(c.Request().Context(), orderID)
	if err != nil {
		// Здесь можно проверить тип ошибки (NotFound vs другие)
		return echo.NewHTTPError(http.StatusNotFound, "Order not found")
	}

	// Преобразуем в DTO для ответа
	response := models.Order{
		ID:          order.ID,
		UserID:      order.UserID,
		Price:       order.Price,
		Description: order.Description,
		Status:      string(models.OrderStatus(order.Status)),
	}

	return c.JSON(http.StatusOK, response)
}

// RegisterRoutes регистрирует маршруты (реализация ServerInterface)
func (h *OrderHandler) PostCreateUserId(c echo.Context, userId string) error {
	c.SetParamNames("user_id")
	c.SetParamValues(userId)
	return h.CreateOrder(c)
}

func (h *OrderHandler) GetOrdersUserId(c echo.Context, userId string) error {
	c.SetParamNames("user_id")
	c.SetParamValues(userId)
	return h.GetUserOrders(c)
}

func (h *OrderHandler) GetStatusOrderId(c echo.Context, orderId string) error {
	c.SetParamNames("order_id")
	c.SetParamValues(orderId)
	return h.GetOrderStatus(c)
}
