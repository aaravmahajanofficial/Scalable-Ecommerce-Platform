package utils

import (
	"context"
	"testing"
	"time"
)

func TestWithDBTimeout(t *testing.T) {
	t.Run("creates context with deadline", func(t *testing.T) {
		start := time.Now()
		ctx, cancel := WithDBTimeout(context.Background())
		defer cancel()

		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("expected context to have a deadline")
		}

		// Calculate the expected deadline
		expected := start.Add(DefaultDBTimeout)

		// The difference should be very small (e.g., < 100ms)
		diff := deadline.Sub(expected)
		if diff < -100*time.Millisecond || diff > 100*time.Millisecond {
			t.Errorf("expected deadline around %v, got %v (diff: %v)", expected, deadline, diff)
		}
	})

	t.Run("cancel function works", func(t *testing.T) {
		ctx, cancel := WithDBTimeout(context.Background())

		// Context should not be done yet
		select {
		case <-ctx.Done():
			t.Fatal("context should not be done immediately")
		default:
		}

		// Cancel the context
		cancel()

		// Context should now be done
		select {
		case <-ctx.Done():
			// expected
			err := ctx.Err()
			if err != context.Canceled {
				t.Errorf("expected context to be canceled, got %v", err)
			}
		default:
			t.Fatal("context should be done after calling cancel")
		}
	})
}
