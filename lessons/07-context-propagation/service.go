package main

import (
	"context"
	"fmt"
	"time"
)

type ReportService struct {
	repo *ReportRepo
}

func NewReportService(repo *ReportRepo) *ReportService {
	return &ReportService{repo: repo}
}

// GetRevenueReport just forwards ctx. If the caller (Gin, via the request's
// http.Request.Context()) disconnects, this cancellation propagates all the
// way into the repo's query — no extra code needed, as long as nobody breaks
// the chain by substituting context.Background() somewhere in the middle.
func (s *ReportService) GetRevenueReport(ctx context.Context, accountID int64) (string, error) {
	return s.repo.BuildRevenueReport(ctx, accountID)
}

// GetRevenueReportWithSLA adds a service-level deadline on top of whatever
// the caller's ctx already carries. context.WithTimeout never loosens a
// deadline that's already tighter — it only ever tightens it. This is how
// you enforce "this endpoint must respond within 1s" independent of
// whatever timeout (or lack of one) the client's request carries.
func (s *ReportService) GetRevenueReportWithSLA(ctx context.Context, accountID int64) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel() // always release the timer, even on the success path

	report, err := s.repo.BuildRevenueReport(ctx, accountID)
	if err != nil {
		return "", fmt.Errorf("GetRevenueReportWithSLA: %w", err)
	}
	return report, nil
}

type OrderService struct {
	auditRepo *AuditLogRepo
}

func NewOrderService(auditRepo *AuditLogRepo) *OrderService {
	return &OrderService{auditRepo: auditRepo}
}

// CreateOrder returns to the client immediately but kicks off an audit-log
// write that must finish even after the HTTP response is sent — Gin cancels
// c.Request.Context() the instant the handler returns, which would kill the
// audit write mid-flight if we reused that ctx in the goroutine.
//
// context.WithoutCancel (stdlib since Go 1.21) gives you a copy of ctx that
// keeps its *values* (request_id, trace IDs, ...) but drops the cancellation
// signal and deadline. That's the correct tool here — NOT context.Background(),
// which would also silently discard the request_id and any other
// request-scoped values a downstream logger relies on.
func (s *OrderService) CreateOrder(ctx context.Context, productID string) error {
	detached := context.WithoutCancel(ctx)

	go func() {
		if err := s.auditRepo.Write(detached, fmt.Sprintf("order created: %s", productID)); err != nil {
			fmt.Printf("[audit] failed: %v\n", err)
		}
	}()

	return nil
}
