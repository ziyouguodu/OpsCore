package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"opscore/backend/internal/auth"
	"opscore/backend/internal/config"
	"opscore/backend/internal/models"
)

func TestDutyCenterCanBeReadByOpsEngineer(t *testing.T) {
	store := newOpsMutationStore()
	store.dutyCenter = models.DutyCenterState{Revision: 3, Data: models.DutyCenterData{Assignments: map[string]models.DutyAssignment{}}}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, _ := signer.Issue(7, "ops.li", []string{auth.RoleOpsEngineer})
	req := httptest.NewRequest(http.MethodGet, "/api/duty-center", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"revision":3`) {
		t.Fatalf("expected persisted duty state, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDutyCenterWriteRequiresSuperAdmin(t *testing.T) {
	store := newOpsMutationStore()
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, _ := signer.Issue(7, "ops.li", []string{auth.RoleOpsEngineer})
	req := httptest.NewRequest(http.MethodPut, "/api/duty-center", strings.NewReader(`{"revision":0,"data":{"teams":[],"members":[],"schedules":[],"assignments":{},"currentPeople":[],"handovers":[],"escalation":{"name":"","team":"","severity":"","levels":[]}}}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected ops engineer write to be forbidden, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDutyCenterReturnsRevisionConflict(t *testing.T) {
	store := &mutationStore{
		userProfile:   models.User{ID: 1, Username: "admin", Roles: []string{auth.RoleSuperAdmin}},
		dutyCenterErr: errDutyRevisionConflict,
	}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, _ := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
	req := httptest.NewRequest(http.MethodPut, "/api/duty-center", strings.NewReader(`{"revision":1,"data":{"teams":[],"members":[],"schedules":[],"assignments":{},"currentPeople":[],"handovers":[],"escalation":{"name":"","team":"","severity":"","levels":[]}}}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected duty revision conflict, got %d: %s", rec.Code, rec.Body.String())
	}
}
