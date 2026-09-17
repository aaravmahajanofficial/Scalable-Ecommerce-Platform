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
	t.Run("Existing X-Request-ID Header", func(t *testing.T) {
		existingID := "test-request-id-12345"
		var contextLogger *slog.Logger

		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			contextLogger = middleware.LoggerFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/api/v1/resource", nil)
		req.Header.Set("X-Request-ID", existingID)
		rr := httptest.NewRecorder()

		handler := middleware.Logging(nextHandler)
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, existingID, rr.Header().Get("X-Request-ID"))
		assert.NotNil(t, contextLogger)
	})

	t.Run("Missing X-Request-ID Header", func(t *testing.T) {
		var contextLogger *slog.Logger

		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			contextLogger = middleware.LoggerFromContext(r.Context())
		})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", nil)
		rr := httptest.NewRecorder()

		handler := middleware.Logging(nextHandler)
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		responseCorrelationID := rr.Header().Get("X-Request-ID")
		require.NotEmpty(t, responseCorrelationID)

		_, err := uuid.Parse(responseCorrelationID)
		assert.NoError(t, err, "Generated X-Request-ID should be a valid UUID")
		assert.NotNil(t, contextLogger)
	})

	t.Run("Custom Status Code Propagation", func(t *testing.T) {
		nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, err := w.Write([]byte(`{"created": true}`))
			require.NoError(t, err)
		})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/users", nil)
		rr := httptest.NewRecorder()

		handler := middleware.Logging(nextHandler)
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		assert.JSONEq(t, `{"created": true}`, rr.Body.String())
	})
}

func TestLoggerFromContext(t *testing.T) {
	t.Run("Valid Logger in Context", func(t *testing.T) {
		customLogger := slog.Default().With(slog.String("test_key", "test_val"))
		ctx := context.WithValue(context.Background(), middleware.LoggerKey, customLogger)

		logger := middleware.LoggerFromContext(ctx)
		assert.Equal(t, customLogger, logger)
	})

	t.Run("Missing Logger in Context", func(t *testing.T) {
		ctx := context.Background()

		logger := middleware.LoggerFromContext(ctx)
		assert.Equal(t, slog.Default(), logger)
	})

	t.Run("Invalid Logger Type in Context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), middleware.LoggerKey, "not-a-logger")

		logger := middleware.LoggerFromContext(ctx)
		assert.Equal(t, slog.Default(), logger)
	})
}
