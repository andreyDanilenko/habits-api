package password

import (
	"strings"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

// ValidateFormat проверяет пароль: мин 8 символов, буквы, цифры и спецсимволы (@$!%*#?&).
// Go regexp не поддерживает lookahead, поэтому проверка в коде.
func ValidateFormat(password string) bool {
	if len(password) < 8 {
		return false
	}
	hasLetter := false
	hasDigit := false
	hasSpecial := false
	special := "@$!%*#?&"
	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		case strings.ContainsRune(special, r):
			hasSpecial = true
		default:
			return false // недопустимый символ
		}
	}
	return hasLetter && hasDigit && hasSpecial
}

// dummyHash — bcrypt-хеш для constant-time сравнения при несуществующем пользователе.
// Предотвращает user enumeration по времени ответа (timing attack)
var dummyHash string

func init() {
	h, _ := bcrypt.GenerateFromPassword([]byte("dummy"), bcrypt.DefaultCost)
	dummyHash = string(h)
}

func Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func Check(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// DummyHash возвращает хеш для constant-time проверки, когда пользователь не найден.
func DummyHash() string {
	return dummyHash
}
