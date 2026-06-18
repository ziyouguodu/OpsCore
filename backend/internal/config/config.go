package config

import "os"

type Config struct {
	ListenAddr           string
	DatabaseURL          string
	JWTSecret            string
	CredentialKey        string
	InitialAdminPassword string
	CORSOrigin           string
}

func FromEnv() Config {
	return Config{
		ListenAddr:           env("OPSCORE_LISTEN_ADDR", ":8080"),
		DatabaseURL:          env("OPSCORE_DATABASE_URL", "postgres://opscore:opscore@localhost:5432/opscore?sslmode=disable"),
		JWTSecret:            env("OPSCORE_JWT_SECRET", "dev-change-me"),
		CredentialKey:        env("OPSCORE_CREDENTIAL_ENCRYPTION_KEY", "dev-credential-key-change-me-32-bytes"),
		InitialAdminPassword: env("OPSCORE_INITIAL_ADMIN_PASSWORD", "ChangeMe123!"),
		CORSOrigin:           env("OPSCORE_CORS_ORIGIN", "http://localhost:5173"),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
