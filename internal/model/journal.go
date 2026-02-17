package model

type JournalEntry struct {
	ID          string                 `json:"id" db:"id"`
	WorkspaceID string                 `json:"workspaceId" db:"workspace_id"`
	UserID      string                 `json:"userId" db:"user_id"`
	Description string                 `json:"description" db:"description"`
	Mood        *int                   `json:"mood,omitempty" db:"mood"`
	Date        string                 `json:"date" db:"date"`
	Tags        []string               `json:"tags,omitempty" db:"tags"`
	ContentType string                 `json:"contentType" db:"content_type"`
	Metadata    map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CreatedAt   string                 `json:"createdAt" db:"created_at"`
	UpdatedAt   string                 `json:"updatedAt" db:"updated_at"`
}

type CreateJournalEntryDto struct {
	Description string                 `json:"description"`
	Mood        *int                   `json:"mood,omitempty"`
	Date        string                 `json:"date,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	ContentType string                 `json:"contentType,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type UpdateJournalEntryDto struct {
	Description *string                `json:"description,omitempty"`
	Mood        *int                   `json:"mood,omitempty"`
	Date        *string                `json:"date,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	ContentType *string                `json:"contentType,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
