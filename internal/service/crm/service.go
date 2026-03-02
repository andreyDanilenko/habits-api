package crm

import (
	"context"
	"errors"

	"backend/internal/model"
	crmRepo "backend/internal/repository/crm"
	userRepo "backend/internal/repository/user"
	workspaceService "backend/internal/service/workspace"

	"github.com/google/uuid"
)

const systemUserID = "00000000-0000-0000-0000-000000000001"

var (
	ErrContactHasDeals   = errors.New("contact has linked deals")
	ErrCompanyHasDeals   = errors.New("company has linked deals or contacts")
	ErrActivityNotNote   = errors.New("cannot update or delete non-note activity")
	ErrEntityNotFound    = errors.New("entity not found")
)

type Service struct {
	repo     *crmRepo.Repository
	wsSvc    *workspaceService.Service
	userRepo *userRepo.PostgresUserRepository
}

func NewService(repo *crmRepo.Repository, wsSvc *workspaceService.Service, userRepo *userRepo.PostgresUserRepository) *Service {
	return &Service{repo: repo, wsSvc: wsSvc, userRepo: userRepo}
}

func (s *Service) resolveUserName(ctx context.Context, userID string) string {
	if userID == "" || userID == systemUserID {
		return "Система"
	}
	u, err := s.userRepo.FindByID(ctx, userID)
	if err != nil || u == nil {
		return "Система"
	}
	if u.Name != nil && *u.Name != "" {
		return *u.Name
	}
	return u.Email
}

func (s *Service) ContactList(ctx context.Context, workspaceID string, opts crmRepo.ContactListOpts) ([]model.Contact, int, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, 0, err
	}
	return s.repo.ContactList(ctx, wsID, opts)
}

func (s *Service) ContactGet(ctx context.Context, workspaceID, id string) (*model.Contact, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	return s.repo.ContactGet(ctx, uid, wsID)
}

func (s *Service) ContactCreate(ctx context.Context, workspaceID string, c *model.Contact, userID string) error {
	if c.OwnerID == "" {
		c.OwnerID = userID
	}
	if err := s.repo.ContactCreate(ctx, workspaceID, c, userID); err != nil {
		return err
	}
	_ = s.CreateSystemActivity(ctx, workspaceID, &model.CrmActivity{
		Type:       "contact_created",
		EntityType: "contact",
		EntityID:   c.ID,
		Title:      "Контакт создан",
	}, userID)
	return nil
}

func (s *Service) ContactUpdate(ctx context.Context, workspaceID string, c *model.Contact, userID string) error {
	if err := s.repo.ContactUpdate(ctx, workspaceID, c, userID); err != nil {
		return err
	}
	_ = s.CreateSystemActivity(ctx, workspaceID, &model.CrmActivity{
		Type:       "contact_updated",
		EntityType: "contact",
		EntityID:   c.ID,
		Title:      "Контакт обновлён",
	}, userID)
	return nil
}

func (s *Service) ContactDelete(ctx context.Context, workspaceID, id string) error {
	wsID, _ := uuid.Parse(workspaceID)
	uid, _ := uuid.Parse(id)
	count, err := s.repo.ContactCountDeals(ctx, uid, wsID)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrContactHasDeals
	}
	return s.repo.ContactDelete(ctx, uid, wsID)
}

func (s *Service) CompanyList(ctx context.Context, workspaceID string, opts crmRepo.CompanyListOpts) ([]model.Company, int, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, 0, err
	}
	return s.repo.CompanyList(ctx, wsID, opts)
}

func (s *Service) CompanyGet(ctx context.Context, workspaceID, id string) (*model.Company, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	return s.repo.CompanyGet(ctx, uid, wsID)
}

func (s *Service) CompanyCreate(ctx context.Context, workspaceID string, co *model.Company, userID string) error {
	if co.OwnerID == "" {
		co.OwnerID = userID
	}
	if err := s.repo.CompanyCreate(ctx, workspaceID, co, userID); err != nil {
		return err
	}
	_ = s.CreateSystemActivity(ctx, workspaceID, &model.CrmActivity{
		Type:       "company_created",
		EntityType: "company",
		EntityID:   co.ID,
		Title:      "Компания создана",
	}, userID)
	return nil
}

