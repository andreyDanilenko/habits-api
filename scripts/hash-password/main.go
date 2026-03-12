// hash-password — генерирует bcrypt-хеш для пароля (для ручного UPDATE в БД).
// Использование: go run ./scripts/hash-password [пароль]
// Если пароль не передан — читает из stdin.
package main

import (
	"fmt"
	"os"

	"backend/pkg/password"
)

func main() {
	var pwd string
	if len(os.Args) > 1 {
		pwd = os.Args[1]
	} else {
		fmt.Fprint(os.Stderr, "Введите пароль: ")
		var b [256]byte
		n, _ := os.Stdin.Read(b[:])
		pwd = string(b[:n])
		for len(pwd) > 0 && (pwd[len(pwd)-1] == '\n' || pwd[len(pwd)-1] == '\r') {
			pwd = pwd[:len(pwd)-1]
		}
	}
	if pwd == "" {
		fmt.Fprintln(os.Stderr, "Пароль не может быть пустым")
		os.Exit(1)
	}
	if !password.ValidateFormat(pwd) {
		fmt.Fprintln(os.Stderr, "Пароль: мин 8 символов, буквы, цифры, спецсимволы (@$!%*#?&)")
		os.Exit(1)
	}
	hash, err := password.Hash(pwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(hash)
}
