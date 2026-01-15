package habits

import (
	"backend/internal/repository/habits"
)

type Service struct {
	repo *habits.Repository
}

func NewService(repo *habits.Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// TODO: implement service methods
