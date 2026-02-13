package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"backend/internal/model"
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByEmailAnyStatus(ctx context.Context, email string) (*model.User, error)
	FindByID(ctx context.Context, id string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
}
type PostgresUserRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (
			id, email, password, name, role, 
			avatar_url, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.Password,
		user.Name,
		user.Role,
		user.AvatarURL,
		user.Status,
		user.CreatedAt,
		user.UpdatedAt,
	)
	return err
}

func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT 
			id, email, password, name, role, 
			avatar_url, status, created_at, updated_at
		FROM users 
		WHERE email = $1 AND status = 'ACTIVE'
	`

	var user model.User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&user.Name,
		&user.Role,
		&user.AvatarURL,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find by email failed: %w", err)
	}

	return &user, nil
}

// FindByEmailAnyStatus возвращает пользователя по email без фильтра по status (в т.ч. DELETED — для повторной регистрации).
func (r *PostgresUserRepository) FindByEmailAnyStatus(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, email, password, name, role, avatar_url, status, created_at, updated_at
		FROM users WHERE email = $1
	`
	var user model.User
	var name, avatarURL sql.NullString
	var status sql.NullString
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&name,
		&user.Role,
		&avatarURL,
		&status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find by email any status: %w", err)
	}
	if name.Valid {
		user.Name = &name.String
	}
	if avatarURL.Valid {
		user.AvatarURL = &avatarURL.String
	}
	if status.Valid {
		s := model.UserStatus(status.String)
		user.Status = &s
	}
	return &user, nil
}

func (r *PostgresUserRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	query := `
		SELECT 
			id, email, password, name, role, 
			avatar_url, status, created_at, updated_at
		FROM users 
		WHERE id = $1 AND status = 'ACTIVE'
	`

	var user model.User
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&user.Name,
		&user.Role,
		&user.AvatarURL,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find by id failed: %w", err)
	}

	return &user, nil
}

func (r *PostgresUserRepository) Update(ctx context.Context, user *model.User) error {
	query := `
		UPDATE users SET
			email = $2,
			password = COALESCE($3, password),
			name = $4,
			role = $5,
			avatar_url = $6,
			status = $7,
			updated_at = $8
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.Password,
		user.Name,
		user.Role,
		user.AvatarURL,
		user.Status,
		user.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected failed: %w", err)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *PostgresUserRepository) ListAll(ctx context.Context) ([]model.User, error) {
	query := `
		SELECT id, email, name, role, avatar_url, status, created_at, updated_at
		FROM users
		WHERE status = 'ACTIVE'
		ORDER BY email
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list all users: %w", err)
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		var name, avatarURL sql.NullString
		var status sql.NullString
		err := rows.Scan(
			&u.ID,
			&u.Email,
			&name,
			&u.Role,
			&avatarURL,
			&status,
			&u.CreatedAt,
			&u.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		if name.Valid {
			u.Name = &name.String
		}
		if avatarURL.Valid {
			u.AvatarURL = &avatarURL.String
		}
		if status.Valid {
			s := model.UserStatus(status.String)
			u.Status = &s
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return users, nil
}

func (r *PostgresUserRepository) Delete(ctx context.Context, id string) error {
	query := `
		UPDATE users 
		SET status = 'DELETED', updated_at = NOW() 
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}
