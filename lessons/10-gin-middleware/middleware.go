package main

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func Auth(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		provided, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !ok || subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
			c.Header("Content-Type", "application/problem+json")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"type":   "https://api.coordeck.com/problems/unauthorized",
				"title":  "Unauthorized",
				"status": http.StatusUnauthorized,
			})
			return
		}

		c.Next()
	}
}

func ProblemRecovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, _ any) {
		c.Header("Content-Type", "application/problem+json")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"type":   "https://api.coordeck.com/problems/internal-server-error",
			"title":  "Internal Server Error",
			"status": http.StatusInternalServerError,
		})
	})
}

func Timeout(duration time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), duration)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
