package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"opscore/backend/internal/auth"
	"opscore/backend/internal/config"
	"opscore/backend/internal/models"
)

func TestCopilotConnectionTestsLocalEndpointWithoutLeakingKey(t *testing.T) {
	var seenPath string
	modelService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
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

	body := `{"provider":"local","localEndpoint":"` + modelService.URL + `","localModel":"ops-test","apiKey":"sk-test-secret"}`
	req := httptest.NewRequest(http.MethodPost, "/api/copilot/test-connection", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if seenPath != "/api/generate" {
		t.Fatalf("expected local provider to call /api/generate, got %q", seenPath)
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

func TestCopilotConnectionBuildsCompatibleProbeWithAuthorization(t *testing.T) {
	req, target, err := buildCopilotProbeRequest(context.Background(), "compatible", "https://llm.example.com/v1", "ops-test", "sk-test-secret")
	if err != nil {
		t.Fatal(err)
	}
	if target != "https://llm.example.com/v1/chat/completions" {
		t.Fatalf("expected compatible target to append /chat/completions, got %q", target)
	}
	if req.Header.Get("Authorization") != "Bearer sk-test-secret" {
		t.Fatalf("expected authorization header to use submitted api key, got %q", req.Header.Get("Authorization"))
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

func TestCopilotConnectionRejectsHostedLoopbackEndpoint(t *testing.T) {
	_, err := testCopilotConnection(context.Background(), copilotConnectionRequest{
		Provider: "compatible",
		Endpoint: "http://127.0.0.1:11434",
		Model:    "ops-test",
		APIKey:   "sk-test-secret",
	})
	if err == nil || !strings.Contains(err.Error(), "loopback") {
		t.Fatalf("expected hosted loopback endpoint to be rejected, got %v", err)
	}
}

func TestCopilotConnectionRejectsMetadataEndpoint(t *testing.T) {
	_, err := testCopilotConnection(context.Background(), copilotConnectionRequest{
		Provider:      "local",
		LocalEndpoint: "http://169.254.169.254/latest/meta-data",
		LocalModel:    "ops-test",
	})
	if err == nil || !strings.Contains(err.Error(), "metadata") {
		t.Fatalf("expected metadata endpoint to be rejected, got %v", err)
	}
}

func TestCopilotConfigRejectsHostedPrivateEndpoint(t *testing.T) {
	err := validateCopilotConfig(models.CopilotConfig{
		Provider: "openai",
		Endpoint: "http://10.0.0.8:8080/v1",
		Model:    "ops-test",
	})
	if err == nil || !strings.Contains(err.Error(), "private") {
		t.Fatalf("expected hosted private endpoint config to be rejected, got %v", err)
	}
}

func TestCopilotRedirectPolicyRejectsHostedRedirectToLoopback(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8080/internal", nil)
	err := copilotRedirectPolicy("compatible")(req, nil)
	if err == nil || !strings.Contains(err.Error(), "loopback") {
		t.Fatalf("expected hosted redirect to loopback to be rejected, got %v", err)
	}
}

func TestCopilotResolvedAddressesRejectHostedPrivateDNSResult(t *testing.T) {
	err := validateCopilotResolvedAddresses("compatible", "llm.example.com", []netip.Addr{
		netip.MustParseAddr("10.0.0.9"),
	})
	if err == nil || !strings.Contains(err.Error(), "private") {
		t.Fatalf("expected hosted DNS result pointing to private address to be rejected, got %v", err)
	}
}

func TestCopilotResolvedAddressesAllowLocalPrivateDNSResult(t *testing.T) {
	err := validateCopilotResolvedAddresses("local", "ollama.internal", []netip.Addr{
		netip.MustParseAddr("192.168.1.25"),
	})
	if err != nil {
		t.Fatalf("expected local provider to allow private model address, got %v", err)
	}
}

func TestCopilotResolvedAddressesRejectMetadataForLocalProvider(t *testing.T) {
	err := validateCopilotResolvedAddresses("local", "metadata.internal", []netip.Addr{
		netip.MustParseAddr("169.254.169.254"),
	})
	if err == nil || !strings.Contains(err.Error(), "metadata") {
		t.Fatalf("expected metadata DNS result to be rejected for local provider, got %v", err)
	}
}

func TestCopilotConfigCanBeSavedAndReadWithoutLeakingAPIKey(t *testing.T) {
	store := &mutationStore{
		userProfile: models.User{ID: 1, Username: "admin", Roles: []string{auth.RoleSuperAdmin}},
	}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
	if err != nil {
		t.Fatal(err)
	}

	payload := `{"provider":"openai","endpoint":"https://api.openai.com/v1","model":"gpt-4.1","apiKey":"sk-live-secret","temperature":"0.2","maxTokens":"4096","enableAssetContext":true,"enableIncidentContext":true,"enableTaskContext":true,"enableOncallContext":true,"auditEnabled":true}`
	putReq := httptest.NewRequest(http.MethodPut, "/api/copilot/config", strings.NewReader(payload))
	putReq.Header.Set("Authorization", "Bearer "+token)
	putRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("expected copilot config save status 200, got %d: %s", putRec.Code, putRec.Body.String())
	}
	if store.copilotConfig.APIKey != "sk-live-secret" {
		t.Fatalf("expected api key to be passed to store for encrypted persistence, got %q", store.copilotConfig.APIKey)
	}
	if strings.Contains(putRec.Body.String(), "sk-live-secret") {
		t.Fatalf("save response must not leak api key: %s", putRec.Body.String())
	}
	var saved models.CopilotConfig
	if err := json.NewDecoder(putRec.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	if !saved.HasAPIKey || saved.APIKey != "" {
		t.Fatalf("expected masked saved config with hasAPIKey only, got %+v", saved)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/copilot/config", nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	getRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected copilot config get status 200, got %d: %s", getRec.Code, getRec.Body.String())
	}
	if strings.Contains(getRec.Body.String(), "sk-live-secret") {
		t.Fatalf("get response must not leak api key: %s", getRec.Body.String())
	}
}

func TestCopilotSanitizesGoogleAPIKeyFromProviderDetails(t *testing.T) {
	detail := sanitizeProviderResponse("request failed for key gemini-secret-key/with+chars and encoded gemini-secret-key%2Fwith%2Bchars", "gemini-secret-key/with+chars")
	if strings.Contains(detail, "gemini-secret-key") || strings.Contains(detail, "gemini-secret-key%2Fwith%2Bchars") {
		t.Fatalf("provider detail must not leak api key, got %q", detail)
	}
}
