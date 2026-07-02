package main

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

func main() {
	logger := NewLogger()
	slog.SetDefault(logger) // only affects package-level slog.Info/Error calls, not *Context calls

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(RequestLogger(logger))

	h := NewOrderHandler(logger)
	r.POST("/orders", h.Create)

	logger.Info("listening on :8080")
	if err := r.Run(":8080"); err != nil {
		logger.Error("server error", slog.String("error", err.Error()))
	}
}
