package adapters

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/topbase/topbase/internal/core"
)

type SQLConnector struct {
	mu        sync.RWMutex
	databases map[string]databaseConnection
	reconnect sync.Map
}

type databaseConnection struct {
	db     *sql.DB
	tunnel *sshTunnel
	input  core.ConnectionRequest
	engine string
}

func NewSQLConnector() *SQLConnector {
	return &SQLConnector{databases: map[string]databaseConnection{}}
}

// NewPostgresConnector remains for source compatibility with earlier Topbase
// integrations. New code should use NewSQLConnector.
func NewPostgresConnector() *SQLConnector { return NewSQLConnector() }

func (p *SQLConnector) Connect(ctx context.Context, input core.ConnectionRequest) (core.Database, error) {
	if input.ID == "" {
		return p.connect(ctx, input)
	}
	lock := p.connectionLock(input.ID)
	lock.Lock()
	defer lock.Unlock()
	return p.connect(ctx, input)
}

func (p *SQLConnector) connect(ctx context.Context, input core.ConnectionRequest) (core.Database, error) {
	if strings.TrimSpace(input.Name) == "" {
		return core.Database{}, fmt.Errorf("connection name is required")
	}
	prepared, err := prepareSQLConnection(ctx, input)
	if err != nil {
		return core.Database{}, err
	}
	db, err := sql.Open(prepared.driver, prepared.dsn)
	if err != nil {
		if prepared.tunnel != nil {
			_ = prepared.tunnel.Close()
		}
		return core.Database{}, fmt.Errorf("open %s: %w", engineLabel(prepared.engine), err)
	}
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(30 * time.Minute)
	pingCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		if prepared.tunnel != nil {
			_ = prepared.tunnel.Close()
		}
		return core.Database{}, fmt.Errorf("connect %s: %w", engineLabel(prepared.engine), err)
	}
	id := input.ID
	if id == "" {
		id = enginePrefix(prepared.engine) + "_" + fmt.Sprintf("%x", time.Now().UnixNano())
	}
	input.Engine = prepared.engine
	input.ID = id
	database := core.Database{ID: id, Name: input.Name, Engine: prepared.engine, Host: prepared.displayHost, Status: "connected", CreatedAt: time.Now().UTC(), Connected: true}
	p.mu.Lock()
	old := p.databases[id]
	p.databases[id] = databaseConnection{db: db, tunnel: prepared.tunnel, input: input, engine: prepared.engine}
	p.mu.Unlock()
	if old.db != nil && old.db != db {
		_ = old.db.Close()
	}
	if old.tunnel != nil {
		_ = old.tunnel.Close()
	}
	return database, nil
}

func withSSHTunnel(ctx context.Context, dsn string, sshConfig core.SSHTunnelRequest) (string, *sshTunnel, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", nil, fmt.Errorf("parse PostgreSQL connection string: %w", err)
	}
	host := u.Hostname()
	port, err := strconv.Atoi(u.Port())
	if err != nil || port == 0 {
		port = 5432
	}
	tunnel, localAddress, err := openSSHTunnel(ctx, sshConfig, host, port)
	if err != nil {
		return "", nil, err
	}
	u.Host = localAddress
	return u.String(), tunnel, nil
}

