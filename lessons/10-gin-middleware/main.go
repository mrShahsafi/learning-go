package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.New()
	router.Use(ProblemRecovery())

	router.GET("/panic", func(*gin.Context) {
		panic("simulated failure")
	})

	router.GET("/slow", Timeout(50*time.Millisecond), func(c *gin.Context) {
		select {
		case <-c.Request.Context().Done():
			c.Header("Content-Type", "application/problem+json")
			c.JSON(http.StatusGatewayTimeout, gin.H{
				"type":   "https://api.coordeck.com/problems/timeout",
				"title":  "Gateway Timeout",
				"status": http.StatusGatewayTimeout,
			})
		case <-time.After(200 * time.Millisecond):
			c.JSON(http.StatusOK, gin.H{"status": "finished"})
		}
	})

	private := router.Group("/private", Auth("dev-token"))
	private.GET("/bookings", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"bookings": []string{"desk A", "room B"}})
	})

	if err := router.Run(":8080"); err != nil {
		panic(err)
	}
}
