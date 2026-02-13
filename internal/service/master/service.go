package master

import (
	"context"

	"backend/internal/model"
	masterRepo "backend/internal/repository/master"

	"github.com/google/uuid"
)

type Service struct {
	repo *masterRepo.Repository
}

func NewService(repo *masterRepo.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListCurrencies(ctx context.Context, workspaceID string) ([]model.Currency, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListCurrencies(ctx, wsID)
}

func (s *Service) GetCurrency(ctx context.Context, workspaceID, id string) (*model.Currency, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	return s.repo.GetCurrency(ctx, uid, wsID)
}

func (s *Service) CreateCurrency(ctx context.Context, c *model.Currency) error {
	return s.repo.CreateCurrency(ctx, c)
}

func (s *Service) UpdateCurrency(ctx context.Context, c *model.Currency) error {
	return s.repo.UpdateCurrency(ctx, c)
}

func (s *Service) DeleteCurrency(ctx context.Context, workspaceID, id string) error {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return err
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return s.repo.DeleteCurrency(ctx, uid, wsID)
}

func (s *Service) ListCounterparties(ctx context.Context, workspaceID string) ([]model.Counterparty, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListCounterparties(ctx, wsID)
}

func (s *Service) GetCounterparty(ctx context.Context, workspaceID, id string) (*model.Counterparty, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	return s.repo.GetCounterparty(ctx, uid, wsID)
}

func (s *Service) CreateCounterparty(ctx context.Context, cp *model.Counterparty) error {
	return s.repo.CreateCounterparty(ctx, cp)
}

func (s *Service) UpdateCounterparty(ctx context.Context, cp *model.Counterparty) error {
	return s.repo.UpdateCounterparty(ctx, cp)
}

func (s *Service) DeleteCounterparty(ctx context.Context, workspaceID, id string) error {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return err
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return s.repo.DeleteCounterparty(ctx, uid, wsID)
}
