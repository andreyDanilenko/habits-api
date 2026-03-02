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

type CompanyListOpts struct {
	Page      int
	Limit     int
	Search    string
	OwnerID   string
	SortBy    string
	SortOrder string
}

func (r *Repository) CompanyList(ctx context.Context, workspaceID uuid.UUID, opts CompanyListOpts) ([]model.Company, int, error) {
	base := `FROM crm_companies WHERE workspace_id = $1 AND deleted_at IS NULL`
	args := []interface{}{workspaceID}
	n := 2
	if opts.Search != "" {
		base += fmt.Sprintf(" AND (name ILIKE $%d OR inn ILIKE $%d)", n, n)
		args = append(args, "%"+opts.Search+"%")
		n++
	}
	if opts.OwnerID != "" {
		base += fmt.Sprintf(" AND owner_id = $%d", n)
		args = append(args, opts.OwnerID)
		n++
	}

	var total int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*)"+base, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	order := "ORDER BY name ASC"
	if opts.SortBy == "createdAt" {
		order = "ORDER BY created_at " + sortOrder(opts.SortOrder)
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

	rows, err := r.db.QueryContext(ctx, `SELECT id, workspace_id, name, inn, kpp, ogrn, phone, email, website, legal_address, actual_address, tags, owner_id, created_at, updated_at `+base, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var list []model.Company
	for rows.Next() {
		co, err := scanCompanyRow(rows)
		if err != nil {
			return nil, 0, err
		}
		co.Contacts, err = r.companyContactIDs(ctx, co.ID)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, *co)
	}
	return list, total, rows.Err()
}

func (r *Repository) companyContactIDs(ctx context.Context, companyID string) ([]string, error) {
	cid, _ := uuid.Parse(companyID)
	rows, err := r.db.QueryContext(ctx, `SELECT contact_id FROM crm_company_contacts WHERE company_id = $1`, cid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id.String())
	}
	return ids, rows.Err()
}

func scanCompanyRow(row interface {
	Scan(dest ...interface{}) error
}) (*model.Company, error) {
	var id, name, ownerID string
	var inn, kpp, ogrn, phone, email, website sql.NullString
	var legalAddr, actualAddr []byte
	var tags pq.StringArray
	var createdAt, updatedAt time.Time
	err := row.Scan(&id, new(string), &name, &inn, &kpp, &ogrn, &phone, &email, &website, &legalAddr, &actualAddr, &tags, &ownerID, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	co := &model.Company{
		ID:        id,
		Name:      name,
		OwnerID:   ownerID,
		CreatedAt: createdAt.Format(time.RFC3339),
		UpdatedAt: updatedAt.Format(time.RFC3339),
		Tags:      tags,
	}
	if inn.Valid {
		co.INN = inn.String
	}
	if kpp.Valid {
		co.KPP = kpp.String
	}
	if ogrn.Valid {
		co.OGRN = ogrn.String
	}
	if phone.Valid {
		co.Phone = phone.String
	}
	if email.Valid {
		co.Email = email.String
	}
	if website.Valid {
		co.Website = website.String
	}
	if len(legalAddr) > 0 {
		var a model.CompanyAddress
		_ = json.Unmarshal(legalAddr, &a)
		co.LegalAddress = &a
	}
	if len(actualAddr) > 0 {
		var a model.CompanyAddress
		_ = json.Unmarshal(actualAddr, &a)
		co.ActualAddress = &a
	}
	return co, nil
}

func (r *Repository) CompanyGet(ctx context.Context, id, workspaceID uuid.UUID) (*model.Company, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, workspace_id, name, inn, kpp, ogrn, phone, email, website, legal_address, actual_address, tags, owner_id, created_at, updated_at FROM crm_companies WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL`, id, workspaceID)
	co, err := scanCompanyRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	co.Contacts, err = r.companyContactIDs(ctx, co.ID)
	if err != nil {
		return nil, err
	}
	return co, nil
}

func (r *Repository) CompanyCreate(ctx context.Context, workspaceID string, co *model.Company, userID string) error {
	id := uuid.New()
	wsID, _ := uuid.Parse(workspaceID)
	ownerID, _ := uuid.Parse(co.OwnerID)
	if co.OwnerID == "" {
		ownerID, _ = uuid.Parse(userID)
	}
	legalJSON, _ := json.Marshal(co.LegalAddress)
	actualJSON, _ := json.Marshal(co.ActualAddress)
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `INSERT INTO crm_companies (id, workspace_id, name, inn, kpp, ogrn, phone, email, website, legal_address, actual_address, tags, owner_id, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$14)`,
		id, wsID, co.Name, nullStr(co.INN), nullStr(co.KPP), nullStr(co.OGRN), nullStr(co.Phone), nullStr(co.Email), nullStr(co.Website), legalJSON, actualJSON, pq.Array(co.Tags), ownerID, now)
	if err != nil {
		return fmt.Errorf("insert company: %w", err)
	}
	co.ID = id.String()
	co.CreatedAt = now.Format(time.RFC3339)
	co.UpdatedAt = co.CreatedAt
	if co.OwnerID == "" {
		co.OwnerID = userID
	}
	return nil
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func (r *Repository) CompanyUpdate(ctx context.Context, workspaceID string, co *model.Company) error {
	id, _ := uuid.Parse(co.ID)
	wsID, _ := uuid.Parse(workspaceID)
	legalJSON, _ := json.Marshal(co.LegalAddress)
	actualJSON, _ := json.Marshal(co.ActualAddress)
	res, err := r.db.ExecContext(ctx, `UPDATE crm_companies SET name=$3, inn=$4, kpp=$5, ogrn=$6, phone=$7, email=$8, website=$9, legal_address=$10, actual_address=$11, tags=$12, owner_id=$13, updated_at=NOW() WHERE id=$1 AND workspace_id=$2`,
		id, wsID, co.Name, nullStr(co.INN), nullStr(co.KPP), nullStr(co.OGRN), nullStr(co.Phone), nullStr(co.Email), nullStr(co.Website), legalJSON, actualJSON, pq.Array(co.Tags), co.OwnerID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) CompanyDelete(ctx context.Context, id, workspaceID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `UPDATE crm_companies SET deleted_at = NOW() WHERE id = $1 AND workspace_id = $2`, id, workspaceID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) CompanyCountDeals(ctx context.Context, companyID, workspaceID uuid.UUID) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM crm_deals WHERE company_id = $1 AND workspace_id = $2 AND deleted_at IS NULL`, companyID, workspaceID).Scan(&n)
	return n, err
}

func (r *Repository) CompanyCountContacts(ctx context.Context, companyID uuid.UUID) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM crm_company_contacts WHERE company_id = $1`, companyID).Scan(&n)
	return n, err
}

func (r *Repository) CompanyAttachContact(ctx context.Context, companyID, contactID uuid.UUID, position string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO crm_company_contacts (company_id, contact_id, position, created_at) VALUES ($1,$2,$3,NOW()) ON CONFLICT (company_id, contact_id) DO UPDATE SET position=$3`, companyID, contactID, position)
	return err
}

func (r *Repository) CompanyDetachContact(ctx context.Context, companyID, contactID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM crm_company_contacts WHERE company_id = $1 AND contact_id = $2`, companyID, contactID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
