package query

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/topbase/topbase/internal/core"
	"github.com/topbase/topbase/internal/core/queryir"
)

type captureExec struct {
	sql  string
	args []any
}

func (c *captureExec) Execute(_ context.Context, _, sql string, args ...any) (core.QueryResult, error) {
	c.sql, c.args = sql, args
	return core.QueryResult{Columns: []string{"id"}, Rows: [][]any{{1}}}, nil
}

func TestRunDashboardCardNativeFieldFilter(t *testing.T) {
	exec := &captureExec{}
	svc := DatasetService{Queries: core.QueryService{Executor: exec}}
	question := core.Question{
		ID: "qst_sql", QueryType: "native", DatabaseID: "pg_1",
		NativeSQL:  "SELECT * FROM orders WHERE {{created_at}}",
		Parameters: []queryir.Parameter{{Name: "created_at", Type: "date", Field: "created_at"}},
	}
	board := core.Dashboard{
		Cards:   []core.DashboardCard{{ID: "crd_1", Type: "question", QuestionID: "qst_sql"}},
		Filters: []core.DashboardFilter{{ID: "flt_1", Type: "date", Mappings: []core.FilterMapping{{CardID: "crd_1", Field: "created_at"}}}},
	}
	_, err := svc.RunDashboardCard(context.Background(), board, "crd_1", map[string]any{
		"flt_1": map[string]any{"start": "2026-01-01", "end": "2026-01-31"},
	}, memQuestions{item: question})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(exec.sql, `"created_at" >= $1`) {
		t.Fatalf("sql %s", exec.sql)
	}
}

func TestRunDashboardCardUsesSavedChartSpec(t *testing.T) {
	exec := &captureExec{}
	svc := DatasetService{Queries: core.QueryService{Executor: exec}}
	question := core.Question{
		ID: "qst_sql", QueryType: "native", DatabaseID: "pg_1", NativeSQL: "SELECT 1 AS id",
		ChartSpec: &core.ChartSpec{Type: "bar", X: "id", Y: []string{"id"}},
	}
	board := core.Dashboard{
		Cards: []core.DashboardCard{{
			ID: "crd_1", Type: "question", QuestionID: "qst_sql",
			Config: map[string]any{"chartspec": map[string]any{"type": "pie"}},
		}},
	}
	result, err := svc.RunDashboardCard(context.Background(), board, "crd_1", map[string]any{}, memQuestions{item: question})
	if err != nil {
		t.Fatal(err)
	}
	if result.ChartSpec.Type != "pie" {
		t.Fatalf("card override should win, got %+v", result.ChartSpec)
	}
}

func TestRunDashboardCardKeepsQuestionTableView(t *testing.T) {
	exec := &captureExec{}
	svc := DatasetService{Queries: core.QueryService{Executor: exec}}
	hidden := false
	question := core.Question{
		ID: "qst_sql", QueryType: "native", DatabaseID: "pg_1", NativeSQL: "SELECT 1 AS id",
		ChartSpec: &core.ChartSpec{
			Type:   "table",
			Search: "已完成",
			Columns: map[string]core.ChartColumnStyle{
				"id": {Filter: "=1", Visible: &hidden},
			},
		},
	}
	board := core.Dashboard{
		Cards: []core.DashboardCard{{
			ID: "crd_1", Type: "question", QuestionID: "qst_sql",
			Config: map[string]any{"chartspec": map[string]any{"type": "table"}},
		}},
	}
	result, err := svc.RunDashboardCard(context.Background(), board, "crd_1", map[string]any{}, memQuestions{item: question})
	if err != nil {
		t.Fatal(err)
	}
	if result.ChartSpec.Type != "table" {
		t.Fatalf("table question should stay a table, got %+v", result.ChartSpec)
	}
	if result.ChartSpec.Search != "已完成" || result.ChartSpec.Columns["id"].Filter != "=1" {
		t.Fatalf("table filters should come from the question: %+v", result.ChartSpec)
	}
}

type memQuestions struct{ item core.Question }

func (m memQuestions) Create(core.Question) error           { return nil }
func (m memQuestions) Update(core.Question) error           { return nil }
func (m memQuestions) ByID(string) (core.Question, error)   { return m.item, nil }
func (m memQuestions) List(bool) ([]core.Question, error)   { return []core.Question{m.item}, nil }
func (m memQuestions) SetArchived(string, *time.Time) error { return nil }
