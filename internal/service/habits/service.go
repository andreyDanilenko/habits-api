package habits

import (
	"context"
	"errors"
	"time"

	"backend/internal/model"
	"backend/internal/repository/habits"

	"github.com/google/uuid"
)

var (
	ErrHabitNotFound   = errors.New("habit not found")
	ErrWorkspaceNeeded = errors.New("workspace not selected")
)

type Service struct {
	repo *habits.Repository
}

func NewService(repo *habits.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, userID, workspaceID string) ([]model.Habit, error) {
	if workspaceID == "" {
		return nil, ErrWorkspaceNeeded
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	return s.repo.List(ctx, uid, wid)
}

func (s *Service) Create(ctx context.Context, dto model.CreateHabitDto, userID, workspaceID string) (*model.Habit, error) {
	if workspaceID == "" {
		return nil, ErrWorkspaceNeeded
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, dto, uid, wid)
}

func (s *Service) Get(ctx context.Context, habitID, userID, workspaceID string) (*model.Habit, error) {
	hid, err := uuid.Parse(habitID)
	if err != nil {
		return nil, err
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	h, err := s.repo.Get(ctx, hid, uid)
	if err != nil || h == nil {
		return nil, ErrHabitNotFound
	}
	if workspaceID != "" && h.WorkspaceID != workspaceID {
		return nil, ErrHabitNotFound
	}
	return h, nil
}

func (s *Service) Update(ctx context.Context, habitID string, dto model.UpdateHabitDto, userID, workspaceID string) (*model.Habit, error) {
	_, err := s.Get(ctx, habitID, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	hid, _ := uuid.Parse(habitID)
	uid, _ := uuid.Parse(userID)
	return s.repo.Update(ctx, hid, uid, dto)
}

func (s *Service) Delete(ctx context.Context, habitID, userID, workspaceID string) error {
	_, err := s.Get(ctx, habitID, userID, workspaceID)
	if err != nil {
		return err
	}
	hid, _ := uuid.Parse(habitID)
	uid, _ := uuid.Parse(userID)
	return s.repo.Delete(ctx, hid, uid)
}

func (s *Service) Complete(ctx context.Context, habitID, userID, workspaceID string, date time.Time, notes string, rating interface{}, completionTime *string) (*model.HabitCompletion, error) {
	_, err := s.Get(ctx, habitID, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	hid, _ := uuid.Parse(habitID)
	uid, _ := uuid.Parse(userID)
	return s.repo.Complete(ctx, hid, uid, date, notes, rating, completionTime)
}

func (s *Service) Toggle(ctx context.Context, habitID, userID, workspaceID string, date time.Time) (bool, *model.HabitCompletion, error) {
	_, err := s.Get(ctx, habitID, userID, workspaceID)
	if err != nil {
		return false, nil, err
	}
	hid, _ := uuid.Parse(habitID)
	uid, _ := uuid.Parse(userID)
	return s.repo.Toggle(ctx, hid, uid, date)
}

func (s *Service) GetStats(ctx context.Context, habitID, userID, workspaceID string) (*model.HabitStats, error) {
	_, err := s.Get(ctx, habitID, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	hid, _ := uuid.Parse(habitID)
	uid, _ := uuid.Parse(userID)
	return s.repo.GetStats(ctx, hid, uid)
}

func (s *Service) GetCompletions(ctx context.Context, habitID, userID, workspaceID string, start, end time.Time) ([]model.HabitCompletion, error) {
	_, err := s.Get(ctx, habitID, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	hid, _ := uuid.Parse(habitID)
	uid, _ := uuid.Parse(userID)
	return s.repo.GetCompletions(ctx, hid, uid, start, end)
}

func (s *Service) GetAllCompletions(ctx context.Context, userID, workspaceID string, start, end time.Time) ([]model.HabitCompletion, error) {
	if workspaceID == "" {
		return nil, ErrWorkspaceNeeded
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	return s.repo.GetAllCompletions(ctx, uid, wid, start, end)
}

func (s *Service) GetCalendar(ctx context.Context, userID, workspaceID string, start, end time.Time) (*model.CalendarResponse, error) {
	if workspaceID == "" {
		return nil, ErrWorkspaceNeeded
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	return s.repo.GetCalendar(ctx, uid, wid, start, end)
}
