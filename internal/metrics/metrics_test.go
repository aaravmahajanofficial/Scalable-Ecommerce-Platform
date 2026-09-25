package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler(t *testing.T) {
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

func TestMiddleware(t *testing.T) {
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

			// httpRequestsInFlight should be decremented back after handler completes
			afterInFlight := testutil.ToFloat64(httpRequestsInFlight)
			assert.Equal(t, inFlightDuringHandler-1, afterInFlight)
		})
	}
}
