package habits

import (
	"context"
	"database/sql"
	"time"

	"backend/internal/model"

	"github.com/google/uuid"
)

// Activity types for habits module
const (
	ActivityTypeHabitCreated   = "HABIT_CREATED"
	ActivityTypeHabitUpdated   = "HABIT_UPDATED"
	ActivityTypeHabitDeleted   = "HABIT_DELETED"
	ActivityTypeHabitCompleted = "HABIT_COMPLETED"
	ActivityTypeJournalCreated = "JOURNAL_CREATED"
	ActivityTypeJournalUpdated = "JOURNAL_UPDATED"
	ActivityTypeJournalDeleted = "JOURNAL_DELETED"
)

// Entity types
const (
	EntityTypeHabit      = "habit"
	EntityTypeCompletion = "completion"
	EntityTypeJournal   = "journal"
)

func (r *Repository) CreateActivity(ctx context.Context, userID, workspaceID, habitID uuid.UUID, activityType, title, emoji string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO activities (id, user_id, workspace_id, type, entity_type, entity_id, title, emoji, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, NOW())
	`, userID, workspaceID, activityType, EntityTypeHabit, habitID, title, emoji)
	return err
}

func (r *Repository) CreateCompletionActivity(ctx context.Context, userID, workspaceID, habitID uuid.UUID, title, emoji string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO activities (id, user_id, workspace_id, type, entity_type, entity_id, title, emoji, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, NOW())
	`, userID, workspaceID, ActivityTypeHabitCompleted, EntityTypeCompletion, habitID, title, emoji)
	return err
}

// CreateJournalActivity записывает активность по журналу (для виджета RecentActivity).
func (r *Repository) CreateJournalActivity(ctx context.Context, userID, workspaceID, entityID uuid.UUID, activityType, title, emoji string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO activities (id, user_id, workspace_id, type, entity_type, entity_id, title, emoji, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, NOW())
	`, userID, workspaceID, activityType, EntityTypeJournal, entityID, title, emoji)
	return err
}

// ListActivities returns recent activities for the workspace with pagination.
// userName — кто выполнил действие (из users).
func (r *Repository) ListActivities(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]model.Activity, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var total int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM activities WHERE workspace_id = $1
	`, workspaceID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT a.id, a.user_id, a.workspace_id, a.type, a.entity_type, a.entity_id, a.title, a.emoji, a.created_at,
			COALESCE(u.name, u.email, '') as user_name
		FROM activities a
		LEFT JOIN users u ON u.id = a.user_id
		WHERE a.workspace_id = $1
		ORDER BY a.created_at DESC
		LIMIT $2 OFFSET $3
	`, workspaceID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []model.Activity
	for rows.Next() {
		var a model.Activity
		var entityID uuid.UUID
		var createdAt time.Time
		var emoji sql.NullString
		var userName sql.NullString
		err := rows.Scan(&a.ID, &a.UserID, &a.WorkspaceID, &a.Type, &a.EntityType, &entityID, &a.Title, &emoji, &createdAt, &userName)
		if err != nil {
			return nil, 0, err
		}
		a.EntityID = entityID.String()
		a.CreatedAt = createdAt.Format(time.RFC3339)
		if emoji.Valid {
			a.Emoji = emoji.String
		}
		if userName.Valid {
			a.UserName = userName.String
		}
		list = append(list, a)
	}
	return list, total, rows.Err()
}
