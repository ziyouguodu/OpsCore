package config

import (
	"errors"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Environment          string
	ListenAddr           string
	DatabaseURL          string
	JWTSecret            string
	CredentialKey        string
	InitialAdminPassword string
	CORSOrigin           string
	TrustProxy           bool
}

func FromEnv() Config {
	return Config{
		Environment:          env("OPSCORE_ENV", "development"),
		ListenAddr:           env("OPSCORE_LISTEN_ADDR", ":8080"),
		DatabaseURL:          env("OPSCORE_DATABASE_URL", "postgres://opscore:opscore@localhost:5432/opscore?sslmode=disable"),
		JWTSecret:            env("OPSCORE_JWT_SECRET", "dev-change-me"),
		CredentialKey:        env("OPSCORE_CREDENTIAL_ENCRYPTION_KEY", "dev-credential-key-change-me-32-bytes"),
		InitialAdminPassword: env("OPSCORE_INITIAL_ADMIN_PASSWORD", "ChangeMe123!"),
		CORSOrigin:           env("OPSCORE_CORS_ORIGIN", "http://localhost:5173"),
		TrustProxy:           envBool("OPSCORE_TRUST_PROXY", false),
	}
}

func (c Config) Validate() error {
	if !strings.EqualFold(c.Environment, "production") {
		return nil
	}
	if len(c.JWTSecret) < 32 || isPlaceholder(c.JWTSecret) {
		return errors.New("OPSCORE_JWT_SECRET must be a non-placeholder value of at least 32 bytes in production")
	}
	if len(c.CredentialKey) < 32 || isPlaceholder(c.CredentialKey) {
		return errors.New("OPSCORE_CREDENTIAL_ENCRYPTION_KEY must be a non-placeholder value of at least 32 bytes in production")
	}
	if len(c.InitialAdminPassword) < 12 || isPlaceholder(c.InitialAdminPassword) {
		return errors.New("OPSCORE_INITIAL_ADMIN_PASSWORD must be a non-placeholder value of at least 12 characters in production")
	}
	databaseURL, err := url.Parse(c.DatabaseURL)
	if err != nil {
		return errors.New("OPSCORE_DATABASE_URL is invalid")
	}
	if password, ok := databaseURL.User.Password(); !ok || isPlaceholder(password) {
		return errors.New("OPSCORE_DATABASE_URL must contain a non-placeholder database password in production")
	}
	return nil
}

func isPlaceholder(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	return normalized == "" ||
		strings.Contains(normalized, "changeme") ||
		strings.Contains(normalized, "change-me") ||
		strings.Contains(normalized, "change-this") ||
		normalized == "opscore" ||
		normalized == "dev-change-me" ||
		strings.HasPrefix(normalized, "dev-")
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
