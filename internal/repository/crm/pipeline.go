package crm

import (
	"context"
	"database/sql"
	"time"

	"backend/internal/model"

	"github.com/google/uuid"
)

func (r *Repository) PipelineList(ctx context.Context, workspaceID uuid.UUID) ([]model.Pipeline, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id::text, workspace_id::text, name, is_default, created_by::text, created_at FROM crm_pipelines WHERE workspace_id = $1 ORDER BY is_default DESC, name`, workspaceID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]model.Pipeline, 0)
	for rows.Next() {
		var id, ws, createdBy string
		var name string
		var isDefault bool
		var createdAt time.Time
		if err := rows.Scan(&id, &ws, &name, &isDefault, &createdBy, &createdAt); err != nil {
			return nil, err
		}
		p := model.Pipeline{ID: id, Name: name, IsDefault: isDefault}
		p.Stages, err = r.stagesByPipeline(ctx, id)
		if err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *Repository) stagesByPipeline(ctx context.Context, pipelineID string) ([]model.Stage, error) {
	pid, _ := uuid.Parse(pipelineID)
	rows, err := r.db.QueryContext(ctx, `SELECT id::text, name, order_index, color, probability, is_final, is_lost FROM crm_stages WHERE pipeline_id = $1 ORDER BY order_index`, pid.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []model.Stage
	for rows.Next() {
		var s model.Stage
		var id string
		var color sql.NullString
		if err := rows.Scan(&id, &s.Name, &s.Order, &color, &s.Probability, &s.IsFinal, &s.IsLost); err != nil {
			return nil, err
		}
		s.ID = id
		if color.Valid {
			s.Color = color.String
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

func (r *Repository) PipelineGetByID(ctx context.Context, pipelineID, workspaceID uuid.UUID) (*model.Pipeline, error) {
	var p model.Pipeline
	var createdAt time.Time
	var id string
	err := r.db.QueryRowContext(ctx, `SELECT id::text, name, is_default, created_at FROM crm_pipelines WHERE id = $1 AND workspace_id = $2`, pipelineID.String(), workspaceID.String()).Scan(&id, &p.Name, &p.IsDefault, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	p.ID = id
	p.Stages, err = r.stagesByPipeline(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repository) StageGet(ctx context.Context, stageID uuid.UUID) (*model.Stage, error) {
	var s model.Stage
	var id string
	var color sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT id::text, name, order_index, color, probability, is_final, is_lost FROM crm_stages WHERE id = $1`, stageID.String()).Scan(&id, &s.Name, &s.Order, &color, &s.Probability, &s.IsFinal, &s.IsLost)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	s.ID = id
	if color.Valid {
		s.Color = color.String
	}
	return &s, nil
}

func (r *Repository) PipelineCreate(ctx context.Context, workspaceID string, p *model.Pipeline, userID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	pipeID := uuid.New()
	wsID, _ := uuid.Parse(workspaceID)
	createdBy, _ := uuid.Parse(userID)
	_, err = tx.ExecContext(ctx, `INSERT INTO crm_pipelines (id, workspace_id, name, is_default, created_by, created_at) VALUES ($1,$2,$3,$4,$5,NOW())`, pipeID.String(), wsID.String(), p.Name, p.IsDefault, createdBy.String())
	if err != nil {
		return err
	}
	for i, s := range p.Stages {
		stageID := uuid.New()
		_, err = tx.ExecContext(ctx, `INSERT INTO crm_stages (id, pipeline_id, name, order_index, color, probability, is_final, is_lost, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())`,
			stageID.String(), pipeID.String(), s.Name, i+1, nullStr(s.Color), s.Probability, s.IsFinal, s.IsLost)
		if err != nil {
			return err
		}
	}
	p.ID = pipeID.String()
	if err := tx.Commit(); err != nil {
		return err
	}
	// Re-fetch pipeline with stage IDs for response
	list, err := r.PipelineList(ctx, wsID)
	if err != nil {
		return err
	}
	for i := range list {
		if list[i].ID == p.ID {
			p.Stages = list[i].Stages
			break
		}
	}
	return nil
}
