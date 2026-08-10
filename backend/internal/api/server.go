package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"opscore/backend/internal/auth"
	"opscore/backend/internal/config"
	"opscore/backend/internal/models"
	"opscore/backend/internal/store"
)

type Server struct {
	store             persistence
	signer            auth.Signer
	cfg               config.Config
	rateLimitersOnce  sync.Once
	loginLimiter      *failureLimiter
	credentialLimiter *failureLimiter
	copilotLimiter    *requestWindowLimiter
}

type ctxKey string

const claimsKey ctxKey = "claims"
const maxJSONBodyBytes = 1 << 20

var errForbiddenAssetDelete = store.ErrForbiddenAssetDelete
var errLastSuperAdmin = store.ErrLastSuperAdmin
var errUserAssignedToDuty = store.ErrUserAssignedToDuty
var errStatusConflict = store.ErrStatusConflict
var errDutyRevisionConflict = store.ErrDutyRevisionConflict
var errCopilotProfileNotFound = store.ErrCopilotProfileNotFound
var errCopilotActiveProfileDelete = store.ErrCopilotActiveProfileDelete
var errCopilotProfileKeyRequired = store.ErrCopilotProfileKeyRequired

func NewServer(store *store.Store, signer auth.Signer, cfg config.Config) *Server {
	return &Server{store: store, signer: signer, cfg: cfg}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("POST /api/auth/login", s.login)
	mux.Handle("GET /api/auth/me", s.requireAuth(http.HandlerFunc(s.me)))
	mux.Handle("POST /api/auth/password", s.requireAuth(http.HandlerFunc(s.changePassword)))
	mux.Handle("GET /api/dashboard", s.requireAuth(http.HandlerFunc(s.dashboard)))
	mux.Handle("GET /api/users", s.requirePermission(auth.PermissionUserManage, http.HandlerFunc(s.users)))
	mux.Handle("GET /api/user-directory", s.requireAuth(http.HandlerFunc(s.userDirectory)))
	mux.Handle("POST /api/users", s.requirePermission(auth.PermissionUserManage, http.HandlerFunc(s.users)))
	mux.Handle("PUT /api/users/{id}", s.requirePermission(auth.PermissionUserManage, http.HandlerFunc(s.userResource)))
	mux.Handle("DELETE /api/users/{id}", s.requirePermission(auth.PermissionUserManage, http.HandlerFunc(s.userResource)))
	mux.Handle("GET /api/security/credential-verification", s.requirePermission(auth.PermissionUserManage, http.HandlerFunc(s.credentialVerification)))
	mux.Handle("PUT /api/security/credential-verification", s.requirePermission(auth.PermissionUserManage, http.HandlerFunc(s.credentialVerification)))
	mux.Handle("GET /api/copilot/config", s.requirePermission(auth.PermissionUserManage, http.HandlerFunc(s.copilotConfig)))
	mux.Handle("PUT /api/copilot/config", s.requirePermission(auth.PermissionUserManage, http.HandlerFunc(s.copilotConfig)))
	mux.Handle("POST /api/copilot/test-connection", s.requirePermission(auth.PermissionUserManage, http.HandlerFunc(s.copilotTestConnection)))
	mux.Handle("GET /api/copilot/configs", s.requirePermission(auth.PermissionUserManage, http.HandlerFunc(s.copilotProfiles)))
	mux.Handle("POST /api/copilot/configs", s.requirePermission(auth.PermissionUserManage, http.HandlerFunc(s.copilotProfiles)))
	mux.Handle("PUT /api/copilot/configs/{id}", s.requirePermission(auth.PermissionUserManage, http.HandlerFunc(s.copilotProfileResource)))
	mux.Handle("DELETE /api/copilot/configs/{id}", s.requirePermission(auth.PermissionUserManage, http.HandlerFunc(s.copilotProfileResource)))
	mux.Handle("POST /api/copilot/configs/{id}/activate", s.requirePermission(auth.PermissionUserManage, http.HandlerFunc(s.copilotProfileActivate)))
	mux.Handle("POST /api/copilot/configs/{id}/test-connection", s.requirePermission(auth.PermissionUserManage, http.HandlerFunc(s.copilotProfileTestConnection)))
	mux.Handle("POST /api/copilot/chat", s.requireAuth(http.HandlerFunc(s.copilotChat)))
	mux.Handle("GET /api/audit-events", s.requirePermission(auth.PermissionUserManage, http.HandlerFunc(s.auditEvents)))
	mux.Handle("GET /api/assets", s.requirePermission(auth.PermissionAssetRead, http.HandlerFunc(s.assets)))
	mux.Handle("POST /api/assets", s.requirePermission(auth.PermissionAssetWrite, http.HandlerFunc(s.assets)))
	mux.Handle("PUT /api/assets/{id}", s.requirePermission(auth.PermissionAssetWrite, http.HandlerFunc(s.assetResource)))
	mux.Handle("DELETE /api/assets/{id}", s.requirePermission(auth.PermissionAssetWrite, http.HandlerFunc(s.assetResource)))
	mux.Handle("GET /api/assets/", s.requirePermission(auth.PermissionAssetCredential, http.HandlerFunc(s.assetCredential)))
	mux.Handle("POST /api/assets/", s.requirePermission(auth.PermissionAssetCredential, http.HandlerFunc(s.assetCredential)))
	mux.Handle("PUT /api/assets/", s.requirePermission(auth.PermissionAssetCredentialWrite, http.HandlerFunc(s.assetCredential)))
	mux.Handle("GET /api/middleware", s.requirePermission(auth.PermissionAssetRead, http.HandlerFunc(s.middleware)))
	mux.Handle("POST /api/middleware", s.requirePermission(auth.PermissionAssetWrite, http.HandlerFunc(s.middleware)))
	mux.Handle("PUT /api/middleware/{id}", s.requirePermission(auth.PermissionAssetWrite, http.HandlerFunc(s.middlewareResource)))
	mux.Handle("DELETE /api/middleware/{id}", s.requirePermission(auth.PermissionAssetWrite, http.HandlerFunc(s.middlewareResource)))
	mux.Handle("GET /api/middleware/", s.requirePermission(auth.PermissionAssetCredential, http.HandlerFunc(s.middlewareCredential)))
	mux.Handle("POST /api/middleware/", s.requirePermission(auth.PermissionAssetCredential, http.HandlerFunc(s.middlewareCredential)))
	mux.Handle("PUT /api/middleware/", s.requirePermission(auth.PermissionAssetCredentialWrite, http.HandlerFunc(s.middlewareCredential)))
	mux.Handle("GET /api/oncall", s.requirePermission(auth.PermissionOnCallRead, http.HandlerFunc(s.oncalls)))
	mux.Handle("POST /api/oncall", s.requirePermission(auth.PermissionOnCallWrite, http.HandlerFunc(s.oncalls)))
	mux.Handle("PUT /api/oncall/{id}", s.requirePermission(auth.PermissionOnCallWrite, http.HandlerFunc(s.oncallResource)))
	mux.Handle("DELETE /api/oncall/{id}", s.requirePermission(auth.PermissionOnCallWrite, http.HandlerFunc(s.oncallResource)))
	mux.Handle("GET /api/duty-center", s.requirePermission(auth.PermissionOnCallRead, http.HandlerFunc(s.dutyCenter)))
	mux.Handle("PUT /api/duty-center", s.requirePermission(auth.PermissionOnCallWrite, http.HandlerFunc(s.dutyCenter)))
	mux.Handle("GET /api/tasks", s.requirePermission(auth.PermissionTaskRead, http.HandlerFunc(s.tasks)))
	mux.Handle("POST /api/tasks", s.requirePermission(auth.PermissionTaskWrite, http.HandlerFunc(s.tasks)))
	mux.Handle("PUT /api/tasks/{id}", s.requirePermission(auth.PermissionTaskWrite, http.HandlerFunc(s.taskResource)))
	mux.Handle("DELETE /api/tasks/{id}", s.requirePermission(auth.PermissionTaskWrite, http.HandlerFunc(s.taskResource)))
	mux.Handle("PATCH /api/tasks/", s.requirePermission(auth.PermissionTaskWrite, http.HandlerFunc(s.taskStatus)))
	mux.Handle("GET /api/incidents", s.requirePermission(auth.PermissionIncidentRead, http.HandlerFunc(s.incidents)))
	mux.Handle("POST /api/incidents", s.requirePermission(auth.PermissionIncidentFollowup, http.HandlerFunc(s.incidents)))
	mux.Handle("PUT /api/incidents/{id}", s.requirePermission(auth.PermissionIncidentFollowup, http.HandlerFunc(s.incidentResource)))
	mux.Handle("DELETE /api/incidents/{id}", s.requirePermission(auth.PermissionIncidentFollowup, http.HandlerFunc(s.incidentResource)))
	mux.Handle("PATCH /api/incidents/", s.requirePermission(auth.PermissionIncidentFollowup, http.HandlerFunc(s.incidentStatus)))
	return s.cors(s.auditRequests(mux))
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Ping(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable", "database": "unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "database": "ok"})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	limitKey := loginRateKey(r, body.Username, s.cfg.TrustProxy)
	if allowed, retryAfter := s.loginAttempts().Allow(limitKey); !allowed {
		writeRateLimit(w, retryAfter)
		s.recordLoginAudit(r, 0, body.Username, "failure", "rate_limited")
		return
	}
	user, ok, err := s.store.Authenticate(r.Context(), body.Username, body.Password)
	if err != nil || !ok {
		s.loginAttempts().Failure(limitKey)
		s.recordLoginAudit(r, 0, body.Username, "failure", "invalid_credentials")
		writeError(w, http.StatusUnauthorized, errors.New("invalid username or password"))
		return
	}
	s.loginAttempts().Success(limitKey)
	token, err := s.signer.Issue(user.ID, user.Username, user.Roles)
	if err != nil {
		writeInternalError(w, err)
		return
	}
	s.recordLoginAudit(r, user.ID, user.Username, "success", "")
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "user": user})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	claims := claimsFrom(r.Context())
	user, err := s.store.GetUser(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if body.CurrentPassword == "" || body.NewPassword == "" {
		writeError(w, http.StatusBadRequest, errors.New("currentPassword and newPassword are required"))
		return
	}
	if len(body.NewPassword) < 8 {
		writeError(w, http.StatusBadRequest, errors.New("newPassword must be at least 8 characters"))
		return
	}
	claims := claimsFrom(r.Context())
	user, err := s.store.ChangePassword(r.Context(), claims.UserID, body.CurrentPassword, body.NewPassword)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	data, err := s.store.Dashboard(r.Context())
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (s *Server) users(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := s.store.ListUsers(r.Context())
		if err != nil {
			writeInternalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var item models.UserMutation
		if err := readJSON(r, &item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := validateUserMutation(item, true); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		saved, err := s.store.CreateUser(r.Context(), item)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, saved)
	}
}

func (s *Server) userDirectory(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListUserDirectory(r.Context())
	if err != nil {
		writeInternalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) userResource(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r.URL.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var item models.UserMutation
		if err := readJSON(r, &item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := validateUserMutation(item, false); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		saved, err := s.store.UpdateUser(r.Context(), id, item)
		if err != nil {
			if errors.Is(err, errLastSuperAdmin) {
				writeError(w, http.StatusConflict, err)
				return
			}
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, saved)
	case http.MethodDelete:
		claims := claimsFrom(r.Context())
		if claims.UserID == id {
			writeError(w, http.StatusBadRequest, errors.New("cannot delete current user"))
			return
		}
		if err := s.store.DeleteUser(r.Context(), id); err != nil {
			if errors.Is(err, errLastSuperAdmin) || errors.Is(err, errUserAssignedToDuty) {
				writeError(w, http.StatusConflict, err)
				return
			}
			writeError(w, http.StatusNotFound, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) assets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		query, err := parseListQuery(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		items, err := s.store.ListAssetsPage(r.Context(), query)
		if err != nil {
			writeInternalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var item models.Asset
		if err := readJSON(r, &item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := validateAsset(item, false); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		item.CreatedBy = claimsFrom(r.Context()).UserID
		saved, err := s.store.UpsertAsset(r.Context(), item)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, saved)
	}
}

func (s *Server) assetResource(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r.URL.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var item models.Asset
		if err := readJSON(r, &item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		item.ID = id
		if err := validateAsset(item, true); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		saved, err := s.store.UpsertAsset(r.Context(), item)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, saved)
	case http.MethodDelete:
		claims := claimsFrom(r.Context())
		if err := s.store.DeleteAsset(r.Context(), id, claims.UserID, auth.HasRole(claims.Roles, auth.RoleSuperAdmin)); err != nil {
			if errors.Is(err, errForbiddenAssetDelete) {
				writeError(w, http.StatusForbidden, err)
				return
			}
			writeError(w, http.StatusNotFound, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) credentialVerification(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		hasPassword, err := s.store.HasCredentialVerificationPassword(r.Context())
		if err != nil {
			writeInternalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"hasPassword": hasPassword})
	case http.MethodPut:
		var body struct {
			Password string `json:"password"`
		}
		if err := readJSON(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if strings.TrimSpace(body.Password) == "" {
			writeError(w, http.StatusBadRequest, errors.New("password is required"))
			return
		}
		if len(body.Password) < 8 {
			writeError(w, http.StatusBadRequest, errors.New("password must be at least 8 characters"))
			return
		}
		if err := s.store.SetCredentialVerificationPassword(r.Context(), body.Password); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"hasPassword": true})
	}
}

func (s *Server) assetCredential(w http.ResponseWriter, r *http.Request) {
	reveal := strings.HasSuffix(r.URL.Path, "/credential/reveal")
	if !strings.HasSuffix(r.URL.Path, "/credential") && !reveal {
		writeError(w, http.StatusNotFound, errors.New("not found"))
		return
	}
	assetID, err := assetIDFromCredentialPath(r.URL.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	switch r.Method {
	case http.MethodGet:
		if reveal {
			writeError(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
			return
		}
		item, err := s.store.GetAssetCredential(r.Context(), assetID)
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeJSON(w, http.StatusOK, credentialResponse(item, false))
	case http.MethodPost:
		if !reveal {
			writeError(w, http.StatusNotFound, errors.New("not found"))
			return
		}
		var body struct {
			Password string `json:"password"`
		}
		if err := readJSON(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		claims := claimsFrom(r.Context())
		limitKey := credentialRateKey(r, claims.UserID, s.cfg.TrustProxy)
		if allowed, retryAfter := s.credentialAttempts().Allow(limitKey); !allowed {
			writeRateLimit(w, retryAfter)
			return
		}
		ok, err := s.verifyCredentialRevealPassword(r.Context(), claims, body.Password)
		if err != nil || !ok {
			s.credentialAttempts().Failure(limitKey)
			writeError(w, http.StatusUnauthorized, errors.New("credential verification password is invalid"))
			return
		}
		s.credentialAttempts().Success(limitKey)
		item, err := s.store.GetAssetCredential(r.Context(), assetID)
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeJSON(w, http.StatusOK, credentialResponse(item, true))
	case http.MethodPut:
		if reveal {
			writeError(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
			return
		}
		var item models.AssetCredential
		if err := readJSON(r, &item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		item.AssetID = assetID
		saved, err := s.store.UpsertAssetCredential(r.Context(), item)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, credentialResponse(saved, false))
	}
}

func (s *Server) middleware(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		query, err := parseListQuery(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		items, err := s.store.ListMiddlewarePage(r.Context(), query)
		if err != nil {
			writeInternalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var item models.MiddlewareInstance
		if err := readJSON(r, &item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := prepareMiddleware(&item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		saved, err := s.store.CreateMiddleware(r.Context(), item)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, saved)
	}
}

func (s *Server) middlewareResource(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r.URL.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var item models.MiddlewareInstance
		if err := readJSON(r, &item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := prepareMiddleware(&item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		saved, err := s.store.UpdateMiddleware(r.Context(), id, item)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, saved)
	case http.MethodDelete:
		if err := s.store.DeleteMiddleware(r.Context(), id); err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) middlewareCredential(w http.ResponseWriter, r *http.Request) {
	reveal := strings.HasSuffix(r.URL.Path, "/credential/reveal")
	if !strings.HasSuffix(r.URL.Path, "/credential") && !reveal {
		writeError(w, http.StatusNotFound, errors.New("not found"))
		return
	}
	middlewareID, err := resourceIDFromCredentialPath(r.URL.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	switch r.Method {
	case http.MethodGet:
		if reveal {
			writeError(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
			return
		}
		item, err := s.store.GetMiddlewareCredential(r.Context(), middlewareID)
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeJSON(w, http.StatusOK, middlewareCredentialResponse(item, false))
	case http.MethodPost:
		if !reveal {
			writeError(w, http.StatusNotFound, errors.New("not found"))
			return
		}
		var body struct {
			Password string `json:"password"`
		}
		if err := readJSON(r, &body); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		claims := claimsFrom(r.Context())
		limitKey := credentialRateKey(r, claims.UserID, s.cfg.TrustProxy)
		if allowed, retryAfter := s.credentialAttempts().Allow(limitKey); !allowed {
			writeRateLimit(w, retryAfter)
			return
		}
		ok, err := s.verifyCredentialRevealPassword(r.Context(), claims, body.Password)
		if err != nil || !ok {
			s.credentialAttempts().Failure(limitKey)
			writeError(w, http.StatusUnauthorized, errors.New("credential verification password is invalid"))
			return
		}
		s.credentialAttempts().Success(limitKey)
		item, err := s.store.GetMiddlewareCredential(r.Context(), middlewareID)
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeJSON(w, http.StatusOK, middlewareCredentialResponse(item, true))
	case http.MethodPut:
		if reveal {
			writeError(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
			return
		}
		var item models.MiddlewareCredential
		if err := readJSON(r, &item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		item.MiddlewareID = middlewareID
		saved, err := s.store.UpsertMiddlewareCredential(r.Context(), item)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, middlewareCredentialResponse(saved, false))
	}
}

func (s *Server) verifyCredentialRevealPassword(ctx context.Context, _ auth.Claims, password string) (bool, error) {
	hasUnifiedPassword, err := s.store.HasCredentialVerificationPassword(ctx)
	if err != nil {
		return false, err
	}
	if hasUnifiedPassword {
		return s.store.VerifyCredentialPassword(ctx, password)
	}
	return false, nil
}

func (s *Server) oncalls(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := s.store.ListOnCalls(r.Context())
		if err != nil {
			writeInternalError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var item models.OnCallSchedule
		if err := readJSON(r, &item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := prepareOnCall(&item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		saved, err := s.store.CreateOnCall(r.Context(), item)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, saved)
	}
}

func (s *Server) oncallResource(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r.URL.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var item models.OnCallSchedule
		if err := readJSON(r, &item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := prepareOnCall(&item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		saved, err := s.store.UpdateOnCall(r.Context(), id, item)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, saved)
	case http.MethodDelete:
		if err := s.store.DeleteOnCall(r.Context(), id); err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeError(w, http.StatusUnauthorized, errors.New("missing bearer token"))
			return
		}
		claims, err := s.signer.Verify(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			writeError(w, http.StatusUnauthorized, err)
			return
		}
		user, err := s.store.GetUser(r.Context(), claims.UserID)
		if err != nil {
			writeError(w, http.StatusUnauthorized, err)
			return
		}
		if user.MustChangePassword && !initialPasswordAllowedPath(r.URL.Path) {
			writeError(w, http.StatusForbidden, errors.New("initial password must be changed before accessing OpsCore APIs"))
			return
		}
		claims.Username = user.Username
		claims.Roles = append([]string(nil), user.Roles...)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claimsKey, claims)))
	})
}

func (s *Server) requirePermission(permission string, next http.Handler) http.Handler {
	return s.requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := claimsFrom(r.Context())
		if !auth.HasPermission(claims.Roles, permission) {
			writeError(w, http.StatusForbidden, errors.New("forbidden"))
			return
		}
		next.ServeHTTP(w, r)
	}))
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Origin", s.cfg.CORSOrigin)
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func claimsFrom(ctx context.Context) auth.Claims {
	claims, _ := ctx.Value(claimsKey).(auth.Claims)
	return claims
}

func initialPasswordAllowedPath(path string) bool {
	return path == "/api/auth/me" || path == "/api/auth/password"
}

func readJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, maxJSONBodyBytes+1))
	if err != nil {
		return err
	}
	if len(body) > maxJSONBodyBytes {
		return errors.New("request body is too large")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("request body must contain a single JSON object")
		}
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": publicError(status, err)})
}

func idFromPath(path string) (int64, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return strconv.ParseInt(parts[len(parts)-1], 10, 64)
}

func assetIDFromCredentialPath(path string) (int64, error) {
	return resourceIDFromCredentialPath(path)
}

func resourceIDFromCredentialPath(path string) (int64, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 5 && parts[len(parts)-2] == "credential" && parts[len(parts)-1] == "reveal" {
		return strconv.ParseInt(parts[len(parts)-3], 10, 64)
	}
	if len(parts) < 4 || parts[len(parts)-1] != "credential" {
		return 0, errors.New("invalid credential path")
	}
	return strconv.ParseInt(parts[len(parts)-2], 10, 64)
}

func credentialResponse(item models.AssetCredential, reveal bool) models.AssetCredential {
	item.HasSecret = item.Secret != ""
	if !reveal {
		item.Secret = ""
	}
	return item
}

func middlewareCredentialResponse(item models.MiddlewareCredential, reveal bool) models.MiddlewareCredential {
	item.HasSecret = item.Secret != ""
	if !reveal {
		item.Secret = ""
	}
	return item
}
