package health

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/config"
	stripeClient "github.com/aaravmahajanofficial/scalable-ecommerce-platform/pkg/stripe"
	"github.com/hellofresh/health-go/v5"
	"github.com/redis/go-redis/v9"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/balance"
)

type HealthEndpoint struct {
	DB           *sql.DB
	RedisClient  *redis.Client
	StripeClient *stripeClient.Client
}

func NewReadinessHandler(cfg *config.Config, healthEndpoint *HealthEndpoint) (http.Handler, error) {
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
				Check: func(ctx context.Context) error {
					if healthEndpoint.DB == nil {
						return errors.New("database connection is not initialized")
					}
					return healthEndpoint.DB.PingContext(ctx)
				},
			},
			health.Config{
				Name:      "redis",
				Timeout:   2 * time.Second,
				SkipOnErr: false,
				Check: func(ctx context.Context) error {
					if healthEndpoint.RedisClient == nil {
						return errors.New("redis client is not initialized")
					}
					return healthEndpoint.RedisClient.Ping(ctx).Err()
				},
			},
			health.Config{
				Name:      "stripe",
				Timeout:   5 * time.Second,
				SkipOnErr: false,
				Check: func(ctx context.Context) error {
					if healthEndpoint.StripeClient == nil {
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
					if err != nil {
						if ctxErr := reqCtx.Err(); errors.Is(ctxErr, context.DeadlineExceeded) {
							return fmt.Errorf("stripe API call timed out: %w", ctxErr)
						}

						return fmt.Errorf("failed to connect to stripe: %w", err)
					}

					return nil
				},
			},
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create readiness health instance: %w", err)
	}

	return h.Handler(), nil
}

func NewLivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "Service is alive. Time: %s\n", time.Now().Format(time.RFC3339))
	}
}
