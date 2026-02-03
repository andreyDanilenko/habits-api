package habits

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"regexp"
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

// convertPreferredTimeToTime конвертирует строковое значение preferredTime в формат времени для PostgreSQL
// "morning" -> "08:00:00", "afternoon" -> "14:00:00", "evening" -> "20:00:00"
// Если значение уже в формате времени (HH:MM:SS), возвращает его как есть
func convertPreferredTimeToTime(preferredTime string) string {
	switch preferredTime {
	case "morning":
		return "08:00:00"
	case "afternoon":
		return "14:00:00"
	case "evening":
		return "20:00:00"
	default:
		if matched, _ := regexp.MatchString(`^\d{1,2}:\d{2}(:\d{2})?$`, preferredTime); matched {
			return preferredTime
		}
		return "08:00:00"
	}
}

// convertTimeToPreferredTime конвертирует время из БД обратно в строковое значение для фронтенда
// "08:00:00" -> "morning", "14:00:00" -> "afternoon", "20:00:00" -> "evening"
// Если время не совпадает с известными значениями, возвращает время как есть
func convertTimeToPreferredTime(timeStr string) string {
	switch timeStr {
	case "08:00:00", "08:00":
		return "morning"
	case "14:00:00", "14:00":
		return "afternoon"
	case "20:00:00", "20:00":
		return "evening"
	default:
		return timeStr
	}
}

func (r *Repository) List(ctx context.Context, userID, workspaceID uuid.UUID, targetDate *time.Time) ([]model.Habit, error) {
	// Если указана дата, используем GetHabitsForDate
	if targetDate != nil {
		return r.GetHabitsForDate(ctx, userID, workspaceID, *targetDate)
	}

	// Иначе возвращаем все привычки
	query := `
		SELECT 
			id, title, description, color, icon, 
			target_days, daily_goal, preferred_time, category,
			schedule_type, recurring_days, one_time_date, is_active,
			user_id, workspace_id, 
			created_at, updated_at
		FROM habits 
		WHERE user_id = $1 AND workspace_id = $2
		ORDER BY preferred_time NULLS LAST, created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query habits: %w", err)
	}
	defer rows.Close()

	return r.scanHabits(rows)
}

// GetHabitsForDate возвращает привычки, активные на указанную дату
// Важно: привычка не должна влиять на прошлые дни - она активна только с даты создания
func (r *Repository) GetHabitsForDate(ctx context.Context, userID, workspaceID uuid.UUID, targetDate time.Time) ([]model.Habit, error) {
	normalizedDate := normalizeDate(targetDate)
	// Используем EXTRACT(DOW FROM date) в SQL для получения дня недели
	// PostgreSQL: 0=воскресенье, 1=понедельник, ..., 6=суббота

	query := `
		SELECT 
			id, title, description, color, icon, 
			target_days, daily_goal, preferred_time, category,
			schedule_type, recurring_days, one_time_date, is_active,
			user_id, workspace_id, 
			created_at, updated_at
		FROM habits 
		WHERE user_id = $1 
			AND workspace_id = $2
			AND is_active = true
			-- Важно: привычка не должна влиять на прошлые дни - только с даты создания
			AND DATE(created_at) <= $3::date
			AND (
				-- Регулярные привычки: проверяем день недели в массиве recurring_days
				-- Используем EXTRACT(DOW FROM date) для получения дня недели (0=воскресенье, 6=суббота)
				(schedule_type = 'recurring' AND EXTRACT(DOW FROM $3::date) = ANY(recurring_days))
				OR
				-- Разовые привычки: проверяем совпадение даты
				(schedule_type = 'one_time' AND one_time_date = $3::date)
			)
		ORDER BY preferred_time NULLS LAST, created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID, workspaceID, normalizedDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query habits for date: %w", err)
	}
	defer rows.Close()

	return r.scanHabits(rows)
}

