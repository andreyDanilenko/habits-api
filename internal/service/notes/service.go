package notes

import (
	"context"

	"backend/internal/model"
	notesRepo "backend/internal/repository/notes"

	"github.com/google/uuid"
)

type Service struct {
	repo *notesRepo.Repository
}

func NewService(repo *notesRepo.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, workspaceID string) ([]model.Note, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	return s.repo.List(ctx, wsID)
}

func (s *Service) Get(ctx context.Context, workspaceID, id string) (*model.Note, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	return s.repo.Get(ctx, uid, wsID)
}

func (s *Service) Create(ctx context.Context, n *model.Note) error {
	return s.repo.Create(ctx, n)
}

func (s *Service) Update(ctx context.Context, n *model.Note) error {
	return s.repo.Update(ctx, n)
}

func (s *Service) Delete(ctx context.Context, workspaceID, id string) error {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return err
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, uid, wsID)
}
