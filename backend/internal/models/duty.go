package models

import "time"

type DutyTeam struct {
	Name string `json:"name"`
}

type DutyMember struct {
	ID       string `json:"id"`
	UserID   int64  `json:"userId"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Team     string `json:"team"`
	Role     string `json:"role"`
	Count    int    `json:"count"`
	Next     string `json:"next"`
	Status   string `json:"status"`
}

type DutyScheduleTemplate struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Team     string   `json:"team"`
	Rotation string   `json:"rotation"`
	Time     string   `json:"time"`
	Members  []string `json:"members"`
	Active   bool     `json:"active"`
}

type DutyAssignment struct {
	Primary string `json:"primary"`
	Backup  string `json:"backup"`
}

type DutyCurrentPerson struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Role   string `json:"role"`
	Team   string `json:"team"`
	Since  string `json:"since"`
	Until  string `json:"until"`
	Phone  string `json:"phone"`
	Status string `json:"status"`
}

type DutyHandover struct {
	ID       string `json:"id"`
	From     string `json:"from"`
	To       string `json:"to"`
	Time     string `json:"time"`
	Content  string `json:"content"`
	Complete bool   `json:"complete"`
}

type DutyEscalationLevel struct {
	Level   int    `json:"level"`
	Target  string `json:"target"`
	Delay   string `json:"delay"`
	Channel string `json:"channel"`
}

type DutyEscalationPolicy struct {
	Name     string                `json:"name"`
	Team     string                `json:"team"`
	Severity string                `json:"severity"`
	Levels   []DutyEscalationLevel `json:"levels"`
}

type DutyCenterData struct {
	Teams         []DutyTeam                `json:"teams"`
	Members       []DutyMember              `json:"members"`
	Schedules     []DutyScheduleTemplate    `json:"schedules"`
	Assignments   map[string]DutyAssignment `json:"assignments"`
	CurrentPeople []DutyCurrentPerson       `json:"currentPeople"`
	Handovers     []DutyHandover            `json:"handovers"`
	Escalation    DutyEscalationPolicy      `json:"escalation"`
}

type DutyCenterState struct {
	Revision  int64          `json:"revision"`
	Data      DutyCenterData `json:"data"`
	UpdatedAt time.Time      `json:"updatedAt,omitempty"`
	UpdatedBy int64          `json:"updatedBy,omitempty"`
}

type DutyCenterMutation struct {
	Revision int64          `json:"revision"`
	Data     DutyCenterData `json:"data"`
}
