package main

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ReportHandler struct {
	svc *ReportService
}

func NewReportHandler(svc *ReportService) *ReportHandler {
	return &ReportHandler{svc: svc}
}

// GetReport uses c.Request.Context() — the Django equivalent of nothing,
// because Django never gives you a handle on "has the client gone away".
// Try it: `curl localhost:8080/reports/1` then hit Ctrl+C before 3s pass.
// The server log prints "query aborted: context canceled" instead of
// wastefully finishing a 3-second query nobody will ever read.
func (h *ReportHandler) GetReport(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	report, err := h.svc.GetRevenueReport(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			// Client is gone — nothing to write a response to. This branch
			// exists for the log line; c.JSON below would be a no-op anyway.
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"report": report})
}

// GetReportFast enforces a 1s SLA regardless of how patient the client is.
// Compare: `curl localhost:8080/reports/1/fast` always fails after ~1s,
// even with no Ctrl+C, because BuildRevenueReport takes 3s.
func (h *ReportHandler) GetReportFast(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	report, err := h.svc.GetRevenueReportWithSLA(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"report": report})
}

type OrderHandler struct {
	svc *OrderService
}

func NewOrderHandler(svc *OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

type createOrderRequest struct {
	ProductID string `json:"product_id"`
}

// CreateOrder returns in ~0ms even though the audit write takes 500ms,
// because the audit write is detached from this request's ctx. Watch the
// server log: "[audit] order created: ..." prints ~500ms *after* curl
// already got its 201 response back.
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	if err := h.svc.CreateOrder(c.Request.Context(), req.ProductID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "accepted"})
}
