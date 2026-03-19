// migrate_force сбрасывает версию миграций (для перехода на новый baseline).
// Запуск: go run ./cmd/migrate_force
//
// После force 0 запустите API — применятся миграции 000001–000027.
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
	if err := m.Force(0); err != nil {
		log.Fatalf("Force version 0: %v", err)
	}
	log.Println("Migration version set to 0. Run the app to apply 000001–000027.")
}
