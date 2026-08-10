package models

import "time"

type AuditEvent struct {
	ID            int64          `json:"id"`
	ActorUserID   int64          `json:"actorUserId,omitempty"`
	ActorUsername string         `json:"actorUsername"`
	Action        string         `json:"action"`
	ResourceType  string         `json:"resourceType"`
	ResourceID    string         `json:"resourceId,omitempty"`
	Outcome       string         `json:"outcome"`
	Detail        map[string]any `json:"detail,omitempty"`
	IPAddress     string         `json:"ipAddress"`
	UserAgent     string         `json:"userAgent"`
	CreatedAt     time.Time      `json:"createdAt"`
}

type ListQuery struct {
	Page        int
	PageSize    int
	Keyword     string
	Type        string
	Kind        string
	Environment string
	Business    string
	NetworkZone string
	Status      string
	Level       string
	Sort        string
	Order       string
}

type PageResult[T any] struct {
	Items     []T                 `json:"items"`
	Total     int64               `json:"total"`
	Page      int                 `json:"page"`
	PageSize  int                 `json:"pageSize"`
	PageCount int                 `json:"pageCount"`
	Counts    map[string]int64    `json:"counts,omitempty"`
	Options   map[string][]string `json:"options,omitempty"`
}

type User struct {
	ID                 int64     `json:"id"`
	Username           string    `json:"username"`
	DisplayName        string    `json:"displayName"`
	MustChangePassword bool      `json:"mustChangePassword"`
	Roles              []string  `json:"roles"`
	CreatedAt          time.Time `json:"createdAt"`
}

