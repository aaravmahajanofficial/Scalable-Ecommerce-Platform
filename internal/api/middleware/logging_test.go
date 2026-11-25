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

func TestLoggingMiddleware_PR194(t *testing.T) { //nolint:gocognit // table-driven middleware test
	// Setup custom logger to capture logs
	var logBuf bytes.Buffer
	logHandler := slog.NewJSONHandler(&logBuf, nil)
	logger := slog.New(logHandler)
	originalLogger := slog.Default()
	slog.SetDefault(logger)
	defer slog.SetDefault(originalLogger)

	tests := []struct {
		name               string
		existingRequestID  string
		handlerStatusCode  int
		expectedStatusCode int
	}{
		{
			name:               "Without existing X-Request-ID",
			existingRequestID:  "",
			handlerStatusCode:  http.StatusCreated,
			expectedStatusCode: http.StatusCreated,
		},
		{
			name:               "With existing X-Request-ID",
			existingRequestID:  "test-correlation-id-123",
			handlerStatusCode:  http.StatusOK,
			expectedStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logBuf.Reset()

			var contextLogger *slog.Logger
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

				contextLogger = middleware.LoggerFromContext(r.Context())
				w.WriteHeader(tt.handlerStatusCode)
				if _, err := w.Write([]byte("OK")); err != nil {
					panic(err)
				}
			})

			handler := middleware.Logging(nextHandler)

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test-path", http.NoBody)
			req.RemoteAddr = "192.168.1.1:12345"
			req.Header.Set("User-Agent", "Test-Agent")
			if tt.existingRequestID != "" {
				req.Header.Set("X-Request-ID", tt.existingRequestID)
			}

			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatusCode, rr.Code)

			resRequestID := rr.Header().Get("X-Request-ID")
			assert.NotEmpty(t, resRequestID)
			if tt.existingRequestID != "" {
				assert.Equal(t, tt.existingRequestID, resRequestID)
			} else {

				_, err := uuid.Parse(resRequestID)
				assert.NoError(t, err)
			}

			logOutput := logBuf.String()

			assert.NotNil(t, contextLogger)
			assert.NotEqual(t, slog.Default(), contextLogger, "context logger should be a derived logger, not default")

			assert.Contains(t, logOutput, "Incoming request")
			assert.Contains(t, logOutput, "Request Completed")
			assert.Contains(t, logOutput, `"http_method":"GET"`)
			assert.Contains(t, logOutput, `"http_path":"/test-path"`)
			assert.Contains(t, logOutput, `"remote_addr":"192.168.1.1:12345"`)
			assert.Contains(t, logOutput, `"user_agent":"Test-Agent"`)

			if tt.existingRequestID != "" {
				assert.Contains(t, logOutput, `"correlation_id":"test-correlation-id-123"`)
			} else {
				assert.Contains(t, logOutput, `"correlation_id":"`+resRequestID+`"`)
			}

			if tt.handlerStatusCode == http.StatusCreated {
				assert.Contains(t, logOutput, `"http_status":201`)
			} else {
				assert.Contains(t, logOutput, `"http_status":200`)
			}
		})
	}
}

func TestLoggerFromContext_PR194(t *testing.T) {

	ctx := context.Background()
	logger := middleware.LoggerFromContext(ctx)
	assert.Equal(t, slog.Default(), logger)

	expectedLogger := slog.Default().With("key", "value")
	ctxWithLogger := context.WithValue(ctx, middleware.LoggerKey, expectedLogger)
	loggerFromCtx := middleware.LoggerFromContext(ctxWithLogger)
	assert.Equal(t, expectedLogger, loggerFromCtx)
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	// Since responseWriter is private, we can only test it indirectly via middleware
	// However, its WriteHeader method is tested in TestLoggingMiddleware
	// We can also test default status code behavior
	var logBuf bytes.Buffer
	logHandler := slog.NewJSONHandler(&logBuf, nil)
	originalLogger := slog.Default()
	defer slog.SetDefault(originalLogger)
	slog.SetDefault(slog.New(logHandler))

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {

		if _, err := w.Write([]byte("OK")); err != nil {
			panic(err)
		}
	})

	handler := middleware.Logging(nextHandler)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/default-status", http.NoBody)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	logOutput := logBuf.String()
	assert.Contains(t, logOutput, `"http_status":200`)
}
