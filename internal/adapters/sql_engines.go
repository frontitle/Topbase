package adapters

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	mysql "github.com/go-sql-driver/mysql"
	_ "github.com/microsoft/go-mssqldb"
	_ "github.com/sijms/go-ora/v2"
	"github.com/topbase/topbase/internal/core"
	"github.com/topbase/topbase/internal/core/queryir"
	_ "modernc.org/sqlite"
)

type EngineDefinition struct {
	ID              string   `json:"id"`
	Label           string   `json:"label"`
	Family          string   `json:"family"`
	DefaultPort     int      `json:"default_port,omitempty"`
	Network         bool     `json:"network"`
	SSH             bool     `json:"ssh"`
	Username        bool     `json:"username"`
	DefaultUsername string   `json:"default_username,omitempty"`
	DSNExample      string   `json:"dsn_example"`
	Compatible      []string `json:"compatible,omitempty"`
}

func EngineDefinitions() []EngineDefinition {
	return []EngineDefinition{
		{ID: "postgres", Label: "PostgreSQL", Family: "postgres", DefaultPort: 5432, Network: true, SSH: true, Username: true, DSNExample: "postgres://user:password@host:5432/database?sslmode=require"},
		{ID: "mysql", Label: "MySQL / MariaDB", Family: "mysql", DefaultPort: 3306, Network: true, SSH: true, Username: true, DSNExample: "user:password@tcp(host:3306)/database?parseTime=true", Compatible: []string{"MariaDB", "TiDB", "OceanBase MySQL", "Apache Doris", "StarRocks"}},
		{ID: "clickhouse", Label: "ClickHouse", Family: "clickhouse", DefaultPort: 9000, Network: true, SSH: true, Username: false, DefaultUsername: "default", DSNExample: "clickhouse://user:password@host:9000/database?secure=true"},
		{ID: "sqlserver", Label: "SQL Server", Family: "sqlserver", DefaultPort: 1433, Network: true, SSH: true, Username: true, DSNExample: "sqlserver://user:password@host:1433?database=database&encrypt=true"},
		{ID: "oracle", Label: "Oracle Database", Family: "oracle", DefaultPort: 1521, Network: true, SSH: true, Username: true, DSNExample: "oracle://user:password@host:1521/service_name"},
		{ID: "sqlite", Label: "SQLite", Family: "sqlite", Network: false, SSH: false, Username: false, DSNExample: "/data/analytics.db"},
	}
}

type preparedSQLConnection struct {
	engine      string
	driver      string
	dsn         string
	displayHost string
	tunnel      *sshTunnel
}

func normalizeEngine(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "postgres", "postgresql", "pg":
		return "postgres", nil
	case "mysql", "mariadb", "tidb", "oceanbase", "doris", "starrocks":
		return "mysql", nil
	case "clickhouse", "ch":
		return "clickhouse", nil
	case "sqlserver", "mssql", "sql-server":
		return "sqlserver", nil
	case "oracle", "ora":
		return "oracle", nil
	case "sqlite", "sqlite3":
		return "sqlite", nil
	default:
		return "", fmt.Errorf("unsupported database engine %q", value)
	}
}

func engineLabel(engine string) string {
	switch engine {
	case "postgres":
		return "PostgreSQL"
	case "mysql":
		return "MySQL / MariaDB"
	case "clickhouse":
		return "ClickHouse"
	case "sqlserver":
		return "SQL Server"
	case "oracle":
		return "Oracle Database"
	case "sqlite":
		return "SQLite"
	default:
		return engine
	}
}

func enginePrefix(engine string) string {
	switch engine {
	case "postgres":
		return "pg"
	case "mysql":
		return "my"
	case "clickhouse":
		return "ch"
	case "sqlserver":
		return "ms"
	case "oracle":
		return "or"
	case "sqlite":
		return "sq"
	default:
		return "db"
	}
}

func defaultPort(engine string) int {
	switch engine {
	case "postgres":
		return 5432
	case "mysql":
		return 3306
	case "clickhouse":
		return 9000
	case "sqlserver":
		return 1433
	case "oracle":
		return 1521
	default:
		return 0
	}
}

