package store

import (
	"context"
	"errors"

	"opscore/backend/internal/domain"
	"opscore/backend/internal/models"
)

func (s *Store) ListTasks(ctx context.Context) ([]models.Task, error) {
	rows, err := s.pool.Query(ctx, `select id, title, type, assignee, assignee_user_id, status, coalesce(due_at::text, due_at_legacy), description, created_at, updated_at from tasks order by updated_at desc`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.Task{}
	for rows.Next() {
		var item models.Task
		if err := rows.Scan(&item.ID, &item.Title, &item.Type, &item.Assignee, &item.AssigneeUserID, &item.Status, &item.DueAt, &item.Description, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateTask(ctx context.Context, item models.Task) (models.Task, error) {
	assignee, err := resolveUserDisplayName(ctx, s.pool, item.AssigneeUserID, item.Assignee)
	if err != nil {
		return models.Task{}, err
	}
	item.Assignee = assignee
	row := s.pool.QueryRow(ctx, `insert into tasks(title, type, assignee, assignee_user_id, status, due_at, due_at_legacy, description) values ($1,$2,$3,$4,$5,nullif($6,'')::timestamptz,$6,$7) returning id, created_at, updated_at`, item.Title, item.Type, item.Assignee, item.AssigneeUserID, item.Status, item.DueAt, item.Description)
	if err := row.Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return models.Task{}, err
	}
	return item, nil
}

func (s *Store) UpdateTask(ctx context.Context, id int64, item models.Task) (models.Task, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.Task{}, err
	}
	defer tx.Rollback(ctx)
	item.Assignee, err = resolveUserDisplayName(ctx, tx, item.AssigneeUserID, item.Assignee)
	if err != nil {
		return models.Task{}, err
	}
	var current string
	if err := tx.QueryRow(ctx, `select status from tasks where id=$1 for update`, id).Scan(&current); err != nil {
		return models.Task{}, err
	}
	if current != item.Status && !domain.CanTransitionTask(domain.TaskStatus(current), domain.TaskStatus(item.Status)) {
		return models.Task{}, ErrStatusConflict
	}
	row := tx.QueryRow(ctx, `
		update tasks set
				title=$2, type=$3, assignee=$4, assignee_user_id=$5, status=$6,
				due_at=nullif($7,'')::timestamptz, due_at_legacy=$7, description=$8, updated_at=now()
		where id=$1
		returning id, created_at, updated_at
		`, id, item.Title, item.Type, item.Assignee, item.AssigneeUserID, item.Status, item.DueAt, item.Description)
	if err := row.Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return models.Task{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return models.Task{}, err
	}
	return item, nil
}

func (s *Store) DeleteTask(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `delete from tasks where id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("task not found")
	}
	return nil
}

func (s *Store) UpdateTaskStatus(ctx context.Context, id int64, status string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var current string
	if err := tx.QueryRow(ctx, `select status from tasks where id=$1 for update`, id).Scan(&current); err != nil {
		return err
	}
	if current != status && !domain.CanTransitionTask(domain.TaskStatus(current), domain.TaskStatus(status)) {
		return ErrStatusConflict
	}
	if _, err := tx.Exec(ctx, `update tasks set status=$2, updated_at=now() where id=$1`, id, status); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) GetTaskStatus(ctx context.Context, id int64) (string, error) {
	var status string
	err := s.pool.QueryRow(ctx, `select status from tasks where id=$1`, id).Scan(&status)
	return status, err
}

func (s *Store) ListIncidents(ctx context.Context) ([]models.Incident, error) {
	rows, err := s.pool.Query(ctx, `select id, title, level, status, owner, owner_user_id, business, coalesce(started_at::text, started_at_legacy), coalesce(recovered_at::text, recovered_at_legacy), summary, created_at, updated_at from incidents order by updated_at desc`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.Incident{}
	for rows.Next() {
		var item models.Incident
		if err := rows.Scan(&item.ID, &item.Title, &item.Level, &item.Status, &item.Owner, &item.OwnerUserID, &item.Business, &item.StartedAt, &item.RecoveredAt, &item.Summary, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateIncident(ctx context.Context, item models.Incident) (models.Incident, error) {
	owner, err := resolveUserDisplayName(ctx, s.pool, item.OwnerUserID, item.Owner)
	if err != nil {
		return models.Incident{}, err
	}
	item.Owner = owner
	row := s.pool.QueryRow(ctx, `insert into incidents(title, level, status, owner, owner_user_id, business, started_at, started_at_legacy, recovered_at, recovered_at_legacy, summary) values ($1,$2,$3,$4,$5,$6,nullif($7,'')::timestamptz,$7,nullif($8,'')::timestamptz,$8,$9) returning id, created_at, updated_at`, item.Title, item.Level, item.Status, item.Owner, item.OwnerUserID, item.Business, item.StartedAt, item.RecoveredAt, item.Summary)
	if err := row.Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return models.Incident{}, err
	}
	return item, nil
}

func (s *Store) UpdateIncident(ctx context.Context, id int64, item models.Incident) (models.Incident, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return models.Incident{}, err
	}
	defer tx.Rollback(ctx)
	item.Owner, err = resolveUserDisplayName(ctx, tx, item.OwnerUserID, item.Owner)
	if err != nil {
		return models.Incident{}, err
	}
	var current string
	if err := tx.QueryRow(ctx, `select status from incidents where id=$1 for update`, id).Scan(&current); err != nil {
		return models.Incident{}, err
	}
	if current != item.Status && !domain.CanTransitionIncident(domain.IncidentStatus(current), domain.IncidentStatus(item.Status)) {
		return models.Incident{}, ErrStatusConflict
	}
	row := tx.QueryRow(ctx, `
		update incidents set
				title=$2, level=$3, status=$4, owner=$5, owner_user_id=$6, business=$7,
				started_at=nullif($8,'')::timestamptz, started_at_legacy=$8,
				recovered_at=nullif($9,'')::timestamptz, recovered_at_legacy=$9,
				summary=$10, updated_at=now()
		where id=$1
		returning id, created_at, updated_at
		`, id, item.Title, item.Level, item.Status, item.Owner, item.OwnerUserID, item.Business, item.StartedAt, item.RecoveredAt, item.Summary)
	if err := row.Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return models.Incident{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return models.Incident{}, err
	}
	return item, nil
}

func (s *Store) DeleteIncident(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `delete from incidents where id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("incident not found")
	}
	return nil
}

func (s *Store) UpdateIncidentStatus(ctx context.Context, id int64, status string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var current string
	if err := tx.QueryRow(ctx, `select status from incidents where id=$1 for update`, id).Scan(&current); err != nil {
		return err
	}
	if current != status && !domain.CanTransitionIncident(domain.IncidentStatus(current), domain.IncidentStatus(status)) {
		return ErrStatusConflict
	}
	if _, err := tx.Exec(ctx, `update incidents set status=$2, updated_at=now() where id=$1`, id, status); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) GetIncidentStatus(ctx context.Context, id int64) (string, error) {
	var status string
	err := s.pool.QueryRow(ctx, `select status from incidents where id=$1`, id).Scan(&status)
	return status, err
}
