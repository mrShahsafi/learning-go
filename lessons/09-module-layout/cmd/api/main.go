// main.go is tiny on purpose: it does exactly one thing — wire the layers
// together in dependency order — and contains zero business logic.
package main

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"lesson09/module-layout/internal/config"
	"lesson09/module-layout/internal/handler"
	"lesson09/module-layout/internal/repository"
	"lesson09/module-layout/internal/service"
)

func main() {
	cfg := config.Load()

	repo := repository.NewMemoryBookingRepo()
	svc := service.NewBookingService(repo)
	h := handler.NewBookingHandler(svc)

	r := gin.Default()
	handler.RegisterRoutes(r, h)

	slog.Info("listening", "port", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		slog.Error("server exited", "err", err)
	}
}
