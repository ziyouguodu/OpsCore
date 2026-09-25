package store

import (
	"context"
	"fmt"
	"strings"

	"opscore/backend/internal/models"
)

type rowScanner interface {
	Scan(...any) error
}

func normalizeListQuery(query models.ListQuery) models.ListQuery {
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}
	query.Keyword = strings.TrimSpace(query.Keyword)
	if strings.ToLower(query.Order) != "asc" {
		query.Order = "desc"
	} else {
		query.Order = "asc"
	}
	return query
}

func pageCount(total int64, pageSize int) int {
	if total == 0 {
		return 1
	}
	return int((total + int64(pageSize) - 1) / int64(pageSize))
}

func clampListPage(query *models.ListQuery, total int64) int {
	pages := pageCount(total, query.PageSize)
	if query.Page > pages {
		query.Page = pages
	}
	return pages
}

func listOptions(ctx context.Context, s *Store, table, column string) ([]string, error) {
	allowed := map[string]map[string]bool{
		"assets":               {"business": true, "network_zone": true},
		"middleware_instances": {"business": true, "network_zone": true},
	}
	if !allowed[table][column] {
		return nil, fmt.Errorf("unsupported list option %s.%s", table, column)
	}
	rows, err := s.pool.Query(ctx, fmt.Sprintf(`select distinct %s from %s where %s <> '' order by %s`, column, table, column, column))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]string, 0)
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (s *Store) ListAssetsPage(ctx context.Context, input models.ListQuery) (models.PageResult[models.Asset], error) {
	query := normalizeListQuery(input)
	orderBy := map[string]string{"assetNo": "asset_no", "type": "type", "environment": "environment", "status": "status", "updatedAt": "updated_at"}[query.Sort]
	if orderBy == "" {
		orderBy = "updated_at"
	}
	args := []any{query.Keyword, query.Type, query.Environment, query.Business, query.NetworkZone, query.Status, query.IPs}
	where := `where ($1='' or (asset_no || ' ' || business || ' ' || ipv4 || ' ' || ipv6 || ' ' || owner || ' ' || deployment_info || ' ' || hostname) ilike '%' || $1 || '%')
		and ($2='' or type=$2) and ($3='' or environment=$3) and ($4='' or business=$4)
		and ($5='' or network_zone=$5) and ($6='' or status=$6)
		and (coalesce(cardinality($7::text[]), 0) = 0 or lower(ipv4) = any($7::text[]) or lower(ipv6) = any($7::text[]))`
	var result models.PageResult[models.Asset]
	if err := s.pool.QueryRow(ctx, `select count(*) from assets `+where, args...).Scan(&result.Total); err != nil {
		return result, err
	}
	pages := clampListPage(&query, result.Total)
	offset := (query.Page - 1) * query.PageSize
	rows, err := s.pool.Query(ctx, `select id, coalesce(created_by, 0), asset_no, type, vendor, cpu_arch, sn, location, business, ipv4, ipv6, environment, os, hostname, network_zone, cpu, memory, disk, deployment_info, owner, status, host_machine, created_at, updated_at from assets `+where+` order by `+orderBy+` `+query.Order+`, id desc limit $8 offset $9`, append(args, query.PageSize, offset)...)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	result.Items = make([]models.Asset, 0, query.PageSize)
	for rows.Next() {
		var item models.Asset
		if err := scanAsset(rows, &item); err != nil {
			return result, err
		}
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		return result, err
	}
	businesses, err := listOptions(ctx, s, "assets", "business")
	if err != nil {
		return result, err
	}
	zones, err := listOptions(ctx, s, "assets", "network_zone")
	if err != nil {
		return result, err
	}
	result.Page, result.PageSize, result.PageCount = query.Page, query.PageSize, pages
	result.Options = map[string][]string{"business": businesses, "networkZone": zones}
	return result, nil
}

func scanAsset(scanner rowScanner, item *models.Asset) error {
	return scanner.Scan(&item.ID, &item.CreatedBy, &item.AssetNo, &item.Type, &item.Vendor, &item.CPUArch, &item.SN, &item.Location, &item.Business, &item.IPv4, &item.IPv6, &item.Environment, &item.OS, &item.Hostname, &item.NetworkZone, &item.CPU, &item.Memory, &item.Disk, &item.DeploymentInfo, &item.Owner, &item.Status, &item.HostMachine, &item.CreatedAt, &item.UpdatedAt)
}

