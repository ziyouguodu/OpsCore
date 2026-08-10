package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"opscore/backend/internal/auth"
	secretcrypto "opscore/backend/internal/crypto"
	"opscore/backend/internal/models"
)

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
