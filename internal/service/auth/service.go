package auth

import (
	"backend/internal/repository/auth"
)

type Service struct {
	repo *auth.Repository
}

func NewService(repo *auth.Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// TODO: implement service methods
