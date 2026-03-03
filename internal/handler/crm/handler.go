package crm

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"backend/internal/middleware"
	"backend/internal/model"
	crmRepo "backend/internal/repository/crm"
	crmService "backend/internal/service/crm"
	workspaceService "backend/internal/service/workspace"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	svc       *crmService.Service
	wsSvc     *workspaceService.Service
	responder *response.Responder
	validate  *validator.Validate
}

func NewHandler(svc *crmService.Service, wsSvc *workspaceService.Service, responder *response.Responder, validate *validator.Validate) *Handler {
	return &Handler{svc: svc, wsSvc: wsSvc, responder: responder, validate: validate}
}

func (h *Handler) requireWorkspaceAccess(c *gin.Context) (workspaceID, userID string, ok bool) {
	userID, ok = middleware.GetUserIDFromGin(c)
	if !ok {
		h.responder.Unauthorized(c, "Authentication required")
		return "", "", false
	}
	workspaceID = c.Param("workspaceId")
	if workspaceID == "" {
		h.responder.BadRequest(c, "Workspace ID required")
		return "", "", false
	}
	roleVal, _ := c.Get(middleware.GinRoleKey)
	role := model.UserRoleUser
	if roleVal != nil {
		role = roleVal.(model.UserRole)
	}
	hasAccess, err := h.wsSvc.HasAccess(c.Request.Context(), workspaceID, userID, role)
	if err != nil || !hasAccess {
		h.responder.Forbidden(c, "Access denied to this workspace")
		return "", "", false
	}
	return workspaceID, userID, true
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	// Mount on group that already has /:workspaceId
	r.GET("/contacts", h.ContactList)
	r.GET("/contacts/:id", h.ContactGet)
	r.POST("/contacts", h.ContactCreate)
	r.PUT("/contacts/:id", h.ContactUpdate)
	r.DELETE("/contacts/:id", h.ContactDelete)

	r.GET("/companies", h.CompanyList)
	r.GET("/companies/:id", h.CompanyGet)
	r.POST("/companies", h.CompanyCreate)
	r.PUT("/companies/:id", h.CompanyUpdate)
	r.DELETE("/companies/:id", h.CompanyDelete)
	r.POST("/companies/:id/contacts/:contactId", h.CompanyAttachContact)
	r.DELETE("/companies/:id/contacts/:contactId", h.CompanyDetachContact)

	r.GET("/pipelines", h.PipelineList)
	r.POST("/pipelines", h.PipelineCreate)
	r.GET("/pipelines/:pipelineId", h.PipelineGet)
	r.PUT("/pipelines/:pipelineId", h.PipelineUpdate)
	r.DELETE("/pipelines/:pipelineId", h.PipelineDelete)

	r.GET("/pipelines/:pipelineId/stages", h.StageList)
	r.GET("/pipelines/:pipelineId/stages/:id", h.StageGet)
	r.POST("/pipelines/:pipelineId/stages", h.StageCreate)
	r.PUT("/pipelines/:pipelineId/stages/:id", h.StageUpdate)
	r.DELETE("/pipelines/:pipelineId/stages/:id", h.StageDelete)
	r.POST("/pipelines/:pipelineId/stages/reorder", h.StageReorder)

	r.GET("/deals", h.DealList)
	r.GET("/deals/:id", h.DealGet)
	r.POST("/deals", h.DealCreate)
	r.PUT("/deals/:id", h.DealUpdate)
	r.DELETE("/deals/:id", h.DealDelete)

	r.GET("/activities", h.ActivityList)
	r.POST("/activities", h.ActivityCreate)
	r.GET("/activities/:id", h.ActivityGet)
	r.PUT("/activities/:id", h.ActivityUpdate)
	r.DELETE("/activities/:id", h.ActivityDelete)
	r.POST("/activities/:id/important", h.ActivityToggleImportant)
}

// Contacts
func (h *Handler) ContactList(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	opts := crmRepo.ContactListOpts{
		Page:      parseInt(c.Query("page"), 1),
		Limit:     parseInt(c.Query("limit"), 50),
		Search:    c.Query("search"),
		CompanyID: c.Query("companyId"),
		OwnerID:   c.Query("ownerId"),
		SortBy:    c.Query("sortBy"),
		SortOrder: c.Query("sortOrder"),
	}
	if opts.SortOrder == "" {
		opts.SortOrder = "desc"
	}
	list, total, err := h.svc.ContactList(c.Request.Context(), workspaceID, opts)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to list contacts")
		return
	}
	h.responder.SuccessWithData(c, gin.H{"contacts": list, "total": total})
}

