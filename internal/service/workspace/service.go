package workspace

import (
	"context"
	"errors"

	"backend/internal/model"
	"backend/internal/repository/workspace"

	"github.com/google/uuid"
)

var (
	ErrWorkspaceNotFound = errors.New("workspace not found")
	ErrAccessDenied      = errors.New("access denied")
)

type Service struct {
	repo *workspace.Repository
}

func NewService(repo *workspace.Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) List(ctx context.Context, userID string) ([]model.Workspace, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	return s.repo.List(ctx, uid)
}

func (s *Service) Create(ctx context.Context, dto model.CreateWorkspaceDto, userID string) (*model.Workspace, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	return s.repo.Create(ctx, dto, uid)
}

func (s *Service) Get(ctx context.Context, workspaceID, userID string, userRole model.UserRole) (*model.Workspace, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	hasAccess, err := s.repo.CheckAccess(ctx, wsID, uid, userRole)
	if err != nil {
		return nil, err
	}
	if !hasAccess {
		return nil, ErrAccessDenied
	}

	ws, err := s.repo.Get(ctx, wsID)
	if err != nil {
		return nil, err
	}
	if ws == nil {
		return nil, ErrWorkspaceNotFound
	}

	return ws, nil
}

func (s *Service) Update(ctx context.Context, workspaceID string, dto model.UpdateWorkspaceDto, userID string, userRole model.UserRole) (*model.Workspace, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	hasAccess, err := s.repo.CheckAccess(ctx, wsID, uid, userRole)
	if err != nil {
		return nil, err
	}
	if !hasAccess {
		return nil, ErrAccessDenied
	}

	ws, err := s.repo.Update(ctx, wsID, dto)
	if err != nil {
		return nil, err
	}
	if ws == nil {
		return nil, ErrWorkspaceNotFound
	}

	return ws, nil
}

func (s *Service) Delete(ctx context.Context, workspaceID, userID string, userRole model.UserRole) error {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return err
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return err
	}

	hasAccess, err := s.repo.CheckAccess(ctx, wsID, uid, userRole)
	if err != nil {
		return err
	}
	if !hasAccess {
		return ErrAccessDenied
	}

	return s.repo.Delete(ctx, wsID)
}
