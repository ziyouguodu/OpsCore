package api

import (
	"errors"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"opscore/backend/internal/models"
)

type dashboardPreviewQuery struct {
	System string
	Period string
	From   time.Time
	To     time.Time
}

func parseDashboardPreviewQuery(r *http.Request, now time.Time) (dashboardPreviewQuery, error) {
	today := now.In(dashboardTimezone)
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, dashboardTimezone)
	query := dashboardPreviewQuery{System: "all", Period: "today", From: today, To: today}
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return query, errors.New("invalid query parameters")
	}
	for name, entries := range values {
		if name != "system" && name != "period" && name != "from" && name != "to" {
			return query, errors.New("only system, period, from and to are supported")
		}
		if len(entries) != 1 || entries[0] == "" {
			return query, errors.New("query parameters must have a single non-empty value")
		}
	}
	if values.Has("system") {
		query.System = values.Get("system")
	}
	switch query.System {
	case "all", "payment", "orders", "portal":
	default:
		return query, errors.New("system must be all, payment, orders or portal in preview")
	}
	if values.Has("period") {
		query.Period = values.Get("period")
	}
	switch query.Period {
	case "today", "7", "30":
		if values.Has("from") || values.Has("to") {
			return query, errors.New("from and to are only valid for period=custom")
		}
		if query.Period != "today" {
			days, _ := strconv.Atoi(query.Period)
			query.From = today.AddDate(0, 0, 1-days)
		}
	case "custom":
		query.From, err = time.ParseInLocation(time.DateOnly, values.Get("from"), dashboardTimezone)
		if err != nil {
			return query, errors.New("from must be a valid YYYY-MM-DD date")
		}
		query.To, err = time.ParseInLocation(time.DateOnly, values.Get("to"), dashboardTimezone)
		if err != nil {
			return query, errors.New("to must be a valid YYYY-MM-DD date")
		}
		if query.From.After(query.To) || query.To.After(today) || query.To.Sub(query.From) > 365*24*time.Hour {
			return query, errors.New("custom range must be ordered, not in the future and at most 366 days inclusive")
		}
	default:
		return query, errors.New("period must be today, 7, 30 or custom")
	}
	return query, nil
}

func (s *Server) dashboardPreview(w http.ResponseWriter, r *http.Request) {
	now := time.Now().In(dashboardTimezone)
	query, err := parseDashboardPreviewQuery(r, now)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, dashboardPreviewData(query, now))
}

func dashboardPreviewTrend(query dashboardPreviewQuery) models.DashboardTrend {
	trend := models.DashboardTrend{
		Source: "demo", Status: "demo", System: query.System, Period: query.Period,
		From: query.From.Format(time.DateOnly), To: query.To.Format(time.DateOnly), Timezone: dashboardTimezone.String(),
		Granularity: "day", Metric: "coreServiceAvailability", Unit: "%", Target: 99.9,
		Definition: "成功请求 / 总请求；仅关联核心服务，不包含资产状态健康率",
		Points:     []models.DashboardTrendPoint{}, Notice: "虚构演示数据，未接入真实监控；今日可展示全日 24 小时，不代表未来真实观测。",
	}
	count := int(query.To.Sub(query.From)/(24*time.Hour)) + 1
	if query.Period == "today" {
		trend.Granularity = "hour"
		count = 24
	}
	weekValues := []float64{99.94, 99.96, 99.95, 99.98, 99.97, 99.99, 99.98}
	for i := 0; i < count; i++ {
		at := query.From.AddDate(0, 0, i)
		label := at.Format(time.DateOnly)
		value := 99.944 + math.Sin(float64(i)*.8)*.022 + float64(i%30)*.001
		if query.Period == "7" {
			value = weekValues[i]
		}
		if query.Period == "today" {
			at = query.From.Add(time.Duration(i) * time.Hour)
			label = at.Format("15:04")
			value = 99.948 + math.Sin(float64(i)*.52)*.018 + float64(i%5)*.002
		}
		trend.Points = append(trend.Points, models.DashboardTrendPoint{
			Time: at.Format(time.RFC3339), Label: label, Availability: math.Round(value*1000) / 1000,
		})
	}
	return trend
}

