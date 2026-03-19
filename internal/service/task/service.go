package task

import (
	"context"
	"database/sql"
	"fmt"

	"backend/internal/model"
	activityRepo "backend/internal/repository/activity"
	taskRepo "backend/internal/repository/task"

	"github.com/google/uuid"
)

type Service struct {
	repo         *taskRepo.Repository
	activityRepo *activityRepo.Repository
}

func NewService(repo *taskRepo.Repository, activityRepo *activityRepo.Repository) *Service {
	return &Service{repo: repo, activityRepo: activityRepo}
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
	_ = s.activityRepo.Create(ctx, creatorID, wsID, activityRepo.TypeTaskCreated, activityRepo.EntityTask, uuid.MustParse(t.ID), "Создал задачу", "📋")
	return s.repo.Get(ctx, uuid.MustParse(t.ID), wsID)
}

func (s *Service) Update(ctx context.Context, workspaceID, taskID, userID string, dto model.UpdateTaskDto) (*model.Task, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	tID, err := uuid.Parse(taskID)
	if err != nil {
		return nil, err
	}
	uid, _ := uuid.Parse(userID)

	existing, err := s.repo.Get(ctx, tID, wsID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, sql.ErrNoRows
	}

	oldTotalSec := existing.SpentMinutes*60 + existing.SpentSeconds
	applyUpdate(existing, dto)
	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, err
	}
	newTotalSec := existing.SpentMinutes*60 + existing.SpentSeconds
	s.emitTaskUpdateActivity(ctx, uid, wsID, tID, dto, oldTotalSec, newTotalSec)
	return s.repo.Get(ctx, tID, wsID)
}

func (s *Service) Delete(ctx context.Context, workspaceID, taskID, userID string) error {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return err
	}
	tID, err := uuid.Parse(taskID)
	if err != nil {
		return err
	}
	uid, _ := uuid.Parse(userID)
	if err := s.repo.Delete(ctx, tID, wsID); err != nil {
		return err
	}
	_ = s.activityRepo.Create(ctx, uid, wsID, activityRepo.TypeTaskDeleted, activityRepo.EntityTask, tID, "Удалил задачу", "🗑️")
	return nil
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
	t, err := s.repo.Complete(ctx, tID, wsID, userID, note)
	if err != nil {
		return nil, err
	}
	if t != nil {
		_ = s.activityRepo.Create(ctx, userID, wsID, activityRepo.TypeTaskCompleted, activityRepo.EntityTask, tID, "Отметил задачу выполненной", "✅")
	}
	return t, err
}

func (s *Service) Reopen(ctx context.Context, workspaceID, taskID, userID string) (*model.Task, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	tID, err := uuid.Parse(taskID)
	if err != nil {
		return nil, err
	}
	uid, _ := uuid.Parse(userID)
	t, err := s.repo.Reopen(ctx, tID, wsID)
	if err != nil {
		return nil, err
	}
	if t != nil {
		_ = s.activityRepo.Create(ctx, uid, wsID, activityRepo.TypeTaskReopened, activityRepo.EntityTask, tID, "Вернул задачу в работу", "🔄")
	}
	return t, err
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

func (s *Service) ListTaskLinks(ctx context.Context, workspaceID, taskID string) ([]model.TaskTaskLink, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	tID, err := uuid.Parse(taskID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListTaskLinks(ctx, tID, wsID)
}

func (s *Service) AddTaskLink(ctx context.Context, workspaceID, taskID string, dto model.CreateTaskTaskLinkDto) (*model.TaskTaskLink, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	tID, err := uuid.Parse(taskID)
	if err != nil {
		return nil, err
	}
	linkedID, err := uuid.Parse(dto.LinkedTaskID)
	if err != nil {
		return nil, err
	}
	return s.repo.AddTaskLink(ctx, tID, wsID, linkedID, dto.LinkType)
}

func (s *Service) DeleteTaskLink(ctx context.Context, workspaceID, linkID string) error {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return err
	}
	lID, err := uuid.Parse(linkID)
	if err != nil {
		return err
	}
	return s.repo.DeleteTaskLink(ctx, lID, wsID)
}

func (s *Service) ListAttachments(ctx context.Context, workspaceID, taskID string) ([]model.TaskAttachment, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	tID, err := uuid.Parse(taskID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListAttachments(ctx, tID, wsID)
}

func (s *Service) CreateAttachment(ctx context.Context, workspaceID string, a *model.TaskAttachment) error {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return err
	}
	return s.repo.CreateAttachment(ctx, wsID, a)
}

func (s *Service) GetAttachment(ctx context.Context, workspaceID, attachmentID string) (*model.TaskAttachment, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	aID, err := uuid.Parse(attachmentID)
	if err != nil {
		return nil, err
	}
	return s.repo.GetAttachment(ctx, aID, wsID)
}

func (s *Service) DeleteAttachment(ctx context.Context, workspaceID, attachmentID string) error {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return err
	}
	aID, err := uuid.Parse(attachmentID)
	if err != nil {
		return err
	}
	return s.repo.DeleteAttachment(ctx, aID, wsID)
}

