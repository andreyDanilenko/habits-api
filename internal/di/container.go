package di

import (
	"backend/internal/config"
	authHandler "backend/internal/handler/auth"
	habitsHandler "backend/internal/handler/habits"
	journalHandler "backend/internal/handler/journal"
	loggerHandler "backend/internal/handler/logger"
	workspaceHandler "backend/internal/handler/workspace"
	"backend/internal/middleware"
	authRepo "backend/internal/repository/auth"
	habitsRepo "backend/internal/repository/habits"
	journalRepo "backend/internal/repository/journal"
	loggerRepo "backend/internal/repository/logger"
	workspaceRepo "backend/internal/repository/workspace"
	"backend/internal/router"
	authService "backend/internal/service/auth"
	habitsService "backend/internal/service/habits"
	journalService "backend/internal/service/journal"
	loggerService "backend/internal/service/logger"
	workspaceService "backend/internal/service/workspace"
	"context"
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Container struct {
	AuthHandler      *authHandler.Handler
	WorkspaceHandler *workspaceHandler.Handler
	HabitsHandler    *habitsHandler.Handler
	JournalHandler   *journalHandler.Handler
	LoggerHandler    *loggerHandler.Handler
	LogService       *loggerService.Service
}

func NewContainer(db *sql.DB, cfg *config.Config) *Container {
	// Logger
	loggerRepository := loggerRepo.NewRepository(db)
	logService := loggerService.NewService(loggerRepository, cfg.Logs.Dir)

	// Создаем таблицу для логов если её нет
	ctx := context.Background()
	if err := loggerRepository.CreateTable(ctx); err != nil {
		// Логируем ошибку, но не паникуем
		_ = err
	}

	// Auth
	authRepository := authRepo.NewRepository(db)
	authSvc := authService.NewService(authRepository)
	authHdlr := authHandler.NewHandler(authSvc)

	// Workspace
	workspaceRepository := workspaceRepo.NewRepository(db)
	workspaceSvc := workspaceService.NewService(workspaceRepository)
	workspaceHdlr := workspaceHandler.NewHandler(workspaceSvc)

	// Habits
	habitsRepository := habitsRepo.NewRepository(db)
	habitsSvc := habitsService.NewService(habitsRepository)
	habitsHdlr := habitsHandler.NewHandler(habitsSvc)

	// Journal
	journalRepository := journalRepo.NewRepository(db)
	journalSvc := journalService.NewService(journalRepository)
	journalHdlr := journalHandler.NewHandler(journalSvc)

	// Logger
	loggerHdlr := loggerHandler.NewHandler(logService)

	return &Container{
		AuthHandler:      authHdlr,
		WorkspaceHandler: workspaceHdlr,
		HabitsHandler:    habitsHdlr,
		JournalHandler:   journalHdlr,
		LoggerHandler:    loggerHdlr,
		LogService:       logService,
	}
}

func (c *Container) RegisterRoutes(r *router.Router) {
	// Применяем middleware для логирования всех запросов
	r.Handler().Use(middleware.RequestLogger(c.LogService))

	// Health check
	r.GET("/health", HealthCheck)
	apiV1 := r.Group("/api/v1")

	// Auth routes
	authGroup := apiV1.Group("/auth")
	c.AuthHandler.RegisterRoutes(authGroup)

	// Workspace routes
	workspaceGroup := apiV1.Group("/workspaces")
	c.WorkspaceHandler.RegisterRoutes(workspaceGroup)

	// Habits routes
	habitsGroup := apiV1.Group("/habits")
	c.HabitsHandler.RegisterRoutes(habitsGroup)

	// Journal routes
	journalGroup := apiV1.Group("/journal")
	c.JournalHandler.RegisterRoutes(journalGroup)

	// Logger routes
	loggerGroup := apiV1.Group("/logs")
	c.LoggerHandler.RegisterRoutes(loggerGroup)
}

func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "backend",
	})
}
