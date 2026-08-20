// Package testutils provides HTTP test helpers.
package testutils

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/api/middleware"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/google/uuid"
)

func CreateTestRequestWithContext(method, target string, body io.Reader, userID uuid.UUID, pathParams map[string]string) *http.Request {
	claims := &models.Claims{UserID: userID, Email: "test@example.com"}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	ctx := context.WithValue(context.Background(), middleware.UserContextKey, claims)
	ctx = context.WithValue(ctx, middleware.LoggerKey, logger)

	req := httptest.NewRequestWithContext(ctx, method, target, body)

	for key, value := range pathParams {
		req.SetPathValue(key, value)
	}

	return req
}

func CreateTestRequestWithoutContext(method, target string, body io.Reader, pathParams map[string]string) *http.Request {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.WithValue(context.Background(), middleware.LoggerKey, logger)

	req := httptest.NewRequestWithContext(ctx, method, target, body)

	for key, value := range pathParams {
		req.SetPathValue(key, value)
	}

	return req
}
