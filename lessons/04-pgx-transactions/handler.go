package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type TransferHandler struct {
	pool *pgxpoolWrapper
	repo *AccountRepo
}

// pgxpoolWrapper is defined in main.go — it holds *pgxpool.Pool
// We keep handler.go import-clean by using the wrapper type.

type TransferRequest struct {
	FromID int64 `json:"from_id" binding:"required"`
	ToID   int64 `json:"to_id"   binding:"required"`
	Amount int64 `json:"amount"  binding:"required,min=1"`
}

// Transfer moves money between two accounts atomically.
// Both Debit and Credit run inside the same transaction — if either fails,
// the whole thing rolls back. No partial state ever reaches the DB.
func (h *TransferHandler) Transfer(c *gin.Context) {
	var req TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := WithTx(c.Request.Context(), h.pool.pool, func(tx pgxTx) error {
		if err := h.repo.Debit(c.Request.Context(), tx, req.FromID, req.Amount); err != nil {
			return err
		}
		return h.repo.Credit(c.Request.Context(), tx, req.ToID, req.Amount)
	})
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "transferred"})
}
