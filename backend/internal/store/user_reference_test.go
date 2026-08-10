package store

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

type displayNameRow struct {
	name string
	err  error
}

func (r displayNameRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	*(dest[0].(*string)) = r.name
	return nil
}

type displayNameQuerier struct{ row displayNameRow }

func (q displayNameQuerier) QueryRow(context.Context, string, ...any) pgx.Row { return q.row }

func TestResolveUserDisplayNameUsesDatabaseIdentity(t *testing.T) {
	id := int64(7)
	name, err := resolveUserDisplayName(context.Background(), displayNameQuerier{row: displayNameRow{name: "李明"}}, &id, "伪造姓名")
	if err != nil || name != "李明" {
		t.Fatalf("expected canonical database display name, got %q, %v", name, err)
	}
}

func TestResolveUserDisplayNameAllowsUnassignedFallback(t *testing.T) {
	name, err := resolveUserDisplayName(context.Background(), displayNameQuerier{}, nil, "待分派")
	if err != nil || name != "待分派" {
		t.Fatalf("expected unassigned fallback, got %q, %v", name, err)
	}
}

func TestResolveUserDisplayNameRejectsUnknownUser(t *testing.T) {
	id := int64(404)
	_, err := resolveUserDisplayName(context.Background(), displayNameQuerier{row: displayNameRow{err: errors.New("not found")}}, &id, "伪造姓名")
	if err == nil {
		t.Fatal("expected unknown user reference to fail")
	}
}
