package domain

import "testing"

func TestTaskStatusFlow(t *testing.T) {
	if !CanTransitionTask(TaskPending, TaskInProgress) {
		t.Fatalf("pending task should move to in-progress")
	}
	if !CanTransitionTask(TaskPendingConfirm, TaskDone) {
		t.Fatalf("pending-confirm task should move to done")
	}
	if CanTransitionTask(TaskClosed, TaskInProgress) {
		t.Fatalf("closed task should not reopen in phase one")
	}
}

func TestIncidentStatusFlow(t *testing.T) {
	if !CanTransitionIncident(IncidentNew, IncidentProcessing) {
		t.Fatalf("new incident should move to processing")
	}
	if !CanTransitionIncident(IncidentProcessing, IncidentRecovered) {
		t.Fatalf("processing incident should move to recovered")
	}
	if CanTransitionIncident(IncidentClosed, IncidentProcessing) {
		t.Fatalf("closed incident should not reopen in phase one")
	}
}