func prepareSQLConnection(ctx context.Context, input core.ConnectionRequest) (preparedSQLConnection, error) {
	engine, err := normalizeEngine(input.Engine)
	if err != nil {
		return preparedSQLConnection{}, err
	}
	if engine == "sqlite" {
		if input.SSH != nil {
			return preparedSQLConnection{}, fmt.Errorf("SQLite is a local file database and does not support SSH tunneling")
		}
		path := strings.TrimSpace(input.DSN)
		if path == "" {
			path = strings.TrimSpace(input.Database)
		}
		if path == "" {
			return preparedSQLConnection{}, fmt.Errorf("SQLite database file is required")
		}
		if !strings.HasPrefix(path, "file:") && path != ":memory:" {
			absolute, err := filepath.Abs(path)
			if err != nil {
				return preparedSQLConnection{}, fmt.Errorf("resolve SQLite file: %w", err)
			}
			path = absolute
		}
		return preparedSQLConnection{engine: engine, driver: "sqlite", dsn: path, displayHost: path}, nil
	}

	if strings.TrimSpace(input.DSN) != "" {
		return prepareDSNConnection(ctx, engine, input)
	}
	if strings.TrimSpace(input.Host) == "" || strings.TrimSpace(input.Database) == "" {
		return preparedSQLConnection{}, fmt.Errorf("host and database name are required")
	}
	if engine != "clickhouse" && strings.TrimSpace(input.Username) == "" {
		return preparedSQLConnection{}, fmt.Errorf("username is required")
	}
	port := input.Port
	if port == 0 {
		port = defaultPort(engine)
	}
	displayHost := net.JoinHostPort(input.Host, strconv.Itoa(port))
	connectHost, connectPort := input.Host, port
	var tunnel *sshTunnel
	if input.SSH != nil {
		tunnel, local, err := openSSHTunnel(ctx, *input.SSH, input.Host, port)
		if err != nil {
			return preparedSQLConnection{}, err
		}
		connectHost, connectPort, err = splitAddress(local, port)
		if err != nil {
			_ = tunnel.Close()
			return preparedSQLConnection{}, err
		}
	}
	dsn, driver, err := buildDSN(engine, input, connectHost, connectPort)
	if err != nil {
		if tunnel != nil {
			_ = tunnel.Close()
		}
		return preparedSQLConnection{}, err
	}
	return preparedSQLConnection{engine: engine, driver: driver, dsn: dsn, displayHost: displayHost, tunnel: tunnel}, nil
}

func prepareDSNConnection(ctx context.Context, engine string, input core.ConnectionRequest) (preparedSQLConnection, error) {
	dsn := strings.TrimSpace(input.DSN)
	var driver, displayHost, targetHost string
	var targetPort int
	switch engine {
	case "postgres", "clickhouse", "sqlserver", "oracle":
		u, err := url.Parse(dsn)
		if err != nil || u.Hostname() == "" {
			return preparedSQLConnection{}, fmt.Errorf("invalid %s connection string", engineLabel(engine))
		}
		targetHost = u.Hostname()
		targetPort, _ = strconv.Atoi(u.Port())
		if targetPort == 0 {
			targetPort = defaultPort(engine)
		}
		displayHost = net.JoinHostPort(targetHost, strconv.Itoa(targetPort))
		if input.SSH != nil {
			tunnel, local, err := openSSHTunnel(ctx, *input.SSH, targetHost, targetPort)
			if err != nil {
				return preparedSQLConnection{}, err
			}
			u.Host = local
			dsn = u.String()
			driver = driverName(engine)
			return preparedSQLConnection{engine: engine, driver: driver, dsn: dsn, displayHost: displayHost, tunnel: tunnel}, nil
		}
	case "mysql":
		config, err := mysql.ParseDSN(dsn)
		if err != nil {
			return preparedSQLConnection{}, fmt.Errorf("invalid MySQL connection string: %w", err)
		}
		targetHost, targetPort, err = splitAddress(config.Addr, 3306)
		if err != nil {
			return preparedSQLConnection{}, err
		}
		displayHost = net.JoinHostPort(targetHost, strconv.Itoa(targetPort))
		if input.SSH != nil {
			tunnel, local, err := openSSHTunnel(ctx, *input.SSH, targetHost, targetPort)
			if err != nil {
				return preparedSQLConnection{}, err
			}
			config.Net = "tcp"
			config.Addr = local
			dsn = config.FormatDSN()
			return preparedSQLConnection{engine: engine, driver: "mysql", dsn: dsn, displayHost: displayHost, tunnel: tunnel}, nil
		}
	}
	return preparedSQLConnection{engine: engine, driver: driverName(engine), dsn: dsn, displayHost: displayHost}, nil
}

