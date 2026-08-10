package config

import "testing"

func TestProductionConfigRejectsPlaceholderSecrets(t *testing.T) {
	cfg := Config{
		Environment:          "production",
		DatabaseURL:          "postgres://opscore:opscore@postgres:5432/opscore",
		JWTSecret:            "change-this-in-production",
		CredentialKey:        "change-this-credential-key-32bytes",
		InitialAdminPassword: "ChangeMe123!",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected production placeholders to be rejected")
	}
}

func TestProductionConfigAcceptsStrongSecrets(t *testing.T) {
	cfg := Config{
		Environment:          "production",
		DatabaseURL:          "postgres://opscore:strong-database-password@postgres:5432/opscore",
		JWTSecret:            "a-strong-jwt-secret-with-more-than-32-bytes",
		CredentialKey:        "a-strong-credential-key-with-32-bytes-minimum",
		InitialAdminPassword: "A-strong-admin-password-2026!",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected production config to pass: %v", err)
	}
}

func TestDevelopmentConfigKeepsLocalDefaultsAvailable(t *testing.T) {
	cfg := Config{Environment: "development"}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("development defaults should remain available: %v", err)
	}
}

func TestTrustProxyIsOptInOutsideProductionCompose(t *testing.T) {
	t.Setenv("OPSCORE_TRUST_PROXY", "true")
	if !FromEnv().TrustProxy {
		t.Fatal("expected explicit proxy trust setting to be parsed")
	}
	t.Setenv("OPSCORE_TRUST_PROXY", "invalid")
	if FromEnv().TrustProxy {
		t.Fatal("invalid proxy trust setting must fall back to false")
	}
}
