package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// One ctx for the whole process lifetime. SIGINT is Ctrl+C in a terminal;
	// SIGTERM is what Kubernetes / docker stop / systemd send first.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		slog.Error("server exited with error", "err", err)
		os.Exit(1)
	}
	slog.Info("clean exit")
}
