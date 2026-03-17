package task

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

func (r *Repository) List(ctx context.Context, workspaceID uuid.UUID, filters *model.TaskListFilters) ([]model.Task, error) {
	base := `
		SELECT t.id, t.workspace_id, t.title, t.description, t.type, t.priority, t.status, t.due_date, t.due_time,
		       t.reminder_date, t.duration, t.completed_at, t.completed_by, t.completion_note,
		       t.is_recurring, t.recurring_pattern, t.parent_id, t.assignee_id, t.created_by,
		       t.created_at, t.updated_at, t.spent_minutes
		FROM tasks t
	`
	args := []interface{}{workspaceID}
	argIdx := 2
	where := " WHERE t.workspace_id = $1 AND t.deleted_at IS NULL"

	if filters != nil {
		if filters.EntityType != "" && filters.EntityID != "" {
			where += fmt.Sprintf(" AND EXISTS (SELECT 1 FROM task_entity_links tel WHERE tel.task_id = t.id AND tel.entity_type = $%d AND tel.entity_id = $%d)", argIdx, argIdx+1)
			args = append(args, filters.EntityType, filters.EntityID)
			argIdx += 2
		}
		if filters.Status != "" {
			where += fmt.Sprintf(" AND t.status = $%d", argIdx)
			args = append(args, filters.Status)
			argIdx++
		}
		if filters.Priority != "" {
			where += fmt.Sprintf(" AND t.priority = $%d", argIdx)
			args = append(args, filters.Priority)
			argIdx++
		}
		if filters.Type != "" {
			where += fmt.Sprintf(" AND t.type = $%d", argIdx)
			args = append(args, filters.Type)
			argIdx++
		}
		if filters.AssigneeID != "" {
			where += fmt.Sprintf(" AND t.assignee_id = $%d", argIdx)
			args = append(args, filters.AssigneeID)
			argIdx++
		}
		if filters.OverdueOnly {
			where += " AND t.status != 'completed' AND t.due_date < CURRENT_DATE"
		}
		if filters.Search != "" {
			// Экранируем % и _ для ILIKE
			safe := strings.ReplaceAll(filters.Search, "\\", "\\\\")
			safe = strings.ReplaceAll(safe, "%", "\\%")
			safe = strings.ReplaceAll(safe, "_", "\\_")
			pattern := "%" + safe + "%"
			where += fmt.Sprintf(" AND (t.title ILIKE $%d OR (t.description IS NOT NULL AND t.description ILIKE $%d))", argIdx, argIdx+1)
			args = append(args, pattern, pattern)
			argIdx += 2
		}
		if filters.ParentID == "root" {
			where += " AND t.parent_id IS NULL"
		} else if filters.ParentID != "" {
			where += fmt.Sprintf(" AND t.parent_id = $%d", argIdx)
			args = append(args, filters.ParentID)
			argIdx++
		}
	}

	query := base + where + " ORDER BY t.due_date ASC, t.created_at DESC"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var list []model.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *t)
	}
	return list, rows.Err()
}

func (r *Repository) Get(ctx context.Context, taskID, workspaceID uuid.UUID) (*model.Task, error) {
	query := `
		SELECT id, workspace_id, title, description, type, priority, status, due_date, due_time,
		       reminder_date, duration, completed_at, completed_by, completion_note,
		       is_recurring, recurring_pattern, parent_id, assignee_id, created_by,
		       created_at, updated_at, spent_minutes
		FROM tasks
		WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
	`
	t, err := scanTaskRow(r.db.QueryRowContext(ctx, query, taskID, workspaceID))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	entities, err := r.listEntityLinks(ctx, taskID)
	if err != nil {
		return nil, err
	}
	t.Entities = entities
	return t, nil
}

