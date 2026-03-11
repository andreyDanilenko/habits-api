package model

import "time"

// RegistrationToken — ожидающая подтверждения регистрация.
// Пользователь создаётся в users только после верификации по ссылке.
type RegistrationToken struct {
	ID           string
	Email        string
	PasswordHash string
	Name         *string
	Token        string
	ExpiresAt    time.Time
	CreatedAt    time.Time
}
