package api

import (
	"context"
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

type mutationStore struct {
	assetDeleted               int64
	assetDeleteActor           int64
	assetDeleteSuper           bool
	assetCreator               int64
	middlewareDeleted          int64
	oncallDeleted              int64
	taskDeleted                int64
	incidentDeleted            int64
	assetUpdated               models.Asset
	middlewareUpdated          models.MiddlewareInstance
	oncallUpdated              models.OnCallSchedule
	taskCreated                models.Task
	taskUpdated                models.Task
	incidentCreated            models.Incident
	incidentUpdated            models.Incident
	userCreated                models.UserListItem
	userUpdated                models.UserListItem
	userDeleted                int64
	middlewareCred             models.MiddlewareCredential
	userProfile                models.User
	passwordUserID             int64
	currentPassword            string
	newPassword                string
	credentialVerifyPassword   string
	credentialVerifyConfigured string
	credentialVerifyChecks     []string
}

func (s *mutationStore) Authenticate(context.Context, string, string) (models.User, bool, error) {
	return models.User{}, false, nil
}
func (s *mutationStore) GetUser(_ context.Context, userID int64) (models.User, error) {
	if s.userProfile.ID == 0 {
		return models.User{ID: userID, Username: "admin", Roles: []string{auth.RoleSuperAdmin}}, nil
	}
	return s.userProfile, nil
}
func (s *mutationStore) ChangePassword(_ context.Context, userID int64, currentPassword string, newPassword string) (models.User, error) {
	s.passwordUserID = userID
	s.currentPassword = currentPassword
	s.newPassword = newPassword
	return models.User{ID: userID, Username: "admin", DisplayName: "超级管理员", Roles: []string{auth.RoleSuperAdmin}, MustChangePassword: false}, nil
}
func (s *mutationStore) Dashboard(context.Context) (models.Dashboard, error) {
	return models.Dashboard{}, nil
}
func (s *mutationStore) ListUsers(context.Context) ([]models.UserListItem, error) {
	return []models.UserListItem{}, nil
}
func (s *mutationStore) CreateUser(_ context.Context, item models.UserMutation) (models.UserListItem, error) {
	s.userCreated = models.UserListItem{
		ID:                 10,
		Username:           item.Username,
		DisplayName:        item.DisplayName,
		MustChangePassword: item.MustChangePassword,
		Roles:              item.Roles,
	}
	return s.userCreated, nil
}
func (s *mutationStore) UpdateUser(_ context.Context, id int64, item models.UserMutation) (models.UserListItem, error) {
	s.userUpdated = models.UserListItem{
		ID:                 id,
		Username:           item.Username,
		DisplayName:        item.DisplayName,
		MustChangePassword: item.MustChangePassword,
		Roles:              item.Roles,
	}
	return s.userUpdated, nil
}
func (s *mutationStore) DeleteUser(_ context.Context, id int64) error {
	s.userDeleted = id
	return nil
}
func (s *mutationStore) ListAssets(context.Context) ([]models.Asset, error) {
	return []models.Asset{}, nil
}
func (s *mutationStore) UpsertAsset(_ context.Context, item models.Asset) (models.Asset, error) {
	s.assetUpdated = item
	item.UpdatedAt = time.Now()
	return item, nil
}
func (s *mutationStore) DeleteAsset(_ context.Context, id int64, actorUserID int64, actorIsSuperAdmin bool) error {
	s.assetDeleteActor = actorUserID
	s.assetDeleteSuper = actorIsSuperAdmin
	if !actorIsSuperAdmin && s.assetCreator != 0 && s.assetCreator != actorUserID {
		return errForbiddenAssetDelete
	}
	s.assetDeleted = id
	return nil
}
func (s *mutationStore) GetAssetCredential(context.Context, int64) (models.AssetCredential, error) {
	return models.AssetCredential{}, nil
}

func TestOpsEngineerCannotDeleteAssetCreatedByAnotherUser(t *testing.T) {
	store := &mutationStore{assetCreator: 99}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(7, "ops.li", []string{auth.RoleOpsEngineer})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/assets/42", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden for non-owner ops engineer delete, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.assetDeleted != 0 {
		t.Fatalf("non-owner delete should not delete asset, got %d", store.assetDeleted)
	}
	if store.assetDeleteActor != 7 {
		t.Fatalf("expected actor user 7 to be passed to store, got %d", store.assetDeleteActor)
	}
}

