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

func TestNewRateLimitRepo(t *testing.T) {
	db, _ := redismock.NewClientMock()
	cfg := &config.Config{}

	repo := repository.NewRateLimitRepo(db, cfg)
	assert.NotNil(t, repo)
}

func TestCheckLoginRateLimit_Success(t *testing.T) {
	db, mock := redismock.NewClientMock()
	cfg := &config.Config{
		RateConfig: config.RateConfig{
			WindowSize:  60 * time.Second,
			MaxAttempts: 5,
		},
	}

	repo := repository.NewRateLimitRepo(db, cfg)
	ctx := context.Background()
	username := "testuser"
	key := "login_attempts:" + username

	now := time.Now().Unix()
	windowStart := strconv.FormatInt(now-int64(cfg.RateConfig.WindowSize.Seconds()), 10)

	mock.ExpectZRemRangeByScore(key, "0", windowStart).SetVal(0)
	mock.ExpectZAdd(key, redis.Z{Score: float64(now), Member: now}).SetVal(1)
	mock.ExpectZCard(key).SetVal(2)
	mock.ExpectExpire(key, cfg.RateConfig.WindowSize).SetVal(true)

	allowed, remaining, resetIn, err := repo.CheckLoginRateLimit(ctx, username)
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, 3, remaining)
	assert.Equal(t, 0, resetIn)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckLoginRateLimit_PipelineError(t *testing.T) {
	db, mock := redismock.NewClientMock()
	cfg := &config.Config{
		RateConfig: config.RateConfig{
			WindowSize:  60 * time.Second,
			MaxAttempts: 5,
		},
	}

	repo := repository.NewRateLimitRepo(db, cfg)
	ctx := context.Background()
	username := "testuser"
	key := "login_attempts:" + username

	now := time.Now().Unix()
	windowStart := strconv.FormatInt(now-int64(cfg.RateConfig.WindowSize.Seconds()), 10)

	mock.ExpectZRemRangeByScore(key, "0", windowStart).SetVal(0)
	mock.ExpectZAdd(key, redis.Z{Score: float64(now), Member: now}).SetVal(1)
	mock.ExpectZCard(key).SetVal(2)
	mock.ExpectExpire(key, cfg.RateConfig.WindowSize).SetErr(errors.New("redis connection failed"))

	allowed, remaining, resetIn, err := repo.CheckLoginRateLimit(ctx, username)
	require.Error(t, err)
	assert.False(t, allowed)
	assert.Equal(t, 0, remaining)
	assert.Equal(t, 0, resetIn)
	assert.Contains(t, err.Error(), "redis pipeline error for rate limit check")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckLoginRateLimit_RateLimitExceeded_Success(t *testing.T) {
	db, mock := redismock.NewClientMock()
	cfg := &config.Config{
		RateConfig: config.RateConfig{
			WindowSize:  60 * time.Second,
			MaxAttempts: 5,
		},
	}

	repo := repository.NewRateLimitRepo(db, cfg)
	ctx := context.Background()
	username := "testuser"
	key := "login_attempts:" + username

	now := time.Now().Unix()
	windowStart := strconv.FormatInt(now-int64(cfg.RateConfig.WindowSize.Seconds()), 10)

	mock.ExpectZRemRangeByScore(key, "0", windowStart).SetVal(0)
	mock.ExpectZAdd(key, redis.Z{Score: float64(now), Member: now}).SetVal(1)
	mock.ExpectZCard(key).SetVal(5)
	mock.ExpectExpire(key, cfg.RateConfig.WindowSize).SetVal(true)

	oldestAttemptTime := float64(now - 10) // 10 seconds ago

	mock.ExpectZRangeArgsWithScores(redis.ZRangeArgs{
		Key: key, Start: 0, Stop: 0,
	}).SetVal([]redis.Z{
		{Score: oldestAttemptTime, Member: oldestAttemptTime},
	})

	allowed, remaining, resetIn, err := repo.CheckLoginRateLimit(ctx, username)
	require.NoError(t, err)
	assert.False(t, allowed)
	assert.Equal(t, 0, remaining)
	assert.GreaterOrEqual(t, resetIn, 45) // ~50s remaining
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckLoginRateLimit_RateLimitExceeded_CalculateRetryAfterError(t *testing.T) {
	db, mock := redismock.NewClientMock()
	cfg := &config.Config{
		RateConfig: config.RateConfig{
			WindowSize:  60 * time.Second,
			MaxAttempts: 5,
		},
	}

	repo := repository.NewRateLimitRepo(db, cfg)
	ctx := context.Background()
	username := "testuser"
	key := "login_attempts:" + username

	now := time.Now().Unix()
	windowStart := strconv.FormatInt(now-int64(cfg.RateConfig.WindowSize.Seconds()), 10)

	mock.ExpectZRemRangeByScore(key, "0", windowStart).SetVal(0)
	mock.ExpectZAdd(key, redis.Z{Score: float64(now), Member: now}).SetVal(1)
	mock.ExpectZCard(key).SetVal(5)
	mock.ExpectExpire(key, cfg.RateConfig.WindowSize).SetVal(true)

	mock.ExpectZRangeArgsWithScores(redis.ZRangeArgs{
		Key: key, Start: 0, Stop: 0,
	}).SetErr(errors.New("failed to fetch oldest attempt"))

	allowed, remaining, resetIn, err := repo.CheckLoginRateLimit(ctx, username)
	require.Error(t, err)
	assert.False(t, allowed)
	assert.Equal(t, 0, remaining)
	assert.Equal(t, 60, resetIn)
	assert.Contains(t, err.Error(), "failed to get oldest attempt time")
	require.NoError(t, mock.ExpectationsWereMet())
}
