package auth

import (
	"backend/internal/model"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
)

var (
	_ = (*model.LoginRequest)(nil)
	_ = (*model.LoginResponse)(nil)
	_ = (*model.RegisterRequest)(nil)
	_ = (*model.User)(nil)
	_ = (*response.ErrorResponse)(nil)
)

// docLogin wraps Login for Swagger. All API docs live in this file so handlers stay clean.
// @Summary      Login user
// @Description  Authenticate user with email and password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body  model.LoginRequest  true  "Login credentials"
// @Success      200   {object}  model.LoginResponse  "User logged in successfully"
// @Failure      400   {object}  response.ErrorResponse  "Validation error"
// @Failure      401   {object}  response.ErrorResponse  "Invalid credentials"
// @Router       /auth/login [post]
func docLogin(h *Handler) gin.HandlerFunc { return h.Login }

// docRegister wraps Register for Swagger.
// @Summary      Register user
// @Description  Register a new user with email, password and name
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body  model.RegisterRequest  true  "Registration data"
// @Success      201   {object}  model.LoginResponse  "User registered successfully"
// @Failure      400   {object}  response.ErrorResponse  "Validation error"
// @Failure      409   {object}  response.ErrorResponse  "User already exists"
// @Router       /auth/register [post]
func docRegister(h *Handler) gin.HandlerFunc { return h.Register }

// docLogout wraps Logout for Swagger.
// @Summary      Logout (clear cookie)
// @Tags         auth
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /auth/logout [post]
func docLogout(h *Handler) gin.HandlerFunc { return h.Logout }

// docMe wraps Me for Swagger.
// @Summary      Get current user
// @Description  Get information about currently authenticated user
// @Tags         auth
// @Produce      json
// @Success      200   {object}  model.User  "User information"
// @Failure      401   {object}  response.ErrorResponse  "Unauthorized"
// @Router       /auth/me [get]
// @Security     BearerAuth
func docMe(h *Handler) gin.HandlerFunc { return h.Me }

// docRefresh wraps Refresh for Swagger.
// @Summary      Refreshing the access token (using a refresh cookie)
// @Tags         auth
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /auth/refresh [post]
func docRefresh(h *Handler) gin.HandlerFunc { return h.Refresh }
