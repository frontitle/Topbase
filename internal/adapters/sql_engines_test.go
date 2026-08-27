package adapters

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/topbase/topbase/internal/core"
	"github.com/topbase/topbase/internal/core/queryir"
)

func TestEngineDefinitionsAreActuallyRegistered(t *testing.T) {
	want := map[string]bool{"postgres": false, "mysql": false, "clickhouse": false, "sqlserver": false, "oracle": false, "sqlite": false}
	for _, engine := range EngineDefinitions() {
		if _, ok := want[engine.ID]; ok {
			want[engine.ID] = true
		}
		if driverName(engine.Family) == "" {
			t.Fatalf("engine %s has no database/sql driver", engine.ID)
		}
	}
	for engine, found := range want {
		if !found {
			t.Fatalf("engine %s is missing", engine)
		}
	}
}

func TestSQLiteConnectionQueryAndMetadata(t *testing.T) {
	connector := NewPostgresConnector()
	database, err := connector.Connect(context.Background(), core.ConnectionRequest{
		Name: "local analytics", Engine: "sqlite", Database: filepath.Join(t.TempDir(), "analytics.db"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer connector.Close(database.ID)
	db, err := connector.db(database.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE stores (id INTEGER PRIMARY KEY, name TEXT NOT NULL, parent_id INTEGER REFERENCES stores(id)); INSERT INTO stores(name) VALUES ('上海')`); err != nil {
		t.Fatal(err)
	}
	result, err := connector.Execute(context.Background(), database.ID, `SELECT "id", "name" FROM "main"."stores" WHERE "name" = $1`, "上海")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 1 || result.Rows[0][1] != "上海" {
		t.Fatalf("unexpected rows: %#v", result.Rows)
	}
	tables, err := connector.Tables(context.Background(), database.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tables) != 1 || tables[0].Name != "stores" || len(tables[0].Columns) != 3 {
		t.Fatalf("unexpected metadata: %#v", tables)
	}
	if !tables[0].Columns[0].PrimaryKey || tables[0].Columns[2].ForeignKey == nil {
		t.Fatalf("primary/foreign key metadata was not read: %#v", tables[0].Columns)
	}
}

func TestCompileForMainstreamSQLDialects(t *testing.T) {
	q := queryir.Query{
		Version:      1,
		Source:       queryir.Source{DatabaseID: "db", Table: &queryir.TableRef{Schema: "analytics", Name: "orders"}},
		Filters:      []queryir.Filter{{Field: "name", Op: "contains", Value: "茶"}},
		Aggregations: []queryir.Aggregation{{Fn: "count"}},
		GroupBy:      []queryir.Breakout{{Field: "created_at", Temporal: "day"}},
		Limit:        50,
	}
	tests := []struct {
		engine string
		parts  []string
	}{
		{engine: "postgres", parts: []string{`date_trunc('day', "created_at")`, `ILIKE '%' || $1 || '%'`, `LIMIT 50`}},
		{engine: "mysql", parts: []string{"DATE(`created_at`)", "LIKE CONCAT('%', $1, '%')", "LIMIT 50"}},
		{engine: "clickhouse", parts: []string{`dateTrunc('day', "created_at")`, "LIMIT 50"}},
		{engine: "sqlserver", parts: []string{`SELECT TOP (50)`, `DATEADD(day, DATEDIFF(day, 0, "created_at"), 0)`, `LIKE '%' + $1 + '%'`}},
		{engine: "oracle", parts: []string{`TRUNC("created_at", 'DD')`, `LIKE '%' || $1 || '%'`, `FETCH FIRST 50 ROWS ONLY`}},
		{engine: "sqlite", parts: []string{`strftime('%Y-%m-%d', "created_at")`, `LIKE '%' || $1 || '%'`, `LIMIT 50`}},
	}
	for _, test := range tests {
		t.Run(test.engine, func(t *testing.T) {
			compiled, err := CompileForEngine(test.engine, q)
			if err != nil {
				t.Fatal(err)
			}
			for _, part := range test.parts {
				if !strings.Contains(compiled.SQL, part) {
					t.Fatalf("%s SQL does not contain %q:\n%s", test.engine, part, compiled.SQL)
				}
			}
		})
	}
}

func TestRebindSQLParameters(t *testing.T) {
	statement := `SELECT * FROM t WHERE a = $1 AND b = $2`
	if got := rebindSQL(statement, "mysql"); got != `SELECT * FROM t WHERE a = ? AND b = ?` {
		t.Fatalf("mysql rebind = %s", got)
	}
	if got := rebindSQL(statement, "sqlserver"); got != `SELECT * FROM t WHERE a = @p1 AND b = @p2` {
		t.Fatalf("sqlserver rebind = %s", got)
	}
}
