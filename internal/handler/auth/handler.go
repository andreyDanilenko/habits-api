package auth

import (
	"fmt"
	"strings"
	"time"

	"backend/internal/middleware"
	"backend/internal/model"
	authService "backend/internal/service/auth"
	"backend/pkg/http/cookies"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	h.RegisterPublicRoutes(r)
	h.RegisterProtectedRoutes(r)
}

func (h *Handler) RegisterPublicRoutes(r *gin.RouterGroup) {
	h.registerPublicRoutesWithLimiters(r, nil)
}

// RegisterPublicRoutesWithRateLimit регистрирует публичные маршруты с per-route rate limit.
func (h *Handler) RegisterPublicRoutesWithRateLimit(r *gin.RouterGroup, cfg *middleware.AuthRateLimitConfig) {
	h.registerPublicRoutesWithLimiters(r, cfg)
}

func (h *Handler) registerPublicRoutesWithLimiters(r *gin.RouterGroup, cfg *middleware.AuthRateLimitConfig) {
	if cfg == nil {
		r.POST(RouteLogin, docLogin(h))
		r.POST(RouteRegister, docRegister(h))
		r.GET(RouteVerifyEmail, docVerifyEmail(h))
		r.POST(RouteLogout, docLogout(h))
		r.POST(RouteRefresh, docRefresh(h))
		return
	}
	post := func(route string, limiter *middleware.AuthRateLimiter, extractor middleware.KeyExtractor, handler gin.HandlerFunc) {
		if limiter != nil {
			r.POST(route, limiter.Middleware(h.responder, extractor), handler)
		} else {
			r.POST(route, handler)
		}
	}
	get := func(route string, limiter *middleware.AuthRateLimiter, extractor middleware.KeyExtractor, handler gin.HandlerFunc) {
		if limiter != nil {
			r.GET(route, limiter.Middleware(h.responder, extractor), handler)
		} else {
			r.GET(route, handler)
		}
	}
	post(RouteLogin, cfg.LoginLimiter, middleware.LoginKeyExtractor, docLogin(h))
	post(RouteRegister, cfg.RegisterLimiter, middleware.IPKeyExtractor, docRegister(h))
	get(RouteVerifyEmail, cfg.RegisterLimiter, middleware.IPKeyExtractor, docVerifyEmail(h))
	post(RouteLogout, cfg.LogoutLimiter, middleware.IPKeyExtractor, docLogout(h))
	post(RouteRefresh, cfg.RefreshLimiter, middleware.IPKeyExtractor, docRefresh(h))
}

func (h *Handler) RegisterProtectedRoutes(r *gin.RouterGroup) {
	r.GET(RouteMe, docMe(h))
	r.POST(RouteChangePassword, docChangePassword(h))
}

type Handler struct {
	service       *authService.AuthService
	cookieManager *cookies.Manager
	validate      *validator.Validate
	responder     *response.Responder
}

func NewHandler(
	service *authService.AuthService,
	cookieManager *cookies.Manager,
	responder *response.Responder,
	validate *validator.Validate,
) *Handler {
	return &Handler{
		service:       service,
		cookieManager: cookieManager,
		validate:      validate,
		responder:     responder,
	}
}

func (h *Handler) Login(c *gin.Context) {
	var req model.LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		h.responder.BadRequest(c, err.Error())
		return
	}

	loginResp, err := h.service.Login(c.Request.Context(), req)

	if err != nil {
		switch err {
		case authService.ErrInvalidCredentials:
			h.responder.Unauthorized(c, "Invalid email or password")
		default:
			h.responder.InternalServerError(c, "Internal server error")
		}
		return
	}
	accessExpiresAt := time.Now().Add(time.Duration(loginResp.ExpiresIn) * time.Second)
	h.cookieManager.SetToken(c.Writer, "access_token", loginResp.AccessToken, accessExpiresAt)
	refreshExpiresAt := time.Now().Add(30 * 24 * time.Hour) // 30 дней для refresh
	h.cookieManager.SetToken(c.Writer, "refresh_token", loginResp.RefreshToken, refreshExpiresAt)

	h.responder.SuccessWithData(c, gin.H{
		"user":       loginResp.User,
		"expires_in": loginResp.ExpiresIn,
	})
}