func (s *Service) emitTaskUpdateActivity(ctx context.Context, userID, workspaceID, taskID uuid.UUID, dto model.UpdateTaskDto, oldTotalSec, newTotalSec int) {
	var title string
	var emoji string
	if (dto.SpentMinutes != nil || dto.SpentSeconds != nil) && newTotalSec > oldTotalSec {
		delta := newTotalSec - oldTotalSec
		if delta < 60 {
			title = fmt.Sprintf("Добавил %d сек", delta)
		} else if delta%60 == 0 {
			title = fmt.Sprintf("Добавил %d мин", delta/60)
		} else {
			title = fmt.Sprintf("Добавил %d мин %d сек", delta/60, delta%60)
		}
		emoji = "⏱"
	} else if dto.Status != nil {
		title = "Изменил статус"
		emoji = "📝"
	} else if dto.Priority != nil {
		title = "Изменил приоритет"
		emoji = "📝"
	} else if dto.AssigneeID != nil {
		title = "Изменил исполнителя"
		emoji = "👤"
	} else if dto.DueDate != nil || dto.DueTime != nil {
		title = "Изменил срок"
		emoji = "📅"
	} else if dto.Title != nil || dto.Description != nil || dto.Type != nil || dto.Entities != nil {
		title = "Изменил задачу"
		emoji = "✏️"
	} else {
		return
	}
	_ = s.activityRepo.Create(ctx, userID, workspaceID, activityRepo.TypeTaskUpdated, activityRepo.EntityTask, taskID, title, emoji)
}

func (s *Service) ListTaskActivities(ctx context.Context, workspaceID, taskID string, limit, offset int) ([]model.Activity, int, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, 0, err
	}
	tID, err := uuid.Parse(taskID)
	if err != nil {
		return nil, 0, err
	}
	return s.activityRepo.ListByEntity(ctx, wsID, activityRepo.EntityTask, tID, limit, offset)
}

func (s *Service) GetTaskActivitiesCount(ctx context.Context, workspaceID, taskID string) (int, error) {
	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return 0, err
	}
	tID, err := uuid.Parse(taskID)
	if err != nil {
		return 0, err
	}
	return s.activityRepo.CountByEntity(ctx, wsID, activityRepo.EntityTask, tID)
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
	if dto.SpentMinutes != nil {
		t.SpentMinutes = *dto.SpentMinutes
	}
	if dto.SpentSeconds != nil {
		t.SpentSeconds = *dto.SpentSeconds
	}
	if dto.AssigneeID != nil {
		t.AssigneeID = *dto.AssigneeID
	}
	if dto.Tags != nil {
		t.Tags = dto.Tags
	}
	if dto.Entities != nil {
		t.Entities = dto.Entities
	}
}
