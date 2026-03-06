package di

import (
	"backend/internal/authz"
	"backend/internal/config"
	adminHandler "backend/internal/handler/admin"
	authHandler "backend/internal/handler/auth"
	crmHandler "backend/internal/handler/crm"
	habitsHandler "backend/internal/handler/habits"
	journalHandler "backend/internal/handler/journal"
	loggerHandler "backend/internal/handler/logger"
	masterHandler "backend/internal/handler/master"
	notesHandler "backend/internal/handler/notes"
	permissionHandler "backend/internal/handler/permission"
	projectHandler "backend/internal/handler/project"
	swaggerHandler "backend/internal/handler/swagger"
	workspaceHandler "backend/internal/handler/workspace"
	"backend/internal/middleware"
	crmRepo "backend/internal/repository/crm"
	habitsRepo "backend/internal/repository/habits"
	journalRepo "backend/internal/repository/journal"
	licenseRepo "backend/internal/repository/license"
	loggerRepo "backend/internal/repository/logger"
	masterRepo "backend/internal/repository/master"
	notesRepo "backend/internal/repository/notes"
	permissionRepo "backend/internal/repository/permission"
	projectRepo "backend/internal/repository/project"
	userRepo "backend/internal/repository/user"
	userPrefsRepo "backend/internal/repository/user_preferences"
	workspaceRepo "backend/internal/repository/workspace"
	"backend/internal/router"
	authService "backend/internal/service/auth"
	crmService "backend/internal/service/crm"
	habitsService "backend/internal/service/habits"
	journalService "backend/internal/service/journal"
	loggerService "backend/internal/service/logger"
	masterService "backend/internal/service/master"
	notesService "backend/internal/service/notes"
	permissionService "backend/internal/service/permission"
	projectService "backend/internal/service/project"
	workspaceService "backend/internal/service/workspace"
	"backend/pkg/auth/token"
	"backend/pkg/http/cookies"
	"backend/pkg/response"
	"database/sql"
	"net/http"
	"time"

	casbin "github.com/casbin/casbin/v3"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

type Container struct {
	Cfg              *config.Config
	Router           *router.Router
	AuthHandler      *authHandler.Handler
	AdminHandler     *adminHandler.Handler
	WorkspaceHandler *workspaceHandler.Handler
	WorkspaceService *workspaceService.Service
	MasterHandler    *masterHandler.Handler
	CrmHandler       *crmHandler.Handler
	NotesHandler     *notesHandler.Handler
	ProjectHandler   *projectHandler.Handler
	HabitsHandler    *habitsHandler.Handler
	JournalHandler   *journalHandler.Handler
	LoggerHandler      *loggerHandler.Handler
	LogService         *loggerService.Service
	Enforcer           *casbin.Enforcer
	PermissionService  *permissionService.Service
	PermissionHandler  *permissionHandler.Handler
	TokenGen           *token.Generator
	Responder        *response.Responder
	Validate         *validator.Validate
}

func NewContainer(db *sql.DB, gormDB *gorm.DB, cfg *config.Config) (*Container, error) {
	responder := response.NewResponder()
	validate := validator.New()
	r := router.New(responder)

	// Logger
	loggerRepository := loggerRepo.NewRepository(db)
	logService := loggerService.NewService(loggerRepository, cfg.Logs.Dir)

	workspaceRepository := workspaceRepo.NewRepository(db)
	userPrefsRepository := userPrefsRepo.NewRepository(db)
	licenseRepository := licenseRepo.NewRepository(db)

	// Permission (нужен для workspace Create — назначение OWNER при создании)
	permissionRepository := permissionRepo.NewRepository(db)
	enforcer, err := authz.InitEnforcer(gormDB)
	if err != nil {
		return nil, err
	}
	permSvc := permissionService.NewService(permissionRepository, enforcer)
	workspaceSvc := workspaceService.NewService(workspaceRepository, userPrefsRepository, licenseRepository, permSvc)

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

	crmRepository := crmRepo.NewRepository(db)
	crmSvc := crmService.NewService(crmRepository, workspaceSvc, userRepository)
	crmHdlr := crmHandler.NewHandler(crmSvc, workspaceSvc, responder, validate)

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

	projectRepository := projectRepo.NewRepository(db)
	projectSvc := projectService.NewService(projectRepository, workspaceSvc)
	projectHdlr := projectHandler.NewHandler(projectSvc, responder, validate)

	// Logger
	loggerHdlr := loggerHandler.NewHandler(logService, responder, validate)

	// Admin (использует workspace service и user repo)
	adminHdlr := adminHandler.NewHandler(workspaceSvc, userRepository, responder)

	// Permission handler (permSvc и enforcer уже созданы выше)
	permissionHdlr := permissionHandler.NewHandler(permSvc, responder, validate)

	return &Container{
		Cfg:              cfg,
		Router:           r,
		AuthHandler:      authHdlr,
		AdminHandler:     adminHdlr,
		WorkspaceHandler: workspaceHdlr,
		WorkspaceService: workspaceSvc,
		MasterHandler:    masterHdlr,
		CrmHandler:       crmHdlr,
		NotesHandler:     notesHdlr,
		ProjectHandler:   projectHdlr,
		HabitsHandler:    habitsHdlr,
		JournalHandler:   journalHdlr,
		LoggerHandler:     loggerHdlr,
		LogService:        logService,
		Enforcer:          enforcer,
		PermissionService: permSvc,
		PermissionHandler: permissionHdlr,
		TokenGen:          tokenGen,
		Responder:        responder,
		Validate:         validate,
	}, nil
}

func (c *Container) RegisterRoutes(r *router.Router) {
	r.Handler().Use(middleware.CORSMiddleware())
	r.Handler().Use(middleware.RequestLogger(c.LogService))

	swaggerHandler.Register(r.Handler(), c.Cfg.Server.ExposeSwagger, c.Cfg.Server.SwaggerUser, c.Cfg.Server.SwaggerPassword)

	// Health check
	r.GET("/api/v1/health", HealthCheck)
	apiV1 := r.Group("/api/v1")

	// Public auth routes (login, register, logout, refresh)
	// Rate limit: 10 попыток в минуту на IP — защита от брутфорса
	authRateLimiter := middleware.NewAuthRateLimiter(10, time.Minute)
	authGroup := apiV1.Group("/auth")
	authGroup.Use(authRateLimiter.Middleware(c.Responder))
	c.AuthHandler.RegisterPublicRoutes(authGroup)

	// Protected routes
	protected := apiV1.Group("")
	protected.Use(middleware.GinAuthMiddleware(c.TokenGen, c.Responder))
	protected.Use(middleware.WorkspacePathMiddleware(c.WorkspaceService, c.Responder))
	protected.Use(middleware.ModuleLicenseMiddleware(c.WorkspaceService, c.Responder))
	protected.Use(middleware.PermissionMiddleware(c.Enforcer, c.PermissionService, c.Responder))

	// Protected auth routes (me)
	protectedAuthGroup := protected.Group("/auth")
	c.AuthHandler.RegisterProtectedRoutes(protectedAuthGroup)

	// Me (текущий пользователь): права в workspace
	meGroup := protected.Group("/me")
	meGroup.GET("/permissions", c.PermissionHandler.GetMyPermissions)

	// Workspace routes (and nested: master data, notes, permissions)
	workspaceGroup := protected.Group("/workspaces")
	c.WorkspaceHandler.RegisterRoutes(workspaceGroup)
	wsIDGroup := workspaceGroup.Group("/:workspaceId")
	c.MasterHandler.RegisterRoutes(wsIDGroup)
	c.ProjectHandler.RegisterRoutes(wsIDGroup)
	c.CrmHandler.RegisterRoutes(wsIDGroup)
	c.NotesHandler.RegisterRoutes(wsIDGroup)
	c.HabitsHandler.RegisterRoutes(wsIDGroup)
	c.JournalHandler.RegisterRoutes(wsIDGroup)
	c.PermissionHandler.RegisterRoutes(wsIDGroup)

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
		"service": "backend check ci",
	})
}
