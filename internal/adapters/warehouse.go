package adapters

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/topbase/topbase/internal/core"
	"github.com/topbase/topbase/internal/core/queryir"
)

func (p *SQLConnector) Materialize(ctx context.Context, spec core.MaterializeSpec) (core.MaterializeResult, error) {
	if err := queryir.CheckIdent("schema", spec.Schema); err != nil {
		return core.MaterializeResult{}, err
	}
	if err := queryir.CheckIdent("table", spec.Table); err != nil {
		return core.MaterializeResult{}, err
	}
	tmpName := spec.Table + "__new"
	if err := queryir.CheckIdent("temp table", tmpName); err != nil {
		return core.MaterializeResult{}, err
	}
	sqlText := strings.TrimSpace(spec.SQL)
	lower := strings.ToLower(sqlText)
	if !strings.HasPrefix(lower, "select") && !strings.HasPrefix(lower, "with") {
		return core.MaterializeResult{}, fmt.Errorf("materialize source must be a SELECT")
	}
	if strings.Contains(sqlText, ";") {
		return core.MaterializeResult{}, fmt.Errorf("multiple statements are not allowed")
	}

	db, err := p.db(spec.DatabaseID)
	if err != nil {
		return core.MaterializeResult{}, err
	}
	engine, err := p.Engine(spec.DatabaseID)
	if err != nil {
		return core.MaterializeResult{}, err
	}
	if engine != "postgres" {
		return core.MaterializeResult{}, fmt.Errorf("materialization target must currently be PostgreSQL; got %s", engineLabel(engine))
	}
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	tx, err := db.BeginTx(queryCtx, nil)
	if err != nil {
		return core.MaterializeResult{}, err
	}
	defer tx.Rollback()

	schemaQ := queryir.Quote(spec.Schema)
	tableQ := queryir.Quote(spec.Table)
	tmpQ := queryir.Quote(tmpName)
	target := schemaQ + "." + tableQ
	tmp := schemaQ + "." + tmpQ

	if _, err := tx.ExecContext(queryCtx, "CREATE SCHEMA IF NOT EXISTS "+schemaQ); err != nil {
		return core.MaterializeResult{}, fmt.Errorf("create schema: %w", err)
	}
	if _, err := tx.ExecContext(queryCtx, "DROP TABLE IF EXISTS "+tmp); err != nil {
		return core.MaterializeResult{}, err
	}
	if _, err := tx.ExecContext(queryCtx, "CREATE TABLE "+tmp+" AS "+sqlText, spec.Args...); err != nil {
		return core.MaterializeResult{}, fmt.Errorf("create table as select: %w", err)
	}

	var exists int
	_ = tx.QueryRowContext(queryCtx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = $1 AND table_name = $2`, spec.Schema, spec.Table).Scan(&exists)

	switch strings.ToLower(spec.Strategy) {
	case "create_only":
		if exists > 0 {
			if _, err := tx.ExecContext(queryCtx, "DROP TABLE "+tmp); err != nil {
				return core.MaterializeResult{}, err
			}
		} else if _, err := tx.ExecContext(queryCtx, "ALTER TABLE "+tmp+" RENAME TO "+tableQ); err != nil {
			return core.MaterializeResult{}, err
		}
	case "truncate_insert":
		if exists > 0 {
			if _, err := tx.ExecContext(queryCtx, "TRUNCATE TABLE "+target); err != nil {
				return core.MaterializeResult{}, err
			}
			if _, err := tx.ExecContext(queryCtx, "INSERT INTO "+target+" SELECT * FROM "+tmp); err != nil {
				return core.MaterializeResult{}, err
			}
			if _, err := tx.ExecContext(queryCtx, "DROP TABLE "+tmp); err != nil {
				return core.MaterializeResult{}, err
			}
		} else if _, err := tx.ExecContext(queryCtx, "ALTER TABLE "+tmp+" RENAME TO "+tableQ); err != nil {
			return core.MaterializeResult{}, err
		}
	case "incremental":
		if exists > 0 {
			if _, err := tx.ExecContext(queryCtx, "INSERT INTO "+target+" SELECT * FROM "+tmp); err != nil {
				return core.MaterializeResult{}, err
			}
			if _, err := tx.ExecContext(queryCtx, "DROP TABLE "+tmp); err != nil {
				return core.MaterializeResult{}, err
			}
		} else if _, err := tx.ExecContext(queryCtx, "ALTER TABLE "+tmp+" RENAME TO "+tableQ); err != nil {
			return core.MaterializeResult{}, err
		}
	default:
		if _, err := tx.ExecContext(queryCtx, "DROP TABLE IF EXISTS "+target); err != nil {
			return core.MaterializeResult{}, err
		}
		if _, err := tx.ExecContext(queryCtx, "ALTER TABLE "+tmp+" RENAME TO "+tableQ); err != nil {
			return core.MaterializeResult{}, err
		}
	}

	var count int
	if err := tx.QueryRowContext(queryCtx, "SELECT COUNT(*) FROM "+target).Scan(&count); err != nil {
		return core.MaterializeResult{}, err
	}
	watermark := ""
	if spec.WatermarkField != "" {
		if err := queryir.CheckIdent("watermark", spec.WatermarkField); err != nil {
			return core.MaterializeResult{}, err
		}
		var value any
		if err := tx.QueryRowContext(queryCtx, "SELECT MAX("+queryir.Quote(spec.WatermarkField)+") FROM "+target).Scan(&value); err == nil && value != nil {
			watermark = fmt.Sprint(value)
		}
	}
	if err := tx.Commit(); err != nil {
		return core.MaterializeResult{}, err
	}
	return core.MaterializeResult{RowCount: count, Watermark: watermark}, nil
}