func TestSuperAdminCanDeleteAssetCreatedByAnotherUser(t *testing.T) {
	store := &mutationStore{assetCreator: 99}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/assets/42", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected super admin delete status 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.assetDeleted != 42 {
		t.Fatalf("expected asset 42 to be deleted, got %d", store.assetDeleted)
	}
	if !store.assetDeleteSuper {
		t.Fatalf("expected super admin flag to be passed to store")
	}
}

func TestMeReturnsUserProfileIncludingMustChangePassword(t *testing.T) {
	store := &mutationStore{
		userProfile: models.User{
			ID:                 1,
			Username:           "admin",
			DisplayName:        "超级管理员",
			Roles:              []string{auth.RoleSuperAdmin},
			MustChangePassword: true,
		},
	}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected me status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var user models.User
	if err := json.NewDecoder(rec.Body).Decode(&user); err != nil {
		t.Fatal(err)
	}
	if !user.MustChangePassword || user.Username != "admin" {
		t.Fatalf("expected user profile with mustChangePassword, got %+v", user)
	}
}

func TestPasswordChangeClearsMustChangePassword(t *testing.T) {
	store := &mutationStore{}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/password", strings.NewReader(`{"currentPassword":"ChangeMe123!","newPassword":"OpsCore2026!"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected password change status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var user models.User
	if err := json.NewDecoder(rec.Body).Decode(&user); err != nil {
		t.Fatal(err)
	}
	if user.MustChangePassword {
		t.Fatalf("expected password change response to clear mustChangePassword")
	}
	if store.passwordUserID != 1 || store.currentPassword != "ChangeMe123!" || store.newPassword != "OpsCore2026!" {
		t.Fatalf("password change request not passed to store correctly: user=%d current=%q new=%q", store.passwordUserID, store.currentPassword, store.newPassword)
	}
}

func TestPasswordChangeRejectsShortNewPassword(t *testing.T) {
	store := &mutationStore{}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/password", strings.NewReader(`{"currentPassword":"ChangeMe123!","newPassword":"short"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected short new password status 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.passwordUserID != 0 {
		t.Fatalf("short password should be rejected before hitting store, got user id %d", store.passwordUserID)
	}
}

