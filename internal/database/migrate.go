package database

import (
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"backend/internal/config"
)

func RunMigrations(db config.DatabaseConfig) error {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		db.User,
		db.Password,
		db.Host,
		db.Port,
		db.DBName,
	)

	// // Для разработки: сбросим миграции
	// conn, err := sql.Open("postgres", dsn)
	// if err != nil {
	// 	return err
	// }
	// defer conn.Close()

	// // Удаляем таблицу миграций чтобы начать заново
	// _, err = conn.Exec("DROP TABLE IF EXISTS schema_migrations")
	// if err != nil {
	// 	log.Printf("Warning: could not drop migrations table: %v", err)
	// }

	m, err := migrate.New(
		"file://./migrations",
		dsn,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("Database migrations applied successfully")
	return nil
}

// RunMigrationsWithDSN применяет миграции с готовой DSN строкой
func RunMigrationsWithDSN(dsn string) error {
	m, err := migrate.New(
		"file://./migrations",
		dsn,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("Database migrations applied successfully")
	return nil
}
