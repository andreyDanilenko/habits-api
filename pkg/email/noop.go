package email

import (
	"context"
	"log"
)

// NoopSender — заглушка для режима без отправки email (например, dev).
// Логирует письмо вместо отправки.
type NoopSender struct{}

func NewNoopSender() *NoopSender {
	return &NoopSender{}
}

func (s *NoopSender) Send(ctx context.Context, msg *Message) error {
	log.Printf("[EMAIL] Would send to %s: %s\nBody: %s", msg.To, msg.Subject, msg.Body)
	return nil
}
