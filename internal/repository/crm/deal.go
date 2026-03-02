package crm

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"backend/internal/model"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type DealListOpts struct {
	Page        int
	Limit       int
	PipelineID  string
	StageID     string
	CompanyID   string
	ContactID   string
	OwnerID     string
	Status      string
	DateFrom    string
	DateTo      string
	SortBy      string
	SortOrder   string
}

func (r *Repository) DealList(ctx context.Context, workspaceID uuid.UUID, opts DealListOpts) ([]model.Deal, int, error) {
	base := `FROM crm_deals WHERE workspace_id = $1 AND deleted_at IS NULL`
	args := []interface{}{workspaceID}
	n := 2
	if opts.PipelineID != "" {
		base += fmt.Sprintf(" AND pipeline_id = $%d", n)
		args = append(args, opts.PipelineID)
		n++
	}
	if opts.StageID != "" {
		base += fmt.Sprintf(" AND stage_id = $%d", n)
		args = append(args, opts.StageID)
		n++
	}
	if opts.CompanyID != "" {
		base += fmt.Sprintf(" AND company_id = $%d", n)
		args = append(args, opts.CompanyID)
		n++
	}
	if opts.ContactID != "" {
		base += fmt.Sprintf(" AND contact_id = $%d", n)
		args = append(args, opts.ContactID)
		n++
	}
	if opts.OwnerID != "" {
		base += fmt.Sprintf(" AND owner_id = $%d", n)
		args = append(args, opts.OwnerID)
		n++
	}
	if opts.Status != "" {
		base += fmt.Sprintf(" AND status = $%d", n)
		args = append(args, opts.Status)
		n++
	}
	if opts.DateFrom != "" {
		base += fmt.Sprintf(" AND created_at >= $%d::date", n)
		args = append(args, opts.DateFrom)
		n++
	}
	if opts.DateTo != "" {
		base += fmt.Sprintf(" AND created_at <= $%d::date", n)
		args = append(args, opts.DateTo)
		n++
	}

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*)"+base, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	order := "ORDER BY created_at DESC"
	if opts.SortBy != "" {
		col := mapSortByToColumn(opts.SortBy)
		if col != "" {
			order = "ORDER BY " + col + " " + sortOrder(opts.SortOrder)
		}
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
	base += fmt.Sprintf(" %s LIMIT $%d OFFSET $%d", order, limitIdx, offsetIdx)

	rows, err := r.db.QueryContext(ctx, `SELECT id::text, workspace_id::text, name, contact_id::text, company_id::text, budget, currency, pipeline_id::text, stage_id::text, expected_close_date, actual_close_date, status, lost_reason, description, source, probability, tags, owner_id::text, created_at, updated_at `+base, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var list []model.Deal
	for rows.Next() {
		d, err := scanDealRow(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, *d)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func mapSortByToColumn(sortBy string) string {
	switch sortBy {
	case "createdAt":
		return "created_at"
	case "updatedAt":
		return "updated_at"
	case "name":
		return "name"
	case "budget":
		return "budget"
	case "expectedCloseDate":
		return "expected_close_date"
	case "status":
		return "status"
	default:
		return ""
	}
}

func scanDealRow(row interface {
	Scan(dest ...interface{}) error
}) (*model.Deal, error) {
	var id, name, currency, pipelineID, stageID, status, ownerID string
	var contactID, companyID sql.NullString
	var budget float64
	var expectedClose, actualClose sql.NullTime
	var lostReason, description, source sql.NullString
	var probability sql.NullInt32
	var tags pq.StringArray
	var createdAt, updatedAt time.Time
	err := row.Scan(&id, new(string), &name, &contactID, &companyID, &budget, &currency, &pipelineID, &stageID, &expectedClose, &actualClose, &status, &lostReason, &description, &source, &probability, &tags, &ownerID, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	d := &model.Deal{
		ID:         id,
		Name:       name,
		Budget:     budget,
		Currency:   currency,
		PipelineID: pipelineID,
		StageID:    stageID,
		Status:     status,
		OwnerID:    ownerID,
		CreatedAt:  createdAt.Format(time.RFC3339),
		UpdatedAt:  updatedAt.Format(time.RFC3339),
		Tags:       tags,
	}
	if contactID.Valid {
		d.ContactID = contactID.String
	}
	if companyID.Valid {
		d.CompanyID = companyID.String
	}
	if expectedClose.Valid {
		d.ExpectedCloseDate = expectedClose.Time.Format("2006-01-02")
	}
	if actualClose.Valid {
		d.ActualCloseDate = actualClose.Time.Format("2006-01-02")
	}
	if lostReason.Valid {
		d.LostReason = lostReason.String
	}
	if description.Valid {
		d.Description = description.String
	}
	if source.Valid {
		d.Source = source.String
	}
	if probability.Valid {
		p := int(probability.Int32)
		d.Probability = &p
	}
	return d, nil
}

func (r *Repository) DealGet(ctx context.Context, id, workspaceID uuid.UUID) (*model.Deal, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id::text, workspace_id::text, name, contact_id::text, company_id::text, budget, currency, pipeline_id::text, stage_id::text, expected_close_date, actual_close_date, status, lost_reason, description, source, probability, tags, owner_id::text, created_at, updated_at FROM crm_deals WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL`, id, workspaceID)
	return scanDealRow(row)
}

func (r *Repository) DealCreate(ctx context.Context, workspaceID string, d *model.Deal, userID string) error {
	id := uuid.New()
	wsID, _ := uuid.Parse(workspaceID)
	ownerID, _ := uuid.Parse(d.OwnerID)
	if d.OwnerID == "" {
		ownerID, _ = uuid.Parse(userID)
	}
	pipelineID, _ := uuid.Parse(d.PipelineID)
	stageID, _ := uuid.Parse(d.StageID)
	var contactID, companyID interface{}
	if d.ContactID != "" {
		contactID, _ = uuid.Parse(d.ContactID)
	}
	if d.CompanyID != "" {
		companyID, _ = uuid.Parse(d.CompanyID)
	}
	var expectedClose interface{}
	if d.ExpectedCloseDate != "" {
		expectedClose = d.ExpectedCloseDate
	}
	var prob interface{}
	if d.Probability != nil {
		prob = *d.Probability
	}
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `INSERT INTO crm_deals (id, workspace_id, name, contact_id, company_id, budget, currency, pipeline_id, stage_id, expected_close_date, status, description, source, probability, tags, owner_id, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'open',$11,$12,$13,$14,$15,$16,$16)`,
		id, wsID, d.Name, contactID, companyID, d.Budget, d.Currency, pipelineID, stageID, expectedClose, nullStr(d.Description), nullStr(d.Source), prob, pq.Array(d.Tags), ownerID, now)
	if err != nil {
		return fmt.Errorf("insert deal: %w", err)
	}
	d.ID = id.String()
	d.CreatedAt = now.Format(time.RFC3339)
	d.UpdatedAt = d.CreatedAt
	d.Status = "open"
	if d.OwnerID == "" {
		d.OwnerID = userID
	}
	return nil
}

func (r *Repository) DealUpdate(ctx context.Context, workspaceID string, d *model.Deal) error {
	id, _ := uuid.Parse(d.ID)
	wsID, _ := uuid.Parse(workspaceID)
	stageID, _ := uuid.Parse(d.StageID)
	var contactID, companyID, expectedClose, actualClose, lostReason, description, source interface{}
	if d.ContactID != "" {
		contactID, _ = uuid.Parse(d.ContactID)
	}
	if d.CompanyID != "" {
		companyID, _ = uuid.Parse(d.CompanyID)
	}
	if d.ExpectedCloseDate != "" {
		expectedClose = d.ExpectedCloseDate
	}
	if d.ActualCloseDate != "" {
		actualClose = d.ActualCloseDate
	}
	if d.LostReason != "" {
		lostReason = d.LostReason
	}
	if d.Description != "" {
		description = d.Description
	}
	if d.Source != "" {
		source = d.Source
	}
	var prob interface{}
	if d.Probability != nil {
		prob = *d.Probability
	}
	res, err := r.db.ExecContext(ctx, `UPDATE crm_deals SET name=$3, contact_id=$4, company_id=$5, budget=$6, currency=$7, stage_id=$8, expected_close_date=$9, actual_close_date=$10, status=$11, lost_reason=$12, description=$13, source=$14, probability=$15, tags=$16, owner_id=$17, updated_at=NOW() WHERE id=$1 AND workspace_id=$2`,
		id, wsID, d.Name, contactID, companyID, d.Budget, d.Currency, stageID, expectedClose, actualClose, d.Status, lostReason, description, source, prob, pq.Array(d.Tags), d.OwnerID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) DealDelete(ctx context.Context, id, workspaceID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `UPDATE crm_deals SET deleted_at = NOW() WHERE id = $1 AND workspace_id = $2`, id, workspaceID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) DealStageBelongsToPipeline(ctx context.Context, stageID, pipelineID uuid.UUID) (bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT 1 FROM crm_stages WHERE id = $1 AND pipeline_id = $2`, stageID, pipelineID).Scan(&n)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}
