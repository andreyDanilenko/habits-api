package model

import "time"

type UserRole string

const (
	UserRoleUser  UserRole = "USER"
	UserRoleAdmin UserRole = "ADMIN"
)

type UserStatus string

const (
	UserStatusActive UserStatus = "ACTIVE"
)

type User struct {
	ID       string   `json:"id" db:"id"`
	Email    string   `json:"email" db:"email"`
	Password string   `json:"-" db:"password"`
	Name     *string  `json:"name,omitempty" db:"name"`
	Role     UserRole `json:"role" db:"role"`

	AvatarURL *string     `json:"avatarUrl,omitempty" db:"avatar_url"`
	Status    *UserStatus `json:"status,omitempty" db:"status"`
	CreatedAt time.Time   `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time   `json:"updatedAt" db:"updated_at"`
}

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Name     string `json:"name" validate:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}
