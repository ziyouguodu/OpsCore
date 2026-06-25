package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"opscore/backend/internal/auth"
	"opscore/backend/internal/config"
	"opscore/backend/internal/domain"
	"opscore/backend/internal/models"
	"opscore/backend/internal/store"
)

type Server struct {
	store  persistence
	signer auth.Signer
	cfg    config.Config
}

type ctxKey string

const claimsKey ctxKey = "claims"

var errForbiddenAssetDelete = store.ErrForbiddenAssetDelete

type persistence interface {
	Authenticate(context.Context, string, string) (models.User, bool, error)
	GetUser(context.Context, int64) (models.User, error)
	ChangePassword(context.Context, int64, string, string) (models.User, error)
	Dashboard(context.Context) (models.Dashboard, error)
	ListUsers(context.Context) ([]models.UserListItem, error)
	CreateUser(context.Context, models.UserMutation) (models.UserListItem, error)
	UpdateUser(context.Context, int64, models.UserMutation) (models.UserListItem, error)
	DeleteUser(context.Context, int64) error
	ListAssets(context.Context) ([]models.Asset, error)
	UpsertAsset(context.Context, models.Asset) (models.Asset, error)
	DeleteAsset(context.Context, int64, int64, bool) error
	GetAssetCredential(context.Context, int64) (models.AssetCredential, error)
	VerifyUserPassword(context.Context, string, string) (bool, error)
	HasCredentialVerificationPassword(context.Context) (bool, error)
	SetCredentialVerificationPassword(context.Context, string) error
	VerifyCredentialPassword(context.Context, string) (bool, error)
	UpsertAssetCredential(context.Context, models.AssetCredential) (models.AssetCredential, error)
	GetMiddlewareCredential(context.Context, int64) (models.MiddlewareCredential, error)
	UpsertMiddlewareCredential(context.Context, models.MiddlewareCredential) (models.MiddlewareCredential, error)
	GetCopilotConfig(context.Context) (models.CopilotConfig, error)
	UpsertCopilotConfig(context.Context, models.CopilotConfig) (models.CopilotConfig, error)
	GetCopilotAPIKey(context.Context) (string, error)
	ListMiddleware(context.Context) ([]models.MiddlewareInstance, error)
	CreateMiddleware(context.Context, models.MiddlewareInstance) (models.MiddlewareInstance, error)
	UpdateMiddleware(context.Context, int64, models.MiddlewareInstance) (models.MiddlewareInstance, error)
	DeleteMiddleware(context.Context, int64) error
	ListOnCalls(context.Context) ([]models.OnCallSchedule, error)
	CreateOnCall(context.Context, models.OnCallSchedule) (models.OnCallSchedule, error)
	UpdateOnCall(context.Context, int64, models.OnCallSchedule) (models.OnCallSchedule, error)
	DeleteOnCall(context.Context, int64) error
	ListTasks(context.Context) ([]models.Task, error)
	CreateTask(context.Context, models.Task) (models.Task, error)
	UpdateTask(context.Context, int64, models.Task) (models.Task, error)
	DeleteTask(context.Context, int64) error
	GetTaskStatus(context.Context, int64) (string, error)
	UpdateTaskStatus(context.Context, int64, string) error
	ListIncidents(context.Context) ([]models.Incident, error)
	CreateIncident(context.Context, models.Incident) (models.Incident, error)
	UpdateIncident(context.Context, int64, models.Incident) (models.Incident, error)
	DeleteIncident(context.Context, int64) error
	GetIncidentStatus(context.Context, int64) (string, error)
	UpdateIncidentStatus(context.Context, int64, string) error
}

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
	mux.Handle("POST /api/users", s.requirePermission(auth.PermissionUserManage, http.HandlerFunc(s.users)))
	mux.Handle("PUT /api/users/{id}", s.requirePermission(auth.PermissionUserManage, http.HandlerFunc(s.userResource)))
	mux.Handle("DELETE /api/users/{id}", s.requirePermission(auth.PermissionUserManage, http.HandlerFunc(s.userResource)))
	mux.Handle("GET /api/security/credential-verification", s.requirePermission(auth.PermissionUserManage, http.HandlerFunc(s.credentialVerification)))
	mux.Handle("PUT /api/security/credential-verification", s.requirePermission(auth.PermissionUserManage, http.HandlerFunc(s.credentialVerification)))
	mux.Handle("GET /api/copilot/config", s.requirePermission(auth.PermissionUserManage, http.HandlerFunc(s.copilotConfig)))
	mux.Handle("PUT /api/copilot/config", s.requirePermission(auth.PermissionUserManage, http.HandlerFunc(s.copilotConfig)))
	mux.Handle("POST /api/copilot/test-connection", s.requirePermission(auth.PermissionUserManage, http.HandlerFunc(s.copilotTestConnection)))
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
	return s.cors(mux)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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
	user, ok, err := s.store.Authenticate(r.Context(), body.Username, body.Password)
	if err != nil || !ok {
		writeError(w, http.StatusUnauthorized, errors.New("invalid username or password"))
		return
	}
	token, err := s.signer.Issue(user.ID, user.Username, user.Roles)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
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
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (s *Server) users(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := s.store.ListUsers(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
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
			writeError(w, http.StatusNotFound, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) assets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := s.store.ListAssets(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
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
			writeError(w, http.StatusInternalServerError, err)
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
		ok, err := s.verifyCredentialRevealPassword(r.Context(), claimsFrom(r.Context()), body.Password)
		if err != nil || !ok {
			writeError(w, http.StatusUnauthorized, errors.New("credential verification password is invalid"))
			return
		}
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
		items, err := s.store.ListMiddleware(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
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
		ok, err := s.verifyCredentialRevealPassword(r.Context(), claimsFrom(r.Context()), body.Password)
		if err != nil || !ok {
			writeError(w, http.StatusUnauthorized, errors.New("credential verification password is invalid"))
			return
		}
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

func (s *Server) verifyCredentialRevealPassword(ctx context.Context, claims auth.Claims, password string) (bool, error) {
	hasUnifiedPassword, err := s.store.HasCredentialVerificationPassword(ctx)
	if err != nil {
		return false, err
	}
	if hasUnifiedPassword {
		return s.store.VerifyCredentialPassword(ctx, password)
	}
	return s.store.VerifyUserPassword(ctx, claims.Username, password)
}

func (s *Server) oncalls(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := s.store.ListOnCalls(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
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

func (s *Server) tasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := s.store.ListTasks(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var item models.Task
		if err := readJSON(r, &item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if item.Status == "" {
			item.Status = string(domain.TaskPending)
		}
		if item.Type == "" {
			item.Type = "任务"
		}
		if err := validateTaskMutation(item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := validateTaskStatus(item.Status); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		saved, err := s.store.CreateTask(r.Context(), item)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, saved)
	}
}

func (s *Server) taskResource(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r.URL.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var item models.Task
		if err := readJSON(r, &item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if item.Status == "" {
			item.Status = string(domain.TaskPending)
		}
		if item.Type == "" {
			item.Type = "任务"
		}
		if err := validateTaskMutation(item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := validateTaskStatus(item.Status); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		current, err := s.store.GetTaskStatus(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		if err := validateTaskTransition(current, item.Status); err != nil {
			writeError(w, http.StatusConflict, err)
			return
		}
		saved, err := s.store.UpdateTask(r.Context(), id, item)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, saved)
	case http.MethodDelete:
		if err := s.store.DeleteTask(r.Context(), id); err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) taskStatus(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r.URL.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := validateTaskStatus(body.Status); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	current, err := s.store.GetTaskStatus(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err := validateTaskTransition(current, body.Status); err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	if err := s.store.UpdateTaskStatus(r.Context(), id, body.Status); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "status": body.Status})
}

func (s *Server) incidents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := s.store.ListIncidents(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var item models.Incident
		if err := readJSON(r, &item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if item.Status == "" {
			item.Status = string(domain.IncidentNew)
		}
		if item.Level == "" {
			item.Level = "P3"
		}
		if err := validateIncidentMutation(item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := validateIncidentLevel(item.Level); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := validateIncidentStatus(item.Status); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		saved, err := s.store.CreateIncident(r.Context(), item)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, saved)
	}
}

func (s *Server) incidentResource(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r.URL.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	switch r.Method {
	case http.MethodPut:
		var item models.Incident
		if err := readJSON(r, &item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if item.Status == "" {
			item.Status = string(domain.IncidentNew)
		}
		if item.Level == "" {
			item.Level = "P3"
		}
		if err := validateIncidentMutation(item); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := validateIncidentLevel(item.Level); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := validateIncidentStatus(item.Status); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		current, err := s.store.GetIncidentStatus(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		if err := validateIncidentTransition(current, item.Status); err != nil {
			writeError(w, http.StatusConflict, err)
			return
		}
		saved, err := s.store.UpdateIncident(r.Context(), id, item)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, saved)
	case http.MethodDelete:
		if err := s.store.DeleteIncident(r.Context(), id); err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *Server) incidentStatus(w http.ResponseWriter, r *http.Request) {
	id, err := idFromPath(r.URL.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := validateIncidentStatus(body.Status); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	current, err := s.store.GetIncidentStatus(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err := validateIncidentTransition(current, body.Status); err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	if err := s.store.UpdateIncidentStatus(r.Context(), id, body.Status); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "status": body.Status})
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
	return json.NewDecoder(r.Body).Decode(target)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
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

func validateUserMutation(item models.UserMutation, requirePassword bool) error {
	if strings.TrimSpace(item.Username) == "" || strings.TrimSpace(item.DisplayName) == "" || len(item.Roles) == 0 {
		return errors.New("username, displayName and roles are required")
	}
	if requirePassword && item.Password == "" {
		return errors.New("password is required")
	}
	if item.Password != "" && len(item.Password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	return nil
}

func validateAsset(item models.Asset, requireAssetNo bool) error {
	required := map[string]string{
		"type":            item.Type,
		"cpuArch":         item.CPUArch,
		"business":        item.Business,
		"ipv4":            item.IPv4,
		"environment":     item.Environment,
		"os":              item.OS,
		"networkZone":     item.NetworkZone,
		"cpu":             item.CPU,
		"memory":          item.Memory,
		"disk":            item.Disk,
		"deploymentInfo":  item.DeploymentInfo,
		"owner":           item.Owner,
		"status":          item.Status,
		"connectedStatus": item.ConnectedStatus,
	}
	if requireAssetNo {
		required["assetNo"] = item.AssetNo
	}
	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("asset %s is required", field)
		}
	}
	if !containsString([]string{"物理机", "虚拟机"}, item.Type) {
		return errors.New("asset type must be 物理机 or 虚拟机")
	}
	if !containsString([]string{"生产", "仿真", "研发"}, item.Environment) {
		return errors.New("asset environment must be 生产, 仿真 or 研发")
	}
	if !containsString([]string{"运行中", "维护中", "停用", "故障"}, item.Status) {
		return errors.New("asset status is invalid")
	}
	if !containsString([]string{"已并网", "未并网", "待确认"}, item.ConnectedStatus) {
		return errors.New("asset connectedStatus is invalid")
	}
	return nil
}

func prepareMiddleware(item *models.MiddlewareInstance) error {
	required := map[string]string{
		"name":        item.Name,
		"kind":        item.Kind,
		"environment": item.Environment,
		"networkZone": item.NetworkZone,
		"endpoint":    item.Endpoint,
		"business":    item.Business,
		"owner":       item.Owner,
	}
	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("middleware %s is required", field)
		}
	}
	if item.Status == "" {
		item.Status = "运行中"
	}
	if !containsString([]string{"MySQL", "Redis", "Kafka", "PostgreSQL", "达梦", "Nginx", "ElasticSearch", "Nacos", "RocketMQ", "MinIO"}, item.Kind) {
		return errors.New("middleware kind is invalid")
	}
	if !containsString([]string{"生产", "仿真", "研发"}, item.Environment) {
		return errors.New("middleware environment must be 生产, 仿真 or 研发")
	}
	if !containsString([]string{"运行中", "维护中", "停用", "故障"}, item.Status) {
		return errors.New("middleware status is invalid")
	}
	return nil
}

func prepareOnCall(item *models.OnCallSchedule) error {
	if item.RuleType == "" {
		item.RuleType = "daily"
	}
	if item.RuleType != "daily" && item.RuleType != "weekly" {
		return errors.New("oncall ruleType must be daily or weekly")
	}
	if strings.TrimSpace(item.Primary) == "" {
		return errors.New("oncall primary is required")
	}
	if item.RuleType == "daily" && strings.TrimSpace(item.Date) == "" {
		return errors.New("oncall date is required for daily rule")
	}
	if item.RuleType == "weekly" && strings.TrimSpace(item.Week) == "" {
		return errors.New("oncall week is required for weekly rule")
	}
	return nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func validateTaskMutation(item models.Task) error {
	if strings.TrimSpace(item.Title) == "" {
		return errors.New("task title is required")
	}
	return nil
}

func validateTaskStatus(status string) error {
	switch domain.TaskStatus(status) {
	case domain.TaskPending, domain.TaskInProgress, domain.TaskPendingConfirm, domain.TaskDone, domain.TaskClosed:
		return nil
	default:
		return errors.New("invalid task status")
	}
}

func validateTaskTransition(from, to string) error {
	if from == to {
		return nil
	}
	if err := validateTaskStatus(from); err != nil {
		return err
	}
	if err := validateTaskStatus(to); err != nil {
		return err
	}
	if !domain.CanTransitionTask(domain.TaskStatus(from), domain.TaskStatus(to)) {
		return errors.New("invalid task status transition")
	}
	return nil
}

func validateIncidentMutation(item models.Incident) error {
	if strings.TrimSpace(item.Title) == "" {
		return errors.New("incident title is required")
	}
	return nil
}

func validateIncidentLevel(level string) error {
	switch level {
	case "P1", "P2", "P3", "P4":
		return nil
	default:
		return errors.New("incident level must be P1, P2, P3 or P4")
	}
}

func validateIncidentStatus(status string) error {
	switch domain.IncidentStatus(status) {
	case domain.IncidentNew, domain.IncidentProcessing, domain.IncidentRecovered, domain.IncidentClosed:
		return nil
	default:
		return errors.New("invalid incident status")
	}
}

func validateIncidentTransition(from, to string) error {
	if from == to {
		return nil
	}
	if err := validateIncidentStatus(from); err != nil {
		return err
	}
	if err := validateIncidentStatus(to); err != nil {
		return err
	}
	if !domain.CanTransitionIncident(domain.IncidentStatus(from), domain.IncidentStatus(to)) {
		return errors.New("invalid incident status transition")
	}
	return nil
}
