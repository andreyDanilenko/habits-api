package crm

import (
	"context"
	"database/sql"
	"fmt"
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

func (r *Repository) PipelineUpdate(ctx context.Context, workspaceID, pipelineID uuid.UUID, p *model.Pipeline) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// if pipeline is set as default, unset default flag on other pipelines in the same workspace
	if p.IsDefault {
		if _, err := tx.ExecContext(ctx, `UPDATE crm_pipelines SET is_default = false WHERE workspace_id = $1 AND id <> $2`, workspaceID.String(), pipelineID.String()); err != nil {
			return err
		}
	}

	// Update pipeline meta
	res, err := tx.ExecContext(ctx, `UPDATE crm_pipelines SET name = $3, is_default = $4 WHERE id = $1 AND workspace_id = $2`,
		pipelineID.String(), workspaceID.String(), p.Name, p.IsDefault)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}

	// If stages not provided, do not touch them
	if p.Stages != nil {
		// Load existing stages for this pipeline
		rows, err := tx.QueryContext(ctx, `SELECT id::text FROM crm_stages WHERE pipeline_id = $1`, pipelineID.String())
		if err != nil {
			return err
		}
		defer rows.Close()

		existingIDs := make(map[string]struct{})
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return err
			}
			existingIDs[id] = struct{}{}
		}
		if err := rows.Err(); err != nil {
			return err
		}

		// Upsert incoming stages
		incomingIDs := make(map[string]struct{}, len(p.Stages))
		for i, st := range p.Stages {
			order := st.Order
			if order <= 0 {
				order = i + 1
			}
			if st.ID == "" {
				// New stage
				stageID := uuid.New()
				if _, err := tx.ExecContext(ctx, `INSERT INTO crm_stages (id, pipeline_id, name, order_index, color, probability, is_final, is_lost, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())`,
					stageID.String(), pipelineID.String(), st.Name, order, nullStr(st.Color), st.Probability, st.IsFinal, st.IsLost); err != nil {
					return err
				}
			} else {
				// Existing stage
				incomingIDs[st.ID] = struct{}{}
				if _, err := tx.ExecContext(ctx, `UPDATE crm_stages SET name=$2, order_index=$3, color=$4, probability=$5, is_final=$6, is_lost=$7 WHERE id=$1 AND pipeline_id=$8`,
					st.ID, st.Name, order, nullStr(st.Color), st.Probability, st.IsFinal, st.IsLost, pipelineID.String()); err != nil {
					return err
				}
			}
		}

		// Delete stages that are not present in incoming list
		for id := range existingIDs {
			if _, ok := incomingIDs[id]; !ok {
				if _, err := tx.ExecContext(ctx, `DELETE FROM crm_stages WHERE id = $1 AND pipeline_id = $2`, id, pipelineID.String()); err != nil {
					return err
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (r *Repository) PipelineDelete(ctx context.Context, workspaceID, pipelineID uuid.UUID) error {
	// ensure there are no ACTIVE deals in this pipeline
	var n int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM crm_deals WHERE pipeline_id = $1 AND workspace_id = $2 AND status = 'open' AND deleted_at IS NULL`, pipelineID.String(), workspaceID.String()).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return fmt.Errorf("pipeline has %d deals", n)
	}

	res, err := r.db.ExecContext(ctx, `DELETE FROM crm_pipelines WHERE id = $1 AND workspace_id = $2`, pipelineID.String(), workspaceID.String())
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) StageList(ctx context.Context, pipelineID uuid.UUID) ([]model.Stage, error) {
	return r.stagesByPipeline(ctx, pipelineID.String())
}

func (r *Repository) StageCreate(ctx context.Context, pipelineID uuid.UUID, s *model.Stage) error {
	// determine order index: either provided or append to the end
	order := s.Order
	if order <= 0 {
		if err := r.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(order_index), 0) + 1 FROM crm_stages WHERE pipeline_id = $1`, pipelineID.String()).Scan(&order); err != nil {
			return err
		}
	}
	stageID := uuid.New()
	_, err := r.db.ExecContext(ctx, `INSERT INTO crm_stages (id, pipeline_id, name, order_index, color, probability, is_final, is_lost, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())`,
		stageID.String(), pipelineID.String(), s.Name, order, nullStr(s.Color), s.Probability, s.IsFinal, s.IsLost)
	if err != nil {
		return err
	}
	s.ID = stageID.String()
	s.Order = order
	return nil
}

func (r *Repository) StageUpdate(ctx context.Context, pipelineID, stageID uuid.UUID, s *model.Stage) error {
	res, err := r.db.ExecContext(ctx, `UPDATE crm_stages SET name=$3, color=$4, probability=$5, is_final=$6, is_lost=$7 WHERE id=$1 AND pipeline_id=$2`,
		stageID.String(), pipelineID.String(), s.Name, nullStr(s.Color), s.Probability, s.IsFinal, s.IsLost)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) StageDelete(ctx context.Context, pipelineID, stageID uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM crm_stages WHERE id = $1 AND pipeline_id = $2`, stageID.String(), pipelineID.String())
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) StageReorder(ctx context.Context, pipelineID uuid.UUID, stageIDs []uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for idx, id := range stageIDs {
		if _, err := tx.ExecContext(ctx, `UPDATE crm_stages SET order_index = $3 WHERE id = $1 AND pipeline_id = $2`,
			id.String(), pipelineID.String(), idx+1); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
