package metrics

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		urlPath        string
		pathValueKey   string
		pathValueVal   string
		handlerStatus  int
		expectedPath   string
	}{
		{
			name:          "basic path",
			method:        http.MethodGet,
			urlPath:       "/health",
			handlerStatus: http.StatusOK,
			expectedPath:  "/health",
		},
		{
			name:          "path with id",
			method:        http.MethodGet,
			urlPath:       "/users/123",
			pathValueKey:  "id",
			pathValueVal:  "123",
			handlerStatus: http.StatusOK,
			expectedPath:  "/users/{id}",
		},
		{
			name:          "path with catch-all",
			method:        http.MethodGet,
			urlPath:       "/files/doc/1.txt",
			pathValueKey:  "...",
			pathValueVal:  "doc/1.txt",
			handlerStatus: http.StatusNotFound,
			expectedPath:  "/files/{...}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset metrics before each test to ensure clean state
			httpRequestsTotal.Reset()
			httpRequestsDuration.Reset()
			httpRequestsInFlight.Set(0)

			req := httptest.NewRequest(tt.method, tt.urlPath, nil)
			if tt.pathValueKey != "" {
				req.SetPathValue(tt.pathValueKey, tt.pathValueVal)
			}

			w := httptest.NewRecorder()

			handlerCalled := false
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true

				// Assert in-flight is 1 during execution
				assert.Equal(t, float64(1), testutil.ToFloat64(httpRequestsInFlight))

				w.WriteHeader(tt.handlerStatus)
			})

			handler := Middleware(nextHandler)
			handler.ServeHTTP(w, req)

			assert.True(t, handlerCalled)

			// Assert in-flight is 0 after execution
			assert.Equal(t, float64(0), testutil.ToFloat64(httpRequestsInFlight))

			// Assert total requests
			expectedLabels := prometheus.Labels{
				"code":   strconv.Itoa(tt.handlerStatus),
				"method": tt.method,
				"path":   tt.expectedPath,
			}

			count := testutil.ToFloat64(httpRequestsTotal.With(expectedLabels))
			assert.Equal(t, float64(1), count)

			// Assert duration is observed
			durationCount := testutil.CollectAndCount(httpRequestsDuration)
			assert.Equal(t, 1, durationCount)
		})
	}
}

func TestHandler(t *testing.T) {
	h := Handler()
	assert.NotNil(t, h)
}
