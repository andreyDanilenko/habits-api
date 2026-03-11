package model

import "time"

const (
	InvitationStatusPending   = "PENDING"
	InvitationStatusAccepted  = "ACCEPTED"
	InvitationStatusExpired   = "EXPIRED"
	InvitationStatusCancelled = "CANCELLED"
)

type Invitation struct {
	ID          string     `json:"id" db:"id"`
	WorkspaceID string     `json:"workspaceId" db:"workspace_id"`
	Email       string     `json:"email" db:"email"`
	InvitedBy   string     `json:"invitedBy" db:"invited_by"`
	SystemRole  string     `json:"systemRole" db:"system_role"`
	Status      string     `json:"status" db:"status"`
	Token       string     `json:"-" db:"token"`
	ExpiresAt   time.Time  `json:"expiresAt" db:"expires_at"`
	CreatedAt   time.Time  `json:"createdAt" db:"created_at"`
	AcceptedAt  *time.Time `json:"acceptedAt,omitempty" db:"accepted_at"`
}

type CreateInvitationRequest struct {
	Email      string   `json:"email" validate:"required,email"`
	SystemRole string   `json:"systemRole" validate:"required,oneof=MEMBER GUEST"`
	ExpiresIn  *int64   `json:"expiresIn,omitempty"` // секунды, по умолчанию 7 дней
}

type InvitationResponse struct {
	ID          string            `json:"id"`
	Email       string            `json:"email"`
	WorkspaceID string            `json:"workspaceId"`
	InvitedBy   *InvitedByUser    `json:"invitedBy"`
	SystemRole  string            `json:"systemRole"`
	Status      string            `json:"status"`
	InviteLink  string            `json:"inviteLink,omitempty"`
	ExpiresAt   string            `json:"expiresAt"`
	CreatedAt   string            `json:"createdAt"`
}

type InvitedByUser struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type PublicInvitationResponse struct {
	Email           string `json:"email"`
	WorkspaceName   string `json:"workspaceName"`
	InvitedByName   string `json:"invitedByName"`
	SystemRole      string `json:"systemRole"`
	ExpiresAt       string `json:"expiresAt"`
	UserExists      bool   `json:"userExists"`
	IsAuthenticated bool   `json:"isAuthenticated"`
}

type AcceptInvitationResponse struct {
	Status     string `json:"status"` // "accepted" | "requires_auth" | "expired"
	RedirectTo string `json:"redirectTo,omitempty"`
	Email      string `json:"email,omitempty"`
	UserExists bool   `json:"userExists,omitempty"`
	Message    string `json:"message,omitempty"`
}
