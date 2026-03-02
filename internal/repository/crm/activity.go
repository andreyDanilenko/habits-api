package crm

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"backend/internal/model"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type ActivityListOpts struct {
	Page          int
	Limit         int
	Types         []string
	DateFrom      string
	DateTo        string
	ImportantOnly bool
	Search        string
}

func (r *Repository) ActivityList(ctx context.Context, workspaceID uuid.UUID, entityType, entityID string, opts ActivityListOpts) ([]model.CrmActivity, int, error) {
	base := `FROM crm_activities WHERE workspace_id = $1 AND entity_type = $2 AND entity_id = $3 AND deleted_at IS NULL`
	args := []interface{}{workspaceID, entityType, entityID}
	n := 4
	if len(opts.Types) > 0 {
		base += fmt.Sprintf(" AND type = ANY($%d)", n)
		args = append(args, pq.Array(opts.Types))
		n++
	}
	if opts.DateFrom != "" {
		base += fmt.Sprintf(" AND created_at >= $%d::date", n)
		args = append(args, opts.DateFrom)
		n++
	}
	if opts.DateTo != "" {
		base += fmt.Sprintf(" AND created_at <= $%d::date + interval '1 day'", n)
		args = append(args, opts.DateTo)
		n++
	}
	if opts.ImportantOnly {
		base += " AND is_important = true"
	}
	if opts.Search != "" {
		base += fmt.Sprintf(" AND (title ILIKE $%d OR description ILIKE $%d)", n, n)
		args = append(args, "%"+opts.Search+"%")
		n++
	}

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*)"+base, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	offset := (opts.Page - 1) * limit
	if offset < 0 {
		offset = 0
	}
	limitIdx := len(args) + 1
	offsetIdx := limitIdx + 1
	args = append(args, limit, offset)
	base += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", limitIdx, offsetIdx)

	rows, err := r.db.QueryContext(ctx, `SELECT id, type, entity_type, entity_id, title, description, metadata, is_important, created_by, created_by_name, created_by_avatar, is_editable, is_deletable, created_at `+base, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var list []model.CrmActivity
	for rows.Next() {
		a, err := scanActivityRow(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, *a)
	}
	return list, total, rows.Err()
}

func scanActivityRow(row interface {
	Scan(dest ...interface{}) error
}) (*model.CrmActivity, error) {
	var id, aType, entityType, entityID, title, createdByName string
	var description, createdByAvatar sql.NullString
	var metadata []byte
	var isImportant, isEditable, isDeletable bool
	var createdBy uuid.UUID
	var createdAt time.Time
	err := row.Scan(&id, &aType, &entityType, &entityID, &title, &description, &metadata, &isImportant, &createdBy, &createdByName, &createdByAvatar, &isEditable, &isDeletable, &createdAt)
	if err != nil {
		return nil, err
	}
	a := &model.CrmActivity{
		ID:          id,
		Type:        aType,
		EntityType:  entityType,
		EntityID:    entityID,
		Title:       title,
		IsImportant: isImportant,
		CreatedBy:   model.CrmActivityCreatedBy{ID: createdBy.String(), Name: createdByName},
		CreatedAt:   createdAt.Format(time.RFC3339),
		IsEditable:  isEditable,
		IsDeletable: isDeletable,
	}
	if createdByAvatar.Valid {
		a.CreatedBy.Avatar = createdByAvatar.String
	}
	if description.Valid {
		a.Description = description.String
	}
	if len(metadata) > 0 {
		_ = json.Unmarshal(metadata, &a.Metadata)
	}
	return a, nil
}

func (r *Repository) ActivityGet(ctx context.Context, id, workspaceID uuid.UUID) (*model.CrmActivity, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, type, entity_type, entity_id, title, description, metadata, is_important, created_by, created_by_name, created_by_avatar, is_editable, is_deletable, created_at FROM crm_activities WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL`, id, workspaceID)
	return scanActivityRow(row)
}

func (r *Repository) ActivityCreate(ctx context.Context, workspaceID string, a *model.CrmActivity, createdByUserID, createdByName string) error {
	aid := uuid.New()
	wsID, _ := uuid.Parse(workspaceID)
	entityID, _ := uuid.Parse(a.EntityID)
	cb, _ := uuid.Parse(createdByUserID)
	metaJSON, _ := json.Marshal(a.Metadata)
	avatar := a.CreatedBy.Avatar
	if avatar == "" {
		avatar = ""
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO crm_activities (id, workspace_id, type, entity_type, entity_id, title, description, metadata, is_important, created_by, created_by_name, created_by_avatar, is_editable, is_deletable, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,NOW(),NOW())`,
		aid, wsID, a.Type, a.EntityType, entityID, a.Title, nullStr(a.Description), metaJSON, a.IsImportant, cb, createdByName, nullStr(avatar), a.IsEditable, a.IsDeletable)
	if err != nil {
		return err
	}
	a.ID = aid.String()
	a.CreatedBy.ID = createdByUserID
	a.CreatedBy.Name = createdByName
	a.CreatedAt = time.Now().Format(time.RFC3339)
	return nil
}

func (r *Repository) ActivityUpdate(ctx context.Context, workspaceID string, a *model.CrmActivity) error {
	id, _ := uuid.Parse(a.ID)
	wsID, _ := uuid.Parse(workspaceID)
	metaJSON, _ := json.Marshal(a.Metadata)
	res, err := r.db.ExecContext(ctx, `UPDATE crm_activities SET title=$3, description=$4, metadata=$5, is_important=$6, updated_at=NOW() WHERE id=$1 AND workspace_id=$2 AND type='note' AND is_editable=true`, id, wsID, a.Title, nullStr(a.Description), metaJSON, a.IsImportant)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) ActivitySetImportant(ctx context.Context, id, workspaceID uuid.UUID, isImportant bool) error {
	res, err := r.db.ExecContext(ctx, `UPDATE crm_activities SET is_important=$3, updated_at=NOW() WHERE id=$1 AND workspace_id=$2 AND deleted_at IS NULL`, id, workspaceID, isImportant)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) ActivityDelete(ctx context.Context, id, workspaceID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `UPDATE crm_activities SET deleted_at=NOW() WHERE id=$1 AND workspace_id=$2 AND type='note' AND is_deletable=true`, id, workspaceID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// CreateSystemActivity creates an automatic (non-editable) activity event.
func (r *Repository) CreateSystemActivity(ctx context.Context, workspaceID string, a *model.CrmActivity, createdByUserID, createdByName string) error {
	a.IsEditable = false
	a.IsDeletable = false
	return r.ActivityCreate(ctx, workspaceID, a, createdByUserID, createdByName)
}
