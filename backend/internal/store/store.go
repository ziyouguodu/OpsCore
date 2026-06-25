package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"opscore/backend/internal/auth"
	secretcrypto "opscore/backend/internal/crypto"
	"opscore/backend/internal/models"
)

type Store struct {
	pool          *pgxpool.Pool
	credentialBox secretcrypto.SecretBox
}

var ErrForbiddenAssetDelete = errors.New("only super admin or asset creator can delete asset")

const credentialVerificationPasswordKey = "credential_verification_password_hash"
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

func Open(ctx context.Context, databaseURL string, credentialBox secretcrypto.SecretBox) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool, credentialBox: credentialBox}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) SeedDefaults(ctx context.Context, adminPassword string) error {
	if _, err := s.pool.Exec(ctx, schemaSQL); err != nil {
		return err
	}

	passwordHash, err := auth.HashPassword(adminPassword)
	if err != nil {
		return err
	}

	if _, err := s.pool.Exec(ctx, `
		insert into roles(code, name) values
			('super_admin', '超级管理员'),
			('ops_engineer', '运维工程师')
		on conflict (code) do nothing
	`); err != nil {
		return err
	}
	if _, err := s.pool.Exec(ctx, `
		insert into users(username, display_name, password_hash, must_change_password)
		values ('admin', '超级管理员', $1, true)
		on conflict (username) do nothing
	`, passwordHash); err != nil {
		return err
	}
	if _, err := s.pool.Exec(ctx, `
		insert into user_roles(user_id, role_id)
		select u.id, r.id from users u, roles r
		where u.username = 'admin' and r.code = 'super_admin'
		on conflict do nothing
	`); err != nil {
		return err
	}
	if err := s.encryptLegacyAssetCredentials(ctx); err != nil {
		return err
	}
	return s.encryptLegacyMiddlewareCredentials(ctx)
}

func (s *Store) Authenticate(ctx context.Context, username, password string) (models.User, bool, error) {
	row := s.pool.QueryRow(ctx, `select id, username, display_name, password_hash, must_change_password, created_at from users where username=$1`, username)
	var user models.User
	var passwordHash string
	if err := row.Scan(&user.ID, &user.Username, &user.DisplayName, &passwordHash, &user.MustChangePassword, &user.CreatedAt); err != nil {
		return models.User{}, false, err
	}
	if !auth.VerifyPassword(passwordHash, password) {
		return models.User{}, false, nil
	}
	roles, err := s.UserRoles(ctx, user.ID)
	if err != nil {
		return models.User{}, false, err
	}
	user.Roles = roles
	return user, true, nil
}

func (s *Store) GetUser(ctx context.Context, userID int64) (models.User, error) {
	row := s.pool.QueryRow(ctx, `select id, username, display_name, must_change_password, created_at from users where id=$1`, userID)
	var user models.User
	if err := row.Scan(&user.ID, &user.Username, &user.DisplayName, &user.MustChangePassword, &user.CreatedAt); err != nil {
		return models.User{}, err
	}
	roles, err := s.UserRoles(ctx, user.ID)
	if err != nil {
		return models.User{}, err
	}
	user.Roles = roles
	return user, nil
}

func (s *Store) VerifyUserPassword(ctx context.Context, username, password string) (bool, error) {
	row := s.pool.QueryRow(ctx, `select password_hash from users where username=$1`, username)
	var passwordHash string
	if err := row.Scan(&passwordHash); err != nil {
		return false, err
	}
	return auth.VerifyPassword(passwordHash, password), nil
}

func (s *Store) HasCredentialVerificationPassword(ctx context.Context) (bool, error) {
	row := s.pool.QueryRow(ctx, `select value from system_settings where key=$1`, credentialVerificationPasswordKey)
	var passwordHash string
	if err := row.Scan(&passwordHash); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return passwordHash != "", nil
}

