package chat

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

func (r *Repository) ListThreads(ctx context.Context, workspaceID, userID string, limit int) ([]model.ChatThread, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			t.id,
			t.workspace_id,
			t.type,
			t.title,
			t.created_by,
			t.created_at,
			t.updated_at,
			(
				SELECT left(m.body, 120)
				FROM chat_messages m
				WHERE m.thread_id = t.id
				ORDER BY m.created_at DESC
				LIMIT 1
			) AS last_message_preview
		FROM chat_threads t
		INNER JOIN chat_thread_participants p
			ON p.thread_id = t.id AND p.user_id = $2
		WHERE t.workspace_id = $1
		ORDER BY t.updated_at DESC
		LIMIT $3
	`, uuid.MustParse(workspaceID), uuid.MustParse(userID), limit)
	if err != nil {
		return nil, fmt.Errorf("query threads: %w", err)
	}
	defer rows.Close()

	var out []model.ChatThread
	for rows.Next() {
		var t model.ChatThread
		var tid, wid, createdBy uuid.UUID
		var typ string
		var title sql.NullString
		var last sql.NullString
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&tid, &wid, &typ, &title, &createdBy, &createdAt, &updatedAt, &last); err != nil {
			return nil, fmt.Errorf("scan thread: %w", err)
		}
		t.ID = tid.String()
		t.WorkspaceID = wid.String()
		t.Type = model.ChatThreadType(typ)
		if title.Valid {
			t.Title = &title.String
		}
		t.CreatedBy = createdBy.String()
		t.CreatedAt = createdAt
		t.UpdatedAt = updatedAt
		if last.Valid {
			t.LastMessagePreview = &last.String
		}
		out = append(out, t)
	}
	return out, nil
}

func (r *Repository) GetOrCreatePrivateThread(ctx context.Context, workspaceID, userA, userB, createdBy string) (*model.ChatThread, bool, error) {
	ws := uuid.MustParse(workspaceID)
	a := uuid.MustParse(userA)
	b := uuid.MustParse(userB)
	creator := uuid.MustParse(createdBy)

	low, high := a, b
	if high.String() < low.String() {
		low, high = high, low
	}

	// Fast path: existing
	var existingThreadID uuid.UUID
	err := r.db.QueryRowContext(ctx, `
		SELECT thread_id
		FROM chat_private_threads
		WHERE workspace_id = $1 AND user_low_id = $2 AND user_high_id = $3
	`, ws, low, high).Scan(&existingThreadID)
	if err == nil {
		t, err := r.getThreadByIDForUser(ctx, existingThreadID, ws, uuid.MustParse(createdBy))
		if err != nil {
			return nil, false, err
		}
		return t, false, nil
	}
	if err != sql.ErrNoRows {
		return nil, false, fmt.Errorf("select private thread: %w", err)
	}

	// Create transactionally
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	threadID := uuid.New()
	now := time.Now().UTC()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO chat_threads (id, workspace_id, type, title, created_by, created_at, updated_at)
		VALUES ($1, $2, 'private', NULL, $3, $4, $4)
	`, threadID, ws, creator, now); err != nil {
		return nil, false, fmt.Errorf("insert chat_threads: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO chat_private_threads (thread_id, workspace_id, user_low_id, user_high_id)
		VALUES ($1, $2, $3, $4)
	`, threadID, ws, low, high); err != nil {
		return nil, false, fmt.Errorf("insert chat_private_threads: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO chat_thread_participants (thread_id, user_id)
		VALUES ($1, $2), ($1, $3)
		ON CONFLICT DO NOTHING
	`, threadID, a, b); err != nil {
		return nil, false, fmt.Errorf("insert participants: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, false, fmt.Errorf("commit: %w", err)
	}

	t := &model.ChatThread{
		ID:          threadID.String(),
		WorkspaceID: workspaceID,
		Type:        model.ChatThreadPrivate,
		CreatedBy:   createdBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	return t, true, nil
}

func (r *Repository) ListMessages(ctx context.Context, workspaceID, threadID, userID string, limit int, before *time.Time) ([]model.ChatMessage, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	// ensure membership
	ok, err := r.isParticipant(ctx, uuid.MustParse(threadID), uuid.MustParse(userID))
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, sql.ErrNoRows
	}

	var rows *sql.Rows
	if before != nil {
		rows, err = r.db.QueryContext(ctx, `
			SELECT id, thread_id, workspace_id, author_id, body, created_at
			FROM chat_messages
			WHERE workspace_id = $1 AND thread_id = $2 AND created_at < $3
			ORDER BY created_at DESC
			LIMIT $4
		`, uuid.MustParse(workspaceID), uuid.MustParse(threadID), *before, limit)
	} else {
		rows, err = r.db.QueryContext(ctx, `
			SELECT id, thread_id, workspace_id, author_id, body, created_at
			FROM chat_messages
			WHERE workspace_id = $1 AND thread_id = $2
			ORDER BY created_at DESC
			LIMIT $3
		`, uuid.MustParse(workspaceID), uuid.MustParse(threadID), limit)
	}
	if err != nil {
		return nil, fmt.Errorf("query messages: %w", err)
	}
	defer rows.Close()

	var out []model.ChatMessage
	for rows.Next() {
		var m model.ChatMessage
		var mid, tid, wid, aid uuid.UUID
		var createdAt time.Time
		if err := rows.Scan(&mid, &tid, &wid, &aid, &m.Body, &createdAt); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		m.ID = mid.String()
		m.ThreadID = tid.String()
		m.WorkspaceID = wid.String()
		m.AuthorID = aid.String()
		m.CreatedAt = createdAt
		out = append(out, m)
	}
	// reverse to ascending
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

func (r *Repository) CreateMessage(ctx context.Context, workspaceID, threadID, authorID, body string) (*model.ChatMessage, error) {
	// ensure membership
	ok, err := r.isParticipant(ctx, uuid.MustParse(threadID), uuid.MustParse(authorID))
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, sql.ErrNoRows
	}

	id := uuid.New()
	now := time.Now().UTC()
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO chat_messages (id, thread_id, workspace_id, author_id, body, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, id, uuid.MustParse(threadID), uuid.MustParse(workspaceID), uuid.MustParse(authorID), body, now)
	if err != nil {
		return nil, fmt.Errorf("insert message: %w", err)
	}
	_, _ = r.db.ExecContext(ctx, `
		UPDATE chat_threads SET updated_at = $2 WHERE id = $1
	`, uuid.MustParse(threadID), now)

	return &model.ChatMessage{
		ID:          id.String(),
		ThreadID:    threadID,
		WorkspaceID: workspaceID,
		AuthorID:    authorID,
		Body:        body,
		CreatedAt:   now,
	}, nil
}

func (r *Repository) isParticipant(ctx context.Context, threadID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM chat_thread_participants WHERE thread_id = $1 AND user_id = $2
		)
	`, threadID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("isParticipant: %w", err)
	}
	return exists, nil
}

func (r *Repository) getThreadByIDForUser(ctx context.Context, threadID, workspaceID, userID uuid.UUID) (*model.ChatThread, error) {
	var t model.ChatThread
	var tid, wid, createdBy uuid.UUID
	var typ string
	var title sql.NullString
	var createdAt, updatedAt time.Time
	err := r.db.QueryRowContext(ctx, `
		SELECT t.id, t.workspace_id, t.type, t.title, t.created_by, t.created_at, t.updated_at
		FROM chat_threads t
		INNER JOIN chat_thread_participants p ON p.thread_id = t.id AND p.user_id = $3
		WHERE t.id = $1 AND t.workspace_id = $2
	`, threadID, workspaceID, userID).Scan(&tid, &wid, &typ, &title, &createdBy, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	t.ID = tid.String()
	t.WorkspaceID = wid.String()
	t.Type = model.ChatThreadType(typ)
	if title.Valid {
		t.Title = &title.String
	}
	t.CreatedBy = createdBy.String()
	t.CreatedAt = createdAt
	t.UpdatedAt = updatedAt
	return &t, nil
}

