package habits

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"backend/internal/model"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func normalizeDate(t time.Time) time.Time {
	utc := t.UTC()
	year, month, day := utc.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func (r *Repository) List(ctx context.Context, userID, workspaceID uuid.UUID) ([]model.Habit, error) {
	query := `
		SELECT 
			id, title, description, color, icon, 
			target_days, daily_goal, preferred_time, category,
			user_id, workspace_id, 
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
		var preferredTimePtr sql.NullString
		var categoryPtr sql.NullString

		err := rows.Scan(
			&habit.ID,
			&habit.Title,
			&habit.Description,
			&habit.Color,
			&habit.Icon,
			&habit.TargetDays,
			&habit.DailyGoal,
			&preferredTimePtr,
			&categoryPtr,
			&habit.UserID,
			&habit.WorkspaceID,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan habit: %w", err)
		}

		if preferredTimePtr.Valid {
			habit.PreferredTime = preferredTimePtr.String
		}
		if categoryPtr.Valid {
			habit.Category = categoryPtr.String
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
			target_days, daily_goal, preferred_time, category,
			user_id, workspace_id, 
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, title, description, color, icon, 
			target_days, daily_goal, preferred_time, category,
			user_id, workspace_id, 
			created_at, updated_at
	`

	now := time.Now().UTC()
	habitID := uuid.New()

	// Обработка пустых значений
	var categoryValue interface{}
	if dto.Category != "" {
		categoryValue = dto.Category
	} else {
		categoryValue = nil
	}

	var preferredTimeValue interface{}
	if dto.PreferredTime != "" && dto.PreferredTime != "any" {
		preferredTimeValue = dto.PreferredTime
	} else {
		preferredTimeValue = nil
	}

	var habit model.Habit
	var createdAt, updatedAt time.Time
	var preferredTimePtr sql.NullString
	var categoryPtr sql.NullString

	targetDays := dto.TargetDays
	if targetDays == 0 {
		targetDays = 7
	}
	dailyGoal := dto.DailyGoal
	if dailyGoal == 0 {
		dailyGoal = 1
	}
	color := dto.Color
	if color == "" {
		color = "#3B82F6"
	}

	err := r.db.QueryRowContext(ctx, query,
		habitID,
		dto.Title,
		dto.Description,
		color,
		dto.Icon,
		targetDays,
		dailyGoal,
		preferredTimeValue,
		categoryValue,
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
		&habit.DailyGoal,
		&preferredTimePtr,
		&categoryPtr,
		&habit.UserID,
		&habit.WorkspaceID,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		log.Printf("Error creating habit: %v, dto: %+v", err, dto)
		return nil, fmt.Errorf("failed to create habit: %w", err)
	}

	if preferredTimePtr.Valid {
		habit.PreferredTime = preferredTimePtr.String
	}
	if categoryPtr.Valid {
		habit.Category = categoryPtr.String
	}

	habit.CreatedAt = createdAt.Format(time.RFC3339)
	habit.UpdatedAt = updatedAt.Format(time.RFC3339)

	return &habit, nil
}

func (r *Repository) Get(ctx context.Context, id, userID uuid.UUID) (*model.Habit, error) {
	query := `
		SELECT 
			id, title, description, color, icon, 
			target_days, daily_goal, preferred_time, category,
			user_id, workspace_id, 
			created_at, updated_at
		FROM habits 
		WHERE id = $1 AND user_id = $2
	`

	var habit model.Habit
	var createdAt, updatedAt time.Time
	var preferredTimePtr sql.NullString
	var categoryPtr sql.NullString

	err := r.db.QueryRowContext(ctx, query, id, userID).Scan(
		&habit.ID,
		&habit.Title,
		&habit.Description,
		&habit.Color,
		&habit.Icon,
		&habit.TargetDays,
		&habit.DailyGoal,
		&preferredTimePtr,
		&categoryPtr,
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

	if preferredTimePtr.Valid {
		habit.PreferredTime = preferredTimePtr.String
	}
	if categoryPtr.Valid {
		habit.Category = categoryPtr.String
	}

	habit.CreatedAt = createdAt.Format(time.RFC3339)
	habit.UpdatedAt = updatedAt.Format(time.RFC3339)

	return &habit, nil
}

func (r *Repository) Update(ctx context.Context, id, userID uuid.UUID, dto model.UpdateHabitDto) (*model.Habit, error) {
	updates := []string{"updated_at = $1"}
	args := []interface{}{time.Now().UTC()}
	argIndex := 2

	if dto.Title != nil {
		updates = append(updates, fmt.Sprintf("title = $%d", argIndex))
		args = append(args, *dto.Title)
		argIndex++
	}
	if dto.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, *dto.Description)
		argIndex++
	}
	if dto.Color != nil {
		updates = append(updates, fmt.Sprintf("color = $%d", argIndex))
		args = append(args, *dto.Color)
		argIndex++
	}
	if dto.Icon != nil {
		updates = append(updates, fmt.Sprintf("icon = $%d", argIndex))
		args = append(args, *dto.Icon)
		argIndex++
	}
	if dto.TargetDays != nil && *dto.TargetDays > 0 {
		updates = append(updates, fmt.Sprintf("target_days = $%d", argIndex))
		args = append(args, *dto.TargetDays)
		argIndex++
	}
	if dto.DailyGoal != nil && *dto.DailyGoal > 0 {
		updates = append(updates, fmt.Sprintf("daily_goal = $%d", argIndex))
		args = append(args, *dto.DailyGoal)
		argIndex++
	}
	if dto.PreferredTime != nil {
		var preferredTimeValue interface{}
		if *dto.PreferredTime != "" && *dto.PreferredTime != "any" {
			preferredTimeValue = *dto.PreferredTime
		} else {
			preferredTimeValue = nil
		}
		updates = append(updates, fmt.Sprintf("preferred_time = $%d", argIndex))
		args = append(args, preferredTimeValue)
		argIndex++
	}

	if dto.Category != nil {
		var categoryValue interface{}
		if *dto.Category != "" {
			categoryValue = *dto.Category
		} else {
			categoryValue = nil // Пустая строка означает удаление категории
		}
		updates = append(updates, fmt.Sprintf("category = $%d", argIndex))
		args = append(args, categoryValue)
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
			target_days, daily_goal, preferred_time, category,
			user_id, workspace_id, 
			created_at, updated_at
	`, strings.Join(updates, ", "), argIndex, argIndex+1)

	args = append(args, id, userID)

	var habit model.Habit
	var createdAt, updatedAt time.Time
	var preferredTimePtr sql.NullString
	var categoryPtr sql.NullString

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&habit.ID,
		&habit.Title,
		&habit.Description,
		&habit.Color,
		&habit.Icon,
		&habit.TargetDays,
		&habit.DailyGoal,
		&preferredTimePtr,
		&categoryPtr,
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

	if preferredTimePtr.Valid {
		habit.PreferredTime = preferredTimePtr.String
	}
	if categoryPtr.Valid {
		habit.Category = categoryPtr.String
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

func (r *Repository) Complete(ctx context.Context, habitID, userID uuid.UUID, date time.Time, notes string, rating interface{}, completionTime *string) (*model.HabitCompletion, error) {
	query := `
		INSERT INTO habit_completions (
			id, habit_id, user_id, date, notes, rating, time, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, habit_id, user_id, date, notes, rating, time, created_at
	`

	completionID := uuid.New()
	now := time.Now().UTC()
	// Нормализуем дату до начала дня для корректного сравнения с полем DATE
	normalizedDate := normalizeDate(date)

	var completion model.HabitCompletion
	var completionDate, createdAt time.Time
	var timePtr sql.NullString
	var ratingPtr sql.NullInt64

	var timeValue interface{}
	if completionTime != nil && *completionTime != "" {
		timeValue = *completionTime
	} else {
		timeValue = nil
	}

	var ratingValue interface{}
	if rating == nil {
		ratingValue = nil
	} else if ratingInt, ok := rating.(int); ok && ratingInt == 0 {
		ratingValue = nil
	} else {
		ratingValue = rating
	}

	err := r.db.QueryRowContext(ctx, query,
		completionID,
		habitID,
		userID,
		normalizedDate,
		notes,
		ratingValue,
		timeValue,
		now,
	).Scan(
		&completion.ID,
		&completion.HabitID,
		&completion.UserID,
		&completionDate,
		&completion.Notes,
		&ratingPtr,
		&timePtr,
		&createdAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create completion: %w", err)
	}

	completion.Date = completionDate.Format("2006-01-02")
	completion.CreatedAt = createdAt.Format(time.RFC3339)
	if timePtr.Valid {
		completion.Time = timePtr.String
	}
	if ratingPtr.Valid {
		completion.Rating = int(ratingPtr.Int64)
	}

	return &completion, nil
}

func (r *Repository) Toggle(ctx context.Context, habitID, userID uuid.UUID, date time.Time) (bool, *model.HabitCompletion, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Нормализуем дату до начала дня для корректного сравнения с полем DATE
	normalizedDate := normalizeDate(date)

	// Проверяем существующее completion (берем первое найденное для этой даты)
	var existing model.HabitCompletion
	var existingDate, createdAt time.Time
	var timePtr sql.NullString
	var ratingPtr sql.NullInt64

	query := `
		SELECT id, habit_id, user_id, date, notes, rating, time, created_at
		FROM habit_completions 
		WHERE habit_id = $1 AND user_id = $2 AND date = $3
		LIMIT 1
	`

	err = tx.QueryRowContext(ctx, query, habitID, userID, normalizedDate).Scan(
		&existing.ID,
		&existing.HabitID,
		&existing.UserID,
		&existingDate,
		&existing.Notes,
		&ratingPtr,
		&timePtr,
		&createdAt,
	)

	if err == nil {
		existing.Date = existingDate.Format("2006-01-02")
		existing.CreatedAt = createdAt.Format(time.RFC3339)
		if timePtr.Valid {
			existing.Time = timePtr.String
		}
		if ratingPtr.Valid {
			existing.Rating = int(ratingPtr.Int64)
		}

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
	completion, err := r.Complete(ctx, habitID, userID, normalizedDate, "", 0, nil)
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

	// Вычисляем общее количество дней (используем UTC)
	today := time.Now().UTC()
	createdAtUTC := createdAt.UTC()
	totalDays := int(today.Sub(createdAtUTC).Hours()/24) + 1
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

	// Получаем все даты выполнений, отсортированные по убыванию
	completionsQuery := `
		SELECT DISTINCT date 
		FROM habit_completions 
		WHERE habit_id = $1 AND user_id = $2 
		ORDER BY date DESC
	`
	rows, err := r.db.QueryContext(ctx, completionsQuery, habitID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query completion dates: %w", err)
	}
	defer rows.Close()

	var completionDates []time.Time
	for rows.Next() {
		var date time.Time
		if err := rows.Scan(&date); err != nil {
			return nil, fmt.Errorf("failed to scan date: %w", err)
		}
		completionDates = append(completionDates, date.UTC())
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	// Вычисляем серии (streaks)
	var currentStreak, longestStreak int

	if len(completionDates) > 0 {
		// Нормализуем сегодняшнюю дату до начала дня в UTC
		todayNormalized := normalizeDate(today)

		// Создаем map для быстрого поиска дат выполнений
		completionMap := make(map[string]bool)
		for _, completionDate := range completionDates {
			normalized := normalizeDate(completionDate)
			dateKey := normalized.Format("2006-01-02")
			completionMap[dateKey] = true
		}

		// Вычисляем currentStreak (от сегодня назад)
		currentStreak = 0
		checkDate := todayNormalized
		for {
			dateKey := checkDate.Format("2006-01-02")
			if completionMap[dateKey] {
				currentStreak++
				checkDate = checkDate.AddDate(0, 0, -1) // Переходим к предыдущему дню
			} else {
				// Если сегодня нет выполнения, streak = 0
				// Если пропущен день в середине, streak прерывается
				break
			}
		}

		// Вычисляем longestStreak (самая длинная последовательность)
		// Сортируем даты по возрастанию
		sortedDates := make([]time.Time, len(completionDates))
		copy(sortedDates, completionDates)
		// Сортируем вручную (простая сортировка пузырьком для небольшого количества дат)
		for i := 0; i < len(sortedDates)-1; i++ {
			for j := 0; j < len(sortedDates)-i-1; j++ {
				date1 := normalizeDate(sortedDates[j])
				date2 := normalizeDate(sortedDates[j+1])
				if date1.After(date2) {
					sortedDates[j], sortedDates[j+1] = sortedDates[j+1], sortedDates[j]
				}
			}
		}

		longestStreak = 0
		currentSequence := 0
		var prevDate time.Time

		for _, date := range sortedDates {
			normalized := normalizeDate(date)
			if prevDate.IsZero() {
				currentSequence = 1
				prevDate = normalized
			} else {
				// Проверяем, что даты идут подряд
				expectedDate := prevDate.AddDate(0, 0, 1)
				if normalized.Equal(expectedDate) {
					currentSequence++
				} else {
					// Последовательность прервалась
					if currentSequence > longestStreak {
						longestStreak = currentSequence
					}
					currentSequence = 1
				}
				prevDate = normalized
			}
		}
		// Проверяем последнюю последовательность
		if currentSequence > longestStreak {
			longestStreak = currentSequence
		}
	}

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
		SELECT id, habit_id, user_id, date, notes, rating, time, created_at
		FROM habit_completions 
		WHERE habit_id = $1 AND user_id = $2 AND date BETWEEN $3 AND $4
		ORDER BY date DESC, time DESC
	`

	// Нормализуем даты до начала дня для корректного сравнения с полем DATE
	normalizedStart := normalizeDate(startDate)
	normalizedEnd := normalizeDate(endDate)

	rows, err := r.db.QueryContext(ctx, query, habitID, userID, normalizedStart, normalizedEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to query completions: %w", err)
	}
	defer rows.Close()

	var completions []model.HabitCompletion
	for rows.Next() {
		var completion model.HabitCompletion
		var completionDate, createdAt time.Time
		var timePtr sql.NullString
		var ratingPtr sql.NullInt64

		err := rows.Scan(
			&completion.ID,
			&completion.HabitID,
			&completion.UserID,
			&completionDate,
			&completion.Notes,
			&ratingPtr,
			&timePtr,
			&createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan completion: %w", err)
		}

		completion.Date = completionDate.Format("2006-01-02")
		completion.CreatedAt = createdAt.Format(time.RFC3339)
		if timePtr.Valid {
			completion.Time = timePtr.String
		}
		if ratingPtr.Valid {
			completion.Rating = int(ratingPtr.Int64)
		}
		completions = append(completions, completion)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return completions, nil
}

func (r *Repository) GetAllCompletions(ctx context.Context, userID, workspaceID uuid.UUID, startDate, endDate time.Time) ([]model.HabitCompletion, error) {
	query := `
		SELECT hc.id, hc.habit_id, hc.user_id, hc.date, hc.notes, hc.rating, hc.time, hc.created_at
		FROM habit_completions hc
		INNER JOIN habits h ON hc.habit_id = h.id
		WHERE hc.user_id = $1 AND h.workspace_id = $2 AND hc.date BETWEEN $3 AND $4
		ORDER BY hc.date DESC, hc.time DESC
	`

	// Нормализуем даты до начала дня для корректного сравнения с полем DATE
	normalizedStart := normalizeDate(startDate)
	normalizedEnd := normalizeDate(endDate)

	rows, err := r.db.QueryContext(ctx, query, userID, workspaceID, normalizedStart, normalizedEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to query completions: %w", err)
	}
	defer rows.Close()

	var completions []model.HabitCompletion
	for rows.Next() {
		var completion model.HabitCompletion
		var completionDate, createdAt time.Time
		var timePtr sql.NullString
		var ratingPtr sql.NullInt64

		err := rows.Scan(
			&completion.ID,
			&completion.HabitID,
			&completion.UserID,
			&completionDate,
			&completion.Notes,
			&ratingPtr,
			&timePtr,
			&createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan completion: %w", err)
		}

		completion.Date = completionDate.Format("2006-01-02")
		completion.CreatedAt = createdAt.Format(time.RFC3339)
		if timePtr.Valid {
			completion.Time = timePtr.String
		}
		if ratingPtr.Valid {
			completion.Rating = int(ratingPtr.Int64)
		}
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
	// Используем pq.Array для корректной передачи массива UUID в PostgreSQL
	// Нормализуем даты до начала дня для корректного сравнения с полем DATE
	normalizedStart := normalizeDate(startDate)
	normalizedEnd := normalizeDate(endDate)

	completionsQuery := `
		SELECT habit_id, date
		FROM habit_completions 
		WHERE user_id = $1 AND habit_id = ANY($2::uuid[]) AND date BETWEEN $3 AND $4
	`

	rows, err = r.db.QueryContext(ctx, completionsQuery, userID, pq.Array(habitIDs), normalizedStart, normalizedEnd)
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
	current := normalizedStart
	normalizedEndDate := normalizedEnd
	for !current.After(normalizedEndDate) {
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
