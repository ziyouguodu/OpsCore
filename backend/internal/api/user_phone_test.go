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

func TestValidateUserMutationPhone(t *testing.T) {
	for _, test := range []struct {
		phone string
		valid bool
	}{
		{"", true}, {"123", true}, {"13800138000", true}, {"+86 138-0013-8000", true},
		{"010-12345678", true}, {strings.Repeat("1", 20), true},
		{"1" + strings.Repeat(" ", 37) + "23", true},
		{"12", false}, {strings.Repeat("1", 21), false},
		{"1" + strings.Repeat(" ", 38) + "23", false},
		{" ", false}, {"+", false}, {"++123", false}, {"12+3", false},
		{"１２３", false}, {"123\n", false}, {"123\t456", false}, {"123<script>", false},
	} {
		t.Run(test.phone, func(t *testing.T) {
			item := models.UserMutation{Username: "ops", DisplayName: "运维", Roles: []string{auth.RoleOpsEngineer}, Phone: &test.phone}
			err := validateUserMutation(item, false)
			if (err == nil) != test.valid {
				t.Fatalf("unexpected validity: %v", err)
			}
			if err != nil && err.Error() != "phone must contain 3 to 20 digits, optional leading +, spaces or hyphens, and at most 40 characters" {
				t.Fatal("phone validation error must not reflect input")
			}
		})
	}
	if err := validateUserMutation(models.UserMutation{Username: "ops", DisplayName: "运维", Roles: []string{auth.RoleOpsEngineer}}, false); err != nil {
		t.Fatalf("omitted phone must be accepted: %v", err)
	}
}

func TestUserPhoneManagementLifecycle(t *testing.T) {
	store := &mutationStore{}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{}}
	token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
	if err != nil {
		t.Fatal(err)
	}
	handler := server.Routes()
	for _, test := range []struct {
		method string
		path   string
		body   string
		status int
		phone  string
	}{
		{http.MethodPost, "/api/users", `{"username":"ops","displayName":"运维","password":"Password123!","roles":["ops_engineer"],"phone":"+86 138-0013-8000"}`, http.StatusCreated, "+86 138-0013-8000"},
		{http.MethodPut, "/api/users/10", `{"username":"ops","displayName":"运维","roles":["ops_engineer"]}`, http.StatusOK, "+86 138-0013-8000"},
		{http.MethodPut, "/api/users/10", `{"username":"ops","displayName":"运维","roles":["ops_engineer"],"phone":""}`, http.StatusOK, ""},
		{http.MethodPut, "/api/users/10", `{"username":"ops","displayName":"运维","roles":["ops_engineer"]}`, http.StatusOK, ""},
	} {
		req := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != test.status {
			t.Fatalf("unexpected user phone mutation status: %d %s", rec.Code, rec.Body.String())
		}
		var saved models.UserListItem
		if err := json.Unmarshal(rec.Body.Bytes(), &saved); err != nil {
			t.Fatal(err)
		}
		if saved.Phone != test.phone {
			t.Fatalf("unexpected saved phone: %+v", saved)
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"phone":""`) {
		t.Fatalf("management list must include phone: %d %s", rec.Code, rec.Body.String())
	}
}

func TestInvalidPhoneRejectedBeforePersistence(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut} {
		store := &mutationStore{}
		signer := auth.NewSigner("secret", time.Hour)
		server := &Server{store: store, signer: signer}
		token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
		if err != nil {
			t.Fatal(err)
		}
		path := "/api/users"
		if method == http.MethodPut {
			path += "/10"
		}
		req := httptest.NewRequest(method, path, strings.NewReader(`{"username":"ops","displayName":"运维","password":"Password123!","roles":["ops_engineer"],"phone":"sensitive-invalid-number"}`))
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		server.Routes().ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest || store.userCreated.ID != 0 || store.userUpdated.ID != 0 || strings.Contains(rec.Body.String(), "sensitive-invalid-number") {
			t.Fatalf("invalid phone must fail safely before persistence: %d %s", rec.Code, rec.Body.String())
		}
	}
}

func TestOpsPhonePrivacyAndManagementBoundary(t *testing.T) {
	store := newOpsMutationStore()
	store.userCreated = models.UserListItem{ID: 7, Phone: "+86 138-0013-8000"}
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: store, signer: signer}
	token, err := signer.Issue(7, "ops.li", []string{auth.RoleOpsEngineer})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		method string
		path   string
		status int
	}{
		{http.MethodGet, "/api/user-directory", http.StatusOK},
		{http.MethodGet, "/api/auth/me", http.StatusOK},
		{http.MethodGet, "/api/users", http.StatusForbidden},
		{http.MethodPost, "/api/users", http.StatusForbidden},
		{http.MethodPut, "/api/users/7", http.StatusForbidden},
	} {
		req := httptest.NewRequest(test.method, test.path, strings.NewReader(`{"username":"ops.li","displayName":"运维","password":"Password123!","roles":["ops_engineer"],"phone":"1234567"}`))
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		server.Routes().ServeHTTP(rec, req)
		if rec.Code != test.status || strings.Contains(rec.Body.String(), "phone") || strings.Contains(rec.Body.String(), store.userCreated.Phone) {
			t.Fatalf("unexpected phone privacy boundary for %s %s: %d %s", test.method, test.path, rec.Code, rec.Body.String())
		}
	}
	if store.userUpdated.ID != 0 || store.userCreated.ID != 7 {
		t.Fatal("ops must not mutate managed users")
	}
}

func TestCopilotDutyContextDoesNotExposePhone(t *testing.T) {
	store := newOpsMutationStore()
	store.dutyCenter.Data = models.DutyCenterData{
		Members:       []models.DutyMember{{UserID: 7, Name: "运维", Phone: "13800138000"}},
		CurrentPeople: []models.DutyCurrentPerson{{Name: "运维", Role: "主值班", Team: "平台", Phone: "13800138000"}},
	}
	server := &Server{store: store}
	result, err := server.copilotContext(context.Background(), auth.Claims{UserID: 7, Roles: []string{auth.RoleOpsEngineer}}, models.CopilotConfig{EnableOncallContext: true})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.OnCall) != 1 || strings.Contains(string(raw), "phone") || strings.Contains(string(raw), "13800138000") {
		t.Fatalf("Copilot must retain manual phone-free duty projection: %s", raw)
	}
}
