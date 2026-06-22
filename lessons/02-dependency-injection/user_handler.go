package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// UserHandler holds HTTP logic. It depends on *UserService, not on a repo or DB directly.
// Each layer only knows about the layer directly below it — this is the chain.
type UserHandler struct {
	svc *UserService
}

func NewUserHandler(svc *UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) GetUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id must be an integer"})
		return
	}

	u, err := h.svc.GetUser(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, u)
}