type UserListItem struct {
	ID                 int64     `json:"id"`
	Username           string    `json:"username"`
	DisplayName        string    `json:"displayName"`
	MustChangePassword bool      `json:"mustChangePassword"`
	Roles              []string  `json:"roles"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type UserDirectoryItem struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
}

type UserMutation struct {
	ID                 int64    `json:"id"`
	Username           string   `json:"username"`
	DisplayName        string   `json:"displayName"`
	Password           string   `json:"password,omitempty"`
	MustChangePassword bool     `json:"mustChangePassword"`
	Roles              []string `json:"roles"`
}

type Asset struct {
	ID             int64     `json:"id"`
	CreatedBy      int64     `json:"createdBy"`
	AssetNo        string    `json:"assetNo"`
	Type           string    `json:"type"`
	Vendor         string    `json:"vendor"`
	CPUArch        string    `json:"cpuArch"`
	SN             string    `json:"sn"`
	Location       string    `json:"location"`
	Business       string    `json:"business"`
	IPv4           string    `json:"ipv4"`
	IPv6           string    `json:"ipv6"`
	Environment    string    `json:"environment"`
	OS             string    `json:"os"`
	Hostname       string    `json:"hostname"`
	NetworkZone    string    `json:"networkZone"`
	CPU            string    `json:"cpu"`
	Memory         string    `json:"memory"`
	Disk           string    `json:"disk"`
	DeploymentInfo string    `json:"deploymentInfo"`
	Owner          string    `json:"owner"`
	Status         string    `json:"status"`
	HostMachine    string    `json:"hostMachine"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type AssetCredential struct {
	AssetID   int64  `json:"assetId"`
	LoginURL  string `json:"loginUrl"`
	Username  string `json:"username"`
	Secret    string `json:"secret,omitempty"`
	HasSecret bool   `json:"hasSecret"`
	Notes     string `json:"notes"`
}

type MiddlewareCredential struct {
	MiddlewareID int64  `json:"middlewareId"`
	LoginURL     string `json:"loginUrl"`
	Username     string `json:"username"`
	Secret       string `json:"secret,omitempty"`
	HasSecret    bool   `json:"hasSecret"`
	Notes        string `json:"notes"`
}

type CopilotConfig struct {
	Provider              string `json:"provider"`
	Endpoint              string `json:"endpoint"`
	Model                 string `json:"model"`
	APIKey                string `json:"apiKey,omitempty"`
	HasAPIKey             bool   `json:"hasApiKey"`
	LocalEndpoint         string `json:"localEndpoint"`
	LocalModel            string `json:"localModel"`
	Temperature           string `json:"temperature"`
	MaxTokens             string `json:"maxTokens"`
	EnableAssetContext    bool   `json:"enableAssetContext"`
	EnableIncidentContext bool   `json:"enableIncidentContext"`
	EnableTaskContext     bool   `json:"enableTaskContext"`
	EnableOncallContext   bool   `json:"enableOncallContext"`
	AuditEnabled          bool   `json:"auditEnabled"`
}

type CopilotModelConfig struct {
	ID                    int64     `json:"id"`
	Name                  string    `json:"name"`
	Provider              string    `json:"provider"`
	Endpoint              string    `json:"endpoint"`
	Model                 string    `json:"model"`
	APIKey                string    `json:"apiKey,omitempty"`
	HasAPIKey             bool      `json:"hasApiKey"`
	LocalEndpoint         string    `json:"localEndpoint"`
	LocalModel            string    `json:"localModel"`
	Temperature           string    `json:"temperature"`
	MaxTokens             string    `json:"maxTokens"`
	EnableAssetContext    bool      `json:"enableAssetContext"`
	EnableIncidentContext bool      `json:"enableIncidentContext"`
	EnableTaskContext     bool      `json:"enableTaskContext"`
	EnableOncallContext   bool      `json:"enableOncallContext"`
	AuditEnabled          bool      `json:"auditEnabled"`
	IsActive              bool      `json:"isActive"`
	CreatedAt             time.Time `json:"createdAt"`
	UpdatedAt             time.Time `json:"updatedAt"`
}

type MiddlewareInstance struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Kind        string    `json:"kind"`
	Version     string    `json:"version"`
	Environment string    `json:"environment"`
	NetworkZone string    `json:"networkZone"`
	Endpoint    string    `json:"endpoint"`
	Business    string    `json:"business"`
	Owner       string    `json:"owner"`
	Status      string    `json:"status"`
	AssetID     *int64    `json:"assetId,omitempty"`
	AssetNo     string    `json:"assetNo,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type OnCallSchedule struct {
	ID            int64     `json:"id"`
	RuleType      string    `json:"ruleType"`
	Date          string    `json:"date"`
	Week          string    `json:"week"`
	Primary       string    `json:"primary"`
	PrimaryUserID *int64    `json:"primaryUserId,omitempty"`
	Backup        string    `json:"backup"`
	BackupUserID  *int64    `json:"backupUserId,omitempty"`
	SwapFrom      string    `json:"swapFrom"`
	SwapTo        string    `json:"swapTo"`
	Notes         string    `json:"notes"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type Task struct {
	ID             int64     `json:"id"`
	Title          string    `json:"title"`
	Type           string    `json:"type"`
	Assignee       string    `json:"assignee"`
	AssigneeUserID *int64    `json:"assigneeUserId,omitempty"`
	Status         string    `json:"status"`
	DueAt          string    `json:"dueAt"`
	Description    string    `json:"description"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type Incident struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Level       string    `json:"level"`
	Status      string    `json:"status"`
	Owner       string    `json:"owner"`
	OwnerUserID *int64    `json:"ownerUserId,omitempty"`
	Business    string    `json:"business"`
	StartedAt   string    `json:"startedAt"`
	RecoveredAt string    `json:"recoveredAt"`
	Summary     string    `json:"summary"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Dashboard struct {
	AssetCount           int64            `json:"assetCount"`
	TodayOnCallCount     int64            `json:"todayOnCallCount"`
	ActiveTaskCount      int64            `json:"activeTaskCount"`
	ActiveIncidentCount  int64            `json:"activeIncidentCount"`
	AssetTypeCounts      map[string]int64 `json:"assetTypeCounts"`
	IncidentLevelCounts  map[string]int64 `json:"incidentLevelCounts"`
	TaskStatusCounts     map[string]int64 `json:"taskStatusCounts"`
	AssetHealthyCount    int64            `json:"assetHealthyCount"`
	AssetAbnormalCount   int64            `json:"assetAbnormalCount"`
	TaskClosedCount      int64            `json:"taskClosedCount"`
	TaskOpenCount        int64            `json:"taskOpenCount"`
	IncidentClosedCount  int64            `json:"incidentClosedCount"`
	ResponseMinutesTotal int64            `json:"responseMinutesTotal"`
	ResponseSampleCount  int64            `json:"responseSampleCount"`
}
