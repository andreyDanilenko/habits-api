package integration

import (
	"backend/internal/middleware"
	notificationModel "backend/internal/model"
	notificationService "backend/internal/service/notification"
	integrationRepo "backend/internal/repository/integration"
	"backend/pkg/response"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	repo           *integrationRepo.TelegramRepository
	responder      *response.Responder
	internalAPIKey string
	botUsername    string
	tokenTTL       time.Duration
	notifications  *notificationService.Service
}

func NewHandler(
	repo *integrationRepo.TelegramRepository,
	responder *response.Responder,
	internalAPIKey string,
	botUsername string,
	tokenTTL time.Duration,
	notifications *notificationService.Service,
) *Handler {
	if tokenTTL <= 0 {
		tokenTTL = 15 * time.Minute
	}
	return &Handler{
		repo:           repo,
		responder:      responder,
		internalAPIKey: internalAPIKey,
		botUsername:    botUsername,
		tokenTTL:       tokenTTL,
		notifications:  notifications,
	}
}

func (h *Handler) RegisterProtectedRoutes(r *gin.RouterGroup) {
	r.POST("/telegram/link", h.CreateTelegramLink)
	r.DELETE("/telegram/link", h.DeleteTelegramLink)
	// Статус привязки Telegram для UI (чтобы не “подвязывать” повторно при каждом открытии страницы)
	r.GET("/telegram/status", h.GetTelegramStatus)
}

func (h *Handler) RegisterInternalRoutes(r *gin.RouterGroup) {
	r.POST("/telegram/confirm", h.ConfirmTelegramLink)
	r.GET("/telegram/linked-chat/:userId", h.GetTelegramLinkedChatInternal)
}

type createTelegramLinkResponse struct {
	URL       string `json:"url"`
	ExpiresIn int64  `json:"expiresIn"`
}

func (h *Handler) CreateTelegramLink(c *gin.Context) {
	userID, ok := middleware.GetUserIDFromGin(c)
	if !ok {
		h.responder.Unauthorized(c, "Authentication required")
		return
	}
	if strings.TrimSpace(h.botUsername) == "" {
		h.responder.InternalServerError(c, "Telegram bot is not configured")
		return
	}

	rawToken, tokenHash, err := generateTokenPair()
	if err != nil {
		h.responder.InternalServerError(c, "Failed to generate link token")
		return
	}

	expiresAt := time.Now().Add(h.tokenTTL)
	if err := h.repo.CreateLinkToken(c.Request.Context(), tokenHash, userID, expiresAt); err != nil {
		h.responder.InternalServerError(c, "Failed to store link token")
		return
	}

	url := fmt.Sprintf("https://t.me/%s?start=%s", h.botUsername, rawToken)
	h.responder.SuccessWithData(c, createTelegramLinkResponse{
		URL:       url,
		ExpiresIn: int64(h.tokenTTL.Seconds()),
	})
}

type confirmTelegramLinkRequest struct {
	Token            string `json:"token"`
	ChatID           string `json:"chatId"`
	TelegramUserID   string `json:"telegramUserId"`
	TelegramUsername string `json:"telegramUsername"`
}

