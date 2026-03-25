package integration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrTokenInvalidOrExpired = errors.New("token invalid or expired")

type TelegramRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *TelegramRepository {
	return &TelegramRepository{db: db}
}

func (r *TelegramRepository) CreateLinkToken(ctx context.Context, tokenHash, userID string, expiresAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO telegram_link_tokens (token_hash, user_id, expires_at)
		VALUES ($1, $2, $3)
	`, tokenHash, userID, expiresAt)
	if err != nil {
		return fmt.Errorf("create telegram link token: %w", err)
	}
	return nil
}

func (r *TelegramRepository) ConsumeLinkToken(ctx context.Context, tokenHash string) (string, error) {
	var userID string
	err := r.db.QueryRowContext(ctx, `
		UPDATE telegram_link_tokens
		SET used_at = NOW()
		WHERE token_hash = $1
		  AND used_at IS NULL
		  AND expires_at > NOW()
		RETURNING user_id::text
	`, tokenHash).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrTokenInvalidOrExpired
		}
		return "", fmt.Errorf("consume telegram link token: %w", err)
	}
	return userID, nil
}

// GetTelegramChatIDByUserID — chat_id для sendMessage, если пользователь привязал user-бота.
func (r *TelegramRepository) GetTelegramChatIDByUserID(ctx context.Context, userID string) (string, error) {
	var chatID string
	err := r.db.QueryRowContext(ctx, `
		SELECT telegram_chat_id
		FROM telegram_user_links
		WHERE user_id = $1::uuid AND is_enabled = TRUE
	`, userID).Scan(&chatID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("get telegram chat id: %w", err)
	}
	return chatID, nil
}

func (r *TelegramRepository) UpsertUserLink(
	ctx context.Context,
	userID, chatID, telegramUserID, telegramUsername string,
) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO telegram_user_links (
			user_id, telegram_chat_id, telegram_user_id, telegram_username, is_enabled, connected_at, updated_at
		) VALUES ($1, $2, $3, $4, TRUE, NOW(), NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			telegram_chat_id = EXCLUDED.telegram_chat_id,
			telegram_user_id = EXCLUDED.telegram_user_id,
			telegram_username = EXCLUDED.telegram_username,
			is_enabled = TRUE,
			updated_at = NOW()
	`, userID, chatID, telegramUserID, telegramUsername)
	if err != nil {
		return fmt.Errorf("upsert telegram user link: %w", err)
	}
	return nil
}

// DeleteUserLink — полное удаление связи пользователя с Telegram (без истории).
func (r *TelegramRepository) DeleteUserLink(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM telegram_user_links
		WHERE user_id = $1::uuid
	`, userID)
	if err != nil {
		return fmt.Errorf("delete telegram user link: %w", err)
	}
	return nil
}