func (h *Handler) ContactGet(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	id := c.Param("id")
	contact, err := h.svc.ContactGet(c.Request.Context(), workspaceID, id)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to get contact")
		return
	}
	if contact == nil {
		h.responder.NotFound(c, "Contact not found")
		return
	}
	h.responder.SuccessWithData(c, contact)
}

func (h *Handler) ContactCreate(c *gin.Context) {
	workspaceID, userID, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	var req model.Contact
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request body")
		return
	}
	if req.FirstName == "" {
		h.responder.BadRequest(c, "firstName is required")
		return
	}
	if req.LastName == "" {
		req.LastName = ""
	}
	if req.OwnerID == "" {
		req.OwnerID = userID
	}
	if err := h.svc.ContactCreate(c.Request.Context(), workspaceID, &req, userID); err != nil {
		h.responder.InternalServerError(c, "Failed to create contact")
		return
	}
	h.responder.Created(c, "", req)
}

func (h *Handler) ContactUpdate(c *gin.Context) {
	workspaceID, userID, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	id := c.Param("id")
	existing, err := h.svc.ContactGet(c.Request.Context(), workspaceID, id)
	if err != nil || existing == nil {
		h.responder.NotFound(c, "Contact not found")
		return
	}
	var req model.Contact
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request body")
		return
	}
	mergeContact(existing, &req)
	existing.ID = id
	if err := h.svc.ContactUpdate(c.Request.Context(), workspaceID, existing, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.responder.NotFound(c, "Contact not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to update contact")
		return
	}
	updated, _ := h.svc.ContactGet(c.Request.Context(), workspaceID, id)
	h.responder.SuccessWithData(c, updated)
}

func mergeContact(dst, src *model.Contact) {
	if src.FirstName != "" {
		dst.FirstName = src.FirstName
	}
	if src.LastName != "" {
		dst.LastName = src.LastName
	}
	if src.MiddleName != "" {
		dst.MiddleName = src.MiddleName
	}
	if src.CompanyID != "" {
		dst.CompanyID = src.CompanyID
	}
	if src.Position != "" {
		dst.Position = src.Position
	}
	if src.Birthday != "" {
		dst.Birthday = src.Birthday
	}
	if src.OwnerID != "" {
		dst.OwnerID = src.OwnerID
	}
	if len(src.Phones) > 0 {
		dst.Phones = src.Phones
	}
	if len(src.Emails) > 0 {
		dst.Emails = src.Emails
	}
	if len(src.Tags) > 0 {
		dst.Tags = src.Tags
	}
}

func (h *Handler) ContactDelete(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	id := c.Param("id")
	if err := h.svc.ContactDelete(c.Request.Context(), workspaceID, id); err != nil {
		if errors.Is(err, crmService.ErrContactHasDeals) {
			h.responder.WriteErrorWithCode(c, http.StatusConflict, "CONFLICT", "Contact has linked deals", nil)
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			h.responder.NotFound(c, "Contact not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to delete contact")
		return
	}
	c.AbortWithStatus(http.StatusNoContent)
}

// Companies
func (h *Handler) CompanyList(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	opts := crmRepo.CompanyListOpts{
		Page:      parseInt(c.Query("page"), 1),
		Limit:     parseInt(c.Query("limit"), 50),
		Search:    c.Query("search"),
		OwnerID:   c.Query("ownerId"),
		SortBy:    c.Query("sortBy"),
		SortOrder: c.Query("sortOrder"),
	}
	list, total, err := h.svc.CompanyList(c.Request.Context(), workspaceID, opts)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to list companies")
		return
	}
	h.responder.SuccessWithData(c, gin.H{"companies": list, "total": total})
}

func (h *Handler) CompanyGet(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	id := c.Param("id")
	company, err := h.svc.CompanyGet(c.Request.Context(), workspaceID, id)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to get company")
		return
	}
	if company == nil {
		h.responder.NotFound(c, "Company not found")
		return
	}
	h.responder.SuccessWithData(c, company)
}

