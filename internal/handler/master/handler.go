package master

import (
	"database/sql"
	"errors"

	"backend/internal/middleware"
	"backend/internal/model"
	masterService "backend/internal/service/master"
	workspaceService "backend/internal/service/workspace"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	masterSvc   *masterService.Service
	workspaceSvc *workspaceService.Service
	responder    *response.Responder
	validate     *validator.Validate
}

func NewHandler(
	masterSvc *masterService.Service,
	workspaceSvc *workspaceService.Service,
	responder *response.Responder,
	validate *validator.Validate,
) *Handler {
	return &Handler{
		masterSvc:   masterSvc,
		workspaceSvc: workspaceSvc,
		responder:    responder,
		validate:     validate,
	}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	currencies := r.Group("/currencies")
	{
		currencies.GET("", h.ListCurrencies)
		currencies.POST("", h.CreateCurrency)
		currencies.GET("/:currencyId", h.GetCurrency)
		currencies.PUT("/:currencyId", h.UpdateCurrency)
		currencies.DELETE("/:currencyId", h.DeleteCurrency)
	}
	counterparties := r.Group("/counterparties")
	{
		counterparties.GET("", h.ListCounterparties)
		counterparties.POST("", h.CreateCounterparty)
		counterparties.GET("/:counterpartyId", h.GetCounterparty)
		counterparties.PUT("/:counterpartyId", h.UpdateCounterparty)
		counterparties.DELETE("/:counterpartyId", h.DeleteCounterparty)
	}
}

func (h *Handler) requireWorkspaceAccess(c *gin.Context) (workspaceID, userID string, ok bool) {
	userID, ok = middleware.GetUserIDFromGin(c)
	if !ok {
		h.responder.Unauthorized(c, "Authentication required")
		return "", "", false
	}
	workspaceID = c.Param("id")
	if workspaceID == "" {
		h.responder.BadRequest(c, "Workspace ID required")
		return "", "", false
	}
	roleVal, _ := c.Get(middleware.GinRoleKey)
	role := model.UserRoleUser
	if roleVal != nil {
		role = roleVal.(model.UserRole)
	}
	hasAccess, err := h.workspaceSvc.HasAccess(c.Request.Context(), workspaceID, userID, role)
	if err != nil || !hasAccess {
		h.responder.Forbidden(c, "Access denied to this workspace")
		return "", "", false
	}
	return workspaceID, userID, true
}

// Currencies
func (h *Handler) ListCurrencies(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	list, err := h.masterSvc.ListCurrencies(c.Request.Context(), workspaceID)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to list currencies")
		return
	}
	h.responder.SuccessWithData(c, gin.H{"currencies": list})
}

func (h *Handler) GetCurrency(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	currencyID := c.Param("currencyId")
	cur, err := h.masterSvc.GetCurrency(c.Request.Context(), workspaceID, currencyID)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to get currency")
		return
	}
	if cur == nil {
		h.responder.NotFound(c, "Currency not found")
		return
	}
	h.responder.SuccessWithData(c, cur)
}

func (h *Handler) CreateCurrency(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	var req struct {
		Code   string  `json:"code" binding:"required"`
		Name   string  `json:"name" binding:"required"`
		Symbol *string `json:"symbol"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "code and name are required")
		return
	}
	cur := &model.Currency{
		WorkspaceID: workspaceID,
		Code:        req.Code,
		Name:        req.Name,
		Symbol:      req.Symbol,
	}
	if err := h.masterSvc.CreateCurrency(c.Request.Context(), cur); err != nil {
		h.responder.InternalServerError(c, "Failed to create currency")
		return
	}
	h.responder.SuccessWithData(c, cur)
}

func (h *Handler) UpdateCurrency(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	currencyID := c.Param("currencyId")
	var req struct {
		Code   string  `json:"code" binding:"required"`
		Name   string  `json:"name" binding:"required"`
		Symbol *string `json:"symbol"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "code and name are required")
		return
	}
	cur := &model.Currency{
		ID:          currencyID,
		WorkspaceID: workspaceID,
		Code:        req.Code,
		Name:        req.Name,
		Symbol:      req.Symbol,
	}
	if err := h.masterSvc.UpdateCurrency(c.Request.Context(), cur); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.responder.NotFound(c, "Currency not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to update currency")
		return
	}
	h.responder.SuccessWithData(c, cur)
}

func (h *Handler) DeleteCurrency(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	currencyID := c.Param("currencyId")
	if err := h.masterSvc.DeleteCurrency(c.Request.Context(), workspaceID, currencyID); err != nil {
		h.responder.InternalServerError(c, "Failed to delete currency")
		return
	}
	h.responder.SuccessWithMessage(c, "Currency deleted")
}

// Counterparties
func (h *Handler) ListCounterparties(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	list, err := h.masterSvc.ListCounterparties(c.Request.Context(), workspaceID)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to list counterparties")
		return
	}
	h.responder.SuccessWithData(c, gin.H{"counterparties": list})
}

func (h *Handler) GetCounterparty(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	counterpartyID := c.Param("counterpartyId")
	cp, err := h.masterSvc.GetCounterparty(c.Request.Context(), workspaceID, counterpartyID)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to get counterparty")
		return
	}
	if cp == nil {
		h.responder.NotFound(c, "Counterparty not found")
		return
	}
	h.responder.SuccessWithData(c, cp)
}

func (h *Handler) CreateCounterparty(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	var req struct {
		Name    string  `json:"name" binding:"required"`
		Type    string  `json:"type"` // client, supplier, both
		Email   *string `json:"email"`
		Phone   *string `json:"phone"`
		Comment *string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "name is required")
		return
	}
	if req.Type == "" {
		req.Type = "client"
	}
	if req.Type != "client" && req.Type != "supplier" && req.Type != "both" {
		h.responder.BadRequest(c, "type must be client, supplier or both")
		return
	}
	cp := &model.Counterparty{
		WorkspaceID: workspaceID,
		Name:        req.Name,
		Type:        req.Type,
		Email:       req.Email,
		Phone:       req.Phone,
		Comment:     req.Comment,
	}
	if err := h.masterSvc.CreateCounterparty(c.Request.Context(), cp); err != nil {
		h.responder.InternalServerError(c, "Failed to create counterparty")
		return
	}
	h.responder.SuccessWithData(c, cp)
}

func (h *Handler) UpdateCounterparty(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	counterpartyID := c.Param("counterpartyId")
	var req struct {
		Name    string  `json:"name" binding:"required"`
		Type    string  `json:"type"`
		Email   *string `json:"email"`
		Phone   *string `json:"phone"`
		Comment *string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "name is required")
		return
	}
	if req.Type == "" {
		req.Type = "client"
	}
	cp := &model.Counterparty{
		ID:          counterpartyID,
		WorkspaceID: workspaceID,
		Name:        req.Name,
		Type:        req.Type,
		Email:       req.Email,
		Phone:       req.Phone,
		Comment:     req.Comment,
	}
	if err := h.masterSvc.UpdateCounterparty(c.Request.Context(), cp); err != nil {
		h.responder.InternalServerError(c, "Failed to update counterparty")
		return
	}
	h.responder.SuccessWithData(c, cp)
}

func (h *Handler) DeleteCounterparty(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	counterpartyID := c.Param("counterpartyId")
	if err := h.masterSvc.DeleteCounterparty(c.Request.Context(), workspaceID, counterpartyID); err != nil {
		h.responder.InternalServerError(c, "Failed to delete counterparty")
		return
	}
	h.responder.SuccessWithMessage(c, "Counterparty deleted")
}
