package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"
)

// run owns the full lifecycle: start everything, wait for the signal,
// then tear everything down IN ORDER. This is the code gunicorn ships
// for you — in Go, you are gunicorn.
func run(ctx context.Context) error {
	// --- startup, in dependency order ---
	pool := NewFakePool() // stands in for pgxpool.New(ctx, dsn)

	var wg sync.WaitGroup
	workerCtx, stopWorker := context.WithCancel(context.Background())
	defer stopWorker() // covers early-return paths; calling cancel twice is a safe no-op
	wg.Add(1)
	go auditFlusher(workerCtx, &wg)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: newRouter(pool),
	}

	// ListenAndServe blocks forever, so it runs in its own goroutine.
	// Its "we shut down on purpose" sentinel is http.ErrServerClosed —
	// that one is NOT an error, everything else is.
	errCh := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	// --- block here for the lifetime of the process ---
	select {
	case err := <-errCh:
		// e.g. port already in use — shutdown never even started
		return err
	case <-ctx.Done():
		slog.Info("shutdown signal received, draining requests")
	}

	// --- teardown, in REVERSE dependency order ---

	// 1. Stop accepting new connections, wait for in-flight requests.
	//    The ctx passed to Shutdown is a fresh DEADLINE ("how long am I
	//    willing to wait"), NOT the signal ctx — that one is already
	//    canceled, and Shutdown would give up immediately.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		// Deadline hit with requests still running. Shutdown already
		// returned; force-close whatever is left.
		slog.Warn("drain deadline exceeded, forcing close", "err", err)
		_ = srv.Close()
	}

	// 2. Requests are done — now it's safe to stop background workers.
	stopWorker()
	wg.Wait()

	// 3. Nothing can touch the pool anymore — close it last.
	pool.Close()

	return nil
}

// auditFlusher pretends to batch-flush audit events every second, like a
// background goroutine you'd start in main for Coordeck. It must exit via
// ctx, and the WaitGroup is how run() knows it actually finished.
func auditFlusher(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Info("audit flusher: final flush, exiting")
			return
		case <-ticker.C:
			slog.Debug("audit flusher: periodic flush")
		}
	}
}
