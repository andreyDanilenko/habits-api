package activitylog

import (
	"context"

	"github.com/google/uuid"
)

// Event — нормализованная запись для ленты workspace (таблица activities).
type Event struct {
	UserID      uuid.UUID
	WorkspaceID uuid.UUID
	Type        string
	EntityType  string
	EntityID    uuid.UUID
	Title       string
	Emoji       string
}

// Writer пишет событие активности. Реализации: БД, Noop, цепочка (MultiWriter).
type Writer interface {
	Write(ctx context.Context, e Event) error
}

// NoopWriter отключает запись (для тестов, флага ACTIVITY_LOG_ENABLED=false).
type NoopWriter struct{}

func (NoopWriter) Write(context.Context, Event) error { return nil }

// MultiWriter вызывает всех подписчиков; ошибки игнорируются (как прежде с _ = repo.Create).
type MultiWriter []Writer

func (m MultiWriter) Write(ctx context.Context, e Event) error {
	for _, w := range m {
		if w == nil {
			continue
		}
		_ = w.Write(ctx, e)
	}
	return nil
}
