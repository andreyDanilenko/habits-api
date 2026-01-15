package workspace

import (
	"backend/internal/repository/workspace"
)

type Service struct {
	repo *workspace.Repository
}

func NewService(repo *workspace.Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// TODO: implement service methods
