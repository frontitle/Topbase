package appdb

import (
	"context"
	"database/sql"
	"regexp"
	"strconv"
	"strings"
)

type database struct {
	*sql.DB
	engine Engine
}

func (d *database) Exec(query string, args ...any) (sql.Result, error) {
	return d.DB.Exec(d.rewrite(query), normalizeArgs(args)...)
}

func (d *database) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return d.DB.ExecContext(ctx, d.rewrite(query), normalizeArgs(args)...)
}

func (d *database) Query(query string, args ...any) (*sql.Rows, error) {
	return d.DB.Query(d.rewrite(query), normalizeArgs(args)...)
}

func (d *database) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return d.DB.QueryContext(ctx, d.rewrite(query), normalizeArgs(args)...)
}

func (d *database) QueryRow(query string, args ...any) *sql.Row {
	return d.DB.QueryRow(d.rewrite(query), normalizeArgs(args)...)
}

func (d *database) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return d.DB.QueryRowContext(ctx, d.rewrite(query), normalizeArgs(args)...)
}

type transaction struct {
	*sql.Tx
	db *database
}

func (d *database) Begin() (*transaction, error) {
	tx, err := d.DB.Begin()
	if err != nil {
		return nil, err
	}
	return &transaction{Tx: tx, db: d}, nil
}

func (t *transaction) Exec(query string, args ...any) (sql.Result, error) {
	return t.Tx.Exec(t.db.rewrite(query), normalizeArgs(args)...)
}

func (t *transaction) Query(query string, args ...any) (*sql.Rows, error) {
	return t.Tx.Query(t.db.rewrite(query), normalizeArgs(args)...)
}

func (t *transaction) QueryRow(query string, args ...any) *sql.Row {
	return t.Tx.QueryRow(t.db.rewrite(query), normalizeArgs(args)...)
}

func normalizeArgs(args []any) []any {
	for index, value := range args {
		if flag, ok := value.(bool); ok {
			if flag {
				args[index] = 1
			} else {
				args[index] = 0
			}
		}
	}
	return args
}

var (
	groupsWord  = regexp.MustCompile(`\bgroups\b`)
	keyWord     = regexp.MustCompile(`\bkey\b`)
	conflictSet = regexp.MustCompile(`(?is)\s+ON\s+CONFLICT\s*\([^)]*\)\s+DO\s+UPDATE\s+SET\s+(.+)$`)
	excludedCol = regexp.MustCompile(`(?i)excluded\.([a-z_][a-z0-9_]*)`)
)

func (d *database) rewrite(query string) string {
	query = d.quotePortableIdentifiers(query)
	switch d.engine {
	case EnginePostgres:
		if strings.Contains(strings.ToUpper(query), "INSERT OR IGNORE INTO") {
			query = strings.Replace(query, "INSERT OR IGNORE INTO", "INSERT INTO", 1) + " ON CONFLICT DO NOTHING"
		}
		query = rebindPostgres(query)
	case EngineMySQL:
		query = strings.Replace(query, "INSERT OR IGNORE INTO", "INSERT IGNORE INTO", 1)
		if match := conflictSet.FindStringSubmatch(query); len(match) == 2 {
			assignments := excludedCol.ReplaceAllString(match[1], "VALUES($1)")
			query = conflictSet.ReplaceAllString(query, " ON DUPLICATE KEY UPDATE "+assignments)
		}
	}
	return query
}

func (d *database) quotePortableIdentifiers(query string) string {
	if d.engine == EngineSQLite {
		return query
	}
	quote := `"groups"`
	keyQuote := `"key"`
	if d.engine == EngineMySQL {
		quote = "`groups`"
		keyQuote = "`key`"
	}
	query = groupsWord.ReplaceAllString(query, quote)
	return keyWord.ReplaceAllString(query, keyQuote)
}

func rebindPostgres(query string) string {
	var out strings.Builder
	out.Grow(len(query) + 16)
	index := 1
	var single, double bool
	for i := 0; i < len(query); i++ {
		ch := query[i]
		if ch == '\'' && !double {
			if single && i+1 < len(query) && query[i+1] == '\'' {
				out.WriteByte(ch)
				out.WriteByte(query[i+1])
				i++
				continue
			}
			single = !single
		}
		if ch == '"' && !single {
			double = !double
		}
		if ch == '?' && !single && !double {
			out.WriteByte('$')
			out.WriteString(strconv.Itoa(index))
			index++
			continue
		}
		out.WriteByte(ch)
	}
	return out.String()
}
