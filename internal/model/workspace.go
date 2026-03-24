package model

type Workspace struct {
	ID          string  `json:"id" db:"id"`
	Name        string  `json:"name" db:"name"`
	Description *string `json:"description,omitempty" db:"description"`
	Color       string  `json:"color" db:"color"`
	LogoPath    *string `json:"-" db:"logo_path"`
	LogoUrl     *string `json:"logoUrl,omitempty"`
	LogoScale   float64 `json:"logoScale" db:"logo_scale"`
	OwnerID     string  `json:"ownerId" db:"owner_id"`
	CreatedAt   string  `json:"createdAt" db:"created_at"`
	UpdatedAt   string  `json:"updatedAt" db:"updated_at"`
}

type CreateWorkspaceDto struct {
	Name        string  `json:"name" validate:"required"`
	Description *string `json:"description,omitempty"`
	Color       *string `json:"color,omitempty"`
}

type UpdateWorkspaceDto struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Color       *string `json:"color,omitempty"`
	LogoScale   *float64 `json:"logoScale,omitempty"`
}

// WorkspaceMember — участник workspace для списка на UI.
type WorkspaceMember struct {
	ID         string  `json:"id"`
	Email      string  `json:"email"`
	Name       string  `json:"name"`
	SystemRole string  `json:"systemRole"`
	JoinedAt   string  `json:"joinedAt"`
	AvatarURL  *string `json:"avatarUrl,omitempty"`
}

// WorkspaceMemberFanout — участник для внутренней рассылки (realtime), не API.
type WorkspaceMemberFanout struct {
	UserID     string
	GlobalRole UserRole
}
