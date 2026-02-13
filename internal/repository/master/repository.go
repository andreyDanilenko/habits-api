package master

import (
	"context"
	"database/sql"
	"fmt"
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

// Currencies
func (r *Repository) ListCurrencies(ctx context.Context, workspaceID uuid.UUID) ([]model.Currency, error) {
	query := `SELECT id, workspace_id, code, name, symbol, created_at, updated_at FROM currencies WHERE workspace_id = $1 ORDER BY code`
	rows, err := r.db.QueryContext(ctx, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list currencies: %w", err)
	}
	defer rows.Close()
	var list []model.Currency
	for rows.Next() {
		var c model.Currency
		var createdAt, updatedAt time.Time
		var symbol sql.NullString
		if err := rows.Scan(&c.ID, &c.WorkspaceID, &c.Code, &c.Name, &symbol, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		c.CreatedAt = createdAt.Format(time.RFC3339)
		c.UpdatedAt = updatedAt.Format(time.RFC3339)
		if symbol.Valid {
			c.Symbol = &symbol.String
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

func (r *Repository) GetCurrency(ctx context.Context, id, workspaceID uuid.UUID) (*model.Currency, error) {
	query := `SELECT id, workspace_id, code, name, symbol, created_at, updated_at FROM currencies WHERE id = $1 AND workspace_id = $2`
	var c model.Currency
	var symbol sql.NullString
	var createdAt, updatedAt time.Time
	err := r.db.QueryRowContext(ctx, query, id, workspaceID).Scan(&c.ID, &c.WorkspaceID, &c.Code, &c.Name, &symbol, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	c.CreatedAt = createdAt.Format(time.RFC3339)
	c.UpdatedAt = updatedAt.Format(time.RFC3339)
	if symbol.Valid {
		c.Symbol = &symbol.String
	}
	return &c, nil
}

func (r *Repository) CreateCurrency(ctx context.Context, c *model.Currency) error {
	query := `INSERT INTO currencies (id, workspace_id, code, name, symbol, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id, created_at, updated_at`
	var symbol interface{}
	if c.Symbol != nil {
		symbol = *c.Symbol
	} else {
		symbol = nil
	}
	wsID, _ := uuid.Parse(c.WorkspaceID)
	id := uuid.New()
	var createdAt, updatedAt time.Time
	err := r.db.QueryRowContext(ctx, query, id, wsID, c.Code, c.Name, symbol).Scan(&c.ID, &createdAt, &updatedAt)
	if err != nil {
		return fmt.Errorf("create currency: %w", err)
	}
	c.CreatedAt = createdAt.Format(time.RFC3339)
	c.UpdatedAt = updatedAt.Format(time.RFC3339)
	return nil
}

func (r *Repository) UpdateCurrency(ctx context.Context, c *model.Currency) error {
	var symbol interface{}
	if c.Symbol != nil {
		symbol = *c.Symbol
	} else {
		symbol = nil
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE currencies SET code = $3, name = $4, symbol = $5, updated_at = NOW() WHERE id = $1 AND workspace_id = $2`,
		c.ID, c.WorkspaceID, c.Code, c.Name, symbol,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) DeleteCurrency(ctx context.Context, id, workspaceID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM currencies WHERE id = $1 AND workspace_id = $2`, id, workspaceID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// Counterparties
func (r *Repository) ListCounterparties(ctx context.Context, workspaceID uuid.UUID) ([]model.Counterparty, error) {
	query := `SELECT id, workspace_id, name, type, email, phone, comment, created_at, updated_at FROM counterparties WHERE workspace_id = $1 ORDER BY name`
	rows, err := r.db.QueryContext(ctx, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list counterparties: %w", err)
	}
	defer rows.Close()
	var list []model.Counterparty
	for rows.Next() {
		var cp model.Counterparty
		var email, phone, comment sql.NullString
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&cp.ID, &cp.WorkspaceID, &cp.Name, &cp.Type, &email, &phone, &comment, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		cp.CreatedAt = createdAt.Format(time.RFC3339)
		cp.UpdatedAt = updatedAt.Format(time.RFC3339)
		if email.Valid {
			cp.Email = &email.String
		}
		if phone.Valid {
			cp.Phone = &phone.String
		}
		if comment.Valid {
			cp.Comment = &comment.String
		}
		list = append(list, cp)
	}
	return list, rows.Err()
}

func (r *Repository) GetCounterparty(ctx context.Context, id, workspaceID uuid.UUID) (*model.Counterparty, error) {
	query := `SELECT id, workspace_id, name, type, email, phone, comment, created_at, updated_at FROM counterparties WHERE id = $1 AND workspace_id = $2`
	var cp model.Counterparty
	var email, phone, comment sql.NullString
	var createdAt, updatedAt time.Time
	err := r.db.QueryRowContext(ctx, query, id, workspaceID).Scan(&cp.ID, &cp.WorkspaceID, &cp.Name, &cp.Type, &email, &phone, &comment, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	cp.CreatedAt = createdAt.Format(time.RFC3339)
	cp.UpdatedAt = updatedAt.Format(time.RFC3339)
	if email.Valid {
		cp.Email = &email.String
	}
	if phone.Valid {
		cp.Phone = &phone.String
	}
	if comment.Valid {
		cp.Comment = &comment.String
	}
	return &cp, nil
}

func (r *Repository) CreateCounterparty(ctx context.Context, cp *model.Counterparty) error {
	query := `INSERT INTO counterparties (id, workspace_id, name, type, email, phone, comment, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING id, created_at, updated_at`
	wsID, _ := uuid.Parse(cp.WorkspaceID)
	id := uuid.New()
	var email, phone, comment interface{}
	if cp.Email != nil {
		email = *cp.Email
	} else {
		email = nil
	}
	if cp.Phone != nil {
		phone = *cp.Phone
	} else {
		phone = nil
	}
	if cp.Comment != nil {
		comment = *cp.Comment
	} else {
		comment = nil
	}
	var createdAt, updatedAt time.Time
	err := r.db.QueryRowContext(ctx, query, id, wsID, cp.Name, cp.Type, email, phone, comment).Scan(&cp.ID, &createdAt, &updatedAt)
	if err != nil {
		return fmt.Errorf("create counterparty: %w", err)
	}
	cp.CreatedAt = createdAt.Format(time.RFC3339)
	cp.UpdatedAt = updatedAt.Format(time.RFC3339)
	return nil
}

func (r *Repository) UpdateCounterparty(ctx context.Context, cp *model.Counterparty) error {
	var email, phone, comment interface{}
	if cp.Email != nil {
		email = *cp.Email
	} else {
		email = nil
	}
	if cp.Phone != nil {
		phone = *cp.Phone
	} else {
		phone = nil
	}
	if cp.Comment != nil {
		comment = *cp.Comment
	} else {
		comment = nil
	}
	res, err := r.db.ExecContext(ctx,
		`UPDATE counterparties SET name = $3, type = $4, email = $5, phone = $6, comment = $7, updated_at = NOW() WHERE id = $1 AND workspace_id = $2`,
		cp.ID, cp.WorkspaceID, cp.Name, cp.Type, email, phone, comment,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) DeleteCounterparty(ctx context.Context, id, workspaceID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM counterparties WHERE id = $1 AND workspace_id = $2`, id, workspaceID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
