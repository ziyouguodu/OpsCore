package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"opscore/backend/internal/auth"
	"opscore/backend/internal/config"
	"opscore/backend/internal/models"
)

func TestParseListQueryNormalizesPaginationAndFilters(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/assets?page=2&pageSize=25&keyword=pay&type=%E7%89%A9%E7%90%86%E6%9C%BA&environment=%E7%94%9F%E4%BA%A7&sort=assetNo&order=asc", nil)
	query, err := parseListQuery(req)
	if err != nil {
		t.Fatal(err)
	}
	if query.Page != 2 || query.PageSize != 25 || query.Keyword != "pay" || query.Type != "物理机" || query.Environment != "生产" || query.Sort != "assetNo" || query.Order != "asc" {
		t.Fatalf("unexpected parsed query: %+v", query)
	}
}

func TestParseListQueryParsesAndDeduplicatesMultipleIPs(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/assets?ips=192.0.2.10%2C2001%3ADB8%3A%3A1%0A192.0.2.10%3B198.51.100.4", nil)
	query, err := parseListQuery(req)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"192.0.2.10", "2001:db8::1", "198.51.100.4"}
	if len(query.IPs) != len(want) {
		t.Fatalf("expected %d unique IPs, got %+v", len(want), query.IPs)
	}
	for index := range want {
		if query.IPs[index] != want[index] {
			t.Fatalf("expected parsed IPs %+v, got %+v", want, query.IPs)
		}
	}
}

func TestAssetsEndpointReturnsPageMetadata(t *testing.T) {
	store := &mutationStore{userProfile: models.User{ID: 7, Username: "ops.li", Roles: []string{auth.RoleOpsEngineer}}}
	signer := auth.NewSigner("pagination-test-secret", time.Hour)
	server := &Server{store: store, signer: signer, cfg: config.Config{CORSOrigin: "http://localhost:5173"}}
	token, err := signer.Issue(7, "ops.li", []string{auth.RoleOpsEngineer})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/assets?page=2&pageSize=20&environment=%E7%94%9F%E4%BA%A7", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected paginated assets response, got %d: %s", rec.Code, rec.Body.String())
	}
	var result models.PageResult[models.Asset]
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Total != 21 || result.Page != 2 || result.PageCount != 2 || len(result.Items) != 1 {
		t.Fatalf("unexpected page response: %+v", result)
	}
	if store.assetListQuery.Environment != "生产" {
		t.Fatalf("expected environment filter to reach store: %+v", store.assetListQuery)
	}
}

func TestParseListQueryRejectsInvalidBounds(t *testing.T) {
	for _, target := range []string{"/api/assets?page=0", "/api/assets?pageSize=101", "/api/assets?order=random"} {
		req := httptest.NewRequest("GET", target, nil)
		if _, err := parseListQuery(req); err == nil {
			t.Fatalf("expected invalid query to fail: %s", target)
		}
	}
}
