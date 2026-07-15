package main

import (
	"github.com/gin-gonic/gin"
)

func main() {
	reportRepo := NewReportRepo()
	reportSvc := NewReportService(reportRepo)
	reportHandler := NewReportHandler(reportSvc)

	auditRepo := NewAuditLogRepo()
	orderSvc := NewOrderService(auditRepo)
	orderHandler := NewOrderHandler(orderSvc)

	r := gin.Default()
	r.GET("/reports/:id", reportHandler.GetReport)
	r.GET("/reports/:id/fast", reportHandler.GetReportFast)
	r.POST("/orders", orderHandler.CreateOrder)

	r.Run(":8080")
}