func TestUserMutationRoutesCreateUpdateAndDeleteByID(t *testing.T) {
	store := &mutationStore{}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
	if err != nil {
		t.Fatal(err)
	}

	createReq := httptest.NewRequest(http.MethodPost, "/api/users", strings.NewReader(`{"username":"ops.li","displayName":"李明","password":"OpsCore2026!","mustChangePassword":true,"roles":["ops_engineer"]}`))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected user create status 201, got %d: %s", createRec.Code, createRec.Body.String())
	}
	var created models.UserListItem
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Username != "ops.li" || created.Roles[0] != auth.RoleOpsEngineer {
		t.Fatalf("unexpected created user: %+v", created)
	}
	if strings.Contains(createRec.Body.String(), "OpsCore2026!") {
		t.Fatalf("user create response must not contain plaintext password")
	}

	updateReq := httptest.NewRequest(http.MethodPut, "/api/users/10", strings.NewReader(`{"username":"ops.li","displayName":"李明-运维","mustChangePassword":false,"roles":["ops_engineer"]}`))
	updateReq.Header.Set("Authorization", "Bearer "+token)
	updateRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected user update status 200, got %d: %s", updateRec.Code, updateRec.Body.String())
	}
	var updated models.UserListItem
	if err := json.NewDecoder(updateRec.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated.ID != 10 || updated.DisplayName != "李明-运维" || store.userUpdated.ID != 10 {
		t.Fatalf("user update did not preserve path id: response=%+v stored=%+v", updated, store.userUpdated)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/users/10", nil)
	deleteReq.Header.Set("Authorization", "Bearer "+token)
	deleteRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("expected user delete status 204, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
	if store.userDeleted != 10 {
		t.Fatalf("expected user 10 to be deleted, got %d", store.userDeleted)
	}
}

func TestUserMutationRejectsShortPasswords(t *testing.T) {
	store := &mutationStore{}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
	if err != nil {
		t.Fatal(err)
	}

	createReq := httptest.NewRequest(http.MethodPost, "/api/users", strings.NewReader(`{"username":"ops.short","displayName":"短密码","password":"short","roles":["ops_engineer"]}`))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusBadRequest {
		t.Fatalf("expected short create password status 400, got %d: %s", createRec.Code, createRec.Body.String())
	}
	if store.userCreated.Username != "" {
		t.Fatalf("short create password should be rejected before hitting store, got %+v", store.userCreated)
	}

	updateReq := httptest.NewRequest(http.MethodPut, "/api/users/10", strings.NewReader(`{"username":"ops.li","displayName":"李明","password":"short","roles":["ops_engineer"]}`))
	updateReq.Header.Set("Authorization", "Bearer "+token)
	updateRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusBadRequest {
		t.Fatalf("expected short update password status 400, got %d: %s", updateRec.Code, updateRec.Body.String())
	}
	if store.userUpdated.Username != "" {
		t.Fatalf("short update password should be rejected before hitting store, got %+v", store.userUpdated)
	}
}

func TestOpsEngineerCannotManageUsers(t *testing.T) {
	store := &mutationStore{}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(7, "ops.li", []string{auth.RoleOpsEngineer})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/users", strings.NewReader(`{"username":"other","displayName":"Other","password":"OpsCore2026!","roles":["ops_engineer"]}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected ops engineer to be forbidden from user management, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCredentialVerificationPasswordCanBeConfiguredBySuperAdmin(t *testing.T) {
	store := &mutationStore{}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/security/credential-verification", strings.NewReader(`{"password":"Verify2026!"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected credential verification config status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.credentialVerifyConfigured != "Verify2026!" {
		t.Fatalf("expected unified credential verification password to be saved, got %q", store.credentialVerifyConfigured)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["hasPassword"] != true {
		t.Fatalf("expected response to report configured password, got %+v", body)
	}
}

func TestOpsEngineerCannotConfigureCredentialVerificationPassword(t *testing.T) {
	store := &mutationStore{}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(7, "ops.li", []string{auth.RoleOpsEngineer})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/security/credential-verification", strings.NewReader(`{"password":"Verify2026!"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected ops engineer to be forbidden from credential verification config, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.credentialVerifyConfigured != "" {
		t.Fatalf("ops engineer should not configure credential verification password, got %q", store.credentialVerifyConfigured)
	}
}

func (s *mutationStore) VerifyUserPassword(context.Context, string, string) (bool, error) {
	return true, nil
}
func (s *mutationStore) HasCredentialVerificationPassword(context.Context) (bool, error) {
	return s.credentialVerifyPassword != "", nil
}
func (s *mutationStore) SetCredentialVerificationPassword(_ context.Context, password string) error {
	s.credentialVerifyConfigured = password
	s.credentialVerifyPassword = password
	return nil
}
func (s *mutationStore) VerifyCredentialPassword(_ context.Context, password string) (bool, error) {
	s.credentialVerifyChecks = append(s.credentialVerifyChecks, password)
	return s.credentialVerifyPassword != "" && s.credentialVerifyPassword == password, nil
}
func (s *mutationStore) UpsertAssetCredential(_ context.Context, item models.AssetCredential) (models.AssetCredential, error) {
	return item, nil
}
func (s *mutationStore) GetMiddlewareCredential(_ context.Context, id int64) (models.MiddlewareCredential, error) {
	item := s.middlewareCred
	item.MiddlewareID = id
	return item, nil
}
func (s *mutationStore) UpsertMiddlewareCredential(_ context.Context, item models.MiddlewareCredential) (models.MiddlewareCredential, error) {
	s.middlewareCred = item
	return item, nil
}
func (s *mutationStore) ListMiddleware(context.Context) ([]models.MiddlewareInstance, error) {
	return []models.MiddlewareInstance{}, nil
}
func (s *mutationStore) CreateMiddleware(_ context.Context, item models.MiddlewareInstance) (models.MiddlewareInstance, error) {
	return item, nil
}
func (s *mutationStore) UpdateMiddleware(_ context.Context, id int64, item models.MiddlewareInstance) (models.MiddlewareInstance, error) {
	s.middlewareUpdated = item
	s.middlewareUpdated.ID = id
	return s.middlewareUpdated, nil
}
func (s *mutationStore) DeleteMiddleware(_ context.Context, id int64) error {
	s.middlewareDeleted = id
	return nil
}
func (s *mutationStore) ListOnCalls(context.Context) ([]models.OnCallSchedule, error) {
	return []models.OnCallSchedule{}, nil
}
func (s *mutationStore) CreateOnCall(_ context.Context, item models.OnCallSchedule) (models.OnCallSchedule, error) {
	return item, nil
}
func (s *mutationStore) UpdateOnCall(_ context.Context, id int64, item models.OnCallSchedule) (models.OnCallSchedule, error) {
	s.oncallUpdated = item
	s.oncallUpdated.ID = id
	return s.oncallUpdated, nil
}
func (s *mutationStore) DeleteOnCall(_ context.Context, id int64) error {
	s.oncallDeleted = id
	return nil
}
func (s *mutationStore) ListTasks(context.Context) ([]models.Task, error) {
	return []models.Task{}, nil
}
func (s *mutationStore) CreateTask(_ context.Context, item models.Task) (models.Task, error) {
	s.taskCreated = item
	return item, nil
}
func (s *mutationStore) UpdateTask(_ context.Context, id int64, item models.Task) (models.Task, error) {
	s.taskUpdated = item
	s.taskUpdated.ID = id
	return s.taskUpdated, nil
}
func (s *mutationStore) DeleteTask(_ context.Context, id int64) error {
	s.taskDeleted = id
	return nil
}
func (s *mutationStore) GetTaskStatus(context.Context, int64) (string, error) {
	return "待处理", nil
}
func (s *mutationStore) UpdateTaskStatus(context.Context, int64, string) error { return nil }
func (s *mutationStore) ListIncidents(context.Context) ([]models.Incident, error) {
	return []models.Incident{}, nil
}
func (s *mutationStore) CreateIncident(_ context.Context, item models.Incident) (models.Incident, error) {
	s.incidentCreated = item
	return item, nil
}
func (s *mutationStore) UpdateIncident(_ context.Context, id int64, item models.Incident) (models.Incident, error) {
	s.incidentUpdated = item
	s.incidentUpdated.ID = id
	return s.incidentUpdated, nil
}
func (s *mutationStore) DeleteIncident(_ context.Context, id int64) error {
	s.incidentDeleted = id
	return nil
}
func (s *mutationStore) GetIncidentStatus(context.Context, int64) (string, error) {
	return "新建", nil
}
func (s *mutationStore) UpdateIncidentStatus(context.Context, int64, string) error { return nil }

func TestAssetMutationRoutesUpdateAndDeleteByID(t *testing.T) {
	store := &mutationStore{}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
	if err != nil {
		t.Fatal(err)
	}

	putReq := httptest.NewRequest(http.MethodPut, "/api/assets/42", strings.NewReader(`{"assetNo":"ASSET-42","type":"物理机","cpuArch":"x86_64","business":"支付服务","ipv4":"10.0.0.42","environment":"生产","os":"Ubuntu","networkZone":"prod-app","cpu":"8C","memory":"32GB","disk":"1TB","deploymentInfo":"Docker","owner":"李明","connectedStatus":"已并网","status":"运行中"}`))
	putReq.Header.Set("Authorization", "Bearer "+token)
	putRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("expected asset update status 200, got %d: %s", putRec.Code, putRec.Body.String())
	}
	var updated models.Asset
	if err := json.NewDecoder(putRec.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated.ID != 42 || updated.AssetNo != "ASSET-42" || store.assetUpdated.ID != 42 {
		t.Fatalf("asset update did not preserve path id: response=%+v stored=%+v", updated, store.assetUpdated)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/assets/42", nil)
	deleteReq.Header.Set("Authorization", "Bearer "+token)
	deleteRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("expected asset delete status 204, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
	if store.assetDeleted != 42 {
		t.Fatalf("expected asset 42 to be deleted, got %d", store.assetDeleted)
	}
}

func TestAssetCreateReturnsCreatedAndTracksCreator(t *testing.T) {
	store := &mutationStore{}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(7, "ops.li", []string{auth.RoleOpsEngineer})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/assets", strings.NewReader(`{"type":"物理机","cpuArch":"x86_64","business":"支付服务","ipv4":"10.0.0.42","environment":"生产","os":"Ubuntu","networkZone":"prod-app","cpu":"8C","memory":"32GB","disk":"1TB","deploymentInfo":"Docker","owner":"李明","connectedStatus":"已并网","status":"运行中"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected asset create status 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.assetUpdated.CreatedBy != 7 {
		t.Fatalf("expected createdBy to be current user 7, got %d", store.assetUpdated.CreatedBy)
	}
}

func TestAssetMutationRejectsMissingOrInvalidFields(t *testing.T) {
	store := &mutationStore{}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
	if err != nil {
		t.Fatal(err)
	}

	missingReq := httptest.NewRequest(http.MethodPut, "/api/assets/42", strings.NewReader(`{"assetNo":"ASSET-42","type":"物理机","cpuArch":"x86_64","business":"支付服务","environment":"生产","os":"Ubuntu","networkZone":"prod-app","cpu":"8C","memory":"32GB","disk":"1TB","deploymentInfo":"Docker","owner":"李明","connectedStatus":"已并网","status":"运行中"}`))
	missingReq.Header.Set("Authorization", "Bearer "+token)
	missingRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(missingRec, missingReq)
	if missingRec.Code != http.StatusBadRequest {
		t.Fatalf("expected missing asset ipv4 status 400, got %d: %s", missingRec.Code, missingRec.Body.String())
	}

	invalidReq := httptest.NewRequest(http.MethodPost, "/api/assets", strings.NewReader(`{"type":"容器","cpuArch":"x86_64","business":"支付服务","ipv4":"10.0.0.42","environment":"生产","os":"Ubuntu","networkZone":"prod-app","cpu":"8C","memory":"32GB","disk":"1TB","deploymentInfo":"Docker","owner":"李明","connectedStatus":"已并网","status":"运行中"}`))
	invalidReq.Header.Set("Authorization", "Bearer "+token)
	invalidRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(invalidRec, invalidReq)
	if invalidRec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid asset type status 400, got %d: %s", invalidRec.Code, invalidRec.Body.String())
	}
	if store.assetUpdated.ID != 0 {
		t.Fatalf("invalid asset should be rejected before hitting store, got %+v", store.assetUpdated)
	}
}

func TestMiddlewareMutationRoutesUpdateAndDeleteByID(t *testing.T) {
	store := &mutationStore{}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
	if err != nil {
		t.Fatal(err)
	}

	putReq := httptest.NewRequest(http.MethodPut, "/api/middleware/7", strings.NewReader(`{"name":"pay-mysql","kind":"MySQL","environment":"生产","networkZone":"prod-db","endpoint":"10.0.0.7:3306","business":"支付服务","owner":"DBA","status":"运行中"}`))
	putReq.Header.Set("Authorization", "Bearer "+token)
	putRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("expected middleware update status 200, got %d: %s", putRec.Code, putRec.Body.String())
	}
	var updated models.MiddlewareInstance
	if err := json.NewDecoder(putRec.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated.ID != 7 || updated.Name != "pay-mysql" || store.middlewareUpdated.ID != 7 {
		t.Fatalf("middleware update did not preserve path id: response=%+v stored=%+v", updated, store.middlewareUpdated)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/middleware/7", nil)
	deleteReq.Header.Set("Authorization", "Bearer "+token)
	deleteRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("expected middleware delete status 204, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
	if store.middlewareDeleted != 7 {
		t.Fatalf("expected middleware 7 to be deleted, got %d", store.middlewareDeleted)
	}
}

func TestMiddlewareMutationRejectsInvalidKindAndStatus(t *testing.T) {
	store := &mutationStore{}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
	if err != nil {
		t.Fatal(err)
	}

	invalidKindReq := httptest.NewRequest(http.MethodPost, "/api/middleware", strings.NewReader(`{"name":"pay-db","kind":"MongoDB","environment":"生产","networkZone":"prod-db","endpoint":"10.0.0.7:3306","business":"支付服务","owner":"DBA","status":"运行中"}`))
	invalidKindReq.Header.Set("Authorization", "Bearer "+token)
	invalidKindRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(invalidKindRec, invalidKindReq)
	if invalidKindRec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid middleware kind 400, got %d: %s", invalidKindRec.Code, invalidKindRec.Body.String())
	}

	invalidStatusReq := httptest.NewRequest(http.MethodPost, "/api/middleware", strings.NewReader(`{"name":"pay-db","kind":"MySQL","environment":"生产","networkZone":"prod-db","endpoint":"10.0.0.7:3306","business":"支付服务","owner":"DBA","status":"未知"}`))
	invalidStatusReq.Header.Set("Authorization", "Bearer "+token)
	invalidStatusRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(invalidStatusRec, invalidStatusReq)
	if invalidStatusRec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid middleware status 400, got %d: %s", invalidStatusRec.Code, invalidStatusRec.Body.String())
	}
	if store.middlewareUpdated.ID != 0 {
		t.Fatalf("invalid middleware should be rejected before hitting store, got %+v", store.middlewareUpdated)
	}
}

func TestMiddlewareCredentialRoutesMaskAndRevealSecret(t *testing.T) {
	store := &mutationStore{credentialVerifyPassword: "Verify2026!"}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
	if err != nil {
		t.Fatal(err)
	}

	putReq := httptest.NewRequest(http.MethodPut, "/api/middleware/7/credential", strings.NewReader(`{"loginUrl":"https://db-console.local","username":"root","secret":"mysql-secret","notes":"生产主库"}`))
	putReq.Header.Set("Authorization", "Bearer "+token)
	putRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("expected middleware credential save status 200, got %d: %s", putRec.Code, putRec.Body.String())
	}

	var masked map[string]any
	if err := json.NewDecoder(putRec.Body).Decode(&masked); err != nil {
		t.Fatal(err)
	}
	if masked["secret"] != nil {
		t.Fatalf("saved middleware credential must not return secret, got %+v", masked)
	}
	if masked["hasSecret"] != true {
		t.Fatalf("saved middleware credential should expose hasSecret=true, got %+v", masked)
	}

	currentPasswordReq := httptest.NewRequest(http.MethodPost, "/api/middleware/7/credential/reveal", strings.NewReader(`{"password":"ChangeMe123!"}`))
	currentPasswordReq.Header.Set("Authorization", "Bearer "+token)
	currentPasswordRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(currentPasswordRec, currentPasswordReq)
	if currentPasswordRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected current login password to be rejected after unified credential password is configured, got %d: %s", currentPasswordRec.Code, currentPasswordRec.Body.String())
	}

	revealReq := httptest.NewRequest(http.MethodPost, "/api/middleware/7/credential/reveal", strings.NewReader(`{"password":"Verify2026!"}`))
	revealReq.Header.Set("Authorization", "Bearer "+token)
	revealRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(revealRec, revealReq)
	if revealRec.Code != http.StatusOK {
		t.Fatalf("expected middleware credential reveal status 200, got %d: %s", revealRec.Code, revealRec.Body.String())
	}
	var revealed map[string]any
	if err := json.NewDecoder(revealRec.Body).Decode(&revealed); err != nil {
		t.Fatal(err)
	}
	if revealed["secret"] != "mysql-secret" {
		t.Fatalf("revealed middleware credential should return secret after password verification, got %+v", revealed)
	}
	if len(store.credentialVerifyChecks) != 2 || store.credentialVerifyChecks[0] != "ChangeMe123!" || store.credentialVerifyChecks[1] != "Verify2026!" {
		t.Fatalf("expected credential reveal to use unified verification password checks, got %+v", store.credentialVerifyChecks)
	}
}

