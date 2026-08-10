package store

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"opscore/backend/internal/models"
)

var ErrCopilotProfileNotFound = errors.New("copilot model configuration not found")
var ErrCopilotActiveProfileDelete = errors.New("active copilot model configuration cannot be deleted")
var ErrCopilotProfileKeyRequired = errors.New("hosted copilot model configuration requires a managed API key before activation")

const copilotProfileColumns = `
	id, name, provider, endpoint, model, api_key_encrypted, local_endpoint, local_model,
	temperature, max_tokens, enable_asset_context, enable_incident_context,
	enable_task_context, enable_oncall_context, audit_enabled, is_active, created_at, updated_at
`

type copilotProfileRow struct {
	ID                    int64
	Name                  string
	Provider              string
	Endpoint              string
	Model                 string
	APIKeyEncrypted       string
	LocalEndpoint         string
	LocalModel            string
	Temperature           float64
	MaxTokens             int
	EnableAssetContext    bool
	EnableIncidentContext bool
	EnableTaskContext     bool
	EnableOncallContext   bool
	AuditEnabled          bool
	IsActive              bool
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type copilotRowScanner interface {
	Scan(...any) error
}

func scanCopilotProfile(row copilotRowScanner) (copilotProfileRow, error) {
	var item copilotProfileRow
	err := row.Scan(
		&item.ID, &item.Name, &item.Provider, &item.Endpoint, &item.Model, &item.APIKeyEncrypted,
		&item.LocalEndpoint, &item.LocalModel, &item.Temperature, &item.MaxTokens,
		&item.EnableAssetContext, &item.EnableIncidentContext, &item.EnableTaskContext,
		&item.EnableOncallContext, &item.AuditEnabled, &item.IsActive, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func publicCopilotProfile(item copilotProfileRow) models.CopilotModelConfig {
	return models.CopilotModelConfig{
		ID:                    item.ID,
		Name:                  item.Name,
		Provider:              item.Provider,
		Endpoint:              item.Endpoint,
		Model:                 item.Model,
		HasAPIKey:             item.APIKeyEncrypted != "",
		LocalEndpoint:         item.LocalEndpoint,
		LocalModel:            item.LocalModel,
		Temperature:           strconv.FormatFloat(item.Temperature, 'f', -1, 64),
		MaxTokens:             strconv.Itoa(item.MaxTokens),
		EnableAssetContext:    item.EnableAssetContext,
		EnableIncidentContext: item.EnableIncidentContext,
		EnableTaskContext:     item.EnableTaskContext,
		EnableOncallContext:   item.EnableOncallContext,
		AuditEnabled:          item.AuditEnabled,
		IsActive:              item.IsActive,
		CreatedAt:             item.CreatedAt,
		UpdatedAt:             item.UpdatedAt,
	}
}

func normalizeCopilotProfile(item models.CopilotModelConfig) models.CopilotModelConfig {
	item.Name = strings.TrimSpace(item.Name)
	item.Provider = strings.ToLower(strings.TrimSpace(item.Provider))
	item.Endpoint = strings.TrimRight(strings.TrimSpace(item.Endpoint), "/")
	item.Model = strings.TrimSpace(item.Model)
	item.APIKey = strings.TrimSpace(item.APIKey)
	item.LocalEndpoint = strings.TrimRight(strings.TrimSpace(item.LocalEndpoint), "/")
	item.LocalModel = strings.TrimSpace(item.LocalModel)
	item.Temperature = strings.TrimSpace(item.Temperature)
	item.MaxTokens = strings.TrimSpace(item.MaxTokens)
	if item.Temperature == "" {
		item.Temperature = "0.2"
	}
	if item.MaxTokens == "" {
		item.MaxTokens = "2048"
	}
	item.AuditEnabled = true
	if item.Provider == "local" {
		item.Endpoint = ""
		item.Model = ""
		item.APIKey = ""
	} else {
		item.LocalEndpoint = ""
		item.LocalModel = ""
	}
	return item
}

func shouldClearCopilotProfileKey(existing, next models.CopilotModelConfig) bool {
	return next.Provider == "local" ||
		strings.TrimSpace(existing.Provider) != strings.TrimSpace(next.Provider) ||
		strings.TrimRight(strings.TrimSpace(existing.Endpoint), "/") != strings.TrimRight(strings.TrimSpace(next.Endpoint), "/")
}

func (s *Store) ListCopilotModelConfigs(ctx context.Context) ([]models.CopilotModelConfig, error) {
	rows, err := s.pool.Query(ctx, `select `+copilotProfileColumns+` from copilot_model_configs order by is_active desc, updated_at desc, id desc`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]models.CopilotModelConfig, 0)
	for rows.Next() {
		item, err := scanCopilotProfile(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, publicCopilotProfile(item))
	}
	return items, rows.Err()
}

func (s *Store) GetCopilotModelConfig(ctx context.Context, id int64) (models.CopilotModelConfig, error) {
	item, err := scanCopilotProfile(s.pool.QueryRow(ctx, `select `+copilotProfileColumns+` from copilot_model_configs where id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return models.CopilotModelConfig{}, ErrCopilotProfileNotFound
	}
	return publicCopilotProfile(item), err
}

func (s *Store) CreateCopilotModelConfig(ctx context.Context, input models.CopilotModelConfig) (models.CopilotModelConfig, error) {
	item := normalizeCopilotProfile(input)
	temperature, _ := strconv.ParseFloat(item.Temperature, 64)
	maxTokens, _ := strconv.Atoi(item.MaxTokens)
	encryptedKey := ""
	if item.APIKey != "" {
		var err error
		encryptedKey, err = s.credentialBox.Encrypt(item.APIKey)
		if err != nil {
			return models.CopilotModelConfig{}, err
		}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.CopilotModelConfig{}, err
	}
	defer tx.Rollback(ctx)
	var activate bool
	if err := tx.QueryRow(ctx, `select not exists(select 1 from copilot_model_configs)`).Scan(&activate); err != nil {
		return models.CopilotModelConfig{}, err
	}
	if activate && item.Provider != "local" && encryptedKey == "" {
		return models.CopilotModelConfig{}, ErrCopilotProfileKeyRequired
	}
	created, err := scanCopilotProfile(tx.QueryRow(ctx, `
		insert into copilot_model_configs(
			name, provider, endpoint, model, api_key_encrypted, local_endpoint, local_model,
			temperature, max_tokens, enable_asset_context, enable_incident_context,
			enable_task_context, enable_oncall_context, audit_enabled, is_active
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,true,$14)
		returning `+copilotProfileColumns,
		item.Name, item.Provider, item.Endpoint, item.Model, encryptedKey, item.LocalEndpoint, item.LocalModel,
		temperature, maxTokens, item.EnableAssetContext, item.EnableIncidentContext,
		item.EnableTaskContext, item.EnableOncallContext, activate,
	))
	if err != nil {
		return models.CopilotModelConfig{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return models.CopilotModelConfig{}, err
	}
	return publicCopilotProfile(created), nil
}

func (s *Store) UpdateCopilotModelConfig(ctx context.Context, id int64, input models.CopilotModelConfig) (models.CopilotModelConfig, error) {
	item := normalizeCopilotProfile(input)
	temperature, _ := strconv.ParseFloat(item.Temperature, 64)
	maxTokens, _ := strconv.Atoi(item.MaxTokens)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.CopilotModelConfig{}, err
	}
	defer tx.Rollback(ctx)
	existingRow, err := scanCopilotProfile(tx.QueryRow(ctx, `select `+copilotProfileColumns+` from copilot_model_configs where id=$1 for update`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return models.CopilotModelConfig{}, ErrCopilotProfileNotFound
	}
	if err != nil {
		return models.CopilotModelConfig{}, err
	}
	existing := publicCopilotProfile(existingRow)
	encryptedKey := existingRow.APIKeyEncrypted
	if item.APIKey != "" {
		encryptedKey, err = s.credentialBox.Encrypt(item.APIKey)
		if err != nil {
			return models.CopilotModelConfig{}, err
		}
	} else if shouldClearCopilotProfileKey(existing, item) {
		encryptedKey = ""
	}
	if existing.IsActive && item.Provider != "local" && encryptedKey == "" {
		return models.CopilotModelConfig{}, ErrCopilotProfileKeyRequired
	}
	updated, err := scanCopilotProfile(tx.QueryRow(ctx, `
		update copilot_model_configs set
			name=$2, provider=$3, endpoint=$4, model=$5, api_key_encrypted=$6,
			local_endpoint=$7, local_model=$8, temperature=$9, max_tokens=$10,
			enable_asset_context=$11, enable_incident_context=$12,
			enable_task_context=$13, enable_oncall_context=$14, audit_enabled=true, updated_at=now()
		where id=$1 returning `+copilotProfileColumns,
		id, item.Name, item.Provider, item.Endpoint, item.Model, encryptedKey,
		item.LocalEndpoint, item.LocalModel, temperature, maxTokens,
		item.EnableAssetContext, item.EnableIncidentContext, item.EnableTaskContext, item.EnableOncallContext,
	))
	if err != nil {
		return models.CopilotModelConfig{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return models.CopilotModelConfig{}, err
	}
	return publicCopilotProfile(updated), nil
}

func (s *Store) DeleteCopilotModelConfig(ctx context.Context, id int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var active bool
	if err := tx.QueryRow(ctx, `select is_active from copilot_model_configs where id=$1 for update`, id).Scan(&active); errors.Is(err, pgx.ErrNoRows) {
		return ErrCopilotProfileNotFound
	} else if err != nil {
		return err
	}
	if active {
		return ErrCopilotActiveProfileDelete
	}
	if _, err := tx.Exec(ctx, `delete from copilot_model_configs where id=$1`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) ActivateCopilotModelConfig(ctx context.Context, id int64) (models.CopilotModelConfig, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.CopilotModelConfig{}, err
	}
	defer tx.Rollback(ctx)
	target, err := scanCopilotProfile(tx.QueryRow(ctx, `select `+copilotProfileColumns+` from copilot_model_configs where id=$1 for update`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return models.CopilotModelConfig{}, ErrCopilotProfileNotFound
	}
	if err != nil {
		return models.CopilotModelConfig{}, err
	}
	if target.Provider != "local" && target.APIKeyEncrypted == "" {
		return models.CopilotModelConfig{}, ErrCopilotProfileKeyRequired
	}
	if _, err := tx.Exec(ctx, `update copilot_model_configs set is_active=false, updated_at=now() where is_active and id<>$1`, id); err != nil {
		return models.CopilotModelConfig{}, err
	}
	activated, err := scanCopilotProfile(tx.QueryRow(ctx, `update copilot_model_configs set is_active=true, updated_at=now() where id=$1 returning `+copilotProfileColumns, id))
	if err != nil {
		return models.CopilotModelConfig{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return models.CopilotModelConfig{}, err
	}
	return publicCopilotProfile(activated), nil
}

func (s *Store) activeCopilotProfileRow(ctx context.Context) (copilotProfileRow, error) {
	item, err := scanCopilotProfile(s.pool.QueryRow(ctx, `select `+copilotProfileColumns+` from copilot_model_configs where is_active`))
	if errors.Is(err, pgx.ErrNoRows) {
		return copilotProfileRow{}, ErrCopilotProfileNotFound
	}
	return item, err
}

func (s *Store) GetActiveCopilotModelConfig(ctx context.Context) (models.CopilotModelConfig, error) {
	item, err := s.activeCopilotProfileRow(ctx)
	if err != nil {
		return models.CopilotModelConfig{}, err
	}
	return publicCopilotProfile(item), nil
}

func (s *Store) GetCopilotModelAPIKey(ctx context.Context, id int64) (string, error) {
	var encrypted string
	if err := s.pool.QueryRow(ctx, `select api_key_encrypted from copilot_model_configs where id=$1`, id).Scan(&encrypted); errors.Is(err, pgx.ErrNoRows) {
		return "", ErrCopilotProfileNotFound
	} else if err != nil {
		return "", err
	}
	return s.credentialBox.Decrypt(encrypted)
}
