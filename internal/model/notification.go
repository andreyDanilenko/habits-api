package model

import (
	"encoding/json"
	"time"
)

// Notification — уведомление для пользователя (универсальное: activity, chat, system).
type Notification struct {
	ID          string          `json:"id"`
	UserID      string          `json:"userId"`
	WorkspaceID *string         `json:"workspaceId,omitempty"`
	Channel     string          `json:"channel"` // activity, chat, system
	EventType   string          `json:"eventType"`
	EventKey    string          `json:"eventKey"`
	Title       string          `json:"title"`
	Subtitle    *string         `json:"subtitle,omitempty"`
	Payload     json.RawMessage `json:"payload,omitempty"`
	CreatedAt   time.Time       `json:"createdAt"`
	ReadAt      *time.Time      `json:"readAt,omitempty"`
}

// CreateNotificationDto — создание/обновление уведомления (идемпотентно по event_key).
type CreateNotificationDto struct {
	WorkspaceID *string         `json:"workspaceId,omitempty"`
	Channel     string          `json:"channel"`
	EventType   string          `json:"eventType"`
	EventKey    string          `json:"eventKey"`
	Title       string          `json:"title"`
	Subtitle    *string         `json:"subtitle,omitempty"`
	Payload     json.RawMessage `json:"payload,omitempty"`
}

// NotificationListOpts — фильтры для списка.
type NotificationListOpts struct {
	Channel  string
	UnreadOnly bool
	Limit    int
	Offset   int
}
