package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/config"
	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

type rateLimitTestEnv struct {
	repo     RateLimitRepository
	mock     redismock.ClientMock
	username string
	key      string
	now      int64
}

func newTestRateLimitEnv(t *testing.T) rateLimitTestEnv {
	t.Helper()

	client, mock := redismock.NewClientMock()
	cfg := &config.Config{
		RateConfig: config.RateConfig{
			MaxAttempts: 5,
			WindowSize:  60 * time.Second,
		},
	}

	username := "testuser"

	return rateLimitTestEnv{
		repo:     NewRateLimitRepo(client, cfg),
		mock:     mock,
		username: username,
		key:      "login_attempts:" + username,
		now:      time.Now().Unix(),
	}
}

func mockPipelineSuccess(env rateLimitTestEnv, count int64) {
	env.mock.Regexp().ExpectZRemRangeByScore(env.key, "0", ".*").SetVal(0)
	env.mock.ExpectZAdd(env.key, redis.Z{Score: float64(env.now), Member: env.now}).SetVal(1)
	env.mock.ExpectZCard(env.key).SetVal(count)
	env.mock.ExpectExpire(env.key, 60*time.Second).SetVal(true)
}

func TestCheckLoginRateLimit_Success(t *testing.T) {
	env := newTestRateLimitEnv(t)
	mockPipelineSuccess(env, 2)

	allowed, remaining, resetIn, err := env.repo.CheckLoginRateLimit(context.Background(), env.username)

	assert.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, 3, remaining)
	assert.Equal(t, 0, resetIn)
	assert.NoError(t, env.mock.ExpectationsWereMet())
}

func TestCheckLoginRateLimit_Exceeded(t *testing.T) {
	env := newTestRateLimitEnv(t)
	mockPipelineSuccess(env, 5)

	env.mock.ExpectZRangeArgsWithScores(redis.ZRangeArgs{
		Key: env.key, Start: 0, Stop: 0,
	}).SetVal([]redis.Z{
		{Score: float64(env.now - 10), Member: env.now - 10},
	})

	allowed, remaining, resetIn, err := env.repo.CheckLoginRateLimit(context.Background(), env.username)

	assert.NoError(t, err)
	assert.False(t, allowed)
	assert.Equal(t, 0, remaining)
	assert.True(t, resetIn > 0)
	assert.NoError(t, env.mock.ExpectationsWereMet())
}

func TestCheckLoginRateLimit_PipelineError(t *testing.T) {
	env := newTestRateLimitEnv(t)
	env.mock.Regexp().ExpectZRemRangeByScore(env.key, "0", ".*").SetErr(errors.New("redis err"))

	allowed, remaining, resetIn, err := env.repo.CheckLoginRateLimit(context.Background(), env.username)

	assert.Error(t, err)
	assert.False(t, allowed)
	assert.Equal(t, 0, remaining)
	assert.Equal(t, 0, resetIn)
	assert.NoError(t, env.mock.ExpectationsWereMet())
}

func TestCheckLoginRateLimit_OldestAttemptError(t *testing.T) {
	env := newTestRateLimitEnv(t)
	mockPipelineSuccess(env, 5)

	env.mock.ExpectZRangeArgsWithScores(redis.ZRangeArgs{
		Key: env.key, Start: 0, Stop: 0,
	}).SetErr(errors.New("zrange error"))

	allowed, remaining, resetIn, err := env.repo.CheckLoginRateLimit(context.Background(), env.username)

	assert.Error(t, err)
	assert.False(t, allowed)
	assert.Equal(t, 0, remaining)
	assert.Equal(t, 60, resetIn)
	assert.NoError(t, env.mock.ExpectationsWereMet())
}

func TestNewRedisClient_InvalidURL(t *testing.T) {
	cfg := &config.Config{
		RedisConnect: config.RedisConnect{
			Host: "invalid url with spaces",
		},
	}

	client, err := NewRedisClient(cfg)
	assert.Error(t, err)
	assert.Nil(t, client)
}

func TestNewRedisClient_PingFail(t *testing.T) {
	cfg := &config.Config{
		RedisConnect: config.RedisConnect{
			Host: "127.0.0.1",
			Port: "1", // unreachable port
		},
	}

	client, err := NewRedisClient(cfg)
	assert.Error(t, err)
	assert.Nil(t, client)
}
