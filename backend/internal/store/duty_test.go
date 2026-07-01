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
