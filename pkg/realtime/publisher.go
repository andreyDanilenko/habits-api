package realtime

import (
	"context"
	"encoding/json"
	"fmt"
)

// Event represents a realtime event to be published.
type Event struct {
	EventType string                 `json:"eventType"`
	Target    Target                 `json:"target"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp int64                  `json:"timestamp,omitempty"`
}

type Target struct {
	Type string `json:"type"` // "user" or "workspace"
	ID   string `json:"id"`
}

// Publisher publishes events to Redis for the realtime satellite.
type Publisher interface {
	Publish(ctx context.Context, channel string, event Event) error
}

// NoopPublisher does nothing (when Redis is disabled).
type NoopPublisher struct{}

func (NoopPublisher) Publish(ctx context.Context, channel string, event Event) error {
	return nil
}

// RedisPublisher publishes to Redis Pub/Sub.
type RedisPublisher struct {
	redis RedisClient
}

type RedisClient interface {
	Publish(ctx context.Context, channel string, message interface{}) error
}

func NewRedisPublisher(redis RedisClient) *RedisPublisher {
	return &RedisPublisher{redis: redis}
}

func (p *RedisPublisher) Publish(ctx context.Context, channel string, event Event) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return p.redis.Publish(ctx, channel, body)
}

// Channel helpers
func WorkspaceChannel(workspaceID string) string {
	return "ws:workspace:" + workspaceID
}

func UserChannel(userID string) string {
	return "ws:user:" + userID
}
