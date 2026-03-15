package notification

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"backend/internal/model"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("notification not found")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Upsert(ctx context.Context, userID string, dto *model.CreateNotificationDto) (*model.Notification, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}

	payload := dto.Payload
	if payload == nil {
		payload = json.RawMessage("{}")
	}

	var wsID interface{}
	if dto.WorkspaceID != nil && *dto.WorkspaceID != "" {
		if w, err := uuid.Parse(*dto.WorkspaceID); err == nil {
			wsID = w
		}
	}

	query := `
		INSERT INTO notifications (user_id, workspace_id, channel, event_type, event_key, title, subtitle, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (user_id, event_key) DO UPDATE SET
			title = EXCLUDED.title,
			subtitle = EXCLUDED.subtitle,
			payload = EXCLUDED.payload,
			created_at = EXCLUDED.created_at,
			read_at = NULL
		RETURNING id, user_id, workspace_id, channel, event_type, event_key, title, subtitle, payload, created_at, read_at
	`
	var n model.Notification
	var userIDOut uuid.UUID
	var wsIDOut sql.NullString
	var subtitle sql.NullString
	var readAt sql.NullTime
	err = r.db.QueryRowContext(ctx, query,
		uid, wsID, dto.Channel, dto.EventType, dto.EventKey, dto.Title, dto.Subtitle, payload,
	).Scan(
		&n.ID, &userIDOut, &wsIDOut, &n.Channel, &n.EventType, &n.EventKey, &n.Title, &subtitle, &n.Payload, &n.CreatedAt, &readAt,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert notification: %w", err)
	}
	n.UserID = userIDOut.String()
	if wsIDOut.Valid {
		n.WorkspaceID = &wsIDOut.String
	}
	if subtitle.Valid {
		n.Subtitle = &subtitle.String
	}
	if readAt.Valid {
		n.ReadAt = &readAt.Time
	}
	return &n, nil
}

func (r *Repository) List(ctx context.Context, userID string, opts model.NotificationListOpts) ([]model.Notification, int, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid user id: %w", err)
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	baseWhere := `WHERE user_id = $1`
	args := []interface{}{uid}
	argIdx := 2

	if opts.Channel != "" {
		baseWhere += fmt.Sprintf(" AND channel = $%d", argIdx)
		args = append(args, opts.Channel)
		argIdx++
	}
	if opts.UnreadOnly {
		baseWhere += " AND read_at IS NULL"
	}

	countQuery := `SELECT COUNT(*) FROM notifications ` + baseWhere
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count notifications: %w", err)
	}

	args = append(args, limit, opts.Offset)
	listQuery := `
		SELECT id, user_id, workspace_id, channel, event_type, event_key, title, subtitle, payload, created_at, read_at
		FROM notifications ` + baseWhere + `
		ORDER BY created_at DESC
		LIMIT $` + fmt.Sprintf("%d", argIdx) + ` OFFSET $` + fmt.Sprintf("%d", argIdx+1)

	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	var list []model.Notification
	for rows.Next() {
		var n model.Notification
		var userID uuid.UUID
		var wsID, subtitle sql.NullString
		var readAt sql.NullTime
		if err := rows.Scan(&n.ID, &userID, &wsID, &n.Channel, &n.EventType, &n.EventKey, &n.Title, &subtitle, &n.Payload, &n.CreatedAt, &readAt); err != nil {
			return nil, 0, fmt.Errorf("scan notification: %w", err)
		}
		if wsID.Valid {
			n.WorkspaceID = &wsID.String
		}
		if subtitle.Valid {
			n.Subtitle = &subtitle.String
		}
		if readAt.Valid {
			n.ReadAt = &readAt.Time
		}
		n.UserID = userID.String()
		list = append(list, n)
	}
	return list, total, nil
}

func (r *Repository) MarkRead(ctx context.Context, userID, notificationID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}
	nid, err := uuid.Parse(notificationID)
	if err != nil {
		return fmt.Errorf("invalid notification id: %w", err)
	}

	query := `UPDATE notifications SET read_at = $1 WHERE id = $2 AND user_id = $3`
	res, err := r.db.ExecContext(ctx, query, time.Now(), nid, uid)
	if err != nil {
		return fmt.Errorf("mark read: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) MarkAllRead(ctx context.Context, userID string, channel string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}

	query := `UPDATE notifications SET read_at = $1 WHERE user_id = $2`
	args := []interface{}{time.Now(), uid}
	if channel != "" {
		query += ` AND channel = $3`
		args = append(args, channel)
	}
	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("mark all read: %w", err)
	}
	return nil
}
