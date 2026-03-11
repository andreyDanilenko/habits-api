package registration_token

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"backend/internal/model"
)

type Repository interface {
	Create(ctx context.Context, rt *model.RegistrationToken) error
	FindByToken(ctx context.Context, token string) (*model.RegistrationToken, error)
	DeleteByToken(ctx context.Context, token string) error
	DeleteByEmail(ctx context.Context, email string) error
	DeleteExpired(ctx context.Context) (int64, error)
}

type PostgresRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(ctx context.Context, rt *model.RegistrationToken) error {
	query := `
		INSERT INTO registration_tokens (id, email, password_hash, name, token, invite_token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, query,
		rt.ID,
		rt.Email,
		rt.PasswordHash,
		rt.Name,
		rt.Token,
		rt.InviteToken,
		rt.ExpiresAt,
		rt.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) FindByToken(ctx context.Context, token string) (*model.RegistrationToken, error) {
	query := `
		SELECT id, email, password_hash, name, token, COALESCE(invite_token, ''), expires_at, created_at
		FROM registration_tokens
		WHERE token = $1
	`
	var rt model.RegistrationToken
	var name sql.NullString
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&rt.ID,
		&rt.Email,
		&rt.PasswordHash,
		&name,
		&rt.Token,
		&rt.InviteToken,
		&rt.ExpiresAt,
		&rt.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find by token: %w", err)
	}
	if name.Valid {
		rt.Name = &name.String
	}
	return &rt, nil
}

func (r *PostgresRepository) DeleteByToken(ctx context.Context, token string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM registration_tokens WHERE token = $1", token)
	return err
}

func (r *PostgresRepository) DeleteByEmail(ctx context.Context, email string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM registration_tokens WHERE email = $1", email)
	return err
}

func (r *PostgresRepository) DeleteExpired(ctx context.Context) (int64, error) {
	result, err := r.db.ExecContext(ctx, "DELETE FROM registration_tokens WHERE expires_at < NOW()")
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
