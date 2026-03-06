package password

import "golang.org/x/crypto/bcrypt"

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