func driverName(engine string) string {
	switch engine {
	case "postgres":
		return "pgx"
	case "mysql":
		return "mysql"
	case "clickhouse":
		return "clickhouse"
	case "sqlserver":
		return "sqlserver"
	case "oracle":
		return "oracle"
	case "sqlite":
		return "sqlite"
	default:
		return ""
	}
}

func buildDSN(engine string, input core.ConnectionRequest, host string, port int) (string, string, error) {
	sslMode := strings.ToLower(strings.TrimSpace(input.SSLMode))
	switch engine {
	case "postgres":
		copyInput := input
		copyInput.Host, copyInput.Port, copyInput.DSN = host, port, ""
		dsn, err := postgresDSN(copyInput)
		return dsn, "pgx", err
	case "mysql":
		config := mysql.NewConfig()
		config.User = input.Username
		config.Passwd = input.Password
		config.Net = "tcp"
		config.Addr = net.JoinHostPort(host, strconv.Itoa(port))
		config.DBName = input.Database
		config.ParseTime = true
		config.Collation = "utf8mb4_unicode_ci"
		switch sslMode {
		case "require", "verify-full", "true":
			config.TLSConfig = "true"
		case "prefer", "preferred":
			config.TLSConfig = "preferred"
		case "skip-verify":
			config.TLSConfig = "skip-verify"
		}
		return config.FormatDSN(), "mysql", nil
	case "clickhouse":
		username := input.Username
		if username == "" {
			username = "default"
		}
		u := &url.URL{Scheme: "clickhouse", User: url.UserPassword(username, input.Password), Host: net.JoinHostPort(host, strconv.Itoa(port)), Path: "/" + input.Database}
		if sslMode == "require" || sslMode == "verify-full" || sslMode == "true" {
			q := u.Query()
			q.Set("secure", "true")
			u.RawQuery = q.Encode()
		}
		return u.String(), "clickhouse", nil
	case "sqlserver":
		u := &url.URL{Scheme: "sqlserver", User: url.UserPassword(input.Username, input.Password), Host: net.JoinHostPort(host, strconv.Itoa(port))}
		q := u.Query()
		q.Set("database", input.Database)
		if sslMode == "disable" {
			q.Set("encrypt", "disable")
		} else {
			q.Set("encrypt", "true")
			if sslMode == "prefer" || sslMode == "skip-verify" {
				q.Set("TrustServerCertificate", "true")
			}
		}
		u.RawQuery = q.Encode()
		return u.String(), "sqlserver", nil
	case "oracle":
		u := &url.URL{Scheme: "oracle", User: url.UserPassword(input.Username, input.Password), Host: net.JoinHostPort(host, strconv.Itoa(port)), Path: "/" + input.Database}
		return u.String(), "oracle", nil
	default:
		return "", "", fmt.Errorf("unsupported database engine %q", engine)
	}
}

func splitAddress(address string, fallbackPort int) (string, int, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return "", 0, fmt.Errorf("database address is required")
	}
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		if strings.Contains(err.Error(), "missing port") {
			return address, fallbackPort, nil
		}
		return "", 0, fmt.Errorf("invalid database address %q: %w", address, err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		return "", 0, fmt.Errorf("invalid database port %q", portText)
	}
	return host, port, nil
}

