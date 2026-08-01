package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/config"
	repository "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories"
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
			MaxAttempts: 5,
			WindowSize:  15 * time.Second,
		},
	}

	repo := repository.NewRateLimitRepo(client, cfg)
	ctx := context.Background()
	username := "test_user_rate_limit"

	t.Run("Initial attempts should succeed", func(t *testing.T) {
		for i := 1; i <= 4; i++ {
			isAllowed, remaining, wait, err := repo.CheckLoginRateLimit(ctx, username)
			require.NoError(t, err)
			assert.True(t, isAllowed, "Attempt %d should be allowed", i)
			assert.Equal(t, int(cfg.RateConfig.MaxAttempts)-i, remaining, "Attempt %d remaining count mismatch", i)
			assert.Equal(t, 0, wait, "Attempt %d wait should be 0", i)

			// We need a sleep to ensure the current Unix time increments for each score
			// as CheckLoginRateLimit uses time.Now().Unix() as the score and member
			time.Sleep(1 * time.Second)
		}
	})

	t.Run("Exceeding max attempts should be blocked", func(t *testing.T) {
		// The 5th attempt will hit >= max attempts condition since count will be 5
		isAllowed, remaining, wait, err := repo.CheckLoginRateLimit(ctx, username)
		require.NoError(t, err)
		assert.False(t, isAllowed, "Exceeding max attempts should be blocked")
		assert.Equal(t, 0, remaining, "Remaining count should be 0 when blocked")
		assert.Greater(t, wait, 0, "Wait time should be greater than 0")
		assert.LessOrEqual(t, wait, 15, "Wait time should not exceed window size")
	})

	t.Run("After window size expires, attempts should be allowed again", func(t *testing.T) {
		// Create a separate instance to avoid clashes, and let's configure max attempts to 2.
		// So the first attempt passes, the second gets blocked.
		cfgFast := &config.Config{
			RateConfig: config.RateConfig{
				MaxAttempts: 2,
				WindowSize:  2 * time.Second,
			},
		}
		repoFast := repository.NewRateLimitRepo(client, cfgFast)
		usernameFast := "fast_user"

		// 1st attempt: allowed (count = 1, MaxAttempts = 2)
		isAllowed, remaining, wait, err := repoFast.CheckLoginRateLimit(ctx, usernameFast)
		require.NoError(t, err)
		assert.True(t, isAllowed)
		assert.Equal(t, 1, remaining)
		assert.Equal(t, 0, wait)
        time.Sleep(1 * time.Second)

		// 2nd attempt: blocked (count = 2, MaxAttempts = 2)
		isAllowed, remaining, wait, err = repoFast.CheckLoginRateLimit(ctx, usernameFast)
		require.NoError(t, err)
		assert.False(t, isAllowed)
		assert.Equal(t, 0, remaining)
		assert.Greater(t, wait, 0)

		// Wait for window to expire
		time.Sleep(2 * time.Second)

		// 3rd attempt: allowed again
		isAllowed, remaining, wait, err = repoFast.CheckLoginRateLimit(ctx, usernameFast)
		require.NoError(t, err)
		assert.True(t, isAllowed)
		assert.Equal(t, 1, remaining)
		assert.Equal(t, 0, wait)
	})
}