func (h *Handler) CompanyCreate(c *gin.Context) {
	workspaceID, userID, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	var req model.Company
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request body")
		return
	}
	if req.Name == "" {
		h.responder.BadRequest(c, "name is required")
		return
	}
	if req.OwnerID == "" {
		req.OwnerID = userID
	}
	if err := h.svc.CompanyCreate(c.Request.Context(), workspaceID, &req, userID); err != nil {
		h.responder.InternalServerError(c, "Failed to create company")
		return
	}
	h.responder.Created(c, "", req)
}

func (h *Handler) CompanyUpdate(c *gin.Context) {
	workspaceID, userID, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	id := c.Param("id")
	existing, err := h.svc.CompanyGet(c.Request.Context(), workspaceID, id)
	if err != nil || existing == nil {
		h.responder.NotFound(c, "Company not found")
		return
	}
	var req model.Company
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request body")
		return
	}
	mergeCompany(existing, &req)
	existing.ID = id
	if err := h.svc.CompanyUpdate(c.Request.Context(), workspaceID, existing, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.responder.NotFound(c, "Company not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to update company")
		return
	}
	updated, _ := h.svc.CompanyGet(c.Request.Context(), workspaceID, id)
	h.responder.SuccessWithData(c, updated)
}

func mergeCompany(dst, src *model.Company) {
	if src.Name != "" {
		dst.Name = src.Name
	}
	if src.INN != "" {
		dst.INN = src.INN
	}
	if src.KPP != "" {
		dst.KPP = src.KPP
	}
	if src.OGRN != "" {
		dst.OGRN = src.OGRN
	}
	if src.Phone != "" {
		dst.Phone = src.Phone
	}
	if src.Email != "" {
		dst.Email = src.Email
	}
	if src.Website != "" {
		dst.Website = src.Website
	}
	if src.OwnerID != "" {
		dst.OwnerID = src.OwnerID
	}
	if src.LegalAddress != nil {
		dst.LegalAddress = src.LegalAddress
	}
	if src.ActualAddress != nil {
		dst.ActualAddress = src.ActualAddress
	}
	if len(src.Tags) > 0 {
		dst.Tags = src.Tags
	}
}

func (h *Handler) CompanyDelete(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	id := c.Param("id")
	if err := h.svc.CompanyDelete(c.Request.Context(), workspaceID, id); err != nil {
		if errors.Is(err, crmService.ErrCompanyHasDeals) {
			h.responder.WriteErrorWithCode(c, http.StatusConflict, "CONFLICT", "Company has linked deals or contacts", nil)
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			h.responder.NotFound(c, "Company not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to delete company")
		return
	}
	c.AbortWithStatus(http.StatusNoContent)
}

func (h *Handler) CompanyAttachContact(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	companyID := c.Param("id")
	contactID := c.Param("contactId")
	var body struct {
		Position string `json:"position"`
	}
	_ = c.ShouldBindJSON(&body)
	if err := h.svc.CompanyAttachContact(c.Request.Context(), workspaceID, companyID, contactID, body.Position); err != nil {
		h.responder.InternalServerError(c, "Failed to attach contact")
		return
	}
	h.responder.SuccessWithData(c, gin.H{})
}

func (h *Handler) CompanyDetachContact(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	companyID := c.Param("id")
	contactID := c.Param("contactId")
	if err := h.svc.CompanyDetachContact(c.Request.Context(), workspaceID, companyID, contactID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.responder.NotFound(c, "Link not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to detach contact")
		return
	}
	c.AbortWithStatus(http.StatusNoContent)
}

// Pipelines
func (h *Handler) PipelineList(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	list, err := h.svc.PipelineList(c.Request.Context(), workspaceID)
	if err != nil {
		log.Printf("[CRM] PipelineList: %v", err)
		h.responder.InternalServerError(c, "Failed to list pipelines")
		return
	}
	h.responder.SuccessWithData(c, gin.H{"pipelines": list})
}

func (h *Handler) PipelineCreate(c *gin.Context) {
	workspaceID, userID, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	var req model.Pipeline
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request body")
		return
	}
	if req.Name == "" {
		h.responder.BadRequest(c, "name is required")
		return
	}
	if err := h.svc.PipelineCreate(c.Request.Context(), workspaceID, &req, userID); err != nil {
		h.responder.InternalServerError(c, "Failed to create pipeline")
		return
	}
	h.responder.Created(c, "", req)
}

