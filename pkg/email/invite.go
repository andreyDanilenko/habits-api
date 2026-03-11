package email

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"time"
)

//go:embed templates/email_invite.html
var emailInviteTpl string

// InviteEmailData — данные для письма приглашения.
type InviteEmailData struct {
	InviteLink    string
	WorkspaceName string
	InvitedByName string
	SystemRole    string
	ExpiresAt     string
}

// BuildInviteEmail формирует subject и body для письма приглашения.
func BuildInviteEmail(to, inviteLink, workspaceName, invitedByName, systemRole string, expiresAt time.Time) *Message {
	if systemRole == "" {
		systemRole = "MEMBER"
	}
	data := InviteEmailData{
		InviteLink:    inviteLink,
		WorkspaceName: workspaceName,
		InvitedByName: invitedByName,
		SystemRole:    systemRole,
		ExpiresAt:     expiresAt.Format("02.01.2006 15:04"),
	}

	body := buildInviteBody(data)
	return &Message{
		To:      to,
		Subject: fmt.Sprintf("Вас пригласили в workspace «%s»", workspaceName),
		Body:    body,
	}
}

func buildInviteBody(data InviteEmailData) string {
	t, err := template.New("invite").Parse(emailInviteTpl)
	if err != nil {
		return fmt.Sprintf("<p>%s приглашает вас в workspace «%s». <a href=\"%s\">Принять приглашение</a></p>",
			data.InvitedByName, data.WorkspaceName, data.InviteLink)
	}
	var buf bytes.Buffer
	_ = t.Execute(&buf, data)
	return buf.String()
}