func dashboardPreviewData(query dashboardPreviewQuery, now time.Time) models.DashboardPreview {
	risks := []models.DashboardPreviewRisk{
		{
			ID: "INC-1042", Level: "P1", Title: "支付网关延迟升高", Asset: "api-gateway-01", Time: "8 分钟前", Owner: "林墨", Status: "处理中",
			Analysis: "数据库连接池饱和可能放大支付链路延迟，建议先核验连接使用率与慢查询。",
			Evidence: "连接使用率 96% · 支付 P95 2.4s", Impact: "交易平台 · 支付失败率 2.1%",
			Plan: "只读核验连接池与慢查询；评估临时限流影响；提交值班长及业务负责人审批；明确错误率上升时的回滚条件。",
		},
		{
			ID: "INC-1040", Level: "P2", Title: "数据库连接池耗尽", Asset: "mysql-prod-03", Time: "24 分钟前", Owner: "李娜", Status: "待确认",
			Analysis: "连接等待与慢查询同步升高，可能存在长事务占用。需先核对阻塞链，不宜直接重启数据库。",
			Evidence: "等待连接 42 个 · 长事务 3 个", Impact: "订单、交易两个业务",
			Plan: "核对长事务与阻塞链；确认连接泄漏范围；评估调整连接上限的资源影响；经审批后实施并观察。",
		},
		{
			ID: "INC-1038", Level: "P2", Title: "门户证书即将到期", Asset: "portal.example.cn", Time: "1 小时前", Owner: "王磊", Status: "待处理",
			Analysis: "门户证书剩余 7 天有效期，尚未核实自动续期状态。建议关联维护窗口与负责人。",
			Evidence: "证书有效期 7 天 · 续期状态待核验", Impact: "企业门户 · 当前服务正常",
			Plan: "核验证书域名及托管平台；确认续期状态；安排维护窗口；预备旧证书回退并验证完整证书链。",
		},
		{
			ID: "INC-1036", Level: "P3", Title: "服务器 CPU 使用率高", Asset: "node-02", Time: "2 小时前", Owner: "陈杰", Status: "处理中",
			Analysis: "CPU 高位运行与批处理窗口重合，可能是周期性负载。建议核对进程与资源配额。",
			Evidence: "CPU 86% · 持续 12 分钟", Impact: "4 个节点 · 未发现用户请求失败",
			Plan: "核验进程与批处理计划；排查资源争用；评估错峰执行；经负责人确认后再调整。",
		},
	}
	data := models.DashboardPreview{
		Source: "demo", GeneratedAt: now, ExecutionAllowed: false,
		Notice: "全部为原型预设虚构演示，未查询生产观测、未调用 AI、未审批、未执行任何生产处置；禁止通过此接口执行操作。风险与事件独立计数，不混入真实统计。",
		BusinessSystems: []models.DashboardPreviewSystem{
			{ID: "all", Name: "全部业务系统", AssetCount: 248, ServiceCount: 38},
			{ID: "payment", Name: "支付交易平台", AssetCount: 24, ServiceCount: 8},
			{ID: "orders", Name: "订单中心", AssetCount: 31, ServiceCount: 6},
			{ID: "portal", Name: "企业门户", AssetCount: 18, ServiceCount: 4},
		},
		Alerts: models.DashboardPreviewAlerts{Source: "demo", Status: "pendingIntegration", UrgentCount: 3, LevelCounts: map[string]int{"P1": 1, "P2": 2, "P3": 0, "P4": 0}},
		Risks:  risks,
		Services: []models.DashboardPreviewService{
			{Name: "Web 服务", Status: "healthy", Availability: 99.98, P95Ms: 48},
			{Name: "数据库", Status: "warning", Availability: 99.97, P95Ms: 99},
			{Name: "中间件", Status: "healthy", Availability: 99.97, P95Ms: 36},
			{Name: "缓存", Status: "healthy", Availability: 99.98, P95Ms: 8},
			{Name: "存储", Status: "healthy", Availability: 99.96, P95Ms: 22},
		},
		Automation: models.DashboardPreviewAutomation{Source: "demo", Status: "pendingIntegration", Succeeded: 148, Failed: 2, SuccessRate: 98.7},
		Insights:   []models.DashboardPreviewInsight{}, Notifications: []models.DashboardPreviewNotification{},
		Trend: dashboardPreviewTrend(query),
	}
	for i, risk := range risks {
		data.Insights = append(data.Insights, models.DashboardPreviewInsight{
			RiskID: risk.ID, Title: risk.Title, Analysis: risk.Analysis,
			Evidence: risk.Evidence + "。演示证据，根因待人工核验。", Plan: risk.Plan, Status: "suggestionOnly",
		})
		if i < 3 {
			data.Notifications = append(data.Notifications, models.DashboardPreviewNotification{RiskID: risk.ID, Title: risk.Title, Time: risk.Time})
		}
	}
	return data
}
