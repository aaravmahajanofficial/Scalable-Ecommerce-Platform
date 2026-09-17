package main

import (
	"net/http"
	"testing"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/api/handlers"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/api/middleware"
	"github.com/stretchr/testify/assert"
)

func TestSetupAPIRouter(t *testing.T) {
	jwtKey := []byte("test-secret-key")
	authMiddleware := middleware.NewAuthMiddleware(jwtKey)

	userHandler := handlers.NewUserHandler(nil)
	productHandler := handlers.NewProductHandler(nil)
	cartHandler := handlers.NewCartHandler(nil)
	orderHandler := handlers.NewOrderHandler(nil)
	paymentHandler := handlers.NewPaymentHandler(nil)
	notificationHandler := handlers.NewNotificationHandler(nil)

	mux := setupAPIRouter(
		userHandler,
		productHandler,
		cartHandler,
		orderHandler,
		paymentHandler,
		notificationHandler,
		authMiddleware,
	)

	assert.NotNil(t, mux)

	routes := []struct {
		method string
		path   string
	}{
		{"POST", "/api/v1/users/register"},
		{"POST", "/api/v1/users/login"},
		{"GET", "/api/v1/users/profile"},
		{"POST", "/api/v1/products"},
		{"GET", "/api/v1/products/123"},
		{"PUT", "/api/v1/products/123"},
		{"GET", "/api/v1/products"},
		{"GET", "/api/v1/carts"},
		{"POST", "/api/v1/carts/items"},
		{"PUT", "/api/v1/carts/items"},
		{"POST", "/api/v1/orders"},
		{"GET", "/api/v1/orders/123"},
		{"GET", "/api/v1/orders"},
		{"PATCH", "/api/v1/orders/123/status"},
		{"POST", "/api/v1/payments"},
		{"GET", "/api/v1/payments/123"},
		{"GET", "/api/v1/payments"},
		{"POST", "/api/v1/payments/webhook"},
		{"POST", "/api/v1/notifications/email"},
		{"GET", "/api/v1/notifications"},
	}

	for _, rt := range routes {
		req, err := http.NewRequest(rt.method, rt.path, http.NoBody)
		assert.NoError(t, err)

		handler, pattern := mux.Handler(req)
		assert.NotNil(t, handler, "Expected handler for %s %s", rt.method, rt.path)
		assert.NotEmpty(t, pattern, "Expected pattern match for %s %s", rt.method, rt.path)
	}
}
