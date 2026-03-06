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
	db          *sql.DB
	versions    *VersionRepository
	completions *CompletionRepository
	statsCalc   *StatsCalculator
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		db:          db,
		versions:    NewVersionRepository(db),
		completions: NewCompletionRepository(db),
		statsCalc:   &StatsCalculator{},
	}
}

// List возвращает все привычки воркспейса (видят все участники, в т.ч. админ в чужом воркспейсе).
func (r *Repository) List(ctx context.Context, workspaceID uuid.UUID, targetDate *time.Time) ([]model.Habit, error) {
	if targetDate != nil {
		return r.GetHabitsForDate(ctx, workspaceID, *targetDate)
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT h.id, h.title, h.description, h.color, h.icon, h.target_days, h.daily_goal, h.preferred_time, h.category,
			h.schedule_type, h.recurring_days, h.one_time_date, h.is_active, h.user_id, h.workspace_id, h.created_at, h.updated_at,
			COALESCE(u.name, u.email, '') as owner_name
		FROM habits h
		LEFT JOIN users u ON u.id = h.user_id
		WHERE h.workspace_id = $1
		ORDER BY h.preferred_time NULLS LAST, h.created_at DESC
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query habits: %w", err)
	}
	defer rows.Close()
	return scanHabitsWithOwner(rows)
}

