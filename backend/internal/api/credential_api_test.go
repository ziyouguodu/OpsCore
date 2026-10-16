package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"opscore/backend/internal/auth"
	"opscore/backend/internal/models"
)

type credentialAPIStore struct {
	mutationStore
	assetCredential models.AssetCredential
	hasPasswordErr  error
	verifyErr       error
	setPasswordErr  error
	userErr         error
}

func (s *credentialAPIStore) GetUser(ctx context.Context, id int64) (models.User, error) {
	if s.userErr != nil {
		return models.User{}, s.userErr
	}
	return s.mutationStore.GetUser(ctx, id)
}

func (s *credentialAPIStore) GetAssetCredential(_ context.Context, id int64) (models.AssetCredential, error) {
	item := s.assetCredential
	item.AssetID = id
	return item, nil
}

func (s *credentialAPIStore) UpsertAssetCredential(_ context.Context, item models.AssetCredential) (models.AssetCredential, error) {
	s.assetCredential = item
	return item, nil
}

func (s *credentialAPIStore) HasCredentialVerificationPassword(ctx context.Context) (bool, error) {
	if s.hasPasswordErr != nil {
		return false, s.hasPasswordErr
	}
	return s.mutationStore.HasCredentialVerificationPassword(ctx)
}

func (s *credentialAPIStore) VerifyCredentialPassword(ctx context.Context, password string) (bool, error) {
	if s.verifyErr != nil {
		return false, s.verifyErr
	}
	return s.mutationStore.VerifyCredentialPassword(ctx, password)
}

func (s *credentialAPIStore) SetCredentialVerificationPassword(ctx context.Context, password string) error {
	if s.setPasswordErr != nil {
		return s.setPasswordErr
	}
	return s.mutationStore.SetCredentialVerificationPassword(ctx, password)
}

