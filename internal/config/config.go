package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config загружается из переменных окружения (или .env, если кто-то загрузит).
type Config struct {
	AppEnv  string
	AppPort int
	BaseURL string

	DatabaseURL string

	JWTSecret   string
	JWTTTLHours int

	CORSAllowedOrigins []string

	StaticPhotosDir string

	AdminEmail    string
	AdminPassword string
	AdminFullName string
}

// Load читает переменные окружения. Возвращает ошибку, если чего-то критичного не хватает.
func Load() (*Config, error) {
	c := &Config{
		AppEnv:             getEnv("APP_ENV", "dev"),
		BaseURL:            getEnv("APP_BASE_URL", "http://localhost:5173"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		StaticPhotosDir:    getEnv("STATIC_PHOTOS_DIR", "./static/photos"),
		AdminEmail:         getEnv("ADMIN_EMAIL", "admin@reshka.local"),
		AdminPassword:      getEnv("ADMIN_PASSWORD", ""),
		AdminFullName:      getEnv("ADMIN_FULL_NAME", "Администратор"),
		CORSAllowedOrigins: splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173")),
	}

	port, err := strconv.Atoi(getEnv("APP_PORT", "8080"))
	if err != nil {
		return nil, fmt.Errorf("APP_PORT: %w", err)
	}
	c.AppPort = port

	ttl, err := strconv.Atoi(getEnv("JWT_TTL_HOURS", "24"))
	if err != nil {
		return nil, fmt.Errorf("JWT_TTL_HOURS: %w", err)
	}
	c.JWTTTLHours = ttl

	if c.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	if len(c.JWTSecret) < 16 {
		return nil, errors.New("JWT_SECRET must be at least 16 characters")
	}

	return c, nil
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
