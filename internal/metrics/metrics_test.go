package metrics

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestRegisterCollector(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	origLogger := slog.Default()
	slog.SetDefault(logger)
	defer slog.SetDefault(origLogger)

	// Test case 1: Registering a collector for the first time
	collector1 := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_register_collector_total_1",
		Help: "Test counter 1",
	})
	registerCollector(collector1, "TestCollector1")

	// Test case 2: Registering the same collector again triggers AlreadyRegisteredError
	logBuf.Reset()
	registerCollector(collector1, "TestCollector1")
	assert.Contains(t, logBuf.String(), "TestCollector1 registration skipped (already registered)")
	assert.Contains(t, logBuf.String(), "level=DEBUG")

	// Test case 3: Registering a conflicting collector triggers error
	logBuf.Reset()
	collector1Duplicate := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_register_collector_total_1",
		Help: "Different help text causes registration error",
	})
	registerCollector(collector1Duplicate, "TestCollector1Duplicate")
	assert.True(t, logBuf.Len() > 0)
}

func TestMiddleware(t *testing.T) {
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))
		assert.NoError(t, err)
	}))

	ctx := context.Background()

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/test/path", http.NoBody)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "OK", rec.Body.String())

	// Test path pattern matching for id
	reqWithID := httptest.NewRequestWithContext(ctx, http.MethodGet, "/users/123", http.NoBody)
	reqWithID.SetPathValue("id", "123")
	recWithID := httptest.NewRecorder()

	handler.ServeHTTP(recWithID, reqWithID)
	assert.Equal(t, http.StatusOK, recWithID.Code)

	// Test path pattern matching for ...
	reqWithWildcard := httptest.NewRequestWithContext(ctx, http.MethodGet, "/files/a/b/c", http.NoBody)
	reqWithWildcard.SetPathValue("...", "a/b/c")
	recWithWildcard := httptest.NewRecorder()

	handler.ServeHTTP(recWithWildcard, reqWithWildcard)
	assert.Equal(t, http.StatusOK, recWithWildcard.Code)
}

func TestHandler(t *testing.T) {
	h := Handler()
	assert.NotNil(t, h)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/metrics", http.NoBody)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "http_requests_total")
}

func TestResponseWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := newResponseWriter(rec)

	assert.Equal(t, http.StatusOK, rw.statusCode)

	rw.WriteHeader(http.StatusCreated)
	assert.Equal(t, http.StatusCreated, rw.statusCode)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestMiddleware_RecordsMetrics(t *testing.T) {

	httpRequestsTotal.Reset()
	httpRequestsDuration.Reset()

	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test-path", http.NoBody)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusAccepted, rec.Code)

	counter, err := httpRequestsTotal.GetMetricWithLabelValues("202", "GET", "/test-path")
	assert.NoError(t, err)
	val := testutil.ToFloat64(counter)
	assert.Equal(t, float64(1), val)

	_, err = httpRequestsDuration.GetMetricWithLabelValues("GET", "/test-path")
	assert.NoError(t, err)

	count := testutil.CollectAndCount(httpRequestsDuration)
	assert.Greater(t, count, 0)
}

func TestMiddleware_PathPatterns(t *testing.T) {
	httpRequestsTotal.Reset()

	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/users/123", http.NoBody)

	req.SetPathValue("...", "123")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	counter, err := httpRequestsTotal.GetMetricWithLabelValues("200", "GET", "/api/v1/users/{...}")
	assert.NoError(t, err)
	val := testutil.ToFloat64(counter)
	assert.Equal(t, float64(1), val)
}

func TestMiddleware_InFlight(t *testing.T) {

	initialVal := testutil.ToFloat64(httpRequestsInFlight)

	var valInsideHandler float64

	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		valInsideHandler = testutil.ToFloat64(httpRequestsInFlight)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/in-flight", http.NoBody)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, initialVal+1, valInsideHandler)

	finalVal := testutil.ToFloat64(httpRequestsInFlight)
	assert.Equal(t, initialVal, finalVal)
}
