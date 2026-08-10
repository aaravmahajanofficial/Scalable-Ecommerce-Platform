package testutils_test

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/api/middleware"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/testutils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCreateTestRequestWithContext(t *testing.T) {
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

			assert.NotNil(t, req)
			assert.Equal(t, tt.method, req.Method)
			assert.Equal(t, tt.target, req.URL.String())

			if tt.bodyStr != "" {
				assert.NotNil(t, req.Body)
				reqBody, err := io.ReadAll(req.Body)
				assert.NoError(t, err)
				assert.Equal(t, tt.bodyStr, string(reqBody))
			}

			for k, v := range tt.pathParams {
				assert.Equal(t, v, req.PathValue(k))
			}

			claims, ok := req.Context().Value(middleware.UserContextKey).(*models.Claims)
			assert.True(t, ok)
			assert.NotNil(t, claims)
			assert.Equal(t, tt.userID, claims.UserID)
			assert.Equal(t, "test@example.com", claims.Email)

			logger, ok := req.Context().Value(middleware.LoggerKey).(*slog.Logger)
			assert.True(t, ok)
			assert.NotNil(t, logger)
		})
	}
}

func TestCreateTestRequestWithoutContext(t *testing.T) {
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

			assert.NotNil(t, req)
			assert.Equal(t, tt.method, req.Method)
			assert.Equal(t, tt.target, req.URL.String())

			if tt.bodyStr != "" {
				assert.NotNil(t, req.Body)
				reqBody, err := io.ReadAll(req.Body)
				assert.NoError(t, err)
				assert.Equal(t, tt.bodyStr, string(reqBody))
			}

			for k, v := range tt.pathParams {
				assert.Equal(t, v, req.PathValue(k))
			}

			claims := req.Context().Value(middleware.UserContextKey)
			assert.Nil(t, claims)

			logger, ok := req.Context().Value(middleware.LoggerKey).(*slog.Logger)
			assert.True(t, ok)
			assert.NotNil(t, logger)
		})
	}
}
