package main

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	logger *slog.Logger
}

func NewOrderHandler(logger *slog.Logger) *OrderHandler {
	return &OrderHandler{logger: logger}
}

type CreateOrderRequest struct {
	ProductID string `json:"product_id" binding:"required"`
	Quantity  int    `json:"quantity"   binding:"required,min=1"`
}

func (h *OrderHandler) Create(c *gin.Context) {
	ctx := c.Request.Context()

	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WarnContext(ctx, "invalid order payload", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Quantity > 100 {
		err := errors.New("quantity exceeds stock")
		h.logger.ErrorContext(ctx, "order rejected",
			slog.String("product_id", req.ProductID),
			slog.Int("quantity", req.Quantity),
			slog.String("error", err.Error()),
		)
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	// The trap, left here on purpose: slog.Info is the package-level default
	// logger. It never receives ctx, so request_id silently never appears on
	// this line even though everything else in this handler has it.
	// slog.Info("order created", "product_id", req.ProductID)

	h.logger.InfoContext(ctx, "order created",
		slog.String("product_id", req.ProductID),
		slog.Int("quantity", req.Quantity),
	)

	c.JSON(http.StatusCreated, gin.H{"status": "created"})
}