// GetHabitsForDate возвращает все привычки воркспейса, активные на указанную дату.
func (r *Repository) GetHabitsForDate(ctx context.Context, workspaceID uuid.UUID, targetDate time.Time) ([]model.Habit, error) {
	normalizedDate := NormalizeDate(targetDate)

	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT ON (hv.habit_id)
			hv.habit_id AS id, hv.title, hv.description, hv.color, hv.icon, hv.target_days, hv.daily_goal, hv.preferred_time, hv.category,
			hv.schedule_type, hv.recurring_days, hv.one_time_date, hv.is_active, hv.user_id, hv.workspace_id,
			(hv.valid_from)::timestamp AS created_at, COALESCE(hv.valid_to, hv.valid_from)::timestamp AS updated_at,
			COALESCE(u.name, u.email, '') as owner_name
		FROM habit_versions hv
		LEFT JOIN users u ON u.id = hv.user_id
		WHERE hv.workspace_id = $1 AND hv.is_active = true
			AND $2::date BETWEEN hv.valid_from AND COALESCE(hv.valid_to, $2::date)
			AND (
				(hv.schedule_type = 'recurring' AND EXTRACT(DOW FROM $2::date) = ANY(hv.recurring_days))
				OR (hv.schedule_type = 'one_time' AND hv.one_time_date = $2::date)
			)
		ORDER BY hv.habit_id, (hv.valid_to IS NOT NULL) DESC, hv.valid_from DESC
	`, workspaceID, normalizedDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query habits for date (versions): %w", err)
	}
	defer rows.Close()

	habits, err := scanHabitsWithOwner(rows)
	if err != nil {
		return nil, err
	}

	todayStart := NormalizeDate(time.Now().UTC())
	if len(habits) == 0 && !normalizedDate.Before(todayStart) {
		fallbackRows, err := r.db.QueryContext(ctx, `
			SELECT h.id, h.title, h.description, h.color, h.icon, h.target_days, h.daily_goal, h.preferred_time, h.category,
				h.schedule_type, h.recurring_days, h.one_time_date, h.is_active, h.user_id, h.workspace_id, h.created_at, h.updated_at,
				COALESCE(u.name, u.email, '') as owner_name
			FROM habits h
			LEFT JOIN users u ON u.id = h.user_id
			WHERE h.workspace_id = $1 AND h.is_active = true AND DATE(h.created_at) <= $2::date
				AND (
					(h.schedule_type = 'recurring' AND EXTRACT(DOW FROM $2::date) = ANY(h.recurring_days))
					OR (h.schedule_type = 'one_time' AND h.one_time_date = $2::date)
				)
			ORDER BY h.preferred_time NULLS LAST, h.created_at DESC
		`, workspaceID, normalizedDate)
		if err != nil {
			return nil, fmt.Errorf("failed to query habits for date (fallback): %w", err)
		}
		defer fallbackRows.Close()
		return scanHabitsWithOwner(fallbackRows)
	}
	return habits, nil
}

func (r *Repository) Create(ctx context.Context, dto model.CreateHabitDto, userID, workspaceID uuid.UUID) (*model.Habit, error) {
	query := `
		INSERT INTO habits (
			id, title, description, color, icon, target_days, daily_goal, preferred_time, category,
			schedule_type, recurring_days, one_time_date, is_active, user_id, workspace_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id, title, description, color, icon, target_days, daily_goal, preferred_time, category,
			schedule_type, recurring_days, one_time_date, is_active, user_id, workspace_id, created_at, updated_at
	`

	now := time.Now().UTC()
	habitID := uuid.New()

	var categoryValue, preferredTimeValue interface{}
	if dto.Category != "" {
		categoryValue = dto.Category
	}
	if dto.PreferredTime != "" && dto.PreferredTime != "any" {
		preferredTimeValue = ConvertPreferredTimeToTime(dto.PreferredTime)
	}

	scheduleType := dto.ScheduleType
	if scheduleType == "" {
		scheduleType = "recurring"
	}

	var recurringDaysValue, oneTimeDateValue interface{}
	targetDays := dto.TargetDays

	if scheduleType == "recurring" {
		recurringDays := dto.RecurringDays
		if len(recurringDays) == 0 {
			recurringDays = []int{0, 1, 2, 3, 4, 5, 6}
		}
		recurringDaysValue = pq.Array(recurringDays)
		if targetDays == 0 {
			targetDays = len(recurringDays)
		}
	} else if scheduleType == "one_time" {
		if dto.OneTimeDate == "" {
			return nil, fmt.Errorf("one_time_date is required for one_time schedule type")
		}
		oneTimeDate, err := time.Parse("2006-01-02", dto.OneTimeDate)
		if err != nil {
			return nil, fmt.Errorf("invalid one_time_date format: %w", err)
		}
		oneTimeDateValue = NormalizeDate(oneTimeDate)
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

	dailyGoal := dto.DailyGoal
	if dailyGoal == 0 {
		dailyGoal = 1
	}
	color := dto.Color
	if color == "" {
		color = "#3B82F6"
	}

	var habit model.Habit
	var createdAt, updatedAt time.Time
	var preferredTimePtr sql.NullString
	var categoryPtr sql.NullString
	var oneTimeDatePtr sql.NullTime
	var recurringDaysArray pq.Int32Array

	err := r.db.QueryRowContext(ctx, query,
		habitID, dto.Title, dto.Description, color, dto.Icon,
		targetDays, dailyGoal, preferredTimeValue, categoryValue, scheduleType,
		recurringDaysValue, oneTimeDateValue, isActive, userID, workspaceID, now, now,
	).Scan(
		&habit.ID, &habit.Title, &habit.Description, &habit.Color, &habit.Icon,
		&habit.TargetDays, &habit.DailyGoal, &preferredTimePtr, &categoryPtr, &habit.ScheduleType,
		&recurringDaysArray, &oneTimeDatePtr, &habit.IsActive, &habit.UserID, &habit.WorkspaceID,
		&createdAt, &updatedAt,
	)
	if err != nil {
		log.Printf("Error creating habit: %v, dto: %+v", err, dto)
		return nil, fmt.Errorf("failed to create habit: %w", err)
	}

	if err := r.versions.Create(ctx, habit.ID, habit.UserID, habit.WorkspaceID,
		habit.Title, habit.Description, habit.Color, habit.Icon,
		habit.TargetDays, habit.DailyGoal, habit.PreferredTime, habit.Category,
		habit.ScheduleType, habit.RecurringDays, habit.OneTimeDate, habit.IsActive,
		NormalizeDate(createdAt)); err != nil {
		log.Printf("Error creating habit version: %v, habitID: %s", err, habit.ID)
	}

	habit = applyScannedHabit(habit, preferredTimePtr, categoryPtr, oneTimeDatePtr, recurringDaysArray, createdAt, updatedAt)
	return &habit, nil
}

// Get возвращает привычку по id и user_id (только свою). Для операций, требующих владельца.
func (r *Repository) Get(ctx context.Context, id, userID uuid.UUID) (*model.Habit, error) {
	return r.scanOneHabit(ctx, `
		SELECT id, title, description, color, icon, target_days, daily_goal, preferred_time, category,
			schedule_type, recurring_days, one_time_date, is_active, user_id, workspace_id, created_at, updated_at
		FROM habits WHERE id = $1 AND user_id = $2
	`, id, userID)
}

// GetByIDAndWorkspace возвращает привычку по id и workspace_id (любой участник workspace может просматривать).
func (r *Repository) GetByIDAndWorkspace(ctx context.Context, id, workspaceID uuid.UUID) (*model.Habit, error) {
	return r.scanOneHabitWithOwner(ctx, `
		SELECT h.id, h.title, h.description, h.color, h.icon, h.target_days, h.daily_goal, h.preferred_time, h.category,
			h.schedule_type, h.recurring_days, h.one_time_date, h.is_active, h.user_id, h.workspace_id, h.created_at, h.updated_at,
			COALESCE(u.name, u.email, '') as owner_name
		FROM habits h
		LEFT JOIN users u ON u.id = h.user_id
		WHERE h.id = $1 AND h.workspace_id = $2
	`, id, workspaceID)
}

func (r *Repository) scanOneHabit(ctx context.Context, query string, args ...interface{}) (*model.Habit, error) {
	var habit model.Habit
	var createdAt, updatedAt time.Time
	var preferredTimePtr sql.NullString
	var categoryPtr sql.NullString
	var oneTimeDatePtr sql.NullTime
	var recurringDaysArray pq.Int32Array

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&habit.ID, &habit.Title, &habit.Description, &habit.Color, &habit.Icon,
		&habit.TargetDays, &habit.DailyGoal, &preferredTimePtr, &categoryPtr, &habit.ScheduleType,
		&recurringDaysArray, &oneTimeDatePtr, &habit.IsActive, &habit.UserID, &habit.WorkspaceID,
		&createdAt, &updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get habit: %w", err)
	}
	h := applyScannedHabit(habit, preferredTimePtr, categoryPtr, oneTimeDatePtr, recurringDaysArray, createdAt, updatedAt)
	return &h, nil
}

func (r *Repository) scanOneHabitWithOwner(ctx context.Context, query string, args ...interface{}) (*model.Habit, error) {
	var habit model.Habit
	var createdAt, updatedAt time.Time
	var preferredTimePtr sql.NullString
	var categoryPtr sql.NullString
	var oneTimeDatePtr sql.NullTime
	var recurringDaysArray pq.Int32Array
	var ownerName sql.NullString

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&habit.ID, &habit.Title, &habit.Description, &habit.Color, &habit.Icon,
		&habit.TargetDays, &habit.DailyGoal, &preferredTimePtr, &categoryPtr, &habit.ScheduleType,
		&recurringDaysArray, &oneTimeDatePtr, &habit.IsActive, &habit.UserID, &habit.WorkspaceID,
		&createdAt, &updatedAt, &ownerName,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get habit: %w", err)
	}
	if ownerName.Valid {
		habit.OwnerName = ownerName.String
	}
	h := applyScannedHabit(habit, preferredTimePtr, categoryPtr, oneTimeDatePtr, recurringDaysArray, createdAt, updatedAt)
	return &h, nil
}

func (r *Repository) Update(ctx context.Context, id, userID uuid.UUID, dto model.UpdateHabitDto) (*model.Habit, error) {
	updates := []string{"updated_at = $1"}
	args := []interface{}{time.Now().UTC()}
	argIndex := 2
	shouldVersion := false

	if dto.Title != nil {
		updates, args, argIndex, shouldVersion = appendUpdate(updates, args, argIndex, "title", *dto.Title, shouldVersion, true)
	}
	if dto.Description != nil {
		updates, args, argIndex, shouldVersion = appendUpdate(updates, args, argIndex, "description", *dto.Description, shouldVersion, true)
	}
	if dto.Color != nil {
		updates, args, argIndex, shouldVersion = appendUpdate(updates, args, argIndex, "color", *dto.Color, shouldVersion, true)
	}
	if dto.Icon != nil {
		updates, args, argIndex, _ = appendUpdate(updates, args, argIndex, "icon", *dto.Icon, shouldVersion, false)
	}
	if dto.TargetDays != nil && *dto.TargetDays > 0 {
		updates, args, argIndex, _ = appendUpdate(updates, args, argIndex, "target_days", *dto.TargetDays, shouldVersion, false)
	}
	if dto.DailyGoal != nil && *dto.DailyGoal > 0 {
		updates, args, argIndex, _ = appendUpdate(updates, args, argIndex, "daily_goal", *dto.DailyGoal, shouldVersion, false)
	}
	if dto.PreferredTime != nil {
		var v interface{}
		if *dto.PreferredTime != "" && *dto.PreferredTime != "any" {
			v = ConvertPreferredTimeToTime(*dto.PreferredTime)
		}
		updates = append(updates, fmt.Sprintf("preferred_time = $%d", argIndex))
		args = append(args, v)
		argIndex++
	}
	if dto.Category != nil {
		var v interface{}
		if *dto.Category != "" {
			v = *dto.Category
		}
		updates = append(updates, fmt.Sprintf("category = $%d", argIndex))
		args = append(args, v)
		argIndex++
	}
	if dto.ScheduleType != nil {
		updates = append(updates, fmt.Sprintf("schedule_type = $%d", argIndex))
		args = append(args, *dto.ScheduleType)
		argIndex++
		shouldVersion = true
		if *dto.ScheduleType == "recurring" {
			updates = append(updates, "one_time_date = NULL")
		} else if *dto.ScheduleType == "one_time" {
			updates = append(updates, "recurring_days = NULL")
		}
	}

	scheduleTypeToCheck := ""
	if dto.ScheduleType != nil {
		scheduleTypeToCheck = *dto.ScheduleType
	} else {
		_ = r.db.QueryRowContext(ctx, "SELECT schedule_type FROM habits WHERE id = $1 AND user_id = $2", id, userID).Scan(&scheduleTypeToCheck)
	}

	if dto.RecurringDays != nil && scheduleTypeToCheck == "recurring" {
		updates = append(updates, fmt.Sprintf("recurring_days = $%d", argIndex))
		args = append(args, pq.Array(*dto.RecurringDays))
		argIndex++
		shouldVersion = true
		if len(*dto.RecurringDays) > 0 {
			updates = append(updates, fmt.Sprintf("target_days = $%d", argIndex))
			args = append(args, len(*dto.RecurringDays))
			argIndex++
		}
	}

	if dto.OneTimeDate != nil && scheduleTypeToCheck == "one_time" {
		shouldVersion = true
		if *dto.OneTimeDate == "" {
			updates = append(updates, "one_time_date = NULL")
		} else {
			oneTimeDate, err := time.Parse("2006-01-02", *dto.OneTimeDate)
			if err != nil {
				return nil, fmt.Errorf("invalid one_time_date format: %w", err)
			}
			updates = append(updates, fmt.Sprintf("one_time_date = $%d", argIndex))
			args = append(args, NormalizeDate(oneTimeDate))
			argIndex++
		}
	}

	if dto.IsActive != nil {
		updates = append(updates, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, *dto.IsActive)
		argIndex++
		shouldVersion = true
	}

	if len(updates) == 1 {
		return r.Get(ctx, id, userID)
	}

	var oldHabit *model.Habit
	if shouldVersion {
		oldHabit, _ = r.Get(ctx, id, userID)
	}

	query := fmt.Sprintf(`
		UPDATE habits SET %s WHERE id = $%d AND user_id = $%d
		RETURNING id, title, description, color, icon, target_days, daily_goal, preferred_time, category,
			schedule_type, recurring_days, one_time_date, is_active, user_id, workspace_id, created_at, updated_at
	`, strings.Join(updates, ", "), argIndex, argIndex+1)
	args = append(args, id, userID)

	var habit model.Habit
	var createdAt, updatedAt time.Time
	var preferredTimePtr sql.NullString
	var categoryPtr sql.NullString
	var oneTimeDatePtr sql.NullTime
	var recurringDaysArray pq.Int32Array

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&habit.ID, &habit.Title, &habit.Description, &habit.Color, &habit.Icon,
		&habit.TargetDays, &habit.DailyGoal, &preferredTimePtr, &categoryPtr, &habit.ScheduleType,
		&recurringDaysArray, &oneTimeDatePtr, &habit.IsActive, &habit.UserID, &habit.WorkspaceID,
		&createdAt, &updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to update habit: %w", err)
	}

	if shouldVersion {
		changeDate := NormalizeDate(time.Now().UTC())
		nextDay := changeDate.AddDate(0, 0, 1)

		closed, err := r.versions.ClosePrevious(ctx, habit.ID, habit.UserID, habit.WorkspaceID, changeDate)
		if err != nil {
			log.Printf("Error closing previous habit version: %v, habitID: %s", err, habit.ID)
		}

		if closed == 0 && oldHabit != nil {
			createdAtParsed, _ := time.Parse(time.RFC3339, oldHabit.CreatedAt)
			validFrom := NormalizeDate(createdAtParsed.UTC())
			if err := r.versions.Create(ctx, oldHabit.ID, oldHabit.UserID, oldHabit.WorkspaceID,
				oldHabit.Title, oldHabit.Description, oldHabit.Color, oldHabit.Icon,
				oldHabit.TargetDays, oldHabit.DailyGoal, oldHabit.PreferredTime, oldHabit.Category,
				oldHabit.ScheduleType, oldHabit.RecurringDays, oldHabit.OneTimeDate, oldHabit.IsActive, validFrom); err != nil {
				log.Printf("Error creating backfill habit version: %v, habitID: %s", err, habit.ID)
			} else {
				_, _ = r.db.ExecContext(ctx, `
					UPDATE habit_versions SET valid_to = $1
					WHERE habit_id = $2 AND user_id = $3 AND workspace_id = $4 AND valid_to IS NULL
				`, changeDate, habit.ID, habit.UserID, habit.WorkspaceID)
			}
		}

		if err := r.versions.Create(ctx, habit.ID, habit.UserID, habit.WorkspaceID,
			habit.Title, habit.Description, habit.Color, habit.Icon,
			habit.TargetDays, habit.DailyGoal, habit.PreferredTime, habit.Category,
			habit.ScheduleType, habit.RecurringDays, habit.OneTimeDate, habit.IsActive, nextDay); err != nil {
			log.Printf("Error creating new habit version: %v, habitID: %s", err, habit.ID)
		}
	}

	h := applyScannedHabit(habit, preferredTimePtr, categoryPtr, oneTimeDatePtr, recurringDaysArray, createdAt, updatedAt)
	return &h, nil
}

// Delete удаляет привычку. Требует habitOwnerID == requestUserID (удаляет только свою) или isWorkspaceOwner.
func (r *Repository) Delete(ctx context.Context, id, habitOwnerID uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var workspaceID uuid.UUID
	if err := tx.QueryRowContext(ctx, "SELECT workspace_id FROM habits WHERE id = $1 AND user_id = $2", id, habitOwnerID).Scan(&workspaceID); err != nil {
		if err == sql.ErrNoRows {
			return sql.ErrNoRows
		}
		return fmt.Errorf("failed to get habit workspace_id: %w", err)
	}

	deleteDate := NormalizeDate(time.Now().UTC())
	_, err = tx.ExecContext(ctx, `
		UPDATE habit_versions SET valid_to = $1
		WHERE habit_id = $2 AND user_id = $3 AND workspace_id = $4 AND valid_to IS NULL
	`, deleteDate, id, habitOwnerID, workspaceID)
	if err != nil {
		return fmt.Errorf("failed to close habit version: %w", err)
	}

	result, err := tx.ExecContext(ctx, "DELETE FROM habits WHERE id = $1 AND user_id = $2", id, habitOwnerID)
	if err != nil {
		return fmt.Errorf("failed to delete habit: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}

func (r *Repository) Complete(ctx context.Context, habitID, userID uuid.UUID, date time.Time, notes string, rating interface{}, completionTime *string) (*model.HabitCompletion, error) {
	return r.completions.Create(ctx, habitID, userID, date, notes, rating, completionTime)
}

func (r *Repository) Toggle(ctx context.Context, habitID, userID uuid.UUID, date time.Time) (bool, *model.HabitCompletion, error) {
	return r.completions.Toggle(ctx, habitID, userID, date)
}

// GetStats возвращает статистику привычки. habitOwnerID — владелец привычки (для подсчёта completions).
func (r *Repository) GetStats(ctx context.Context, habitID, habitOwnerID uuid.UUID) (*model.HabitStats, error) {
	var scheduleType string
	var recurringDaysArray pq.Int32Array
	var oneTimeDatePtr sql.NullTime
	var createdAt time.Time

	err := r.db.QueryRowContext(ctx, `
		SELECT schedule_type, recurring_days, one_time_date, created_at
		FROM habits WHERE id = $1 AND user_id = $2
	`, habitID, habitOwnerID).Scan(&scheduleType, &recurringDaysArray, &oneTimeDatePtr, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get habit: %w", err)
	}

	createdAtUTC := NormalizeDate(createdAt.UTC())
	today := NormalizeDate(time.Now().UTC())

	var totalDays int
	if scheduleType == "recurring" {
		err = r.db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM generate_series(DATE($1::timestamp), DATE($2::timestamp), '1 day'::interval) AS day
			WHERE EXTRACT(DOW FROM day) = ANY($3::integer[])
		`, createdAtUTC, today, pq.Array(recurringDaysArray)).Scan(&totalDays)
		if err != nil {
			return nil, fmt.Errorf("failed to count active days: %w", err)
		}
	} else if scheduleType == "one_time" && oneTimeDatePtr.Valid {
		oneTimeDate := NormalizeDate(oneTimeDatePtr.Time.UTC())
		if !oneTimeDate.Before(createdAtUTC) && !oneTimeDate.After(today) {
			totalDays = 1
		}
	}

	completedDays, err := r.completions.CountByHabit(ctx, habitID, habitOwnerID)
	if err != nil {
		return nil, fmt.Errorf("failed to count completions: %w", err)
	}

	completionDates, err := r.completions.GetCompletionDates(ctx, habitID, habitOwnerID)
	if err != nil {
		return nil, fmt.Errorf("failed to query completion dates: %w", err)
	}

	info := HabitScheduleInfo{
		ScheduleType:  scheduleType,
		RecurringDays: ConvertRecurringDays(recurringDaysArray),
		CreatedAtUTC:  createdAtUTC,
	}
	if oneTimeDatePtr.Valid {
		t := oneTimeDatePtr.Time
		info.OneTimeDate = &t
	}

	currentStreak, longestStreak := r.statsCalc.CalculateStreaks(completionDates, info)

	completionRate := 0.0
	if totalDays > 0 {
		completionRate = float64(completedDays) / float64(totalDays)
	}

	return &model.HabitStats{
		HabitID:        habitID.String(),
		CompletedDays:  completedDays,
		TotalDays:      totalDays,
		CompletionRate: completionRate,
		CurrentStreak:  currentStreak,
		LongestStreak:  longestStreak,
	}, nil
}

