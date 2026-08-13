package utils

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithDBTimeout(t *testing.T) {
	// Call the function under test
	ctx, cancel := WithDBTimeout(context.Background())
	defer cancel()

	// Ensure the returned context and cancel function are not nil
	require.NotNil(t, ctx)
	require.NotNil(t, cancel)

	// Extract the deadline from the returned context
	deadline, ok := ctx.Deadline()

	// Assert that a deadline is set
	require.True(t, ok, "Expected a deadline to be set on the context")

	// Calculate the expected deadline based on when the function was likely called
	expectedDeadline := time.Now().Add(DefaultDBTimeout)

	// Assert the deadline is within a reasonable tolerance (e.g., 50ms)
	assert.WithinDuration(t, expectedDeadline, deadline, 50*time.Millisecond, "Deadline should match DefaultDBTimeout from now")
}
