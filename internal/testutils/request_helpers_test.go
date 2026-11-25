package testutils_test

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/api/middleware"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/testutils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateTestRequestWithContext(t *testing.T) {
	testUserID := uuid.New()

	t.Run("creates request with claims, logger, body, and path params", func(t *testing.T) {
		method := http.MethodPost
		target := "/api/v1/products/123"
		bodyContent := `{"name":"Test Product"}`
		body := strings.NewReader(bodyContent)
		pathParams := map[string]string{
			"id":       "123",
			"category": "electronics",
		}

		req := testutils.CreateTestRequestWithContext(method, target, body, testUserID, pathParams)

		require.NotNil(t, req)
		assert.Equal(t, method, req.Method)
		assert.Equal(t, target, req.URL.Path)

		// Assert body
		readBody, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		assert.Equal(t, bodyContent, string(readBody))

		// Assert path parameters
		assert.Equal(t, "123", req.PathValue("id"))
		assert.Equal(t, "electronics", req.PathValue("category"))

		// Assert claims context value
		claimsVal := req.Context().Value(middleware.UserContextKey)
		require.NotNil(t, claimsVal)
		claims, ok := claimsVal.(*models.Claims)
		require.True(t, ok)
		assert.Equal(t, testUserID, claims.UserID)
		assert.Equal(t, "test@example.com", claims.Email)

		// Assert logger context value
		loggerVal := req.Context().Value(middleware.LoggerKey)
		require.NotNil(t, loggerVal)
		logger, ok := loggerVal.(*slog.Logger)
		require.True(t, ok)
		assert.NotNil(t, logger)
	})

	t.Run("handles nil body and nil path params", func(t *testing.T) {
		req := testutils.CreateTestRequestWithContext(http.MethodGet, "/api/v1/users/me", nil, testUserID, nil)

		require.NotNil(t, req)
		assert.Equal(t, http.MethodGet, req.Method)
		assert.Equal(t, "/api/v1/users/me", req.URL.Path)

		claimsVal := req.Context().Value(middleware.UserContextKey)
		require.NotNil(t, claimsVal)
		claims, ok := claimsVal.(*models.Claims)
		require.True(t, ok)
		assert.Equal(t, testUserID, claims.UserID)

		loggerVal := req.Context().Value(middleware.LoggerKey)
		require.NotNil(t, loggerVal)
		_, ok = loggerVal.(*slog.Logger)
		require.True(t, ok)
	})
}

func TestCreateTestRequestWithoutContext(t *testing.T) {
	t.Run("creates request without user context but with logger and path params", func(t *testing.T) {
		method := http.MethodGet
		target := "/api/v1/products"
		pathParams := map[string]string{
			"page": "1",
		}

		req := testutils.CreateTestRequestWithoutContext(method, target, nil, pathParams)

		require.NotNil(t, req)
		assert.Equal(t, method, req.Method)
		assert.Equal(t, target, req.URL.Path)

		// Assert path parameters
		assert.Equal(t, "1", req.PathValue("page"))

		// Assert no user claims context
		claimsVal := req.Context().Value(middleware.UserContextKey)
		assert.Nil(t, claimsVal)

		// Assert logger context value
		loggerVal := req.Context().Value(middleware.LoggerKey)
		require.NotNil(t, loggerVal)
		logger, ok := loggerVal.(*slog.Logger)
		require.True(t, ok)
		assert.NotNil(t, logger)
	})

	t.Run("handles nil path params and body", func(t *testing.T) {
		req := testutils.CreateTestRequestWithoutContext(http.MethodGet, "/health", nil, nil)

		require.NotNil(t, req)
		assert.Nil(t, req.Context().Value(middleware.UserContextKey))

		loggerVal := req.Context().Value(middleware.LoggerKey)
		require.NotNil(t, loggerVal)
		_, ok := loggerVal.(*slog.Logger)
		require.True(t, ok)
	})
}

func TestCreateTestRequestWithContext_PR214(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name       string
		method     string
		target     string
		bodyStr    string
		userID     uuid.UUID
		pathParams map[string]string
	}{
		{
			name:       "GET request with no path params and no body",
			method:     http.MethodGet,
			target:     "/api/test",
			bodyStr:    "",
			userID:     userID,
			pathParams: nil,
		},
		{
			name:       "POST request with body and path params",
			method:     http.MethodPost,
			target:     "/api/test/123",
			bodyStr:    `{"test":"test"}`,
			userID:     userID,
			pathParams: map[string]string{"id": "123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body io.Reader
			if tt.bodyStr != "" {
				body = bytes.NewReader([]byte(tt.bodyStr))
			}

			req := testutils.CreateTestRequestWithContext(tt.method, tt.target, body, tt.userID, tt.pathParams)

			assertCommonRequestProperties(t, req, tt.method, tt.target, tt.bodyStr, tt.pathParams)

			claims, ok := req.Context().Value(middleware.UserContextKey).(*models.Claims)
			assert.True(t, ok)
			assert.NotNil(t, claims)
			assert.Equal(t, tt.userID, claims.UserID)
			assert.Equal(t, "test@example.com", claims.Email)
		})
	}
}

func TestCreateTestRequestWithoutContext_PR214(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		target     string
		bodyStr    string
		pathParams map[string]string
	}{
		{
			name:       "GET request with no path params and no body",
			method:     http.MethodGet,
			target:     "/api/test",
			bodyStr:    "",
			pathParams: nil,
		},
		{
			name:       "POST request with body and path params",
			method:     http.MethodPost,
			target:     "/api/test/123",
			bodyStr:    `{"test":"test"}`,
			pathParams: map[string]string{"id": "123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body io.Reader
			if tt.bodyStr != "" {
				body = bytes.NewReader([]byte(tt.bodyStr))
			}

			req := testutils.CreateTestRequestWithoutContext(tt.method, tt.target, body, tt.pathParams)

			assertCommonRequestProperties(t, req, tt.method, tt.target, tt.bodyStr, tt.pathParams)

			claims := req.Context().Value(middleware.UserContextKey)
			assert.Nil(t, claims)
		})
	}
}

func assertCommonRequestProperties(t *testing.T, req *http.Request, method, target, bodyStr string, pathParams map[string]string) {
	assert.NotNil(t, req)
	assert.Equal(t, method, req.Method)
	assert.Equal(t, target, req.URL.Path)

	if bodyStr != "" {
		assert.NotNil(t, req.Body)
		reqBody, err := io.ReadAll(req.Body)
		assert.NoError(t, err)
		assert.Equal(t, bodyStr, string(reqBody))
	}

	for k, v := range pathParams {
		assert.Equal(t, v, req.PathValue(k))
	}

	logger, ok := req.Context().Value(middleware.LoggerKey).(*slog.Logger)
	assert.True(t, ok)
	assert.NotNil(t, logger)
}
