package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"opscore/backend/internal/auth"
	"opscore/backend/internal/config"
	"opscore/backend/internal/models"
)

func TestMutationRequestsProduceSanitizedAuditEvents(t *testing.T) {
	store := &mutationStore{userProfile: models.User{ID: 1, Username: "admin", Roles: []string{auth.RoleSuperAdmin}}}
	signer := auth.NewSigner("test-audit-secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/tasks/42", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "OpsCore-Test")
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)

	if len(store.auditEvents) != 1 {
		t.Fatalf("expected one audit event, got %d", len(store.auditEvents))
	}
	event := store.auditEvents[0]
	if event.ActorUserID != 1 || event.ActorUsername != "admin" || event.ResourceType != "tasks" || event.ResourceID != "42" {
		t.Fatalf("unexpected audit actor/resource: %+v", event)
	}
	if event.Detail["path"] != "/api/tasks/42" || event.UserAgent != "OpsCore-Test" {
		t.Fatalf("unexpected sanitized audit detail: %+v", event)
	}
}

func TestAuditEventsRequireSuperAdminPermission(t *testing.T) {
	store := newOpsMutationStore()
	signer := auth.NewSigner("test-audit-secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(store.userProfile.ID, store.userProfile.Username, store.userProfile.Roles)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/audit-events", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected audit endpoint to reject ops engineer, got %d", rec.Code)
	}
}

func TestCopilotChatUsesDedicatedAuditActionWithoutPromptContent(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/copilot/chat", nil)
	action, resourceType, resourceID := auditDescriptor(req)
	if action != "copilot.chat" || resourceType != "copilot" || resourceID != "" {
		t.Fatalf("unexpected Copilot audit descriptor: %q %q %q", action, resourceType, resourceID)
	}
}
