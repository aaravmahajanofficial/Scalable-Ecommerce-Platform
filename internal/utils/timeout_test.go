package utils_test

import (
	"context"
	"testing"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestWithDBTimeout(t *testing.T) {
	t.Run("sets default db timeout", func(t *testing.T) {
		ctx := context.Background()
		now := time.Now()

		ctxWithTimeout, cancel := utils.WithDBTimeout(ctx)
		defer cancel()

		deadline, ok := ctxWithTimeout.Deadline()

		assert.True(t, ok, "Expected context to have a deadline")

		expectedDeadline := now.Add(utils.DefaultDBTimeout)
		assert.WithinDuration(t, expectedDeadline, deadline, 100*time.Millisecond, "Deadline should be roughly 5 seconds from now")
	})

	t.Run("returns working cancel function", func(t *testing.T) {
		ctx := context.Background()
		ctxWithTimeout, cancel := utils.WithDBTimeout(ctx)

		// Context should not be canceled initially
		assert.NoError(t, ctxWithTimeout.Err(), "Context should not be canceled initially")

		cancel()

		// Context should be canceled now
		assert.ErrorIs(t, ctxWithTimeout.Err(), context.Canceled, "Context should be canceled after calling CancelFunc")
	})
}
