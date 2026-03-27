package model

import "time"

type ChatThreadType string

const (
	ChatThreadPrivate ChatThreadType = "private"
	ChatThreadGroup   ChatThreadType = "group"
)

type ChatThread struct {
	ID          string         `json:"id"`
	WorkspaceID string         `json:"workspaceId"`
	Type        ChatThreadType `json:"type"`
	Title       *string        `json:"title,omitempty"`
	CreatedBy   string         `json:"createdBy"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`

	// Convenience (computed in queries)
	LastMessagePreview *string `json:"lastMessagePreview,omitempty"`
}

type ChatMessage struct {
	ID          string    `json:"id"`
	ThreadID    string    `json:"threadId"`
	WorkspaceID string    `json:"workspaceId"`
	AuthorID    string    `json:"authorId"`
	Body        string    `json:"body"`
	CreatedAt   time.Time `json:"createdAt"`
}

