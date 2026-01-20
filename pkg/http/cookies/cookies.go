package cookies

import (
	"net/http"
	"os"
	"time"
)

type Manager struct {
	secure   bool
	sameSite http.SameSite
}

func NewManager(secure bool, sameSite http.SameSite) *Manager {
	return &Manager{
		secure:   secure,
		sameSite: sameSite,
	}
}

// SameSite - защита от CSRF атак:
//   - Lax: куки отправляются при переходе по ссылкам, но не при AJAX с другого сайта (для localhost)
//   - Strict: куки только с того же сайта (самая строгая защита)
//   - None: куки всегда отправляются, но требует Secure=true (HTTPS) - для кросс-доменных запросов
func NewManagerFromEnv() *Manager {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	var secure bool
	var sameSite http.SameSite

	if env == "production" {
		// Production: HTTPS + Secure + SameSite=None для кросс-доменных запросов
		secure = true
		sameSite = http.SameSiteNoneMode
	} else {
		// Development (localhost): HTTP + SameSite=Lax
		// Lax работает для localhost и разрешает cookies при навигации
		secure = false
		sameSite = http.SameSiteLaxMode
	}

	return NewManager(secure, sameSite)
}

// SetToken устанавливает cookie с токеном
func (m *Manager) SetToken(w http.ResponseWriter, name, token string, expires time.Time) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    token,
		Expires:  expires,
		HttpOnly: true, // Защита от XSS - JavaScript не может получить доступ
		Secure:   m.secure,
		SameSite: m.sameSite,
		Path:     "/",
	}

	http.SetCookie(w, cookie)
}

func (m *Manager) GetToken(r *http.Request, name string) (string, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

// Delete удаляет cookie, устанавливая пустое значение и прошедшую дату
func (m *Manager) Delete(w http.ResponseWriter, name string) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    "",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   m.secure,
		SameSite: m.sameSite,
		Path:     "/",
	}
	http.SetCookie(w, cookie)
}
