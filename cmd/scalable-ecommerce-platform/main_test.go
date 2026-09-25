package main

import (
	"net/http"
	"testing"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/config"
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