func (r *Repository) GetCompletions(ctx context.Context, habitID, userID uuid.UUID, startDate, endDate time.Time) ([]model.HabitCompletion, error) {
	return r.completions.GetByHabitAndDateRange(ctx, habitID, userID, startDate, endDate)
}

func (r *Repository) GetAllCompletions(ctx context.Context, userID, workspaceID uuid.UUID, startDate, endDate time.Time) ([]model.HabitCompletion, error) {
	return r.completions.GetAllByWorkspaceAndDateRange(ctx, userID, workspaceID, startDate, endDate)
}

func (r *Repository) GetCalendar(ctx context.Context, userID, workspaceID uuid.UUID, startDate, endDate time.Time) (*model.CalendarResponse, error) {
	normalizedStart := NormalizeDate(startDate)
	normalizedEnd := NormalizeDate(endDate)
	todayStart := NormalizeDate(time.Now().UTC())

	completionMap, err := r.completions.GetCompletionMap(ctx, workspaceID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	days := make([]model.CalendarDay, 0)
	current := normalizedStart
	for !current.After(normalizedEnd) {
		dateKey := current.Format("2006-01-02")
		dayHabits, err := r.GetHabitsForDate(ctx, workspaceID, current)
		if err != nil {
			return nil, fmt.Errorf("failed to get habits for date %s: %w", dateKey, err)
		}

		dayHabitsList := make([]struct {
			ID        string `json:"id"`
			Title     string `json:"title"`
			Completed bool   `json:"completed"`
			Color     string `json:"color"`
		}, 0)
		seenIDs := make(map[string]bool)

		for _, habit := range dayHabits {
			hid, _ := uuid.Parse(habit.ID)
			seenIDs[habit.ID] = true
			completed := false
			if compMap, ok := completionMap[dateKey]; ok {
				completed = compMap[hid]
			}
			dayHabitsList = append(dayHabitsList, struct {
				ID        string `json:"id"`
				Title     string `json:"title"`
				Completed bool   `json:"completed"`
				Color     string `json:"color"`
			}{habit.ID, habit.Title, completed, habit.Color})
		}

		if current.Before(todayStart) && completionMap[dateKey] != nil {
			for habitID := range completionMap[dateKey] {
				if seenIDs[habitID.String()] {
					continue
				}
				vid, vtitle, vcolor, ok := r.versions.GetForDate(ctx, habitID, workspaceID, current)
				if !ok {
					continue
				}
				seenIDs[vid] = true
				dayHabitsList = append(dayHabitsList, struct {
					ID        string `json:"id"`
					Title     string `json:"title"`
					Completed bool   `json:"completed"`
					Color     string `json:"color"`
				}{vid, vtitle, true, vcolor})
			}
		}

		days = append(days, model.CalendarDay{Date: dateKey, Habits: dayHabitsList})
		current = current.AddDate(0, 0, 1)
	}

	return &model.CalendarResponse{Days: days}, nil
}

func applyScannedHabit(habit model.Habit, preferredTimePtr sql.NullString, categoryPtr sql.NullString, oneTimeDatePtr sql.NullTime, recurringDaysArray pq.Int32Array, createdAt, updatedAt time.Time) model.Habit {
	if preferredTimePtr.Valid {
		habit.PreferredTime = ConvertTimeToPreferredTime(preferredTimePtr.String)
	}
	if categoryPtr.Valid {
		habit.Category = categoryPtr.String
	}
	if oneTimeDatePtr.Valid {
		habit.OneTimeDate = oneTimeDatePtr.Time.Format("2006-01-02")
	}
	habit.RecurringDays = ConvertRecurringDays(recurringDaysArray)
	habit.CreatedAt = createdAt.Format(time.RFC3339)
	habit.UpdatedAt = updatedAt.Format(time.RFC3339)
	return habit
}

func appendUpdate(updates []string, args []interface{}, argIndex int, col string, val interface{}, shouldVersion, version bool) ([]string, []interface{}, int, bool) {
	updates = append(updates, fmt.Sprintf("%s = $%d", col, argIndex))
	args = append(args, val)
	return updates, args, argIndex + 1, shouldVersion || version
}
