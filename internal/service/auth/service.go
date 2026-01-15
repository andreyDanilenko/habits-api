package auth

import (
	"backend/internal/repository/auth"
)

type Service struct {
	repo *auth.PostgresUserRepository
}

func NewService(repo *auth.PostgresUserRepository) *Service {
	return &Service{
		repo: repo,
	}
}

// TODO: implement service methods
