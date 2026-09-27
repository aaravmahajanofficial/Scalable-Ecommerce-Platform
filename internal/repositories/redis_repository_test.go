package repository_test

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/config"
	repository "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories"
	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type rateLimitTestFixture struct {
	repo     repository.RateLimitRepository
	mock     redismock.ClientMock
	ctx      context.Context
	username string
	key      string
	now      int64
	cfg      *config.Config
}

func setupRateLimitTest(countVal int64, expireErr error) rateLimitTestFixture {
	db, mock := redismock.NewClientMock()
	cfg := &config.Config{
		RateConfig: config.RateConfig{
			WindowSize:  60 * time.Second,
			MaxAttempts: 5,
		},
	}

	repo := repository.NewRateLimitRepo(db, cfg)
	username := "testuser"
	key := "login_attempts:" + username

	now := time.Now().Unix()
	windowStart := strconv.FormatInt(now-int64(cfg.RateConfig.WindowSize.Seconds()), 10)

	mock.ExpectZRemRangeByScore(key, "0", windowStart).SetVal(0)
	mock.ExpectZAdd(key, redis.Z{Score: float64(now), Member: now}).SetVal(1)
	mock.ExpectZCard(key).SetVal(countVal)

	expCmd := mock.ExpectExpire(key, cfg.RateConfig.WindowSize)
	if expireErr != nil {
		expCmd.SetErr(expireErr)
	} else {
		expCmd.SetVal(true)
	}

	return rateLimitTestFixture{
		repo:     repo,
		mock:     mock,
		ctx:      context.Background(),
		username: username,
		key:      key,
		now:      now,
		cfg:      cfg,
	}
}

func TestNewRateLimitRepo(t *testing.T) {
	db, _ := redismock.NewClientMock()
	cfg := &config.Config{}

	repo := repository.NewRateLimitRepo(db, cfg)
	assert.NotNil(t, repo)
}

func TestCheckLoginRateLimit_Success(t *testing.T) {
	f := setupRateLimitTest(2, nil)

	allowed, remaining, resetIn, err := f.repo.CheckLoginRateLimit(f.ctx, f.username)
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, 3, remaining)
	assert.Equal(t, 0, resetIn)
	require.NoError(t, f.mock.ExpectationsWereMet())
}

func TestCheckLoginRateLimit_PipelineError(t *testing.T) {
	f := setupRateLimitTest(2, errors.New("redis connection failed"))

	allowed, remaining, resetIn, err := f.repo.CheckLoginRateLimit(f.ctx, f.username)
	require.Error(t, err)
	assert.False(t, allowed)
	assert.Equal(t, 0, remaining)
	assert.Equal(t, 0, resetIn)
	assert.Contains(t, err.Error(), "redis pipeline error for rate limit check")
	require.NoError(t, f.mock.ExpectationsWereMet())
}

func TestCheckLoginRateLimit_RateLimitExceeded_Success(t *testing.T) {
	f := setupRateLimitTest(5, nil)

	oldestAttemptTime := float64(f.now - 10) // 10 seconds ago

	f.mock.ExpectZRangeArgsWithScores(redis.ZRangeArgs{
		Key: f.key, Start: 0, Stop: 0,
	}).SetVal([]redis.Z{
		{Score: oldestAttemptTime, Member: oldestAttemptTime},
	})

	allowed, remaining, resetIn, err := f.repo.CheckLoginRateLimit(f.ctx, f.username)
	require.NoError(t, err)
	assert.False(t, allowed)
	assert.Equal(t, 0, remaining)
	assert.GreaterOrEqual(t, resetIn, 45) // ~50s remaining
	require.NoError(t, f.mock.ExpectationsWereMet())
}

func TestCheckLoginRateLimit_RateLimitExceeded_CalculateRetryAfterError(t *testing.T) {
	f := setupRateLimitTest(5, nil)

	f.mock.ExpectZRangeArgsWithScores(redis.ZRangeArgs{
		Key: f.key, Start: 0, Stop: 0,
	}).SetErr(errors.New("failed to fetch oldest attempt"))

	allowed, remaining, resetIn, err := f.repo.CheckLoginRateLimit(f.ctx, f.username)
	require.Error(t, err)
	assert.False(t, allowed)
	assert.Equal(t, 0, remaining)
	assert.Equal(t, 60, resetIn)
	assert.Contains(t, err.Error(), "failed to get oldest attempt time")
	require.NoError(t, f.mock.ExpectationsWereMet())
}
