package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetryUntilSuccess(t *testing.T) {
	attempts := 0
	err := retry(t.Context(), 0, func(context.Context) error {
		attempts++

		if attempts < 3 {
			return errors.New("storage unavailable")
		}

		return nil
	})

	if err != nil || attempts != 3 {
		t.Fatalf("attempts = %d, error = %v", attempts, err)
	}
}

func TestRetryStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())

	defer cancel()

	failure := errors.New("storage unavailable")

	attempts := 0
	err := retry(ctx, time.Hour, func(context.Context) error {
		attempts++

		cancel()

		return failure
	})

	if attempts != 1 || !errors.Is(err, context.Canceled) || !errors.Is(err, failure) {
		t.Fatalf("attempts = %d, error = %v", attempts, err)
	}
}

func TestRetrySkipsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())

	cancel()

	err := retry(ctx, 0, func(context.Context) error {
		t.Fatal("operation called after cancellation")

		return nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want cancellation", err)
	}
}

func TestRetryDeadlineInterruptsWait(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)

	defer cancel()

	err := retry(ctx, time.Hour, func(context.Context) error {
		return errors.New("unavailable")
	})

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want deadline", err)
	}
}