func (h *Handler) Register(c *gin.Context) {
	var req model.RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request format")
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.Password = strings.TrimSpace(req.Password)
	req.Name = strings.TrimSpace(req.Name)

	if err := h.validate.Struct(req); err != nil {
		validationErrors := make(map[string]string)
		if validationErr, ok := err.(validator.ValidationErrors); ok {
			for _, fieldErr := range validationErr {
				field := fieldErr.Field()
				tag := fieldErr.Tag()

				var message string
				switch field {
				case "Email":
					if tag == "required" {
						message = "Email is required"
					} else if tag == "email" {
						message = "Invalid email format"
					}
				case "Password":
					if tag == "required" {
						message = "Password is required"
					} else if tag == "min" {
						message = fmt.Sprintf("Password must be at least %s characters long", fieldErr.Param())
					} else if tag == "password_format" {
						message = "Password must contain letters, numbers and special characters (@$!%*#?&), min 8 chars"
					}
				case "Name":
					if tag == "required" {
						message = "Name is required"
					} else if tag == "min" {
						message = "Name cannot be empty"
					}
				default:
					if tag == "required" {
						message = fmt.Sprintf("%s is required", field)
					} else if tag == "min" {
						message = fmt.Sprintf("%s must be at least %s characters long", field, fieldErr.Param())
					} else {
						message = fmt.Sprintf("Invalid %s", field)
					}
				}

				validationErrors[strings.ToLower(field)] = message
			}
		} else {
			validationErrors["general"] = err.Error()
		}

		h.responder.WriteErrorWithCode(c, 400, "VALIDATION_ERROR", "Validation failed", validationErrors)
		return
	}

	out, err := h.service.Register(c.Request.Context(), req)

	if err != nil {
		switch err {
		case authService.ErrUserExists:
			h.responder.WriteErrorWithCode(c, 409, "USER_EXISTS", "User already exists", nil)
		default:
			h.responder.InternalServerError(c, "Failed to register user")
		}
		return
	}

	if out.LoginResponse != nil {
		// Реактивация удалённого пользователя — сразу логин
		accessExpiresAt := time.Now().Add(time.Duration(out.LoginResponse.ExpiresIn) * time.Second)
		h.cookieManager.SetToken(c.Writer, "access_token", out.LoginResponse.AccessToken, accessExpiresAt)
		refreshExpiresAt := time.Now().Add(30 * 24 * time.Hour)
		h.cookieManager.SetToken(c.Writer, "refresh_token", out.LoginResponse.RefreshToken, refreshExpiresAt)
		h.responder.Created(c, "User registered successfully", gin.H{
			"user":       out.LoginResponse.User,
			"expires_in": out.LoginResponse.ExpiresIn,
		})
		return
	}

	// Ожидание подтверждения email
	h.responder.Created(c, out.Message, gin.H{"message": out.Message})
}

func (h *Handler) Logout(c *gin.Context) {
	h.cookieManager.Delete(c.Writer, "access_token")
	h.cookieManager.Delete(c.Writer, "refresh_token")
	h.responder.SuccessWithMessage(c, "Logged out successfully")
}

func (h *Handler) Me(c *gin.Context) {
	userID, ok := middleware.GetUserIDFromGin(c)
	if !ok {
		h.responder.Unauthorized(c, "Authentication required")
		return
	}

	user, err := h.service.GetUserProfile(c.Request.Context(), userID)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to get user profile")
		return
	}

	h.responder.SuccessWithData(c, gin.H{"user": user})
}

