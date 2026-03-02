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

func (r *Repository) ContactList(ctx context.Context, workspaceID uuid.UUID, opts ContactListOpts) ([]model.Contact, int, error) {
	base := `FROM crm_contacts WHERE workspace_id = $1 AND deleted_at IS NULL`
	args := []interface{}{workspaceID}
	n := 2
	if opts.Search != "" {
		base += fmt.Sprintf(" AND (first_name ILIKE $%d OR last_name ILIKE $%d OR middle_name ILIKE $%d)", n, n, n)
		args = append(args, "%"+opts.Search+"%")
		n++
	}
	if opts.CompanyID != "" {
		base += fmt.Sprintf(" AND company_id = $%d", n)
		args = append(args, opts.CompanyID)
		n++
	}
	if opts.OwnerID != "" {
		base += fmt.Sprintf(" AND owner_id = $%d", n)
		args = append(args, opts.OwnerID)
		n++
	}

	var total int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*)"+base, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	order := "ORDER BY created_at DESC"
	if opts.SortBy == "firstName" {
		order = "ORDER BY first_name " + sortOrder(opts.SortOrder) + ", last_name " + sortOrder(opts.SortOrder)
	} else if opts.SortBy == "lastName" {
		order = "ORDER BY last_name " + sortOrder(opts.SortOrder) + ", first_name " + sortOrder(opts.SortOrder)
	} else if opts.SortBy == "createdAt" {
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

	rows, err := r.db.QueryContext(ctx, `SELECT id, workspace_id, first_name, last_name, middle_name, company_id, position, birthday, tags, owner_id, created_by, updated_by, created_at, updated_at, custom_fields `+base, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var list []model.Contact
	for rows.Next() {
		c, err := scanContact(rows)
		if err != nil {
			return nil, 0, err
		}
		c.Phones, c.Emails, err = r.contactPhonesEmails(ctx, c.ID)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, *c)
	}
	return list, total, rows.Err()
}

func sortOrder(o string) string {
	if o == "asc" {
		return "ASC"
	}
	return "DESC"
}

type ContactListOpts struct {
	Page      int
	Limit     int
	Search    string
	CompanyID string
	OwnerID   string
	SortBy    string
	SortOrder string
}

func scanContactRow(row interface {
	Scan(dest ...interface{}) error
}) (*model.Contact, error) {
	var id, wsID, firstName, lastName, ownerID, createdBy, updatedBy string
	var middleName, companyID, position sql.NullString
	var birthday sql.NullTime
	var tags pq.StringArray
	var createdAt, updatedAt time.Time
	var customFields []byte
	err := row.Scan(&id, &wsID, &firstName, &lastName, &middleName, &companyID, &position, &birthday, &tags, &ownerID, &createdBy, &updatedBy, &createdAt, &updatedAt, &customFields)
	if err != nil {
		return nil, err
	}
	c := &model.Contact{
		ID:        id,
		FirstName: firstName,
		LastName:  lastName,
		OwnerID:   ownerID,
		CreatedBy: createdBy,
		UpdatedBy: updatedBy,
		CreatedAt: createdAt.Format(time.RFC3339),
		UpdatedAt: updatedAt.Format(time.RFC3339),
		Tags:      tags,
	}
	if middleName.Valid {
		c.MiddleName = middleName.String
	}
	if companyID.Valid {
		c.CompanyID = companyID.String
	}
	if position.Valid {
		c.Position = position.String
	}
	if birthday.Valid {
		c.Birthday = birthday.Time.Format("2006-01-02")
	}
	if len(customFields) > 0 {
		_ = json.Unmarshal(customFields, &c.CustomFields)
	}
	return c, nil
}

func scanContact(rows *sql.Rows) (*model.Contact, error) {
	return scanContactRow(rows)
}

func (r *Repository) contactPhonesEmails(ctx context.Context, contactID string) ([]model.ContactPhone, []model.ContactEmail, error) {
	cid, _ := uuid.Parse(contactID)
	var phones []model.ContactPhone
	rows, err := r.db.QueryContext(ctx, `SELECT type, number, is_primary FROM crm_contact_phones WHERE contact_id = $1`, cid)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var p model.ContactPhone
		if err := rows.Scan(&p.Type, &p.Number, &p.IsPrimary); err != nil {
			rows.Close()
			return nil, nil, err
		}
		phones = append(phones, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	var emails []model.ContactEmail
	rows, err = r.db.QueryContext(ctx, `SELECT type, address, is_primary FROM crm_contact_emails WHERE contact_id = $1`, cid)
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var e model.ContactEmail
		if err := rows.Scan(&e.Type, &e.Address, &e.IsPrimary); err != nil {
			rows.Close()
			return nil, nil, err
		}
		emails = append(emails, e)
	}
	rows.Close()
	return phones, emails, rows.Err()
}

