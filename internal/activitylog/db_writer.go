package activitylog

import (
	"context"

	activityRepo "backend/internal/repository/activity"
)

// DBWriter пишет в таблицу activities через репозиторий.
type DBWriter struct {
	repo *activityRepo.Repository
}

func NewDBWriter(repo *activityRepo.Repository) *DBWriter {
	return &DBWriter{repo: repo}
}

func (w *DBWriter) Write(ctx context.Context, e Event) error {
	if w == nil || w.repo == nil {
		return nil
	}
	return w.repo.Create(ctx, e.UserID, e.WorkspaceID, e.Type, e.EntityType, e.EntityID, e.Title, e.Emoji)
}
