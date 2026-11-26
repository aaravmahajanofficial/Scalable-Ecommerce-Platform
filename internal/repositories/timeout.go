package repository

import (
	"context"
	"time"
)

const dbTimeout = 5 * time.Second

func withDBTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, dbTimeout)
}