func executeForEngine(ctx context.Context, connection databaseConnection, statement string, args ...any) (core.QueryResult, error) {
	statement = rebindSQL(statement, connection.engine)
	if connection.engine == "postgres" {
		return executePostgres(ctx, connection.db, statement, args...)
	}
	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	rows, err := connection.db.QueryContext(queryCtx, statement, args...)
	if err != nil {
		return core.QueryResult{}, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return core.QueryResult{}, err
	}
	result := core.QueryResult{Columns: columns, Rows: make([][]any, 0), Meta: map[string]any{"row_limit": 1000, "engine": connection.engine}}
	for rows.Next() {
		if len(result.Rows) == 1000 {
			break
		}
		values := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return core.QueryResult{}, err
		}
		for i, value := range values {
			switch item := value.(type) {
			case []byte:
				values[i] = string(item)
			case time.Time:
				values[i] = item.Format(time.RFC3339Nano)
			}
		}
		result.Rows = append(result.Rows, values)
	}
	if err := rows.Err(); err != nil {
		return core.QueryResult{}, err
	}
	return result, nil
}

var postgresParameter = regexp.MustCompile(`\$([0-9]+)`)

func rebindSQL(statement, engine string) string {
	switch engine {
	case "mysql", "clickhouse", "sqlite":
		return postgresParameter.ReplaceAllString(statement, "?")
	case "sqlserver":
		return postgresParameter.ReplaceAllString(statement, "@p$1")
	case "oracle":
		return postgresParameter.ReplaceAllString(statement, ":$1")
	default:
		return statement
	}
}

func scanTablesForEngine(ctx context.Context, connection databaseConnection) ([]core.Table, error) {
	switch connection.engine {
	case "postgres":
		return scanPostgresTables(ctx, connection.db)
	case "mysql":
		return scanCatalogRows(ctx, connection.db, mysqlCatalogQuery)
	case "clickhouse":
		return scanCatalogRows(ctx, connection.db, clickHouseCatalogQuery)
	case "sqlserver":
		return scanCatalogRows(ctx, connection.db, sqlServerCatalogQuery)
	case "oracle":
		return scanCatalogRows(ctx, connection.db, oracleCatalogQuery)
	case "sqlite":
		return scanSQLiteTables(ctx, connection.db)
	default:
		return nil, fmt.Errorf("metadata sync is not implemented for %s", connection.engine)
	}
}

