package domain

type TaskStatus string

const (
	TaskPending        TaskStatus = "待处理"
	TaskInProgress     TaskStatus = "处理中"
	TaskPendingConfirm TaskStatus = "待确认"
	TaskDone           TaskStatus = "已完成"
	TaskClosed         TaskStatus = "已关闭"
)

type IncidentStatus string

const (
	IncidentNew        IncidentStatus = "新建"
	IncidentProcessing IncidentStatus = "处理中"
	IncidentRecovered  IncidentStatus = "已恢复"
	IncidentClosed     IncidentStatus = "已关闭"
)

func CanTransitionTask(from, to TaskStatus) bool {
	allowed := map[TaskStatus][]TaskStatus{
		TaskPending:        {TaskInProgress, TaskClosed},
		TaskInProgress:     {TaskPendingConfirm, TaskClosed},
		TaskPendingConfirm: {TaskDone, TaskInProgress, TaskClosed},
		TaskDone:           {TaskClosed},
		TaskClosed:         {},
	}
	return containsTaskStatus(allowed[from], to)
}

func CanTransitionIncident(from, to IncidentStatus) bool {
	allowed := map[IncidentStatus][]IncidentStatus{
		IncidentNew:        {IncidentProcessing, IncidentClosed},
		IncidentProcessing: {IncidentRecovered, IncidentClosed},
		IncidentRecovered:  {IncidentClosed, IncidentProcessing},
		IncidentClosed:     {},
	}
	return containsIncidentStatus(allowed[from], to)
}

func containsTaskStatus(values []TaskStatus, value TaskStatus) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func containsIncidentStatus(values []IncidentStatus, value IncidentStatus) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}