func (r *Repository) Create(ctx context.Context, t *model.Task, createdBy uuid.UUID) error {
	wsID, _ := uuid.Parse(t.WorkspaceID)
	assigneeID, _ := uuid.Parse(t.AssigneeID)
	id := uuid.New()
	now := time.Now()

	dueDate, err := parseTimestamp(t.DueDate)
	if err != nil {
		return fmt.Errorf("invalid due_date: %w", err)
	}
	var reminderDate *time.Time
	if t.ReminderDate != "" {
		rd, err := parseTimestamp(t.ReminderDate)
		if err != nil {
			return fmt.Errorf("invalid reminder_date: %w", err)
		}
		reminderDate = &rd
	}

	var parentID *uuid.UUID
	if t.ParentID != "" {
		if pid, err := uuid.Parse(t.ParentID); err == nil {
			parentID = &pid
		}
	}

	query := `
		INSERT INTO tasks (id, workspace_id, title, description, type, priority, status, due_date, due_time,
		                   reminder_date, duration, parent_id, assignee_id, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING id, created_at, updated_at
	`
	var createdAt, updatedAt time.Time
	err = r.db.QueryRowContext(ctx, query,
		id, wsID, t.Title, nullStr(t.Description), t.Type, t.Priority, t.Status,
		dueDate, nullStr(t.DueTime), reminderDate, t.Duration,
		parentID, assigneeID, createdBy, now, now,
	).Scan(&t.ID, &createdAt, &updatedAt)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}
	t.CreatedAt = createdAt.Format(time.RFC3339)
	t.UpdatedAt = updatedAt.Format(time.RFC3339)
	t.CreatedBy = createdBy.String()

	for _, e := range t.Entities {
		if err := r.insertEntityLink(ctx, id, e); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) Update(ctx context.Context, t *model.Task) error {
	wsID, _ := uuid.Parse(t.WorkspaceID)
	taskID, _ := uuid.Parse(t.ID)
	now := time.Now()

	dueDate, err := parseTimestamp(t.DueDate)
	if err != nil {
		return fmt.Errorf("invalid due_date: %w", err)
	}
	var reminderDate *time.Time
	if t.ReminderDate != "" {
		rd, err := parseTimestamp(t.ReminderDate)
		if err != nil {
			return fmt.Errorf("invalid reminder_date: %w", err)
		}
		reminderDate = &rd
	}

	query := `
		UPDATE tasks SET
			title = $3, description = $4, type = $5, priority = $6, status = $7,
			due_date = $8, due_time = $9, reminder_date = $10, duration = $11,
			assignee_id = $12, updated_at = $13
		WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
		RETURNING id, created_at, updated_at
	`
	assigneeID, _ := uuid.Parse(t.AssigneeID)
	var createdAt, updatedAt time.Time
	err = r.db.QueryRowContext(ctx, query,
		taskID, wsID, t.Title, nullStr(t.Description), t.Type, t.Priority, t.Status,
		dueDate, nullStr(t.DueTime), reminderDate, t.Duration,
		assigneeID, now,
	).Scan(&t.ID, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return sql.ErrNoRows
		}
		return fmt.Errorf("update task: %w", err)
	}
	t.CreatedAt = createdAt.Format(time.RFC3339)
	t.UpdatedAt = updatedAt.Format(time.RFC3339)

	if err := r.replaceEntityLinks(ctx, taskID, t.Entities); err != nil {
		return err
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, taskID, workspaceID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE tasks SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL`,
		taskID, workspaceID,
	)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) Complete(ctx context.Context, taskID, workspaceID uuid.UUID, completedBy uuid.UUID, note string) (*model.Task, error) {
	now := time.Now()
	query := `
		UPDATE tasks SET
			status = 'completed', completed_at = $3, completed_by = $4, completion_note = $5,
			updated_at = $6
		WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
		RETURNING id, workspace_id, title, description, type, priority, status, due_date, due_time,
		          reminder_date, duration, completed_at, completed_by, completion_note,
		          is_recurring, recurring_pattern, parent_id, assignee_id, created_by,
		          created_at, updated_at, spent_minutes
	`
	t, err := scanTaskRow(r.db.QueryRowContext(ctx, query, taskID, workspaceID, now, completedBy, nullStr(note), now))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("complete task: %w", err)
	}
	t.Entities, _ = r.listEntityLinks(ctx, taskID)
	return t, nil
}

func (r *Repository) Reopen(ctx context.Context, taskID, workspaceID uuid.UUID) (*model.Task, error) {
	query := `
		UPDATE tasks SET
			status = 'pending', completed_at = NULL, completed_by = NULL, completion_note = NULL,
			updated_at = $3
		WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL AND status = 'completed'
		RETURNING id, workspace_id, title, description, type, priority, status, due_date, due_time,
		          reminder_date, duration, completed_at, completed_by, completion_note,
		          is_recurring, recurring_pattern, parent_id, assignee_id, created_by,
		          created_at, updated_at, spent_minutes
	`
	t, err := scanTaskRow(r.db.QueryRowContext(ctx, query, taskID, workspaceID, time.Now()))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("reopen task: %w", err)
	}
	t.Entities, _ = r.listEntityLinks(ctx, taskID)
	return t, nil
}

func (r *Repository) listEntityLinks(ctx context.Context, taskID uuid.UUID) ([]model.TaskEntityLink, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT entity_type, entity_id, entity_name FROM task_entity_links WHERE task_id = $1 ORDER BY created_at`,
		taskID,
	)
	if err != nil {
		return nil, fmt.Errorf("list entity links: %w", err)
	}
	defer rows.Close()

	var links []model.TaskEntityLink
	for rows.Next() {
		var e model.TaskEntityLink
		var entityName sql.NullString
		if err := rows.Scan(&e.EntityType, &e.EntityID, &entityName); err != nil {
			return nil, err
		}
		if entityName.Valid {
			e.EntityName = entityName.String
		}
		links = append(links, e)
	}
	return links, rows.Err()
}

