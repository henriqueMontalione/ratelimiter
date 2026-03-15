package tests

import (
	"context"
	"os"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	redistore "github.com/henriquemontalione/ratelimiter/internal/adapters/redis"
	"github.com/henriquemontalione/ratelimiter/internal/config"
	"github.com/henriquemontalione/ratelimiter/internal/limiter"
)

func newTestClient(t *testing.T) *goredis.Client {
	t.Helper()

	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	client := goredis.NewClient(&goredis.Options{Addr: addr})

	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Skipf("redis unavailable at %s: %v", addr, err)
	}

	t.Cleanup(func() {
		client.FlushDB(context.Background())
		client.Close()
	})

	client.FlushDB(context.Background())
	return client
}

func newTestLimiter(client *goredis.Client, ipLimit, tokenLimit, blockSecs int, tokenLimits map[string]int) *limiter.RateLimiter {
	cfg := &config.Config{
		IPRateLimit:       ipLimit,
		TokenRateLimit:    tokenLimit,
		BlockDurationSecs: blockSecs,
		TokenLimits:       tokenLimits,
	}
	store := redistore.NewStore(client)
	return limiter.NewRateLimiter(store, cfg)
}

func TestIPRateLimit(t *testing.T) {
	client := newTestClient(t)
	rl := newTestLimiter(client, 3, 10, 60, nil)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		allowed, err := rl.Allow(ctx, "1.2.3.4", "")
		require.NoError(t, err)
		assert.True(t, allowed, "request %d should be allowed", i+1)
	}

	// 4th request exceeds limit
	allowed, err := rl.Allow(ctx, "1.2.3.4", "")
	require.NoError(t, err)
	assert.False(t, allowed, "request exceeding limit should be blocked")

	// subsequent requests stay blocked
	allowed, err = rl.Allow(ctx, "1.2.3.4", "")
	require.NoError(t, err)
	assert.False(t, allowed, "blocked IP should remain blocked")
}

func TestTokenRateLimit(t *testing.T) {
	client := newTestClient(t)
	rl := newTestLimiter(client, 10, 3, 60, nil)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		allowed, err := rl.Allow(ctx, "1.2.3.4", "my-token")
		require.NoError(t, err)
		assert.True(t, allowed, "request %d should be allowed", i+1)
	}

	allowed, err := rl.Allow(ctx, "1.2.3.4", "my-token")
	require.NoError(t, err)
	assert.False(t, allowed, "request exceeding token limit should be blocked")
}

func TestTokenPrecedenceOverIP(t *testing.T) {
	client := newTestClient(t)
	// IP limit=2, token limit=10 — token should allow more requests than IP limit
	rl := newTestLimiter(client, 2, 10, 60, nil)
	ctx := context.Background()

	// Exhaust IP limit (if IP were consulted, 3rd request would fail)
	for i := 0; i < 5; i++ {
		allowed, err := rl.Allow(ctx, "1.2.3.4", "my-token")
		require.NoError(t, err)
		assert.True(t, allowed, "token request %d should be allowed regardless of IP limit", i+1)
	}
}

func TestTokenIndividualLimitOverridesDefault(t *testing.T) {
	client := newTestClient(t)
	// default token limit=2, vip token limit=10
	rl := newTestLimiter(client, 2, 2, 60, map[string]int{"vip": 10})
	ctx := context.Background()

	// regular token gets blocked after 2
	for i := 0; i < 2; i++ {
		allowed, err := rl.Allow(ctx, "1.2.3.4", "regular")
		require.NoError(t, err)
		assert.True(t, allowed)
	}
	allowed, err := rl.Allow(ctx, "1.2.3.4", "regular")
	require.NoError(t, err)
	assert.False(t, allowed, "regular token should be blocked at default limit")

	// vip token still has room
	for i := 0; i < 10; i++ {
		allowed, err := rl.Allow(ctx, "1.2.3.4", "vip")
		require.NoError(t, err)
		assert.True(t, allowed, "vip token request %d should be allowed", i+1)
	}
	allowed, err = rl.Allow(ctx, "1.2.3.4", "vip")
	require.NoError(t, err)
	assert.False(t, allowed, "vip token should be blocked after individual limit")
}

func TestBlockPersists(t *testing.T) {
	client := newTestClient(t)
	rl := newTestLimiter(client, 1, 10, 2, nil) // block duration = 2s
	ctx := context.Background()

	// exhaust limit
	rl.Allow(ctx, "1.2.3.4", "")
	allowed, err := rl.Allow(ctx, "1.2.3.4", "")
	require.NoError(t, err)
	assert.False(t, allowed)

	// still blocked within block duration
	allowed, err = rl.Allow(ctx, "1.2.3.4", "")
	require.NoError(t, err)
	assert.False(t, allowed, "should remain blocked within block duration")

	// wait for block to expire
	time.Sleep(2500 * time.Millisecond)

	allowed, err = rl.Allow(ctx, "1.2.3.4", "")
	require.NoError(t, err)
	assert.True(t, allowed, "should be allowed after block duration expires")
}

func TestCounterResetsAfterWindow(t *testing.T) {
	client := newTestClient(t)
	// Use limit=2, send exactly 2 requests — at limit but no block triggered
	rl := newTestLimiter(client, 2, 10, 60, nil)
	ctx := context.Background()

	rl.Allow(ctx, "5.5.5.5", "")
	allowed, err := rl.Allow(ctx, "5.5.5.5", "")
	require.NoError(t, err)
	assert.True(t, allowed, "second request should be at limit but still allowed")

	// wait for the 1s counter window to expire
	time.Sleep(1100 * time.Millisecond)

	// counter expired — window reset, requests allowed again from zero
	allowed, err = rl.Allow(ctx, "5.5.5.5", "")
	require.NoError(t, err)
	assert.True(t, allowed, "counter should reset after 1s window")
}