func (h *Handler) PipelineGet(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	pipelineID := c.Param("pipelineId")
	p, err := h.svc.PipelineGet(c.Request.Context(), workspaceID, pipelineID)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to get pipeline")
		return
	}
	if p == nil {
		h.responder.NotFound(c, "Pipeline not found")
		return
	}
	h.responder.SuccessWithData(c, p)
}

func (h *Handler) PipelineUpdate(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	pipelineID := c.Param("pipelineId")
	existing, err := h.svc.PipelineGet(c.Request.Context(), workspaceID, pipelineID)
	if err != nil || existing == nil {
		h.responder.NotFound(c, "Pipeline not found")
		return
	}
	var req model.Pipeline
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request body")
		return
	}
	mergePipeline(existing, &req)
	existing.ID = pipelineID
	if err := h.svc.PipelineUpdate(c.Request.Context(), workspaceID, existing); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.responder.NotFound(c, "Pipeline not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to update pipeline")
		return
	}
	updated, _ := h.svc.PipelineGet(c.Request.Context(), workspaceID, pipelineID)
	h.responder.SuccessWithData(c, updated)
}

func mergePipeline(dst, src *model.Pipeline) {
	if src.Name != "" {
		dst.Name = src.Name
	}
	// zero-value bool is false; we want ability to set default flag explicitly.
	// Здесь предполагаем, что фронт отправляет isDefault всегда.
	dst.IsDefault = src.IsDefault
	if src.Stages != nil && len(src.Stages) > 0 {
		dst.Stages = src.Stages
	}
}

func (h *Handler) PipelineDelete(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	pipelineID := c.Param("pipelineId")
	if err := h.svc.PipelineDelete(c.Request.Context(), workspaceID, pipelineID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.responder.NotFound(c, "Pipeline not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to delete pipeline")
		return
	}
	c.AbortWithStatus(http.StatusNoContent)
}

// Stages

func (h *Handler) StageList(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	pipelineID := c.Param("pipelineId")
	stages, err := h.svc.StageList(c.Request.Context(), workspaceID, pipelineID)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to list stages")
		return
	}
	if stages == nil {
		h.responder.NotFound(c, "Pipeline not found")
		return
	}
	h.responder.SuccessWithData(c, gin.H{"stages": stages})
}

func (h *Handler) StageGet(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	pipelineID := c.Param("pipelineId")
	id := c.Param("id")
	stage, err := h.svc.StageGet(c.Request.Context(), workspaceID, pipelineID, id)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to get stage")
		return
	}
	if stage == nil {
		h.responder.NotFound(c, "Stage not found")
		return
	}
	h.responder.SuccessWithData(c, stage)
}

func (h *Handler) StageCreate(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	pipelineID := c.Param("pipelineId")
	var req model.Stage
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request body")
		return
	}
	if req.Name == "" {
		h.responder.BadRequest(c, "name is required")
		return
	}
	if err := h.svc.StageCreate(c.Request.Context(), workspaceID, pipelineID, &req); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.responder.NotFound(c, "Pipeline not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to create stage")
		return
	}
	h.responder.Created(c, "", req)
}

func (h *Handler) StageUpdate(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	pipelineID := c.Param("pipelineId")
	id := c.Param("id")
	existing, err := h.svc.StageGet(c.Request.Context(), workspaceID, pipelineID, id)
	if err != nil || existing == nil {
		h.responder.NotFound(c, "Stage not found")
		return
	}
	var req model.Stage
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request body")
		return
	}
	mergeStage(existing, &req)
	if err := h.svc.StageUpdate(c.Request.Context(), workspaceID, pipelineID, id, existing); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.responder.NotFound(c, "Stage not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to update stage")
		return
	}
	updated, _ := h.svc.StageGet(c.Request.Context(), workspaceID, pipelineID, id)
	h.responder.SuccessWithData(c, updated)
}

func mergeStage(dst, src *model.Stage) {
	if src.Name != "" {
		dst.Name = src.Name
	}
	if src.Color != "" {
		dst.Color = src.Color
	}
	if src.Probability != 0 {
		dst.Probability = src.Probability
	}
	// boolean flags: предполагаем, что фронт всегда шлёт оба поля
	dst.IsFinal = src.IsFinal
	dst.IsLost = src.IsLost
}

