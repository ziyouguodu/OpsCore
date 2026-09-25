package api

import (
	"context"
	"encoding/json"
	"io"
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

func TestCopilotConnectionUsesGenerationRequestTimeout(t *testing.T) {
	client := newCopilotHTTPClient("compatible")
	if client.Timeout != 45*time.Second {
		t.Fatalf("expected connection test timeout to match the 45-second generation timeout, got %s", client.Timeout)
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

func TestCopilotConfigRejectsInvalidGenerationBounds(t *testing.T) {
	for _, item := range []models.CopilotConfig{
		{Provider: "openai", Endpoint: "https://api.openai.com/v1", Model: "gpt-4.1", Temperature: "2.5"},
		{Provider: "openai", Endpoint: "https://api.openai.com/v1", Model: "gpt-4.1", MaxTokens: "5000"},
	} {
		if err := validateCopilotConfig(item); err == nil {
			t.Fatalf("expected invalid generation bounds to be rejected: %+v", item)
		}
	}
}

func TestCopilotEndpointRejectsEmbeddedCredentialsAndQuery(t *testing.T) {
	for _, endpoint := range []string{"https://user:secret@llm.example.com/v1", "https://llm.example.com/v1?key=secret"} {
		if _, err := normalizeHTTPBase(endpoint); err == nil {
			t.Fatalf("expected unsafe endpoint to be rejected: %s", endpoint)
		}
	}
}

func TestCopilotRedirectPolicyRejectsHostedRedirectToLoopback(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8080/internal", nil)
	err := copilotRedirectPolicy("compatible")(req, nil)
	if err == nil || !strings.Contains(err.Error(), "loopback") {
		t.Fatalf("expected hosted redirect to loopback to be rejected, got %v", err)
	}
}

func TestCopilotRedirectPolicyRejectsCrossHostRedirect(t *testing.T) {
	original := httptest.NewRequest(http.MethodPost, "https://api.example.com/v1/chat/completions", nil)
	redirect := httptest.NewRequest(http.MethodGet, "https://capture.example.net/redirect", nil)
	err := copilotRedirectPolicy("compatible")(redirect, []*http.Request{original})
	if err == nil || !strings.Contains(err.Error(), "cross-host") {
		t.Fatalf("expected cross-host model redirect to be rejected, got %v", err)
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

func TestCopilotModelProfilesListIsMaskedAndRestricted(t *testing.T) {
	store := &mutationStore{
		userProfile: models.User{ID: 1, Username: "admin", Roles: []string{auth.RoleSuperAdmin}},
		copilotProfiles: []models.CopilotModelConfig{{
			ID: 1, Name: "生产分析模型", Provider: "openai", Model: "gpt-4.1",
			APIKey: "must-not-leak", HasAPIKey: true, IsActive: true,
		}},
	}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	adminToken, _ := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})

	req := httptest.NewRequest(http.MethodGet, "/api/copilot/configs", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected profile list status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "must-not-leak") || !strings.Contains(rec.Body.String(), `"hasApiKey":true`) {
		t.Fatalf("profile list must expose only managed-key state: %s", rec.Body.String())
	}

	store.userProfile = models.User{ID: 7, Username: "ops.li", Roles: []string{auth.RoleOpsEngineer}}
	opsToken, _ := signer.Issue(7, "ops.li", []string{auth.RoleOpsEngineer})
	denied := httptest.NewRequest(http.MethodGet, "/api/copilot/configs", nil)
	denied.Header.Set("Authorization", "Bearer "+opsToken)
	deniedRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(deniedRec, denied)
	if deniedRec.Code != http.StatusForbidden {
		t.Fatalf("expected ops engineer to be denied, got %d", deniedRec.Code)
	}
}

func TestCopilotModelProfileLifecycle(t *testing.T) {
	store := &mutationStore{userProfile: models.User{ID: 1, Username: "admin", Roles: []string{auth.RoleSuperAdmin}}}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, _ := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
	request := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		server.Routes().ServeHTTP(rec, req)
		return rec
	}

	local := request(http.MethodPost, "/api/copilot/configs", `{"name":"内网应急模型","provider":"local","endpoint":"https://must-clear.example/v1","model":"must-clear","localEndpoint":"http://host.docker.internal:11434","localModel":"qwen2.5:7b","temperature":"0.2","maxTokens":"2048"}`)
	if local.Code != http.StatusCreated || !strings.Contains(local.Body.String(), `"isActive":true`) {
		t.Fatalf("expected first local profile to become active, got %d: %s", local.Code, local.Body.String())
	}
	if strings.Contains(local.Body.String(), "must-clear") {
		t.Fatalf("local profile response retained hosted fields: %s", local.Body.String())
	}

	hosted := request(http.MethodPost, "/api/copilot/configs", `{"name":"生产分析模型","provider":"openai","endpoint":"https://api.openai.com/v1","model":"gpt-4.1","apiKey":"sk-profile-secret","localEndpoint":"http://must-clear","localModel":"must-clear","temperature":"0.2","maxTokens":"2048"}`)
	if hosted.Code != http.StatusCreated || !strings.Contains(hosted.Body.String(), `"hasApiKey":true`) {
		t.Fatalf("expected hosted profile to be saved with managed key, got %d: %s", hosted.Code, hosted.Body.String())
	}
	if strings.Contains(hosted.Body.String(), "sk-profile-secret") || strings.Contains(hosted.Body.String(), "http://must-clear") {
		t.Fatalf("hosted profile response leaked a key or local fields: %s", hosted.Body.String())
	}

	activated := request(http.MethodPost, "/api/copilot/configs/2/activate", "")
	if activated.Code != http.StatusOK || !strings.Contains(activated.Body.String(), `"isActive":true`) {
		t.Fatalf("expected hosted profile activation, got %d: %s", activated.Code, activated.Body.String())
	}
	if store.copilotProfiles[0].IsActive || !store.copilotProfiles[1].IsActive {
		t.Fatalf("expected exactly the selected profile to be active: %+v", store.copilotProfiles)
	}

	activeDelete := request(http.MethodDelete, "/api/copilot/configs/2", "")
	if activeDelete.Code != http.StatusConflict {
		t.Fatalf("expected active profile deletion conflict, got %d: %s", activeDelete.Code, activeDelete.Body.String())
	}
	inactiveDelete := request(http.MethodDelete, "/api/copilot/configs/1", "")
	if inactiveDelete.Code != http.StatusNoContent {
		t.Fatalf("expected inactive profile deletion, got %d: %s", inactiveDelete.Code, inactiveDelete.Body.String())
	}
}

func TestCopilotSanitizesGoogleAPIKeyFromProviderDetails(t *testing.T) {
	detail := sanitizeProviderResponse("request failed for key gemini-secret-key/with+chars and encoded gemini-secret-key%2Fwith%2Bchars", "gemini-secret-key/with+chars")
	if strings.Contains(detail, "gemini-secret-key") || strings.Contains(detail, "gemini-secret-key%2Fwith%2Bchars") {
		t.Fatalf("provider detail must not leak api key, got %q", detail)
	}
}

func TestCopilotChatUsesConfiguredLocalModelForOpsEngineer(t *testing.T) {
	var seenPath string
	var seenBody string
	modelService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		payload, _ := io.ReadAll(r.Body)
		seenBody = string(payload)
		writeJSON(w, http.StatusOK, map[string]any{"message": map[string]string{"content": "建议先确认支付服务影响范围。"}})
	}))
	defer modelService.Close()

	store := newOpsMutationStore()
	store.copilotConfig = models.CopilotConfig{Provider: "local", LocalEndpoint: modelService.URL, LocalModel: "ops-test"}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(7, "ops.li", []string{auth.RoleOpsEngineer})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/copilot/chat", strings.NewReader(`{"question":"支付服务发生了什么？"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected chat status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if seenPath != "/api/chat" || !strings.Contains(seenBody, "支付服务发生了什么") {
		t.Fatalf("expected local chat request with question, path=%q body=%s", seenPath, seenBody)
	}
	if strings.Contains(seenBody, "password") || strings.Contains(seenBody, "secret") {
		t.Fatalf("chat context must not include credential fields: %s", seenBody)
	}
	if !strings.Contains(rec.Body.String(), "建议先确认支付服务影响范围") {
		t.Fatalf("expected model answer, got %s", rec.Body.String())
	}
}

