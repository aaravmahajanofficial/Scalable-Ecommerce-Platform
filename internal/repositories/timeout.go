package repository

import (
	"context"
	"time"
)

const defaultDBTimeout = 5 * time.Second

func withDBTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, defaultDBTimeout)
}
