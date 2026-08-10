package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"opscore/backend/internal/models"
)

const copilotConfigKey = "copilot_config"

type storedCopilotConfig struct {
	Provider              string `json:"provider"`
	Endpoint              string `json:"endpoint"`
	Model                 string `json:"model"`
	APIKeyEncrypted       string `json:"apiKeyEncrypted,omitempty"`
	LocalEndpoint         string `json:"localEndpoint"`
	LocalModel            string `json:"localModel"`
	Temperature           string `json:"temperature"`
	MaxTokens             string `json:"maxTokens"`
	EnableAssetContext    bool   `json:"enableAssetContext"`
	EnableIncidentContext bool   `json:"enableIncidentContext"`
	EnableTaskContext     bool   `json:"enableTaskContext"`
	EnableOncallContext   bool   `json:"enableOncallContext"`
	AuditEnabled          bool   `json:"auditEnabled"`
}

func (s *Store) GetCopilotConfig(ctx context.Context) (models.CopilotConfig, error) {
	active, err := s.activeCopilotProfileRow(ctx)
	if errors.Is(err, ErrCopilotProfileNotFound) {
		return publicCopilotConfig(defaultStoredCopilotConfig()), nil
	}
	if err != nil {
		return models.CopilotConfig{}, err
	}
	return copilotConfigFromProfile(active), nil
}

func (s *Store) UpsertCopilotConfig(ctx context.Context, item models.CopilotConfig) (models.CopilotConfig, error) {
	profile := copilotProfileFromConfig(item)
	active, err := s.GetActiveCopilotModelConfig(ctx)
	var saved models.CopilotModelConfig
	if errors.Is(err, ErrCopilotProfileNotFound) {
		profile.Name = "默认模型配置"
		saved, err = s.CreateCopilotModelConfig(ctx, profile)
	} else if err == nil {
		profile.Name = active.Name
		saved, err = s.UpdateCopilotModelConfig(ctx, active.ID, profile)
	}
	if err != nil {
		return models.CopilotConfig{}, err
	}
	return copilotConfigFromPublicProfile(saved), nil
}

func (s *Store) GetCopilotAPIKey(ctx context.Context) (string, error) {
	active, err := s.activeCopilotProfileRow(ctx)
	if errors.Is(err, ErrCopilotProfileNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return s.credentialBox.Decrypt(active.APIKeyEncrypted)
}

func copilotProfileFromConfig(item models.CopilotConfig) models.CopilotModelConfig {
	return models.CopilotModelConfig{
		Provider: item.Provider, Endpoint: item.Endpoint, Model: item.Model, APIKey: item.APIKey,
		LocalEndpoint: item.LocalEndpoint, LocalModel: item.LocalModel,
		Temperature: item.Temperature, MaxTokens: item.MaxTokens,
		EnableAssetContext: item.EnableAssetContext, EnableIncidentContext: item.EnableIncidentContext,
		EnableTaskContext: item.EnableTaskContext, EnableOncallContext: item.EnableOncallContext,
		AuditEnabled: true,
	}
}

func copilotConfigFromProfile(item copilotProfileRow) models.CopilotConfig {
	return copilotConfigFromPublicProfile(publicCopilotProfile(item))
}

func copilotConfigFromPublicProfile(item models.CopilotModelConfig) models.CopilotConfig {
	return models.CopilotConfig{
		Provider: item.Provider, Endpoint: item.Endpoint, Model: item.Model, HasAPIKey: item.HasAPIKey,
		LocalEndpoint: item.LocalEndpoint, LocalModel: item.LocalModel,
		Temperature: item.Temperature, MaxTokens: item.MaxTokens,
		EnableAssetContext: item.EnableAssetContext, EnableIncidentContext: item.EnableIncidentContext,
		EnableTaskContext: item.EnableTaskContext, EnableOncallContext: item.EnableOncallContext,
		AuditEnabled: true,
	}
}

func (s *Store) readStoredCopilotConfig(ctx context.Context) (storedCopilotConfig, error) {
	row := s.pool.QueryRow(ctx, `select value from system_settings where key=$1`, copilotConfigKey)
	var payload string
	if err := row.Scan(&payload); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return defaultStoredCopilotConfig(), nil
		}
		return storedCopilotConfig{}, err
	}
	var stored storedCopilotConfig
	if err := json.Unmarshal([]byte(payload), &stored); err != nil {
		return storedCopilotConfig{}, err
	}
	return stored, nil
}

func defaultStoredCopilotConfig() storedCopilotConfig {
	return storedCopilotConfig{
		Provider:              "openai",
		Endpoint:              "https://api.openai.com/v1",
		Model:                 "gpt-4.1",
		LocalEndpoint:         "http://host.docker.internal:11434",
		LocalModel:            "qwen2.5:7b",
		Temperature:           "0.2",
		MaxTokens:             "2048",
		EnableAssetContext:    true,
		EnableIncidentContext: true,
		EnableTaskContext:     true,
		EnableOncallContext:   true,
		AuditEnabled:          true,
	}
}

func publicCopilotConfig(stored storedCopilotConfig) models.CopilotConfig {
	return models.CopilotConfig{
		Provider:              stored.Provider,
		Endpoint:              stored.Endpoint,
		Model:                 stored.Model,
		HasAPIKey:             stored.APIKeyEncrypted != "",
		LocalEndpoint:         stored.LocalEndpoint,
		LocalModel:            stored.LocalModel,
		Temperature:           stored.Temperature,
		MaxTokens:             stored.MaxTokens,
		EnableAssetContext:    stored.EnableAssetContext,
		EnableIncidentContext: stored.EnableIncidentContext,
		EnableTaskContext:     stored.EnableTaskContext,
		EnableOncallContext:   stored.EnableOncallContext,
		AuditEnabled:          stored.AuditEnabled,
	}
}
