package http_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	httpadapter "github.com/henriquemontalione/ratelimiter/internal/adapters/http"
	"github.com/henriquemontalione/ratelimiter/internal/config"
	"github.com/henriquemontalione/ratelimiter/internal/limiter"
	"github.com/henriquemontalione/ratelimiter/internal/ports"
)

func init() {
	gin.SetMode(gin.TestMode)
}

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

func newRouter(store ports.Store) *gin.Engine {
	cfg := &config.Config{
		IPRateLimit:       10,
		TokenRateLimit:    20,
		BlockDurationSecs: 300,
		TokenLimits:       map[string]int{},
	}
	rl := limiter.NewRateLimiter(store, cfg)
	r := gin.New()
	r.Use(httpadapter.RateLimit(rl))
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	return r
}

func TestMiddleware_AllowedRequest(t *testing.T) {
	store := &mockStore{}
	store.On("IsBlocked", mock.Anything, mock.Anything).Return(false, nil)
	store.On("Increment", mock.Anything, mock.Anything, mock.Anything).Return(int64(1), nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	newRouter(store).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMiddleware_BlockedReturns429(t *testing.T) {
	store := &mockStore{}
	store.On("IsBlocked", mock.Anything, mock.Anything).Return(true, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	newRouter(store).ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Contains(t, w.Body.String(), "you have reached the maximum number of requests or actions allowed within a certain time frame")
}

func TestMiddleware_ExactErrorMessage(t *testing.T) {
	store := &mockStore{}
	store.On("IsBlocked", mock.Anything, mock.Anything).Return(true, nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	newRouter(store).ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t,
		"you have reached the maximum number of requests or actions allowed within a certain time frame",
		w.Body.String(),
	)
}

func TestMiddleware_StoreErrorReturns500(t *testing.T) {
	store := &mockStore{}
	store.On("IsBlocked", mock.Anything, mock.Anything).Return(false, errors.New("redis down"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	newRouter(store).ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestMiddleware_TokenExtractedFromHeader(t *testing.T) {
	store := &mockStore{}
	// token key must be used, not IP key
	store.On("IsBlocked", mock.Anything, "rl:blocked:token:my-secret").Return(false, nil)
	store.On("Increment", mock.Anything, "rl:counter:token:my-secret", mock.Anything).Return(int64(1), nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("API_KEY", "my-secret")
	newRouter(store).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	store.AssertExpectations(t)
}
