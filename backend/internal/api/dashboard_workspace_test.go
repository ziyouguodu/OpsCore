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

type dashboardWorkspaceStore struct {
	mutationStore
	statistics    models.Dashboard
	queues        models.DashboardQueues
	statisticsErr error
	queuesErr     error
	readCalls     int
}

func (s *dashboardWorkspaceStore) Dashboard(context.Context) (models.Dashboard, error) {
	s.readCalls++
	return s.statistics, s.statisticsErr
}

func (s *dashboardWorkspaceStore) DashboardQueues(context.Context) (models.DashboardQueues, error) {
	s.readCalls++
	return s.queues, s.queuesErr
}

func dashboardRequest(t *testing.T, database persistence, endpoint string, authenticated bool) *httptest.ResponseRecorder {
	t.Helper()
	signer := auth.NewSigner("secret", time.Hour)
	server := &Server{store: database, signer: signer}
	req := httptest.NewRequest(http.MethodGet, endpoint, nil)
	if authenticated {
		token, err := signer.Issue(7, "admin", []string{auth.RoleSuperAdmin})
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)
	return rec
}

func TestDashboardEndpointsEnforceAuthenticationAndCurrentPermissions(t *testing.T) {
	for _, endpoint := range []string{"/api/dashboard/workspace", "/api/dashboard/preview"} {
		t.Run(endpoint, func(t *testing.T) {
			for _, tc := range []struct {
				name          string
				authenticated bool
				user          models.User
				want          int
			}{
				{name: "missing token", want: http.StatusUnauthorized},
				{name: "first password", authenticated: true, user: models.User{ID: 7, Roles: []string{auth.RoleSuperAdmin}, MustChangePassword: true}, want: http.StatusForbidden},
				{name: "revoked roles override token", authenticated: true, user: models.User{ID: 7, Roles: []string{}}, want: http.StatusForbidden},
				{name: "ops reader", authenticated: true, user: models.User{ID: 7, Roles: []string{auth.RoleOpsEngineer}}, want: http.StatusOK},
				{name: "admin reader", authenticated: true, user: models.User{ID: 7, Roles: []string{auth.RoleSuperAdmin}}, want: http.StatusOK},
			} {
				t.Run(tc.name, func(t *testing.T) {
					database := &dashboardWorkspaceStore{mutationStore: mutationStore{userProfile: tc.user}}
					rec := dashboardRequest(t, database, endpoint, tc.authenticated)
					if rec.Code != tc.want {
						t.Fatalf("expected %d, got %d: %s", tc.want, rec.Code, rec.Body.String())
					}
					if tc.want != http.StatusOK && database.readCalls != 0 {
						t.Fatal("unauthorized request read dashboard data")
					}
				})
			}
		})
	}
}

