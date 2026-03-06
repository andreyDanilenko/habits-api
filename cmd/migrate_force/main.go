// Откат версии миграций до 21 (после удаления 022-025).
// Запуск: go run ./cmd/migrate_force
package main

import (
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"backend/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Load config: %v", err)
	}
	db := cfg.Database
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		db.User, db.Password, db.Host, db.Port, db.DBName,
	)
	m, err := migrate.New("file://./migrations", dsn)
	if err != nil {
		log.Fatalf("Create migrate: %v", err)
	}
	defer m.Close()
	if err := m.Force(21); err != nil {
		log.Fatalf("Force version 21: %v", err)
	}
	log.Println("Migration version set to 21. Run the app to apply 000022.")
}
