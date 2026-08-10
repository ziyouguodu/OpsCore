package api

import (
	"context"

	"opscore/backend/internal/models"
)

type healthPersistence interface {
	Ping(context.Context) error
}

type authPersistence interface {
	Authenticate(context.Context, string, string) (models.User, bool, error)
	GetUser(context.Context, int64) (models.User, error)
	ChangePassword(context.Context, int64, string, string) (models.User, error)
}

type dashboardPersistence interface {
	Dashboard(context.Context) (models.Dashboard, error)
}

type userPersistence interface {
	ListUsers(context.Context) ([]models.UserListItem, error)
	ListUserDirectory(context.Context) ([]models.UserDirectoryItem, error)
	CreateUser(context.Context, models.UserMutation) (models.UserListItem, error)
	UpdateUser(context.Context, int64, models.UserMutation) (models.UserListItem, error)
	DeleteUser(context.Context, int64) error
}

type assetPersistence interface {
	ListAssets(context.Context) ([]models.Asset, error)
	ListAssetsPage(context.Context, models.ListQuery) (models.PageResult[models.Asset], error)
	UpsertAsset(context.Context, models.Asset) (models.Asset, error)
	DeleteAsset(context.Context, int64, int64, bool) error
	GetAssetCredential(context.Context, int64) (models.AssetCredential, error)
	UpsertAssetCredential(context.Context, models.AssetCredential) (models.AssetCredential, error)
}

type middlewarePersistence interface {
	GetMiddlewareCredential(context.Context, int64) (models.MiddlewareCredential, error)
	UpsertMiddlewareCredential(context.Context, models.MiddlewareCredential) (models.MiddlewareCredential, error)
	ListMiddleware(context.Context) ([]models.MiddlewareInstance, error)
	ListMiddlewarePage(context.Context, models.ListQuery) (models.PageResult[models.MiddlewareInstance], error)
	CreateMiddleware(context.Context, models.MiddlewareInstance) (models.MiddlewareInstance, error)
	UpdateMiddleware(context.Context, int64, models.MiddlewareInstance) (models.MiddlewareInstance, error)
	DeleteMiddleware(context.Context, int64) error
}

type credentialSecurityPersistence interface {
	HasCredentialVerificationPassword(context.Context) (bool, error)
	SetCredentialVerificationPassword(context.Context, string) error
	VerifyCredentialPassword(context.Context, string) (bool, error)
}

type copilotPersistence interface {
	GetCopilotConfig(context.Context) (models.CopilotConfig, error)
	UpsertCopilotConfig(context.Context, models.CopilotConfig) (models.CopilotConfig, error)
	GetCopilotAPIKey(context.Context) (string, error)
	ListCopilotModelConfigs(context.Context) ([]models.CopilotModelConfig, error)
	GetCopilotModelConfig(context.Context, int64) (models.CopilotModelConfig, error)
	GetActiveCopilotModelConfig(context.Context) (models.CopilotModelConfig, error)
	CreateCopilotModelConfig(context.Context, models.CopilotModelConfig) (models.CopilotModelConfig, error)
	UpdateCopilotModelConfig(context.Context, int64, models.CopilotModelConfig) (models.CopilotModelConfig, error)
	DeleteCopilotModelConfig(context.Context, int64) error
	ActivateCopilotModelConfig(context.Context, int64) (models.CopilotModelConfig, error)
	GetCopilotModelAPIKey(context.Context, int64) (string, error)
}

type oncallPersistence interface {
	ListOnCalls(context.Context) ([]models.OnCallSchedule, error)
	CreateOnCall(context.Context, models.OnCallSchedule) (models.OnCallSchedule, error)
	UpdateOnCall(context.Context, int64, models.OnCallSchedule) (models.OnCallSchedule, error)
	DeleteOnCall(context.Context, int64) error
	GetDutyCenter(context.Context) (models.DutyCenterState, error)
	SaveDutyCenter(context.Context, models.DutyCenterMutation, int64) (models.DutyCenterState, error)
}

type taskPersistence interface {
	ListTasks(context.Context) ([]models.Task, error)
	ListTasksPage(context.Context, models.ListQuery) (models.PageResult[models.Task], error)
	CreateTask(context.Context, models.Task) (models.Task, error)
	UpdateTask(context.Context, int64, models.Task) (models.Task, error)
	DeleteTask(context.Context, int64) error
	GetTaskStatus(context.Context, int64) (string, error)
	UpdateTaskStatus(context.Context, int64, string) error
}

type incidentPersistence interface {
	ListIncidents(context.Context) ([]models.Incident, error)
	ListIncidentsPage(context.Context, models.ListQuery) (models.PageResult[models.Incident], error)
	CreateIncident(context.Context, models.Incident) (models.Incident, error)
	UpdateIncident(context.Context, int64, models.Incident) (models.Incident, error)
	DeleteIncident(context.Context, int64) error
	GetIncidentStatus(context.Context, int64) (string, error)
	UpdateIncidentStatus(context.Context, int64, string) error
}

type auditPersistence interface {
	RecordAudit(context.Context, models.AuditEvent) error
	ListAuditEvents(context.Context, int) ([]models.AuditEvent, error)
}

type persistence interface {
	healthPersistence
	authPersistence
	dashboardPersistence
	userPersistence
	assetPersistence
	middlewarePersistence
	credentialSecurityPersistence
	copilotPersistence
	oncallPersistence
	taskPersistence
	incidentPersistence
	auditPersistence
}
