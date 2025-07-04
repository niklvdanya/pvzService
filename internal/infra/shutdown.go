package infra

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func Graceful(cb ...func(context.Context)) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	slog.Info("shutdown signal received")

	ctx := context.Background()
	for _, f := range cb {
		f(ctx)
	}
	slog.Info("shutdown complete")
}
