package store

import (
	"testing"

	"opscore/backend/internal/models"
)

func TestValidateDutyCenterRejectsUnknownTeams(t *testing.T) {
	data := models.DutyCenterData{
		Teams:   []models.DutyTeam{{Name: "基础运维组"}},
		Members: []models.DutyMember{{UserID: 1, Name: "李明", Team: "不存在的团队"}},
	}
	if err := ValidateDutyCenterData(data); err == nil {
		t.Fatal("expected unknown member team to be rejected")
	}
}

func TestValidateDutyCenterRejectsDuplicateSystemUsers(t *testing.T) {
	data := models.DutyCenterData{
		Teams: []models.DutyTeam{{Name: "基础运维组"}},
		Members: []models.DutyMember{
			{UserID: 1, Name: "李明", Team: "基础运维组"},
			{UserID: 1, Name: "李明副本", Team: "基础运维组"},
		},
	}
	if err := ValidateDutyCenterData(data); err == nil {
		t.Fatal("expected duplicate system user to be rejected")
	}
}

func TestValidateDutyCenterRejectsUnknownCurrentPerson(t *testing.T) {
	data := models.DutyCenterData{
		Teams:         []models.DutyTeam{{Name: "基础运维组"}},
		Members:       []models.DutyMember{{UserID: 1, Name: "李明", Team: "基础运维组"}},
		CurrentPeople: []models.DutyCurrentPerson{{Name: "未纳管人员", Team: "基础运维组"}},
	}
	if err := ValidateDutyCenterData(data); err == nil {
		t.Fatal("expected current duty person outside roster to be rejected")
	}
}

func TestValidateDutyCenterRejectsIncompleteEscalationLevels(t *testing.T) {
	data := models.DutyCenterData{
		Teams: []models.DutyTeam{{Name: "基础运维组"}},
		Escalation: models.DutyEscalationPolicy{
			Name: "P1 策略", Team: "基础运维组", Severity: "P1",
			Levels: []models.DutyEscalationLevel{{Level: 1}},
		},
	}
	if err := ValidateDutyCenterData(data); err == nil {
		t.Fatal("expected incomplete escalation level to be rejected")
	}
}

func TestValidateDutyCenterAcceptsRelationalReferences(t *testing.T) {
	data := models.DutyCenterData{
		Teams: []models.DutyTeam{{Name: "基础运维组"}},
		Members: []models.DutyMember{
			{ID: "member-1", UserID: 1, Username: "ops.li", Name: "李明", Team: "基础运维组"},
			{ID: "member-2", UserID: 2, Username: "ops.wang", Name: "王敏", Team: "基础运维组"},
		},
		Schedules:     []models.DutyScheduleTemplate{{ID: "schedule-1", Name: "日常排班", Team: "基础运维组", Rotation: "daily", Members: []string{"李明", "王敏"}}},
		Assignments:   map[string]models.DutyAssignment{"2026-07-10": {Primary: "李明", Backup: "王敏"}},
		CurrentPeople: []models.DutyCurrentPerson{{ID: "current-1", Name: "李明", Team: "基础运维组"}},
		Handovers:     []models.DutyHandover{{ID: "handover-1", From: "李明", To: "王敏", Content: "无未结事项"}},
		Escalation:    models.DutyEscalationPolicy{Name: "生产 P1", Team: "基础运维组", Severity: "P1", Levels: []models.DutyEscalationLevel{{Level: 1, Target: "李明", Delay: "5 分钟", Channel: "电话"}}},
	}
	if err := ValidateDutyCenterData(data); err != nil {
		t.Fatalf("expected relational duty data to be valid: %v", err)
	}
}

func TestValidateDutyCenterRejectsMissingStableIDs(t *testing.T) {
	data := models.DutyCenterData{
		Teams:   []models.DutyTeam{{Name: "基础运维组"}},
		Members: []models.DutyMember{{UserID: 1, Username: "ops.li", Name: "李明", Team: "基础运维组"}},
	}
	if err := ValidateDutyCenterData(data); err == nil {
		t.Fatal("expected missing stable member id to be rejected")
	}
}
