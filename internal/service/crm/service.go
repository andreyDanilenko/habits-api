package crm

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"backend/internal/model"
	crmRepo "backend/internal/repository/crm"
	userRepo "backend/internal/repository/user"
	workspaceService "backend/internal/service/workspace"
	"backend/pkg/realtime"

	"github.com/google/uuid"
)

const systemUserID = "00000000-0000-0000-0000-000000000001"

var (
	ErrContactHasDeals = errors.New("contact has linked deals")
	ErrCompanyHasDeals = errors.New("company has linked deals or contacts")
	ErrActivityNotNote = errors.New("cannot update or delete non-note activity")
	ErrEntityNotFound  = errors.New("entity not found")
)

type Service struct {
	repo      *crmRepo.Repository
	wsSvc     *workspaceService.Service
	userRepo  *userRepo.PostgresUserRepository
	publisher realtime.Publisher
}

func NewService(repo *crmRepo.Repository, wsSvc *workspaceService.Service, userRepo *userRepo.PostgresUserRepository, publisher realtime.Publisher) *Service {
	if publisher == nil {
		publisher = realtime.NoopPublisher{}
	}
	return &Service{repo: repo, wsSvc: wsSvc, userRepo: userRepo, publisher: publisher}
}

func (s *Service) emitDealEvent(ctx context.Context, workspaceID, eventType string, payload map[string]interface{}) {
	ch := realtime.WorkspaceChannel(workspaceID)
	event := realtime.Event{
		EventType: eventType,
		Target:    realtime.Target{Type: "workspace", ID: workspaceID},
		Payload:   payload,
		Timestamp: time.Now().UnixMilli(),
	}
	_ = s.publisher.Publish(ctx, ch, event)
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

func (s *Service) PipelineGet(ctx context.Context, workspaceID, id string) (*model.Pipeline, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	pid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	return s.repo.PipelineGetByID(ctx, pid, wsID)
}

func (s *Service) PipelineUpdate(ctx context.Context, workspaceID string, p *model.Pipeline) error {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return err
	}
	pid, err := uuid.Parse(p.ID)
	if err != nil {
		return err
	}
	return s.repo.PipelineUpdate(ctx, wsID, pid, p)
}

func (s *Service) PipelineDelete(ctx context.Context, workspaceID, id string) error {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return err
	}
	pid, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return s.repo.PipelineDelete(ctx, wsID, pid)
}

func (s *Service) StageList(ctx context.Context, workspaceID, pipelineID string) ([]model.Stage, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	pid, err := uuid.Parse(pipelineID)
	if err != nil {
		return nil, err
	}
	p, err := s.repo.PipelineGetByID(ctx, pid, wsID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, nil
	}
	return p.Stages, nil
}

func (s *Service) StageGet(ctx context.Context, workspaceID, pipelineID, stageID string) (*model.Stage, error) {
	stages, err := s.StageList(ctx, workspaceID, pipelineID)
	if err != nil {
		return nil, err
	}
	if stages == nil {
		return nil, nil
	}
	for i := range stages {
		if stages[i].ID == stageID {
			return &stages[i], nil
		}
	}
	return nil, nil
}

func (s *Service) StageCreate(ctx context.Context, workspaceID, pipelineID string, st *model.Stage) error {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return err
	}
	pid, err := uuid.Parse(pipelineID)
	if err != nil {
		return err
	}
	// ensure pipeline belongs to workspace
	p, err := s.repo.PipelineGetByID(ctx, pid, wsID)
	if err != nil {
		return err
	}
	if p == nil {
		return sql.ErrNoRows
	}
	return s.repo.StageCreate(ctx, pid, st)
}

func (s *Service) StageUpdate(ctx context.Context, workspaceID, pipelineID, stageID string, st *model.Stage) error {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return err
	}
	pid, err := uuid.Parse(pipelineID)
	if err != nil {
		return err
	}
	// ensure pipeline belongs to workspace
	p, err := s.repo.PipelineGetByID(ctx, pid, wsID)
	if err != nil {
		return err
	}
	if p == nil {
		return sql.ErrNoRows
	}
	sid, err := uuid.Parse(stageID)
	if err != nil {
		return err
	}
	return s.repo.StageUpdate(ctx, pid, sid, st)
}

