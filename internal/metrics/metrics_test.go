package metrics

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

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

	req := httptest.NewRequest(http.MethodGet, "/test/path", http.NoBody)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "OK", rec.Body.String())

	// Test path pattern matching for id
	reqWithID := httptest.NewRequest(http.MethodGet, "/users/123", http.NoBody)
	reqWithID.SetPathValue("id", "123")
	recWithID := httptest.NewRecorder()

	handler.ServeHTTP(recWithID, reqWithID)
	assert.Equal(t, http.StatusOK, recWithID.Code)

	// Test path pattern matching for ...
	reqWithWildcard := httptest.NewRequest(http.MethodGet, "/files/a/b/c", http.NoBody)
	reqWithWildcard.SetPathValue("...", "a/b/c")
	recWithWildcard := httptest.NewRecorder()

	handler.ServeHTTP(recWithWildcard, reqWithWildcard)
	assert.Equal(t, http.StatusOK, recWithWildcard.Code)
}

func TestHandler(t *testing.T) {
	h := Handler()
	assert.NotNil(t, h)

	req := httptest.NewRequest(http.MethodGet, "/metrics", http.NoBody)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "http_requests_total")
}
