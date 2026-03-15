package habits

import (
	"context"
	"errors"
	"time"

	"backend/internal/model"
	habitsRepo "backend/internal/repository/habits"
	workspaceRepo "backend/internal/repository/workspace"
	"backend/pkg/realtime"

	"github.com/google/uuid"
)

var (
	ErrHabitNotFound   = errors.New("habit not found")
	ErrWorkspaceNeeded = errors.New("workspace not selected")
	ErrCannotDelete    = errors.New("cannot delete another user's habit")
)

type Service struct {
	repo      *habitsRepo.Repository
	wsRepo    *workspaceRepo.Repository
	publisher realtime.Publisher
}

func NewService(repo *habitsRepo.Repository, wsRepo *workspaceRepo.Repository, publisher realtime.Publisher) *Service {
	if publisher == nil {
		publisher = realtime.NoopPublisher{}
	}
	return &Service{repo: repo, wsRepo: wsRepo, publisher: publisher}
}

func (s *Service) emitHabitEvent(ctx context.Context, workspaceID, eventType string, payload map[string]interface{}) {
	ch := realtime.WorkspaceChannel(workspaceID)
	event := realtime.Event{
		EventType: eventType,
		Target:    realtime.Target{Type: "workspace", ID: workspaceID},
		Payload:   payload,
		Timestamp: time.Now().UnixMilli(),
	}
	_ = s.publisher.Publish(ctx, ch, event)
}

