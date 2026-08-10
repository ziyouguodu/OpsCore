package store

import (
	"context"

	"opscore/backend/internal/models"
)

func (s *Store) Dashboard(ctx context.Context) (models.Dashboard, error) {
	var data models.Dashboard
	data.AssetTypeCounts = map[string]int64{}
	data.IncidentLevelCounts = map[string]int64{}
	data.TaskStatusCounts = map[string]int64{}
	var serverCount, databaseCount, middlewareCount int64
	if err := s.pool.QueryRow(ctx, `
		select
			(select count(*) from assets) + (select count(*) from middleware_instances),
			coalesce(
				nullif((select count(*) from duty_current_people), 0),
				(select count(*) from oncall_schedules where date_value = current_date or (rule_type = 'weekly' and week_value = to_char(current_date, 'IYYY-"W"IW')))
			),
			(select count(*) from tasks where status in ('待处理','处理中','待确认')),
			(select count(*) from incidents where status in ('新建','处理中','已恢复')),
			(select count(*) from assets where status in ('运行中','正常','启用')) +
			(select count(*) from middleware_instances where status in ('运行中','正常','启用')),
			(select count(*) from assets where status not in ('运行中','正常','启用')) +
			(select count(*) from middleware_instances where status not in ('运行中','正常','启用')),
			(select count(*) from tasks where status in ('已完成','已关闭')),
			(select count(*) from tasks where status not in ('已完成','已关闭')),
			(select count(*) from incidents where status='已关闭'),
			(select coalesce(sum(greatest(0, extract(epoch from (recovered_at - started_at)) / 60)) filter (where started_at is not null and recovered_at is not null), 0)::bigint from incidents),
			(select count(*) from incidents where started_at is not null and recovered_at is not null),
			(select count(*) from assets),
			(select count(*) from middleware_instances where kind in ('MySQL','PostgreSQL','达梦')),
			(select count(*) from middleware_instances where kind not in ('MySQL','PostgreSQL','达梦'))
	`).Scan(
		&data.AssetCount,
		&data.TodayOnCallCount,
		&data.ActiveTaskCount,
		&data.ActiveIncidentCount,
		&data.AssetHealthyCount,
		&data.AssetAbnormalCount,
		&data.TaskClosedCount,
		&data.TaskOpenCount,
		&data.IncidentClosedCount,
		&data.ResponseMinutesTotal,
		&data.ResponseSampleCount,
		&serverCount,
		&databaseCount,
		&middlewareCount,
	); err != nil {
		return data, err
	}
	data.AssetTypeCounts["服务器"] = serverCount
	data.AssetTypeCounts["数据库"] = databaseCount
	data.AssetTypeCounts["中间件"] = middlewareCount
	rows, err := s.pool.Query(ctx, `select status, count(*) from tasks group by status`)
	if err != nil {
		return data, err
	}
	for rows.Next() {
		var key string
		var value int64
		if err := rows.Scan(&key, &value); err != nil {
			rows.Close()
			return data, err
		}
		data.TaskStatusCounts[key] = value
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return data, err
	}
	rows.Close()
	rows, err = s.pool.Query(ctx, `select level, count(*) from incidents where status <> '已关闭' group by level`)
	if err != nil {
		return data, err
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var value int64
		if err := rows.Scan(&key, &value); err != nil {
			return data, err
		}
		data.IncidentLevelCounts[key] = value
	}
	return data, rows.Err()
}
