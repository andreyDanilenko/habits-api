package di

import (
	"backend/internal/authz"
	"backend/internal/config"
	adminHandler "backend/internal/handler/admin"
	authHandler "backend/internal/handler/auth"
	crmHandler "backend/internal/handler/crm"
	habitsHandler "backend/internal/handler/habits"
	invitationHandler "backend/internal/handler/invitation"
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
	invitationRepo "backend/internal/repository/invitation"
	journalRepo "backend/internal/repository/journal"
	licenseRepo "backend/internal/repository/license"
	loggerRepo "backend/internal/repository/logger"
	masterRepo "backend/internal/repository/master"
	notesRepo "backend/internal/repository/notes"
	permissionRepo "backend/internal/repository/permission"
	projectRepo "backend/internal/repository/project"
	regTokenRepo "backend/internal/repository/registration_token"
	userRepo "backend/internal/repository/user"
	userPrefsRepo "backend/internal/repository/user_preferences"
	workspaceRepo "backend/internal/repository/workspace"
	"backend/internal/router"
	authService "backend/internal/service/auth"
	crmService "backend/internal/service/crm"
	habitsService "backend/internal/service/habits"
	invitationService "backend/internal/service/invitation"
	journalService "backend/internal/service/journal"
	loggerService "backend/internal/service/logger"
	masterService "backend/internal/service/master"
	notesService "backend/internal/service/notes"
	permissionService "backend/internal/service/permission"
	projectService "backend/internal/service/project"
	workspaceService "backend/internal/service/workspace"
	"backend/pkg/auth/token"
	"backend/pkg/email"
	"backend/pkg/http/cookies"
	"backend/pkg/password"
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
	InvitationHandler  *invitationHandler.Handler
	TokenGen           *token.Generator
	UserRepository     userRepo.UserRepository
	Responder        *response.Responder
	Validate         *validator.Validate
}

func NewContainer(db *sql.DB, gormDB *gorm.DB, cfg *config.Config) (*Container, error) {
	responder := response.NewResponder()
	validate := validator.New()
	_ = validate.RegisterValidation("password_format", func(fl validator.FieldLevel) bool {
		return password.ValidateFormat(fl.Field().String())
	})
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
	permSvc := permissionService.NewService(permissionRepository, enforcer, workspaceRepository)
	workspaceSvc := workspaceService.NewService(workspaceRepository, userPrefsRepository, licenseRepository, permSvc)

	// Auth
	userRepository := userRepo.NewRepository(db)
	regTokenRepository := regTokenRepo.NewRepository(db)
	tokenGen := token.NewGenerator(cfg.Auth.JWTSecretKey, cfg.Auth.JWTExpiration)

	var emailSender email.Sender
	if cfg.Email.Enabled && cfg.Email.SMTPHost != "" {
		emailSender = email.NewSMTPSender(email.SMTPConfig{
			Host:     cfg.Email.SMTPHost,
			Port:     cfg.Email.SMTPPort,
			Username: cfg.Email.SMTPUsername,
			Password: cfg.Email.SMTPPassword,
		})
	} else {
		emailSender = email.NewNoopSender()
	}

	// Invitation (нужен для auth — AcceptAfterRegistration при регистрации по invite)
	invitationRepository := invitationRepo.NewRepository(db)
	invitationSvc := invitationService.NewService(invitationRepository, workspaceRepository, userRepository, permSvc, emailSender, cfg)

	authSvc := authService.NewService(
		userRepository,
		regTokenRepository,
		workspaceSvc,
		invitationSvc,
		tokenGen,
		emailSender,
		cfg.Auth.JWTExpiration,
		cfg.Auth.RefreshExpiration,
		cfg.Auth.RegistrationTokenLifetime,
		cfg.Auth.VerificationBaseURL,
	)

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
	habitsSvc := habitsService.NewService(habitsRepository, workspaceRepository)
	habitsHdlr := habitsHandler.NewHandler(habitsSvc, responder, validate)

	// Journal
	journalRepository := journalRepo.NewRepository(db)
	journalSvc := journalService.NewService(journalRepository, habitsRepository)
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

	invitationHdlr := invitationHandler.NewHandler(invitationSvc, responder)

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
		InvitationHandler: invitationHdlr,
		TokenGen:          tokenGen,
		UserRepository:    userRepository,
		Responder:         responder,
		Validate:          validate,
	}, nil
}

func (c *Container) RegisterRoutes(r *router.Router) {
	r.Handler().Use(middleware.CORSMiddleware())
	r.Handler().Use(middleware.RequestLogger(c.LogService))

	swaggerHandler.Register(r.Handler(), c.Cfg.Server.ExposeSwagger, c.Cfg.Server.SwaggerUser, c.Cfg.Server.SwaggerPassword)

	// Health check
	r.GET("/api/v1/health", HealthCheck)
	apiV1 := r.Group("/api/v1")

	// Public auth routes с per-route rate limit:
	// login: 5/min per (IP, email) — защита от брутфорса по конкретному email
	// register: 10/min per IP
	// refresh, logout: 30/min per IP
	authGroup := apiV1.Group("/auth")
	authRateLimitConfig := &middleware.AuthRateLimitConfig{
		LoginLimiter:    middleware.NewAuthRateLimiter(5, time.Minute),
		RegisterLimiter: middleware.NewAuthRateLimiter(10, time.Minute),
		RefreshLimiter:  middleware.NewAuthRateLimiter(30, time.Minute),
		LogoutLimiter:   middleware.NewAuthRateLimiter(30, time.Minute),
	}
	c.AuthHandler.RegisterPublicRoutesWithRateLimit(authGroup, authRateLimitConfig)

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
	invitationsGroup := wsIDGroup.Group("/invitations")
	c.InvitationHandler.RegisterRoutes(invitationsGroup)

	// Public invitation routes (optional auth)
	publicGroup := apiV1.Group("/public")
	publicGroup.Use(middleware.OptionalGinAuthMiddleware(c.TokenGen, c.UserRepository))
	publicInvitationsGroup := publicGroup.Group("/invitations")
	c.InvitationHandler.RegisterPublicRoutes(publicInvitationsGroup)

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