func (s *Store) SetCredentialVerificationPassword(ctx context.Context, password string) error {
	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		insert into system_settings(key, value)
		values ($1, $2)
		on conflict (key) do update set value=excluded.value, updated_at=now()
	`, credentialVerificationPasswordKey, passwordHash)
	return err
}

func (s *Store) VerifyCredentialPassword(ctx context.Context, password string) (bool, error) {
	row := s.pool.QueryRow(ctx, `select value from system_settings where key=$1`, credentialVerificationPasswordKey)
	var passwordHash string
	if err := row.Scan(&passwordHash); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return auth.VerifyPassword(passwordHash, password), nil
}

func (s *Store) GetCopilotConfig(ctx context.Context) (models.CopilotConfig, error) {
	stored, err := s.readStoredCopilotConfig(ctx)
	if err != nil {
		return models.CopilotConfig{}, err
	}
	return publicCopilotConfig(stored), nil
}

func (s *Store) UpsertCopilotConfig(ctx context.Context, item models.CopilotConfig) (models.CopilotConfig, error) {
	stored, err := s.readStoredCopilotConfig(ctx)
	if err != nil {
		return models.CopilotConfig{}, err
	}
	if strings.TrimSpace(item.APIKey) != "" {
		encrypted, err := s.credentialBox.Encrypt(item.APIKey)
		if err != nil {
			return models.CopilotConfig{}, err
		}
		stored.APIKeyEncrypted = encrypted
	} else if strings.TrimSpace(item.Provider) != strings.TrimSpace(stored.Provider) || strings.TrimSpace(item.Endpoint) != strings.TrimSpace(stored.Endpoint) {
		stored.APIKeyEncrypted = ""
	}
	stored.Provider = item.Provider
	stored.Endpoint = item.Endpoint
	stored.Model = item.Model
	stored.LocalEndpoint = item.LocalEndpoint
	stored.LocalModel = item.LocalModel
	stored.Temperature = item.Temperature
	stored.MaxTokens = item.MaxTokens
	stored.EnableAssetContext = item.EnableAssetContext
	stored.EnableIncidentContext = item.EnableIncidentContext
	stored.EnableTaskContext = item.EnableTaskContext
	stored.EnableOncallContext = item.EnableOncallContext
	stored.AuditEnabled = item.AuditEnabled

	payload, err := json.Marshal(stored)
	if err != nil {
		return models.CopilotConfig{}, err
	}
	_, err = s.pool.Exec(ctx, `
		insert into system_settings(key, value)
		values ($1, $2)
		on conflict (key) do update set value=excluded.value, updated_at=now()
	`, copilotConfigKey, string(payload))
	if err != nil {
		return models.CopilotConfig{}, err
	}
	return publicCopilotConfig(stored), nil
}

func (s *Store) GetCopilotAPIKey(ctx context.Context) (string, error) {
	stored, err := s.readStoredCopilotConfig(ctx)
	if err != nil {
		return "", err
	}
	if stored.APIKeyEncrypted == "" {
		return "", nil
	}
	return s.credentialBox.Decrypt(stored.APIKeyEncrypted)
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
		MaxTokens:             "4096",
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

func (s *Store) ChangePassword(ctx context.Context, userID int64, currentPassword string, newPassword string) (models.User, error) {
	row := s.pool.QueryRow(ctx, `select username, password_hash from users where id=$1`, userID)
	var username string
	var passwordHash string
	if err := row.Scan(&username, &passwordHash); err != nil {
		return models.User{}, err
	}
	if !auth.VerifyPassword(passwordHash, currentPassword) {
		return models.User{}, errors.New("current password is invalid")
	}
	nextHash, err := auth.HashPassword(newPassword)
	if err != nil {
		return models.User{}, err
	}
	if _, err := s.pool.Exec(ctx, `update users set password_hash=$2, must_change_password=false, updated_at=now() where id=$1`, userID, nextHash); err != nil {
		return models.User{}, err
	}
	return s.GetUser(ctx, userID)
}

func (s *Store) ResetUserPassword(ctx context.Context, username string, newPassword string, mustChangePassword bool) (models.User, error) {
	passwordHash, err := auth.HashPassword(newPassword)
	if err != nil {
		return models.User{}, err
	}
	row := s.pool.QueryRow(ctx, `
		update users
		set password_hash=$2, must_change_password=$3, updated_at=now()
		where username=$1
		returning id
	`, username, passwordHash, mustChangePassword)
	var userID int64
	if err := row.Scan(&userID); err != nil {
		return models.User{}, err
	}
	return s.GetUser(ctx, userID)
}

func (s *Store) UserRoles(ctx context.Context, userID int64) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		select r.code from roles r
		join user_roles ur on ur.role_id = r.id
		where ur.user_id = $1
		order by r.code
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

func (s *Store) ListUsers(ctx context.Context) ([]models.UserListItem, error) {
	rows, err := s.pool.Query(ctx, `
		select
			u.id,
			u.username,
			u.display_name,
			u.must_change_password,
			coalesce(array_agg(r.code order by r.code) filter (where r.code is not null), '{}') as roles,
			u.created_at,
			u.updated_at
		from users u
		left join user_roles ur on ur.user_id = u.id
		left join roles r on r.id = ur.role_id
		group by u.id
		order by u.created_at desc
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.UserListItem{}
	for rows.Next() {
		var item models.UserListItem
		if err := rows.Scan(&item.ID, &item.Username, &item.DisplayName, &item.MustChangePassword, &item.Roles, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateUser(ctx context.Context, item models.UserMutation) (models.UserListItem, error) {
	if item.Username == "" || item.DisplayName == "" || item.Password == "" || len(item.Roles) == 0 {
		return models.UserListItem{}, errors.New("username, displayName, password and roles are required")
	}
	passwordHash, err := auth.HashPassword(item.Password)
	if err != nil {
		return models.UserListItem{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.UserListItem{}, err
	}
	defer tx.Rollback(ctx)

	var id int64
	if err := tx.QueryRow(ctx, `
		insert into users(username, display_name, password_hash, must_change_password)
		values ($1,$2,$3,$4)
		returning id
	`, item.Username, item.DisplayName, passwordHash, item.MustChangePassword).Scan(&id); err != nil {
		return models.UserListItem{}, err
	}
	if err := setUserRoles(ctx, tx, id, item.Roles); err != nil {
		return models.UserListItem{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return models.UserListItem{}, err
	}
	return s.userListItem(ctx, id)
}

func (s *Store) UpdateUser(ctx context.Context, id int64, item models.UserMutation) (models.UserListItem, error) {
	if item.Username == "" || item.DisplayName == "" || len(item.Roles) == 0 {
		return models.UserListItem{}, errors.New("username, displayName and roles are required")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.UserListItem{}, err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		update users set username=$2, display_name=$3, must_change_password=$4, updated_at=now()
		where id=$1
	`, id, item.Username, item.DisplayName, item.MustChangePassword)
	if err != nil {
		return models.UserListItem{}, err
	}
	if tag.RowsAffected() == 0 {
		return models.UserListItem{}, errors.New("user not found")
	}
	if item.Password != "" {
		passwordHash, err := auth.HashPassword(item.Password)
		if err != nil {
			return models.UserListItem{}, err
		}
		if _, err := tx.Exec(ctx, `update users set password_hash=$2, updated_at=now() where id=$1`, id, passwordHash); err != nil {
			return models.UserListItem{}, err
		}
	}
	if err := setUserRoles(ctx, tx, id, item.Roles); err != nil {
		return models.UserListItem{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return models.UserListItem{}, err
	}
	return s.userListItem(ctx, id)
}

func (s *Store) DeleteUser(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `delete from users where id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("user not found")
	}
	return nil
}

type roleSetter interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func setUserRoles(ctx context.Context, tx roleSetter, userID int64, roles []string) error {
	if _, err := tx.Exec(ctx, `delete from user_roles where user_id=$1`, userID); err != nil {
		return err
	}
	for _, role := range roles {
		tag, err := tx.Exec(ctx, `
			insert into user_roles(user_id, role_id)
			select $1, id from roles where code=$2
			on conflict do nothing
		`, userID, role)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("role %s not found", role)
		}
	}
	return nil
}

func (s *Store) userListItem(ctx context.Context, id int64) (models.UserListItem, error) {
	rows, err := s.pool.Query(ctx, `
		select
			u.id,
			u.username,
			u.display_name,
			u.must_change_password,
			coalesce(array_agg(r.code order by r.code) filter (where r.code is not null), '{}') as roles,
			u.created_at,
			u.updated_at
		from users u
		left join user_roles ur on ur.user_id = u.id
		left join roles r on r.id = ur.role_id
		where u.id = $1
		group by u.id
	`, id)
	if err != nil {
		return models.UserListItem{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return models.UserListItem{}, errors.New("user not found")
	}
	var item models.UserListItem
	if err := rows.Scan(&item.ID, &item.Username, &item.DisplayName, &item.MustChangePassword, &item.Roles, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return models.UserListItem{}, err
	}
	return item, rows.Err()
}

func (s *Store) ListAssets(ctx context.Context) ([]models.Asset, error) {
	rows, err := s.pool.Query(ctx, `select id, coalesce(created_by, 0), asset_no, type, vendor, cpu_arch, sn, location, business, ipv4, ipv6, environment, os, hostname, network_zone, cpu, memory, disk, deployment_info, owner, status, connected_status, host_machine, created_at, updated_at from assets order by updated_at desc`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.Asset{}
	for rows.Next() {
		var item models.Asset
		if err := rows.Scan(&item.ID, &item.CreatedBy, &item.AssetNo, &item.Type, &item.Vendor, &item.CPUArch, &item.SN, &item.Location, &item.Business, &item.IPv4, &item.IPv6, &item.Environment, &item.OS, &item.Hostname, &item.NetworkZone, &item.CPU, &item.Memory, &item.Disk, &item.DeploymentInfo, &item.Owner, &item.Status, &item.ConnectedStatus, &item.HostMachine, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) UpsertAsset(ctx context.Context, item models.Asset) (models.Asset, error) {
	if item.Type == "" || item.CPUArch == "" || item.Business == "" || item.IPv4 == "" || item.Environment == "" || item.OS == "" || item.NetworkZone == "" || item.CPU == "" || item.Memory == "" || item.Disk == "" || item.DeploymentInfo == "" || item.Owner == "" {
		return models.Asset{}, errors.New("type, cpuArch, business, ipv4, environment, os, networkZone, cpu, memory, disk, deploymentInfo and owner are required")
	}
	if item.ID > 0 {
		return s.updateAsset(ctx, item.ID, item)
	}
	if item.AssetNo == "" {
		assetNo, err := s.nextAssetNo(ctx)
		if err != nil {
			return models.Asset{}, err
		}
		item.AssetNo = assetNo
	}
	if item.Status == "" {
		item.Status = "运行中"
	}
	if item.ConnectedStatus == "" {
		item.ConnectedStatus = "已并网"
	}
	row := s.pool.QueryRow(ctx, `
		insert into assets(created_by, asset_no, type, vendor, cpu_arch, sn, location, business, ipv4, ipv6, environment, os, hostname, network_zone, cpu, memory, disk, deployment_info, owner, status, connected_status, host_machine)
		values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)
		on conflict (asset_no) do update set
			type=excluded.type, vendor=excluded.vendor, cpu_arch=excluded.cpu_arch, sn=excluded.sn, location=excluded.location,
			business=excluded.business, ipv4=excluded.ipv4, ipv6=excluded.ipv6, environment=excluded.environment, os=excluded.os,
			hostname=excluded.hostname, network_zone=excluded.network_zone, cpu=excluded.cpu, memory=excluded.memory, disk=excluded.disk,
			deployment_info=excluded.deployment_info, owner=excluded.owner, status=excluded.status,
			connected_status=excluded.connected_status, host_machine=excluded.host_machine, updated_at=now()
		returning id, created_at, updated_at
	`, nullableUserID(item.CreatedBy), item.AssetNo, item.Type, item.Vendor, item.CPUArch, item.SN, item.Location, item.Business, item.IPv4, item.IPv6, item.Environment, item.OS, item.Hostname, item.NetworkZone, item.CPU, item.Memory, item.Disk, item.DeploymentInfo, item.Owner, item.Status, item.ConnectedStatus, item.HostMachine)
	if err := row.Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return models.Asset{}, err
	}
	return item, nil
}

func (s *Store) updateAsset(ctx context.Context, id int64, item models.Asset) (models.Asset, error) {
	if item.Status == "" {
		item.Status = "运行中"
	}
	if item.ConnectedStatus == "" {
		item.ConnectedStatus = "已并网"
	}
	row := s.pool.QueryRow(ctx, `
		update assets set
			asset_no=$2, type=$3, vendor=$4, cpu_arch=$5, sn=$6, location=$7, business=$8, ipv4=$9, ipv6=$10,
			environment=$11, os=$12, hostname=$13, network_zone=$14, cpu=$15, memory=$16, disk=$17,
			deployment_info=$18, owner=$19, status=$20, connected_status=$21, host_machine=$22, updated_at=now()
		where id=$1
		returning id, created_at, updated_at
	`, id, item.AssetNo, item.Type, item.Vendor, item.CPUArch, item.SN, item.Location, item.Business, item.IPv4, item.IPv6, item.Environment, item.OS, item.Hostname, item.NetworkZone, item.CPU, item.Memory, item.Disk, item.DeploymentInfo, item.Owner, item.Status, item.ConnectedStatus, item.HostMachine)
	if err := row.Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return models.Asset{}, err
	}
	return item, nil
}

func (s *Store) DeleteAsset(ctx context.Context, id int64, actorUserID int64, actorIsSuperAdmin bool) error {
	var createdBy int64
	if err := s.pool.QueryRow(ctx, `select coalesce(created_by, 0) from assets where id=$1`, id).Scan(&createdBy); err != nil {
		return errors.New("asset not found")
	}
	if !actorIsSuperAdmin && (createdBy == 0 || createdBy != actorUserID) {
		return ErrForbiddenAssetDelete
	}
	tag, err := s.pool.Exec(ctx, `delete from assets where id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("asset not found")
	}
	return nil
}

func nullableUserID(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

func (s *Store) nextAssetNo(ctx context.Context) (string, error) {
	var seq int64
	if err := s.pool.QueryRow(ctx, `select nextval('assets_id_seq')`).Scan(&seq); err != nil {
		return "", err
	}
	return fmt.Sprintf("ASSET-%s-%04d", time.Now().Format("200601"), seq), nil
}

func (s *Store) GetAssetCredential(ctx context.Context, assetID int64) (models.AssetCredential, error) {
	row := s.pool.QueryRow(ctx, `select asset_id, login_url, username, secret, notes from asset_credentials where asset_id=$1`, assetID)
	var item models.AssetCredential
	if err := row.Scan(&item.AssetID, &item.LoginURL, &item.Username, &item.Secret, &item.Notes); err != nil {
		return models.AssetCredential{}, err
	}
	item.HasSecret = item.Secret != ""
	secret, err := s.credentialBox.Decrypt(item.Secret)
	if err != nil {
		return models.AssetCredential{}, err
	}
	item.Secret = secret
	return item, nil
}

func (s *Store) UpsertAssetCredential(ctx context.Context, item models.AssetCredential) (models.AssetCredential, error) {
	encryptedSecret, err := s.credentialBox.Encrypt(item.Secret)
	if err != nil {
		return models.AssetCredential{}, err
	}
	row := s.pool.QueryRow(ctx, `
		insert into asset_credentials(asset_id, login_url, username, secret, notes)
		values ($1,$2,$3,$4,$5)
		on conflict (asset_id) do update set
			login_url=excluded.login_url, username=excluded.username,
			secret=case when excluded.secret = '' then asset_credentials.secret else excluded.secret end,
			notes=excluded.notes, updated_at=now()
		returning asset_id, login_url, username, secret, notes
	`, item.AssetID, item.LoginURL, item.Username, encryptedSecret, item.Notes)
	if err := row.Scan(&item.AssetID, &item.LoginURL, &item.Username, &item.Secret, &item.Notes); err != nil {
		return models.AssetCredential{}, err
	}
	item.HasSecret = item.Secret != ""
	secret, err := s.credentialBox.Decrypt(item.Secret)
	if err != nil {
		return models.AssetCredential{}, err
	}
	item.Secret = secret
	return item, nil
}

func (s *Store) GetMiddlewareCredential(ctx context.Context, middlewareID int64) (models.MiddlewareCredential, error) {
	row := s.pool.QueryRow(ctx, `select middleware_id, login_url, username, secret, notes from middleware_credentials where middleware_id=$1`, middlewareID)
	var item models.MiddlewareCredential
	if err := row.Scan(&item.MiddlewareID, &item.LoginURL, &item.Username, &item.Secret, &item.Notes); err != nil {
		return models.MiddlewareCredential{}, err
	}
	item.HasSecret = item.Secret != ""
	secret, err := s.credentialBox.Decrypt(item.Secret)
	if err != nil {
		return models.MiddlewareCredential{}, err
	}
	item.Secret = secret
	return item, nil
}

func (s *Store) UpsertMiddlewareCredential(ctx context.Context, item models.MiddlewareCredential) (models.MiddlewareCredential, error) {
	encryptedSecret, err := s.credentialBox.Encrypt(item.Secret)
	if err != nil {
		return models.MiddlewareCredential{}, err
	}
	row := s.pool.QueryRow(ctx, `
		insert into middleware_credentials(middleware_id, login_url, username, secret, notes)
		values ($1,$2,$3,$4,$5)
		on conflict (middleware_id) do update set
			login_url=excluded.login_url, username=excluded.username,
			secret=case when excluded.secret = '' then middleware_credentials.secret else excluded.secret end,
			notes=excluded.notes, updated_at=now()
		returning middleware_id, login_url, username, secret, notes
	`, item.MiddlewareID, item.LoginURL, item.Username, encryptedSecret, item.Notes)
	if err := row.Scan(&item.MiddlewareID, &item.LoginURL, &item.Username, &item.Secret, &item.Notes); err != nil {
		return models.MiddlewareCredential{}, err
	}
	item.HasSecret = item.Secret != ""
	secret, err := s.credentialBox.Decrypt(item.Secret)
	if err != nil {
		return models.MiddlewareCredential{}, err
	}
	item.Secret = secret
	return item, nil
}

func (s *Store) encryptLegacyAssetCredentials(ctx context.Context) error {
	rows, err := s.pool.Query(ctx, `select asset_id, secret from asset_credentials where secret <> ''`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type legacyCredential struct {
		assetID int64
		secret  string
	}
	var items []legacyCredential
	for rows.Next() {
		var item legacyCredential
		if err := rows.Scan(&item.assetID, &item.secret); err != nil {
			return err
		}
		if !secretcrypto.IsEncryptedSecret(item.secret) {
			items = append(items, item)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, item := range items {
		encrypted, err := s.credentialBox.Encrypt(item.secret)
		if err != nil {
			return err
		}
		if _, err := s.pool.Exec(ctx, `update asset_credentials set secret=$2, updated_at=now() where asset_id=$1`, item.assetID, encrypted); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) encryptLegacyMiddlewareCredentials(ctx context.Context) error {
	rows, err := s.pool.Query(ctx, `select middleware_id, secret from middleware_credentials where secret <> ''`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type legacyCredential struct {
		middlewareID int64
		secret       string
	}
	var items []legacyCredential
	for rows.Next() {
		var item legacyCredential
		if err := rows.Scan(&item.middlewareID, &item.secret); err != nil {
			return err
		}
		if !secretcrypto.IsEncryptedSecret(item.secret) {
			items = append(items, item)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, item := range items {
		encrypted, err := s.credentialBox.Encrypt(item.secret)
		if err != nil {
			return err
		}
		if _, err := s.pool.Exec(ctx, `update middleware_credentials set secret=$2, updated_at=now() where middleware_id=$1`, item.middlewareID, encrypted); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ListMiddleware(ctx context.Context) ([]models.MiddlewareInstance, error) {
	rows, err := s.pool.Query(ctx, `select id, name, kind, version, environment, network_zone, endpoint, business, owner, status, asset_id, created_at, updated_at from middleware_instances order by updated_at desc`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.MiddlewareInstance{}
	for rows.Next() {
		var item models.MiddlewareInstance
		if err := rows.Scan(&item.ID, &item.Name, &item.Kind, &item.Version, &item.Environment, &item.NetworkZone, &item.Endpoint, &item.Business, &item.Owner, &item.Status, &item.AssetID, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateMiddleware(ctx context.Context, item models.MiddlewareInstance) (models.MiddlewareInstance, error) {
	if item.Name == "" || item.Kind == "" || item.Environment == "" || item.NetworkZone == "" || item.Endpoint == "" || item.Business == "" || item.Owner == "" {
		return models.MiddlewareInstance{}, errors.New("name, kind, environment, networkZone, endpoint, business and owner are required")
	}
	if item.Status == "" {
		item.Status = "运行中"
	}
	row := s.pool.QueryRow(ctx, `insert into middleware_instances(name, kind, version, environment, network_zone, endpoint, business, owner, status, asset_id) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) returning id, created_at, updated_at`, item.Name, item.Kind, item.Version, item.Environment, item.NetworkZone, item.Endpoint, item.Business, item.Owner, item.Status, item.AssetID)
	if err := row.Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return models.MiddlewareInstance{}, err
	}
	return item, nil
}

func (s *Store) UpdateMiddleware(ctx context.Context, id int64, item models.MiddlewareInstance) (models.MiddlewareInstance, error) {
	if item.Name == "" || item.Kind == "" || item.Environment == "" || item.NetworkZone == "" || item.Endpoint == "" || item.Business == "" || item.Owner == "" {
		return models.MiddlewareInstance{}, errors.New("name, kind, environment, networkZone, endpoint, business and owner are required")
	}
	if item.Status == "" {
		item.Status = "运行中"
	}
	row := s.pool.QueryRow(ctx, `
		update middleware_instances set
			name=$2, kind=$3, version=$4, environment=$5, network_zone=$6, endpoint=$7,
			business=$8, owner=$9, status=$10, asset_id=$11, updated_at=now()
		where id=$1
		returning id, created_at, updated_at
	`, id, item.Name, item.Kind, item.Version, item.Environment, item.NetworkZone, item.Endpoint, item.Business, item.Owner, item.Status, item.AssetID)
	if err := row.Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return models.MiddlewareInstance{}, err
	}
	return item, nil
}

func (s *Store) DeleteMiddleware(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `delete from middleware_instances where id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("middleware instance not found")
	}
	return nil
}

func (s *Store) ListOnCalls(ctx context.Context) ([]models.OnCallSchedule, error) {
	rows, err := s.pool.Query(ctx, `select id, rule_type, date_value, week_value, primary_user, backup_user, swap_from, swap_to, notes, created_at, updated_at from oncall_schedules order by date_value desc nulls last, updated_at desc`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.OnCallSchedule{}
	for rows.Next() {
		var item models.OnCallSchedule
		if err := rows.Scan(&item.ID, &item.RuleType, &item.Date, &item.Week, &item.Primary, &item.Backup, &item.SwapFrom, &item.SwapTo, &item.Notes, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateOnCall(ctx context.Context, item models.OnCallSchedule) (models.OnCallSchedule, error) {
	row := s.pool.QueryRow(ctx, `insert into oncall_schedules(rule_type, date_value, week_value, primary_user, backup_user, swap_from, swap_to, notes) values ($1,$2,$3,$4,$5,$6,$7,$8) returning id, created_at, updated_at`, item.RuleType, item.Date, item.Week, item.Primary, item.Backup, item.SwapFrom, item.SwapTo, item.Notes)
	if err := row.Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return models.OnCallSchedule{}, err
	}
	return item, nil
}

func (s *Store) UpdateOnCall(ctx context.Context, id int64, item models.OnCallSchedule) (models.OnCallSchedule, error) {
	row := s.pool.QueryRow(ctx, `
		update oncall_schedules set
			rule_type=$2, date_value=$3, week_value=$4, primary_user=$5, backup_user=$6,
			swap_from=$7, swap_to=$8, notes=$9, updated_at=now()
		where id=$1
		returning id, created_at, updated_at
	`, id, item.RuleType, item.Date, item.Week, item.Primary, item.Backup, item.SwapFrom, item.SwapTo, item.Notes)
	if err := row.Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return models.OnCallSchedule{}, err
	}
	return item, nil
}

func (s *Store) DeleteOnCall(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `delete from oncall_schedules where id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("oncall schedule not found")
	}
	return nil
}

func (s *Store) ListTasks(ctx context.Context) ([]models.Task, error) {
	rows, err := s.pool.Query(ctx, `select id, title, type, assignee, status, due_at, description, created_at, updated_at from tasks order by updated_at desc`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.Task{}
	for rows.Next() {
		var item models.Task
		if err := rows.Scan(&item.ID, &item.Title, &item.Type, &item.Assignee, &item.Status, &item.DueAt, &item.Description, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateTask(ctx context.Context, item models.Task) (models.Task, error) {
	row := s.pool.QueryRow(ctx, `insert into tasks(title, type, assignee, status, due_at, description) values ($1,$2,$3,$4,$5,$6) returning id, created_at, updated_at`, item.Title, item.Type, item.Assignee, item.Status, item.DueAt, item.Description)
	if err := row.Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return models.Task{}, err
	}
	return item, nil
}

func (s *Store) UpdateTask(ctx context.Context, id int64, item models.Task) (models.Task, error) {
	row := s.pool.QueryRow(ctx, `
		update tasks set
			title=$2, type=$3, assignee=$4, status=$5, due_at=$6, description=$7, updated_at=now()
		where id=$1
		returning id, created_at, updated_at
	`, id, item.Title, item.Type, item.Assignee, item.Status, item.DueAt, item.Description)
	if err := row.Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return models.Task{}, err
	}
	return item, nil
}

func (s *Store) DeleteTask(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `delete from tasks where id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("task not found")
	}
	return nil
}

func (s *Store) UpdateTaskStatus(ctx context.Context, id int64, status string) error {
	_, err := s.pool.Exec(ctx, `update tasks set status=$2, updated_at=now() where id=$1`, id, status)
	return err
}

func (s *Store) GetTaskStatus(ctx context.Context, id int64) (string, error) {
	var status string
	err := s.pool.QueryRow(ctx, `select status from tasks where id=$1`, id).Scan(&status)
	return status, err
}

func (s *Store) ListIncidents(ctx context.Context) ([]models.Incident, error) {
	rows, err := s.pool.Query(ctx, `select id, title, level, status, owner, business, started_at, recovered_at, summary, created_at, updated_at from incidents order by updated_at desc`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.Incident{}
	for rows.Next() {
		var item models.Incident
		if err := rows.Scan(&item.ID, &item.Title, &item.Level, &item.Status, &item.Owner, &item.Business, &item.StartedAt, &item.RecoveredAt, &item.Summary, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateIncident(ctx context.Context, item models.Incident) (models.Incident, error) {
	row := s.pool.QueryRow(ctx, `insert into incidents(title, level, status, owner, business, started_at, recovered_at, summary) values ($1,$2,$3,$4,$5,$6,$7,$8) returning id, created_at, updated_at`, item.Title, item.Level, item.Status, item.Owner, item.Business, item.StartedAt, item.RecoveredAt, item.Summary)
	if err := row.Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return models.Incident{}, err
	}
	return item, nil
}

func (s *Store) UpdateIncident(ctx context.Context, id int64, item models.Incident) (models.Incident, error) {
	row := s.pool.QueryRow(ctx, `
		update incidents set
			title=$2, level=$3, status=$4, owner=$5, business=$6, started_at=$7,
			recovered_at=$8, summary=$9, updated_at=now()
		where id=$1
		returning id, created_at, updated_at
	`, id, item.Title, item.Level, item.Status, item.Owner, item.Business, item.StartedAt, item.RecoveredAt, item.Summary)
	if err := row.Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return models.Incident{}, err
	}
	return item, nil
}

func (s *Store) DeleteIncident(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `delete from incidents where id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("incident not found")
	}
	return nil
}

func (s *Store) UpdateIncidentStatus(ctx context.Context, id int64, status string) error {
	_, err := s.pool.Exec(ctx, `update incidents set status=$2, updated_at=now() where id=$1`, id, status)
	return err
}

func (s *Store) GetIncidentStatus(ctx context.Context, id int64) (string, error) {
	var status string
	err := s.pool.QueryRow(ctx, `select status from incidents where id=$1`, id).Scan(&status)
	return status, err
}

func (s *Store) Dashboard(ctx context.Context) (models.Dashboard, error) {
	var data models.Dashboard
	data.AssetTypeCounts = map[string]int64{}
	data.IncidentLevelCounts = map[string]int64{}
	if err := s.pool.QueryRow(ctx, `select count(*) from assets`).Scan(&data.AssetCount); err != nil {
		return data, err
	}
	if err := s.pool.QueryRow(ctx, `select count(*) from oncall_schedules where date_value = current_date::text or rule_type = 'weekly'`).Scan(&data.TodayOnCallCount); err != nil {
		return data, err
	}
	if err := s.pool.QueryRow(ctx, `select count(*) from tasks where status in ('待处理','处理中','待确认')`).Scan(&data.ActiveTaskCount); err != nil {
		return data, err
	}
	if err := s.pool.QueryRow(ctx, `select count(*) from incidents where status in ('新建','处理中','已恢复')`).Scan(&data.ActiveIncidentCount); err != nil {
		return data, err
	}
	rows, err := s.pool.Query(ctx, `select type, count(*) from assets group by type`)
	if err != nil {
		return data, err
	}
	for rows.Next() {
		var key string
		var value int64
		if err := rows.Scan(&key, &value); err != nil {
			rows.Close()
			return data, err
		}
		data.AssetTypeCounts[key] = value
	}
	rows.Close()
	rows, err = s.pool.Query(ctx, `select level, count(*) from incidents group by level`)
	if err != nil {
		return data, err
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var value int64
		if err := rows.Scan(&key, &value); err != nil {
			return data, err
		}
		data.IncidentLevelCounts[key] = value
	}
	return data, rows.Err()
}
