package main

import (
	"context"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// pgxpoolWrapper exists so handler.go can hold a *pgxpool.Pool without importing pgxpool itself.
type pgxpoolWrapper struct {
	pool *pgxpool.Pool
}

// pgxTx is a local alias so handler.go doesn't need to import pgx/v5 directly.
type pgxTx = pgx.Tx

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/lesson04?sslmode=disable"
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("ping: %v\n\nHint: start Postgres and set DATABASE_URL, or run:\n  docker run --rm -p 5432:5432 -e POSTGRES_PASSWORD=postgres postgres:15", err)
	}

	repo := NewAccountRepo(pool)
	h := &TransferHandler{pool: &pgxpoolWrapper{pool: pool}, repo: repo}

	r := gin.Default()
	r.POST("/transfer", h.Transfer)

	log.Println("listening on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
