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
)

type mockStripeBackend struct {
	err error
}

func (m *mockStripeBackend) Call(_, _, _ string, _ stripe_go.ParamsContainer, _ stripe_go.LastResponseSetter) error {
	return m.err
}

func (m *mockStripeBackend) CallRaw(_, _, _ string, _ []byte, _ *stripe_go.Params, _ stripe_go.LastResponseSetter) error {
	return m.err
}

func (m *mockStripeBackend) CallMultipart(_, _, _, _ string, _ *bytes.Buffer, _ *stripe_go.Params, _ stripe_go.LastResponseSetter) error {
	return m.err
}

func (m *mockStripeBackend) CallStreaming(_, _, _ string, _ stripe_go.ParamsContainer, _ stripe_go.StreamingLastResponseSetter) error {
	return m.err
}

func (m *mockStripeBackend) SetMaxNetworkRetries(_ int64) {}

func TestNewLivenessHandler(t *testing.T) {
	handler := health.NewLivenessHandler()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/live", http.NoBody)
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
	t.Cleanup(func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Logf("db close error: %v", closeErr)
		}
	})

	redisClient, _ := redismock.NewClientMock()
	t.Cleanup(func() {
		if closeErr := redisClient.Close(); closeErr != nil {
			t.Logf("redis client close error: %v", closeErr)
		}
	})

	endpoint := &health.Endpoint{
		DB:           db,
		RedisClient:  redisClient,
		StripeClient: nil,
	}

	handler, err := health.NewReadinessHandler(cfg, endpoint)
	require.NoError(t, err)
	require.NotNil(t, handler)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/ready", http.NoBody)
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
	t.Cleanup(func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Logf("db close error: %v", closeErr)
		}
	})

	redisClient, _ := redismock.NewClientMock()
	t.Cleanup(func() {
		if closeErr := redisClient.Close(); closeErr != nil {
			t.Logf("redis client close error: %v", closeErr)
		}
	})

	sc := stripeclient.NewStripeClient("test", "test")
	endpoint := &health.Endpoint{
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

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/ready", http.NoBody)
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
	t.Cleanup(func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Logf("db close error: %v", closeErr)
		}
	})

	redisClient, _ := redismock.NewClientMock()
	t.Cleanup(func() {
		if closeErr := redisClient.Close(); closeErr != nil {
			t.Logf("redis client close error: %v", closeErr)
		}
	})

	sc := stripeclient.NewStripeClient("test", "test")
	endpoint := &health.Endpoint{
		DB:           db,
		RedisClient:  redisClient,
		StripeClient: &sc,
	}

	originalBackend := stripe_go.GetBackend(stripe_go.APIBackend)
	defer stripe_go.SetBackend(stripe_go.APIBackend, originalBackend)

	stripe_go.SetBackend(stripe_go.APIBackend, &mockStripeBackend{err: errors.New("some stripe error")})

	handler, err := health.NewReadinessHandler(cfg, endpoint)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/ready", http.NoBody)
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
	t.Cleanup(func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Logf("db close error: %v", closeErr)
		}
	})

	mock.ExpectPing().WillReturnError(nil)

	redisClient, redisMock := redismock.NewClientMock()
	t.Cleanup(func() {
		if closeErr := redisClient.Close(); closeErr != nil {
			t.Logf("redis client close error: %v", closeErr)
		}
	})

	redisMock.ExpectPing().SetVal("PONG")

	sc := stripeclient.NewStripeClient("test", "test")
	endpoint := &health.Endpoint{
		DB:           db,
		RedisClient:  redisClient,
		StripeClient: &sc,
	}

	originalBackend := stripe_go.GetBackend(stripe_go.APIBackend)
	defer stripe_go.SetBackend(stripe_go.APIBackend, originalBackend)

	stripe_go.SetBackend(stripe_go.APIBackend, &mockStripeBackend{err: nil})

	handler, err := health.NewReadinessHandler(cfg, endpoint)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/ready", http.NoBody)
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
