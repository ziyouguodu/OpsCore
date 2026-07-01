package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"opscore/backend/internal/models"
)

var ErrDutyRevisionConflict = errors.New("duty center data was changed by another user; reload and retry")

func (s *Store) GetDutyCenter(ctx context.Context) (models.DutyCenterState, error) {
	var state models.DutyCenterState
	var raw []byte
	err := s.pool.QueryRow(ctx, `select revision, state, updated_at, coalesce(updated_by, 0) from duty_center_state where singleton=true`).Scan(
		&state.Revision, &raw, &state.UpdatedAt, &state.UpdatedBy,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		state.Data.Assignments = map[string]models.DutyAssignment{}
		return state, nil
	}
	if err != nil {
		return models.DutyCenterState{}, err
	}
	if err := json.Unmarshal(raw, &state.Data); err != nil {
		return models.DutyCenterState{}, fmt.Errorf("decode duty center state: %w", err)
	}
	if state.Data.Assignments == nil {
		state.Data.Assignments = map[string]models.DutyAssignment{}
	}
	return state, nil
}

func (s *Store) SaveDutyCenter(ctx context.Context, mutation models.DutyCenterMutation, actorUserID int64) (models.DutyCenterState, error) {
	if err := ValidateDutyCenterData(mutation.Data); err != nil {
		return models.DutyCenterState{}, err
	}
	raw, err := json.Marshal(mutation.Data)
	if err != nil {
		return models.DutyCenterState{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.DutyCenterState{}, err
	}
	defer tx.Rollback(ctx)

	var saved models.DutyCenterState
	var savedRaw []byte
	err = tx.QueryRow(ctx, `
		update duty_center_state set
			revision=revision + 1,
			state=$1,
			updated_by=$2,
			updated_at=now()
		where singleton=true and revision=$3
		returning revision, state, updated_at, updated_by
	`, raw, actorUserID, mutation.Revision).Scan(&saved.Revision, &savedRaw, &saved.UpdatedAt, &saved.UpdatedBy)
	if errors.Is(err, pgx.ErrNoRows) && mutation.Revision == 0 {
		err = tx.QueryRow(ctx, `
			insert into duty_center_state(singleton, revision, state, updated_by)
			values (true, 1, $1, $2)
			on conflict (singleton) do nothing
			returning revision, state, updated_at, updated_by
		`, raw, actorUserID).Scan(&saved.Revision, &savedRaw, &saved.UpdatedAt, &saved.UpdatedBy)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return models.DutyCenterState{}, ErrDutyRevisionConflict
	}
	if err != nil {
		return models.DutyCenterState{}, err
	}
	if err := json.Unmarshal(savedRaw, &saved.Data); err != nil {
		return models.DutyCenterState{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return models.DutyCenterState{}, err
	}
	return saved, nil
}

func ValidateDutyCenterData(data models.DutyCenterData) error {
	teams := make(map[string]struct{}, len(data.Teams))
	for _, team := range data.Teams {
		name := strings.TrimSpace(team.Name)
		if name == "" {
			return errors.New("duty team name is required")
		}
		if _, exists := teams[name]; exists {
			return fmt.Errorf("duplicate duty team %q", name)
		}
		teams[name] = struct{}{}
	}

	members := make(map[string]struct{}, len(data.Members))
	userIDs := make(map[int64]struct{}, len(data.Members))
	for _, member := range data.Members {
		if member.UserID <= 0 || strings.TrimSpace(member.Name) == "" {
			return errors.New("duty member must reference a system user")
		}
		if _, exists := userIDs[member.UserID]; exists {
			return fmt.Errorf("system user %d already belongs to the duty roster", member.UserID)
		}
		if _, exists := teams[member.Team]; !exists {
			return fmt.Errorf("duty member team %q does not exist", member.Team)
		}
		if _, exists := members[member.Name]; exists {
			return fmt.Errorf("duplicate duty member name %q", member.Name)
		}
		userIDs[member.UserID] = struct{}{}
		members[member.Name] = struct{}{}
	}

	for _, schedule := range data.Schedules {
		if strings.TrimSpace(schedule.Name) == "" {
			return errors.New("duty schedule name is required")
		}
		if _, exists := teams[schedule.Team]; !exists {
			return fmt.Errorf("duty schedule team %q does not exist", schedule.Team)
		}
		if schedule.Rotation != "daily" && schedule.Rotation != "weekly" {
			return errors.New("duty schedule rotation must be daily or weekly")
		}
		for _, name := range schedule.Members {
			if _, exists := members[name]; !exists {
				return fmt.Errorf("duty schedule member %q is not in the roster", name)
			}
		}
	}

	for date, assignment := range data.Assignments {
		if _, err := time.Parse("2006-01-02", date); err != nil {
			return fmt.Errorf("invalid duty assignment date %q", date)
		}
		for _, name := range []string{assignment.Primary, assignment.Backup} {
			if name != "" {
				if _, exists := members[name]; !exists {
					return fmt.Errorf("duty assignment member %q is not in the roster", name)
				}
			}
		}
	}

	for _, person := range data.CurrentPeople {
		if _, exists := members[person.Name]; !exists {
			return fmt.Errorf("current duty person %q is not in the roster", person.Name)
		}
		if _, exists := teams[person.Team]; !exists {
			return fmt.Errorf("current duty person team %q does not exist", person.Team)
		}
	}

	for _, handover := range data.Handovers {
		if _, exists := members[handover.From]; !exists {
			return fmt.Errorf("handover source %q is not in the roster", handover.From)
		}
		if _, exists := members[handover.To]; !exists {
			return fmt.Errorf("handover target %q is not in the roster", handover.To)
		}
		if strings.TrimSpace(handover.Content) == "" {
			return errors.New("duty handover content is required")
		}
	}

	if data.Escalation.Team != "" {
		if _, exists := teams[data.Escalation.Team]; !exists {
			return fmt.Errorf("duty escalation team %q does not exist", data.Escalation.Team)
		}
	}
	if data.Escalation.Severity != "" && data.Escalation.Severity != "P1" && data.Escalation.Severity != "P2" && data.Escalation.Severity != "P3" && data.Escalation.Severity != "P4" {
		return errors.New("duty escalation severity must be P1, P2, P3 or P4")
	}
	if data.Escalation.Name != "" || data.Escalation.Team != "" || len(data.Escalation.Levels) > 0 {
		if strings.TrimSpace(data.Escalation.Name) == "" || data.Escalation.Team == "" || data.Escalation.Severity == "" || len(data.Escalation.Levels) == 0 {
			return errors.New("duty escalation policy is incomplete")
		}
		for index, level := range data.Escalation.Levels {
			if level.Level != index+1 || strings.TrimSpace(level.Target) == "" || strings.TrimSpace(level.Delay) == "" || strings.TrimSpace(level.Channel) == "" {
				return fmt.Errorf("duty escalation level %d is incomplete", index+1)
			}
		}
	}
	return nil
}
