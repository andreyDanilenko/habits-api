package integration

import (
	"backend/internal/middleware"
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
}

func NewHandler(
	repo *integrationRepo.TelegramRepository,
	responder *response.Responder,
	internalAPIKey string,
	botUsername string,
	tokenTTL time.Duration,
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
	}
}

func (h *Handler) RegisterProtectedRoutes(r *gin.RouterGroup) {
	r.POST("/telegram/link", h.CreateTelegramLink)
}

func (h *Handler) RegisterInternalRoutes(r *gin.RouterGroup) {
	r.POST("/telegram/confirm", h.ConfirmTelegramLink)
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
