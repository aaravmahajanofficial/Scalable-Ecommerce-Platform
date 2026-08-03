package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"bytes"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/config"
	stripeClient "github.com/aaravmahajanofficial/scalable-ecommerce-platform/pkg/stripe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/form"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-redis/redismock/v9"
)

// Minimal mock for stripe backend that avoids code duplication with other mocks.
type StripeBackendMock struct {
	mock.Mock
}

func (m *StripeBackendMock) Call(method, path, key string, params stripe.ParamsContainer, v stripe.LastResponseSetter) error {
	return m.Called(method, path, key, params, v).Error(0)
}
func (m *StripeBackendMock) CallRaw(method, path, key string, body *form.Values, params *stripe.Params, v stripe.LastResponseSetter) error { return nil }
func (m *StripeBackendMock) CallMultipart(method, path, key, boundary string, body *bytes.Buffer, params *stripe.Params, v stripe.LastResponseSetter) error { return nil }
func (m *StripeBackendMock) CallStreaming(method, path, key string, params stripe.ParamsContainer, v stripe.StreamingLastResponseSetter) error { return nil }
func (m *StripeBackendMock) SetMaxNetworkRetries(maxNetworkRetries int64) {}

func TestNewLivenessHandler(t *testing.T) {
	handler := NewLivenessHandler()

	req := httptest.NewRequest(http.MethodGet, "/live", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Service is alive")
}

func setupMocks(t *testing.T, stripeErr error) (*HealthEndpoint, sqlmock.Sqlmock, redismock.ClientMock) {
	t.Helper()

	db, sqlMock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	assert.NoError(t, err)
	sqlMock.ExpectPing().WillReturnError(nil)

	redisClient, redisMock := redismock.NewClientMock()
	redisMock.ExpectPing().SetVal("PONG")

	var sc stripeClient.Client

	endpoint := &HealthEndpoint{
		DB:           db,
		RedisClient:  redisClient,
		StripeClient: &sc,
	}

	mockBackend := new(StripeBackendMock)
	mockBackend.On("Call", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(stripeErr)

	originalBackend := stripe.GetBackend(stripe.APIBackend)
	t.Cleanup(func() {
		stripe.SetBackend(stripe.APIBackend, originalBackend)
		db.Close()
	})
	stripe.SetBackend(stripe.APIBackend, mockBackend)

	return endpoint, sqlMock, redisMock
}

func TestNewReadinessHandler(t *testing.T) {
	cfg := &config.Config{
		OTel: config.OTelConfig{
			ServiceName: "test-service",
		},
	}

	t.Run("Success", func(t *testing.T) {
		endpoint, sqlMock, redisMock := setupMocks(t, nil)

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
		endpoint, _, _ := setupMocks(t, errors.New("stripe error"))

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
		endpoint, _, _ := setupMocks(t, context.DeadlineExceeded)

		handler, err := NewReadinessHandler(cfg, endpoint)

		assert.NoError(t, err)
		assert.NotNil(t, handler)

		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
	})
}
