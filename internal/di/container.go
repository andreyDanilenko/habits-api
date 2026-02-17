package di

import (
	"backend/internal/config"
	adminHandler "backend/internal/handler/admin"
	authHandler "backend/internal/handler/auth"
	habitsHandler "backend/internal/handler/habits"
	journalHandler "backend/internal/handler/journal"
	loggerHandler "backend/internal/handler/logger"
	masterHandler "backend/internal/handler/master"
	notesHandler "backend/internal/handler/notes"
	swaggerHandler "backend/internal/handler/swagger"
	workspaceHandler "backend/internal/handler/workspace"
	"backend/internal/middleware"
	habitsRepo "backend/internal/repository/habits"
	journalRepo "backend/internal/repository/journal"
	licenseRepo "backend/internal/repository/license"
	loggerRepo "backend/internal/repository/logger"
	masterRepo "backend/internal/repository/master"
	notesRepo "backend/internal/repository/notes"
	userRepo "backend/internal/repository/user"
	userPrefsRepo "backend/internal/repository/user_preferences"
	workspaceRepo "backend/internal/repository/workspace"
	"backend/internal/router"
	authService "backend/internal/service/auth"
	habitsService "backend/internal/service/habits"
	journalService "backend/internal/service/journal"
	loggerService "backend/internal/service/logger"
	masterService "backend/internal/service/master"
	notesService "backend/internal/service/notes"
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
	Cfg              *config.Config
	Router           *router.Router
	AuthHandler      *authHandler.Handler
	AdminHandler     *adminHandler.Handler
	WorkspaceHandler *workspaceHandler.Handler
	WorkspaceService *workspaceService.Service
	MasterHandler    *masterHandler.Handler
	NotesHandler     *notesHandler.Handler
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
	userPrefsRepository := userPrefsRepo.NewRepository(db)
	licenseRepository := licenseRepo.NewRepository(db)
	workspaceSvc := workspaceService.NewService(workspaceRepository, userPrefsRepository, licenseRepository)

	// Auth
	userRepository := userRepo.NewRepository(db)
	tokenGen := token.NewGenerator(cfg.Auth.JWTSecretKey, cfg.Auth.JWTExpiration)
	authSvc := authService.NewService(userRepository, workspaceSvc, tokenGen, cfg.Auth.JWTExpiration)

	cookieManager := cookies.NewManagerFromEnv()
	authHdlr := authHandler.NewHandler(authSvc, cookieManager, responder, validate)

	// Workspace handler
	workspaceHdlr := workspaceHandler.NewHandler(workspaceSvc, responder, validate)

	// Master data (Shared Schema: currencies, counterparties)
	masterRepository := masterRepo.NewRepository(db)
	masterSvc := masterService.NewService(masterRepository)
	masterHdlr := masterHandler.NewHandler(masterSvc, workspaceSvc, responder, validate)

	// Notes module
	notesRepository := notesRepo.NewRepository(db)
	notesSvc := notesService.NewService(notesRepository)
	notesHdlr := notesHandler.NewHandler(notesSvc, workspaceSvc, responder, validate)

	// Habits
	habitsRepository := habitsRepo.NewRepository(db)
	habitsSvc := habitsService.NewService(habitsRepository)
	habitsHdlr := habitsHandler.NewHandler(habitsSvc, responder, validate)

	// Journal
	journalRepository := journalRepo.NewRepository(db)
	journalSvc := journalService.NewService(journalRepository)
	journalHdlr := journalHandler.NewHandler(journalSvc, workspaceSvc, responder, validate)

	// Logger
	loggerHdlr := loggerHandler.NewHandler(logService, responder, validate)

	// Admin (использует workspace service и user repo)
	adminHdlr := adminHandler.NewHandler(workspaceSvc, userRepository, responder)

	return &Container{
		Cfg:              cfg,
		Router:           r,
		AuthHandler:      authHdlr,
		AdminHandler:     adminHdlr,
		WorkspaceHandler: workspaceHdlr,
		WorkspaceService: workspaceSvc,
		MasterHandler:    masterHdlr,
		NotesHandler:     notesHdlr,
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

	swaggerHandler.Register(r.Handler(), c.Cfg.Server.ExposeSwagger)

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

	// Workspace routes (and nested: master data, notes)
	workspaceGroup := protected.Group("/workspaces")
	c.WorkspaceHandler.RegisterRoutes(workspaceGroup)
	wsIDGroup := workspaceGroup.Group("/:workspaceId")
	c.MasterHandler.RegisterRoutes(wsIDGroup)
	c.NotesHandler.RegisterRoutes(wsIDGroup)
	c.HabitsHandler.RegisterRoutes(wsIDGroup)
	c.JournalHandler.RegisterRoutes(wsIDGroup)

	adminGroup := protected.Group("/admin")
	adminGroup.Use(middleware.RequireAdmin(c.Responder))
	c.AdminHandler.RegisterRoutes(adminGroup)

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