func (s *Service) List(ctx context.Context, workspaceID string, targetDate *time.Time) ([]model.Habit, error) {
	if workspaceID == "" {
		return nil, ErrWorkspaceNeeded
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	return s.repo.List(ctx, wid, targetDate)
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
	habit, err := s.repo.Create(ctx, dto, uid, wid)
	if err != nil {
		return nil, err
	}
	hid, _ := uuid.Parse(habit.ID)
	_ = s.repo.CreateActivity(ctx, uid, wid, hid, habitsRepo.ActivityTypeHabitCreated,
		"Создана привычка \""+habit.Title+"\"", "➕")
	s.emitHabitEvent(ctx, workspaceID, "habit.created", map[string]interface{}{
		"habit": habit,
		"userId": userID,
	})
	return habit, nil
}

// Get возвращает привычку по id и workspace. Любой участник workspace может просматривать любую привычку.
func (s *Service) Get(ctx context.Context, habitID, userID, workspaceID string) (*model.Habit, error) {
	hid, err := uuid.Parse(habitID)
	if err != nil {
		return nil, err
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, err
	}
	h, err := s.repo.GetByIDAndWorkspace(ctx, hid, wid)
	if err != nil || h == nil {
		return nil, ErrHabitNotFound
	}
	return h, nil
}

func (s *Service) Update(ctx context.Context, habitID string, dto model.UpdateHabitDto, userID, workspaceID string) (*model.Habit, error) {
	habit, err := s.Get(ctx, habitID, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	uid, _ := uuid.Parse(userID)
	if habit.UserID != uid.String() {
		return nil, ErrHabitNotFound // только владелец может редактировать
	}
	hid, _ := uuid.Parse(habitID)
	wid, _ := uuid.Parse(workspaceID)
	updated, err := s.repo.Update(ctx, hid, uid, dto)
	if err != nil {
		return nil, err
	}
	_ = s.repo.CreateActivity(ctx, uid, wid, hid, habitsRepo.ActivityTypeHabitUpdated,
		"Обновлена привычка \""+updated.Title+"\"", "✏️")
	s.emitHabitEvent(ctx, workspaceID, "habit.updated", map[string]interface{}{
		"habit": updated,
	})
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, habitID, userID, workspaceID string) error {
	habit, err := s.Get(ctx, habitID, userID, workspaceID)
	if err != nil {
		return err
	}
	uid, _ := uuid.Parse(userID)
	wid, _ := uuid.Parse(workspaceID)
	habitOwnerID, _ := uuid.Parse(habit.UserID)

	// Удалять может только владелец привычки или владелец workspace
	if habit.UserID != uid.String() {
		isOwner, err := s.wsRepo.IsOwner(ctx, wid, uid)
		if err != nil || !isOwner {
			return ErrCannotDelete
		}
	}

	hid, _ := uuid.Parse(habitID)
	if err := s.repo.Delete(ctx, hid, habitOwnerID); err != nil {
		return err
	}
	_ = s.repo.CreateActivity(ctx, habitOwnerID, wid, hid, habitsRepo.ActivityTypeHabitDeleted,
		"Удалена привычка \""+habit.Title+"\"", "🗑️")
	s.emitHabitEvent(ctx, workspaceID, "habit.deleted", map[string]interface{}{
		"habitId": habitID,
		"title":   habit.Title,
	})
	return nil
}

func (s *Service) Complete(ctx context.Context, habitID, userID, workspaceID string, date time.Time, notes string, rating interface{}, completionTime *string) (*model.HabitCompletion, error) {
	habit, err := s.Get(ctx, habitID, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	hid, _ := uuid.Parse(habitID)
	uid, _ := uuid.Parse(userID)
	wid, _ := uuid.Parse(workspaceID)
	// Завершать может любой участник workspace (в т.ч. чужую привычку)
	completion, err := s.repo.Complete(ctx, hid, uid, date, notes, rating, completionTime)
	if err != nil {
		return nil, err
	}
	_ = s.repo.CreateCompletionActivity(ctx, uid, wid, hid,
		"Завершена привычка \""+habit.Title+"\"", "✅")
	s.emitHabitEvent(ctx, workspaceID, "habit.completed", map[string]interface{}{
		"habitId":    habitID,
		"completion": completion,
		"userId":     userID,
	})
	return completion, nil
}

func (s *Service) Toggle(ctx context.Context, habitID, userID, workspaceID string, date time.Time) (bool, *model.HabitCompletion, error) {
	habit, err := s.Get(ctx, habitID, userID, workspaceID)
	if err != nil {
		return false, nil, err
	}
	hid, _ := uuid.Parse(habitID)
	uid, _ := uuid.Parse(userID)
	wid, _ := uuid.Parse(workspaceID)
	added, completion, err := s.repo.Toggle(ctx, hid, uid, date)
	if err != nil {
		return false, nil, err
	}
	if added {
		_ = s.repo.CreateCompletionActivity(ctx, uid, wid, hid,
			"Завершена привычка \""+habit.Title+"\"", "✅")
		s.emitHabitEvent(ctx, workspaceID, "habit.completed", map[string]interface{}{
			"habitId":    habitID,
			"completion": completion,
			"userId":     userID,
		})
	}
	return added, completion, nil
}

func (s *Service) GetStats(ctx context.Context, habitID, userID, workspaceID string) (*model.HabitStats, error) {
	habit, err := s.Get(ctx, habitID, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	hid, _ := uuid.Parse(habitID)
	habitOwnerID, _ := uuid.Parse(habit.UserID)
	return s.repo.GetStats(ctx, hid, habitOwnerID)
}

func (s *Service) GetCompletions(ctx context.Context, habitID, userID, workspaceID string, start, end time.Time) ([]model.HabitCompletion, error) {
	habit, err := s.Get(ctx, habitID, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	hid, _ := uuid.Parse(habitID)
	habitOwnerID, _ := uuid.Parse(habit.UserID)
	return s.repo.GetCompletions(ctx, hid, habitOwnerID, start, end)
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

func (s *Service) ListActivities(ctx context.Context, workspaceID string, limit, offset int) ([]model.Activity, int, error) {
	if workspaceID == "" {
		return nil, 0, ErrWorkspaceNeeded
	}
	wid, err := uuid.Parse(workspaceID)
	if err != nil {
		return nil, 0, err
	}
	return s.repo.ListActivities(ctx, wid, limit, offset)
}
