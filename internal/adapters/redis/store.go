package redis

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

var incrementScript = goredis.NewScript(`
local current = redis.call('INCR', KEYS[1])
if current == 1 then
    redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return current
`)

type Store struct {
	client *goredis.Client
}

func NewStore(client *goredis.Client) *Store {
	return &Store{client: client}
}

func (r *Store) IsBlocked(ctx context.Context, key string) (bool, error) {
	val, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return val > 0, nil
}

func (r *Store) Increment(ctx context.Context, key string, windowSecs int) (int64, error) {
	result, err := incrementScript.Run(ctx, r.client, []string{key}, windowSecs).Int64()
	if err != nil {
		return 0, err
	}
	return result, nil
}

func (r *Store) Block(ctx context.Context, key string, duration time.Duration) error {
	return r.client.Set(ctx, key, 1, duration).Err()
}
