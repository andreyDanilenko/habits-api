package journal

import (
	"context"
	"time"

	"backend/internal/model"
	activityRepo "backend/internal/repository/activity"
	journalRepo "backend/internal/repository/journal"
	"backend/pkg/realtime"

	"github.com/google/uuid"
)

type Service struct {
	repo        *journalRepo.Repository
	activityRepo *activityRepo.Repository
	publisher   realtime.Publisher
}

func NewService(repo *journalRepo.Repository, activityRepo *activityRepo.Repository, publisher realtime.Publisher) *Service {
	if publisher == nil {
		publisher = realtime.NoopPublisher{}
	}
	return &Service{repo: repo, activityRepo: activityRepo, publisher: publisher}
}

func (s *Service) emitActivityEvent(ctx context.Context, workspaceID string) {
	ch := realtime.WorkspaceChannel(workspaceID)
	event := realtime.Event{
		EventType: realtime.EventActivityCreated,
		Target:    realtime.Target{Type: "workspace", ID: workspaceID},
		Payload:   map[string]interface{}{},
		Timestamp: time.Now().UnixMilli(),
	}
	_ = s.publisher.Publish(ctx, ch, event)
}

func (s *Service) List(ctx context.Context, workspaceID string, date *time.Time) ([]model.JournalEntry, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	return s.repo.List(ctx, wsID, date)
}

func (s *Service) Get(ctx context.Context, workspaceID, entryID string) (*model.JournalEntry, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(entryID)
	if err != nil {
		return nil, err
	}
	return s.repo.Get(ctx, id, wsID)
}

func (s *Service) Create(ctx context.Context, workspaceID, userID string, dto model.CreateJournalEntryDto) (*model.JournalEntry, error) {
	date := dto.Date
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	contentType := dto.ContentType
	if contentType == "" {
		contentType = "text"
	}
	e := &model.JournalEntry{
		WorkspaceID: workspaceID,
		UserID:      userID,
		Description: dto.Description,
		Mood:        dto.Mood,
		Date:        date,
		Tags:        dto.Tags,
		ContentType: contentType,
		Metadata:    dto.Metadata,
	}
	if err := s.repo.Create(ctx, e); err != nil {
		return nil, err
	}
	uid, _ := uuid.Parse(e.UserID)
	wsID, _ := uuid.Parse(e.WorkspaceID)
	entryID, _ := uuid.Parse(e.ID)
	_ = s.activityRepo.Create(ctx, uid, wsID, activityRepo.TypeJournalCreated, activityRepo.EntityJournal, entryID,
		"Добавлена запись в дневник", "📝")
	s.emitActivityEvent(ctx, workspaceID)
	return e, nil
}

func (s *Service) Update(ctx context.Context, workspaceID, entryID string, dto model.UpdateJournalEntryDto) (*model.JournalEntry, error) {
	existing, err := s.repo.Get(ctx, uuid.MustParse(entryID), uuid.MustParse(workspaceID))
	if err != nil || existing == nil {
		return nil, err
	}
	if dto.Description != nil {
		existing.Description = *dto.Description
	}
	if dto.Mood != nil {
		existing.Mood = dto.Mood
	}
	if dto.Date != nil {
		existing.Date = *dto.Date
	}
	if dto.Tags != nil {
		existing.Tags = dto.Tags
	}
	if dto.ContentType != nil {
		existing.ContentType = *dto.ContentType
	}
	if dto.Metadata != nil {
		existing.Metadata = dto.Metadata
	}
	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}
	uid, _ := uuid.Parse(existing.UserID)
	wsID, _ := uuid.Parse(existing.WorkspaceID)
	entryUUID, _ := uuid.Parse(existing.ID)
	_ = s.activityRepo.Create(ctx, uid, wsID, activityRepo.TypeJournalUpdated, activityRepo.EntityJournal, entryUUID,
		"Обновлена запись в дневнике", "✏️")
	s.emitActivityEvent(ctx, workspaceID)
	return existing, nil
}

func (s *Service) Delete(ctx context.Context, workspaceID, entryID string) error {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(entryID)
	if err != nil {
		return err
	}
	entry, _ := s.repo.Get(ctx, id, wsID)
	if err := s.repo.Delete(ctx, id, wsID); err != nil {
		return err
	}
	if entry != nil {
		uid, _ := uuid.Parse(entry.UserID)
		_ = s.activityRepo.Create(ctx, uid, wsID, activityRepo.TypeJournalDeleted, activityRepo.EntityJournal, id,
			"Удалена запись из дневника", "🗑️")
		s.emitActivityEvent(ctx, workspaceID)
	}
	return nil
}