func TestDashboardWorkspaceUsesRealStatisticsQueuesAndDuty(t *testing.T) {
	today := time.Now().In(dashboardTimezone).Format(time.DateOnly)
	database := &dashboardWorkspaceStore{
		statistics: models.Dashboard{AssetCount: 1200, ActiveIncidentCount: 80, ActiveTaskCount: 900},
		queues: models.DashboardQueues{
			BusinessSystems: []string{"仅存在于全局数据的业务"},
			Incidents:       []models.Incident{{ID: 101, Title: "全局 P1", Level: "P1", Status: "处理中"}},
			Tasks:           []models.Task{{ID: 201, Title: "未完成任务", Status: "待确认"}},
		},
		mutationStore: mutationStore{dutyCenter: models.DutyCenterState{Revision: 9, Data: models.DutyCenterData{
			Assignments: map[string]models.DutyAssignment{today: {Primary: "真实主值", Backup: "真实备值"}},
			CurrentPeople: []models.DutyCurrentPerson{
				{ID: "primary", Name: "真实主值", Role: "主值班", Since: "09:30", Until: "次日 09:30"},
				{ID: "backup", Name: "真实备值", Role: "备份值班", Since: "10:00", Until: "22:00"},
			},
			Handovers: []models.DutyHandover{
				{ID: "pending", Content: "真实待交接事项", Complete: false},
				{ID: "done", Content: "已完成交接", Complete: true},
			},
		}}},
	}
	rec := dashboardRequest(t, database, "/api/dashboard/workspace", true)
	var data models.DashboardWorkspace
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &data) != nil {
		t.Fatalf("unexpected response: %d %s", rec.Code, rec.Body.String())
	}
	if data.Source != "real" || data.Statistics.AssetCount != 1200 || data.Statistics.ActiveIncidentCount != 80 || data.Statistics.ActiveTaskCount != 900 {
		t.Fatalf("statistics not reused: %+v", data)
	}
	if len(data.BusinessSystems) != 1 || data.BusinessSystems[0] != "仅存在于全局数据的业务" || data.Incidents[0].ID != 101 || data.Tasks[0].ID != 201 {
		t.Fatalf("workspace did not use global queues: %+v", data)
	}
	if data.Duty.Source != "real" || data.Duty.Revision != 9 || data.Duty.Primary.Name != "真实主值" || data.Duty.Backup.Name != "真实备值" || data.Duty.Primary.Since != "09:30" || data.Duty.Primary.Until != "次日 09:30" {
		t.Fatalf("duty names or shifts are not real: %+v", data.Duty)
	}
	if data.Duty.PendingCount != 1 || len(data.Duty.PendingHandovers) != 1 || len(data.Duty.Handovers) != 2 || data.Duty.TodayAssignment.Primary != "真实主值" {
		t.Fatalf("unexpected handover: %+v", data.Duty)
	}
	if data.Trend.Source != "real" || data.Trend.Status != "unavailable" || len(data.Trend.Points) != 0 || data.Trend.IncludesAssetHealth {
		t.Fatalf("real observations were fabricated: %+v", data.Trend)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &fields); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"alerts", "risks", "services", "automation", "insights", "notifications"} {
		if _, exists := fields[name]; exists {
			t.Fatalf("demo module %s leaked into workspace", name)
		}
	}
	if len(database.auditEvents) != 0 || database.taskCreated.ID != 0 || database.incidentCreated.ID != 0 {
		t.Fatal("read-only dashboard wrote data")
	}
}

func TestDashboardDutyFallsBackToTodayMembersWithoutInventingShifts(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, dashboardTimezone)
	state := models.DutyCenterState{Data: models.DutyCenterData{
		Members:     []models.DutyMember{{ID: "p", Name: "当日主值", Team: "运维", Phone: "+86 138-0013-8000"}, {ID: "b", Name: "当日备值", Team: "运维", Phone: "010-12345678"}},
		Assignments: map[string]models.DutyAssignment{"2026-10-16": {Primary: "当日主值", Backup: "当日备值"}},
	}}
	data := dashboardDuty(state, now)
	if data.Primary == nil || data.Backup == nil || data.Primary.Name != "当日主值" || data.Backup.ID != "b" || data.Backup.Since != "" || data.Backup.Until != "" {
		t.Fatalf("expected real roster fallback without fabricated shifts: %+v", data)
	}
	if data.Primary.Phone != "+86 138-0013-8000" || data.Backup.Phone != "010-12345678" {
		t.Fatalf("roster fallback must retain authoritative member phones: %+v", data)
	}
	state.Data.CurrentPeople = []models.DutyCurrentPerson{{ID: "override", Name: "已接班人员", Role: "主值班", Since: "10:00"}}
	data = dashboardDuty(state, now)
	if data.Primary.ID != "override" || data.Primary.Since != "10:00" || data.Backup.Name != "当日备值" {
		t.Fatalf("current shift must take precedence: %+v", data)
	}
	state.Data.Assignments["2026-10-16"] = models.DutyAssignment{Backup: "不存在的人员"}
	data = dashboardDuty(state, now)
	if data.Backup != nil {
		t.Fatal("unknown roster member must not become a current backup")
	}
}

func TestDashboardDutyEmptyDoesNotInventNamesOrShifts(t *testing.T) {
	data := dashboardDuty(models.DutyCenterState{}, time.Now().In(dashboardTimezone))
	if data.Primary != nil || data.Backup != nil || data.PendingCount != 0 || data.CurrentPeople == nil || data.PendingHandovers == nil || data.Handovers == nil {
		t.Fatalf("unexpected empty duty data: %+v", data)
	}
}

