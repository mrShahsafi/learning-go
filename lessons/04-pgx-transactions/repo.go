package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Querier is satisfied by *pgxpool.Pool, pgx.Tx, and *pgxpool.Conn.
// Repos that accept Querier work both inside and outside transactions.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type AccountRepo struct {
	pool *pgxpool.Pool // used for non-transactional reads
}

func NewAccountRepo(pool *pgxpool.Pool) *AccountRepo {
	return &AccountRepo{pool: pool}
}

// Debit subtracts amount from an account. q can be pool or a tx.
func (r *AccountRepo) Debit(ctx context.Context, q Querier, accountID int64, amount int64) error {
	tag, err := q.Exec(ctx,
		`UPDATE accounts SET balance = balance - $1 WHERE id = $2 AND balance >= $1`,
		amount, accountID,
	)
	if err != nil {
		return fmt.Errorf("debit account %d: %w", accountID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("account %d has insufficient funds or does not exist", accountID)
	}
	return nil
}

// Credit adds amount to an account. q can be pool or a tx.
func (r *AccountRepo) Credit(ctx context.Context, q Querier, accountID int64, amount int64) error {
	tag, err := q.Exec(ctx,
		`UPDATE accounts SET balance = balance + $1 WHERE id = $2`,
		amount, accountID,
	)
	if err != nil {
		return fmt.Errorf("credit account %d: %w", accountID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("account %d does not exist", accountID)
	}
	return nil
}

// GetBalance reads the current balance — no transaction needed for a simple read.
func (r *AccountRepo) GetBalance(ctx context.Context, accountID int64) (int64, error) {
	var balance int64
	err := r.pool.QueryRow(ctx,
		`SELECT balance FROM accounts WHERE id = $1`, accountID,
	).Scan(&balance)
	if err != nil {
		return 0, fmt.Errorf("get balance for account %d: %w", accountID, err)
	}
	return balance, nil
}
