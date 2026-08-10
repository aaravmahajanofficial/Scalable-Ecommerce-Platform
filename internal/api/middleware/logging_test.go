package middleware_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/api/middleware"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoggingMiddleware(t *testing.T) {
	tests := []struct {
		name          string
		reqIDHeader   string
		expectedCode  int
		expectedBody  string
	}{
		{
			name:         "Without X-Request-ID Header",
			reqIDHeader:  "",
			expectedCode: http.StatusCreated,
			expectedBody: `{"success": true}`,
		},
		{
			name:         "With X-Request-ID Header",
			reqIDHeader:  "test-correlation-id-123",
			expectedCode: http.StatusOK,
			expectedBody: `{"success": true}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Mock handler to be called by middleware
			mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Assert that the logger is present in the context
				logger := middleware.LoggerFromContext(r.Context())
				require.NotNil(t, logger, "Logger should be present in context")

				// Assert that X-Request-ID is in the context logger and response header
				reqID := w.Header().Get("X-Request-ID")
				assert.NotEmpty(t, reqID, "X-Request-ID should be set in response header")

				if tc.reqIDHeader != "" {
					assert.Equal(t, tc.reqIDHeader, reqID, "X-Request-ID should match the provided header")
				} else {
					// Verify it's a valid UUID if generated
					_, err := uuid.Parse(reqID)
					assert.NoError(t, err, "Generated X-Request-ID should be a valid UUID")
				}

				w.WriteHeader(tc.expectedCode)
				_, err := w.Write([]byte(tc.expectedBody))
				require.NoError(t, err)
			})

			// Set up request
			req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
			if tc.reqIDHeader != "" {
				req.Header.Set("X-Request-ID", tc.reqIDHeader)
			}

			rr := httptest.NewRecorder()

			// Create a buffer to capture logs if we want to ensure no panics during logging
			var logBuf bytes.Buffer
			handler := slog.NewJSONHandler(&logBuf, nil)
			logger := slog.New(handler)

			// Cache original default logger and restore it after the test
			originalLogger := slog.Default()
			slog.SetDefault(logger)
			t.Cleanup(func() { slog.SetDefault(originalLogger) })

			// Execute middleware
			middlewareToTest := middleware.Logging(mockHandler)
			middlewareToTest.ServeHTTP(rr, req)

			// Assertions on response
			assert.Equal(t, tc.expectedCode, rr.Code, "Unexpected status code")
			assert.JSONEq(t, tc.expectedBody, rr.Body.String(), "Unexpected response body")

			// Check that X-Request-ID is set in response headers
			respReqID := rr.Header().Get("X-Request-ID")
			assert.NotEmpty(t, respReqID, "X-Request-ID should be set in final response header")

			if tc.reqIDHeader != "" {
				assert.Equal(t, tc.reqIDHeader, respReqID, "Response X-Request-ID should match the one provided in request")
			}

			// Validate logs were written
			logOutput := logBuf.String()
			assert.Contains(t, logOutput, "Incoming request", "Should log incoming request")
			assert.Contains(t, logOutput, "Request Completed", "Should log completed request")
			if tc.reqIDHeader != "" {
				assert.Contains(t, logOutput, tc.reqIDHeader, "Logs should contain correlation_id")
			}
		})
	}
}

func TestLoggerFromContext(t *testing.T) {
	t.Run("Context Without Logger", func(t *testing.T) {
		ctx := context.Background()
		logger := middleware.LoggerFromContext(ctx)

		// It should return the default logger
		assert.NotNil(t, logger, "Should return a logger even if not in context")
		assert.Equal(t, slog.Default(), logger, "Should fall back to default logger")
	})

	t.Run("Context With Logger", func(t *testing.T) {
		// Create a custom logger
		var buf bytes.Buffer
		customLogger := slog.New(slog.NewJSONHandler(&buf, nil))

		ctx := context.WithValue(context.Background(), middleware.LoggerKey, customLogger)
		logger := middleware.LoggerFromContext(ctx)

		assert.NotNil(t, logger, "Should return a logger")
		assert.Equal(t, customLogger, logger, "Should return the logger stored in context")
	})
}
