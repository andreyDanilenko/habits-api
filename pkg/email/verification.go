package email

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
)

//go:embed templates/email_verification.html
var emailVerificationTpl string

// VerificationEmailData — данные для письма подтверждения регистрации.
type VerificationEmailData struct {
	VerificationLink string
	ExpiresInHours   int
}

// BuildVerificationEmail формирует subject и body для письма подтверждения.
// Логика формирования письма отделена от отправки (Sender).
func BuildVerificationEmail(to, verificationLink string, expiresInHours int) *Message {
	data := VerificationEmailData{
		VerificationLink: verificationLink,
		ExpiresInHours:   expiresInHours,
	}

	body := buildVerificationBody(data)
	return &Message{
		To:      to,
		Subject: "Подтвердите регистрацию",
		Body:    body,
	}
}

func buildVerificationBody(data VerificationEmailData) string {
	t, err := template.New("verification").Parse(emailVerificationTpl)
	if err != nil {
		return fmt.Sprintf("<p>Подтвердите регистрацию: <a href=\"%s\">Скопировать ссылку</a></p>", data.VerificationLink)
	}
	var buf bytes.Buffer
	_ = t.Execute(&buf, data)
	return buf.String()
}
