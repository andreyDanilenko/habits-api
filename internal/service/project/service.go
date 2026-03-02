package project

import (
	"context"
	"errors"

	"backend/internal/model"
	"backend/internal/repository/project"
	workspaceService "backend/internal/service/workspace"

	"github.com/google/uuid"
)

var (
	ErrProjectNotFound = errors.New("project not found")
	ErrAccessDenied    = errors.New("access denied")
)

// EntityType — типы сущностей для привязки к проектам. Список расширяется с появлением новых модулей.
// Бэкенд не валидирует entity_type по этому списку: в project_entities можно сохранять любую строку,
// чтобы добавление модулей (Tasks, HR и т.д.) не требовало изменений в Core.
const (
	EntityTypeCrmContact = "crm_contact"
	EntityTypeCrmCompany = "crm_company"
	EntityTypeCrmDeal    = "crm_deal"
	// EntityTypeTask      = "task"       // модуль Tasks
	// EntityTypeHrEmployee = "hr_employee" // модуль HR
)

type Service struct {
	repo    *project.Repository
	wsSvc   *workspaceService.Service
}

func NewService(repo *project.Repository, wsSvc *workspaceService.Service) *Service {
	return &Service{
		repo:  repo,
		wsSvc: wsSvc,
	}
}

func (s *Service) ensureAccess(ctx context.Context, workspaceID, userID string, userRole model.UserRole) error {
	hasAccess, err := s.wsSvc.HasAccess(ctx, workspaceID, userID, userRole)
	if err != nil || !hasAccess {
		return ErrAccessDenied
	}
	return nil
}

func (s *Service) List(ctx context.Context, workspaceID, userID string, userRole model.UserRole) ([]model.Project, error) {
	if err := s.ensureAccess(ctx, workspaceID, userID, userRole); err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListByWorkspace(ctx, wsID)
}

func (s *Service) Get(ctx context.Context, workspaceID, projectID, userID string, userRole model.UserRole) (*model.Project, error) {
	if err := s.ensureAccess(ctx, workspaceID, userID, userRole); err != nil {
		return nil, err
	}
	wsID, _ := uuid.Parse(workspaceID)
	prID, err := uuid.Parse(projectID)
	if err != nil {
		return nil, err
	}
	p, err := s.repo.Get(ctx, prID, wsID)
	if err != nil || p == nil {
		return nil, ErrProjectNotFound
	}
	return p, nil
}

func (s *Service) Create(ctx context.Context, workspaceID, userID string, userRole model.UserRole, dto model.CreateProjectDto) (*model.Project, error) {
	if err := s.ensureAccess(ctx, workspaceID, userID, userRole); err != nil {
		return nil, err
	}
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, wsID, dto)
}

func (s *Service) Update(ctx context.Context, workspaceID, projectID, userID string, userRole model.UserRole, dto model.UpdateProjectDto) (*model.Project, error) {
	if err := s.ensureAccess(ctx, workspaceID, userID, userRole); err != nil {
		return nil, err
	}
	wsID, _ := uuid.Parse(workspaceID)
	prID, err := uuid.Parse(projectID)
	if err != nil {
		return nil, err
	}
	p, err := s.repo.Update(ctx, prID, wsID, dto)
	if err != nil || p == nil {
		return nil, ErrProjectNotFound
	}
	return p, nil
}

func (s *Service) Delete(ctx context.Context, workspaceID, projectID, userID string, userRole model.UserRole) error {
	if err := s.ensureAccess(ctx, workspaceID, userID, userRole); err != nil {
		return err
	}
	wsID, _ := uuid.Parse(workspaceID)
	prID, err := uuid.Parse(projectID)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, prID, wsID)
}

func (s *Service) AttachEntity(ctx context.Context, workspaceID, projectID, userID string, userRole model.UserRole, entityType string, entityID string) error {
	if err := s.ensureAccess(ctx, workspaceID, userID, userRole); err != nil {
		return err
	}
	prID, err := uuid.Parse(projectID)
	if err != nil {
		return err
	}
	eID, err := uuid.Parse(entityID)
	if err != nil {
		return err
	}
	// Проверяем, что проект принадлежит workspace
	p, err := s.repo.Get(ctx, prID, uuid.MustParse(workspaceID))
	if err != nil || p == nil {
		return ErrProjectNotFound
	}
	_ = p
	return s.repo.AttachEntity(ctx, prID, entityType, eID)
}

func (s *Service) DetachEntity(ctx context.Context, workspaceID, projectID, userID string, userRole model.UserRole, entityType string, entityID string) error {
	if err := s.ensureAccess(ctx, workspaceID, userID, userRole); err != nil {
		return err
	}
	prID, err := uuid.Parse(projectID)
	if err != nil {
		return err
	}
	eID, err := uuid.Parse(entityID)
	if err != nil {
		return err
	}
	p, err := s.repo.Get(ctx, prID, uuid.MustParse(workspaceID))
	if err != nil || p == nil {
		return ErrProjectNotFound
	}
	_ = p
	return s.repo.DetachEntity(ctx, prID, entityType, eID)
}

// ListEntityIDs возвращает entity_id по проекту и опционально типу (для BFF: запросить у CRM сделки по этим id).
func (s *Service) ListEntityIDs(ctx context.Context, workspaceID, projectID, userID string, userRole model.UserRole, entityType string) ([]string, error) {
	if err := s.ensureAccess(ctx, workspaceID, userID, userRole); err != nil {
		return nil, err
	}
	prID, err := uuid.Parse(projectID)
	if err != nil {
		return nil, err
	}
	p, err := s.repo.Get(ctx, prID, uuid.MustParse(workspaceID))
	if err != nil || p == nil {
		return nil, ErrProjectNotFound
	}
	_ = p
	ids, err := s.repo.ListEntityIDsByProject(ctx, prID, entityType)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(ids))
	for i := range ids {
		out[i] = ids[i].String()
	}
	return out, nil
}

// GetProjectIDsForEntity возвращает project_id, к которым привязана сущность.
func (s *Service) GetProjectIDsForEntity(ctx context.Context, workspaceID, userID string, userRole model.UserRole, entityType string, entityID string) ([]string, error) {
	if err := s.ensureAccess(ctx, workspaceID, userID, userRole); err != nil {
		return nil, err
	}
	eID, err := uuid.Parse(entityID)
	if err != nil {
		return nil, err
	}
	wsID, _ := uuid.Parse(workspaceID)
	ids, err := s.repo.GetProjectIDsForEntity(ctx, wsID, entityType, eID)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(ids))
	for i := range ids {
		out[i] = ids[i].String()
	}
	return out, nil
}
