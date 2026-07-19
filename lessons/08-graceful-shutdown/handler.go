package main

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// FakePool stands in for *pgxpool.Pool so the lesson runs without Postgres.
type FakePool struct{ closed bool }

func NewFakePool() *FakePool { return &FakePool{} }

func (p *FakePool) Query(d time.Duration) string {
	if p.closed {
		// This is what a request sees if you close the pool BEFORE
		// draining requests — teardown order matters.
		return "ERROR: pool is closed"
	}
	time.Sleep(d) // pretend this is a pgx query
	return "42 rows"
}

func (p *FakePool) Close() {
	p.closed = true
	slog.Info("db pool closed")
}

func newRouter(pool *FakePool) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	// A deliberately slow endpoint: start it, then Ctrl+C the server.
	// Graceful shutdown means this request still gets its 200.
	r.GET("/slow", func(c *gin.Context) {
		slog.Info("slow request started")
		result := pool.Query(5 * time.Second)
		slog.Info("slow request finished")
		c.JSON(http.StatusOK, gin.H{"result": result})
	})

	r.GET("/healthz", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	return r
}
