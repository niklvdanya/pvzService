package app

import (
	"context"
	"time"
)

func WithTimeoutAndContextCheck[T any](parent context.Context, timeout time.Duration, fn func(ctx context.Context) (T, error)) (T, error) {
	var zero T
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	if ctx.Err() != nil {
		return zero, ctx.Err()
	}
	return fn(ctx)
}