func (s *Service) CompanyUpdate(ctx context.Context, workspaceID string, co *model.Company, userID string) error {
	if err := s.repo.CompanyUpdate(ctx, workspaceID, co); err != nil {
		return err
	}
	_ = s.CreateSystemActivity(ctx, workspaceID, &model.CrmActivity{
		Type:       "company_updated",
		EntityType: "company",
		EntityID:   co.ID,
		Title:      "Изменены реквизиты компании",
	}, userID)
	return nil
}

func (s *Service) CompanyDelete(ctx context.Context, workspaceID, id string) error {
	wsID, _ := uuid.Parse(workspaceID)
	uid, _ := uuid.Parse(id)
	deals, err := s.repo.CompanyCountDeals(ctx, uid, wsID)
	if err != nil {
		return err
	}
	if deals > 0 {
		return ErrCompanyHasDeals
	}
	contacts, err := s.repo.CompanyCountContacts(ctx, uid)
	if err != nil {
		return err
	}
	if contacts > 0 {
		return ErrCompanyHasDeals
	}
	return s.repo.CompanyDelete(ctx, uid, wsID)
}

func (s *Service) CompanyAttachContact(ctx context.Context, workspaceID, companyID, contactID string, position string) error {
	cid, _ := uuid.Parse(companyID)
	coid, _ := uuid.Parse(contactID)
	return s.repo.CompanyAttachContact(ctx, cid, coid, position)
}

func (s *Service) CompanyDetachContact(ctx context.Context, workspaceID, companyID, contactID string) error {
	cid, _ := uuid.Parse(companyID)
	coid, _ := uuid.Parse(contactID)
	return s.repo.CompanyDetachContact(ctx, cid, coid)
}

func (s *Service) PipelineList(ctx context.Context, workspaceID string) ([]model.Pipeline, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	return s.repo.PipelineList(ctx, wsID)
}

func (s *Service) PipelineCreate(ctx context.Context, workspaceID string, p *model.Pipeline, userID string) error {
	return s.repo.PipelineCreate(ctx, workspaceID, p, userID)
}

func (s *Service) DealList(ctx context.Context, workspaceID string, opts crmRepo.DealListOpts) ([]model.Deal, int, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, 0, err
	}
	return s.repo.DealList(ctx, wsID, opts)
}

func (s *Service) DealGet(ctx context.Context, workspaceID, id string) (*model.Deal, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	return s.repo.DealGet(ctx, uid, wsID)
}

func (s *Service) DealCreate(ctx context.Context, workspaceID string, d *model.Deal, userID string) error {
	if d.Currency == "" {
		d.Currency = "RUB"
	}
	if d.Budget == 0 && d.Currency != "" {
		d.Budget = 0
	}
	if d.OwnerID == "" {
		d.OwnerID = userID
	}
	if err := s.repo.DealCreate(ctx, workspaceID, d, userID); err != nil {
		return err
	}
	stageName := ""
	if stageID, err := uuid.Parse(d.StageID); err == nil {
		if st, _ := s.repo.StageGet(ctx, stageID); st != nil {
			stageName = st.Name
		}
	}
	title := "Сделка создана"
	if stageName != "" {
		title = "Сделка создана на этапе " + stageName
	}
	_ = s.CreateSystemActivity(ctx, workspaceID, &model.CrmActivity{
		Type:       "deal_created",
		EntityType: "deal",
		EntityID:   d.ID,
		Title:      title,
	}, userID)
	return nil
}

func (s *Service) DealUpdate(ctx context.Context, workspaceID string, d *model.Deal, userID string) error {
	oldDeal, _ := s.repo.DealGet(ctx, uuid.MustParse(d.ID), uuid.MustParse(workspaceID))
	if err := s.repo.DealUpdate(ctx, workspaceID, d); err != nil {
		return err
	}
	if oldDeal != nil && oldDeal.StageID != d.StageID {
		oldStage, _ := s.repo.StageGet(ctx, uuid.MustParse(oldDeal.StageID))
		newStage, _ := s.repo.StageGet(ctx, uuid.MustParse(d.StageID))
		fromName, toName := "", ""
		if oldStage != nil {
			fromName = oldStage.Name
		}
		if newStage != nil {
			toName = newStage.Name
		}
		title := "Сделка перешла: " + fromName + " → " + toName
		meta := map[string]interface{}{
			"fromStage":  map[string]interface{}{"id": oldDeal.StageID, "name": fromName},
			"toStage":    map[string]interface{}{"id": d.StageID, "name": toName},
			"dealValue":  d.Budget,
		}
		_ = s.CreateSystemActivity(ctx, workspaceID, &model.CrmActivity{
			Type:       "deal_stage_changed",
			EntityType: "deal",
			EntityID:   d.ID,
			Title:      title,
			Metadata:   meta,
		}, userID)
	}
	return nil
}

