package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"opscore/backend/internal/auth"
	"opscore/backend/internal/config"
	"opscore/backend/internal/models"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestReadJSONRejectsUnknownFields(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"OpsCore","unexpected":true}`))
	var body struct {
		Name string `json:"name"`
	}
	if err := readJSON(req, &body); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown field error, got %v", err)
	}
}

func TestReadJSONRejectsTrailingDocuments(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"OpsCore"}{"name":"second"}`))
	var body struct {
		Name string `json:"name"`
	}
	if err := readJSON(req, &body); err == nil || !strings.Contains(err.Error(), "single JSON object") {
		t.Fatalf("expected trailing JSON error, got %v", err)
	}
}

func TestReadJSONRejectsOversizedBodies(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"value":"`+strings.Repeat("x", 1<<20)+`"}`))
	var body map[string]string
	if err := readJSON(req, &body); err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected body too large error, got %v", err)
	}
}

func TestPublicErrorHidesInternalAndDatabaseDetails(t *testing.T) {
	if got := publicError(http.StatusInternalServerError, errors.New("dial postgres at secret-host")); got != "internal server error" {
		t.Fatalf("internal error leaked details: %s", got)
	}
	databaseError := &pgconn.PgError{Code: "22001", Message: "value too long", Detail: "sensitive table detail"}
	if got := publicError(http.StatusBadRequest, databaseError); got != "request could not be processed" {
		t.Fatalf("database error leaked details: %s", got)
	}
}

func TestCORSAllowsConfiguredFrontendOrigin(t *testing.T) {
	server := &Server{cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	handler := server.cors(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected no-content preflight, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("unexpected CORS origin: %s", got)
	}
}

func TestHealthIncludesDatabaseReadiness(t *testing.T) {
	healthy := &Server{store: &mutationStore{}}
	healthyRecorder := httptest.NewRecorder()
	healthy.health(healthyRecorder, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if healthyRecorder.Code != http.StatusOK || !strings.Contains(healthyRecorder.Body.String(), `"database":"ok"`) {
		t.Fatalf("expected healthy database status, got %d: %s", healthyRecorder.Code, healthyRecorder.Body.String())
	}

	unhealthy := &Server{store: &mutationStore{pingErr: errors.New("database unavailable")}}
	unhealthyRecorder := httptest.NewRecorder()
	unhealthy.health(unhealthyRecorder, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if unhealthyRecorder.Code != http.StatusServiceUnavailable || !strings.Contains(unhealthyRecorder.Body.String(), `"database":"unavailable"`) {
		t.Fatalf("expected unhealthy database status, got %d: %s", unhealthyRecorder.Code, unhealthyRecorder.Body.String())
	}
}

func TestRequireAuthRejectsMissingToken(t *testing.T) {
	server := &Server{signer: auth.NewSigner("secret", time.Hour)}
	handler := server.requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized without token, got %d", rec.Code)
	}
}

func TestRequireAuthBlocksBusinessAPIsUntilInitialPasswordChanged(t *testing.T) {
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{
		store:  &mutationStore{userProfile: models.User{ID: 1, Username: "admin", Roles: []string{auth.RoleSuperAdmin}, MustChangePassword: true}},
		signer: signer,
	}
	token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
	if err != nil {
		t.Fatal(err)
	}
	handler := server.requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	dashboardReq := httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
	dashboardReq.Header.Set("Authorization", "Bearer "+token)
	dashboardRec := httptest.NewRecorder()
	handler.ServeHTTP(dashboardRec, dashboardReq)
	if dashboardRec.Code != http.StatusForbidden {
		t.Fatalf("expected dashboard to be blocked before password initialization, got %d", dashboardRec.Code)
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+token)
	meRec := httptest.NewRecorder()
	handler.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf("expected auth/me to remain available before password initialization, got %d", meRec.Code)
	}

	passwordReq := httptest.NewRequest(http.MethodPost, "/api/auth/password", nil)
	passwordReq.Header.Set("Authorization", "Bearer "+token)
	passwordRec := httptest.NewRecorder()
	handler.ServeHTTP(passwordRec, passwordReq)
	if passwordRec.Code != http.StatusOK {
		t.Fatalf("expected auth/password to remain available before password initialization, got %d", passwordRec.Code)
	}
}

func TestRequirePermissionUsesCurrentStoredRoles(t *testing.T) {
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{
		store: &mutationStore{userProfile: models.User{
			ID:       1,
			Username: "admin",
			Roles:    []string{auth.RoleOpsEngineer},
		}},
		signer: signer,
	}
	token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
	if err != nil {
		t.Fatal(err)
	}
	handler := server.requirePermission(auth.PermissionUserManage, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected stored role downgrade to revoke user management, got %d", rec.Code)
	}
}