func (s *Store) ListMiddlewarePage(ctx context.Context, input models.ListQuery) (models.PageResult[models.MiddlewareInstance], error) {
	query := normalizeListQuery(input)
	orderBy := map[string]string{"name": "name", "kind": "kind", "environment": "environment", "status": "status", "updatedAt": "updated_at"}[query.Sort]
	if orderBy == "" {
		orderBy = "updated_at"
	}
	args := []any{query.Keyword, query.Kind, query.Environment, query.Business, query.NetworkZone, query.Status, query.IPs}
	endpointHost := `split_part(split_part(split_part(regexp_replace(endpoint, '^[a-zA-Z][a-zA-Z0-9+.-]*://', ''), '/', 1), '?', 1), '#', 1)`
	endpointIP := `case when ` + endpointHost + ` like '[%' then substring(` + endpointHost + ` from '^\[([0-9A-Fa-f:.]+)\]') when ` + endpointHost + ` ~ '^[0-9]{1,3}(\.[0-9]{1,3}){3}:[0-9]+$' then split_part(` + endpointHost + `, ':', 1) else ` + endpointHost + ` end`
	where := `where ($1='' or (name || ' ' || kind || ' ' || endpoint || ' ' || business || ' ' || owner) ilike '%' || $1 || '%')
		and ($2='' or kind=$2) and ($3='' or environment=$3) and ($4='' or business=$4)
		and ($5='' or network_zone=$5) and ($6='' or status=$6)
		and (coalesce(cardinality($7::text[]), 0) = 0 or lower(` + endpointIP + `) = any($7::text[]))`
	pageWhere := `where ($1='' or (m.name || ' ' || m.kind || ' ' || m.endpoint || ' ' || m.business || ' ' || m.owner) ilike '%' || $1 || '%')
		and ($2='' or m.kind=$2) and ($3='' or m.environment=$3) and ($4='' or m.business=$4)
		and ($5='' or m.network_zone=$5) and ($6='' or m.status=$6)
		and (coalesce(cardinality($7::text[]), 0) = 0 or lower(` + strings.ReplaceAll(endpointIP, "endpoint", "m.endpoint") + `) = any($7::text[]))`
	var result models.PageResult[models.MiddlewareInstance]
	if err := s.pool.QueryRow(ctx, `select count(*) from middleware_instances `+where, args...).Scan(&result.Total); err != nil {
		return result, err
	}
	pages := clampListPage(&query, result.Total)
	offset := (query.Page - 1) * query.PageSize
	rows, err := s.pool.Query(ctx, `select m.id, m.name, m.kind, m.version, m.environment, m.network_zone, m.endpoint, m.business, m.owner, m.status, m.asset_id, coalesce(a.asset_no, ''), m.created_at, m.updated_at from middleware_instances m left join assets a on a.id=m.asset_id `+pageWhere+` order by m.`+orderBy+` `+query.Order+`, m.id desc limit $8 offset $9`, append(args, query.PageSize, offset)...)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	result.Items = make([]models.MiddlewareInstance, 0, query.PageSize)
	for rows.Next() {
		var item models.MiddlewareInstance
		if err := rows.Scan(&item.ID, &item.Name, &item.Kind, &item.Version, &item.Environment, &item.NetworkZone, &item.Endpoint, &item.Business, &item.Owner, &item.Status, &item.AssetID, &item.AssetNo, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return result, err
		}
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		return result, err
	}
	businesses, err := listOptions(ctx, s, "middleware_instances", "business")
	if err != nil {
		return result, err
	}
	zones, err := listOptions(ctx, s, "middleware_instances", "network_zone")
	if err != nil {
		return result, err
	}
	result.Page, result.PageSize, result.PageCount = query.Page, query.PageSize, pages
	result.Options = map[string][]string{"business": businesses, "networkZone": zones}
	return result, nil
}