// scanHabits - вспомогательная функция для сканирования привычек из rows
func (r *Repository) scanHabits(rows *sql.Rows) ([]model.Habit, error) {
	var habits []model.Habit
	for rows.Next() {
		var habit model.Habit
		var createdAt, updatedAt time.Time
		var preferredTimePtr sql.NullString
		var categoryPtr sql.NullString
		var oneTimeDatePtr sql.NullTime
		var recurringDaysArray pq.Int32Array

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
			&habit.ScheduleType,
			&recurringDaysArray,
			&oneTimeDatePtr,
			&habit.IsActive,
			&habit.UserID,
			&habit.WorkspaceID,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan habit: %w", err)
		}

		if preferredTimePtr.Valid {
			// Конвертируем время из БД обратно в строковое значение для фронтенда
			habit.PreferredTime = convertTimeToPreferredTime(preferredTimePtr.String)
		}
		if categoryPtr.Valid {
			habit.Category = categoryPtr.String
		}
		if oneTimeDatePtr.Valid {
			habit.OneTimeDate = oneTimeDatePtr.Time.Format("2006-01-02")
		}

		// Конвертируем pq.Int32Array в []int
		if recurringDaysArray != nil {
			habit.RecurringDays = make([]int, len(recurringDaysArray))
			for i, v := range recurringDaysArray {
				habit.RecurringDays[i] = int(v)
			}
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
			schedule_type, recurring_days, one_time_date, is_active,
			user_id, workspace_id, 
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id, title, description, color, icon, 
			target_days, daily_goal, preferred_time, category,
			schedule_type, recurring_days, one_time_date, is_active,
			user_id, workspace_id, 
			created_at, updated_at
	`

	now := time.Now().UTC()
	habitID := uuid.New()

	var categoryValue interface{}
	if dto.Category != "" {
		categoryValue = dto.Category
	} else {
		categoryValue = nil
	}

	var preferredTimeValue interface{}
	if dto.PreferredTime != "" && dto.PreferredTime != "any" {
		preferredTimeValue = convertPreferredTimeToTime(dto.PreferredTime)
	} else {
		preferredTimeValue = nil
	}

	scheduleType := dto.ScheduleType
	if scheduleType == "" {
		scheduleType = "recurring"
	}

	var recurringDaysValue interface{}
	var oneTimeDateValue interface{}

	targetDays := dto.TargetDays

	if scheduleType == "recurring" {
		// PostgreSQL EXTRACT(DOW): 0=воскресенье, 1=понедельник, ..., 6=суббота
		var recurringDays []int
		if len(dto.RecurringDays) == 0 {
			// По умолчанию все дни недели: [0,1,2,3,4,5,6] (Вс, Пн, Вт, Ср, Чт, Пт, Сб)
			recurringDays = []int{0, 1, 2, 3, 4, 5, 6}
		} else {
			recurringDays = dto.RecurringDays
		}
		recurringDaysValue = pq.Array(recurringDays)
		oneTimeDateValue = nil

		// Автоматически устанавливаем target_days на количество выбранных дней
		if targetDays == 0 {
			targetDays = len(recurringDays)
		}
	} else if scheduleType == "one_time" {
		// Для разовых привычек нужна дата
		if dto.OneTimeDate == "" {
			return nil, fmt.Errorf("one_time_date is required for one_time schedule type")
		}
		oneTimeDate, err := time.Parse("2006-01-02", dto.OneTimeDate)
		if err != nil {
			return nil, fmt.Errorf("invalid one_time_date format: %w", err)
		}
		oneTimeDateValue = normalizeDate(oneTimeDate)
		recurringDaysValue = nil

		// Для разовых привычек target_days = 1
		if targetDays == 0 {
			targetDays = 1
		}
	}

	if targetDays == 0 {
		targetDays = 7
	}

	isActive := true
	if dto.IsActive != nil {
		isActive = *dto.IsActive
	}

	var habit model.Habit
	var createdAt, updatedAt time.Time
	var preferredTimePtr sql.NullString
	var categoryPtr sql.NullString
	var oneTimeDatePtr sql.NullTime
	var recurringDaysArray pq.Int32Array
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
		scheduleType,
		recurringDaysValue,
		oneTimeDateValue,
		isActive,
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
		&habit.ScheduleType,
		&recurringDaysArray,
		&oneTimeDatePtr,
		&habit.IsActive,
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
	if oneTimeDatePtr.Valid {
		habit.OneTimeDate = oneTimeDatePtr.Time.Format("2006-01-02")
	}

	if recurringDaysArray != nil {
		habit.RecurringDays = make([]int, len(recurringDaysArray))
		for i, v := range recurringDaysArray {
			habit.RecurringDays[i] = int(v)
		}
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
			schedule_type, recurring_days, one_time_date, is_active,
			user_id, workspace_id, 
			created_at, updated_at
		FROM habits 
		WHERE id = $1 AND user_id = $2
	`

	var habit model.Habit
	var createdAt, updatedAt time.Time
	var preferredTimePtr sql.NullString
	var categoryPtr sql.NullString
	var oneTimeDatePtr sql.NullTime
	var recurringDaysArray pq.Int32Array

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
		&habit.ScheduleType,
		&recurringDaysArray,
		&oneTimeDatePtr,
		&habit.IsActive,
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
	if oneTimeDatePtr.Valid {
		habit.OneTimeDate = oneTimeDatePtr.Time.Format("2006-01-02")
	}

	if recurringDaysArray != nil {
		habit.RecurringDays = make([]int, len(recurringDaysArray))
		for i, v := range recurringDaysArray {
			habit.RecurringDays[i] = int(v)
		}
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
			preferredTimeValue = convertPreferredTimeToTime(*dto.PreferredTime)
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
			categoryValue = nil
		}
		updates = append(updates, fmt.Sprintf("category = $%d", argIndex))
		args = append(args, categoryValue)
		argIndex++
	}

	// Обновление новых полей расписания
	// Важно: если меняется scheduleType, нужно очистить неиспользуемые поля
	if dto.ScheduleType != nil {
		updates = append(updates, fmt.Sprintf("schedule_type = $%d", argIndex))
		args = append(args, *dto.ScheduleType)
		argIndex++

		// При смене типа расписания автоматически очищаем неиспользуемые поля
		if *dto.ScheduleType == "recurring" {
			// Если меняем на recurring, очищаем one_time_date
			updates = append(updates, "one_time_date = NULL")
		} else if *dto.ScheduleType == "one_time" {
			// Если меняем на one_time, очищаем recurring_days
			updates = append(updates, "recurring_days = NULL")
		}
	}

	// Обновляем recurringDays
	// Если передан null (пустой слайс или nil), очищаем поле
	// Если передан непустой массив, обновляем только если scheduleType = recurring
	if dto.RecurringDays != nil {
		// Определяем актуальный тип расписания
		var scheduleTypeToCheck string
		if dto.ScheduleType != nil {
			scheduleTypeToCheck = *dto.ScheduleType
		} else {
			// Если scheduleType не меняется, проверяем текущий тип
			err := r.db.QueryRowContext(ctx, "SELECT schedule_type FROM habits WHERE id = $1 AND user_id = $2", id, userID).Scan(&scheduleTypeToCheck)
			if err != nil && err != sql.ErrNoRows {
				return nil, fmt.Errorf("failed to get current schedule type: %w", err)
			}
		}

		// Обновляем recurringDays только если тип recurring
		if scheduleTypeToCheck == "recurring" {
			updates = append(updates, fmt.Sprintf("recurring_days = $%d", argIndex))
			args = append(args, pq.Array(*dto.RecurringDays))
			argIndex++

			// Автоматически обновляем target_days на количество выбранных дней
			// target_days = количество дней в recurring_days
			targetDays := len(*dto.RecurringDays)
			if targetDays > 0 {
				updates = append(updates, fmt.Sprintf("target_days = $%d", argIndex))
				args = append(args, targetDays)
				argIndex++
			}
		}
	}

	// Обновляем oneTimeDate (только если scheduleType = one_time)
	// Примечание: если scheduleType меняется на recurring, oneTimeDate уже очищен выше
	if dto.OneTimeDate != nil {
		// Определяем актуальный тип расписания
		var scheduleTypeToCheck string
		if dto.ScheduleType != nil {
			scheduleTypeToCheck = *dto.ScheduleType
		} else {
			// Если scheduleType не меняется, проверяем текущий тип
			err := r.db.QueryRowContext(ctx, "SELECT schedule_type FROM habits WHERE id = $1 AND user_id = $2", id, userID).Scan(&scheduleTypeToCheck)
			if err != nil && err != sql.ErrNoRows {
				return nil, fmt.Errorf("failed to get current schedule type: %w", err)
			}
		}

		// Обновляем oneTimeDate только если тип one_time
		if scheduleTypeToCheck == "one_time" {
			if *dto.OneTimeDate == "" {
				// Очищаем поле, если передана пустая строка
				updates = append(updates, "one_time_date = NULL")
			} else {
				// Обновляем дату
				oneTimeDate, err := time.Parse("2006-01-02", *dto.OneTimeDate)
				if err != nil {
					return nil, fmt.Errorf("invalid one_time_date format: %w", err)
				}
				oneTimeDateValue := normalizeDate(oneTimeDate)
				updates = append(updates, fmt.Sprintf("one_time_date = $%d", argIndex))
				args = append(args, oneTimeDateValue)
				argIndex++
			}
		}
	}

	if dto.IsActive != nil {
		updates = append(updates, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, *dto.IsActive)
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
			schedule_type, recurring_days, one_time_date, is_active,
			user_id, workspace_id, 
			created_at, updated_at
	`, strings.Join(updates, ", "), argIndex, argIndex+1)

	args = append(args, id, userID)

	var habit model.Habit
	var createdAt, updatedAt time.Time
	var preferredTimePtr sql.NullString
	var categoryPtr sql.NullString
	var oneTimeDatePtr sql.NullTime
	var recurringDaysArray pq.Int32Array

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
		&habit.ScheduleType,
		&recurringDaysArray,
		&oneTimeDatePtr,
		&habit.IsActive,
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
	if oneTimeDatePtr.Valid {
		habit.OneTimeDate = oneTimeDatePtr.Time.Format("2006-01-02")
	}

	// Конвертируем pq.Int32Array в []int
	if recurringDaysArray != nil {
		habit.RecurringDays = make([]int, len(recurringDaysArray))
		for i, v := range recurringDaysArray {
			habit.RecurringDays[i] = int(v)
		}
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
	var habit model.Habit
	var createdAt time.Time
	var oneTimeDatePtr sql.NullTime
	var recurringDaysArray pq.Int32Array

	err := r.db.QueryRowContext(ctx,
		`SELECT 
			schedule_type, recurring_days, one_time_date, created_at
		FROM habits WHERE id = $1 AND user_id = $2`,
		habitID, userID,
	).Scan(
		&habit.ScheduleType,
		&recurringDaysArray,
		&oneTimeDatePtr,
		&createdAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get habit: %w", err)
	}

	createdAtUTC := normalizeDate(createdAt.UTC())
	today := normalizeDate(time.Now().UTC())

	// Вычисляем общее количество дней активности
	// Для регулярных: считаем только дни недели из recurring_days с момента создания до сегодня
	// Для разовых: 1 день (если дата >= created_at и <= today) или 0
	var totalDays int
	if habit.ScheduleType == "recurring" {
		// Используем SQL для эффективного подсчета дней
		// Генерируем серию дат от created_at до today и считаем только те, где день недели в recurring_days
		countQuery := `
			SELECT COUNT(*) 
			FROM generate_series(
				DATE($1::timestamp),
				DATE($2::timestamp),
				'1 day'::interval
			) AS day
			WHERE EXTRACT(DOW FROM day) = ANY($3::integer[])
		`
		err = r.db.QueryRowContext(ctx, countQuery, createdAtUTC, today, pq.Array(recurringDaysArray)).Scan(&totalDays)
		if err != nil {
			return nil, fmt.Errorf("failed to count active days: %w", err)
		}
	} else if habit.ScheduleType == "one_time" {
		if oneTimeDatePtr.Valid {
			oneTimeDate := normalizeDate(oneTimeDatePtr.Time.UTC())
			// Проверяем, что дата выполнения >= даты создания и <= today
			if !oneTimeDate.Before(createdAtUTC) && !oneTimeDate.After(today) {
				totalDays = 1
			} else {
				totalDays = 0
			}
		} else {
			totalDays = 0
		}
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

	// Вычисляем серии (streaks) - учитываем только дни, когда привычка была активна
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

		// Функция для проверки, была ли привычка активна на указанную дату
		isHabitActiveOnDate := func(date time.Time) bool {
			normalizedDate := normalizeDate(date)
			// Привычка не может быть активна до даты создания
			if normalizedDate.Before(createdAtUTC) {
				return false
			}

			if habit.ScheduleType == "recurring" {
				dayOfWeek := int(normalizedDate.Weekday())
				// Конвертируем recurringDaysArray если еще не сделано
				var recurringDays []int
				if recurringDaysArray != nil {
					recurringDays = make([]int, len(recurringDaysArray))
					for i, v := range recurringDaysArray {
						recurringDays[i] = int(v)
					}
				}
				// Проверяем, есть ли этот день недели в recurring_days
				for _, day := range recurringDays {
					if day == dayOfWeek {
						return true
					}
				}
				return false
			} else if habit.ScheduleType == "one_time" {
				if oneTimeDatePtr.Valid {
					oneTimeDate := normalizeDate(oneTimeDatePtr.Time.UTC())
					return normalizedDate.Equal(oneTimeDate)
				}
				return false
			}
			return false
		}

		// Вычисляем currentStreak (от сегодня назад, только по активным дням)
		currentStreak = 0
		checkDate := todayNormalized
		for {
			// Проверяем, была ли привычка активна на эту дату
			if !isHabitActiveOnDate(checkDate) {
				break
			}

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
		// Фильтруем completionDates - оставляем только те, когда привычка была активна
		var activeCompletionDates []time.Time
		for _, date := range completionDates {
			if isHabitActiveOnDate(date) {
				activeCompletionDates = append(activeCompletionDates, date)
			}
		}

		if len(activeCompletionDates) > 0 {
			// Сортируем даты по возрастанию
			sortedDates := make([]time.Time, len(activeCompletionDates))
			copy(sortedDates, activeCompletionDates)
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
					// Проверяем, что даты идут подряд И между ними нет пропущенных активных дней
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
	// Нормализуем даты до начала дня для корректного сравнения с полем DATE
	normalizedStart := normalizeDate(startDate)
	normalizedEnd := normalizeDate(endDate)

	// Получаем completion за период для всех привычек пользователя
	completionsQuery := `
		SELECT hc.habit_id, hc.date
		FROM habit_completions hc
		INNER JOIN habits h ON hc.habit_id = h.id
		WHERE hc.user_id = $1 AND h.workspace_id = $2 AND hc.date BETWEEN $3 AND $4
	`

	rows, err := r.db.QueryContext(ctx, completionsQuery, userID, workspaceID, normalizedStart, normalizedEnd)
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
	for !current.After(normalizedEnd) {
		dateKey := current.Format("2006-01-02")

		// Получаем активные привычки для этого дня
		dayHabits, err := r.GetHabitsForDate(ctx, userID, workspaceID, current)
		if err != nil {
			return nil, fmt.Errorf("failed to get habits for date %s: %w", dateKey, err)
		}

		dayHabitsList := make([]struct {
			ID        string `json:"id"`
			Title     string `json:"title"`
			Completed bool   `json:"completed"`
			Color     string `json:"color"`
		}, 0)

		for _, habit := range dayHabits {
			habitID, err := uuid.Parse(habit.ID)
			if err != nil {
				continue
			}

			completed := false
			if compMap, exists := completionMap[dateKey]; exists {
				completed = compMap[habitID]
			}

			dayHabitsList = append(dayHabitsList, struct {
				ID        string `json:"id"`
				Title     string `json:"title"`
				Completed bool   `json:"completed"`
				Color     string `json:"color"`
			}{
				ID:        habit.ID,
				Title:     habit.Title,
				Completed: completed,
				Color:     habit.Color,
			})
		}

		days = append(days, model.CalendarDay{
			Date:   dateKey,
			Habits: dayHabitsList,
		})

		current = current.AddDate(0, 0, 1)
	}

	return &model.CalendarResponse{Days: days}, nil
}