func (h *Handler) ConfirmTelegramLink(c *gin.Context) {
	if strings.TrimSpace(h.internalAPIKey) == "" {
		h.responder.InternalServerError(c, "Internal integration key is not configured")
		return
	}
	if c.GetHeader("x-internal-api-key") != h.internalAPIKey {
		h.responder.Unauthorized(c, "Invalid internal api key")
		return
	}

	var req confirmTelegramLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responder.BadRequest(c, "Invalid request body")
		return
	}
	req.Token = strings.TrimSpace(req.Token)
	req.ChatID = strings.TrimSpace(req.ChatID)
	req.TelegramUserID = strings.TrimSpace(req.TelegramUserID)
	req.TelegramUsername = strings.TrimSpace(req.TelegramUsername)

	if req.Token == "" || req.ChatID == "" || req.TelegramUserID == "" {
		h.responder.BadRequest(c, "token, chatId and telegramUserId are required")
		return
	}

	userID, err := h.repo.ConsumeLinkToken(c.Request.Context(), hashToken(req.Token))
	if err != nil {
		if err == integrationRepo.ErrTokenInvalidOrExpired {
			h.responder.Unauthorized(c, "Invalid or expired token")
			return
		}
		h.responder.InternalServerError(c, "Failed to consume token")
		return
	}

	if err := h.repo.UpsertUserLink(
		c.Request.Context(),
		userID,
		req.ChatID,
		req.TelegramUserID,
		req.TelegramUsername,
	); err != nil {
		h.responder.InternalServerError(c, "Failed to save telegram link")
		return
	}

	// "Отдача" из Telegram в приложение: фиксируем в notifications, чтобы пользователь увидел факт привязки.
	// Это демонстрирует принцип интеграций: Telegram (внешний канал) -> ERP (центр уведомлений/логики).
	if h.notifications != nil {
		eventKey := fmt.Sprintf("telegram_connected_%s_%s", req.TelegramUserID, req.ChatID)
		var subtitle *string
		if strings.TrimSpace(req.TelegramUsername) != "" {
			s := "@" + strings.TrimSpace(req.TelegramUsername)
			subtitle = &s
		}

		_, _ = h.notifications.Upsert(c.Request.Context(), userID, &notificationModel.CreateNotificationDto{
			Channel:    "chat",
			EventType:  "telegram",
			EventKey:   eventKey,
			Title:      "Telegram подключен",
			Subtitle:   subtitle,
			WorkspaceID: nil,
		})
	}

	h.responder.SuccessWithData(c, gin.H{"success": true})
}

// GetTelegramLinkedChatInternal — для Nest: chat_id пользователя, если он привязал user-бота (x-internal-api-key).
func (h *Handler) GetTelegramLinkedChatInternal(c *gin.Context) {
	if strings.TrimSpace(h.internalAPIKey) == "" {
		h.responder.InternalServerError(c, "Internal integration key is not configured")
		return
	}
	if c.GetHeader("x-internal-api-key") != h.internalAPIKey {
		h.responder.Unauthorized(c, "Invalid internal api key")
		return
	}
	userID := strings.TrimSpace(c.Param("userId"))
	if userID == "" {
		h.responder.BadRequest(c, "userId required")
		return
	}
	chatID, err := h.repo.GetTelegramChatIDByUserID(c.Request.Context(), userID)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to load telegram link")
		return
	}
	if chatID == "" {
		h.responder.NotFound(c, "No telegram link for user")
		return
	}
	h.responder.SuccessWithData(c, gin.H{"chatId": chatID})
}

// GetTelegramStatus — текущий статус привязки Telegram для авторизованного пользователя.
// Возвращает флаг connected и chatId (если привязано).
func (h *Handler) GetTelegramStatus(c *gin.Context) {
	userID, ok := middleware.GetUserIDFromGin(c)
	if !ok {
		h.responder.Unauthorized(c, "Authentication required")
		return
	}

	chatID, err := h.repo.GetTelegramChatIDByUserID(c.Request.Context(), userID)
	if err != nil {
		h.responder.InternalServerError(c, "Failed to load telegram link")
		return
	}

	linked := strings.TrimSpace(chatID) != ""
	if !linked {
		h.responder.SuccessWithData(c, gin.H{
			"linked":       false,
			"chatId":       "",
			"botUsername": h.botUsername,
		})
		return
	}

	h.responder.SuccessWithData(c, gin.H{
		"linked":       true,
		"chatId":       chatID,
		"botUsername": h.botUsername,
	})
}

// DeleteTelegramLink — полное удаление привязки Telegram для текущего пользователя.
func (h *Handler) DeleteTelegramLink(c *gin.Context) {
	userID, ok := middleware.GetUserIDFromGin(c)
	if !ok {
		h.responder.Unauthorized(c, "Authentication required")
		return
	}

	if err := h.repo.DeleteUserLink(c.Request.Context(), userID); err != nil {
		h.responder.InternalServerError(c, "Failed to delete telegram link")
		return
	}

	h.responder.SuccessWithData(c, gin.H{"success": true})
}

func generateTokenPair() (rawToken, tokenHash string, err error) {
	// 32 random bytes -> 64 hex chars, safe for Telegram /start payload.
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	rawToken = hex.EncodeToString(b)
	return rawToken, hashToken(rawToken), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
