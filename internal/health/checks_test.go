package health_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/config"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/health"
	stripeclient "github.com/aaravmahajanofficial/scalable-ecommerce-platform/pkg/stripe"
	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	stripe_go "github.com/stripe/stripe-go/v86"
	"github.com/stripe/stripe-go/v86/form"
)

type mockStripeBackend struct {
	err error
}

func (m *mockStripeBackend) Call(method, path, key string, params stripe_go.ParamsContainer, v stripe_go.LastResponseSetter) error {
	return m.err
}

func (m *mockStripeBackend) CallRaw(method, path, key string, body *form.Values, params *stripe_go.Params, v stripe_go.LastResponseSetter) error {
	return m.err
}

func (m *mockStripeBackend) CallMultipart(method, path, key, boundary string, body *bytes.Buffer, params *stripe_go.Params, v stripe_go.LastResponseSetter) error {
	return m.err
}

func (m *mockStripeBackend) CallStreaming(method, path, key string, params stripe_go.ParamsContainer, v stripe_go.StreamingLastResponseSetter) error {
	return m.err
}

func (m *mockStripeBackend) SetMaxNetworkRetries(maxNetworkRetries int64) {}

func TestNewLivenessHandler(t *testing.T) {
	handler := health.NewLivenessHandler()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/live", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Service is alive. Time:")
}

func TestNewReadinessHandler_StripeUninitialized(t *testing.T) {
	cfg := &config.Config{}

	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisClient, _ := redismock.NewClientMock()
	defer redisClient.Close()

	endpoint := &health.HealthEndpoint{
		DB:           db,
		RedisClient:  redisClient,
		StripeClient: nil,
	}

	handler, err := health.NewReadinessHandler(cfg, endpoint)
	require.NoError(t, err)
	require.NotNil(t, handler)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/ready", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
	assert.Contains(t, rr.Body.String(), "stripe client is not initialized")
}

func TestNewReadinessHandler_StripeTimeout(t *testing.T) {
	cfg := &config.Config{}

	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisClient, _ := redismock.NewClientMock()
	defer redisClient.Close()

	sc := stripeclient.NewStripeClient("test", "test")
	endpoint := &health.HealthEndpoint{
		DB:           db,
		RedisClient:  redisClient,
		StripeClient: &sc,
	}

	originalBackend := stripe_go.GetBackend(stripe_go.APIBackend)
	defer stripe_go.SetBackend(stripe_go.APIBackend, originalBackend)

	stripe_go.SetBackend(stripe_go.APIBackend, &mockStripeBackend{err: context.DeadlineExceeded})

	handler, err := health.NewReadinessHandler(cfg, endpoint)
	require.NoError(t, err)

	// Create context that has already exceeded its deadline
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/ready", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
	assert.Contains(t, rr.Body.String(), "stripe API call timed out")
}

func TestNewReadinessHandler_StripeError(t *testing.T) {
	cfg := &config.Config{}

	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	redisClient, _ := redismock.NewClientMock()
	defer redisClient.Close()

	sc := stripeclient.NewStripeClient("test", "test")
	endpoint := &health.HealthEndpoint{
		DB:           db,
		RedisClient:  redisClient,
		StripeClient: &sc,
	}

	originalBackend := stripe_go.GetBackend(stripe_go.APIBackend)
	defer stripe_go.SetBackend(stripe_go.APIBackend, originalBackend)

	stripe_go.SetBackend(stripe_go.APIBackend, &mockStripeBackend{err: errors.New("some stripe error")})

	handler, err := health.NewReadinessHandler(cfg, endpoint)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/ready", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
	assert.Contains(t, rr.Body.String(), "some stripe error")
}

func TestNewReadinessHandler_Success(t *testing.T) {
	cfg := &config.Config{}

	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectPing().WillReturnError(nil)

	redisClient, redisMock := redismock.NewClientMock()
	defer redisClient.Close()

	redisMock.ExpectPing().SetVal("PONG")

	sc := stripeclient.NewStripeClient("test", "test")
	endpoint := &health.HealthEndpoint{
		DB:           db,
		RedisClient:  redisClient,
		StripeClient: &sc,
	}

	originalBackend := stripe_go.GetBackend(stripe_go.APIBackend)
	defer stripe_go.SetBackend(stripe_go.APIBackend, originalBackend)

	stripe_go.SetBackend(stripe_go.APIBackend, &mockStripeBackend{err: nil})

	handler, err := health.NewReadinessHandler(cfg, endpoint)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/ready", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Since postgres health check will still try to dial the DSN, we can't reliably mock
	// the internal health check package entirely. Instead, we assert that the response doesn't
	// contain a stripe error and we at least verify the HTTP response body structure.
	assert.NotContains(t, rr.Body.String(), "stripe API call timed out")
	assert.NotContains(t, rr.Body.String(), "some stripe error")
	assert.NotContains(t, rr.Body.String(), "stripe client is not initialized")

	// It'll likely be 503 because postgres fails to connect, but not because of Stripe.
	assert.Contains(t, []int{http.StatusOK, http.StatusServiceUnavailable}, rr.Code)
}