func (h *Handler) StageDelete(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	pipelineID := c.Param("pipelineId")
	id := c.Param("id")
	if err := h.svc.StageDelete(c.Request.Context(), workspaceID, pipelineID, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.responder.NotFound(c, "Stage not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to delete stage")
		return
	}
	c.AbortWithStatus(http.StatusNoContent)
}

func (h *Handler) StageReorder(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	pipelineID := c.Param("pipelineId")
	var body struct {
		StageIDs []string `json:"stageIds"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		h.responder.BadRequest(c, "Invalid request body")
		return
	}
	if len(body.StageIDs) == 0 {
		h.responder.BadRequest(c, "stageIds is required")
		return
	}
	if err := h.svc.StageReorder(c.Request.Context(), workspaceID, pipelineID, body.StageIDs); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.responder.NotFound(c, "Pipeline not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to reorder stages")
		return
	}
	h.responder.SuccessWithData(c, gin.H{})
}

// Deals
func (h *Handler) DealList(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	opts := crmRepo.DealListOpts{
		Page:       parseInt(c.Query("page"), 1),
		Limit:      parseInt(c.Query("limit"), 50),
		PipelineID: c.Query("pipelineId"),
		StageID:    c.Query("stageId"),
		CompanyID:  c.Query("companyId"),
		ContactID:  c.Query("contactId"),
		OwnerID:    c.Query("ownerId"),
		Status:     c.Query("status"),
		DateFrom:   c.Query("dateFrom"),
		DateTo:     c.Query("dateTo"),
		SortBy:     c.Query("sortBy"),
		SortOrder:  c.Query("sortOrder"),
	}
	list, total, err := h.svc.DealList(c.Request.Context(), workspaceID, opts)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to list deals")
		return
	}
	h.responder.SuccessWithData(c, gin.H{"deals": list, "total": total})
}

func (h *Handler) DealGet(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	id := c.Param("id")
	deal, err := h.svc.DealGet(c.Request.Context(), workspaceID, id)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to get deal")
		return
	}
	if deal == nil {
		h.responder.NotFound(c, "Deal not found")
		return
	}
	h.responder.SuccessWithData(c, deal)
}

func (h *Handler) DealCreate(c *gin.Context) {
	workspaceID, userID, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	var req model.Deal
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request body")
		return
	}
	if req.Name == "" || req.PipelineID == "" || req.StageID == "" {
		h.responder.BadRequest(c, "name, pipelineId and stageId are required")
		return
	}
	if req.Currency == "" {
		req.Currency = "RUB"
	}
	if req.Budget == 0 {
		req.Budget = 0
	}
	if req.OwnerID == "" {
		req.OwnerID = userID
	}
	if err := h.svc.DealCreate(c.Request.Context(), workspaceID, &req, userID); err != nil {
		h.responder.InternalServerError(c, "Failed to create deal")
		return
	}
	h.responder.Created(c, "", req)
}

func (h *Handler) DealUpdate(c *gin.Context) {
	workspaceID, userID, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	id := c.Param("id")
	existing, err := h.svc.DealGet(c.Request.Context(), workspaceID, id)
	if err != nil || existing == nil {
		h.responder.NotFound(c, "Deal not found")
		return
	}
	var req model.Deal
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request body")
		return
	}
	mergeDeal(existing, &req)
	existing.ID = id
	if err := h.svc.DealUpdate(c.Request.Context(), workspaceID, existing, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.responder.NotFound(c, "Deal not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to update deal")
		return
	}
	updated, _ := h.svc.DealGet(c.Request.Context(), workspaceID, id)
	h.responder.SuccessWithData(c, updated)
}

func mergeDeal(dst, src *model.Deal) {
	if src.Name != "" {
		dst.Name = src.Name
	}
	if src.ContactID != "" {
		dst.ContactID = src.ContactID
	}
	if src.CompanyID != "" {
		dst.CompanyID = src.CompanyID
	}
	if src.Budget != 0 {
		dst.Budget = src.Budget
	}
	if src.Currency != "" {
		dst.Currency = src.Currency
	}
	if src.PipelineID != "" {
		dst.PipelineID = src.PipelineID
	}
	if src.StageID != "" {
		dst.StageID = src.StageID
	}
	if src.ExpectedCloseDate != "" {
		dst.ExpectedCloseDate = src.ExpectedCloseDate
	}
	if src.ActualCloseDate != "" {
		dst.ActualCloseDate = src.ActualCloseDate
	}
	if src.Status != "" {
		dst.Status = src.Status
	}
	if src.LostReason != "" {
		dst.LostReason = src.LostReason
	}
	if src.Description != "" {
		dst.Description = src.Description
	}
	if src.Source != "" {
		dst.Source = src.Source
	}
	if src.Probability != nil {
		dst.Probability = src.Probability
	}
	if src.OwnerID != "" {
		dst.OwnerID = src.OwnerID
	}
	if len(src.Tags) > 0 {
		dst.Tags = src.Tags
	}
}

func (h *Handler) DealDelete(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	id := c.Param("id")
	if err := h.svc.DealDelete(c.Request.Context(), workspaceID, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			h.responder.NotFound(c, "Deal not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to delete deal")
		return
	}
	c.AbortWithStatus(http.StatusNoContent)
}

// Activities (SPEC_BACK_2)
func (h *Handler) ActivityList(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	entityType := c.Query("entityType")
	entityId := c.Query("entityId")
	if entityType == "" || entityId == "" {
		h.responder.BadRequest(c, "entityType and entityId are required")
		return
	}
	if entityType != "contact" && entityType != "company" && entityType != "deal" {
		h.responder.BadRequest(c, "entityType must be contact, company or deal")
		return
	}
	var types []string
	if t := c.Query("types"); t != "" {
		types = strings.Split(t, ",")
		for i := range types {
			types[i] = strings.TrimSpace(types[i])
		}
	}
	opts := crmRepo.ActivityListOpts{
		Page:          parseInt(c.Query("page"), 1),
		Limit:         parseInt(c.Query("limit"), 50),
		Types:         types,
		DateFrom:      c.Query("dateFrom"),
		DateTo:        c.Query("dateTo"),
		ImportantOnly: c.Query("importantOnly") == "true",
		Search:        c.Query("search"),
	}
	list, total, err := h.svc.ActivityList(c.Request.Context(), workspaceID, entityType, entityId, opts)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to list activities")
		return
	}
	h.responder.SuccessWithData(c, gin.H{"data": list, "total": total, "page": opts.Page, "limit": opts.Limit})
}

func (h *Handler) ActivityGet(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	id := c.Param("id")
	a, err := h.svc.ActivityGet(c.Request.Context(), workspaceID, id)
	if err != nil || a == nil {
		h.responder.NotFound(c, "Activity not found")
		return
	}
	h.responder.SuccessWithData(c, a)
}

func (h *Handler) ActivityCreate(c *gin.Context) {
	workspaceID, userID, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	var body struct {
		Type        string                 `json:"type"`
		EntityType  string                 `json:"entityType"`
		EntityID    string                 `json:"entityId"`
		Title       string                 `json:"title"`
		Description string                 `json:"description"`
		IsImportant bool                   `json:"isImportant"`
		Metadata    map[string]interface{} `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		h.responder.BadRequest(c, "Invalid request body")
		return
	}
	if body.EntityType == "" || body.EntityID == "" || body.Title == "" {
		h.responder.BadRequest(c, "entityType, entityId and title are required")
		return
	}
	if body.EntityType != "contact" && body.EntityType != "company" && body.EntityType != "deal" {
		h.responder.BadRequest(c, "entityType must be contact, company or deal")
		return
	}
	if len(body.Title) > 500 {
		h.responder.BadRequest(c, "title max 500 characters")
		return
	}
	a := &model.CrmActivity{
		EntityType:  body.EntityType,
		EntityID:    body.EntityID,
		Title:       body.Title,
		Description: body.Description,
		IsImportant: body.IsImportant,
		Metadata:    body.Metadata,
	}
	switch body.Type {
	case "note":
		if err := h.svc.ActivityCreateNote(c.Request.Context(), workspaceID, a, userID); err != nil {
			if errors.Is(err, crmService.ErrEntityNotFound) {
				h.responder.NotFound(c, "Entity not found")
				return
			}
			h.responder.InternalServerError(c, "Failed to create note")
			return
		}
	case "call":
		if body.Metadata == nil {
			body.Metadata = make(map[string]interface{})
		}
		direction, _ := body.Metadata["callDirection"].(string)
		if direction == "" {
			direction, _ = body.Metadata["direction"].(string)
		}
		status, _ := body.Metadata["callStatus"].(string)
		if status == "" {
			status, _ = body.Metadata["status"].(string)
		}
		if direction == "" || status == "" {
			h.responder.BadRequest(c, "callDirection and callStatus are required for call")
			return
		}
		if status == "answered" {
			if dur, _ := body.Metadata["callDuration"].(float64); dur < 1 {
				if di, _ := body.Metadata["duration"].(float64); di < 1 {
					h.responder.BadRequest(c, "callDuration required when callStatus is answered")
					return
				}
			}
		}
		a.Metadata = body.Metadata
		if err := h.svc.ActivityCreateCall(c.Request.Context(), workspaceID, a, userID); err != nil {
			if errors.Is(err, crmService.ErrEntityNotFound) {
				h.responder.NotFound(c, "Entity not found")
				return
			}
			h.responder.InternalServerError(c, "Failed to create call")
			return
		}
	default:
		h.responder.BadRequest(c, "type must be note or call")
		return
	}
	h.responder.Created(c, "", a)
}

