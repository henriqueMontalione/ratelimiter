package limiter_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/henriquemontalione/ratelimiter/internal/config"
	"github.com/henriquemontalione/ratelimiter/internal/limiter"
)

type mockStore struct {
	mock.Mock
}

func (m *mockStore) IsBlocked(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *mockStore) Increment(ctx context.Context, key string, windowSecs int) (int64, error) {
	args := m.Called(ctx, key, windowSecs)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockStore) Block(ctx context.Context, key string, duration time.Duration) error {
	args := m.Called(ctx, key, duration)
	return args.Error(0)
}

func newTestConfig() *config.Config {
	return &config.Config{
		IPRateLimit:       10,
		TokenRateLimit:    20,
		BlockDurationSecs: 300,
		TokenLimits:       map[string]int{"vip": 100},
	}
}

func TestAllow_IPAllowed(t *testing.T) {
	store := &mockStore{}
	rl := limiter.NewRateLimiter(store, newTestConfig())
	ctx := context.Background()

	store.On("IsBlocked", ctx, "rl:blocked:ip:1.2.3.4").Return(false, nil)
	store.On("Increment", ctx, "rl:counter:ip:1.2.3.4", 1).Return(int64(1), nil)

	allowed, err := rl.Allow(ctx, "1.2.3.4", "")

	assert.NoError(t, err)
	assert.True(t, allowed)
	store.AssertExpectations(t)
}

func TestAllow_IPBlocked(t *testing.T) {
	store := &mockStore{}
	rl := limiter.NewRateLimiter(store, newTestConfig())
	ctx := context.Background()

	store.On("IsBlocked", ctx, "rl:blocked:ip:1.2.3.4").Return(true, nil)

	allowed, err := rl.Allow(ctx, "1.2.3.4", "")

	assert.NoError(t, err)
	assert.False(t, allowed)
	store.AssertExpectations(t)
}

func TestAllow_IPExceedsLimit(t *testing.T) {
	store := &mockStore{}
	cfg := newTestConfig()
	rl := limiter.NewRateLimiter(store, cfg)
	ctx := context.Background()

	blockDuration := time.Duration(cfg.BlockDurationSecs) * time.Second

	store.On("IsBlocked", ctx, "rl:blocked:ip:1.2.3.4").Return(false, nil)
	store.On("Increment", ctx, "rl:counter:ip:1.2.3.4", 1).Return(int64(11), nil)
	store.On("Block", ctx, "rl:blocked:ip:1.2.3.4", blockDuration).Return(nil)

	allowed, err := rl.Allow(ctx, "1.2.3.4", "")

	assert.NoError(t, err)
	assert.False(t, allowed)
	store.AssertExpectations(t)
}

func TestAllow_TokenTakesPrecedenceOverIP(t *testing.T) {
	store := &mockStore{}
	rl := limiter.NewRateLimiter(store, newTestConfig())
	ctx := context.Background()

	// Token keys must be used — IP keys must never be called
	store.On("IsBlocked", ctx, "rl:blocked:token:mytoken").Return(false, nil)
	store.On("Increment", ctx, "rl:counter:token:mytoken", 1).Return(int64(1), nil)

	allowed, err := rl.Allow(ctx, "1.2.3.4", "mytoken")

	assert.NoError(t, err)
	assert.True(t, allowed)
	store.AssertExpectations(t)
}

func TestAllow_TokenUsesIndividualLimit(t *testing.T) {
	store := &mockStore{}
	cfg := newTestConfig() // vip token has limit 100
	rl := limiter.NewRateLimiter(store, cfg)
	ctx := context.Background()

	blockDuration := time.Duration(cfg.BlockDurationSecs) * time.Second

	// count=21 exceeds default TOKEN_RATE_LIMIT(20) but NOT vip limit(100)
	store.On("IsBlocked", ctx, "rl:blocked:token:vip").Return(false, nil)
	store.On("Increment", ctx, "rl:counter:token:vip", 1).Return(int64(21), nil)

	allowed, err := rl.Allow(ctx, "1.2.3.4", "vip")

	assert.NoError(t, err)
	assert.True(t, allowed)
	// Block must NOT have been called
	store.AssertNotCalled(t, "Block", ctx, mock.Anything, blockDuration)
}

func TestAllow_TokenExceedsIndividualLimit(t *testing.T) {
	store := &mockStore{}
	cfg := newTestConfig() // vip token has limit 100
	rl := limiter.NewRateLimiter(store, cfg)
	ctx := context.Background()

	blockDuration := time.Duration(cfg.BlockDurationSecs) * time.Second

	store.On("IsBlocked", ctx, "rl:blocked:token:vip").Return(false, nil)
	store.On("Increment", ctx, "rl:counter:token:vip", 1).Return(int64(101), nil)
	store.On("Block", ctx, "rl:blocked:token:vip", blockDuration).Return(nil)

	allowed, err := rl.Allow(ctx, "1.2.3.4", "vip")

	assert.NoError(t, err)
	assert.False(t, allowed)
	store.AssertExpectations(t)
}
