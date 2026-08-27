package query

import (
	"testing"

	"github.com/topbase/topbase/internal/core"
	"github.com/topbase/topbase/internal/core/queryir"
)

type memFields struct{ items []core.FieldMeta }

func (m memFields) SaveField(core.FieldMeta) error { return nil }
func (m memFields) ListFields(string, string, string) ([]core.FieldMeta, error) {
	return m.items, nil
}
func (m memFields) ListDatabaseFields(string) ([]core.FieldMeta, error) { return m.items, nil }

func TestExpanderImplicitJoin(t *testing.T) {
	e := &Expander{Fields: memFields{items: []core.FieldMeta{{
		DatabaseID: "pg_1", Schema: "public", Table: "orders", Name: "user_id", SemanticType: "ForeignKey",
		FKTarget: &core.FieldRef{Schema: "public", Table: "users", Name: "id"},
	}}}}
	q := queryir.Query{
		Version: 1,
		Source:  queryir.Source{DatabaseID: "pg_1", Table: &queryir.TableRef{Schema: "public", Name: "orders"}},
		Fields:  []string{"orders.id", "users.email"},
	}
	out, err := e.Expand(q)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Joins) != 1 || out.Joins[0].Table.Name != "users" {
		t.Fatalf("joins %+v", out.Joins)
	}
}