func TestCopilotChatIncludesSanitizedProviderErrorDetails(t *testing.T) {
	const apiKey = "nvidia-test-secret"
	modelService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"model is overloaded; credential `+apiKey+`"}`, http.StatusServiceUnavailable)
	}))
	defer modelService.Close()

	answer, _, _, err := callCopilotModel(context.Background(), models.CopilotConfig{
		Provider: "local", LocalEndpoint: modelService.URL, LocalModel: "ops-test",
	}, apiKey, "status?", copilotAuthorizedContext{})
	if err == nil {
		t.Fatal("expected an upstream service error")
	}
	if answer != "" {
		t.Fatalf("expected no answer, got %q", answer)
	}
	if !strings.Contains(err.Error(), "HTTP 503") || !strings.Contains(err.Error(), "model is overloaded") {
		t.Fatalf("expected the upstream status and useful error detail, got %q", err)
	}
	if strings.Contains(err.Error(), apiKey) {
		t.Fatalf("provider error must not expose the API key: %q", err)
	}
}

func TestCopilotAnswerParsers(t *testing.T) {
	cases := []struct {
		provider string
		payload  string
	}{
		{"compatible", `{"choices":[{"message":{"content":"OpenAI answer"}}]}`},
		{"anthropic", `{"content":[{"text":"Claude answer"}]}`},
		{"google", `{"candidates":[{"content":{"parts":[{"text":"Gemini answer"}]}}]}`},
		{"local", `{"message":{"content":"Local answer"}}`},
	}
	for _, test := range cases {
		t.Run(test.provider, func(t *testing.T) {
			answer, err := parseCopilotAnswer(test.provider, []byte(test.payload))
			if err != nil || answer == "" {
				t.Fatalf("expected parsed answer, got %q, %v", answer, err)
			}
		})
	}
}

func TestCopilotChatRejectsEmptyQuestion(t *testing.T) {
	store := newOpsMutationStore()
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, _ := signer.Issue(7, "ops.li", []string{auth.RoleOpsEngineer})
	req := httptest.NewRequest(http.MethodPost, "/api/copilot/chat", strings.NewReader(`{"question":"  "}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected empty question to be rejected, got %d: %s", rec.Code, rec.Body.String())
	}
}
