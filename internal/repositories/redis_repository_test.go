package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/config"
	repository "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckLoginRateLimit(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer client.Close()

	cfg := &config.Config{
		RateConfig: config.RateConfig{
			MaxAttempts: 3,
			WindowSize:  5 * time.Second,
		},
	}

	repo := repository.NewRateLimitRepo(client, cfg)
	ctx := context.Background()
	username := "testuser"

	// Mock time
	mockTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	repository.SetTimeNow(func() time.Time { return mockTime })
	defer repository.SetTimeNow(time.Now)

	// 1. First attempt: should succeed and return remaining=2
	allowed, remaining, retryAfter, err := repo.CheckLoginRateLimit(ctx, username)
	assert.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, 2, remaining)
	assert.Equal(t, 0, retryAfter)

	// Advance time by 1s for second attempt
	mockTime = mockTime.Add(1 * time.Second)
	mr.FastForward(1 * time.Second)

	// 2. Second attempt: should succeed and return remaining=1
	allowed, remaining, retryAfter, err = repo.CheckLoginRateLimit(ctx, username)
	assert.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, 1, remaining)
	assert.Equal(t, 0, retryAfter)

	// Advance time by 1s for third attempt
	mockTime = mockTime.Add(1 * time.Second)
	mr.FastForward(1 * time.Second)

	// 3. Third attempt: should fail due to rate limit and return retryAfter > 0
	allowed, remaining, retryAfter, err = repo.CheckLoginRateLimit(ctx, username)
	assert.NoError(t, err)
	assert.False(t, allowed)
	assert.Equal(t, 0, remaining)
	assert.Equal(t, 3, retryAfter) // First attempt was at T=0, window is 5s. Retry is at T=5. Current time is T=2. 5 - 2 = 3s.

	// Advance time to just after the window expires for the first attempt (T=6)
	mockTime = mockTime.Add(4 * time.Second)
	mr.FastForward(4 * time.Second)

	// 4. Fourth attempt: First attempt expired, but second (T=1) and third (T=2, count=3 total) are still in window?
	// Wait:
	// At T=0, count = 1.
	// At T=1, count = 2.
	// At T=2, count = 3.
	// At T=6, Window start = 6 - 5 = 1.
	// Elements: T=0 (expired), T=1 (still there, but it's exactly windowStart, wait, ZRemRangeByScore includes min/max: "0" to "1". So T=1 is removed!)
	// Wait, windowStart = 6 - 5 = 1. ZRemRangeByScore removes elements with score <= 1.
	// So T=0 and T=1 are removed.
	// Remaining elements: T=2 (1 element).
	// Current attempt adds T=6 (1 element).
	// Total elements: 2.
	// MaxAttempts: 3.
	// Remaining: 1.
	allowed, remaining, retryAfter, err = repo.CheckLoginRateLimit(ctx, username)
	assert.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, 1, remaining)
	assert.Equal(t, 0, retryAfter)
}

func TestCheckLoginRateLimit_Error(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	cfg := &config.Config{
		RateConfig: config.RateConfig{
			MaxAttempts: 3,
			WindowSize:  5 * time.Second,
		},
	}

	repo := repository.NewRateLimitRepo(client, cfg)
	ctx := context.Background()
	username := "erroruser"

	err = client.Close()
	require.NoError(t, err)

	allowed, remaining, retryAfter, err := repo.CheckLoginRateLimit(ctx, username)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "redis pipeline error for rate limit check")
	assert.False(t, allowed)
	assert.Equal(t, 0, remaining)
	assert.Equal(t, 0, retryAfter)
}
