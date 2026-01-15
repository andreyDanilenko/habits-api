package cookies

import (
	"net/http"
	"time"
)

type Manager struct {
	domain   string
	secure   bool
	sameSite http.SameSite
}

func NewManager(domain string, secure bool, sameSite http.SameSite) *Manager {
	return &Manager{
		domain:   domain,
		secure:   secure,
		sameSite: sameSite,
	}
}

func (m *Manager) SetToken(w http.ResponseWriter, name, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    token,
		Expires:  expires,
		HttpOnly: true,
		Secure:   m.secure,
		SameSite: m.sameSite,
		Path:     "/",
		Domain:   m.domain,
	})
}

func (m *Manager) GetToken(r *http.Request, name string) (string, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

func (m *Manager) Delete(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Expires:  time.Now().Add(-24 * time.Hour),
		HttpOnly: true,
		Secure:   m.secure,
		SameSite: m.sameSite,
		Path:     "/",
		Domain:   m.domain,
	})
}
