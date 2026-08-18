package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestAuthRejectsRequestBeforeHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	router := gin.New()
	router.Use(Auth("correct-token"))
	router.GET("/private", func(c *gin.Context) {
		called = true
		c.Status(http.StatusNoContent)
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/private", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if called {
		t.Fatal("protected handler ran after auth aborted the chain")
	}
	if got := response.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Fatalf("Content-Type = %q, want application/problem+json", got)
	}
}

func TestProblemRecoveryConvertsPanicToProblem(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ProblemRecovery())
	router.GET("/panic", func(*gin.Context) {
		panic("database password must not leak")
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/panic", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if got := response.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Fatalf("Content-Type = %q, want application/problem+json", got)
	}
	if strings.Contains(response.Body.String(), "password") {
		t.Fatal("panic details leaked into the response")
	}
}

func TestTimeoutCancelsRequestContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Timeout(10 * time.Millisecond))
	router.GET("/slow", func(c *gin.Context) {
		select {
		case <-c.Request.Context().Done():
			c.Status(http.StatusGatewayTimeout)
		case <-time.After(time.Second):
			c.Status(http.StatusInternalServerError)
		}
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/slow", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusGatewayTimeout {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusGatewayTimeout)
	}
}
