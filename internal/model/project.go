package model

// Project — контекст связки модулей внутри workspace (Core).
type Project struct {
	ID          string  `json:"id"`
	WorkspaceID string  `json:"workspaceId"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
}

type CreateProjectDto struct {
	Name        string  `json:"name" validate:"required"`
	Description *string `json:"description,omitempty"`
}

type UpdateProjectDto struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// ProjectEntity — привязка сущности модуля к проекту (вариант B: только в Core).
type ProjectEntity struct {
	ID         string `json:"id"`
	ProjectID  string `json:"projectId"`
	EntityType string `json:"entityType"` // например "crm_deal", "task"
	EntityID   string `json:"entityId"`
	CreatedAt  string `json:"createdAt"`
}

type AttachEntityToProjectDto struct {
	EntityType string `json:"entityType" validate:"required"`
	EntityID   string `json:"entityId" validate:"required"`
}
