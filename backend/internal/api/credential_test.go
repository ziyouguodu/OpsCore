package api

import (
	"testing"

	"opscore/backend/internal/models"
)

func TestCredentialResponseMasksSecretByDefault(t *testing.T) {
	credential := models.AssetCredential{
		AssetID:  12,
		LoginURL: "https://console.example",
		Username: "root",
		Secret:   "do-not-leak",
		Notes:    "break-glass",
	}

	got := credentialResponse(credential, false)

	if got.Secret != "" {
		t.Fatalf("expected masked credential response to omit secret, got %q", got.Secret)
	}
	if !got.HasSecret {
		t.Fatalf("expected masked credential response to expose that a secret exists")
	}
}

func TestCredentialResponseRevealsSecretOnlyWhenRequested(t *testing.T) {
	credential := models.AssetCredential{Secret: "visible-after-password"}

	got := credentialResponse(credential, true)

	if got.Secret != "visible-after-password" {
		t.Fatalf("expected revealed credential response to include secret")
	}
	if !got.HasSecret {
		t.Fatalf("expected revealed credential response to keep hasSecret true")
	}
}