func TestOpsEngineerCannotReadMiddlewareCredential(t *testing.T) {
	store := &mutationStore{}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(7, "ops.li", []string{auth.RoleOpsEngineer})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/middleware/7/credential", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected ops engineer to be forbidden from middleware credentials, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestOnCallMutationRoutesUpdateAndDeleteByID(t *testing.T) {
	store := &mutationStore{}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
	if err != nil {
		t.Fatal(err)
	}

	putReq := httptest.NewRequest(http.MethodPut, "/api/oncall/9", strings.NewReader(`{"ruleType":"daily","date":"2026-06-08","primary":"李明","backup":"赵晨","swapFrom":"王敏","swapTo":"陈浩","notes":"换班已确认"}`))
	putReq.Header.Set("Authorization", "Bearer "+token)
	putRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("expected oncall update status 200, got %d: %s", putRec.Code, putRec.Body.String())
	}
	var updated models.OnCallSchedule
	if err := json.NewDecoder(putRec.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated.ID != 9 || updated.Primary != "李明" || store.oncallUpdated.ID != 9 {
		t.Fatalf("oncall update did not preserve path id: response=%+v stored=%+v", updated, store.oncallUpdated)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/oncall/9", nil)
	deleteReq.Header.Set("Authorization", "Bearer "+token)
	deleteRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("expected oncall delete status 204, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
	if store.oncallDeleted != 9 {
		t.Fatalf("expected oncall 9 to be deleted, got %d", store.oncallDeleted)
	}
}

func TestOnCallRejectsInvalidRuleType(t *testing.T) {
	store := &mutationStore{}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/oncall", strings.NewReader(`{"ruleType":"monthly","primary":"李明"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid oncall rule status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestOpsEngineerCannotWriteOnCall(t *testing.T) {
	store := &mutationStore{}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(7, "ops.li", []string{auth.RoleOpsEngineer})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/oncall", strings.NewReader(`{"ruleType":"daily","date":"2026-06-08","primary":"李明"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected ops engineer oncall write status 403, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.oncallUpdated.ID != 0 || store.oncallDeleted != 0 {
		t.Fatalf("forbidden oncall write should not mutate store: updated=%+v deleted=%d", store.oncallUpdated, store.oncallDeleted)
	}
}

func TestTaskMutationRoutesUpdateAndDeleteByID(t *testing.T) {
	store := &mutationStore{}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
	if err != nil {
		t.Fatal(err)
	}

	putReq := httptest.NewRequest(http.MethodPut, "/api/tasks/11", strings.NewReader(`{"title":"巡检确认","type":"任务","assignee":"SRE","status":"处理中","dueAt":"今天 18:00","description":"确认数据库巡检结果"}`))
	putReq.Header.Set("Authorization", "Bearer "+token)
	putRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("expected task update status 200, got %d: %s", putRec.Code, putRec.Body.String())
	}
	var updated models.Task
	if err := json.NewDecoder(putRec.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated.ID != 11 || updated.Title != "巡检确认" || store.taskUpdated.ID != 11 {
		t.Fatalf("task update did not preserve path id: response=%+v stored=%+v", updated, store.taskUpdated)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/tasks/11", nil)
	deleteReq.Header.Set("Authorization", "Bearer "+token)
	deleteRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("expected task delete status 204, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
	if store.taskDeleted != 11 {
		t.Fatalf("expected task 11 to be deleted, got %d", store.taskDeleted)
	}
}

func TestTaskCreateDefaultsTypeAndRejectsInvalidStatus(t *testing.T) {
	store := &mutationStore{}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
	if err != nil {
		t.Fatal(err)
	}

	createReq := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(`{"title":"巡检确认","assignee":"SRE"}`))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected task create status 201, got %d: %s", createRec.Code, createRec.Body.String())
	}
	if store.taskCreated.Type != "任务" || store.taskCreated.Status != "待处理" {
		t.Fatalf("expected task defaults type/status, got %+v", store.taskCreated)
	}

	invalidReq := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(`{"title":"巡检确认","status":"挂起"}`))
	invalidReq.Header.Set("Authorization", "Bearer "+token)
	invalidRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(invalidRec, invalidReq)
	if invalidRec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid task status 400, got %d: %s", invalidRec.Code, invalidRec.Body.String())
	}
}

