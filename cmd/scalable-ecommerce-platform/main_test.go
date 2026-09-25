package main

import (
	"net/http"
	"testing"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/config"
	repository "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories"
	repoMocks "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories/mocks"
	sendgridMocks "github.com/aaravmahajanofficial/scalable-ecommerce-platform/pkg/sendgrid/mocks"
	stripeMocks "github.com/aaravmahajanofficial/scalable-ecommerce-platform/pkg/stripe/mocks"
	"github.com/stretchr/testify/assert"
)

func TestGetSwaggerHost(t *testing.T) {
	t.Run("returns addr when present", func(t *testing.T) {
		cfg := &config.Config{
			HTTPServer: config.HTTPServer{
				Addr: "localhost:8085",
			},
		}
		host := getSwaggerHost(cfg)
		assert.Equal(t, "localhost:8085", host)
	})

	t.Run("returns default when addr empty", func(t *testing.T) {
		cfg := &config.Config{
			HTTPServer: config.HTTPServer{
				Addr: "",
			},
		}
		host := getSwaggerHost(cfg)
		assert.Equal(t, "local:8085", host)
	})
}

func TestNewServer(t *testing.T) {
	cfg := &config.Config{
		HTTPServer: config.HTTPServer{
			Addr:         ":8080",
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  15 * time.Second,
		},
	}
	handler := http.NewServeMux()
	server := newServer(cfg, handler)

	assert.Equal(t, ":8080", server.Addr)
	assert.Equal(t, 5*time.Second, server.ReadTimeout)
	assert.Equal(t, 10*time.Second, server.WriteTimeout)
	assert.Equal(t, 15*time.Second, server.IdleTimeout)
	assert.Equal(t, handler, server.Handler)
}

func TestSetupRouter(t *testing.T) {
	cfg := &config.Config{
		Env: "testing",
		HTTPServer: config.HTTPServer{
			Addr: "localhost:8085",
		},
		OTel: config.OTelConfig{
			ServiceName: "test-service",
		},
	}

	repos := &repository.Repositories{
		User:         repoMocks.NewMockUserRepository(t),
		Product:      repoMocks.NewMockProductRepository(t),
		Cart:         repoMocks.NewMockCartRepository(t),
		Order:        repoMocks.NewMockOrderRepository(t),
		Payment:      repoMocks.NewMockPaymentRepository(t),
		Notification: repoMocks.NewMockNotificationRepository(t),
		RateLimiter:  repoMocks.NewMockRateLimitRepository(t),
	}

	jwtKey := []byte("secret")
	stripeClient := stripeMocks.NewMockClient(t)
	sendGridClient := sendgridMocks.NewMockEmailService(t)

	handler := setupRouter(cfg, repos, jwtKey, stripeClient, sendGridClient, "localhost:8085")
	assert.NotNil(t, handler)
}