func (h *Handler) VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		h.responder.BadRequest(c, "Token is required")
		return
	}

	resp, err := h.service.VerifyEmail(c.Request.Context(), token)
	if err != nil {
		switch err {
		case authService.ErrInvalidVerificationToken:
			h.responder.WriteErrorWithCode(c, 400, "INVALID_TOKEN", "Invalid or expired verification link", nil)
		case authService.ErrUserExists:
			h.responder.WriteErrorWithCode(c, 409, "USER_EXISTS", "User already exists", nil)
		default:
			h.responder.InternalServerError(c, "Failed to verify email")
		}
		return
	}

	accessExpiresAt := time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)
	h.cookieManager.SetToken(c.Writer, "access_token", resp.AccessToken, accessExpiresAt)
	refreshExpiresAt := time.Now().Add(30 * 24 * time.Hour)
	h.cookieManager.SetToken(c.Writer, "refresh_token", resp.RefreshToken, refreshExpiresAt)

	h.responder.SuccessWithData(c, gin.H{
		"user":       resp.User,
		"expires_in": resp.ExpiresIn,
	})
}

func (h *Handler) ChangePassword(c *gin.Context) {
	userID, ok := middleware.GetUserIDFromGin(c)
	if !ok {
		h.responder.Unauthorized(c, "Authentication required")
		return
	}

	var req model.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request")
		return
	}

	req.CurrentPassword = strings.TrimSpace(req.CurrentPassword)
	req.NewPassword = strings.TrimSpace(req.NewPassword)

	if err := h.validate.Struct(req); err != nil {
		validationErrors := make(map[string]string)
		if validationErr, ok := err.(validator.ValidationErrors); ok {
			for _, fieldErr := range validationErr {
				field := strings.ToLower(fieldErr.Field())
				tag := fieldErr.Tag()
				switch fieldErr.Field() {
				case "CurrentPassword":
					if tag == "required" {
						validationErrors[field] = "Current password is required"
					}
				case "NewPassword":
					if tag == "required" {
						validationErrors[field] = "New password is required"
					} else if tag == "min" {
						validationErrors[field] = fmt.Sprintf("Password must be at least %s characters", fieldErr.Param())
					} else if tag == "password_format" {
						validationErrors[field] = "Password must contain letters, numbers and special characters (@$!%*#?&), min 8 chars"
					}
				default:
					validationErrors[field] = fieldErr.Error()
				}
			}
		} else {
			validationErrors["general"] = err.Error()
		}
		h.responder.WriteErrorWithCode(c, 400, "VALIDATION_ERROR", "Validation failed", validationErrors)
		return
	}

	if req.CurrentPassword == req.NewPassword {
		h.responder.WriteErrorWithCode(c, 400, "SAME_PASSWORD", "New password must differ from current", map[string]string{"newPassword": "New password must differ from current"})
		return
	}

	err := h.service.ChangePassword(c.Request.Context(), userID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		switch err {
		case authService.ErrWrongCurrentPassword:
			h.responder.WriteErrorWithCode(c, 400, "WRONG_CURRENT_PASSWORD", "Current password is incorrect", map[string]string{"currentPassword": "Current password is incorrect"})
		case authService.ErrUserNotFound:
			h.responder.Unauthorized(c, "User not found")
		default:
			h.responder.InternalServerError(c, "Failed to change password")
		}
		return
	}

	h.responder.SuccessWithMessage(c, "Password changed successfully")
}

func (h *Handler) Refresh(c *gin.Context) {
	refreshToken, err := h.cookieManager.GetToken(c.Request, "refresh_token")
	if err != nil || refreshToken == "" {
		h.responder.Unauthorized(c, "Refresh token required")
		return
	}

	resp, err := h.service.Refresh(c.Request.Context(), refreshToken)
	if err != nil {
		if err == authService.ErrInvalidRefreshToken {
			h.cookieManager.Delete(c.Writer, "refresh_token")
			h.responder.Unauthorized(c, "Invalid or expired refresh token")
			return
		}
		h.responder.InternalServerError(c, "Failed to refresh token")
		return
	}

	accessExpiresAt := time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)
	h.cookieManager.SetToken(c.Writer, "access_token", resp.AccessToken, accessExpiresAt)
	refreshExpiresAt := time.Now().Add(30 * 24 * time.Hour)
	h.cookieManager.SetToken(c.Writer, "refresh_token", resp.RefreshToken, refreshExpiresAt)

	h.responder.SuccessWithData(c, gin.H{
		"user":       resp.User,
		"expires_in": resp.ExpiresIn,
	})
}