func (h *Handler) ActivityUpdate(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	id := c.Param("id")
	existing, err := h.svc.ActivityGet(c.Request.Context(), workspaceID, id)
	if err != nil || existing == nil {
		h.responder.NotFound(c, "Activity not found")
		return
	}
	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		IsImportant *bool  `json:"isImportant"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		h.responder.BadRequest(c, "Invalid request body")
		return
	}
	if body.Title != "" {
		existing.Title = body.Title
	}
	if body.Description != "" {
		existing.Description = body.Description
	}
	if body.IsImportant != nil {
		existing.IsImportant = *body.IsImportant
	}
	if err := h.svc.ActivityUpdate(c.Request.Context(), workspaceID, existing); err != nil {
		if errors.Is(err, crmService.ErrActivityNotNote) {
			h.responder.WriteErrorWithCode(c, http.StatusConflict, "CONFLICT", "Cannot update system-generated activity", nil)
			return
		}
		h.responder.InternalServerError(c, "Failed to update activity")
		return
	}
	updated, _ := h.svc.ActivityGet(c.Request.Context(), workspaceID, id)
	h.responder.SuccessWithData(c, updated)
}

func (h *Handler) ActivityDelete(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	id := c.Param("id")
	if err := h.svc.ActivityDelete(c.Request.Context(), workspaceID, id); err != nil {
		if errors.Is(err, crmService.ErrActivityNotNote) {
			h.responder.WriteErrorWithCode(c, http.StatusConflict, "CONFLICT", "Cannot delete system-generated activity", nil)
			return
		}
		if errors.Is(err, sql.ErrNoRows) {
			h.responder.NotFound(c, "Activity not found")
			return
		}
		h.responder.InternalServerError(c, "Failed to delete activity")
		return
	}
	c.AbortWithStatus(http.StatusNoContent)
}

func (h *Handler) ActivityToggleImportant(c *gin.Context) {
	workspaceID, _, ok := h.requireWorkspaceAccess(c)
	if !ok {
		return
	}
	id := c.Param("id")
	existing, err := h.svc.ActivityGet(c.Request.Context(), workspaceID, id)
	if err != nil || existing == nil {
		h.responder.NotFound(c, "Activity not found")
		return
	}
	var body struct {
		IsImportant *bool `json:"isImportant"`
	}
	_ = c.ShouldBindJSON(&body)
	isImportant := !existing.IsImportant
	if body.IsImportant != nil {
		isImportant = *body.IsImportant
	}
	if err := h.svc.ActivitySetImportant(c.Request.Context(), workspaceID, id, isImportant); err != nil {
		h.responder.InternalServerError(c, "Failed to update importance")
		return
	}
	updated, _ := h.svc.ActivityGet(c.Request.Context(), workspaceID, id)
	h.responder.SuccessWithData(c, updated)
}

func parseInt(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	n, _ := strconv.Atoi(s)
	if n <= 0 {
		return defaultVal
	}
	return n
}
