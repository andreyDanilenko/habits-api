package project

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"backend/internal/model"

	"github.com/google/uuid"
)

var ErrProjectNotFound = errors.New("project not found")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]model.Project, error) {
	query := `
		SELECT id, workspace_id, name, description, created_at, updated_at
		FROM projects
		WHERE workspace_id = $1
		ORDER BY updated_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	var list []model.Project
	for rows.Next() {
		var p model.Project
		var desc sql.NullString
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&p.ID, &p.WorkspaceID, &p.Name, &desc, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		p.CreatedAt = createdAt.Format(time.RFC3339)
		p.UpdatedAt = updatedAt.Format(time.RFC3339)
		if desc.Valid {
			p.Description = &desc.String
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (r *Repository) Get(ctx context.Context, projectID, workspaceID uuid.UUID) (*model.Project, error) {
	query := `
		SELECT id, workspace_id, name, description, created_at, updated_at
		FROM projects
		WHERE id = $1 AND workspace_id = $2
	`
	var p model.Project
	var desc sql.NullString
	var createdAt, updatedAt time.Time
	err := r.db.QueryRowContext(ctx, query, projectID, workspaceID).Scan(
		&p.ID, &p.WorkspaceID, &p.Name, &desc, &createdAt, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get project: %w", err)
	}
	p.CreatedAt = createdAt.Format(time.RFC3339)
	p.UpdatedAt = updatedAt.Format(time.RFC3339)
	if desc.Valid {
		p.Description = &desc.String
	}
	return &p, nil
}

func (r *Repository) Create(ctx context.Context, workspaceID uuid.UUID, dto model.CreateProjectDto) (*model.Project, error) {
	id := uuid.New()
	now := time.Now()
	query := `
		INSERT INTO projects (id, workspace_id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, workspace_id, name, description, created_at, updated_at
	`
	var p model.Project
	var desc sql.NullString
	var createdAt, updatedAt time.Time
	err := r.db.QueryRowContext(ctx, query,
		id, workspaceID, dto.Name, dto.Description, now, now,
	).Scan(&p.ID, &p.WorkspaceID, &p.Name, &desc, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}
	p.CreatedAt = createdAt.Format(time.RFC3339)
	p.UpdatedAt = updatedAt.Format(time.RFC3339)
	if desc.Valid {
		p.Description = &desc.String
	}
	return &p, nil
}

func (r *Repository) Update(ctx context.Context, projectID, workspaceID uuid.UUID, dto model.UpdateProjectDto) (*model.Project, error) {
	now := time.Now()
	query := `
		UPDATE projects SET
			name = COALESCE($1, name),
			description = COALESCE($2, description),
			updated_at = $3
		WHERE id = $4 AND workspace_id = $5
		RETURNING id, workspace_id, name, description, created_at, updated_at
	`
	var p model.Project
	var desc sql.NullString
	var createdAt, updatedAt time.Time
	err := r.db.QueryRowContext(ctx, query,
		dto.Name, dto.Description, now, projectID, workspaceID,
	).Scan(&p.ID, &p.WorkspaceID, &p.Name, &desc, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("update project: %w", err)
	}
	p.CreatedAt = createdAt.Format(time.RFC3339)
	p.UpdatedAt = updatedAt.Format(time.RFC3339)
	if desc.Valid {
		p.Description = &desc.String
	}
	return &p, nil
}

func (r *Repository) Delete(ctx context.Context, projectID, workspaceID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM projects WHERE id = $1 AND workspace_id = $2`,
		projectID, workspaceID,
	)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrProjectNotFound
	}
	return nil
}

// AttachEntity добавляет привязку сущности модуля к проекту.
func (r *Repository) AttachEntity(ctx context.Context, projectID uuid.UUID, entityType string, entityID uuid.UUID) error {
	query := `
		INSERT INTO project_entities (project_id, entity_type, entity_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (project_id, entity_type, entity_id) DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, query, projectID, entityType, entityID)
	if err != nil {
		return fmt.Errorf("attach entity: %w", err)
	}
	return nil
}

// DetachEntity удаляет привязку.
func (r *Repository) DetachEntity(ctx context.Context, projectID uuid.UUID, entityType string, entityID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM project_entities WHERE project_id = $1 AND entity_type = $2 AND entity_id = $3`,
		projectID, entityType, entityID,
	)
	if err != nil {
		return fmt.Errorf("detach entity: %w", err)
	}
	_, _ = res.RowsAffected()
	return nil
}

// ListEntityIDsByProject возвращает entity_id по project_id и опционально entity_type (пустая строка = все типы).
func (r *Repository) ListEntityIDsByProject(ctx context.Context, projectID uuid.UUID, entityType string) ([]uuid.UUID, error) {
	var rows *sql.Rows
	var err error
	if entityType == "" {
		rows, err = r.db.QueryContext(ctx,
			`SELECT entity_id FROM project_entities WHERE project_id = $1 ORDER BY created_at`,
			projectID,
		)
	} else {
		rows, err = r.db.QueryContext(ctx,
			`SELECT entity_id FROM project_entities WHERE project_id = $1 AND entity_type = $2 ORDER BY created_at`,
			projectID, entityType,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("list entity ids: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// GetProjectIDsForEntity возвращает список project_id в рамках workspace, к которым привязана сущность.
func (r *Repository) GetProjectIDsForEntity(ctx context.Context, workspaceID uuid.UUID, entityType string, entityID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT pe.project_id FROM project_entities pe
		 INNER JOIN projects p ON p.id = pe.project_id AND p.workspace_id = $1
		 WHERE pe.entity_type = $2 AND pe.entity_id = $3`,
		workspaceID, entityType, entityID,
	)
	if err != nil {
		return nil, fmt.Errorf("get projects for entity: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
