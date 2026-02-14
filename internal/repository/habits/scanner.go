package habits

import (
	"database/sql"
	"fmt"
	"time"

	"backend/internal/model"

	"github.com/lib/pq"
)

// scanHabits сканирует привычки из rows
func scanHabits(rows *sql.Rows) ([]model.Habit, error) {
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
		habits = append(habits, habit)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return habits, nil
}

// scanCompletions сканирует completions из rows
func scanCompletions(rows *sql.Rows) ([]model.HabitCompletion, error) {
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
			&completion.WorkspaceID,
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