func TestDashboardWorkspaceRejectsDemoAndQueryParameters(t *testing.T) {
	for _, query := range []string{"?demo=true", "?system=payment", "?period=7", "?from=2026-01-01", "?to=2026-01-02", "?bad;query"} {
		database := &dashboardWorkspaceStore{}
		rec := dashboardRequest(t, database, "/api/dashboard/workspace"+query, true)
		if rec.Code != http.StatusBadRequest || database.readCalls != 0 {
			t.Fatalf("expected bad request without data reads for %s, got %d", query, rec.Code)
		}
	}
}

func TestDashboardWorkspaceErrorsAreSanitized(t *testing.T) {
	secret := errors.New("postgres secret-host internal detail")
	for _, database := range []persistence{
		&dashboardWorkspaceStore{statisticsErr: secret},
		&dashboardWorkspaceStore{queuesErr: secret},
		&dashboardWorkspaceStore{mutationStore: mutationStore{dutyCenterErr: secret}},
		&mutationStore{},
	} {
		rec := dashboardRequest(t, database, "/api/dashboard/workspace", true)
		if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "secret-host") {
			t.Fatalf("expected sanitized error, got %d: %s", rec.Code, rec.Body.String())
		}
	}
}

func TestDashboardPreviewIsIsolatedReadOnlyPrototypeData(t *testing.T) {
	database := &dashboardWorkspaceStore{statisticsErr: errors.New("must not read real statistics"), queuesErr: errors.New("must not read real queues")}
	rec := dashboardRequest(t, database, "/api/dashboard/preview", true)
	var data models.DashboardPreview
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &data) != nil {
		t.Fatalf("unexpected response: %d %s", rec.Code, rec.Body.String())
	}
	if data.Source != "demo" || data.ExecutionAllowed || !strings.Contains(data.Notice, "禁止") || !strings.Contains(data.Notice, "未执行") {
		t.Fatalf("demo boundary missing: %+v", data)
	}
	if data.Alerts.Source != "demo" || data.Alerts.UrgentCount != 3 || data.Alerts.LevelCounts["P1"] != 1 || len(data.Risks) != 4 || data.Risks[0].Title != "支付网关延迟升高" {
		t.Fatalf("prototype alerts or risks missing: %+v", data)
	}
	if len(data.Services) != 5 || data.Services[0].P95Ms != 48 || data.Services[1].Availability != 99.97 || data.Automation.Succeeded != 148 || data.Automation.Failed != 2 || data.Automation.SuccessRate != 98.7 {
		t.Fatalf("prototype service or automation fixture differs: %+v", data)
	}
	if len(data.Insights) != 4 || data.Insights[0].Status != "suggestionOnly" || len(data.Notifications) != 3 || len(data.BusinessSystems) != 4 {
		t.Fatalf("prototype insights, notifications or systems missing: %+v", data)
	}
	if database.readCalls != 0 || len(database.auditEvents) != 0 {
		t.Fatal("preview touched real business data")
	}
	if strings.Contains(rec.Body.String(), `"statistics"`) || strings.Contains(rec.Body.String(), `"assetHealthyCount"`) {
		t.Fatal("preview mixed in real statistics or asset health")
	}
}

