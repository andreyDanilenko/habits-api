package invitation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"backend/internal/model"

	"github.com/google/uuid"
)

var (
	ErrInvitationNotFound = errors.New("invitation not found")
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, inv *model.Invitation) error {
	query := `
		INSERT INTO invitations (id, workspace_id, email, invited_by, system_role, status, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	id := uuid.New().String()
	_, err := r.db.ExecContext(ctx, query,
		id, inv.WorkspaceID, inv.Email, inv.InvitedBy, inv.SystemRole, inv.Status, inv.Token,
		inv.ExpiresAt, inv.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create invitation: %w", err)
	}
	inv.ID = id
	return nil
}

func (r *Repository) GetByToken(ctx context.Context, token string) (*model.Invitation, error) {
	query := `
		SELECT id, workspace_id, email, invited_by, system_role, status, token, expires_at, created_at, accepted_at
		FROM invitations
		WHERE token = $1
	`
	var inv model.Invitation
	var acceptedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&inv.ID, &inv.WorkspaceID, &inv.Email, &inv.InvitedBy, &inv.SystemRole, &inv.Status,
		&inv.Token, &inv.ExpiresAt, &inv.CreatedAt, &acceptedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get invitation by token: %w", err)
	}
	if acceptedAt.Valid {
		inv.AcceptedAt = &acceptedAt.Time
	}
	return &inv, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*model.Invitation, error) {
	query := `
		SELECT id, workspace_id, email, invited_by, system_role, status, token, expires_at, created_at, accepted_at
		FROM invitations
		WHERE id = $1
	`
	var inv model.Invitation
	var acceptedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&inv.ID, &inv.WorkspaceID, &inv.Email, &inv.InvitedBy, &inv.SystemRole, &inv.Status,
		&inv.Token, &inv.ExpiresAt, &inv.CreatedAt, &acceptedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get invitation by id: %w", err)
	}
	if acceptedAt.Valid {
		inv.AcceptedAt = &acceptedAt.Time
	}
	return &inv, nil
}

func (r *Repository) ListByWorkspace(ctx context.Context, workspaceID string, status *string, limit, offset int) ([]model.Invitation, int, error) {
	countQuery := `SELECT COUNT(*) FROM invitations WHERE workspace_id = $1`
	args := []interface{}{workspaceID}
	if status != nil && *status != "" {
		countQuery += ` AND status = $2`
		args = append(args, *status)
	}
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count invitations: %w", err)
	}

	query := `
		SELECT id, workspace_id, email, invited_by, system_role, status, expires_at, created_at, accepted_at
		FROM invitations
		WHERE workspace_id = $1
	`
	args = []interface{}{workspaceID}
	argIdx := 2
	if status != nil && *status != "" {
		query += fmt.Sprintf(` AND status = $%d`, argIdx)
		args = append(args, *status)
		argIdx++
	}
	query += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list invitations: %w", err)
	}
	defer rows.Close()

	var list []model.Invitation
	for rows.Next() {
		var inv model.Invitation
		var acceptedAt sql.NullTime
		err := rows.Scan(&inv.ID, &inv.WorkspaceID, &inv.Email, &inv.InvitedBy, &inv.SystemRole, &inv.Status,
			&inv.ExpiresAt, &inv.CreatedAt, &acceptedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("scan invitation: %w", err)
		}
		if acceptedAt.Valid {
			inv.AcceptedAt = &acceptedAt.Time
		}
		list = append(list, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, id, status string, acceptedAt *time.Time) error {
	var res sql.Result
	var err error
	if acceptedAt != nil {
		res, err = r.db.ExecContext(ctx, `UPDATE invitations SET status = $2, accepted_at = $3 WHERE id = $1`, id, status, *acceptedAt)
	} else {
		res, err = r.db.ExecContext(ctx, `UPDATE invitations SET status = $2 WHERE id = $1`, id, status)
	}
	if err != nil {
		return fmt.Errorf("update invitation status: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrInvitationNotFound
	}
	return nil
}

func (r *Repository) Cancel(ctx context.Context, id string) error {
	return r.UpdateStatus(ctx, id, model.InvitationStatusCancelled, nil)
}

func (r *Repository) MarkAccepted(ctx context.Context, id string) error {
	now := time.Now()
	query := `UPDATE invitations SET status = $2, accepted_at = $3 WHERE id = $1`
	res, err := r.db.ExecContext(ctx, query, id, model.InvitationStatusAccepted, now)
	if err != nil {
		return fmt.Errorf("mark invitation accepted: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrInvitationNotFound
	}
	return nil
}
