package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
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
	result, err := testCopilotConnection(r.Context(), body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
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
	request, target, err := buildCopilotProbeRequest(ctx, provider, base, model, apiKey)
	if err != nil {
		return copilotConnectionResponse{}, err
	}

	start := time.Now()
	client := &http.Client{Timeout: 8 * time.Second}
	response, err := client.Do(request)
	latencyMs := time.Since(start).Milliseconds()
	result := copilotConnectionResponse{
		Provider:  provider,
		Endpoint:  target,
		Model:     model,
		LatencyMs: latencyMs,
	}
	if err != nil {
		result.Message = "连接失败：" + err.Error()
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
	}
	if len(cleaned) > 220 {
		cleaned = cleaned[:220] + "..."
	}
	return cleaned
}

func redactQueryKey(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	query := parsed.Query()
	if query.Has("key") {
		query.Set("key", "[redacted]")
		parsed.RawQuery = query.Encode()
	}
	return parsed.String()
}
