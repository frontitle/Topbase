package adapters

import (
	"strings"
	"testing"

	"github.com/topbase/topbase/internal/core/queryir"
)

func TestCompilePostgresSnapshots(t *testing.T) {
	cases := []struct {
		name string
		q    queryir.Query
		sql  string
		args int
	}{
		{
			name: "table scan",
			q: queryir.Query{
				Version: 1,
				Source:  queryir.Source{DatabaseID: "pg_1", Table: &queryir.TableRef{Schema: "public", Name: "orders"}},
				Fields:  []string{"id", "amount"},
				Limit:   50,
			},
			sql:  `SELECT "id", "amount" FROM "public"."orders" LIMIT 50`,
			args: 0,
		},
		{
			name: "count by day with filter",
			q: queryir.Query{
				Version:      1,
				Source:       queryir.Source{DatabaseID: "pg_1", Table: &queryir.TableRef{Schema: "public", Name: "orders"}},
				Filters:      []queryir.Filter{{Field: "status", Op: "eq", Value: "paid"}},
				Aggregations: []queryir.Aggregation{{Fn: "count"}},
				GroupBy:      []queryir.Breakout{{Field: "created_at", Temporal: "day"}},
				OrderBy:      []queryir.Order{{Field: "created_at", Dir: "desc"}},
			},
			sql:  `SELECT date_trunc('day', "created_at") AS "created_at_day", count(*) AS "count" FROM "public"."orders" WHERE "status" = $1 GROUP BY date_trunc('day', "created_at") ORDER BY "created_at" DESC LIMIT 1000`,
			args: 1,
		},
		{
			name: "string contains and in list",
			q: queryir.Query{
				Version: 1,
				Source:  queryir.Source{DatabaseID: "pg_1", Table: &queryir.TableRef{Schema: "public", Name: "orders"}},
				Fields:  []string{"id"},
				Filters: []queryir.Filter{
					{Field: "name", Op: "contains", Value: "茶"},
					{Field: "status", Op: "in", Value: []any{"paid", "shipped"}},
				},
				Limit: 20,
			},
			sql:  `SELECT "id" FROM "public"."orders" WHERE "name" ILIKE '%' || $1 || '%' AND "status" IN ($2, $3) LIMIT 20`,
			args: 3,
		},
		{
			name: "left join",
			q: queryir.Query{
				Version: 1,
				Source:  queryir.Source{DatabaseID: "pg_1", Table: &queryir.TableRef{Schema: "public", Name: "orders"}},
				Joins: []queryir.Join{{
					Type: "left", Alias: "users", Table: &queryir.TableRef{Schema: "public", Name: "users"},
					Conditions: []queryir.JoinCondition{{Left: "orders.user_id", Right: "users.id"}},
				}},
				Fields: []string{"orders.id", "users.email"},
				Limit:  20,
			},
			sql:  `SELECT "orders"."id", "users"."email" FROM "public"."orders" AS "orders" LEFT JOIN "public"."users" AS "users" ON "orders"."user_id" = "users"."id" LIMIT 20`,
			args: 0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			compiled, err := CompilePostgres(tc.q)
			if err != nil {
				t.Fatal(err)
			}
			if compiled.SQL != tc.sql {
				t.Fatalf("sql\n got %s\nwant %s", compiled.SQL, tc.sql)
			}
			if len(compiled.Args) != tc.args {
				t.Fatalf("args %d, want %d", len(compiled.Args), tc.args)
			}
		})
	}
}

func TestCompilePostgresRejectsInjection(t *testing.T) {
	q := queryir.Query{
		Version: 1,
		Source:  queryir.Source{DatabaseID: "pg_1", Table: &queryir.TableRef{Schema: "public", Name: "orders;drop"}},
	}
	_, err := CompilePostgres(q)
	if err == nil || !strings.Contains(err.Error(), "invalid table") {
		t.Fatalf("expected identifier rejection, got %v", err)
	}
}

func TestCompilePostgresWarehouseOmitsLimit(t *testing.T) {
	q := queryir.Query{
		Version: 1,
		Source:  queryir.Source{DatabaseID: "pg_1", Table: &queryir.TableRef{Schema: "public", Name: "orders"}},
		Fields:  []string{"id"},
	}
	compiled, err := CompilePostgresWarehouse(q)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(compiled.SQL, "LIMIT") {
		t.Fatalf("warehouse compile should omit LIMIT: %s", compiled.SQL)
	}
}
