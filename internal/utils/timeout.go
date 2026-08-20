// Package apputils provides shared application utilities.
package apputils

import (
	"context"
	"time"
)

const DefaultDBTimeout = 5 * time.Second

func WithDBTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, DefaultDBTimeout)
}
