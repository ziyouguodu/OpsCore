package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"opscore/backend/internal/auth"
	"opscore/backend/internal/config"
	"opscore/backend/internal/models"
)

func TestCopilotConnectionTestsCompatibleEndpointWithoutLeakingKey(t *testing.T) {
	var seenPath string
	var seenAuth string
	modelService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		seenAuth = r.Header.Get("Authorization")
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST to model service, got %s", r.Method)
		}
		writeJSON(w, http.StatusOK, map[string]any{"id": "chatcmpl-test"})
	}))
	defer modelService.Close()

	store := &mutationStore{
		userProfile: models.User{ID: 1, Username: "admin", Roles: []string{auth.RoleSuperAdmin}},
	}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
	if err != nil {
		t.Fatal(err)
	}

	body := `{"provider":"compatible","endpoint":"` + modelService.URL + `","model":"ops-test","apiKey":"sk-test-secret"}`
	req := httptest.NewRequest(http.MethodPost, "/api/copilot/test-connection", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if seenPath != "/chat/completions" {
		t.Fatalf("expected compatible provider to call /chat/completions, got %q", seenPath)
	}
	if seenAuth != "Bearer sk-test-secret" {
		t.Fatalf("expected model service authorization header to use submitted api key, got %q", seenAuth)
	}
	if strings.Contains(rec.Body.String(), "sk-test-secret") {
		t.Fatalf("response must not leak submitted api key: %s", rec.Body.String())
	}
	var payload struct {
		OK      bool   `json:"ok"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if !payload.OK {
		t.Fatalf("expected successful connection test, got %+v", payload)
	}
}

func TestCopilotConnectionRequiresAPIKeyForHostedProvider(t *testing.T) {
	store := &mutationStore{
		userProfile: models.User{ID: 1, Username: "admin", Roles: []string{auth.RoleSuperAdmin}},
	}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/copilot/test-connection", strings.NewReader(`{"provider":"openai","endpoint":"https://api.openai.com/v1","model":"gpt-4.1"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for missing hosted provider api key, got %d: %s", rec.Code, rec.Body.String())
	}
}