func (s *Store) ListTasksPage(ctx context.Context, input models.ListQuery) (models.PageResult[models.Task], error) {
	query := normalizeListQuery(input)
	orderBy := map[string]string{"title": "title", "status": "status", "dueAt": "due_at", "updatedAt": "updated_at"}[query.Sort]
	if orderBy == "" {
		orderBy = "updated_at"
	}
	args := []any{query.Keyword, query.Status}
	where := `where ($1='' or (title || ' ' || assignee || ' ' || description) ilike '%' || $1 || '%') and ($2='' or status=$2)`
	var result models.PageResult[models.Task]
	if err := s.pool.QueryRow(ctx, `select count(*) from tasks `+where, args...).Scan(&result.Total); err != nil {
		return result, err
	}
	pages := clampListPage(&query, result.Total)
	offset := (query.Page - 1) * query.PageSize
	rows, err := s.pool.Query(ctx, `select id, title, type, assignee, assignee_user_id, status, coalesce(due_at::text, due_at_legacy), description, created_at, updated_at from tasks `+where+` order by `+orderBy+` `+query.Order+`, id desc limit $3 offset $4`, append(args, query.PageSize, offset)...)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	result.Items = make([]models.Task, 0, query.PageSize)
	for rows.Next() {
		var item models.Task
		if err := rows.Scan(&item.ID, &item.Title, &item.Type, &item.Assignee, &item.AssigneeUserID, &item.Status, &item.DueAt, &item.Description, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return result, err
		}
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		return result, err
	}
	result.Counts, err = s.taskStatusCounts(ctx)
	if err != nil {
		return result, err
	}
	result.Page, result.PageSize, result.PageCount = query.Page, query.PageSize, pages
	return result, nil
}

func (s *Store) taskStatusCounts(ctx context.Context) (map[string]int64, error) {
	return groupedCounts(ctx, s, `select status, count(*) from tasks group by status`)
}

func (s *Store) ListIncidentsPage(ctx context.Context, input models.ListQuery) (models.PageResult[models.Incident], error) {
	query := normalizeListQuery(input)
	orderBy := map[string]string{"title": "title", "level": "level", "status": "status", "startedAt": "started_at", "updatedAt": "updated_at"}[query.Sort]
	if orderBy == "" {
		orderBy = "updated_at"
	}
	args := []any{query.Keyword, query.Level, query.Status, query.Business}
	where := `where ($1='' or (title || ' ' || owner || ' ' || business || ' ' || summary) ilike '%' || $1 || '%') and ($2='' or level=$2) and ($3='' or status=$3) and ($4='' or business=$4)`
	var result models.PageResult[models.Incident]
	if err := s.pool.QueryRow(ctx, `select count(*) from incidents `+where, args...).Scan(&result.Total); err != nil {
		return result, err
	}
	pages := clampListPage(&query, result.Total)
	offset := (query.Page - 1) * query.PageSize
	rows, err := s.pool.Query(ctx, `select id, title, level, status, owner, owner_user_id, business, coalesce(started_at::text, started_at_legacy), coalesce(recovered_at::text, recovered_at_legacy), summary, created_at, updated_at from incidents `+where+` order by `+orderBy+` `+query.Order+`, id desc limit $5 offset $6`, append(args, query.PageSize, offset)...)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	result.Items = make([]models.Incident, 0, query.PageSize)
	for rows.Next() {
		var item models.Incident
		if err := rows.Scan(&item.ID, &item.Title, &item.Level, &item.Status, &item.Owner, &item.OwnerUserID, &item.Business, &item.StartedAt, &item.RecoveredAt, &item.Summary, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return result, err
		}
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		return result, err
	}
	result.Counts, err = groupedCounts(ctx, s, `select level, count(*) from incidents group by level`)
	if err != nil {
		return result, err
	}
	result.Page, result.PageSize, result.PageCount = query.Page, query.PageSize, pages
	return result, nil
}

func groupedCounts(ctx context.Context, s *Store, query string) (map[string]int64, error) {
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := make(map[string]int64)
	for rows.Next() {
		var key string
		var count int64
		if err := rows.Scan(&key, &count); err != nil {
			return nil, err
		}
		counts[key] = count
	}
	return counts, rows.Err()
}