func postgresDSN(input core.ConnectionRequest) (string, error) {
	if strings.TrimSpace(input.DSN) != "" {
		return input.DSN, nil
	}
	if strings.TrimSpace(input.Host) == "" || strings.TrimSpace(input.Database) == "" || strings.TrimSpace(input.Username) == "" {
		return "", fmt.Errorf("host, database name, and username are required")
	}
	port := input.Port
	if port == 0 {
		port = 5432
	}
	sslMode := input.SSLMode
	if sslMode == "" {
		sslMode = "prefer"
	}
	u := &url.URL{Scheme: "postgres", User: url.UserPassword(input.Username, input.Password), Host: fmt.Sprintf("%s:%d", input.Host, port), Path: input.Database}
	query := u.Query()
	query.Set("sslmode", sslMode)
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func (p *SQLConnector) db(databaseID string) (*sql.DB, error) {
	connection, err := p.connection(databaseID)
	if err != nil {
		return nil, err
	}
	return connection.db, nil
}

func (p *SQLConnector) connection(databaseID string) (databaseConnection, error) {
	p.mu.RLock()
	connection, ok := p.databases[databaseID]
	p.mu.RUnlock()
	if !ok {
		return databaseConnection{}, fmt.Errorf("database %q is not connected in this process", databaseID)
	}
	return connection, nil
}

func (p *SQLConnector) Connected(id string) bool {
	p.mu.RLock()
	connection, ok := p.databases[id]
	p.mu.RUnlock()
	if !ok || connection.db == nil {
		return false
	}
	return connection.tunnel == nil || connection.tunnel.Alive()
}

func (p *SQLConnector) Execute(ctx context.Context, databaseID, statement string, args ...any) (core.QueryResult, error) {
	connection, err := p.connection(databaseID)
	if err != nil {
		return core.QueryResult{}, err
	}
	result, err := executeForEngine(ctx, connection, statement, args...)
	if err == nil || !recoverableConnectionError(err) {
		return result, err
	}
	if reconnectErr := p.reconnectDatabase(ctx, databaseID, connection); reconnectErr != nil {
		return core.QueryResult{}, fmt.Errorf("%w; automatic reconnect failed: %v", err, reconnectErr)
	}
	connection, err = p.connection(databaseID)
	if err != nil {
		return core.QueryResult{}, err
	}
	result, err = executeForEngine(ctx, connection, statement, args...)
	if err == nil {
		if result.Meta == nil {
			result.Meta = map[string]any{}
		}
		result.Meta["reconnected"] = true
	}
	return result, err
}

func executePostgres(ctx context.Context, connectionDB *sql.DB, statement string, args ...any) (core.QueryResult, error) {
	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	tx, err := connectionDB.BeginTx(queryCtx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return core.QueryResult{}, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(queryCtx, statement, args...)
	if err != nil {
		return core.QueryResult{}, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return core.QueryResult{}, err
	}
	result := core.QueryResult{Columns: columns, Rows: make([][]any, 0), Meta: map[string]any{"row_limit": 1000}}
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
			if bytes, ok := value.([]byte); ok {
				values[i] = string(bytes)
			}
		}
		result.Rows = append(result.Rows, values)
	}
	if err := rows.Err(); err != nil {
		return core.QueryResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return core.QueryResult{}, err
	}
	return result, nil
}

func (p *SQLConnector) Tables(ctx context.Context, databaseID string) ([]core.Table, error) {
	connection, err := p.connection(databaseID)
	if err != nil {
		return nil, err
	}
	items, err := scanTablesForEngine(ctx, connection)
	if err == nil || !recoverableConnectionError(err) {
		return items, err
	}
	if reconnectErr := p.reconnectDatabase(ctx, databaseID, connection); reconnectErr != nil {
		return nil, fmt.Errorf("%w; automatic reconnect failed: %v", err, reconnectErr)
	}
	connection, err = p.connection(databaseID)
	if err != nil {
		return nil, err
	}
	return scanTablesForEngine(ctx, connection)
}

func (p *SQLConnector) Engine(databaseID string) (string, error) {
	connection, err := p.connection(databaseID)
	if err != nil {
		return "", err
	}
	return connection.engine, nil
}