func scanCatalogRows(ctx context.Context, db *sql.DB, statement string) ([]core.Table, error) {
	queryCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	rows, err := db.QueryContext(queryCtx, statement)
	if err != nil {
		return nil, fmt.Errorf("scan tables: %w", err)
	}
	defer rows.Close()
	byKey := map[string]*core.Table{}
	keys := []string{}
	for rows.Next() {
		var schema, tableName string
		var tableDescription, column, dataType, columnDescription, defaultValue sql.NullString
		var nullable, primaryKey sql.NullBool
		var foreignSchema, foreignTable, foreignColumn sql.NullString
		if err := rows.Scan(&schema, &tableName, &tableDescription, &column, &dataType, &nullable, &columnDescription, &defaultValue, &primaryKey, &foreignSchema, &foreignTable, &foreignColumn); err != nil {
			return nil, err
		}
		key := schema + "." + tableName
		item := byKey[key]
		if item == nil {
			item = &core.Table{Schema: schema, Name: tableName, Description: tableDescription.String}
			byKey[key] = item
			keys = append(keys, key)
		}
		if column.Valid {
			field := core.Column{Name: column.String, DataType: dataType.String, Nullable: nullable.Bool, Description: columnDescription.String, DefaultValue: defaultValue.String, PrimaryKey: primaryKey.Bool}
			if foreignTable.Valid && foreignColumn.Valid {
				field.ForeignKey = &core.FieldRef{Schema: foreignSchema.String, Table: foreignTable.String, Name: foreignColumn.String}
			}
			item.Columns = append(item.Columns, field)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	items := make([]core.Table, 0, len(keys))
	for _, key := range keys {
		items = append(items, *byKey[key])
	}
	return items, nil
}

const mysqlCatalogQuery = `
SELECT t.TABLE_SCHEMA,
       t.TABLE_NAME,
       NULLIF(t.TABLE_COMMENT, ''),
       c.COLUMN_NAME,
       c.COLUMN_TYPE,
       c.IS_NULLABLE = 'YES',
       NULLIF(c.COLUMN_COMMENT, ''),
       c.COLUMN_DEFAULT,
       c.COLUMN_KEY = 'PRI',
       k.REFERENCED_TABLE_SCHEMA,
       k.REFERENCED_TABLE_NAME,
       k.REFERENCED_COLUMN_NAME
FROM information_schema.TABLES t
LEFT JOIN information_schema.COLUMNS c
  ON c.TABLE_SCHEMA = t.TABLE_SCHEMA AND c.TABLE_NAME = t.TABLE_NAME
LEFT JOIN information_schema.KEY_COLUMN_USAGE k
  ON k.TABLE_SCHEMA = c.TABLE_SCHEMA AND k.TABLE_NAME = c.TABLE_NAME
 AND k.COLUMN_NAME = c.COLUMN_NAME AND k.REFERENCED_TABLE_NAME IS NOT NULL
WHERE t.TABLE_SCHEMA = DATABASE()
  AND t.TABLE_TYPE IN ('BASE TABLE', 'VIEW', 'SYSTEM VIEW')
ORDER BY t.TABLE_SCHEMA, t.TABLE_NAME, c.ORDINAL_POSITION`

const clickHouseCatalogQuery = `
SELECT t.database,
       t.name,
       nullIf(t.comment, ''),
       c.name,
       c.type,
       startsWith(c.type, 'Nullable('),
       nullIf(c.comment, ''),
       nullIf(c.default_expression, ''),
       c.is_in_primary_key,
       NULL,
       NULL,
       NULL
FROM system.tables t
LEFT JOIN system.columns c ON c.database = t.database AND c.table = t.name
WHERE t.database = currentDatabase()
  AND NOT startsWith(t.name, '.inner_id.')
ORDER BY t.database, t.name, c.position`

const sqlServerCatalogQuery = `
SELECT s.name,
       t.name,
       CAST(table_description.value AS nvarchar(max)),
       c.name,
       TYPE_NAME(c.user_type_id),
       c.is_nullable,
       CAST(column_description.value AS nvarchar(max)),
       OBJECT_DEFINITION(c.default_object_id),
       CAST(CASE WHEN pk.column_id IS NULL THEN 0 ELSE 1 END AS bit),
       foreign_schema.name,
       foreign_table.name,
       foreign_column.name
FROM sys.tables t
JOIN sys.schemas s ON s.schema_id = t.schema_id
LEFT JOIN sys.columns c ON c.object_id = t.object_id
LEFT JOIN sys.extended_properties table_description
  ON table_description.major_id = t.object_id AND table_description.minor_id = 0 AND table_description.name = 'MS_Description'
LEFT JOIN sys.extended_properties column_description
  ON column_description.major_id = t.object_id AND column_description.minor_id = c.column_id AND column_description.name = 'MS_Description'
LEFT JOIN (
  SELECT ic.object_id, ic.column_id
  FROM sys.indexes i
  JOIN sys.index_columns ic ON ic.object_id = i.object_id AND ic.index_id = i.index_id
  WHERE i.is_primary_key = 1
) pk ON pk.object_id = c.object_id AND pk.column_id = c.column_id
LEFT JOIN sys.foreign_key_columns fkc ON fkc.parent_object_id = c.object_id AND fkc.parent_column_id = c.column_id
LEFT JOIN sys.tables foreign_table ON foreign_table.object_id = fkc.referenced_object_id
LEFT JOIN sys.schemas foreign_schema ON foreign_schema.schema_id = foreign_table.schema_id
LEFT JOIN sys.columns foreign_column ON foreign_column.object_id = fkc.referenced_object_id AND foreign_column.column_id = fkc.referenced_column_id
WHERE t.is_ms_shipped = 0
ORDER BY s.name, t.name, c.column_id`

const oracleCatalogQuery = `
SELECT SYS_CONTEXT('USERENV', 'CURRENT_SCHEMA'),
       objects.object_name,
       table_comments.comments,
       columns.column_name,
       columns.data_type,
       CASE WHEN columns.nullable = 'Y' THEN 1 ELSE 0 END,
       column_comments.comments,
       columns.data_default,
       CASE WHEN primary_columns.column_name IS NULL THEN 0 ELSE 1 END,
       referenced_constraints.owner,
       referenced_constraints.table_name,
       referenced_columns.column_name
FROM user_objects objects
LEFT JOIN user_tab_columns columns ON columns.table_name = objects.object_name
LEFT JOIN user_tab_comments table_comments ON table_comments.table_name = objects.object_name
LEFT JOIN user_col_comments column_comments ON column_comments.table_name = columns.table_name AND column_comments.column_name = columns.column_name
LEFT JOIN (
  SELECT constraint_columns.table_name, constraint_columns.column_name
  FROM user_constraints constraints
  JOIN user_cons_columns constraint_columns ON constraint_columns.constraint_name = constraints.constraint_name
  WHERE constraints.constraint_type = 'P'
) primary_columns ON primary_columns.table_name = columns.table_name AND primary_columns.column_name = columns.column_name
LEFT JOIN user_cons_columns source_columns ON source_columns.table_name = columns.table_name AND source_columns.column_name = columns.column_name
LEFT JOIN user_constraints source_constraints ON source_constraints.constraint_name = source_columns.constraint_name AND source_constraints.constraint_type = 'R'
LEFT JOIN all_constraints referenced_constraints ON referenced_constraints.constraint_name = source_constraints.r_constraint_name AND referenced_constraints.owner = source_constraints.r_owner
LEFT JOIN all_cons_columns referenced_columns ON referenced_columns.constraint_name = referenced_constraints.constraint_name AND referenced_columns.owner = referenced_constraints.owner AND referenced_columns.position = source_columns.position
WHERE objects.object_type IN ('TABLE', 'VIEW', 'MATERIALIZED VIEW')
ORDER BY objects.object_name, columns.column_id`

func scanSQLiteTables(ctx context.Context, db *sql.DB) ([]core.Table, error) {
	queryCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	rows, err := db.QueryContext(queryCtx, `SELECT name FROM sqlite_master WHERE type IN ('table','view') AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("scan SQLite tables: %w", err)
	}
	names := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return nil, err
		}
		names = append(names, name)
	}
	rows.Close()
	items := make([]core.Table, 0, len(names))
	for _, name := range names {
		foreignKeys, err := sqliteForeignKeys(queryCtx, db, name)
		if err != nil {
			return nil, err
		}
		columns, err := db.QueryContext(queryCtx, "PRAGMA table_xinfo("+queryir.Quote(name)+")")
		if err != nil {
			return nil, err
		}
		item := core.Table{Schema: "main", Name: name}
		for columns.Next() {
			var cid, notNull, primaryKey, hidden int
			var column, dataType string
			var defaultValue sql.NullString
			if err := columns.Scan(&cid, &column, &dataType, &notNull, &defaultValue, &primaryKey, &hidden); err != nil {
				columns.Close()
				return nil, err
			}
			if hidden != 0 {
				continue
			}
			item.Columns = append(item.Columns, core.Column{Name: column, DataType: dataType, Nullable: notNull == 0 && primaryKey == 0, DefaultValue: defaultValue.String, PrimaryKey: primaryKey > 0, ForeignKey: foreignKeys[column]})
		}
		columns.Close()
		items = append(items, item)
	}
	return items, nil
}

func sqliteForeignKeys(ctx context.Context, db *sql.DB, table string) (map[string]*core.FieldRef, error) {
	rows, err := db.QueryContext(ctx, "PRAGMA foreign_key_list("+queryir.Quote(table)+")")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := map[string]*core.FieldRef{}
	for rows.Next() {
		var id, seq int
		var targetTable, from, to, onUpdate, onDelete, match string
		if err := rows.Scan(&id, &seq, &targetTable, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			return nil, err
		}
		items[from] = &core.FieldRef{Schema: "main", Table: targetTable, Name: to}
	}
	return items, rows.Err()
}
