package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server       ServerConfig
	Database     DatabaseConfig
	Logs         LogsConfig
	Auth         AuthConfig
	Email        EmailConfig
	Realtime     RealtimeConfig
	Uploads      UploadsConfig
	Activity     ActivityConfig
	Telegram     TelegramConfig
	Integrations IntegrationsConfig
}

// ActivityConfig — запись событий в общую ленту (таблица activities).
type ActivityConfig struct {
	Enabled bool // ACTIVITY_LOG_ENABLED, по умолчанию true
}

type UploadsConfig struct {
	Dir string `env:"UPLOADS_DIR" envDefault:"./uploads"`
}

type RealtimeConfig struct {
	RedisURL string `env:"REDIS_URL"` // e.g. redis://localhost:6379, empty = disabled
}

type ServerConfig struct {
	Port            string
	Host            string
	ExposeSwagger   bool
	SwaggerUser     string
	SwaggerPassword string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

type AuthConfig struct {
	JWTSecretKey              string        `env:"JWT_SECRET_KEY,required"`
	JWTExpiration             time.Duration `env:"JWT_EXPIRATION" envDefault:"1h"`       // Access: 1 ч
	RefreshExpiration         time.Duration `env:"REFRESH_EXPIRATION" envDefault:"240h"` // Refresh: 10 дней
	CookieDomain              string        `env:"COOKIE_DOMAIN"`
	SecureCookies             bool          `env:"SECURE_COOKIES" envDefault:"false"`
	RegistrationTokenLifetime time.Duration `env:"REGISTRATION_TOKEN_LIFETIME" envDefault:"24h"` // Время жизни ссылки подтверждения
	VerificationBaseURL       string        `env:"VERIFICATION_BASE_URL"`                        // Базовый URL для ссылки (например https://app.example.com)
	InvitationTokenLifetime   time.Duration `env:"INVITATION_TOKEN_LIFETIME" envDefault:"168h"`  // 7 дней
}

type EmailConfig struct {
	SMTPHost     string `env:"SMTP_HOST"`
	SMTPPort     string `env:"SMTP_PORT"`
	SMTPUsername string `env:"SMTP_USERNAME"`
	SMTPPassword string `env:"SMTP_PASSWORD"`
	Enabled      bool   `env:"EMAIL_ENABLED" envDefault:"false"`
}

type TelegramConfig struct {
	// Legacy/admin bot (used by backend auth notifications).
	BotToken string `env:"TELEGRAM_BOT_TOKEN"`
	ChatID   int64  `env:"TELEGRAM_CHAT_ID"`
	// User-facing bot (used by n8n user workflows and /start links).
	UserBotToken    string `env:"TELEGRAM_USER_BOT_TOKEN"`
	UserBotUsername string `env:"TELEGRAM_USER_BOT_USERNAME"`
}

type IntegrationsConfig struct {
	InternalAPIKey       string
	TelegramLinkTokenTTL time.Duration
}

type LogsConfig struct {
	Dir string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	return &Config{
		Server: ServerConfig{
			Port:            getEnv("SERVER_PORT", ""),
			Host:            getEnv("SERVER_HOST", ""),
			ExposeSwagger:   getEnvBool("EXPOSE_SWAGGER", true),
			SwaggerUser:     getEnv("SWAGGER_USER", ""),
			SwaggerPassword: getEnv("SWAGGER_PASSWORD", ""),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", ""),
			Port:     getEnv("DB_PORT", ""),
			User:     getEnv("DB_USER", ""),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", ""),
		},
		Logs: LogsConfig{
			Dir: getEnv("LOGS_DIR", "./logs"),
		},
		Auth: AuthConfig{
			JWTSecretKey:              getEnv("JWT_SECRET_KEY", ""),
			JWTExpiration:             getEnvDuration("JWT_EXPIRATION", time.Hour),
			RefreshExpiration:         getEnvDuration("REFRESH_EXPIRATION", 240*time.Hour),
			CookieDomain:              getEnv("COOKIE_DOMAIN", ""),
			SecureCookies:             getEnvBool("SECURE_COOKIES", true),
			RegistrationTokenLifetime: getEnvDuration("REGISTRATION_TOKEN_LIFETIME", 24*time.Hour),
			VerificationBaseURL:       getEnv("VERIFICATION_BASE_URL", "http://localhost:3000"),
			InvitationTokenLifetime:   getEnvDuration("INVITATION_TOKEN_LIFETIME", 168*time.Hour),
		},
		Email: EmailConfig{
			SMTPHost:     getEnv("SMTP_HOST", ""),
			SMTPPort:     getEnv("SMTP_PORT", "587"),
			SMTPUsername: getEnv("SMTP_USERNAME", ""),
			SMTPPassword: getEnv("SMTP_PASSWORD", ""),
			Enabled:      getEnvBool("EMAIL_ENABLED", false),
		},
		Realtime: RealtimeConfig{
			RedisURL: getEnv("REDIS_URL", ""),
		},
		Uploads: UploadsConfig{
			Dir: getEnv("UPLOADS_DIR", "./uploads"),
		},
		Activity: ActivityConfig{
			Enabled: getEnvBool("ACTIVITY_LOG_ENABLED", true),
		},
		Telegram: TelegramConfig{
			BotToken:        getEnv("TELEGRAM_BOT_TOKEN", ""),
			ChatID:          getEnvInt64("TELEGRAM_CHAT_ID", 0),
			UserBotToken:    getEnv("TELEGRAM_USER_BOT_TOKEN", ""),
			UserBotUsername: getEnv("TELEGRAM_USER_BOT_USERNAME", ""),
		},
		Integrations: IntegrationsConfig{
			InternalAPIKey:       getEnv("INTERNAL_NOTIFICATIONS_API_KEY", ""),
			TelegramLinkTokenTTL: getEnvDuration("TELEGRAM_LINK_TOKEN_TTL", 15*time.Minute),
		},
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value == "true" || value == "1" || value == "yes" || value == "on"
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return duration
}

func getEnvInt64(key string, defaultValue int64) int64 {
	raw := os.Getenv(key)
	if raw == "" {
		return defaultValue
	}
	var v int64
	if _, err := fmt.Sscan(raw, &v); err != nil {
		return defaultValue
	}
	return v
}
