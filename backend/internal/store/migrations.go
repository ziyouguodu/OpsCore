package store

import (
	"context"
	"fmt"
)

type schemaMigration struct {
	version string
	sql     string
}

var schemaMigrations = []schemaMigration{
	{version: "001_initial", sql: schemaSQL},
	{version: "002_duty_center", sql: dutyCenterSchemaSQL},
	{version: "003_remove_connected_status", sql: removeConnectedStatusSchemaSQL},
}

func (s *Store) runMigrations(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, `
		create table if not exists schema_migrations (
			version text primary key,
			applied_at timestamptz not null default now()
		)
	`); err != nil {
		return fmt.Errorf("create schema migrations table: %w", err)
	}

	for _, migration := range schemaMigrations {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return err
		}
		var applied bool
		if err := tx.QueryRow(ctx, `select exists(select 1 from schema_migrations where version=$1)`, migration.version).Scan(&applied); err != nil {
			tx.Rollback(ctx)
			return err
		}
		if applied {
			tx.Rollback(ctx)
			continue
		}
		if _, err := tx.Exec(ctx, migration.sql); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("apply migration %s: %w", migration.version, err)
		}
		if _, err := tx.Exec(ctx, `insert into schema_migrations(version) values ($1)`, migration.version); err != nil {
			tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}
