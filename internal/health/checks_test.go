package health

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"context"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/config"
	stripeClient "github.com/aaravmahajanofficial/scalable-ecommerce-platform/pkg/stripe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-redis/redismock/v9"
)

type MockStripeBackend struct {
	mock.Mock
}

func (m *MockStripeBackend) Call(method, path, key string, params stripe.ParamsContainer, v stripe.LastResponseSetter) error {
	args := m.Called(method, path, key, params, v)
	return args.Error(0)
}

func (m *MockStripeBackend) CallRaw(method, path, key string, body *form.Values, params *stripe.Params, v stripe.LastResponseSetter) error {
	args := m.Called(method, path, key, body, params, v)
	return args.Error(0)
}

func (m *MockStripeBackend) CallMultipart(method, path, key, boundary string, body *bytes.Buffer, params *stripe.Params, v stripe.LastResponseSetter) error {
	args := m.Called(method, path, key, boundary, body, params, v)
	return args.Error(0)
}

func (m *MockStripeBackend) CallStreaming(method, path, key string, params stripe.ParamsContainer, v stripe.StreamingLastResponseSetter) error {
	args := m.Called(method, path, key, params, v)
	return args.Error(0)
}

func (m *MockStripeBackend) SetMaxNetworkRetries(maxNetworkRetries int64) {
	m.Called(maxNetworkRetries)
}

func TestNewLivenessHandler(t *testing.T) {
	handler := NewLivenessHandler()

	req := httptest.NewRequest(http.MethodGet, "/live", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Service is alive")
}

func TestNewReadinessHandler(t *testing.T) {
	cfg := &config.Config{
		OTel: config.OTelConfig{
			ServiceName: "test-service",
		},
	}

	t.Run("Success", func(t *testing.T) {
		// Mock SQL Database
		db, sqlMock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		assert.NoError(t, err)
		defer db.Close()
		sqlMock.ExpectPing().WillReturnError(nil)

		// Mock Redis Client
		redisClient, redisMock := redismock.NewClientMock()
		redisMock.ExpectPing().SetVal("PONG")

		// Mock Stripe Client
		var sc stripeClient.Client

		endpoint := &HealthEndpoint{
			DB:           db,
			RedisClient:  redisClient,
			StripeClient: &sc,
		}

		mockBackend := new(MockStripeBackend)
		mockBackend.On("Call", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		// Set mock backend and ensure we restore original backend
		originalBackend := stripe.GetBackend(stripe.APIBackend)
		defer stripe.SetBackend(stripe.APIBackend, originalBackend)
		stripe.SetBackend(stripe.APIBackend, mockBackend)

		handler, err := NewReadinessHandler(cfg, endpoint)

		assert.NoError(t, err)
		assert.NotNil(t, handler)

		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.NoError(t, sqlMock.ExpectationsWereMet())
		assert.NoError(t, redisMock.ExpectationsWereMet())
	})

	t.Run("Failure - Missing Clients", func(t *testing.T) {
		endpoint := &HealthEndpoint{
			DB:           nil,
			RedisClient:  nil,
			StripeClient: nil,
		}

		handler, err := NewReadinessHandler(cfg, endpoint)

		assert.NoError(t, err)
		assert.NotNil(t, handler)

		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
		assert.Contains(t, rr.Body.String(), "database connection is not initialized")
		assert.Contains(t, rr.Body.String(), "redis client is not initialized")
		assert.Contains(t, rr.Body.String(), "stripe client is not initialized")
	})

	t.Run("Stripe API Connection Error", func(t *testing.T) {
		db, sqlMock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		assert.NoError(t, err)
		defer db.Close()
		sqlMock.ExpectPing().WillReturnError(nil)

		redisClient, redisMock := redismock.NewClientMock()
		redisMock.ExpectPing().SetVal("PONG")

		var sc stripeClient.Client
		endpoint := &HealthEndpoint{
			DB:           db,
			RedisClient:  redisClient,
			StripeClient: &sc,
		}

		mockBackend := new(MockStripeBackend)
		mockBackend.On("Call", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("stripe error"))

		originalBackend := stripe.GetBackend(stripe.APIBackend)
		defer stripe.SetBackend(stripe.APIBackend, originalBackend)
		stripe.SetBackend(stripe.APIBackend, mockBackend)

		handler, err := NewReadinessHandler(cfg, endpoint)

		assert.NoError(t, err)
		assert.NotNil(t, handler)

		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
		assert.Contains(t, rr.Body.String(), "failed to connect to stripe: stripe error")
	})

	t.Run("Stripe API Timeout Error", func(t *testing.T) {
		db, sqlMock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		assert.NoError(t, err)
		defer db.Close()
		sqlMock.ExpectPing().WillReturnError(nil)

		redisClient, redisMock := redismock.NewClientMock()
		redisMock.ExpectPing().SetVal("PONG")

		var sc stripeClient.Client
		endpoint := &HealthEndpoint{
			DB:           db,
			RedisClient:  redisClient,
			StripeClient: &sc,
		}

		mockBackend := new(MockStripeBackend)
		mockBackend.On("Call", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(context.DeadlineExceeded)

		originalBackend := stripe.GetBackend(stripe.APIBackend)
		defer stripe.SetBackend(stripe.APIBackend, originalBackend)
		stripe.SetBackend(stripe.APIBackend, mockBackend)

		handler, err := NewReadinessHandler(cfg, endpoint)

		assert.NoError(t, err)
		assert.NotNil(t, handler)

		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
	})
}
