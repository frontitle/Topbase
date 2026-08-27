package adapters

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/topbase/topbase/internal/core/queryir"
)

var (
	dateTruncCall = regexp.MustCompile(`date_trunc\('(minute|hour|day|week|month|quarter|year)', ([^)]+)\)`)
	limitSuffix   = regexp.MustCompile(` LIMIT ([0-9]+)$`)
	doubleQuoted  = regexp.MustCompile(`"([A-Za-z_][A-Za-z0-9_$]*)"`)
	middleLike    = regexp.MustCompile(`'%' \|\| (\$[0-9]+) \|\| '%'`)
	startLike     = regexp.MustCompile(`(\$[0-9]+) \|\| '%'`)
	endLike       = regexp.MustCompile(`'%' \|\| (\$[0-9]+)`)
	stddevCall    = regexp.MustCompile(`stddev_samp\(([^)]+)\)`)
	varianceCall  = regexp.MustCompile(`var_samp\(([^)]+)\)`)
)

func (p *SQLConnector) Compile(q queryir.Query) (queryir.Compiled, error) {
	engine, err := p.Engine(q.Source.DatabaseID)
	if err != nil {
		return queryir.Compiled{}, err
	}
	return CompileForEngine(engine, q)
}

func CompileForEngine(engine string, q queryir.Query) (queryir.Compiled, error) {
	engine, err := normalizeEngine(engine)
	if err != nil {
		return queryir.Compiled{}, err
	}
	compiled, err := CompilePostgres(q)
	if err != nil {
		return queryir.Compiled{}, err
	}
	compiled.SQL = rewriteGeneratedSQL(compiled.SQL, engine)
	return compiled, nil
}

func rewriteGeneratedSQL(statement, engine string) string {
	if engine == "postgres" {
		return statement
	}
	statement = dateTruncCall.ReplaceAllStringFunc(statement, func(call string) string {
		matches := dateTruncCall.FindStringSubmatch(call)
		unit, field := matches[1], matches[2]
		switch engine {
		case "mysql":
			switch unit {
			case "minute":
				return "DATE_FORMAT(" + field + ", '%Y-%m-%d %H:%i:00')"
			case "hour":
				return "DATE_FORMAT(" + field + ", '%Y-%m-%d %H:00:00')"
			case "day":
				return "DATE(" + field + ")"
			case "week":
				return "DATE_SUB(DATE(" + field + "), INTERVAL WEEKDAY(" + field + ") DAY)"
			case "month":
				return "DATE_FORMAT(" + field + ", '%Y-%m-01')"
			case "quarter":
				return "CONCAT(YEAR(" + field + "), '-Q', QUARTER(" + field + "))"
			case "year":
				return "YEAR(" + field + ")"
			}
		case "clickhouse":
			return "dateTrunc('" + unit + "', " + field + ")"
		case "sqlserver":
			return "DATEADD(" + unit + ", DATEDIFF(" + unit + ", 0, " + field + "), 0)"
		case "oracle":
			formats := map[string]string{"minute": "MI", "hour": "HH24", "day": "DD", "week": "IW", "month": "MM", "quarter": "Q", "year": "YYYY"}
			return "TRUNC(" + field + ", '" + formats[unit] + "')"
		case "sqlite":
			formats := map[string]string{
				"minute": "%Y-%m-%d %H:%M:00", "hour": "%Y-%m-%d %H:00:00", "day": "%Y-%m-%d",
				"week": "%Y-%W", "month": "%Y-%m-01", "quarter": "%Y-Q", "year": "%Y-01-01",
			}
			if unit == "quarter" {
				return "strftime('%Y', " + field + ") || '-Q' || ((CAST(strftime('%m', " + field + ") AS INTEGER) + 2) / 3)"
			}
			return "strftime('" + formats[unit] + "', " + field + ")"
		}
		return call
	})

	switch engine {
	case "mysql":
		statement = middleLike.ReplaceAllString(statement, "CONCAT('%', $1, '%')")
		statement = startLike.ReplaceAllString(statement, "CONCAT($1, '%')")
		statement = endLike.ReplaceAllString(statement, "CONCAT('%', $1)")
		statement = strings.ReplaceAll(statement, " ILIKE ", " LIKE ")
		statement = doubleQuoted.ReplaceAllString(statement, "`$1`")
	case "sqlserver":
		statement = strings.ReplaceAll(statement, " ILIKE ", " LIKE ")
		statement = strings.ReplaceAll(statement, " || ", " + ")
		if match := limitSuffix.FindStringSubmatch(statement); len(match) == 2 {
			statement = limitSuffix.ReplaceAllString(statement, "")
			statement = strings.Replace(statement, "SELECT ", "SELECT TOP ("+match[1]+") ", 1)
		}
		statement = strings.ReplaceAll(statement, "stddev_samp(", "STDEV(")
		statement = strings.ReplaceAll(statement, "var_samp(", "VAR(")
	case "clickhouse":
		statement = strings.ReplaceAll(statement, "stddev_samp(", "stddevSamp(")
		statement = strings.ReplaceAll(statement, "var_samp(", "varSamp(")
	case "oracle":
		statement = strings.ReplaceAll(statement, " ILIKE ", " LIKE ")
		if match := limitSuffix.FindStringSubmatch(statement); len(match) == 2 {
			statement = limitSuffix.ReplaceAllString(statement, " FETCH FIRST "+match[1]+" ROWS ONLY")
		}
	case "sqlite":
		statement = strings.ReplaceAll(statement, " ILIKE ", " LIKE ")
		statement = stddevCall.ReplaceAllString(statement, "sqrt(avg(($1) * ($1)) - avg($1) * avg($1))")
		statement = varianceCall.ReplaceAllString(statement, "(avg(($1) * ($1)) - avg($1) * avg($1))")
	}
	return statement
}

func (p *SQLConnector) CompileWarehouse(q queryir.Query) (queryir.Compiled, error) {
	engine, err := p.Engine(q.Source.DatabaseID)
	if err != nil {
		return queryir.Compiled{}, err
	}
	if engine != "postgres" {
		return queryir.Compiled{}, fmt.Errorf("materialization currently requires a PostgreSQL target; %s remains fully available for interactive analysis", engineLabel(engine))
	}
	return CompilePostgresWarehouse(q)
}