func (r *Repository) insertEntityLink(ctx context.Context, taskID uuid.UUID, e model.TaskEntityLink) error {
	entityID, err := uuid.Parse(e.EntityID)
	if err != nil {
		return fmt.Errorf("invalid entity_id: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO task_entity_links (task_id, entity_type, entity_id, entity_name) VALUES ($1, $2, $3, $4)`,
		taskID, e.EntityType, entityID, nullStr(e.EntityName),
	)
	if err != nil {
		return fmt.Errorf("insert entity link: %w", err)
	}
	return nil
}

func (r *Repository) replaceEntityLinks(ctx context.Context, taskID uuid.UUID, entities []model.TaskEntityLink) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM task_entity_links WHERE task_id = $1`, taskID)
	if err != nil {
		return fmt.Errorf("delete entity links: %w", err)
	}
	for _, e := range entities {
		if err := r.insertEntityLink(ctx, taskID, e); err != nil {
			return err
		}
	}
	return nil
}

func scanTask(rows *sql.Rows) (*model.Task, error) {
	var t model.Task
	var desc, dueTime, completedBy, completionNote, parentID sql.NullString
	var reminderDate, completedAt sql.NullTime
	var duration, spentMinutes sql.NullInt64
	var dueDate, createdAt, updatedAt time.Time
	var recurringPattern []byte

	err := rows.Scan(
		&t.ID, &t.WorkspaceID, &t.Title, &desc, &t.Type, &t.Priority, &t.Status,
		&dueDate, &dueTime, &reminderDate, &duration, &completedAt, &completedBy, &completionNote,
		&t.IsRecurring, &recurringPattern, &parentID, &t.AssigneeID, &t.CreatedBy,
		&createdAt, &updatedAt, &spentMinutes,
	)
	if err != nil {
		return nil, err
	}
	t.DueDate = dueDate.Format(time.RFC3339)
	t.CreatedAt = createdAt.Format(time.RFC3339)
	t.UpdatedAt = updatedAt.Format(time.RFC3339)
	if desc.Valid {
		t.Description = desc.String
	}
	if dueTime.Valid {
		t.DueTime = dueTime.String
	}
	if reminderDate.Valid {
		t.ReminderDate = reminderDate.Time.Format(time.RFC3339)
	}
	if duration.Valid {
		d := int(duration.Int64)
		t.Duration = &d
	}
	if completedAt.Valid {
		t.CompletedAt = completedAt.Time.Format(time.RFC3339)
	}
	if completedBy.Valid {
		t.CompletedBy = completedBy.String
	}
	if completionNote.Valid {
		t.CompletionNote = completionNote.String
	}
	if len(recurringPattern) > 0 {
		t.RecurringPattern = recurringPattern
	}
	if parentID.Valid {
		t.ParentID = parentID.String
	}
	if spentMinutes.Valid {
		t.SpentMinutes = int(spentMinutes.Int64)
	}
	return &t, nil
}

func scanTaskRow(row *sql.Row) (*model.Task, error) {
	var t model.Task
	var desc, dueTime, completedBy, completionNote, parentID sql.NullString
	var reminderDate, completedAt sql.NullTime
	var duration, spentMinutes sql.NullInt64
	var dueDate, createdAt, updatedAt time.Time
	var recurringPattern []byte

	err := row.Scan(
		&t.ID, &t.WorkspaceID, &t.Title, &desc, &t.Type, &t.Priority, &t.Status,
		&dueDate, &dueTime, &reminderDate, &duration, &completedAt, &completedBy, &completionNote,
		&t.IsRecurring, &recurringPattern, &parentID, &t.AssigneeID, &t.CreatedBy,
		&createdAt, &updatedAt, &spentMinutes,
	)
	if err != nil {
		return nil, err
	}
	t.DueDate = dueDate.Format(time.RFC3339)
	t.CreatedAt = createdAt.Format(time.RFC3339)
	t.UpdatedAt = updatedAt.Format(time.RFC3339)
	if desc.Valid {
		t.Description = desc.String
	}
	if dueTime.Valid {
		t.DueTime = dueTime.String
	}
	if reminderDate.Valid {
		t.ReminderDate = reminderDate.Time.Format(time.RFC3339)
	}
	if duration.Valid {
		d := int(duration.Int64)
		t.Duration = &d
	}
	if completedAt.Valid {
		t.CompletedAt = completedAt.Time.Format(time.RFC3339)
	}
	if completedBy.Valid {
		t.CompletedBy = completedBy.String
	}
	if completionNote.Valid {
		t.CompletionNote = completionNote.String
	}
	if len(recurringPattern) > 0 {
		t.RecurringPattern = recurringPattern
	}
	if parentID.Valid {
		t.ParentID = parentID.String
	}
	if spentMinutes.Valid {
		t.SpentMinutes = int(spentMinutes.Int64)
	}
	return &t, nil
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func parseTimestamp(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t, err = time.Parse("2006-01-02", s)
		if err != nil {
			return time.Time{}, err
		}
	}
	return t, nil
}

func (r *Repository) ListComments(ctx context.Context, taskID, workspaceID uuid.UUID) ([]model.TaskComment, error) {
	query := `
		SELECT tc.id, tc.task_id, tc.body, tc.created_by, tc.created_at
		FROM task_comments tc
		JOIN tasks t ON t.id = tc.task_id AND t.workspace_id = $2 AND t.deleted_at IS NULL
		WHERE tc.task_id = $1
		ORDER BY tc.created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, taskID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}
	defer rows.Close()

	var list []model.TaskComment
	for rows.Next() {
		var c model.TaskComment
		var createdAt time.Time
		if err := rows.Scan(&c.ID, &c.TaskID, &c.Body, &c.CreatedBy, &createdAt); err != nil {
			return nil, err
		}
		c.CreatedAt = createdAt.Format(time.RFC3339)
		list = append(list, c)
	}
	return list, rows.Err()
}

func (r *Repository) CreateComment(ctx context.Context, taskID, workspaceID, createdBy uuid.UUID, body string) (*model.TaskComment, error) {
	var c model.TaskComment
	query := `
		INSERT INTO task_comments (task_id, body, created_by)
		SELECT $1, $2, $3
		FROM tasks t
		WHERE t.id = $1 AND t.workspace_id = $4 AND t.deleted_at IS NULL
		RETURNING id, task_id, body, created_by, created_at
	`
	var createdAt time.Time
	err := r.db.QueryRowContext(ctx, query, taskID, body, createdBy, workspaceID).Scan(
		&c.ID, &c.TaskID, &c.Body, &c.CreatedBy, &createdAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("create comment: %w", err)
	}
	c.CreatedAt = createdAt.Format(time.RFC3339)
	return &c, nil
}

func (r *Repository) DeleteComment(ctx context.Context, commentID, workspaceID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM task_comments tc
		USING tasks t
		WHERE tc.id = $1 AND tc.task_id = t.id AND t.workspace_id = $2 AND t.deleted_at IS NULL
	`, commentID, workspaceID)
	if err != nil {
		return fmt.Errorf("delete comment: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
