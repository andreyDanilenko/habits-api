package habits

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"backend/internal/model"

	"github.com/google/uuid"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(ctx context.Context, userID, workspaceID uuid.UUID) ([]model.Habit, error) {
	query := `
		SELECT 
			id, title, description, color, icon, 
			target_days, user_id, workspace_id, 
			created_at, updated_at
		FROM habits 
		WHERE user_id = $1 AND workspace_id = $2
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query habits: %w", err)
	}
	defer rows.Close()

	var habits []model.Habit
	for rows.Next() {
		var habit model.Habit
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&habit.ID,
			&habit.Title,
			&habit.Description,
			&habit.Color,
			&habit.Icon,
			&habit.TargetDays,
			&habit.UserID,
			&habit.WorkspaceID,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan habit: %w", err)
		}

		habit.CreatedAt = createdAt.Format(time.RFC3339)
		habit.UpdatedAt = updatedAt.Format(time.RFC3339)
		habits = append(habits, habit)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return habits, nil
}

func (r *Repository) Create(ctx context.Context, dto model.CreateHabitDto, userID, workspaceID uuid.UUID) (*model.Habit, error) {
	query := `
		INSERT INTO habits (
			id, title, description, color, icon, 
			target_days, user_id, workspace_id, 
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, title, description, color, icon, 
			target_days, user_id, workspace_id, 
			created_at, updated_at
	`

	now := time.Now()
	habitID := uuid.New()

	var habit model.Habit
	var createdAt, updatedAt time.Time

	err := r.db.QueryRowContext(ctx, query,
		habitID,
		dto.Title,
		dto.Description,
		dto.Color,
		dto.Icon,
		dto.TargetDays,
		userID,
		workspaceID,
		now,
		now,
	).Scan(
		&habit.ID,
		&habit.Title,
		&habit.Description,
		&habit.Color,
		&habit.Icon,
		&habit.TargetDays,
		&habit.UserID,
		&habit.WorkspaceID,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create habit: %w", err)
	}

	habit.CreatedAt = createdAt.Format(time.RFC3339)
	habit.UpdatedAt = updatedAt.Format(time.RFC3339)

	return &habit, nil
}

func (r *Repository) Get(ctx context.Context, id, userID uuid.UUID) (*model.Habit, error) {
	query := `
		SELECT 
			id, title, description, color, icon, 
			target_days, user_id, workspace_id, 
			created_at, updated_at
		FROM habits 
		WHERE id = $1 AND user_id = $2
	`

	var habit model.Habit
	var createdAt, updatedAt time.Time

	err := r.db.QueryRowContext(ctx, query, id, userID).Scan(
		&habit.ID,
		&habit.Title,
		&habit.Description,
		&habit.Color,
		&habit.Icon,
		&habit.TargetDays,
		&habit.UserID,
		&habit.WorkspaceID,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get habit: %w", err)
	}

	habit.CreatedAt = createdAt.Format(time.RFC3339)
	habit.UpdatedAt = updatedAt.Format(time.RFC3339)

	return &habit, nil
}

func (r *Repository) Update(ctx context.Context, id, userID uuid.UUID, dto model.UpdateHabitDto) (*model.Habit, error) {
	updates := []string{"updated_at = $1"}
	args := []interface{}{time.Now()}
	argIndex := 2

	if dto.Title != "" {
		updates = append(updates, fmt.Sprintf("title = $%d", argIndex))
		args = append(args, dto.Title)
		argIndex++
	}
	if dto.Description != "" {
		updates = append(updates, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, dto.Description)
		argIndex++
	}
	if dto.Color != "" {
		updates = append(updates, fmt.Sprintf("color = $%d", argIndex))
		args = append(args, dto.Color)
		argIndex++
	}
	if dto.Icon != "" {
		updates = append(updates, fmt.Sprintf("icon = $%d", argIndex))
		args = append(args, dto.Icon)
		argIndex++
	}
	if dto.TargetDays > 0 {
		updates = append(updates, fmt.Sprintf("target_days = $%d", argIndex))
		args = append(args, dto.TargetDays)
		argIndex++
	}

	if len(updates) == 1 { // только updated_at изменился
		return r.Get(ctx, id, userID)
	}

	query := fmt.Sprintf(`
		UPDATE habits 
		SET %s 
		WHERE id = $%d AND user_id = $%d
		RETURNING id, title, description, color, icon, 
			target_days, user_id, workspace_id, 
			created_at, updated_at
	`, strings.Join(updates, ", "), argIndex, argIndex+1)

	args = append(args, id, userID)

	var habit model.Habit
	var createdAt, updatedAt time.Time

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&habit.ID,
		&habit.Title,
		&habit.Description,
		&habit.Color,
		&habit.Icon,
		&habit.TargetDays,
		&habit.UserID,
		&habit.WorkspaceID,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to update habit: %w", err)
	}

	habit.CreatedAt = createdAt.Format(time.RFC3339)
	habit.UpdatedAt = updatedAt.Format(time.RFC3339)

	return &habit, nil
}

func (r *Repository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Удаляем связанные completions
	_, err = tx.ExecContext(ctx,
		"DELETE FROM habit_completions WHERE habit_id = $1 AND user_id = $2",
		id, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete habit completions: %w", err)
	}

	// Удаляем привычку
	result, err := tx.ExecContext(ctx,
		"DELETE FROM habits WHERE id = $1 AND user_id = $2",
		id, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete habit: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return tx.Commit()
}

func (r *Repository) Complete(ctx context.Context, habitID, userID uuid.UUID, date time.Time, notes string, rating int) (*model.HabitCompletion, error) {
	query := `
		INSERT INTO habit_completions (
			id, habit_id, user_id, date, notes, rating, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, habit_id, user_id, date, notes, rating, created_at
	`

	completionID := uuid.New()
	now := time.Now()

	var completion model.HabitCompletion
	var completionDate, createdAt time.Time

	err := r.db.QueryRowContext(ctx, query,
		completionID,
		habitID,
		userID,
		date,
		notes,
		rating,
		now,
	).Scan(
		&completion.ID,
		&completion.HabitID,
		&completion.UserID,
		&completionDate,
		&completion.Notes,
		&completion.Rating,
		&createdAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create completion: %w", err)
	}

	completion.Date = completionDate.Format("2006-01-02")
	completion.CreatedAt = createdAt.Format(time.RFC3339)

	return &completion, nil
}

func (r *Repository) Toggle(ctx context.Context, habitID, userID uuid.UUID, date time.Time) (bool, *model.HabitCompletion, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Проверяем существующее completion
	var existing model.HabitCompletion
	var existingDate, createdAt time.Time

	query := `
		SELECT id, habit_id, user_id, date, notes, rating, created_at
		FROM habit_completions 
		WHERE habit_id = $1 AND user_id = $2 AND date = $3
	`

	err = tx.QueryRowContext(ctx, query, habitID, userID, date).Scan(
		&existing.ID,
		&existing.HabitID,
		&existing.UserID,
		&existingDate,
		&existing.Notes,
		&existing.Rating,
		&createdAt,
	)

	if err == nil {
		// Удаляем если существует
		existing.Date = existingDate.Format("2006-01-02")
		existing.CreatedAt = createdAt.Format(time.RFC3339)

		_, err = tx.ExecContext(ctx,
			"DELETE FROM habit_completions WHERE id = $1",
			existing.ID,
		)
		if err != nil {
			return false, nil, fmt.Errorf("failed to delete completion: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return false, nil, fmt.Errorf("failed to commit transaction: %w", err)
		}

		return false, &existing, nil
	}

	if err != sql.ErrNoRows {
		return false, nil, fmt.Errorf("failed to check existing completion: %w", err)
	}

	// Создаем новое если не существует
	completion, err := r.Complete(ctx, habitID, userID, date, "", 0)
	if err != nil {
		return false, nil, err
	}

	if err := tx.Commit(); err != nil {
		return false, nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return true, completion, nil
}

func (r *Repository) GetStats(ctx context.Context, habitID, userID uuid.UUID) (*model.HabitStats, error) {
	// Получаем информацию о привычке
	var createdAt time.Time
	err := r.db.QueryRowContext(ctx,
		"SELECT created_at FROM habits WHERE id = $1 AND user_id = $2",
		habitID, userID,
	).Scan(&createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get habit: %w", err)
	}

	// Вычисляем общее количество дней
	today := time.Now()
	totalDays := int(today.Sub(createdAt).Hours()/24) + 1
	if totalDays < 1 {
		totalDays = 1
	}

	// Получаем количество выполненных дней
	var completedDays int
	err = r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM habit_completions WHERE habit_id = $1 AND user_id = $2",
		habitID, userID,
	).Scan(&completedDays)
	if err != nil {
		return nil, fmt.Errorf("failed to count completions: %w", err)
	}

	// Вычисляем серии (streaks)
	var currentStreak, longestStreak int
	// TODO: Реализовать более эффективный расчет серий

	completionRate := 0.0
	if totalDays > 0 {
		completionRate = float64(completedDays) / float64(totalDays)
	}

	stats := &model.HabitStats{
		HabitID:        habitID.String(),
		CompletedDays:  completedDays,
		TotalDays:      totalDays,
		CompletionRate: completionRate,
		CurrentStreak:  currentStreak,
		LongestStreak:  longestStreak,
	}

	return stats, nil
}

func (r *Repository) GetCompletions(ctx context.Context, habitID, userID uuid.UUID, startDate, endDate time.Time) ([]model.HabitCompletion, error) {
	query := `
		SELECT id, habit_id, user_id, date, notes, rating, created_at
		FROM habit_completions 
		WHERE habit_id = $1 AND user_id = $2 AND date BETWEEN $3 AND $4
		ORDER BY date DESC
	`

	rows, err := r.db.QueryContext(ctx, query, habitID, userID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query completions: %w", err)
	}
	defer rows.Close()

	var completions []model.HabitCompletion
	for rows.Next() {
		var completion model.HabitCompletion
		var completionDate, createdAt time.Time

		err := rows.Scan(
			&completion.ID,
			&completion.HabitID,
			&completion.UserID,
			&completionDate,
			&completion.Notes,
			&completion.Rating,
			&createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan completion: %w", err)
		}

		completion.Date = completionDate.Format("2006-01-02")
		completion.CreatedAt = createdAt.Format(time.RFC3339)
		completions = append(completions, completion)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return completions, nil
}

func (r *Repository) GetCalendar(ctx context.Context, userID, workspaceID uuid.UUID, startDate, endDate time.Time) (*model.CalendarResponse, error) {
	// Получаем все привычки пользователя в workspace
	habitsQuery := `
		SELECT id, title, color
		FROM habits 
		WHERE user_id = $1 AND workspace_id = $2
	`

	rows, err := r.db.QueryContext(ctx, habitsQuery, userID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query habits: %w", err)
	}
	defer rows.Close()

	habits := make([]struct {
		ID    uuid.UUID
		Title string
		Color string
	}, 0)

	habitIDs := make([]uuid.UUID, 0)
	for rows.Next() {
		var habit struct {
			ID    uuid.UUID
			Title string
			Color string
		}
		err := rows.Scan(&habit.ID, &habit.Title, &habit.Color)
		if err != nil {
			return nil, fmt.Errorf("failed to scan habit: %w", err)
		}
		habits = append(habits, habit)
		habitIDs = append(habitIDs, habit.ID)
	}

	if len(habitIDs) == 0 {
		return &model.CalendarResponse{Days: []model.CalendarDay{}}, nil
	}

	// Получаем completion за период
	completionsQuery := `
		SELECT habit_id, date
		FROM habit_completions 
		WHERE user_id = $1 AND habit_id = ANY($2) AND date BETWEEN $3 AND $4
	`

	rows, err = r.db.QueryContext(ctx, completionsQuery, userID, habitIDs, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query completions: %w", err)
	}
	defer rows.Close()

	// Создаем мапу для быстрого поиска completion по дате и habit_id
	completionMap := make(map[string]map[uuid.UUID]bool)
	for rows.Next() {
		var habitID uuid.UUID
		var date time.Time
		err := rows.Scan(&habitID, &date)
		if err != nil {
			return nil, fmt.Errorf("failed to scan completion: %w", err)
		}

		dateKey := date.Format("2006-01-02")
		if completionMap[dateKey] == nil {
			completionMap[dateKey] = make(map[uuid.UUID]bool)
		}
		completionMap[dateKey][habitID] = true
	}

	// Генерируем дни календаря
	days := make([]model.CalendarDay, 0)
	current := startDate
	for !current.After(endDate) {
		dateKey := current.Format("2006-01-02")

		dayHabits := make([]struct {
			ID        string `json:"id"`
			Title     string `json:"title"`
			Completed bool   `json:"completed"`
			Color     string `json:"color"`
		}, 0)

		for _, habit := range habits {
			completed := false
			if compMap, exists := completionMap[dateKey]; exists {
				completed = compMap[habit.ID]
			}

			dayHabits = append(dayHabits, struct {
				ID        string `json:"id"`
				Title     string `json:"title"`
				Completed bool   `json:"completed"`
				Color     string `json:"color"`
			}{
				ID:        habit.ID.String(),
				Title:     habit.Title,
				Completed: completed,
				Color:     habit.Color,
			})
		}

		days = append(days, model.CalendarDay{
			Date:   dateKey,
			Habits: dayHabits,
		})

		current = current.AddDate(0, 0, 1)
	}

	return &model.CalendarResponse{Days: days}, nil
}
