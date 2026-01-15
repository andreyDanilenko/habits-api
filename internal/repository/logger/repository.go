package logger

import (
	"backend/internal/model"
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CreateTable создает таблицу для логов
func (r *Repository) CreateTable(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS request_logs (
		id SERIAL PRIMARY KEY,
		timestamp TIMESTAMP NOT NULL,
		status_code INTEGER NOT NULL,
		duration_ms DECIMAL(10, 6) NOT NULL,
		client_ip VARCHAR(45) NOT NULL,
		method VARCHAR(10) NOT NULL,
		path TEXT NOT NULL,
		raw_log TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE INDEX IF NOT EXISTS idx_request_logs_timestamp ON request_logs(timestamp);
	`
	_, err := r.db.ExecContext(ctx, query)
	return err
}

// BatchInsert вставляет множество записей в БД
func (r *Repository) BatchInsert(entries []*model.LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	ctx := context.Background()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO request_logs (timestamp, status_code, duration_ms, client_ip, method, path, raw_log)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, entry := range entries {
		durationMs := float64(entry.Duration.Nanoseconds()) / 1000000.0
		_, err := stmt.ExecContext(ctx,
			entry.Timestamp,
			entry.StatusCode,
			durationMs,
			entry.ClientIP,
			entry.Method,
			entry.Path,
			entry.RawLog,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetLogsByDate возвращает логи за определенную дату
func (r *Repository) GetLogsByDate(ctx context.Context, date time.Time) ([]*model.LogEntry, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	rows, err := r.db.QueryContext(ctx, `
		SELECT timestamp, status_code, duration_ms, client_ip, method, path, raw_log
		FROM request_logs
		WHERE timestamp >= $1 AND timestamp < $2
		ORDER BY timestamp ASC
	`, startOfDay, endOfDay)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fmt.Println("GetLogsByDate", rows)

	entries := make([]*model.LogEntry, 0)
	for rows.Next() {
		var entry model.LogEntry
		var durationMs float64
		err := rows.Scan(
			&entry.Timestamp,
			&entry.StatusCode,
			&durationMs,
			&entry.ClientIP,
			&entry.Method,
			&entry.Path,
			&entry.RawLog,
		)
		if err != nil {
			return nil, err
		}
		entry.Duration = time.Duration(durationMs * 1000000)
		entries = append(entries, &entry)
	}

	return entries, rows.Err()
}
