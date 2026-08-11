package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"lesson09/module-layout/internal/service"
)

// BookingHandler only knows about the service INTERFACE — it never sees
// repository or domain's internals directly, and repository/service never
// import this package back. That one-way arrow is the whole point.
type BookingHandler struct {
	svc service.BookingService
}

func NewBookingHandler(svc service.BookingService) *BookingHandler {
	return &BookingHandler{svc: svc}
}

func RegisterRoutes(r *gin.Engine, h *BookingHandler) {
	r.POST("/bookings", h.Create)
	r.GET("/bookings/:id", h.Get)
}

type createBookingRequest struct {
	Title string `json:"title"`
}

func (h *BookingHandler) Create(c *gin.Context) {
	var req createBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	b, err := h.svc.CreateBooking(c.Request.Context(), req.Title)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, b)
}

func (h *BookingHandler) Get(c *gin.Context) {
	b, err := h.svc.GetBooking(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, b)
}
