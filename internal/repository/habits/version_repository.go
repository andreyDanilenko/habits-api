package habits

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// VersionRepository управляет версиями привычек для исторического календаря
type VersionRepository struct {
	db *sql.DB
}

// NewVersionRepository создает новый VersionRepository
func NewVersionRepository(db *sql.DB) *VersionRepository {
	return &VersionRepository{db: db}
}

// Create создает запись в habit_versions
func (r *VersionRepository) Create(
	ctx context.Context,
	habitID, userID, workspaceID string,
	title, description, color, icon string,
	targetDays, dailyGoal int,
	preferredTime, category, scheduleType string,
	recurringDays []int,
	oneTimeDate string,
	isActive bool,
	validFrom time.Time,
) error {
	var recurringDaysValue interface{}
	if len(recurringDays) > 0 {
		recurringDaysValue = pq.Array(recurringDays)
	} else {
		recurringDaysValue = nil
	}

	var oneTimeDateValue interface{}
	if oneTimeDate != "" {
		parsedDate, err := time.Parse("2006-01-02", oneTimeDate)
		if err == nil {
			oneTimeDateValue = NormalizeDate(parsedDate)
		}
	}

	var preferredTimeValue interface{}
	if preferredTime != "" && preferredTime != "any" {
		preferredTimeValue = ConvertPreferredTimeToTime(preferredTime)
	} else {
		preferredTimeValue = nil
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO habit_versions (
			habit_id, user_id, workspace_id,
			title, description, color, icon,
			target_days, daily_goal, preferred_time, category,
			schedule_type, recurring_days, one_time_date, is_active,
			valid_from
		) VALUES (
			$1, $2, $3,
			$4, $5, $6, $7,
			$8, $9, $10, $11,
			$12, $13, $14, $15,
			$16
		)
	`, habitID, userID, workspaceID,
		title, description, color, icon,
		targetDays, dailyGoal, preferredTimeValue, category,
		scheduleType, recurringDaysValue, oneTimeDateValue, isActive,
		NormalizeDate(validFrom),
	)
	return err
}

// GetForDate возвращает версию привычки (id, title, color) на указанную дату
func (r *VersionRepository) GetForDate(ctx context.Context, habitID, workspaceID uuid.UUID, targetDate time.Time) (id, title, color string, ok bool) {
	normalized := NormalizeDate(targetDate)
	var hid, t, c string
	err := r.db.QueryRowContext(ctx, `
		SELECT habit_id, title, color
		FROM habit_versions
		WHERE habit_id = $1 AND workspace_id = $2
		  AND $3::date BETWEEN valid_from AND COALESCE(valid_to, $3::date)
		ORDER BY valid_from DESC
		LIMIT 1
	`, habitID, workspaceID, normalized).Scan(&hid, &t, &c)
	if err != nil {
		return "", "", "", false
	}
	return hid, t, c, true
}

// ClosePrevious закрывает текущую открытую версию (устанавливает valid_to)
func (r *VersionRepository) ClosePrevious(ctx context.Context, habitID, userID, workspaceID string, validTo time.Time) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE habit_versions
		SET valid_to = $1
		WHERE habit_id = $2 
			AND user_id = $3
			AND workspace_id = $4
			AND valid_to IS NULL
	`, NormalizeDate(validTo), habitID, userID, workspaceID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
