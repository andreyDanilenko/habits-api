package email

import "context"

// Message — универсальное письмо для отправки.
// Логика формирования (subject, body, шаблоны) — в вызывающем коде.
type Message struct {
	To      string
	Subject string
	Body    string // HTML body
}

// Sender — интерфейс отправки email. Реализация (SMTP и т.д.) отделена от бизнес-логики.
type Sender interface {
	Send(ctx context.Context, msg *Message) error
}
