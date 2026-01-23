package model

import "time"

type UserPreferences struct {
	ID                 string    `json:"id" db:"id"`
	UserID             string    `json:"user_id" db:"user_id"`
	CurrentWorkspaceID *string   `json:"current_workspace_id,omitempty" db:"current_workspace_id"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}
