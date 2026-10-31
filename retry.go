package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/wyndure/octelium-initializer/internal/bootstrap"
)

// retry attempts initialization immediately, then waits between transient failures.
// A missing domain is a configuration error and is returned without retrying.
func retry(ctx context.Context, interval time.Duration, operation func(context.Context) error) error {
	var lastErr error
	for {
		if err := ctx.Err(); err != nil {
			return retryError(err, lastErr)
		}

		lastErr = operation(ctx)
		if lastErr == nil || errors.Is(lastErr, bootstrap.ErrDomainRequired) {
			return lastErr
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
