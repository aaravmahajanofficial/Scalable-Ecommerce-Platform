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
)

func TestLoggingMiddleware(t *testing.T) {
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
				// Capture context logger inside the handler
				contextLogger = middleware.LoggerFromContext(r.Context())
				w.WriteHeader(tt.handlerStatusCode)
				w.Write([]byte("OK"))
			})

			handler := middleware.Logging(nextHandler)

			req := httptest.NewRequest(http.MethodGet, "/test-path", nil)
			req.RemoteAddr = "192.168.1.1:12345"
			req.Header.Set("User-Agent", "Test-Agent")
			if tt.existingRequestID != "" {
				req.Header.Set("X-Request-ID", tt.existingRequestID)
			}

			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			// Assert Response Status Code
			assert.Equal(t, tt.expectedStatusCode, rr.Code)

			// Assert X-Request-ID Header
			resRequestID := rr.Header().Get("X-Request-ID")
			assert.NotEmpty(t, resRequestID)
			if tt.existingRequestID != "" {
				assert.Equal(t, tt.existingRequestID, resRequestID)
			} else {
				// Should be a valid UUID
				_, err := uuid.Parse(resRequestID)
				assert.NoError(t, err)
			}

			// Check logs
			logOutput := logBuf.String()

			// Verify context logger was set
			assert.NotNil(t, contextLogger)
			assert.NotEqual(t, slog.Default(), contextLogger, "context logger should be a derived logger, not default")

			// Check if logs contain expected fields
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

			// Check status code in log
			// We can't simply check for `"http_status":201` directly because we need to parse JSON to be robust,
			// but for this simple test, string check might be enough if formatted properly.
			// Let's do a simple string check.
			if tt.handlerStatusCode == http.StatusCreated {
				assert.Contains(t, logOutput, `"http_status":201`)
			} else {
				assert.Contains(t, logOutput, `"http_status":200`)
			}
		})
	}
}

func TestLoggerFromContext(t *testing.T) {
	// Case 1: Context without logger
	ctx := context.Background()
	logger := middleware.LoggerFromContext(ctx)
	assert.Equal(t, slog.Default(), logger)

	// Case 2: Context with logger
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

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do not write header explicitly
		w.Write([]byte("OK"))
	})

	handler := middleware.Logging(nextHandler)
	req := httptest.NewRequest(http.MethodGet, "/default-status", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	logOutput := logBuf.String()
	assert.Contains(t, logOutput, `"http_status":200`)
}
