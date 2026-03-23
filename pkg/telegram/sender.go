package telegram

import "context"

// Sender — порт для отправки сообщений в Telegram (без привязки к конкретной реализации).
type Sender interface {
	SendMessage(ctx context.Context, text string) error
}

type NoopSender struct{}

func (NoopSender) SendMessage(ctx context.Context, text string) error { return nil }

func NewNoopSender() Sender { return NoopSender{} }

