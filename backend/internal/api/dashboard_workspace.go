package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"opscore/backend/internal/auth"
	"opscore/backend/internal/models"
)

type dashboardWorkspacePersistence interface {
	DashboardQueues(context.Context) (models.DashboardQueues, error)
}

var dashboardTimezone = time.FixedZone("Asia/Shanghai", 8*60*60)

func (s *Server) requireDashboardRead(next http.Handler) http.Handler {
	return s.requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		roles := claimsFrom(r.Context()).Roles
		for _, permission := range []string{auth.PermissionAssetRead, auth.PermissionIncidentRead, auth.PermissionTaskRead, auth.PermissionOnCallRead} {
			if !auth.HasPermission(roles, permission) {
				writeError(w, http.StatusForbidden, errors.New("forbidden"))
				return
			}
		}
		next.ServeHTTP(w, r)
	}))
}

func (s *Server) dashboardWorkspace(w http.ResponseWriter, r *http.Request) {
	if r.URL.RawQuery != "" {
		writeError(w, http.StatusBadRequest, errors.New("workspace does not accept query parameters; use dashboard/preview for demo trends"))
		return
	}
	reader, ok := s.store.(dashboardWorkspacePersistence)
	if !ok {
		writeInternalError(w, errors.New("dashboard workspace persistence is unavailable"))
		return
	}
	statistics, err := s.store.Dashboard(r.Context())
	if err != nil {
		writeInternalError(w, err)
		return
	}
	queues, err := reader.DashboardQueues(r.Context())
	if err != nil {
		writeInternalError(w, err)
		return
	}
	state, err := s.store.GetDutyCenter(r.Context())
	if err != nil {
		writeInternalError(w, err)
		return
	}
	now := time.Now().In(dashboardTimezone)
	writeJSON(w, http.StatusOK, models.DashboardWorkspace{
		Source: "real", GeneratedAt: now, Statistics: statistics,
		BusinessSystems: queues.BusinessSystems, Incidents: queues.Incidents, Tasks: queues.Tasks,
		Duty: dashboardDuty(state, now),
		Trend: models.DashboardTrend{
			Source: "real", Status: "unavailable", System: "all", Period: "today",
			From: now.Format(time.DateOnly), To: now.Format(time.DateOnly), Timezone: dashboardTimezone.String(),
			Granularity: "hour", Metric: "coreServiceAvailability", Unit: "%",
			Definition: "成功请求 / 总请求；仅关联核心服务，不包含资产状态健康率", Target: 99.9,
			Points: []models.DashboardTrendPoint{}, Notice: "真实服务观测待接入，无有效样本，不生成当前或未来观测值。",
		},
	})
}

func dashboardDuty(state models.DutyCenterState, now time.Time) models.DashboardDuty {
	date := now.Format(time.DateOnly)
	data := models.DashboardDuty{
		Source: "real", Revision: state.Revision, Date: date,
		CurrentPeople:    append([]models.DutyCurrentPerson{}, state.Data.CurrentPeople...),
		TodayAssignment:  state.Data.Assignments[date],
		NextAssignment:   state.Data.Assignments[now.AddDate(0, 0, 1).Format(time.DateOnly)],
		Handovers:        append([]models.DutyHandover{}, state.Data.Handovers...),
		PendingHandovers: []models.DutyHandover{},
	}
	for i := range data.CurrentPeople {
		person := &data.CurrentPeople[i]
		if data.Primary == nil && (person.Role == "主值班" || person.Role == "primary") {
			data.Primary = person
		}
		if data.Backup == nil && (strings.HasPrefix(person.Role, "备") || person.Role == "backup") {
			data.Backup = person
		}
	}
	for _, slot := range []struct {
		name   string
		role   string
		target **models.DutyCurrentPerson
	}{
		{data.TodayAssignment.Primary, "主值班", &data.Primary},
		{data.TodayAssignment.Backup, "备份值班", &data.Backup},
	} {
		if *slot.target != nil || slot.name == "" {
			continue
		}
		for _, member := range state.Data.Members {
			if member.Name == slot.name {
				*slot.target = &models.DutyCurrentPerson{ID: member.ID, Name: member.Name, Role: slot.role, Team: member.Team, Phone: member.Phone, Status: "已排班"}
				break
			}
		}
	}
	for _, item := range data.Handovers {
		if !item.Complete {
			data.PendingHandovers = append(data.PendingHandovers, item)
		}
	}
	data.PendingCount = len(data.PendingHandovers)
	return data
}
