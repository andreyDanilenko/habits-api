package realtime

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisAdapter adapts go-redis to our RedisClient interface.
type RedisAdapter struct {
	client *redis.Client
}

func NewRedisAdapter(redisURL string) (*RedisAdapter, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	if opt.ReadTimeout == 0 {
		opt.ReadTimeout = 5 * time.Second
	}
	if opt.WriteTimeout == 0 {
		opt.WriteTimeout = 5 * time.Second
	}
	client := redis.NewClient(opt)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return &RedisAdapter{client: client}, nil
}

func (r *RedisAdapter) Publish(ctx context.Context, channel string, message interface{}) error {
	return r.client.Publish(ctx, channel, message).Err()
}

func (r *RedisAdapter) Close() error {
	return r.client.Close()
}
