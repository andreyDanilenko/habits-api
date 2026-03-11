package email

import (
	"context"
	"fmt"
	"net/smtp"
)

// SMTPConfig — настройки SMTP.
type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
}

// SMTPSender — реализация Sender через SMTP.
type SMTPSender struct {
	cfg SMTPConfig
}

// NewSMTPSender создаёт SMTP-отправитель.
func NewSMTPSender(cfg SMTPConfig) *SMTPSender {
	return &SMTPSender{cfg: cfg}
}

// Send отправляет письмо через SMTP.
func (s *SMTPSender) Send(ctx context.Context, msg *Message) error {
	if msg.To == "" || msg.Subject == "" {
		return fmt.Errorf("email: to and subject are required")
	}

	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	headers := "MIME-Version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n"
	fullMsg := []byte("Subject: " + msg.Subject + "\r\n" + headers + "\r\n" + msg.Body)

	addr := fmt.Sprintf("%s:%s", s.cfg.Host, s.cfg.Port)
	return smtp.SendMail(addr, auth, s.cfg.Username, []string{msg.To}, fullMsg)
}