func credentialAPIRequest(handler http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.RemoteAddr = "127.0.0.1:1234"
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestCredentialAPIMasksAndRevealsOnlyWithUnifiedPassword(t *testing.T) {
	for _, resource := range []string{"assets", "middleware"} {
		t.Run(resource, func(t *testing.T) {
			store := &credentialAPIStore{mutationStore: mutationStore{credentialVerifyPassword: "Verify2026!"}}
			signer := auth.NewSigner("credential-test-secret", time.Hour)
			server := &Server{store: store, signer: signer}
			token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
			if err != nil {
				t.Fatal(err)
			}
			handler := server.Routes()
			path := "/api/" + resource + "/7/credential"
			for _, method := range []string{http.MethodPut, http.MethodGet} {
				rec := credentialAPIRequest(handler, method, path, `{"username":"root","secret":"credential-plaintext"}`, token)
				if rec.Code != http.StatusOK {
					t.Fatalf("%s: expected 200, got %d: %s", method, rec.Code, rec.Body.String())
				}
				var body map[string]any
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body["secret"] != nil || body["hasSecret"] != true || strings.Contains(rec.Body.String(), "credential-plaintext") {
					t.Fatalf("%s must mask the saved credential: %s", method, rec.Body.String())
				}
			}
			rec := credentialAPIRequest(handler, http.MethodPost, path+"/reveal", `{"password":"ChangeMe123!"}`, token)
			if rec.Code != http.StatusForbidden || strings.Contains(rec.Body.String(), "credential-plaintext") {
				t.Fatalf("wrong secondary password must return 403 without the credential, got %d: %s", rec.Code, rec.Body.String())
			}
			rec = credentialAPIRequest(handler, http.MethodGet, "/api/auth/me", "", token)
			if rec.Code != http.StatusOK {
				t.Fatalf("secondary verification failure must leave the session valid, got %d", rec.Code)
			}
			rec = credentialAPIRequest(handler, http.MethodPost, path+"/reveal", `{"password":"Verify2026!"}`, token)
			var revealed map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &revealed); err != nil {
				t.Fatal(err)
			}
			if rec.Code != http.StatusOK || revealed["secret"] != "credential-plaintext" || revealed["hasSecret"] != true {
				t.Fatalf("authorized reveal must return plaintext, got %d: %s", rec.Code, rec.Body.String())
			}
			if len(server.credentialAttempts().buckets) != 0 {
				t.Fatal("successful reveal must clear verification failures")
			}
			store.credentialVerifyPassword = ""
			rec = credentialAPIRequest(handler, http.MethodPost, path+"/reveal", `{"password":"ChangeMe123!"}`, token)
			if rec.Code != http.StatusForbidden || strings.Contains(rec.Body.String(), "credential-plaintext") {
				t.Fatalf("unconfigured unified password must not fall back to login password, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestCredentialAPIRequiresLiveSessionAndCurrentRole(t *testing.T) {
	for _, resource := range []string{"assets", "middleware"} {
		t.Run(resource, func(t *testing.T) {
			store := &credentialAPIStore{mutationStore: mutationStore{credentialVerifyPassword: "Verify2026!"}}
			signer := auth.NewSigner("credential-test-secret", time.Hour)
			server := &Server{store: store, signer: signer}
			handler := server.Routes()
			token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
			if err != nil {
				t.Fatal(err)
			}
			expiredSigner := auth.NewSigner("credential-test-secret", -time.Hour)
			expiredToken, err := expiredSigner.Issue(1, "admin", []string{auth.RoleSuperAdmin})
			if err != nil {
				t.Fatal(err)
			}
			path := "/api/" + resource + "/7/credential"
			for _, invalidToken := range []string{"", "invalid-token", expiredToken} {
				rec := credentialAPIRequest(handler, http.MethodPost, path+"/reveal", `{"password":"Verify2026!"}`, invalidToken)
				if rec.Code != http.StatusUnauthorized {
					t.Fatalf("missing, invalid or expired session must still return 401, got %d", rec.Code)
				}
			}
			store.userErr = errors.New("user no longer exists")
			rec := credentialAPIRequest(handler, http.MethodPost, path+"/reveal", `{"password":"Verify2026!"}`, token)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("deleted user session must return 401, got %d", rec.Code)
			}
			store.userErr = nil
			store.userProfile = models.User{ID: 1, Username: "admin", Roles: []string{auth.RoleOpsEngineer}}
			for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodPost} {
				requestPath := path
				body := `{"secret":"blocked-secret"}`
				if method == http.MethodPost {
					requestPath += "/reveal"
					body = `{"password":"Verify2026!"}`
				}
				rec := credentialAPIRequest(handler, method, requestPath, body, token)
				if rec.Code != http.StatusForbidden {
					t.Fatalf("%s must enforce current database role instead of token role, got %d", method, rec.Code)
				}
			}
			if len(store.credentialVerifyChecks) != 0 || store.assetCredential.Secret != "" || store.middlewareCred.Secret != "" {
				t.Fatal("unauthorized requests must not access or save credentials")
			}
		})
	}
}

func TestCredentialVerificationConfigReturnsOnlyPresence(t *testing.T) {
	store := &credentialAPIStore{}
	signer := auth.NewSigner("credential-test-secret", time.Hour)
	server := &Server{store: store, signer: signer}
	token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
	if err != nil {
		t.Fatal(err)
	}
	handler := server.Routes()
	path := "/api/security/credential-verification"
	for _, test := range []struct {
		method  string
		present bool
	}{
		{http.MethodGet, false},
		{http.MethodPut, true},
		{http.MethodGet, true},
	} {
		rec := credentialAPIRequest(handler, test.method, path, `{"password":"Verify2026!"}`, token)
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusOK || len(body) != 1 || body["hasPassword"] != test.present {
			t.Fatalf("%s response must contain only password presence, got %d: %s", test.method, rec.Code, rec.Body.String())
		}
	}
	store.userProfile = models.User{ID: 1, Username: "admin", Roles: []string{auth.RoleOpsEngineer}}
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		rec := credentialAPIRequest(handler, method, path, `{"password":"Blocked2026!"}`, token)
		if rec.Code != http.StatusForbidden || store.credentialVerifyConfigured != "Verify2026!" {
			t.Fatalf("%s config must reject ordinary roles without changing the password, got %d", method, rec.Code)
		}
	}
}

func TestCredentialVerificationStoreErrorsAreSafeAndDoNotCountAsFailures(t *testing.T) {
	for _, resource := range []string{"assets", "middleware"} {
		for _, stage := range []string{"presence", "verify"} {
			t.Run(resource+"/"+stage, func(t *testing.T) {
				store := &credentialAPIStore{mutationStore: mutationStore{credentialVerifyPassword: "Verify2026!"}}
				storeErr := errors.New("private database failure detail")
				if stage == "presence" {
					store.hasPasswordErr = storeErr
				} else {
					store.verifyErr = storeErr
				}
				signer := auth.NewSigner("credential-test-secret", time.Hour)
				server := &Server{store: store, signer: signer}
				token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
				if err != nil {
					t.Fatal(err)
				}
				handler := server.Routes()
				limiter := server.credentialAttempts()
				for i := 0; i < 4; i++ {
					limiter.Failure("1|127.0.0.1")
				}
				path := "/api/" + resource + "/7/credential/reveal"
				for i := 0; i < 6; i++ {
					rec := credentialAPIRequest(handler, http.MethodPost, path, `{"password":"Verify2026!"}`, token)
					if rec.Code != http.StatusInternalServerError || rec.Body.String() != "{\"error\":\"internal server error\"}\n" {
						t.Fatalf("store failure must return safe 500, got %d: %s", rec.Code, rec.Body.String())
					}
				}
				if len(limiter.buckets["1|127.0.0.1"].failures) != 4 {
					t.Fatal("store errors must neither increment nor clear password failures")
				}
				store.hasPasswordErr = nil
				store.verifyErr = nil
				rec := credentialAPIRequest(handler, http.MethodPost, path, `{"password":"wrong"}`, token)
				if rec.Code != http.StatusForbidden {
					t.Fatalf("next invalid password must be the fifth failure, got %d", rec.Code)
				}
				rec = credentialAPIRequest(handler, http.MethodPost, path, `{"password":"Verify2026!"}`, token)
				if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") != "600" {
					t.Fatalf("five password failures must still lock for ten minutes, got %d retry=%s", rec.Code, rec.Header().Get("Retry-After"))
				}
			})
		}
	}
}

func TestCredentialVerificationConfigStoreErrorsReturnSafe500(t *testing.T) {
	storeErr := errors.New("private database failure detail")
	store := &credentialAPIStore{hasPasswordErr: storeErr, setPasswordErr: storeErr}
	signer := auth.NewSigner("credential-test-secret", time.Hour)
	server := &Server{store: store, signer: signer}
	token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
	if err != nil {
		t.Fatal(err)
	}
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		rec := credentialAPIRequest(server.Routes(), method, "/api/security/credential-verification", `{"password":"Verify2026!"}`, token)
		if rec.Code != http.StatusInternalServerError || rec.Body.String() != "{\"error\":\"internal server error\"}\n" {
			t.Fatalf("%s store failure must return safe 500, got %d: %s", method, rec.Code, rec.Body.String())
		}
	}
}

