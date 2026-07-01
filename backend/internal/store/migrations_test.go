package store

import "testing"

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