func (r *Repository) ContactGet(ctx context.Context, id, workspaceID uuid.UUID) (*model.Contact, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, workspace_id, first_name, last_name, middle_name, company_id, position, birthday, tags, owner_id, created_by, updated_by, created_at, updated_at, custom_fields FROM crm_contacts WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL`, id, workspaceID)
	c, err := scanContactRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	c.Phones, c.Emails, err = r.contactPhonesEmails(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (r *Repository) ContactCreate(ctx context.Context, workspaceID string, c *model.Contact, userID string) error {
	id := uuid.New()
	wsID, _ := uuid.Parse(workspaceID)
	ownerID, _ := uuid.Parse(userID)
	if c.OwnerID != "" {
		ownerID, _ = uuid.Parse(c.OwnerID)
	}
	createdBy, _ := uuid.Parse(userID)
	updatedBy := createdBy
	now := time.Now()
	customJSON, _ := json.Marshal(c.CustomFields)
	var companyID interface{}
	if c.CompanyID != "" {
		companyID, _ = uuid.Parse(c.CompanyID)
	}
	var middleName, position, birthday interface{}
	if c.MiddleName != "" {
		middleName = c.MiddleName
	}
	if c.Position != "" {
		position = c.Position
	}
	if c.Birthday != "" {
		birthday = c.Birthday
	}

	_, err := r.db.ExecContext(ctx, `INSERT INTO crm_contacts (id, workspace_id, first_name, last_name, middle_name, company_id, position, birthday, tags, owner_id, created_by, updated_by, created_at, updated_at, custom_fields) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$13,$14)`,
		id, wsID, c.FirstName, c.LastName, middleName, companyID, position, birthday, pq.Array(c.Tags), ownerID, createdBy, updatedBy, now, customJSON)
	if err != nil {
		return fmt.Errorf("insert contact: %w", err)
	}
	c.ID = id.String()
	c.CreatedAt = now.Format(time.RFC3339)
	c.UpdatedAt = c.CreatedAt
	c.CreatedBy = userID
	c.UpdatedBy = userID
	if c.OwnerID == "" {
		c.OwnerID = userID
	}

	for _, p := range c.Phones {
		_, err = r.db.ExecContext(ctx, `INSERT INTO crm_contact_phones (id, contact_id, type, number, is_primary) VALUES (gen_random_uuid(), $1, $2, $3, $4)`, id, p.Type, p.Number, p.IsPrimary)
		if err != nil {
			return err
		}
	}
	for _, e := range c.Emails {
		_, err = r.db.ExecContext(ctx, `INSERT INTO crm_contact_emails (id, contact_id, type, address, is_primary) VALUES (gen_random_uuid(), $1, $2, $3, $4)`, id, e.Type, e.Address, e.IsPrimary)
		if err != nil {
			return err
		}
	}
	if c.CompanyID != "" {
		companyUUID, _ := uuid.Parse(c.CompanyID)
		_, _ = r.db.ExecContext(ctx, `INSERT INTO crm_company_contacts (company_id, contact_id, created_at) VALUES ($1, $2, NOW()) ON CONFLICT (company_id, contact_id) DO NOTHING`, companyUUID, id)
	}
	return nil
}

func (r *Repository) ContactUpdate(ctx context.Context, workspaceID string, c *model.Contact, userID string) error {
	updatedBy, _ := uuid.Parse(userID)
	var companyID, middleName, position, birthday interface{}
	if c.CompanyID != "" {
		companyID, _ = uuid.Parse(c.CompanyID)
	}
	if c.MiddleName != "" {
		middleName = c.MiddleName
	}
	if c.Position != "" {
		position = c.Position
	}
	if c.Birthday != "" {
		birthday = c.Birthday
	}
	id, _ := uuid.Parse(c.ID)
	wsID, _ := uuid.Parse(workspaceID)
	res, err := r.db.ExecContext(ctx, `UPDATE crm_contacts SET first_name=$3, last_name=$4, middle_name=$5, company_id=$6, position=$7, birthday=$8, tags=$9, owner_id=$10, updated_by=$11, updated_at=NOW() WHERE id=$1 AND workspace_id=$2`,
		id, wsID, c.FirstName, c.LastName, middleName, companyID, position, birthday, pq.Array(c.Tags), c.OwnerID, updatedBy)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	// Update company link: remove from any company, then add to new if set
	_, _ = r.db.ExecContext(ctx, `DELETE FROM crm_company_contacts WHERE contact_id = $1`, id)
	if c.CompanyID != "" {
		companyUUID, _ := uuid.Parse(c.CompanyID)
		_, _ = r.db.ExecContext(ctx, `INSERT INTO crm_company_contacts (company_id, contact_id, created_at) VALUES ($1, $2, NOW()) ON CONFLICT (company_id, contact_id) DO NOTHING`, companyUUID, id)
	}
	// Update phones/emails: delete and re-insert
	_, _ = r.db.ExecContext(ctx, `DELETE FROM crm_contact_phones WHERE contact_id = $1`, id)
	_, _ = r.db.ExecContext(ctx, `DELETE FROM crm_contact_emails WHERE contact_id = $1`, id)
	for _, p := range c.Phones {
		_, _ = r.db.ExecContext(ctx, `INSERT INTO crm_contact_phones (id, contact_id, type, number, is_primary) VALUES (gen_random_uuid(), $1, $2, $3, $4)`, id, p.Type, p.Number, p.IsPrimary)
	}
	for _, e := range c.Emails {
		_, _ = r.db.ExecContext(ctx, `INSERT INTO crm_contact_emails (id, contact_id, type, address, is_primary) VALUES (gen_random_uuid(), $1, $2, $3, $4)`, id, e.Type, e.Address, e.IsPrimary)
	}
	return nil
}

func (r *Repository) ContactDelete(ctx context.Context, id, workspaceID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `UPDATE crm_contacts SET deleted_at = NOW() WHERE id = $1 AND workspace_id = $2`, id, workspaceID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) ContactCountDeals(ctx context.Context, contactID, workspaceID uuid.UUID) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM crm_deals WHERE contact_id = $1 AND workspace_id = $2 AND deleted_at IS NULL`, contactID, workspaceID).Scan(&n)
	return n, err
}
