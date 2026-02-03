package app

import (
	"backend/internal/config"
	"backend/internal/database"
	"backend/internal/di"
	"backend/internal/router"
	"backend/internal/worker"
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"
)

type App struct {
	cfg          *config.Config
	router       *router.Router
	server       *http.Server
	logProcessor *worker.LogProcessor
	logService   *di.Container
	db           *sql.DB
}

func New(cfg *config.Config) (*App, error) {

	db, err := database.InitDB(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	if err := database.RunMigrations(cfg.Database); err != nil {
		return nil, fmt.Errorf("Failed to run migrations: %w", err)
	}

	container := di.NewContainer(db, cfg)
	r := container.Router
	container.RegisterRoutes(r)

	// Создаем и запускаем воркер для обработки логов
	logProcessor := worker.NewLogProcessor(container.LogService)
	logProcessor.Start(context.Background())

	return &App{
		cfg: cfg,

		router:       r,
		logProcessor: logProcessor,
		logService:   container,
		db:           db,
	}, nil
}

func (a *App) Run() error {
	addr := fmt.Sprintf("%s:%s", a.cfg.Server.Host, a.cfg.Server.Port)

	a.server = &http.Server{
		Addr:         addr,
		Handler:      a.router.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	fmt.Printf("Server starting on %s\n", addr)
	return a.server.ListenAndServe()
}

// https://habr.com/ru/articles/908344/
func (a *App) Shutdown(ctx context.Context) error {
	if a.logProcessor != nil {
		a.logProcessor.Stop()
	}

	if a.db != nil {
		a.db.Close()
	}

	if a.server != nil {
		return a.server.Shutdown(ctx)
	}
	return nil
}
