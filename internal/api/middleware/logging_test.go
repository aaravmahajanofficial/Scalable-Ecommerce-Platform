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
	t.Run("With existing X-Request-ID header", func(t *testing.T) {
		customID := "test-request-id-123"
		var capturedLogger *slog.Logger

		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedLogger = middleware.LoggerFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		})

		handler := middleware.Logging(nextHandler)

		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test-path", http.NoBody)
		req.Header.Set("X-Request-ID", customID)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, customID, rr.Header().Get("X-Request-ID"))
		assert.NotNil(t, capturedLogger)
	})

	t.Run("Without X-Request-ID header generates UUID", func(t *testing.T) {
		var capturedLogger *slog.Logger

		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedLogger = middleware.LoggerFromContext(r.Context())
			w.WriteHeader(http.StatusCreated)
		})

		handler := middleware.Logging(nextHandler)

		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/create", http.NoBody)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)

		requestID := rr.Header().Get("X-Request-ID")
		require.NotEmpty(t, requestID)
		_, err := uuid.Parse(requestID)
		assert.NoError(t, err, "Generated request ID should be a valid UUID")
		assert.NotNil(t, capturedLogger)
	})

	t.Run("Captures response status code", func(t *testing.T) {
		testCases := []struct {
			name           string
			handlerStatus  int
			expectedStatus int
		}{
			{
				name:           "Default 200 OK when WriteHeader is not called",
				handlerStatus:  0,
				expectedStatus: http.StatusOK,
			},
			{
				name:           "Explicit status code 400 Bad Request",
				handlerStatus:  http.StatusBadRequest,
				expectedStatus: http.StatusBadRequest,
			},
			{
				name:           "Explicit status code 500 Internal Server Error",
				handlerStatus:  http.StatusInternalServerError,
				expectedStatus: http.StatusInternalServerError,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				nextHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					if tc.handlerStatus != 0 {
						w.WriteHeader(tc.handlerStatus)
					}
					_, err := w.Write([]byte("response"))
					require.NoError(t, err)
				})

				handler := middleware.Logging(nextHandler)

				req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/status-check", http.NoBody)
				rr := httptest.NewRecorder()

				handler.ServeHTTP(rr, req)

				assert.Equal(t, tc.expectedStatus, rr.Code)
			})
		}
	})
}

func TestLoggerFromContext(t *testing.T) {
	t.Run("Returns logger from context when present", func(t *testing.T) {
		expectedLogger := slog.Default()
		ctx := context.WithValue(context.Background(), middleware.LoggerKey, expectedLogger)

		logger := middleware.LoggerFromContext(ctx)
		assert.Equal(t, expectedLogger, logger)
	})

	t.Run("Returns default logger when logger key is missing in context", func(t *testing.T) {
		ctx := context.Background()

		logger := middleware.LoggerFromContext(ctx)
		assert.Equal(t, slog.Default(), logger)
	})

	t.Run("Returns default logger when context value is not *slog.Logger", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), middleware.LoggerKey, "not-a-logger")

		logger := middleware.LoggerFromContext(ctx)
		assert.Equal(t, slog.Default(), logger)
	})
}

func TestLoggingMiddleware_PR210(t *testing.T) {
	tests := []struct {
		name         string
		reqIDHeader  string
		expectedCode int
		expectedBody string
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

			mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

				logger := middleware.LoggerFromContext(r.Context())
				require.NotNil(t, logger, "Logger should be present in context")

				reqID := w.Header().Get("X-Request-ID")
				assert.NotEmpty(t, reqID, "X-Request-ID should be set in response header")

				if tc.reqIDHeader != "" {
					assert.Equal(t, tc.reqIDHeader, reqID, "X-Request-ID should match the provided header")
				} else {

					_, err := uuid.Parse(reqID)
					assert.NoError(t, err, "Generated X-Request-ID should be a valid UUID")
				}

				w.WriteHeader(tc.expectedCode)
				_, err := w.Write([]byte(tc.expectedBody))
				require.NoError(t, err)
			})

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", http.NoBody)
			if tc.reqIDHeader != "" {
				req.Header.Set("X-Request-ID", tc.reqIDHeader)
			}

			rr := httptest.NewRecorder()

			// Create a buffer to capture logs if we want to ensure no panics during logging
			var logBuf bytes.Buffer
			handler := slog.NewJSONHandler(&logBuf, nil)
			logger := slog.New(handler)

			originalLogger := slog.Default()
			slog.SetDefault(logger)
			t.Cleanup(func() { slog.SetDefault(originalLogger) })

			middlewareToTest := middleware.Logging(mockHandler)
			middlewareToTest.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedCode, rr.Code, "Unexpected status code")
			assert.JSONEq(t, tc.expectedBody, rr.Body.String(), "Unexpected response body")

			respReqID := rr.Header().Get("X-Request-ID")
			assert.NotEmpty(t, respReqID, "X-Request-ID should be set in final response header")

			if tc.reqIDHeader != "" {
				assert.Equal(t, tc.reqIDHeader, respReqID, "Response X-Request-ID should match the one provided in request")
			}

			logOutput := logBuf.String()
			assert.Contains(t, logOutput, "Incoming request", "Should log incoming request")
			assert.Contains(t, logOutput, "Request Completed", "Should log completed request")
			if tc.reqIDHeader != "" {
				assert.Contains(t, logOutput, tc.reqIDHeader, "Logs should contain correlation_id")
			}
		})
	}
}

func TestLoggerFromContext_PR210(t *testing.T) {
	t.Run("Context Without Logger", func(t *testing.T) {
		ctx := context.Background()
		logger := middleware.LoggerFromContext(ctx)

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