func (s *Service) DealDelete(ctx context.Context, workspaceID, id string) error {
	wsID, _ := uuid.Parse(workspaceID)
	uid, _ := uuid.Parse(id)
	return s.repo.DealDelete(ctx, uid, wsID)
}

// Activity (SPEC_BACK_2)
func (s *Service) ActivityList(ctx context.Context, workspaceID string, entityType, entityID string, opts crmRepo.ActivityListOpts) ([]model.CrmActivity, int, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, 0, err
	}
	return s.repo.ActivityList(ctx, wsID, entityType, entityID, opts)
}

func (s *Service) ActivityGet(ctx context.Context, workspaceID, id string) (*model.CrmActivity, error) {
	wsID, _ := uuid.Parse(workspaceID)
	uid, _ := uuid.Parse(id)
	return s.repo.ActivityGet(ctx, uid, wsID)
}

func (s *Service) activityEnsureEntityExists(ctx context.Context, workspaceID, entityType, entityID string) error {
	wsID, _ := uuid.Parse(workspaceID)
	eid, _ := uuid.Parse(entityID)
	switch entityType {
	case "contact":
		c, _ := s.repo.ContactGet(ctx, eid, wsID)
		if c == nil {
			return ErrEntityNotFound
		}
	case "company":
		co, _ := s.repo.CompanyGet(ctx, eid, wsID)
		if co == nil {
			return ErrEntityNotFound
		}
	case "deal":
		d, _ := s.repo.DealGet(ctx, eid, wsID)
		if d == nil {
			return ErrEntityNotFound
		}
	default:
		return ErrEntityNotFound
	}
	return nil
}

func (s *Service) ActivityCreateNote(ctx context.Context, workspaceID string, a *model.CrmActivity, userID string) error {
	if err := s.activityEnsureEntityExists(ctx, workspaceID, a.EntityType, a.EntityID); err != nil {
		return err
	}
	a.Type = "note"
	a.IsEditable = true
	a.IsDeletable = true
	return s.repo.ActivityCreate(ctx, workspaceID, a, userID, s.resolveUserName(ctx, userID))
}

func (s *Service) ActivityCreateCall(ctx context.Context, workspaceID string, a *model.CrmActivity, userID string) error {
	if err := s.activityEnsureEntityExists(ctx, workspaceID, a.EntityType, a.EntityID); err != nil {
		return err
	}
	a.Type = "call"
	a.IsEditable = false
	a.IsDeletable = false
	return s.repo.ActivityCreate(ctx, workspaceID, a, userID, s.resolveUserName(ctx, userID))
}

func (s *Service) ActivityUpdate(ctx context.Context, workspaceID string, a *model.CrmActivity) error {
	existing, err := s.repo.ActivityGet(ctx, uuid.MustParse(a.ID), uuid.MustParse(workspaceID))
	if err != nil || existing == nil {
		return err
	}
	if existing.Type != "note" || !existing.IsEditable {
		return ErrActivityNotNote
	}
	return s.repo.ActivityUpdate(ctx, workspaceID, a)
}

func (s *Service) ActivityDelete(ctx context.Context, workspaceID, id string) error {
	wsID, _ := uuid.Parse(workspaceID)
	uid, _ := uuid.Parse(id)
	existing, err := s.repo.ActivityGet(ctx, uid, wsID)
	if err != nil || existing == nil {
		return err
	}
	if existing.Type != "note" || !existing.IsDeletable {
		return ErrActivityNotNote
	}
	return s.repo.ActivityDelete(ctx, uid, wsID)
}

func (s *Service) ActivitySetImportant(ctx context.Context, workspaceID, id string, isImportant bool) error {
	wsID, _ := uuid.Parse(workspaceID)
	uid, _ := uuid.Parse(id)
	return s.repo.ActivitySetImportant(ctx, uid, wsID, isImportant)
}

func (s *Service) CreateSystemActivity(ctx context.Context, workspaceID string, a *model.CrmActivity, userID string) error {
	if userID == "" {
		userID = systemUserID
	}
	name := s.resolveUserName(ctx, userID)
	if name == "Система" && userID == systemUserID {
		name = "Система"
	}
	return s.repo.CreateSystemActivity(ctx, workspaceID, a, userID, name)
}
