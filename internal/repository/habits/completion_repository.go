package habits

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"backend/internal/model"

	"github.com/google/uuid"
)

// CompletionRepository управляет completions привычек
type CompletionRepository struct {
	db *sql.DB
}

// NewCompletionRepository создает новый CompletionRepository
func NewCompletionRepository(db *sql.DB) *CompletionRepository {
	return &CompletionRepository{db: db}
}

// Create создает запись о выполнении привычки
func (r *CompletionRepository) Create(ctx context.Context, habitID, userID uuid.UUID, date time.Time, notes string, rating interface{}, completionTime *string) (*model.HabitCompletion, error) {
	query := `
		INSERT INTO habit_completions (
			id, habit_id, user_id, workspace_id, date, notes, rating, time, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, habit_id, user_id, workspace_id, date, notes, rating, time, created_at
	`

	completionID := uuid.New()
	now := time.Now().UTC()
	normalizedDate := NormalizeDate(date)

	var workspaceID uuid.UUID
	err := r.db.QueryRowContext(ctx,
		"SELECT workspace_id FROM habits WHERE id = $1 AND user_id = $2",
		habitID, userID,
	).Scan(&workspaceID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("habit not found for completion")
		}
		return nil, fmt.Errorf("failed to get workspace_id for completion: %w", err)
	}

	var timeValue, ratingValue interface{}
	if completionTime != nil && *completionTime != "" {
		timeValue = *completionTime
	}
	if rating != nil {
		if ratingInt, ok := rating.(int); ok && ratingInt == 0 {
			ratingValue = nil
		} else {
			ratingValue = rating
		}
	}

	var completion model.HabitCompletion
	var completionDate, createdAt time.Time
	var timePtr sql.NullString
	var ratingPtr sql.NullInt64

	err = r.db.QueryRowContext(ctx, query,
		completionID, habitID, userID, workspaceID, normalizedDate,
		notes, ratingValue, timeValue, now,
	).Scan(
		&completion.ID, &completion.HabitID, &completion.UserID, &completion.WorkspaceID,
		&completionDate, &completion.Notes, &ratingPtr, &timePtr, &createdAt,
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

// Toggle переключает выполнение привычки на дату (добавляет или удаляет)
func (r *CompletionRepository) Toggle(ctx context.Context, habitID, userID uuid.UUID, date time.Time) (bool, *model.HabitCompletion, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	normalizedDate := NormalizeDate(date)
	var existing model.HabitCompletion
	var existingDate, createdAt time.Time
	var timePtr sql.NullString
	var ratingPtr sql.NullInt64

	err = tx.QueryRowContext(ctx, `
		SELECT id, habit_id, user_id, date, notes, rating, time, created_at
		FROM habit_completions 
		WHERE habit_id = $1 AND user_id = $2 AND date = $3
		LIMIT 1
	`, habitID, userID, normalizedDate).Scan(
		&existing.ID, &existing.HabitID, &existing.UserID, &existingDate,
		&existing.Notes, &ratingPtr, &timePtr, &createdAt,
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

		_, err = tx.ExecContext(ctx, "DELETE FROM habit_completions WHERE id = $1", existing.ID)
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

	completion, err := r.Create(ctx, habitID, userID, normalizedDate, "", nil, nil)
	if err != nil {
		return false, nil, err
	}
	if err := tx.Commit(); err != nil {
		return false, nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	return true, completion, nil
}

// GetByHabitAndDateRange возвращает completions для привычки за период
func (r *CompletionRepository) GetByHabitAndDateRange(ctx context.Context, habitID, userID uuid.UUID, startDate, endDate time.Time) ([]model.HabitCompletion, error) {
	query := `
		SELECT id, habit_id, user_id, workspace_id, date, notes, rating, time, created_at
		FROM habit_completions 
		WHERE habit_id = $1 AND user_id = $2 AND date BETWEEN $3 AND $4
		ORDER BY date DESC, time DESC
	`
	rows, err := r.db.QueryContext(ctx, query, habitID, userID, NormalizeDate(startDate), NormalizeDate(endDate))
	if err != nil {
		return nil, fmt.Errorf("failed to query completions: %w", err)
	}
	defer rows.Close()
	return scanCompletions(rows)
}

// GetCompletionDates возвращает даты выполнений для привычки (для расчёта streaks)
func (r *CompletionRepository) GetCompletionDates(ctx context.Context, habitID, userID uuid.UUID) ([]time.Time, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT date FROM habit_completions
		WHERE habit_id = $1 AND user_id = $2 ORDER BY date DESC
	`, habitID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query completion dates: %w", err)
	}
	defer rows.Close()
	var dates []time.Time
	for rows.Next() {
		var d time.Time
		if err := rows.Scan(&d); err != nil {
			return nil, fmt.Errorf("failed to scan date: %w", err)
		}
		dates = append(dates, d.UTC())
	}
	return dates, rows.Err()
}

// CountByHabit возвращает количество выполнений для привычки
func (r *CompletionRepository) CountByHabit(ctx context.Context, habitID, userID uuid.UUID) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM habit_completions WHERE habit_id = $1 AND user_id = $2",
		habitID, userID,
	).Scan(&n)
	return n, err
}

// GetCompletionMap возвращает мапу dateKey -> habitID -> true для быстрого поиска completions
func (r *CompletionRepository) GetCompletionMap(ctx context.Context, userID, workspaceID uuid.UUID, startDate, endDate time.Time) (map[string]map[uuid.UUID]bool, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT habit_id, date FROM habit_completions
		WHERE user_id = $1 AND workspace_id = $2 AND date BETWEEN $3 AND $4
	`, userID, workspaceID, NormalizeDate(startDate), NormalizeDate(endDate))
	if err != nil {
		return nil, fmt.Errorf("failed to query completions: %w", err)
	}
	defer rows.Close()
	m := make(map[string]map[uuid.UUID]bool)
	for rows.Next() {
		var habitID uuid.UUID
		var date time.Time
		if err := rows.Scan(&habitID, &date); err != nil {
			return nil, fmt.Errorf("failed to scan: %w", err)
		}
		key := date.Format("2006-01-02")
		if m[key] == nil {
			m[key] = make(map[uuid.UUID]bool)
		}
		m[key][habitID] = true
	}
	return m, rows.Err()
}

// GetAllByWorkspaceAndDateRange возвращает все completions воркспейса за период
func (r *CompletionRepository) GetAllByWorkspaceAndDateRange(ctx context.Context, userID, workspaceID uuid.UUID, startDate, endDate time.Time) ([]model.HabitCompletion, error) {
	query := `
		SELECT id, habit_id, user_id, workspace_id, date, notes, rating, time, created_at
		FROM habit_completions
		WHERE user_id = $1 AND workspace_id = $2 AND date BETWEEN $3 AND $4
		ORDER BY date DESC, time DESC
	`
	rows, err := r.db.QueryContext(ctx, query, userID, workspaceID, NormalizeDate(startDate), NormalizeDate(endDate))
	if err != nil {
		return nil, fmt.Errorf("failed to query completions: %w", err)
	}
	defer rows.Close()
	return scanCompletions(rows)
}
