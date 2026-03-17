package task

import (
	"context"
	"database/sql"

	"backend/internal/model"
	taskRepo "backend/internal/repository/task"

	"github.com/google/uuid"
)

type Service struct {
	repo *taskRepo.Repository
}

func NewService(repo *taskRepo.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, workspaceID string, filters *model.TaskListFilters) ([]model.Task, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	return s.repo.List(ctx, wsID, filters)
}

func (s *Service) Get(ctx context.Context, workspaceID, taskID string) (*model.Task, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	tID, err := uuid.Parse(taskID)
	if err != nil {
		return nil, err
	}
	return s.repo.Get(ctx, tID, wsID)
}

func (s *Service) Create(ctx context.Context, workspaceID, createdBy string, dto model.CreateTaskDto) (*model.Task, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	creatorID, err := uuid.Parse(createdBy)
	if err != nil {
		return nil, err
	}

	status := dto.Status
	if status == "" {
		status = "pending"
	}

	t := &model.Task{
		WorkspaceID:  workspaceID,
		Title:        dto.Title,
		Description:  dto.Description,
		Type:         dto.Type,
		Priority:     dto.Priority,
		Status:       status,
		DueDate:      dto.DueDate,
		DueTime:      dto.DueTime,
		ReminderDate: dto.ReminderDate,
		Duration:     dto.Duration,
		AssigneeID:   dto.AssigneeID,
		ParentID:     dto.ParentID,
		Entities:     dto.Entities,
	}
	if err := s.repo.Create(ctx, t, creatorID); err != nil {
		return nil, err
	}
	return s.repo.Get(ctx, uuid.MustParse(t.ID), wsID)
}

func (s *Service) Update(ctx context.Context, workspaceID, taskID string, dto model.UpdateTaskDto) (*model.Task, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	tID, err := uuid.Parse(taskID)
	if err != nil {
		return nil, err
	}

	existing, err := s.repo.Get(ctx, tID, wsID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, sql.ErrNoRows
	}

	applyUpdate(existing, dto)
	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return s.repo.Get(ctx, tID, wsID)
}

func (s *Service) Delete(ctx context.Context, workspaceID, taskID string) error {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return err
	}
	tID, err := uuid.Parse(taskID)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, tID, wsID)
}

func (s *Service) Complete(ctx context.Context, workspaceID, taskID, completedBy string, note string) (*model.Task, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	tID, err := uuid.Parse(taskID)
	if err != nil {
		return nil, err
	}
	userID, err := uuid.Parse(completedBy)
	if err != nil {
		return nil, err
	}
	return s.repo.Complete(ctx, tID, wsID, userID, note)
}

func (s *Service) Reopen(ctx context.Context, workspaceID, taskID string) (*model.Task, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	tID, err := uuid.Parse(taskID)
	if err != nil {
		return nil, err
	}
	return s.repo.Reopen(ctx, tID, wsID)
}

func (s *Service) ListComments(ctx context.Context, workspaceID, taskID string) ([]model.TaskComment, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	tID, err := uuid.Parse(taskID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListComments(ctx, tID, wsID)
}

func (s *Service) CreateComment(ctx context.Context, workspaceID, taskID, createdBy string, body string, parentID string) (*model.TaskComment, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	tID, err := uuid.Parse(taskID)
	if err != nil {
		return nil, err
	}
	userID, err := uuid.Parse(createdBy)
	if err != nil {
		return nil, err
	}
	var pID *uuid.UUID
	if parentID != "" {
		parsed, err := uuid.Parse(parentID)
		if err == nil {
			pID = &parsed
		}
	}
	return s.repo.CreateComment(ctx, tID, wsID, userID, body, pID)
}

func (s *Service) UpdateComment(ctx context.Context, workspaceID, commentID, userID string, body string) (*model.TaskComment, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	cID, err := uuid.Parse(commentID)
	if err != nil {
		return nil, err
	}
	uID, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	return s.repo.UpdateComment(ctx, cID, wsID, uID, body)
}

func (s *Service) DeleteComment(ctx context.Context, workspaceID, commentID string) error {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return err
	}
	cID, err := uuid.Parse(commentID)
	if err != nil {
		return err
	}
	return s.repo.DeleteComment(ctx, cID, wsID)
}

func applyUpdate(t *model.Task, dto model.UpdateTaskDto) {
	if dto.Title != nil {
		t.Title = *dto.Title
	}
	if dto.Description != nil {
		t.Description = *dto.Description
	}
	if dto.Type != nil {
		t.Type = *dto.Type
	}
	if dto.Priority != nil {
		t.Priority = *dto.Priority
	}
	if dto.Status != nil {
		t.Status = *dto.Status
	}
	if dto.DueDate != nil {
		t.DueDate = *dto.DueDate
	}
	if dto.DueTime != nil {
		t.DueTime = *dto.DueTime
	}
	if dto.ReminderDate != nil {
		t.ReminderDate = *dto.ReminderDate
	}
	if dto.Duration != nil {
		t.Duration = dto.Duration
	}
	if dto.AssigneeID != nil {
		t.AssigneeID = *dto.AssigneeID
	}
	if dto.Entities != nil {
		t.Entities = dto.Entities
	}
}
