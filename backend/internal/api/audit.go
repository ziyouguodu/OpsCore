package api

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"

	"opscore/backend/internal/models"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusRecorder) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(body)
}

func (s *Server) auditRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)
		if !shouldAuditRequest(r) {
			return
		}
		event := s.requestAuditEvent(r, recorder.status)
		if err := s.store.RecordAudit(context.WithoutCancel(r.Context()), event); err != nil {
			log.Printf("record audit event: %v", err)
		}
	})
}

func shouldAuditRequest(r *http.Request) bool {
	if r.URL.Path == "/api/auth/login" || r.Method == http.MethodOptions {
		return false
	}
	return r.Method != http.MethodGet || strings.HasSuffix(r.URL.Path, "/credential/reveal")
}

func (s *Server) requestAuditEvent(r *http.Request, status int) models.AuditEvent {
	action, resourceType, resourceID := auditDescriptor(r)
	event := models.AuditEvent{
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Outcome:      auditOutcome(status),
		Detail:       map[string]any{"method": r.Method, "path": r.URL.Path, "status": status},
		IPAddress:    clientIP(r, s.cfg.TrustProxy),
		UserAgent:    r.UserAgent(),
	}
	header := r.Header.Get("Authorization")
	if strings.HasPrefix(header, "Bearer ") {
		if claims, err := s.signer.Verify(strings.TrimPrefix(header, "Bearer ")); err == nil {
			event.ActorUserID = claims.UserID
			event.ActorUsername = claims.Username
			if user, err := s.store.GetUser(r.Context(), claims.UserID); err == nil {
				event.ActorUsername = user.Username
			}
		}
	}
	return event
}

func auditDescriptor(r *http.Request) (string, string, string) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	resourceType := "system"
	resourceID := ""
	if len(parts) >= 2 {
		resourceType = parts[1]
	}
	if len(parts) >= 3 {
		resourceID = parts[2]
	}
	action := strings.ToLower(r.Method) + "." + resourceType
	if strings.HasSuffix(r.URL.Path, "/credential/reveal") {
		action = "credential.reveal"
	}
	if r.URL.Path == "/api/copilot/chat" {
		action = "copilot.chat"
		resourceID = ""
	}
	if resourceType == "copilot" && len(parts) >= 3 && parts[2] == "configs" {
		resourceType = "copilot_model_config"
		if len(parts) >= 4 {
			resourceID = parts[3]
		}
		if strings.HasSuffix(r.URL.Path, "/activate") {
			action = "copilot_config.activate"
		} else if strings.HasSuffix(r.URL.Path, "/test-connection") {
			action = "copilot_config.test"
		} else {
			action = strings.ToLower(r.Method) + ".copilot_config"
		}
	}
	return action, resourceType, resourceID
}

func auditOutcome(status int) string {
	if status >= 200 && status < 400 {
		return "success"
	}
	return "failure"
}

func (s *Server) recordLoginAudit(r *http.Request, userID int64, username, outcome, reason string) {
	detail := map[string]any{}
	if reason != "" {
		detail["reason"] = reason
	}
	event := models.AuditEvent{
		ActorUserID:   userID,
		ActorUsername: strings.TrimSpace(username),
		Action:        "auth.login",
		ResourceType:  "session",
		Outcome:       outcome,
		Detail:        detail,
		IPAddress:     clientIP(r, s.cfg.TrustProxy),
		UserAgent:     r.UserAgent(),
	}
	if err := s.store.RecordAudit(context.WithoutCancel(r.Context()), event); err != nil {
		log.Printf("record login audit event: %v", err)
	}
}

func (s *Server) auditEvents(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 500 {
			writeError(w, http.StatusBadRequest, errInvalidAuditLimit)
			return
		}
		limit = parsed
	}
	events, err := s.store.ListAuditEvents(r.Context(), limit)
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, events)
}

var errInvalidAuditLimit = &auditValidationError{"audit limit must be between 1 and 500"}

type auditValidationError struct{ message string }

func (e *auditValidationError) Error() string { return e.message }
