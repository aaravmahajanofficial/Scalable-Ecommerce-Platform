package metrics

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

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

func TestHandler_PR271(t *testing.T) {
	handler := Handler()
	require.NotNil(t, handler)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/metrics", http.NoBody)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "http_requests_in_flight")
}

func TestResponseWriter_DefaultStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := newResponseWriter(rec)

	assert.Equal(t, http.StatusOK, rw.statusCode)

	_, err := rw.Write([]byte("ok"))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rw.statusCode)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestResponseWriter_ExplicitWriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := newResponseWriter(rec)

	rw.WriteHeader(http.StatusCreated)
	assert.Equal(t, http.StatusCreated, rw.statusCode)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestMiddleware_PR271(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		urlPath        string
		pathValueKey   string
		pathValueVal   string
		handlerStatus  int
		handlerBody    string
		expectedPath   string
		expectedStatus int
	}{
		{
			name:           "Standard Request with Default 200 OK",
			method:         http.MethodGet,
			urlPath:        "/api/v1/products",
			handlerStatus:  http.StatusOK,
			handlerBody:    "products list",
			expectedPath:   "/api/v1/products",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Request with 201 Created Status",
			method:         http.MethodPost,
			urlPath:        "/api/v1/orders",
			handlerStatus:  http.StatusCreated,
			handlerBody:    "order created",
			expectedPath:   "/api/v1/orders",
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "Path Parameter id",
			method:         http.MethodGet,
			urlPath:        "/api/v1/products/123",
			pathValueKey:   "id",
			pathValueVal:   "123",
			handlerStatus:  http.StatusOK,
			handlerBody:    "product 123",
			expectedPath:   "/api/v1/products/{id}",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Wildcard Path Parameter ...",
			method:         http.MethodGet,
			urlPath:        "/static/css/main.css",
			pathValueKey:   "...",
			pathValueVal:   "css/main.css",
			handlerStatus:  http.StatusOK,
			handlerBody:    "css content",
			expectedPath:   "/static/{...}",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inFlightDuringHandler := float64(-1)

			dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				inFlightDuringHandler = testutil.ToFloat64(httpRequestsInFlight)
				if tt.handlerStatus != http.StatusOK {
					w.WriteHeader(tt.handlerStatus)
				}
				_, writeErr := w.Write([]byte(tt.handlerBody))
				assert.NoError(t, writeErr)
			})

			wrappedHandler := Middleware(dummyHandler)

			req, err := http.NewRequestWithContext(context.Background(), tt.method, tt.urlPath, http.NoBody)
			require.NoError(t, err)

			if tt.pathValueKey != "" {
				req.SetPathValue(tt.pathValueKey, tt.pathValueVal)
			}

			statusStr := strconv.Itoa(tt.expectedStatus)
			initialCount := testutil.ToFloat64(httpRequestsTotal.WithLabelValues(statusStr, tt.method, tt.expectedPath))

			rr := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			assert.Equal(t, tt.handlerBody, rr.Body.String())
			assert.GreaterOrEqual(t, inFlightDuringHandler, float64(1))

			finalCount := testutil.ToFloat64(httpRequestsTotal.WithLabelValues(statusStr, tt.method, tt.expectedPath))
			assert.Equal(t, initialCount+1, finalCount)

			afterInFlight := testutil.ToFloat64(httpRequestsInFlight)
			assert.Equal(t, inFlightDuringHandler-1, afterInFlight)
		})
	}
}
