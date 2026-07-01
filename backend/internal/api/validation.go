package api

import (
	"errors"
	"fmt"
	"strings"

	"opscore/backend/internal/domain"
	"opscore/backend/internal/models"
)

func validateUserMutation(item models.UserMutation, requirePassword bool) error {
	if strings.TrimSpace(item.Username) == "" || strings.TrimSpace(item.DisplayName) == "" || len(item.Roles) == 0 {
		return errors.New("username, displayName and roles are required")
	}
	if requirePassword && item.Password == "" {
		return errors.New("password is required")
	}
	if item.Password != "" && len(item.Password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	return nil
}

func validateAsset(item models.Asset, requireAssetNo bool) error {
	required := map[string]string{
		"type":           item.Type,
		"cpuArch":        item.CPUArch,
		"business":       item.Business,
		"ipv4":           item.IPv4,
		"environment":    item.Environment,
		"os":             item.OS,
		"networkZone":    item.NetworkZone,
		"cpu":            item.CPU,
		"memory":         item.Memory,
		"disk":           item.Disk,
		"deploymentInfo": item.DeploymentInfo,
		"owner":          item.Owner,
		"status":         item.Status,
	}
	if requireAssetNo {
		required["assetNo"] = item.AssetNo
	}
	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("asset %s is required", field)
		}
	}
	if !containsString([]string{"物理机", "虚拟机"}, item.Type) {
		return errors.New("asset type must be 物理机 or 虚拟机")
	}
	if !containsString([]string{"生产", "仿真", "研发"}, item.Environment) {
		return errors.New("asset environment must be 生产, 仿真 or 研发")
	}
	if !containsString([]string{"运行中", "维护中", "停用", "故障"}, item.Status) {
		return errors.New("asset status is invalid")
	}
	return nil
}

func prepareMiddleware(item *models.MiddlewareInstance) error {
	required := map[string]string{
		"name":        item.Name,
		"kind":        item.Kind,
		"environment": item.Environment,
		"networkZone": item.NetworkZone,
		"endpoint":    item.Endpoint,
		"business":    item.Business,
		"owner":       item.Owner,
	}
	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("middleware %s is required", field)
		}
	}
	if item.Status == "" {
		item.Status = "运行中"
	}
	if !containsString([]string{"MySQL", "Redis", "Kafka", "PostgreSQL", "达梦", "Nginx", "ElasticSearch", "Nacos", "RocketMQ", "MinIO"}, item.Kind) {
		return errors.New("middleware kind is invalid")
	}
	if !containsString([]string{"生产", "仿真", "研发"}, item.Environment) {
		return errors.New("middleware environment must be 生产, 仿真 or 研发")
	}
	if !containsString([]string{"运行中", "维护中", "停用", "故障"}, item.Status) {
		return errors.New("middleware status is invalid")
	}
	return nil
}

func prepareOnCall(item *models.OnCallSchedule) error {
	if item.RuleType == "" {
		item.RuleType = "daily"
	}
	if item.RuleType != "daily" && item.RuleType != "weekly" {
		return errors.New("oncall ruleType must be daily or weekly")
	}
	if strings.TrimSpace(item.Primary) == "" {
		return errors.New("oncall primary is required")
	}
	if item.RuleType == "daily" && strings.TrimSpace(item.Date) == "" {
		return errors.New("oncall date is required for daily rule")
	}
	if item.RuleType == "weekly" && strings.TrimSpace(item.Week) == "" {
		return errors.New("oncall week is required for weekly rule")
	}
	return nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func validateTaskMutation(item models.Task) error {
	if strings.TrimSpace(item.Title) == "" {
		return errors.New("task title is required")
	}
	return nil
}

func validateTaskStatus(status string) error {
	switch domain.TaskStatus(status) {
	case domain.TaskPending, domain.TaskInProgress, domain.TaskPendingConfirm, domain.TaskDone, domain.TaskClosed:
		return nil
	default:
		return errors.New("invalid task status")
	}
}

func validateTaskTransition(from, to string) error {
	if from == to {
		return nil
	}
	if err := validateTaskStatus(from); err != nil {
		return err
	}
	if err := validateTaskStatus(to); err != nil {
		return err
	}
	if !domain.CanTransitionTask(domain.TaskStatus(from), domain.TaskStatus(to)) {
		return errors.New("invalid task status transition")
	}
	return nil
}

func validateIncidentMutation(item models.Incident) error {
	if strings.TrimSpace(item.Title) == "" {
		return errors.New("incident title is required")
	}
	return nil
}

func validateIncidentLevel(level string) error {
	switch level {
	case "P1", "P2", "P3", "P4":
		return nil
	default:
		return errors.New("incident level must be P1, P2, P3 or P4")
	}
}

func validateIncidentStatus(status string) error {
	switch domain.IncidentStatus(status) {
	case domain.IncidentNew, domain.IncidentProcessing, domain.IncidentRecovered, domain.IncidentClosed:
		return nil
	default:
		return errors.New("invalid incident status")
	}
}

func validateIncidentTransition(from, to string) error {
	if from == to {
		return nil
	}
	if err := validateIncidentStatus(from); err != nil {
		return err
	}
	if err := validateIncidentStatus(to); err != nil {
		return err
	}
	if !domain.CanTransitionIncident(domain.IncidentStatus(from), domain.IncidentStatus(to)) {
		return errors.New("invalid incident status transition")
	}
	return nil
}
