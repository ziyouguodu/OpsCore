package api

import "testing"

func TestValidateTaskTransitionRejectsInvalidMove(t *testing.T) {
	if err := validateTaskTransition("已关闭", "处理中"); err == nil {
		t.Fatalf("expected closed task to reject transition back to processing")
	}
	if err := validateTaskTransition("待处理", "处理中"); err != nil {
		t.Fatalf("expected pending task to move to processing: %v", err)
	}
}

func TestValidateIncidentTransitionRejectsInvalidMove(t *testing.T) {
	if err := validateIncidentTransition("已关闭", "处理中"); err == nil {
		t.Fatalf("expected closed incident to reject transition back to processing")
	}
	if err := validateIncidentTransition("处理中", "已恢复"); err != nil {
		t.Fatalf("expected processing incident to move to recovered: %v", err)
	}
}
