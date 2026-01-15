package journal

import (
	"backend/internal/repository/journal"
)

type Service struct {
	repo *journal.Repository
}

func NewService(repo *journal.Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// TODO: implement service methods
