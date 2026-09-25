// Package health provides liveness and readiness HTTP handlers.
package health

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/config"
	stripeClient "github.com/aaravmahajanofficial/scalable-ecommerce-platform/pkg/stripe"
	"github.com/hellofresh/health-go/v5"
	"github.com/hellofresh/health-go/v5/checks/postgres"
	healthRedis "github.com/hellofresh/health-go/v5/checks/redis"
	"github.com/redis/go-redis/v9"
	"github.com/stripe/stripe-go/v86"
	"github.com/stripe/stripe-go/v86/balance"
)

type Endpoint struct {
	DB           *sql.DB
	RedisClient  *redis.Client
	StripeClient *stripeClient.Client
}

func NewReadinessHandler(cfg *config.Config, healthEndpoint *Endpoint) (http.Handler, error) {
	h, err := health.New(

		health.WithComponent(health.Component{
			Name:    cfg.OTel.ServiceName,
			Version: "1.0.0",
		}),
		health.WithSystemInfo(),
		health.WithChecks(
			health.Config{
				Name:      "database",
				Timeout:   3 * time.Second,
				SkipOnErr: false,
				Check: postgres.New(postgres.Config{
					DSN: cfg.Database.GetDSN(),
				}),
			},
			health.Config{
				Name:      "redis",
				Timeout:   2 * time.Second,
				SkipOnErr: false,
				Check: healthRedis.New(
					healthRedis.Config{
						DSN: cfg.RedisConnect.GetDSN(),
					},
				),
			},
			health.Config{
				Name:      "stripe",
				Timeout:   5 * time.Second,
				SkipOnErr: false,
				Check: func(ctx context.Context) error {
					return checkStripeHealth(ctx, healthEndpoint.StripeClient)
				},
			},
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create readiness health instance: %w", err)
	}

	return h.Handler(), nil
}

//nolint:gocritic // client parameter matches Endpoint.StripeClient (*stripeClient.Client)
func checkStripeHealth(ctx context.Context, client *stripeClient.Client) error {
	if client == nil || *client == nil {
		return errors.New("stripe client is not initialized")
	}

	reqCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	params := &stripe.BalanceParams{
		Params: stripe.Params{
			Context: reqCtx,
		},
	}
	_, err := balance.Get(params)
	if err == nil {
		return nil
	}

	if ctxErr := reqCtx.Err(); errors.Is(ctxErr, context.DeadlineExceeded) {
		return fmt.Errorf("stripe API call timed out: %w", ctxErr)
	}

	return fmt.Errorf("failed to connect to stripe: %w", err)
}

func NewLivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		if _, err := fmt.Fprintf(w, "Service is alive. Time: %s\n", time.Now().Format(time.RFC3339)); err != nil {
			slog.Error("failed to write liveness response", slog.Any("error", err))
		}
	}
}