func (s *Service) StageDelete(ctx context.Context, workspaceID, pipelineID, stageID string) error {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return err
	}
	pid, err := uuid.Parse(pipelineID)
	if err != nil {
		return err
	}
	// ensure pipeline belongs to workspace
	p, err := s.repo.PipelineGetByID(ctx, pid, wsID)
	if err != nil {
		return err
	}
	if p == nil {
		return sql.ErrNoRows
	}
	sid, err := uuid.Parse(stageID)
	if err != nil {
		return err
	}
	return s.repo.StageDelete(ctx, pid, sid)
}

func (s *Service) StageReorder(ctx context.Context, workspaceID, pipelineID string, stageIDs []string) error {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return err
	}
	pid, err := uuid.Parse(pipelineID)
	if err != nil {
		return err
	}
	// ensure pipeline belongs to workspace
	p, err := s.repo.PipelineGetByID(ctx, pid, wsID)
	if err != nil {
		return err
	}
	if p == nil {
		return sql.ErrNoRows
	}
	ids := make([]uuid.UUID, 0, len(stageIDs))
	for _, sidStr := range stageIDs {
		sid, err := uuid.Parse(sidStr)
		if err != nil {
			return err
		}
		ids = append(ids, sid)
	}
	return s.repo.StageReorder(ctx, pid, ids)
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
	payload := map[string]interface{}{"deal": d}
	if stageName != "" {
		payload["stageName"] = stageName
	}
	s.emitDealEvent(ctx, workspaceID, realtime.EventDealCreated, payload)
	return nil
}

func (s *Service) DealUpdate(ctx context.Context, workspaceID string, d *model.Deal, userID string) error {
	oldDeal, _ := s.repo.DealGet(ctx, uuid.MustParse(d.ID), uuid.MustParse(workspaceID))
	if err := s.repo.DealUpdate(ctx, workspaceID, d); err != nil {
		return err
	}
	stageName := ""
	stageFromName, stageToName := "", ""
	if oldDeal != nil && oldDeal.StageID != d.StageID {
		oldStage, _ := s.repo.StageGet(ctx, uuid.MustParse(oldDeal.StageID))
		newStage, _ := s.repo.StageGet(ctx, uuid.MustParse(d.StageID))
		if oldStage != nil {
			stageFromName = oldStage.Name
		}
		if newStage != nil {
			stageToName = newStage.Name
			stageName = stageToName
		}
		title := "Сделка перешла: " + stageFromName + " → " + stageToName
		meta := map[string]interface{}{
			"fromStage": map[string]interface{}{"id": oldDeal.StageID, "name": stageFromName},
			"toStage":   map[string]interface{}{"id": d.StageID, "name": stageToName},
			"dealValue": d.Budget,
		}
		_ = s.CreateSystemActivity(ctx, workspaceID, &model.CrmActivity{
			Type:       "deal_stage_changed",
			EntityType: "deal",
			EntityID:   d.ID,
			Title:      title,
			Metadata:   meta,
		}, userID)
	} else if stageID, err := uuid.Parse(d.StageID); err == nil {
		if st, _ := s.repo.StageGet(ctx, stageID); st != nil {
			stageName = st.Name
		}
	}
	payload := map[string]interface{}{"deal": d}
	if stageName != "" {
		payload["stageName"] = stageName
	}
	if stageFromName != "" || stageToName != "" {
		payload["stageFromName"] = stageFromName
		payload["stageToName"] = stageToName
	}
	s.emitDealEvent(ctx, workspaceID, realtime.EventDealUpdated, payload)
	return nil
}

func (s *Service) DealDelete(ctx context.Context, workspaceID, id string) error {
	wsID, _ := uuid.Parse(workspaceID)
	uid, _ := uuid.Parse(id)
	if err := s.repo.DealDelete(ctx, uid, wsID); err != nil {
		return err
	}
	s.emitDealEvent(ctx, workspaceID, realtime.EventDealDeleted, map[string]interface{}{
		"dealId": id,
	})
	return nil
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
