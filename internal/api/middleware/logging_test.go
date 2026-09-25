package middleware_test

import (
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
