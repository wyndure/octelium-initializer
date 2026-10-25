package main

import (
	"context"
	"fmt"
	"time"
)

// retry attempts the operation immediately, then waits between failed attempts.
func retry(ctx context.Context, interval time.Duration, operation func(context.Context) error) error {
	var lastErr error
	for {
		if err := ctx.Err(); err != nil {
			return retryError(err, lastErr)
		}

		lastErr = operation(ctx)
		if lastErr == nil {
			return nil
		}

		timer := time.NewTimer(interval)

		select {
		case <-ctx.Done():
			timer.Stop()
			return retryError(ctx.Err(), lastErr)
		case <-timer.C:
		}
	}
}

func retryError(ctxErr error, lastErr error) error {
	if lastErr == nil {
		return ctxErr
	}

	return fmt.Errorf("%w; last attempt: %w", ctxErr, lastErr)
}
