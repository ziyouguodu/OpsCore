package store

import (
	"context"

	"github.com/jackc/pgx/v5"

	"opscore/backend/internal/models"
)

type dashboardQueueQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func (s *Store) DashboardQueues(ctx context.Context) (models.DashboardQueues, error) {
	return readDashboardQueues(ctx, s.pool)
}

func readDashboardQueues(ctx context.Context, querier dashboardQueueQuerier) (models.DashboardQueues, error) {
	data := models.DashboardQueues{
		BusinessSystems: []string{},
		Incidents:       []models.Incident{},
		Tasks:           []models.Task{},
	}
	rows, err := querier.Query(ctx, `select business from (
		select btrim(business) as business from assets
		union select btrim(business) as business from middleware_instances
	) systems where business <> '' order by business`)
	if err != nil {
		return data, err
	}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return data, err
		}
		data.BusinessSystems = append(data.BusinessSystems, name)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return data, err
	}
	rows.Close()

	rows, err = querier.Query(ctx, `select i.id, i.title, i.level, i.status, coalesce(u.display_name, i.owner), i.owner_user_id,
		i.business, coalesce(i.started_at::text, i.started_at_legacy), coalesce(i.recovered_at::text, i.recovered_at_legacy),
		i.summary, i.created_at, i.updated_at
		from incidents i left join users u on u.id=i.owner_user_id
		where i.status <> '已关闭'
		order by case i.level when 'P1' then 1 when 'P2' then 2 when 'P3' then 3 else 4 end,
		i.updated_at desc, i.id desc limit 4`)
	if err != nil {
		return data, err
	}
	for rows.Next() {
		var item models.Incident
		if err := rows.Scan(&item.ID, &item.Title, &item.Level, &item.Status, &item.Owner, &item.OwnerUserID, &item.Business, &item.StartedAt, &item.RecoveredAt, &item.Summary, &item.CreatedAt, &item.UpdatedAt); err != nil {
			rows.Close()
			return data, err
		}
		data.Incidents = append(data.Incidents, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return data, err
	}
	rows.Close()

	rows, err = querier.Query(ctx, `select t.id, t.title, t.type, coalesce(u.display_name, t.assignee), t.assignee_user_id,
		t.status, coalesce(t.due_at::text, t.due_at_legacy), t.description, t.created_at, t.updated_at
		from tasks t left join users u on u.id=t.assignee_user_id
		where t.status not in ('已完成','已关闭')
		order by t.due_at asc nulls last, t.updated_at desc, t.id desc limit 6`)
	if err != nil {
		return data, err
	}
	defer rows.Close()
	for rows.Next() {
		var item models.Task
		if err := rows.Scan(&item.ID, &item.Title, &item.Type, &item.Assignee, &item.AssigneeUserID, &item.Status, &item.DueAt, &item.Description, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return data, err
		}
		data.Tasks = append(data.Tasks, item)
	}
	return data, rows.Err()
}