func TestCredentialRevealSharesFiveFailureTenMinuteLockAcrossResources(t *testing.T) {
	store := &credentialAPIStore{mutationStore: mutationStore{credentialVerifyPassword: "Verify2026!"}}
	signer := auth.NewSigner("credential-test-secret", time.Hour)
	server := &Server{store: store, signer: signer}
	token, err := signer.Issue(1, "admin", []string{auth.RoleSuperAdmin})
	if err != nil {
		t.Fatal(err)
	}
	handler := server.Routes()
	now := time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)
	server.credentialAttempts().now = func() time.Time { return now }
	paths := []string{"/api/assets/7/credential/reveal", "/api/middleware/7/credential/reveal"}
	for i := 0; i < 5; i++ {
		rec := credentialAPIRequest(handler, http.MethodPost, paths[i%2], `{"password":"wrong"}`, token)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("failure %d must return 403, got %d", i+1, rec.Code)
		}
	}
	for _, path := range paths {
		rec := credentialAPIRequest(handler, http.MethodPost, path, `{"password":"Verify2026!"}`, token)
		if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") != "600" {
			t.Fatalf("both resources must share the ten minute lock, got %d retry=%s", rec.Code, rec.Header().Get("Retry-After"))
		}
	}
	if len(store.credentialVerifyChecks) != 5 {
		t.Fatal("locked requests must not reach password verification")
	}
	now = now.Add(9 * time.Minute)
	rec := credentialAPIRequest(handler, http.MethodPost, paths[0], `{"password":"Verify2026!"}`, token)
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") != "60" {
		t.Fatalf("lock must remain active before ten minutes, got %d retry=%s", rec.Code, rec.Header().Get("Retry-After"))
	}
	now = now.Add(time.Minute)
	for _, path := range paths {
		rec := credentialAPIRequest(handler, http.MethodPost, path, `{"password":"Verify2026!"}`, token)
		if rec.Code != http.StatusOK {
			t.Fatalf("correct password must work after the lock expires, got %d", rec.Code)
		}
	}
}
