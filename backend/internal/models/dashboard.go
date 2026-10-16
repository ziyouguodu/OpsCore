package models

import "time"

type DashboardQueues struct {
	BusinessSystems []string   `json:"businessSystems"`
	Incidents       []Incident `json:"incidents"`
	Tasks           []Task     `json:"tasks"`
}

type DashboardDuty struct {
	Source           string              `json:"source"`
	Revision         int64               `json:"revision"`
	Date             string              `json:"date"`
	Primary          *DutyCurrentPerson  `json:"primary"`
	Backup           *DutyCurrentPerson  `json:"backup"`
	CurrentPeople    []DutyCurrentPerson `json:"currentPeople"`
	TodayAssignment  DutyAssignment      `json:"todayAssignment"`
	NextAssignment   DutyAssignment      `json:"nextAssignment"`
	Handovers        []DutyHandover      `json:"handovers"`
	PendingHandovers []DutyHandover      `json:"pendingHandovers"`
	PendingCount     int                 `json:"pendingCount"`
}

type DashboardTrendPoint struct {
	Time         string  `json:"time"`
	Label        string  `json:"label"`
	Availability float64 `json:"availability"`
}

type DashboardTrend struct {
	Source              string                `json:"source"`
	Status              string                `json:"status"`
	System              string                `json:"system"`
	Period              string                `json:"period"`
	From                string                `json:"from"`
	To                  string                `json:"to"`
	Timezone            string                `json:"timezone"`
	Granularity         string                `json:"granularity"`
	Metric              string                `json:"metric"`
	Unit                string                `json:"unit"`
	Definition          string                `json:"definition"`
	IncludesAssetHealth bool                  `json:"includesAssetHealth"`
	Target              float64               `json:"target"`
	Points              []DashboardTrendPoint `json:"points"`
	Notice              string                `json:"notice"`
}

type DashboardWorkspace struct {
	Source          string         `json:"source"`
	GeneratedAt     time.Time      `json:"generatedAt"`
	Statistics      Dashboard      `json:"statistics"`
	BusinessSystems []string       `json:"businessSystems"`
	Incidents       []Incident     `json:"incidents"`
	Tasks           []Task         `json:"tasks"`
	Duty            DashboardDuty  `json:"duty"`
	Trend           DashboardTrend `json:"trend"`
}

type DashboardPreviewSystem struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	AssetCount   int    `json:"assetCount"`
	ServiceCount int    `json:"serviceCount"`
}

type DashboardPreviewAlerts struct {
	Source      string         `json:"source"`
	Status      string         `json:"status"`
	UrgentCount int            `json:"urgentCount"`
	LevelCounts map[string]int `json:"levelCounts"`
}

type DashboardPreviewRisk struct {
	ID       string `json:"id"`
	Level    string `json:"level"`
	Title    string `json:"title"`
	Asset    string `json:"asset"`
	Time     string `json:"time"`
	Owner    string `json:"owner"`
	Status   string `json:"status"`
	Analysis string `json:"analysis"`
	Evidence string `json:"evidence"`
	Impact   string `json:"impact"`
	Plan     string `json:"plan"`
}

type DashboardPreviewService struct {
	Name         string  `json:"name"`
	Status       string  `json:"status"`
	Availability float64 `json:"availability"`
	P95Ms        int     `json:"p95Ms"`
}

type DashboardPreviewAutomation struct {
	Source      string  `json:"source"`
	Status      string  `json:"status"`
	Succeeded   int     `json:"succeeded"`
	Failed      int     `json:"failed"`
	SuccessRate float64 `json:"successRate"`
}

type DashboardPreviewInsight struct {
	RiskID   string `json:"riskId"`
	Title    string `json:"title"`
	Analysis string `json:"analysis"`
	Evidence string `json:"evidence"`
	Plan     string `json:"plan"`
	Status   string `json:"status"`
}

type DashboardPreviewNotification struct {
	RiskID string `json:"riskId"`
	Title  string `json:"title"`
	Time   string `json:"time"`
}

type DashboardPreview struct {
	Source           string                         `json:"source"`
	GeneratedAt      time.Time                      `json:"generatedAt"`
	ExecutionAllowed bool                           `json:"executionAllowed"`
	Notice           string                         `json:"notice"`
	BusinessSystems  []DashboardPreviewSystem       `json:"businessSystems"`
	Alerts           DashboardPreviewAlerts         `json:"alerts"`
	Risks            []DashboardPreviewRisk         `json:"risks"`
	Services         []DashboardPreviewService      `json:"services"`
	Automation       DashboardPreviewAutomation     `json:"automation"`
	Insights         []DashboardPreviewInsight      `json:"insights"`
	Notifications    []DashboardPreviewNotification `json:"notifications"`
	Trend            DashboardTrend                 `json:"trend"`
}
