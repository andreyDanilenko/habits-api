package config

import (
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Logs     LogsConfig
	Auth     AuthConfig
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
	JWTSecretKey      string        `env:"JWT_SECRET_KEY,required"`
	JWTExpiration     time.Duration `env:"JWT_EXPIRATION" envDefault:"24h"`
	RefreshExpiration time.Duration `env:"REFRESH_EXPIRATION" envDefault:"720h"`
	CookieDomain      string        `env:"COOKIE_DOMAIN"`
	SecureCookies     bool          `env:"SECURE_COOKIES" envDefault:"false"`
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
			JWTSecretKey:      getEnv("JWT_SECRET_KEY", ""),
			JWTExpiration:     getEnvDuration("JWT_EXPIRATION", 24*time.Hour),
			RefreshExpiration: getEnvDuration("REFRESH_EXPIRATION", 720*time.Hour),
			CookieDomain:      getEnv("COOKIE_DOMAIN", ""),
			SecureCookies:     getEnvBool("SECURE_COOKIES", true),
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
