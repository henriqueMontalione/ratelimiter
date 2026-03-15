package ports

import (
	"context"
	"time"
)

type Store interface {
	IsBlocked(ctx context.Context, key string) (bool, error)
	Increment(ctx context.Context, key string, windowSecs int) (int64, error)
	Block(ctx context.Context, key string, duration time.Duration) error
}
