package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"opscore/backend/internal/models"
)

type copilotConnectionRequest struct {
	Provider      string `json:"provider"`
	Endpoint      string `json:"endpoint"`
	Model         string `json:"model"`
	APIKey        string `json:"apiKey"`
	LocalEndpoint string `json:"localEndpoint"`
	LocalModel    string `json:"localModel"`
}

type copilotConnectionResponse struct {
	OK         bool   `json:"ok"`
	Provider   string `json:"provider"`
	Endpoint   string `json:"endpoint"`
	Model      string `json:"model"`
	StatusCode int    `json:"statusCode,omitempty"`
	LatencyMs  int64  `json:"latencyMs"`
	Message    string `json:"message"`
}

func (s *Server) copilotTestConnection(w http.ResponseWriter, r *http.Request) {
	var body copilotConnectionRequest
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if normalizeCopilotProvider(body.Provider) != "local" && strings.TrimSpace(body.APIKey) == "" {
		apiKey, err := s.store.GetCopilotAPIKey(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		body.APIKey = apiKey
	}
	result, err := testCopilotConnection(r.Context(), body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) copilotConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		item, err := s.store.GetCopilotConfig(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodPut:
		var item models.CopilotConfig
		if err := readJSON(r, &item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := validateCopilotConfig(item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		saved, err := s.store.UpsertCopilotConfig(r.Context(), item)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, saved)
	}
}

func testCopilotConnection(ctx context.Context, body copilotConnectionRequest) (copilotConnectionResponse, error) {
	provider := normalizeCopilotProvider(body.Provider)
	endpoint := strings.TrimSpace(body.Endpoint)
	model := strings.TrimSpace(body.Model)
	apiKey := strings.TrimSpace(body.APIKey)

	if provider == "local" {
		if strings.TrimSpace(body.LocalEndpoint) != "" {
			endpoint = strings.TrimSpace(body.LocalEndpoint)
		}
		if strings.TrimSpace(body.LocalModel) != "" {
			model = strings.TrimSpace(body.LocalModel)
		}
	}
	if endpoint == "" {
		return copilotConnectionResponse{}, errors.New("endpoint is required")
	}
	if model == "" {
		return copilotConnectionResponse{}, errors.New("model is required")
	}
	if provider != "local" && apiKey == "" {
		return copilotConnectionResponse{}, errors.New("apiKey is required for hosted model providers")
	}

	base, err := normalizeHTTPBase(endpoint)
	if err != nil {
		return copilotConnectionResponse{}, err
	}
	if err := validateCopilotEndpointAccess(base, provider); err != nil {
		return copilotConnectionResponse{}, err
	}
	request, target, err := buildCopilotProbeRequest(ctx, provider, base, model, apiKey)
	if err != nil {
		return copilotConnectionResponse{}, err
	}

	start := time.Now()
	client := newCopilotHTTPClient(provider)
	response, err := client.Do(request)
	latencyMs := time.Since(start).Milliseconds()
	result := copilotConnectionResponse{
		Provider:  provider,
		Endpoint:  target,
		Model:     model,
		LatencyMs: latencyMs,
	}
	if err != nil {
		result.Message = connectionFailureMessage(err, apiKey, provider, endpoint)
		return result, nil
	}
	defer response.Body.Close()

	result.StatusCode = response.StatusCode
	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
		result.OK = true
		result.Message = "连接测试通过，模型服务已返回成功响应"
		return result, nil
	}

	bodyBytes, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
	detail := sanitizeProviderResponse(string(bodyBytes), apiKey)
	if detail != "" {
		detail = "：" + detail
	}
	result.Message = fmt.Sprintf("连接失败，模型服务返回 HTTP %d%s", response.StatusCode, detail)
	return result, nil
}

func normalizeCopilotProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "openai", "gpt":
		return "openai"
	case "anthropic", "claude":
		return "anthropic"
	case "google", "gemini":
		return "google"
	case "local", "ollama":
		return "local"
	default:
		return "compatible"
	}
}

func validateCopilotConfig(item models.CopilotConfig) error {
	provider := normalizeCopilotProvider(item.Provider)
	endpoint := strings.TrimSpace(item.Endpoint)
	model := strings.TrimSpace(item.Model)
	if provider == "local" {
		endpoint = strings.TrimSpace(item.LocalEndpoint)
		model = strings.TrimSpace(item.LocalModel)
	}
	if endpoint == "" {
		return errors.New("endpoint is required")
	}
	if model == "" {
		return errors.New("model is required")
	}
	base, err := normalizeHTTPBase(endpoint)
	if err != nil {
		return err
	}
	return validateCopilotEndpointAccess(base, provider)
}

func normalizeHTTPBase(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("endpoint must be a valid http or https URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("endpoint must use http or https")
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func validateCopilotEndpointAccess(base string, provider string) error {
	parsed, err := url.Parse(base)
	if err != nil {
		return errors.New("endpoint must be a valid http or https URL")
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if host == "" {
		return errors.New("endpoint host is required")
	}
	if host == "metadata.google.internal" || host == "169.254.169.254" {
		return errors.New("metadata service endpoints are not allowed")
	}
	if host == "host.docker.internal" {
		if provider == "local" {
			return nil
		}
		return errors.New("host.docker.internal is only allowed for local model provider")
	}
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		if provider == "local" {
			return nil
		}
		return errors.New("loopback endpoints are only allowed for local model provider")
	}
	if addr, err := netip.ParseAddr(host); err == nil {
		return validateCopilotAddress(provider, addr)
	}
	return nil
}

func newCopilotHTTPClient(provider string) *http.Client {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	transport := &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, fmt.Errorf("invalid model service address: %w", err)
			}
			addresses, err := resolveCopilotHost(ctx, host)
			if err != nil {
				return nil, err
			}
			if err := validateCopilotResolvedAddresses(provider, host, addresses); err != nil {
				return nil, err
			}

			var lastErr error
			for _, addr := range addresses {
				conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(addr.String(), port))
				if dialErr == nil {
					return conn, nil
				}
				lastErr = dialErr
			}
			if lastErr == nil {
				lastErr = errors.New("endpoint host did not resolve to an IP address")
			}
			return nil, lastErr
		},
	}
	return &http.Client{
		Timeout:       8 * time.Second,
		Transport:     transport,
		CheckRedirect: copilotRedirectPolicy(provider),
	}
}

