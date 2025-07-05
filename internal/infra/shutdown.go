package infra

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const timeout = 10 * time.Second

func Graceful(cb ...func(context.Context)) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	<-sigCh
	slog.Info("graceful shutdown started")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	go func() {
		select {
		case <-sigCh:
			slog.Warn("second interrupt — forcing exit")
			os.Exit(1)
		case <-ctx.Done():
		}
	}()

	var wg sync.WaitGroup
	for _, f := range cb {
		wg.Add(1)
		go func(fn func(context.Context)) {
			defer wg.Done()
			fn(ctx)
		}(f)
	}
	wg.Wait()
	slog.Info("shutdown complete")
}
