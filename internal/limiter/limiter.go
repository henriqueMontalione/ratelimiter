package limiter

import (
	"context"
	"fmt"
	"time"

	"github.com/henriquemontalione/ratelimiter/internal/config"
	"github.com/henriquemontalione/ratelimiter/internal/ports"
)

const (
	prefixCounter = "rl:counter"
	prefixBlocked = "rl:blocked"
	windowSecs    = 1
)

type RateLimiter struct {
	store ports.Store
	cfg   *config.Config
}

func NewRateLimiter(store ports.Store, cfg *config.Config) *RateLimiter {
	return &RateLimiter{store: store, cfg: cfg}
}

func (r *RateLimiter) Allow(ctx context.Context, ip, token string) (bool, error) {
	counterKey, blockedKey, limit := r.resolveKeys(ip, token)

	blocked, err := r.store.IsBlocked(ctx, blockedKey)
	if err != nil {
		return false, err
	}
	if blocked {
		return false, nil
	}

	count, err := r.store.Increment(ctx, counterKey, windowSecs)
	if err != nil {
		return false, err
	}

	if int(count) > limit {
		blockDuration := time.Duration(r.cfg.BlockDurationSecs) * time.Second
		if err := r.store.Block(ctx, blockedKey, blockDuration); err != nil {
			return false, err
		}
		return false, nil
	}

	return true, nil
}

func (r *RateLimiter) resolveKeys(ip, token string) (counterKey, blockedKey string, limit int) {
	if token != "" {
		return fmt.Sprintf("%s:token:%s", prefixCounter, token),
			fmt.Sprintf("%s:token:%s", prefixBlocked, token),
			r.tokenLimit(token)
	}
	return fmt.Sprintf("%s:ip:%s", prefixCounter, ip),
		fmt.Sprintf("%s:ip:%s", prefixBlocked, ip),
		r.cfg.IPRateLimit
}

func (r *RateLimiter) tokenLimit(token string) int {
	if limit, ok := r.cfg.TokenLimits[token]; ok {
		return limit
	}
	return r.cfg.TokenRateLimit
}