func TestTaskMutationRejectsMissingTitle(t *testing.T) {
	store := &mutationStore{}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(`{"assignee":"SRE"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected missing task title 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.taskCreated.Title != "" || store.taskCreated.Status != "" {
		t.Fatalf("missing task title should be rejected before hitting store, got %+v", store.taskCreated)
	}
}

func TestIncidentMutationRoutesUpdateAndDeleteByID(t *testing.T) {
	store := &mutationStore{}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
	if err != nil {
		t.Fatal(err)
	}

	putReq := httptest.NewRequest(http.MethodPut, "/api/incidents/13", strings.NewReader(`{"title":"支付延迟","level":"P2","status":"处理中","owner":"运维工程师","business":"支付服务","startedAt":"2026-06-08 10:00","recoveredAt":"","summary":"交易耗时升高"}`))
	putReq.Header.Set("Authorization", "Bearer "+token)
	putRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("expected incident update status 200, got %d: %s", putRec.Code, putRec.Body.String())
	}
	var updated models.Incident
	if err := json.NewDecoder(putRec.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated.ID != 13 || updated.Title != "支付延迟" || store.incidentUpdated.ID != 13 {
		t.Fatalf("incident update did not preserve path id: response=%+v stored=%+v", updated, store.incidentUpdated)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/incidents/13", nil)
	deleteReq.Header.Set("Authorization", "Bearer "+token)
	deleteRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("expected incident delete status 204, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
	if store.incidentDeleted != 13 {
		t.Fatalf("expected incident 13 to be deleted, got %d", store.incidentDeleted)
	}
}

func TestIncidentCreateDefaultsAndRejectsInvalidLevel(t *testing.T) {
	store := &mutationStore{}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
	if err != nil {
		t.Fatal(err)
	}

	createReq := httptest.NewRequest(http.MethodPost, "/api/incidents", strings.NewReader(`{"title":"支付延迟"}`))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected incident create status 201, got %d: %s", createRec.Code, createRec.Body.String())
	}
	if store.incidentCreated.Level != "P3" || store.incidentCreated.Status != "新建" {
		t.Fatalf("expected incident defaults level/status, got %+v", store.incidentCreated)
	}

	invalidReq := httptest.NewRequest(http.MethodPost, "/api/incidents", strings.NewReader(`{"title":"支付延迟","level":"P0"}`))
	invalidReq.Header.Set("Authorization", "Bearer "+token)
	invalidRec := httptest.NewRecorder()
	server.Routes().ServeHTTP(invalidRec, invalidReq)
	if invalidRec.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid incident level 400, got %d: %s", invalidRec.Code, invalidRec.Body.String())
	}
}

func TestIncidentMutationRejectsMissingTitle(t *testing.T) {
	store := &mutationStore{}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/incidents", strings.NewReader(`{"level":"P3"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected missing incident title 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.incidentCreated.Title != "" || store.incidentCreated.Status != "" {
		t.Fatalf("missing incident title should be rejected before hitting store, got %+v", store.incidentCreated)
	}
}
