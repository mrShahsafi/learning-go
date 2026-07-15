package main

import (
	"context"
	"fmt"
	"time"
)

// ReportRepo simulates a slow analytics query — the kind that hits a
// reporting replica and can legitimately take seconds. Real pgx behaves
// exactly like this: every Query/Exec/QueryRow call selects on ctx.Done()
// internally and aborts the query (sending a Postgres CancelRequest) the
// moment ctx is canceled. You get that for free — but only if you pass ctx.
type ReportRepo struct{}

func NewReportRepo() *ReportRepo { return &ReportRepo{} }

func (r *ReportRepo) BuildRevenueReport(ctx context.Context, accountID int64) (string, error) {
	const queryTime = 3 * time.Second

	select {
	case <-time.After(queryTime):
		return fmt.Sprintf("revenue report for account %d: $42,000", accountID), nil
	case <-ctx.Done():
		// This is what pgx does when the client disconnects or a deadline
		// fires mid-query: the in-flight query is aborted, not left running.
		return "", fmt.Errorf("BuildRevenueReport: %w", ctx.Err())
	}
}

// AuditLogRepo simulates writing an audit trail row. It's intentionally
// slow enough to outlive a request's context in the fire-and-forget demo.
type AuditLogRepo struct{}

func NewAuditLogRepo() *AuditLogRepo { return &AuditLogRepo{} }

func (r *AuditLogRepo) Write(ctx context.Context, event string) error {
	const writeTime = 500 * time.Millisecond

	select {
	case <-time.After(writeTime):
		fmt.Printf("[audit] %s\n", event)
		return nil
	case <-ctx.Done():
		return fmt.Errorf("AuditLogRepo.Write: %w", ctx.Err())
	}
}
