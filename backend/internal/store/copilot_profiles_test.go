package store

import (
	"testing"

	"opscore/backend/internal/models"
)

func TestNormalizeCopilotProfileSeparatesLocalAndHostedFields(t *testing.T) {
	local := normalizeCopilotProfile(models.CopilotModelConfig{
		Provider:      "local",
		Endpoint:      "https://should-clear.example/v1",
		Model:         "should-clear",
		APIKey:        "should-clear",
		LocalEndpoint: " http://host.docker.internal:11434/ ",
		LocalModel:    " qwen2.5:7b ",
	})
	if local.Endpoint != "" || local.Model != "" || local.APIKey != "" {
		t.Fatalf("local profile retained hosted fields: %+v", local)
	}
	if local.LocalEndpoint != "http://host.docker.internal:11434" || local.LocalModel != "qwen2.5:7b" {
		t.Fatalf("local profile was not normalized: %+v", local)
	}

	hosted := normalizeCopilotProfile(models.CopilotModelConfig{
		Provider:      "openai",
		Endpoint:      " https://api.openai.com/v1/ ",
		Model:         " gpt-4.1 ",
		LocalEndpoint: "http://should-clear",
		LocalModel:    "should-clear",
	})
	if hosted.LocalEndpoint != "" || hosted.LocalModel != "" {
		t.Fatalf("hosted profile retained local fields: %+v", hosted)
	}
	if hosted.Endpoint != "https://api.openai.com/v1" || hosted.Model != "gpt-4.1" {
		t.Fatalf("hosted profile was not normalized: %+v", hosted)
	}
}

func TestCopilotProfileKeyRetentionPolicy(t *testing.T) {
	existing := models.CopilotModelConfig{Provider: "openai", Endpoint: "https://api.openai.com/v1"}
	if shouldClearCopilotProfileKey(existing, models.CopilotModelConfig{Provider: "openai", Endpoint: "https://api.openai.com/v1", Model: "gpt-4.1"}) {
		t.Fatal("model-only update must retain the hosted key")
	}
	if !shouldClearCopilotProfileKey(existing, models.CopilotModelConfig{Provider: "anthropic", Endpoint: "https://api.anthropic.com"}) {
		t.Fatal("provider change must clear the hosted key")
	}
	if !shouldClearCopilotProfileKey(existing, models.CopilotModelConfig{Provider: "openai", Endpoint: "https://gateway.example/v1"}) {
		t.Fatal("endpoint change must clear the hosted key")
	}
	if !shouldClearCopilotProfileKey(existing, models.CopilotModelConfig{Provider: "local"}) {
		t.Fatal("switching to local must clear the hosted key")
	}
}

func TestPublicCopilotProfileMasksSecrets(t *testing.T) {
	item := publicCopilotProfile(copilotProfileRow{APIKeyEncrypted: "enc:v1:ciphertext"})
	if item.APIKey != "" || !item.HasAPIKey {
		t.Fatalf("public profile leaked or failed to mask its key: %+v", item)
	}
}
