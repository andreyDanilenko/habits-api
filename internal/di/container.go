package di

import (
	"backend/internal/config"
	authHandler "backend/internal/handler/auth"
	habitsHandler "backend/internal/handler/habits"
	journalHandler "backend/internal/handler/journal"
	loggerHandler "backend/internal/handler/logger"
	workspaceHandler "backend/internal/handler/workspace"
	"backend/internal/middleware"
	habitsRepo "backend/internal/repository/habits"
	journalRepo "backend/internal/repository/journal"
	loggerRepo "backend/internal/repository/logger"
	userRepo "backend/internal/repository/user"
	workspaceRepo "backend/internal/repository/workspace"
	"backend/internal/router"
	authService "backend/internal/service/auth"
	habitsService "backend/internal/service/habits"
	journalService "backend/internal/service/journal"
	loggerService "backend/internal/service/logger"
	workspaceService "backend/internal/service/workspace"
	"backend/pkg/auth/token"
	"backend/pkg/http/cookies"
	"backend/pkg/response"
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Container struct {
	Router           *router.Router
	AuthHandler      *authHandler.Handler
	WorkspaceHandler *workspaceHandler.Handler
	HabitsHandler    *habitsHandler.Handler
	JournalHandler   *journalHandler.Handler
	LoggerHandler    *loggerHandler.Handler
	LogService       *loggerService.Service
	TokenGen         *token.Generator
	Responder        *response.Responder
	Validate         *validator.Validate
}

func NewContainer(db *sql.DB, cfg *config.Config) *Container {
	responder := response.NewResponder()
	validate := validator.New()
	r := router.New(responder)

	// Logger
	loggerRepository := loggerRepo.NewRepository(db)
	logService := loggerService.NewService(loggerRepository, cfg.Logs.Dir)

	workspaceRepository := workspaceRepo.NewRepository(db)
	workspaceSvc := workspaceService.NewService(workspaceRepository)

	// Auth
	userRepository := userRepo.NewRepository(db)
	tokenGen := token.NewGenerator(cfg.Auth.JWTSecretKey, cfg.Auth.JWTExpiration)
	authSvc := authService.NewService(userRepository, workspaceSvc, tokenGen, cfg.Auth.JWTExpiration)

	cookieManager := cookies.NewManagerFromEnv()
	authHdlr := authHandler.NewHandler(authSvc, cookieManager, responder, validate)

	// Workspace handler
	workspaceHdlr := workspaceHandler.NewHandler(workspaceSvc, responder, validate)

	// Habits
	habitsRepository := habitsRepo.NewRepository(db)
	habitsSvc := habitsService.NewService(habitsRepository)
	habitsHdlr := habitsHandler.NewHandler(habitsSvc, responder, validate)

	// Journal
	journalRepository := journalRepo.NewRepository(db)
	journalSvc := journalService.NewService(journalRepository)
	journalHdlr := journalHandler.NewHandler(journalSvc, responder, validate)

	// Logger
	loggerHdlr := loggerHandler.NewHandler(logService, responder, validate)

	return &Container{
		Router:           r,
		AuthHandler:      authHdlr,
		WorkspaceHandler: workspaceHdlr,
		HabitsHandler:    habitsHdlr,
		JournalHandler:   journalHdlr,
		LoggerHandler:    loggerHdlr,
		LogService:       logService,
		TokenGen:         tokenGen,
		Responder:        responder,
		Validate:         validate,
	}
}

func (c *Container) RegisterRoutes(r *router.Router) {
	r.Handler().Use(middleware.CORSMiddleware())
	r.Handler().Use(middleware.RequestLogger(c.LogService))

	// Health check
	r.GET("/health", HealthCheck)
	apiV1 := r.Group("/api/v1")

	// Public auth routes (login, register, logout, refresh)
	authGroup := apiV1.Group("/auth")
	c.AuthHandler.RegisterPublicRoutes(authGroup)

	// Protected routes
	protected := apiV1.Group("")
	protected.Use(middleware.GinAuthMiddleware(c.TokenGen, c.Responder))

	// Protected auth routes (me)
	protectedAuthGroup := protected.Group("/auth")
	c.AuthHandler.RegisterProtectedRoutes(protectedAuthGroup)

	// Workspace routes
	workspaceGroup := protected.Group("/workspaces")
	c.WorkspaceHandler.RegisterRoutes(workspaceGroup)

	// Habits routes
	habitsGroup := protected.Group("/habits")
	c.HabitsHandler.RegisterRoutes(habitsGroup)

	// Journal routes
	journalGroup := protected.Group("/journal")
	c.JournalHandler.RegisterRoutes(journalGroup)

	// Logger routes
	loggerGroup := protected.Group("/logs")
	c.LoggerHandler.RegisterRoutes(loggerGroup)
}

func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "backend",
	})
}
