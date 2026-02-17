package journal

import (
	"context"
	"time"

	"backend/internal/model"
	journalRepo "backend/internal/repository/journal"

	"github.com/google/uuid"
)

type Service struct {
	repo *journalRepo.Repository
}

func NewService(repo *journalRepo.Repository) *Service {
	return &Service{repo: repo}
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
	return s.repo.Delete(ctx, id, wsID)
}
