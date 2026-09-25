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

func TestCheckLoginRateLimit_Success(t *testing.T) {
	client, mock := redismock.NewClientMock()
	cfg := &config.Config{
		RateConfig: config.RateConfig{
			MaxAttempts: 5,
			WindowSize:  60 * time.Second,
		},
	}

	repo := NewRateLimitRepo(client, cfg)
	username := "testuser"
	key := "login_attempts:" + username

	now := time.Now().Unix()

	mock.Regexp().ExpectZRemRangeByScore(key, "0", ".*").SetVal(0)
	mock.ExpectZAdd(key, redis.Z{Score: float64(now), Member: now}).SetVal(1)
	mock.ExpectZCard(key).SetVal(2)
	mock.ExpectExpire(key, 60*time.Second).SetVal(true)

	allowed, remaining, resetIn, err := repo.CheckLoginRateLimit(context.Background(), username)

	assert.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, 3, remaining)
	assert.Equal(t, 0, resetIn)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckLoginRateLimit_Exceeded(t *testing.T) {
	client, mock := redismock.NewClientMock()
	cfg := &config.Config{
		RateConfig: config.RateConfig{
			MaxAttempts: 5,
			WindowSize:  60 * time.Second,
		},
	}

	repo := NewRateLimitRepo(client, cfg)
	username := "testuser"
	key := "login_attempts:" + username
	now := time.Now().Unix()

	mock.Regexp().ExpectZRemRangeByScore(key, "0", ".*").SetVal(0)
	mock.ExpectZAdd(key, redis.Z{Score: float64(now), Member: now}).SetVal(1)
	mock.ExpectZCard(key).SetVal(5)
	mock.ExpectExpire(key, 60*time.Second).SetVal(true)

	mock.ExpectZRangeArgsWithScores(redis.ZRangeArgs{
		Key: key, Start: 0, Stop: 0,
	}).SetVal([]redis.Z{
		{Score: float64(now - 10), Member: now - 10},
	})

	allowed, remaining, resetIn, err := repo.CheckLoginRateLimit(context.Background(), username)

	assert.NoError(t, err)
	assert.False(t, allowed)
	assert.Equal(t, 0, remaining)
	assert.True(t, resetIn > 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckLoginRateLimit_PipelineError(t *testing.T) {
	client, mock := redismock.NewClientMock()
	cfg := &config.Config{
		RateConfig: config.RateConfig{
			MaxAttempts: 5,
			WindowSize:  60 * time.Second,
		},
	}

	repo := NewRateLimitRepo(client, cfg)
	username := "testuser"
	key := "login_attempts:" + username

	mock.Regexp().ExpectZRemRangeByScore(key, "0", ".*").SetErr(errors.New("redis err"))

	allowed, remaining, resetIn, err := repo.CheckLoginRateLimit(context.Background(), username)

	assert.Error(t, err)
	assert.False(t, allowed)
	assert.Equal(t, 0, remaining)
	assert.Equal(t, 0, resetIn)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckLoginRateLimit_OldestAttemptError(t *testing.T) {
	client, mock := redismock.NewClientMock()
	cfg := &config.Config{
		RateConfig: config.RateConfig{
			MaxAttempts: 5,
			WindowSize:  60 * time.Second,
		},
	}

	repo := NewRateLimitRepo(client, cfg)
	username := "testuser"
	key := "login_attempts:" + username
	now := time.Now().Unix()

	mock.Regexp().ExpectZRemRangeByScore(key, "0", ".*").SetVal(0)
	mock.ExpectZAdd(key, redis.Z{Score: float64(now), Member: now}).SetVal(1)
	mock.ExpectZCard(key).SetVal(5)
	mock.ExpectExpire(key, 60*time.Second).SetVal(true)

	mock.ExpectZRangeArgsWithScores(redis.ZRangeArgs{
		Key: key, Start: 0, Stop: 0,
	}).SetErr(errors.New("zrange error"))

	allowed, remaining, resetIn, err := repo.CheckLoginRateLimit(context.Background(), username)

	assert.Error(t, err)
	assert.False(t, allowed)
	assert.Equal(t, 0, remaining)
	assert.Equal(t, 60, resetIn)
	assert.NoError(t, mock.ExpectationsWereMet())
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
