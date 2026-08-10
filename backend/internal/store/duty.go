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
	var relationsRevision int64
	err := s.pool.QueryRow(ctx, `select revision, relations_revision, state, updated_at, coalesce(updated_by, 0) from duty_center_state where singleton=true`).Scan(
		&state.Revision, &relationsRevision, &raw, &state.UpdatedAt, &state.UpdatedBy,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		state.Data.Assignments = map[string]models.DutyAssignment{}
		return state, nil
	}
	if err != nil {
		return models.DutyCenterState{}, err
	}
	if relationsRevision == state.Revision {
		state.Data, err = s.readDutyRelations(ctx)
		if err != nil {
			return models.DutyCenterState{}, err
		}
	} else if err := json.Unmarshal(raw, &state.Data); err != nil {
		return models.DutyCenterState{}, fmt.Errorf("decode duty center compatibility snapshot: %w", err)
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
	if err := s.validateDutyMemberUsers(ctx, mutation.Data.Members); err != nil {
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
	if err := s.syncDutyRelationsTx(ctx, tx, mutation.Data); err != nil {
		return models.DutyCenterState{}, err
	}
	if _, err := tx.Exec(ctx, `update duty_center_state set relations_revision=revision where singleton=true`); err != nil {
		return models.DutyCenterState{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return models.DutyCenterState{}, err
	}
	saved.Data = mutation.Data
	if saved.Data.Assignments == nil {
		saved.Data.Assignments = map[string]models.DutyAssignment{}
	}
	return saved, nil
}

func (s *Store) ensureDutyRelations(ctx context.Context) error {
	var revision, relationsRevision int64
	var raw []byte
	err := s.pool.QueryRow(ctx, `select revision, relations_revision, state from duty_center_state where singleton=true`).Scan(&revision, &relationsRevision, &raw)
	if errors.Is(err, pgx.ErrNoRows) || revision == relationsRevision {
		return nil
	}
	if err != nil {
		return err
	}
	var data models.DutyCenterData
	if err := json.Unmarshal(raw, &data); err != nil {
		return fmt.Errorf("decode duty center compatibility snapshot: %w", err)
	}
	if err := ValidateDutyCenterData(data); err != nil {
		return fmt.Errorf("validate duty center compatibility snapshot: %w", err)
	}
	if err := s.validateDutyMemberUsers(ctx, data.Members); err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := s.syncDutyRelationsTx(ctx, tx, data); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `update duty_center_state set relations_revision=$1 where singleton=true and revision=$1`, revision); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) syncDutyRelationsTx(ctx context.Context, tx pgx.Tx, data models.DutyCenterData) error {
	for _, table := range []string{
		"duty_escalation_levels", "duty_escalation_policies", "duty_handovers", "duty_current_people",
		"duty_assignments", "duty_schedule_members", "duty_schedule_templates", "duty_members", "duty_teams",
	} {
		if _, err := tx.Exec(ctx, `delete from `+table); err != nil {
			return err
		}
	}
	for index, team := range data.Teams {
		if _, err := tx.Exec(ctx, `insert into duty_teams(name, sort_order) values ($1,$2)`, team.Name, index); err != nil {
			return err
		}
	}
	memberIDByName := make(map[string]string, len(data.Members))
	for index, member := range data.Members {
		if _, err := tx.Exec(ctx, `insert into duty_members(id,user_id,username,name,team,role,duty_count,next_value,status,sort_order) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, member.ID, member.UserID, member.Username, member.Name, member.Team, member.Role, member.Count, member.Next, member.Status, index); err != nil {
			return err
		}
		memberIDByName[member.Name] = member.ID
	}
	for index, schedule := range data.Schedules {
		if _, err := tx.Exec(ctx, `insert into duty_schedule_templates(id,name,team,rotation,time_window,active,sort_order) values ($1,$2,$3,$4,$5,$6,$7)`, schedule.ID, schedule.Name, schedule.Team, schedule.Rotation, schedule.Time, schedule.Active, index); err != nil {
			return err
		}
		for memberIndex, name := range schedule.Members {
			if _, err := tx.Exec(ctx, `insert into duty_schedule_members(schedule_id,member_id,sort_order) values ($1,$2,$3)`, schedule.ID, memberIDByName[name], memberIndex); err != nil {
				return err
			}
		}
	}
	for date, assignment := range data.Assignments {
		var primaryID, backupID any
		if assignment.Primary != "" {
			primaryID = memberIDByName[assignment.Primary]
		}
		if assignment.Backup != "" {
			backupID = memberIDByName[assignment.Backup]
		}
		if primaryID == nil && backupID == nil {
			continue
		}
		if _, err := tx.Exec(ctx, `insert into duty_assignments(date_value,primary_member_id,backup_member_id) values ($1,$2,$3)`, date, primaryID, backupID); err != nil {
			return err
		}
	}
	for index, person := range data.CurrentPeople {
		if _, err := tx.Exec(ctx, `insert into duty_current_people(id,member_id,role,team,since_value,until_value,phone,status,sort_order) values ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, person.ID, memberIDByName[person.Name], person.Role, person.Team, person.Since, person.Until, person.Phone, person.Status, index); err != nil {
			return err
		}
	}
	for index, handover := range data.Handovers {
		if _, err := tx.Exec(ctx, `insert into duty_handovers(id,from_member_id,to_member_id,time_value,content,complete,sort_order) values ($1,$2,$3,$4,$5,$6,$7)`, handover.ID, memberIDByName[handover.From], memberIDByName[handover.To], handover.Time, handover.Content, handover.Complete, index); err != nil {
			return err
		}
	}
	if data.Escalation.Name != "" {
		if _, err := tx.Exec(ctx, `insert into duty_escalation_policies(singleton,name,team,severity) values (true,$1,$2,$3)`, data.Escalation.Name, data.Escalation.Team, data.Escalation.Severity); err != nil {
			return err
		}
		for _, level := range data.Escalation.Levels {
			if _, err := tx.Exec(ctx, `insert into duty_escalation_levels(singleton,level,target,delay_value,channel) values (true,$1,$2,$3,$4)`, level.Level, level.Target, level.Delay, level.Channel); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Store) readDutyRelations(ctx context.Context) (models.DutyCenterData, error) {
	data := models.DutyCenterData{Assignments: map[string]models.DutyAssignment{}}
	rows, err := s.pool.Query(ctx, `select name from duty_teams order by sort_order`)
	if err != nil {
		return data, err
	}
	for rows.Next() {
		var item models.DutyTeam
		if err := rows.Scan(&item.Name); err != nil {
			rows.Close()
			return data, err
		}
		data.Teams = append(data.Teams, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return data, err
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `select id,user_id,username,name,team,role,duty_count,next_value,status from duty_members order by sort_order`)
	if err != nil {
		return data, err
	}
	for rows.Next() {
		var item models.DutyMember
		if err := rows.Scan(&item.ID, &item.UserID, &item.Username, &item.Name, &item.Team, &item.Role, &item.Count, &item.Next, &item.Status); err != nil {
			rows.Close()
			return data, err
		}
		data.Members = append(data.Members, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return data, err
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `select s.id,s.name,s.team,s.rotation,s.time_window,s.active,coalesce(array_agg(m.name order by sm.sort_order) filter (where m.id is not null),'{}') from duty_schedule_templates s left join duty_schedule_members sm on sm.schedule_id=s.id left join duty_members m on m.id=sm.member_id group by s.id,s.name,s.team,s.rotation,s.time_window,s.active,s.sort_order order by s.sort_order`)
	if err != nil {
		return data, err
	}
	for rows.Next() {
		var item models.DutyScheduleTemplate
		if err := rows.Scan(&item.ID, &item.Name, &item.Team, &item.Rotation, &item.Time, &item.Active, &item.Members); err != nil {
			rows.Close()
			return data, err
		}
		data.Schedules = append(data.Schedules, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return data, err
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `select a.date_value::text,coalesce(pm.name,''),coalesce(bm.name,'') from duty_assignments a left join duty_members pm on pm.id=a.primary_member_id left join duty_members bm on bm.id=a.backup_member_id order by a.date_value`)
	if err != nil {
		return data, err
	}
	for rows.Next() {
		var date string
		var item models.DutyAssignment
		if err := rows.Scan(&date, &item.Primary, &item.Backup); err != nil {
			rows.Close()
			return data, err
		}
		data.Assignments[date] = item
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return data, err
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `select p.id,m.name,p.role,p.team,p.since_value,p.until_value,p.phone,p.status from duty_current_people p join duty_members m on m.id=p.member_id order by p.sort_order`)
	if err != nil {
		return data, err
	}
	for rows.Next() {
		var item models.DutyCurrentPerson
		if err := rows.Scan(&item.ID, &item.Name, &item.Role, &item.Team, &item.Since, &item.Until, &item.Phone, &item.Status); err != nil {
			rows.Close()
			return data, err
		}
		data.CurrentPeople = append(data.CurrentPeople, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return data, err
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `select h.id,fm.name,tm.name,h.time_value,h.content,h.complete from duty_handovers h join duty_members fm on fm.id=h.from_member_id join duty_members tm on tm.id=h.to_member_id order by h.sort_order`)
	if err != nil {
		return data, err
	}
	for rows.Next() {
		var item models.DutyHandover
		if err := rows.Scan(&item.ID, &item.From, &item.To, &item.Time, &item.Content, &item.Complete); err != nil {
			rows.Close()
			return data, err
		}
		data.Handovers = append(data.Handovers, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return data, err
	}
	rows.Close()

	err = s.pool.QueryRow(ctx, `select name,team,severity from duty_escalation_policies where singleton=true`).Scan(&data.Escalation.Name, &data.Escalation.Team, &data.Escalation.Severity)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return data, err
	}
	if err == nil {
		rows, err = s.pool.Query(ctx, `select level,target,delay_value,channel from duty_escalation_levels where singleton=true order by level`)
		if err != nil {
			return data, err
		}
		for rows.Next() {
			var item models.DutyEscalationLevel
			if err := rows.Scan(&item.Level, &item.Target, &item.Delay, &item.Channel); err != nil {
				rows.Close()
				return data, err
			}
			data.Escalation.Levels = append(data.Escalation.Levels, item)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return data, err
		}
		rows.Close()
	}
	return data, nil
}

func (s *Store) validateDutyMemberUsers(ctx context.Context, members []models.DutyMember) error {
	if len(members) == 0 {
		return nil
	}
	userIDs := make([]int64, 0, len(members))
	for _, member := range members {
		userIDs = append(userIDs, member.UserID)
	}
	rows, err := s.pool.Query(ctx, `select id, username, display_name from users where id = any($1)`, userIDs)
	if err != nil {
		return err
	}
	defer rows.Close()
	type identity struct{ username, displayName string }
	found := make(map[int64]identity, len(members))
	for rows.Next() {
		var id int64
		var username, displayName string
		if err := rows.Scan(&id, &username, &displayName); err != nil {
			return err
		}
		found[id] = identity{username: username, displayName: displayName}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, member := range members {
		user, ok := found[member.UserID]
		if !ok {
			return fmt.Errorf("duty member references unknown system user %d", member.UserID)
		}
		if member.Username != user.username || member.Name != user.displayName {
			return fmt.Errorf("duty member identity for user %d is stale; reload the user directory", member.UserID)
		}
	}
	return nil
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
	memberIDs := make(map[string]struct{}, len(data.Members))
	userIDs := make(map[int64]struct{}, len(data.Members))
	for _, member := range data.Members {
		if strings.TrimSpace(member.ID) == "" || member.UserID <= 0 || strings.TrimSpace(member.Name) == "" || strings.TrimSpace(member.Username) == "" {
			return errors.New("duty member must reference a system user")
		}
		if _, exists := memberIDs[member.ID]; exists {
			return fmt.Errorf("duplicate duty member id %q", member.ID)
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
		memberIDs[member.ID] = struct{}{}
		members[member.Name] = struct{}{}
	}

	for _, schedule := range data.Schedules {
		if strings.TrimSpace(schedule.ID) == "" || strings.TrimSpace(schedule.Name) == "" {
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
		if strings.TrimSpace(person.ID) == "" {
			return errors.New("current duty person id is required")
		}
		if _, exists := members[person.Name]; !exists {
			return fmt.Errorf("current duty person %q is not in the roster", person.Name)
		}
		if _, exists := teams[person.Team]; !exists {
			return fmt.Errorf("current duty person team %q does not exist", person.Team)
		}
	}

	for _, handover := range data.Handovers {
		if strings.TrimSpace(handover.ID) == "" {
			return errors.New("duty handover id is required")
		}
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
