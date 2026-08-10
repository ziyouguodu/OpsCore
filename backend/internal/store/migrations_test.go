package store

import (
	"strings"
	"testing"
)

func TestSchemaMigrationsAreOrderedAndUnique(t *testing.T) {
	seen := map[string]bool{}
	previous := ""
	for _, migration := range schemaMigrations {
		if migration.version <= previous {
			t.Fatalf("migration %s is not ordered after %s", migration.version, previous)
		}
		if seen[migration.version] {
			t.Fatalf("duplicate migration %s", migration.version)
		}
		if migration.sql == "" {
			t.Fatalf("migration %s has no SQL", migration.version)
		}
		seen[migration.version] = true
		previous = migration.version
	}
}

func TestRelationalDutyMigrationDefinesQueryableTables(t *testing.T) {
	for _, table := range []string{"duty_teams", "duty_members", "duty_schedule_templates", "duty_assignments", "duty_current_people", "duty_handovers", "duty_escalation_policies"} {
		if !strings.Contains(relationalDutyCenterSchemaSQL, "create table "+table) {
			t.Fatalf("relational duty migration is missing %s", table)
		}
	}
}

func TestTrigramMigrationCoversEveryPagedKeywordList(t *testing.T) {
	for _, index := range []string{"assets_search_trgm_idx", "middleware_search_trgm_idx", "tasks_search_trgm_idx", "incidents_search_trgm_idx"} {
		if !strings.Contains(trigramSearchIndexesSchemaSQL, index) {
			t.Fatalf("trigram search migration is missing %s", index)
		}
	}
}

func TestCopilotModelProfilesMigrationDefinesLifecycleConstraints(t *testing.T) {
	if !strings.Contains(copilotModelProfilesSchemaSQL, "create table copilot_model_configs") {
		t.Fatal("copilot model profiles migration is missing its table")
	}
	for _, fragment := range []string{
		"copilot_model_configs_active_unique_idx",
		"where is_active",
		"copilot_model_configs_name_unique_idx",
		"lower(name)",
		"system_settings",
		"apiKeyEncrypted",
		"默认模型配置",
	} {
		if !strings.Contains(copilotModelProfilesSchemaSQL, fragment) {
			t.Fatalf("copilot model profiles migration is missing %q", fragment)
		}
	}
	if schemaMigrations[len(schemaMigrations)-1].version != "009_copilot_model_configs" {
		t.Fatalf("expected latest migration to be 009_copilot_model_configs, got %s", schemaMigrations[len(schemaMigrations)-1].version)
	}
}