func TestDashboardPreviewTimeRangesAndServiceOnlyMetric(t *testing.T) {
	now := time.Date(2026, 10, 14, 10, 42, 0, 0, dashboardTimezone)
	for _, tc := range []struct {
		query, from, to, first, last, granularity string
		count                                     int
	}{
		{"", "2026-10-14", "2026-10-14", "00:00", "23:00", "hour", 24},
		{"?period=7", "2026-10-08", "2026-10-14", "2026-10-08", "2026-10-14", "day", 7},
		{"?period=30", "2026-09-15", "2026-10-14", "2026-09-15", "2026-10-14", "day", 30},
		{"?period=custom&from=2026-09-29&to=2026-10-02", "2026-09-29", "2026-10-02", "2026-09-29", "2026-10-02", "day", 4},
		{"?period=custom&from=2026-10-01&to=2026-10-01", "2026-10-01", "2026-10-01", "2026-10-01", "2026-10-01", "day", 1},
		{"?period=custom&from=2026-09-01&to=2026-10-01", "2026-09-01", "2026-10-01", "2026-09-01", "2026-10-01", "day", 31},
		{"?system=payment&period=today", "2026-10-14", "2026-10-14", "00:00", "23:00", "hour", 24},
	} {
		t.Run(tc.query, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/dashboard/preview"+tc.query, nil)
			query, err := parseDashboardPreviewQuery(req, now)
			if err != nil {
				t.Fatal(err)
			}
			trend := dashboardPreviewTrend(query)
			if trend.Source != "demo" || trend.From != tc.from || trend.To != tc.to || len(trend.Points) != tc.count || trend.Granularity != tc.granularity || trend.Points[0].Label != tc.first || trend.Points[tc.count-1].Label != tc.last {
				t.Fatalf("wrong time range: %+v", trend)
			}
			if trend.Metric != "coreServiceAvailability" || trend.Unit != "%" || trend.IncludesAssetHealth || !strings.Contains(trend.Definition, "成功请求 / 总请求") || !strings.Contains(trend.Notice, "不代表未来真实观测") {
				t.Fatalf("service-only metric or demo boundary missing: %+v", trend)
			}
			for i, point := range trend.Points {
				at, err := time.Parse(time.RFC3339, point.Time)
				if err != nil || point.Availability < 0 || point.Availability > 100 {
					t.Fatalf("invalid point: %+v", point)
				}
				if i > 0 {
					previous, _ := time.Parse(time.RFC3339, trend.Points[i-1].Time)
					step := 24 * time.Hour
					if trend.Granularity == "hour" {
						step = time.Hour
					}
					if at.Sub(previous) != step {
						t.Fatalf("nonconsecutive time labels: %+v", trend.Points)
					}
				}
			}
			if strings.Contains(tc.query, "system=payment") && trend.System != "payment" {
				t.Fatal("system scope was dropped")
			}
		})
	}
}

func TestDashboardPreviewRoutesReturnRequestedRanges(t *testing.T) {
	for _, tc := range []struct {
		query string
		count int
	}{
		{"", 24},
		{"?period=7&system=orders", 7},
		{"?period=30&system=portal", 30},
		{"?period=custom&from=2020-02-28&to=2020-03-01&system=payment", 3},
	} {
		rec := dashboardRequest(t, &dashboardWorkspaceStore{}, "/api/dashboard/preview"+tc.query, true)
		var data models.DashboardPreview
		if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &data) != nil || len(data.Trend.Points) != tc.count {
			t.Fatalf("unexpected route response for %s: %d %s", tc.query, rec.Code, rec.Body.String())
		}
		if tc.count == 3 && (data.Trend.Points[1].Label != "2020-02-29" || data.Trend.System != "payment") {
			t.Fatalf("custom leap-day range or system lost: %+v", data.Trend)
		}
	}
}

func TestDashboardPreviewUsesShanghaiCalendar(t *testing.T) {
	now := time.Date(2026, 10, 13, 20, 0, 0, 0, time.UTC)
	query, err := parseDashboardPreviewQuery(httptest.NewRequest(http.MethodGet, "/", nil), now)
	if err != nil || query.From.Format(time.DateOnly) != "2026-10-14" {
		t.Fatalf("expected Shanghai current date, got %+v %v", query, err)
	}
}

func TestDashboardPreviewRejectsInvalidParameters(t *testing.T) {
	for _, query := range []string{
		"system=unknown", "system=", "system=all&system=payment", "period=", "period=hour", "period=7&period=30",
		"from=2026-01-01", "to=2026-01-01", "metric=assetHealth", "demo=true", "bad;query",
		"period=custom", "period=custom&from=2026-01-01", "period=custom&to=2026-01-02",
		"period=custom&from=2026-02-30&to=2026-03-01", "period=custom&from=bad&to=bad",
		"period=custom&from=2026-02-03&to=2026-02-01", "period=custom&from=2099-01-01&to=2099-01-02",
		"period=custom&from=2020-01-01&to=2026-01-01", "period=today&from=2026-01-01&to=2026-01-01",
	} {
		t.Run(query, func(t *testing.T) {
			database := &dashboardWorkspaceStore{}
			rec := dashboardRequest(t, database, "/api/dashboard/preview?"+query, true)
			if rec.Code != http.StatusBadRequest || database.readCalls != 0 {
				t.Fatalf("expected validation error without business reads, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}
