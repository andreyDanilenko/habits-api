package auth

import (
	"fmt"
	"net/http"
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
	r.POST(RouteLogin, h.Login)
	r.POST(RouteRegister, h.Register)
	r.POST(RouteLogout, h.Logout)
	r.POST(RouteRefresh, h.Refresh)
}

func (h *Handler) RegisterProtectedRoutes(r *gin.RouterGroup) {
	r.GET(RouteMe, h.Me)
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

	expiresAt := time.Now().Add(time.Duration(loginResp.ExpiresIn) * time.Second)
	h.cookieManager.SetToken(c.Writer, "access_token", loginResp.AccessToken, expiresAt)

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

	registerResp, err := h.service.Register(c.Request.Context(), req)

	if err != nil {
		switch err {
		case authService.ErrUserExists:
			h.responder.WriteErrorWithCode(c, 409, "USER_EXISTS", "User already exists", nil)
		default:
			h.responder.InternalServerError(c, "Failed to register user")
		}
		return
	}

	expiresAt := time.Now().Add(time.Duration(registerResp.ExpiresIn) * time.Second)
	h.cookieManager.SetToken(c.Writer, "access_token", registerResp.AccessToken, expiresAt)

	h.responder.Created(c, "User registered successfully", gin.H{
		"user":       registerResp.User,
		"expires_in": registerResp.ExpiresIn,
	})
}

func (h *Handler) Logout(c *gin.Context) {
	h.cookieManager.Delete(c.Writer, "access_token")
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

func (h *Handler) Refresh(c *gin.Context) {
	h.responder.WriteError(c, http.StatusNotImplemented, "Refresh endpoint is not implemented yet")
}
