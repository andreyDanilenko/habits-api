package model

// Currency — валюта (Shared Schema движка ERP).
type Currency struct {
	ID          string  `json:"id" db:"id"`
	WorkspaceID string  `json:"workspaceId" db:"workspace_id"`
	Code        string  `json:"code" db:"code"`
	Name        string  `json:"name" db:"name"`
	Symbol      *string `json:"symbol,omitempty" db:"symbol"`
	CreatedAt   string  `json:"createdAt" db:"created_at"`
	UpdatedAt   string  `json:"updatedAt" db:"updated_at"`
}

// Counterparty — контрагент: клиент/поставщик (Shared Schema).
type Counterparty struct {
	ID          string  `json:"id" db:"id"`
	WorkspaceID string  `json:"workspaceId" db:"workspace_id"`
	Name        string  `json:"name" db:"name"`
	Type        string  `json:"type" db:"type"` // client, supplier, both
	Email       *string `json:"email,omitempty" db:"email"`
	Phone       *string `json:"phone,omitempty" db:"phone"`
	Comment     *string `json:"comment,omitempty" db:"comment"`
	CreatedAt   string  `json:"createdAt" db:"created_at"`
	UpdatedAt   string  `json:"updatedAt" db:"updated_at"`
}

// Note — заметка (модуль Заметки).
type Note struct {
	ID          string `json:"id" db:"id"`
	WorkspaceID string `json:"workspaceId" db:"workspace_id"`
	UserID      string `json:"userId" db:"user_id"`
	Title       string `json:"title" db:"title"`
	Content     string `json:"content" db:"content"`
	CreatedAt   string `json:"createdAt" db:"created_at"`
	UpdatedAt   string `json:"updatedAt" db:"updated_at"`
}