func scanPostgresTables(ctx context.Context, connectionDB *sql.DB) ([]core.Table, error) {
	queryCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	rows, err := connectionDB.QueryContext(queryCtx, postgresCatalogQuery)
	if err != nil {
		return nil, fmt.Errorf("scan tables: %w", err)
	}
	defer rows.Close()
	byKey := map[string]*core.Table{}
	keys := []string{}
	for rows.Next() {
		var schema, name string
		var tableDescription, column, dataType, columnDescription, defaultValue sql.NullString
		var nullable, primaryKey sql.NullBool
		var foreignSchema, foreignTable, foreignColumn sql.NullString
		if err := rows.Scan(
			&schema, &name, &tableDescription, &column, &dataType, &nullable,
			&columnDescription, &defaultValue, &primaryKey,
			&foreignSchema, &foreignTable, &foreignColumn,
		); err != nil {
			return nil, err
		}
		key := schema + "." + name
		table := byKey[key]
		if table == nil {
			table = &core.Table{Schema: schema, Name: name, Description: tableDescription.String}
			byKey[key] = table
			keys = append(keys, key)
		}
		if column.Valid {
			field := core.Column{
				Name:         column.String,
				DataType:     dataType.String,
				Nullable:     nullable.Bool,
				Description:  columnDescription.String,
				DefaultValue: defaultValue.String,
				PrimaryKey:   primaryKey.Bool,
			}
			if foreignTable.Valid && foreignColumn.Valid {
				field.ForeignKey = &core.FieldRef{Schema: foreignSchema.String, Table: foreignTable.String, Name: foreignColumn.String}
			}
			table.Columns = append(table.Columns, field)
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

const postgresCatalogQuery = `
SELECT n.nspname,
       c.relname,
       pg_catalog.obj_description(c.oid, 'pg_class'),
       a.attname,
       pg_catalog.format_type(a.atttypid, a.atttypmod),
       NOT a.attnotnull,
       pg_catalog.col_description(c.oid, a.attnum),
       pg_catalog.pg_get_expr(ad.adbin, ad.adrelid),
       EXISTS (
         SELECT 1
         FROM pg_catalog.pg_index i
         WHERE i.indrelid = c.oid
           AND i.indisprimary
           AND a.attnum = ANY(i.indkey)
       ),
       fk.target_schema,
       fk.target_table,
       fk.target_column
FROM pg_catalog.pg_class c
JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
LEFT JOIN pg_catalog.pg_attribute a ON a.attrelid = c.oid AND a.attnum > 0 AND NOT a.attisdropped
LEFT JOIN pg_catalog.pg_attrdef ad ON ad.adrelid = c.oid AND ad.adnum = a.attnum
LEFT JOIN LATERAL (
  SELECT target_namespace.nspname AS target_schema,
         target_class.relname AS target_table,
         target_attribute.attname AS target_column
  FROM pg_catalog.pg_constraint constraint_row
  JOIN pg_catalog.pg_class target_class ON target_class.oid = constraint_row.confrelid
  JOIN pg_catalog.pg_namespace target_namespace ON target_namespace.oid = target_class.relnamespace
  JOIN unnest(constraint_row.conkey) WITH ORDINALITY source_key(attnum, position)
    ON source_key.attnum = a.attnum
  JOIN unnest(constraint_row.confkey) WITH ORDINALITY target_key(attnum, position)
    ON target_key.position = source_key.position
  JOIN pg_catalog.pg_attribute target_attribute
    ON target_attribute.attrelid = constraint_row.confrelid
   AND target_attribute.attnum = target_key.attnum
  WHERE constraint_row.conrelid = c.oid
    AND constraint_row.contype = 'f'
  ORDER BY constraint_row.oid
  LIMIT 1
) fk ON TRUE
WHERE c.relkind IN ('r', 'p', 'v', 'm', 'f')
  AND n.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
  AND n.nspname NOT LIKE 'pg_temp_%'
  AND n.nspname NOT LIKE 'pg_toast_temp_%'
ORDER BY n.nspname, c.relname, a.attnum NULLS LAST`

func (p *SQLConnector) reconnectDatabase(ctx context.Context, databaseID string, failed databaseConnection) error {
	lock := p.connectionLock(databaseID)
	lock.Lock()
	defer lock.Unlock()

	current, err := p.connection(databaseID)
	if err != nil {
		return err
	}
	if current.db != failed.db && (current.tunnel == nil || current.tunnel.Alive()) {
		return nil
	}
	input := current.input
	if strings.TrimSpace(input.Engine) == "" {
		input = failed.input
	}
	if strings.TrimSpace(input.Engine) == "" {
		return fmt.Errorf("saved in-process connection settings are unavailable")
	}
	input.ID = databaseID
	_, err = p.connect(ctx, input)
	return err
}

func (p *SQLConnector) connectionLock(databaseID string) *sync.Mutex {
	lockValue, _ := p.reconnect.LoadOrStore(databaseID, &sync.Mutex{})
	return lockValue.(*sync.Mutex)
}

func recoverableConnectionError(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if errors.Is(err, driver.ErrBadConn) || errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
		return true
	}
	var networkError net.Error
	if errors.As(err, &networkError) && !networkError.Timeout() {
		return true
	}
	message := strings.ToLower(err.Error())
	for _, fragment := range []string{
		"connection reset by peer",
		"broken pipe",
		"unexpected eof",
		"conn closed",
		"connection refused",
		"network is unreachable",
		"server closed the connection unexpectedly",
		"failed to connect",
		"tls error",
	} {
		if strings.Contains(message, fragment) {
			return true
		}
	}
	return false
}

func (p *SQLConnector) Close(id string) error {
	p.mu.Lock()
	connection, ok := p.databases[id]
	delete(p.databases, id)
	p.mu.Unlock()
	if !ok {
		return nil
	}
	var closeErrors []error
	if connection.db != nil {
		if err := connection.db.Close(); err != nil {
			closeErrors = append(closeErrors, err)
		}
	}
	if connection.tunnel != nil {
		if err := connection.tunnel.Close(); err != nil {
			closeErrors = append(closeErrors, err)
		}
	}
	return errors.Join(closeErrors...)
}
