package store

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

type dashboardRows struct {
	pgx.Rows
	values  [][]any
	index   int
	closed  bool
	rowErr  error
	scanErr error
}

func (r *dashboardRows) Next() bool {
	if r.index == len(r.values) {
		return false
	}
	r.index++
	return true
}

func (r *dashboardRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	for i, value := range r.values[r.index-1] {
		reflect.ValueOf(dest[i]).Elem().Set(reflect.ValueOf(value))
	}
	return nil
}

func (r *dashboardRows) Close()     { r.closed = true }
func (r *dashboardRows) Err() error { return r.rowErr }

type dashboardQuerier struct {
	rows       []*dashboardRows
	queries    []string
	queryErrAt int
}

func (q *dashboardQuerier) Query(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
	q.queries = append(q.queries, strings.Join(strings.Fields(sql), " "))
	if len(args) != 0 {
		return nil, errors.New("dashboard queues must not depend on page or client filters")
	}
	if q.queryErrAt == len(q.queries) {
		return nil, errors.New("query failed")
	}
	return q.rows[len(q.queries)-1], nil
}

func TestDashboardQueuesReadGlobalBusinessAndBoundedOpenWork(t *testing.T) {
	now := time.Now()
	var userID *int64
	querier := &dashboardQuerier{rows: []*dashboardRows{
		{values: [][]any{{"仅实例业务"}, {"资产业务"}}},
		{values: [][]any{{int64(42), "全局事件", "P1", "已恢复", "当前负责人", userID, "业务", "", "", "摘要", now, now}}},
		{values: [][]any{{int64(81), "全局任务", "任务", "当前处理人", userID, "待确认", "", "描述", now, now}}},
	}}
	data, err := readDashboardQueues(context.Background(), querier)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.BusinessSystems) != 2 || data.BusinessSystems[0] != "仅实例业务" || len(data.Incidents) != 1 || data.Incidents[0].ID != 42 || data.Incidents[0].Owner != "当前负责人" || len(data.Tasks) != 1 || data.Tasks[0].ID != 81 || data.Tasks[0].Assignee != "当前处理人" {
		t.Fatalf("unexpected queues: %+v", data)
	}
	for i, required := range [][]string{
		{"from assets", "union select btrim(business) as business from middleware_instances", "where business <> ''", "order by business"},
		{"where i.status <> '已关闭'", "case i.level when 'P1'", "i.updated_at desc, i.id desc limit 4", "coalesce(u.display_name, i.owner)"},
		{"where t.status not in ('已完成','已关闭')", "t.due_at asc nulls last", "t.id desc limit 6", "coalesce(u.display_name, t.assignee)"},
	} {
		for _, fragment := range required {
			if !strings.Contains(querier.queries[i], fragment) {
				t.Fatalf("query %d missing %q: %s", i, fragment, querier.queries[i])
			}
		}
		if !querier.rows[i].closed {
			t.Fatalf("rows %d were not closed", i)
		}
	}
}

func TestDashboardQueuesEmptyListsAreArrays(t *testing.T) {
	querier := &dashboardQuerier{rows: []*dashboardRows{{}, {}, {}}}
	data, err := readDashboardQueues(context.Background(), querier)
	if err != nil || data.BusinessSystems == nil || data.Incidents == nil || data.Tasks == nil {
		t.Fatalf("expected initialized empty arrays: %+v %v", data, err)
	}
}

func TestDashboardQueuesPropagateReadErrorsAndCloseRows(t *testing.T) {
	for queryIndex := 0; queryIndex < 3; queryIndex++ {
		querier := &dashboardQuerier{rows: []*dashboardRows{{}, {}, {}}, queryErrAt: queryIndex + 1}
		if _, err := readDashboardQueues(context.Background(), querier); err == nil {
			t.Fatalf("expected query %d failure", queryIndex)
		}
		for _, failure := range []string{"scan", "rows"} {
			querier := &dashboardQuerier{rows: []*dashboardRows{{}, {}, {}}}
			row := querier.rows[queryIndex]
			if failure == "scan" {
				row.values = [][]any{{"unused"}}
				row.scanErr = errors.New("scan failed")
			} else {
				row.rowErr = errors.New("rows failed")
			}
			if _, err := readDashboardQueues(context.Background(), querier); err == nil || !row.closed {
				t.Fatalf("expected %s error and rows closed at query %d, got %v", failure, queryIndex, err)
			}
		}
	}
}
