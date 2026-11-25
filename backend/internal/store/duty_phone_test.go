package store

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"opscore/backend/internal/models"
)

type dutyPhoneQuery struct {
	rows    *dashboardRows
	err     error
	calls   int
	userIDs []int64
}

func (q *dutyPhoneQuery) Query(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
	q.calls++
	if strings.Join(strings.Fields(sql), " ") != "select id, phone from users where id = any($1) order by id for share" || len(args) != 1 {
		return nil, errors.New("unexpected duty phone query")
	}
	q.userIDs = args[0].([]int64)
	return q.rows, q.err
}

func TestResolveDutyPhonesOverridesSubmittedAndSnapshotPhones(t *testing.T) {
	for _, source := range []string{"relations", "snapshot"} {
		t.Run(source, func(t *testing.T) {
			data := models.DutyCenterData{
				Members: []models.DutyMember{
					{ID: "a", UserID: 7, Name: "主值", Phone: "stale-member"},
					{ID: "b", UserID: 8, Name: "备值", Phone: "stale-cleared"},
					{ID: "c", UserID: 9, Name: "缺失用户", Phone: "stale-missing"},
				},
				CurrentPeople: []models.DutyCurrentPerson{
					{Name: "主值", Phone: "submitted-current"},
					{Name: "备值", Phone: "old-cleared"},
					{Name: "缺失用户", Phone: "old-missing"},
					{Name: "未关联成员", Phone: "untrusted"},
				},
			}
			if source == "snapshot" {
				raw, err := json.Marshal(data)
				if err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal(raw, &data); err != nil {
					t.Fatal(err)
				}
			}
			querier := &dutyPhoneQuery{rows: &dashboardRows{values: [][]any{{int64(7), "+86 138-0013-8000"}, {int64(8), ""}}}}
			if err := resolveDutyPhones(context.Background(), querier, &data); err != nil {
				t.Fatal(err)
			}
			if querier.calls != 1 || !reflect.DeepEqual(querier.userIDs, []int64{7, 8, 9}) || !querier.rows.closed {
				t.Fatalf("must batch read members by user ID and close rows: %+v", querier)
			}
			for i, member := range data.Members {
				want := ""
				if i == 0 {
					want = "+86 138-0013-8000"
				}
				if member.Phone != want || data.CurrentPeople[i].Phone != want {
					t.Fatalf("non-authoritative duty phone at %d: %+v", i, data)
				}
			}
			if data.CurrentPeople[3].Phone != "" {
				t.Fatal("unlinked current person must not retain a submitted phone")
			}
			raw, err := json.Marshal(data)
			if err != nil {
				t.Fatal(err)
			}
			for _, forbidden := range []string{"stale", "submitted", "old-", "untrusted"} {
				if strings.Contains(string(raw), forbidden) {
					t.Fatalf("snapshot must not persist untrusted phones: %s", raw)
				}
			}
		})
	}
}

func TestResolveDutyPhonesWithoutMembersClearsOrphanedPhone(t *testing.T) {
	data := models.DutyCenterData{CurrentPeople: []models.DutyCurrentPerson{{Name: "orphan", Phone: "stale"}}}
	querier := &dutyPhoneQuery{}
	if err := resolveDutyPhones(context.Background(), querier, &data); err != nil || querier.calls != 0 || data.CurrentPeople[0].Phone != "" {
		t.Fatalf("empty roster must clear old phones without a query: %+v %v", data, err)
	}
}

func TestResolveDutyPhonesPropagatesReadFailures(t *testing.T) {
	for _, failure := range []string{"query", "scan", "rows"} {
		t.Run(failure, func(t *testing.T) {
			querier := &dutyPhoneQuery{rows: &dashboardRows{}}
			switch failure {
			case "query":
				querier.err = errors.New("query failed")
			case "scan":
				querier.rows.values = [][]any{{int64(7), ""}}
				querier.rows.scanErr = errors.New("scan failed")
			case "rows":
				querier.rows.rowErr = errors.New("rows failed")
			}
			data := models.DutyCenterData{Members: []models.DutyMember{{UserID: 7}}}
			if err := resolveDutyPhones(context.Background(), querier, &data); err == nil {
				t.Fatal("phone read error must not be hidden")
			}
			if failure != "query" && !querier.rows.closed {
				t.Fatal("phone rows must close on error")
			}
		})
	}
}
