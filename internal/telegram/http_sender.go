package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	telegramPort "backend/pkg/telegram"
)

type httpBotSender struct {
	botToken string
	chatID   int64
	client   *http.Client
}

// NewHTTPBotSender — инфраструктурная реализация Telegram Sender.
func NewHTTPBotSender(botToken string, chatID int64) telegramPort.Sender {
	return &httpBotSender{
		botToken: botToken,
		chatID:   chatID,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

type telegramSendMessageResponse struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result"`
	Description string      `json:"description"`
}

func (s *httpBotSender) SendMessage(ctx context.Context, text string) error {
	// Telegram Bot API принимает application/x-www-form-urlencoded параметры.
	form := url.Values{}
	form.Set("chat_id", fmt.Sprintf("%d", s.chatID))
	form.Set("text", text)

	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", s.botToken)
	encoded := form.Encode()

	var lastErr error
	for attempt := 1; attempt <= 2; attempt++ {
		// Каждая попытка использует свой контекст, но мы не хотим бесконечно ждать.
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(encoded))
		if err != nil {
			return fmt.Errorf("create telegram request: %w", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := s.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("telegram request failed (attempt=%d): %w", attempt, err)
			continue
		}
		defer resp.Body.Close()

		var decoded telegramSendMessageResponse
		if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
			// Иногда Telegram может вернуть не-JSON (HTML/текст) при ошибках сети/прокси.
			b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			return fmt.Errorf("decode telegram response (attempt=%d): %w; http_status=%d body=%s", attempt, err, resp.StatusCode, string(b))
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("telegram http status=%d ok=%v desc=%s", resp.StatusCode, decoded.OK, decoded.Description)
		}
		if !decoded.OK {
			if decoded.Description != "" {
				return fmt.Errorf("telegram api returned ok=false: %s", decoded.Description)
			}
			return fmt.Errorf("telegram api returned ok=false")
		}

		return nil
	}

	return lastErr
}