func resolveCopilotHost(ctx context.Context, host string) ([]netip.Addr, error) {
	if addr, err := netip.ParseAddr(host); err == nil {
		return []netip.Addr{addr}, nil
	}
	addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("resolve endpoint host: %w", err)
	}
	return addresses, nil
}

func copilotRedirectPolicy(provider string) func(*http.Request, []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("too many model service redirects")
		}
		return validateCopilotEndpointAccess(req.URL.String(), provider)
	}
}

func validateCopilotResolvedAddresses(provider string, host string, addresses []netip.Addr) error {
	if len(addresses) == 0 {
		return errors.New("endpoint host did not resolve to an IP address")
	}
	for _, addr := range addresses {
		if err := validateCopilotAddress(provider, addr); err != nil {
			return fmt.Errorf("endpoint host %s resolved to disallowed address: %w", host, err)
		}
	}
	return nil
}

func validateCopilotAddress(provider string, addr netip.Addr) error {
	addr = addr.Unmap()
	if isMetadataAddress(addr) {
		return errors.New("metadata service endpoints are not allowed")
	}
	if addr.IsLoopback() || addr.IsPrivate() {
		if provider == "local" {
			return nil
		}
		return errors.New("private or loopback endpoints are only allowed for local model provider")
	}
	if addr.IsUnspecified() || addr.IsMulticast() || addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() {
		return errors.New("endpoint host is not allowed")
	}
	return nil
}

func isMetadataAddress(addr netip.Addr) bool {
	metadataV4 := netip.MustParseAddr("169.254.169.254")
	metadataV6 := netip.MustParseAddr("fd00:ec2::254")
	return addr == metadataV4 || addr == metadataV6
}

func buildCopilotProbeRequest(ctx context.Context, provider string, base string, model string, apiKey string) (*http.Request, string, error) {
	switch provider {
	case "anthropic":
		target := copilotEndpoint(base, "/v1/messages")
		payload := map[string]any{
			"model":      model,
			"max_tokens": 1,
			"messages":   []map[string]string{{"role": "user", "content": "ping"}},
		}
		req, err := jsonRequest(ctx, target, payload)
		if err != nil {
			return nil, "", err
		}
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
		return req, target, nil
	case "google":
		target := copilotEndpoint(base, "/v1beta/models/"+url.PathEscape(model)+":generateContent") + "?key=" + url.QueryEscape(apiKey)
		payload := map[string]any{
			"contents": []map[string]any{{"parts": []map[string]string{{"text": "ping"}}}},
			"generationConfig": map[string]any{
				"maxOutputTokens": 1,
			},
		}
		req, err := jsonRequest(ctx, target, payload)
		return req, redactQueryKey(target), err
	case "local":
		target := copilotEndpoint(base, "/api/generate")
		payload := map[string]any{"model": model, "prompt": "ping", "stream": false}
		req, err := jsonRequest(ctx, target, payload)
		return req, target, err
	default:
		target := copilotEndpoint(base, "/chat/completions")
		payload := map[string]any{
			"model":       model,
			"messages":    []map[string]string{{"role": "user", "content": "ping"}},
			"max_tokens":  1,
			"temperature": 0,
		}
		req, err := jsonRequest(ctx, target, payload)
		if err != nil {
			return nil, "", err
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		return req, target, nil
	}
}

func jsonRequest(ctx context.Context, target string, payload any) (*http.Request, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func copilotEndpoint(base string, path string) string {
	if strings.HasSuffix(base, path) {
		return base
	}
	if path == "/v1/messages" && strings.HasSuffix(base, "/v1") {
		return strings.TrimRight(base, "/") + "/messages"
	}
	if path == "/api/generate" && strings.HasSuffix(base, "/api") {
		return strings.TrimRight(base, "/") + "/generate"
	}
	return strings.TrimRight(base, "/") + path
}

func sanitizeProviderResponse(body string, apiKey string) string {
	cleaned := strings.TrimSpace(body)
	if apiKey != "" {
		cleaned = strings.ReplaceAll(cleaned, apiKey, "[redacted]")
		cleaned = strings.ReplaceAll(cleaned, url.QueryEscape(apiKey), "[redacted]")
	}
	if len(cleaned) > 220 {
		cleaned = cleaned[:220] + "..."
	}
	return cleaned
}

func connectionFailureMessage(err error, apiKey string, provider string, endpoint string) string {
	detail := sanitizeProviderResponse(err.Error(), apiKey)
	if provider == "local" && pointsToLoopback(endpoint) {
		detail += "；如果后端运行在 Docker 中，请将本地模型地址改为 http://host.docker.internal:11434"
	}
	return "连接失败：" + detail
}

func pointsToLoopback(endpoint string) bool {
	parsed, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func redactQueryKey(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	query := parsed.Query()
	if query.Has("key") {
		query.Set("key", "redacted")
		parsed.RawQuery = query.Encode()
	}
	return parsed.String()
}
