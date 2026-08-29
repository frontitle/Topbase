package appdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"

	"github.com/topbase/topbase/internal/core"
	"github.com/topbase/topbase/internal/core/queryir"
)

type Store struct {
	db            *database
	schemaVersion int
}

func Open(path string) (*Store, error) {
	return OpenWithVersion(path, "dev")
}

func OpenWithVersion(path, appVersion string) (*Store, error) {
	return OpenConfig(Config{Engine: EngineSQLite, Path: path, AppVersion: appVersion})
}

func OpenConfig(cfg Config) (*Store, error) {
	if cfg.Engine == EnginePostgres && cfg.Schema == "" {
		cfg.Schema = "public"
	}
	if cfg.Engine == EnginePostgres && cfg.Port == 0 {
		cfg.Port = 5432
	}
	if cfg.Engine == EngineMySQL && cfg.Port == 0 {
		cfg.Port = 3306
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if cfg.Engine == EngineSQLite && cfg.Path != "" {
		if err := os.MkdirAll(filepath.Dir(cfg.Path), 0700); err != nil {
			return nil, err
		}
	}
	driver, dsn, err := cfg.driverAndDSN()
	if err != nil {
		return nil, err
	}
	rawDB, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	if cfg.MaxOpenConns > 0 {
		rawDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns >= 0 {
		rawDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		rawDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
	db := &database{DB: rawDB, engine: cfg.Engine}
	connectTimeout := cfg.ConnectTimeout
	if connectTimeout <= 0 {
		connectTimeout = 10 * time.Second
	}
	connectCtx, connectCancel := context.WithTimeout(context.Background(), connectTimeout)
	defer connectCancel()
	if err := db.PingContext(connectCtx); err != nil {
		rawDB.Close()
		return nil, fmt.Errorf("connect application database: %w", err)
	}
	if cfg.Engine == EngineSQLite {
		if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
			rawDB.Close()
			return nil, err
		}
	}
	if cfg.Engine == EnginePostgres && cfg.Schema != "" && cfg.Schema != "public" {
		if _, err := db.Exec(`CREATE SCHEMA IF NOT EXISTS ` + quotePostgresIdentifier(cfg.Schema)); err != nil {
			rawDB.Close()
			return nil, fmt.Errorf("prepare application database schema: %w", err)
		}
	}
	migrationCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	schemaVersion, err := applyMigrations(migrationCtx, db, cfg.AppVersion)
	if err != nil {
		rawDB.Close()
		return nil, fmt.Errorf("migrate app db: %w", err)
	}
	return &Store{db: db, schemaVersion: schemaVersion}, nil
}

func (s *Store) Engine() Engine { return s.db.engine }

func validIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for i, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' || (i > 0 && r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return true
}

func quotePostgresIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

func (s *Store) SchemaVersion() int { return s.schemaVersion }

func (s *Store) Get(key string) (string, bool, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	return value, err == nil, err
}

func (s *Store) Set(key, value string) error {
	_, err := s.db.Exec(`INSERT INTO settings(key, value) VALUES(?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

func (s *Store) Create(user core.User) error {
	_, err := s.db.Exec(`INSERT INTO users(id, email, name, avatar_url, password_hash, locale, theme, is_active, created_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		user.ID, user.Email, user.Name, user.AvatarURL, user.PasswordHash, user.Locale, user.Theme, boolInt(user.IsActive), user.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) ByEmail(email string) (core.User, error) {
	return s.scanUser(s.db.QueryRow(`SELECT id, email, name, avatar_url, password_hash, locale, theme, is_active, created_at FROM users WHERE email = ?`, email))
}

func (s *Store) ByID(id string) (core.User, error) {
	return s.scanUser(s.db.QueryRow(`SELECT id, email, name, avatar_url, password_hash, locale, theme, is_active, created_at FROM users WHERE id = ?`, id))
}

func (s *Store) ListUsers() ([]core.User, error) {
	rows, err := s.db.Query(`SELECT id, email, name, avatar_url, password_hash, locale, theme, is_active, created_at FROM users ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []core.User{}
	for rows.Next() {
		user, err := scanUserRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, user)
	}
	return items, rows.Err()
}

func (s *Store) SetActive(id string, active bool) error {
	res, err := s.db.Exec(`UPDATE users SET is_active=? WHERE id=?`, boolInt(active), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return core.ErrNotFound
	}
	return nil
}

func (s *Store) SetPassword(id, passwordHash string) error {
	res, err := s.db.Exec(`UPDATE users SET password_hash=? WHERE id=?`, passwordHash, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return core.ErrNotFound
	}
	return nil
}

func (s *Store) UpdateUserProfile(id, name, email, locale, theme, avatarURL string) error {
	res, err := s.db.Exec(`UPDATE users SET name=?, email=?, locale=?, theme=?, avatar_url=? WHERE id=?`, name, email, locale, theme, avatarURL, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return core.ErrNotFound
	}
	return nil
}

func scanUserRow(row scanner) (core.User, error) {
	var user core.User
	var hash sql.NullString
	var created string
	var active int
	err := row.Scan(&user.ID, &user.Email, &user.Name, &user.AvatarURL, &hash, &user.Locale, &user.Theme, &active, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return core.User{}, core.ErrNotFound
	}
	if err != nil {
		return core.User{}, err
	}
	user.PasswordHash = hash.String
	user.IsActive = active == 1
	user.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return user, nil
}

func (s *Store) scanUser(row *sql.Row) (core.User, error) {
	return scanUserRow(row)
}

func (s *Store) CreateGroup(group core.Group) error {
	_, err := s.db.Exec(`INSERT INTO groups(id, name, kind) VALUES(?,?,?)`, group.ID, group.Name, group.Kind)
	return err
}

func (s *Store) AddMember(groupID, userID string) error {
	_, err := s.db.Exec(`INSERT OR IGNORE INTO group_members(group_id, user_id) VALUES(?,?)`, groupID, userID)
	return err
}

func (s *Store) CreateSession(session core.Session) error {
	_, err := s.db.Exec(`INSERT INTO sessions(id, user_id, expires_at) VALUES(?,?,?)`, session.ID, session.UserID, session.ExpiresAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) SessionByID(id string) (core.Session, error) {
	var session core.Session
	var expires string
	err := s.db.QueryRow(`SELECT id, user_id, expires_at FROM sessions WHERE id = ?`, id).Scan(&session.ID, &session.UserID, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Session{}, core.ErrNotFound
	}
	if err != nil {
		return core.Session{}, err
	}
	session.ExpiresAt, _ = time.Parse(time.RFC3339, expires)
	return session, nil
}

func (s *Store) DeleteSession(id string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	return err
}

func (s *Store) DeleteExpired(now time.Time) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE expires_at < ?`, now.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) CreateCollection(c core.Collection) error {
	_, err := s.db.Exec(`INSERT INTO collections(id, parent_id, name, personal_owner_user_id, owner_group_id, kind, created_at) VALUES(?,?,?,?,?,?,?)`,
		c.ID, nullString(c.ParentID), c.Name, nullString(c.PersonalOwnerUserID), nullString(c.OwnerGroupID), c.Kind, c.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) ListCollections() ([]core.Collection, error) {
	rows, err := s.db.Query(`SELECT id, parent_id, name, personal_owner_user_id, owner_group_id, kind, created_at FROM collections ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []core.Collection{}
	for rows.Next() {
		var item core.Collection
		var parent, owner, ownerGroup, kind, created sql.NullString
		if err := rows.Scan(&item.ID, &parent, &item.Name, &owner, &ownerGroup, &kind, &created); err != nil {
			return nil, err
		}
		item.ParentID = parent.String
		item.PersonalOwnerUserID = owner.String
		item.OwnerGroupID = ownerGroup.String
		item.Kind = kind.String
		if item.Kind == "" {
			if item.PersonalOwnerUserID != "" {
				item.Kind = "personal_project"
			} else {
				item.Kind = "team_project"
			}
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339, created.String)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) UpdateCollection(c core.Collection) error {
	_, err := s.db.Exec(`UPDATE collections SET parent_id=?, name=? WHERE id=?`, nullString(c.ParentID), c.Name, c.ID)
	return err
}

func (s *Store) DeleteCollection(id string) error {
	_, err := s.db.Exec(`DELETE FROM collections WHERE id=?`, id)
	return err
}

func (s *Store) ListCollectionShares(collectionID string) ([]core.CollectionShare, error) {
	rows, err := s.db.Query(`SELECT collection_id, user_id, created_at FROM collection_shares WHERE collection_id=? ORDER BY created_at`, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []core.CollectionShare{}
	for rows.Next() {
		var item core.CollectionShare
		var created string
		if err := rows.Scan(&item.CollectionID, &item.UserID, &created); err != nil {
			return nil, err
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339, created)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) IsCollectionSharedWith(collectionID, userID string) (bool, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(1) FROM collection_shares WHERE collection_id=? AND user_id=?`, collectionID, userID).Scan(&count)
	return count > 0, err
}

func (s *Store) ReplaceCollectionShares(collectionID string, userIDs []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM collection_shares WHERE collection_id=?`, collectionID); err != nil {
		_ = tx.Rollback()
		return err
	}
	for _, userID := range userIDs {
		if userID == "" {
			continue
		}
		if _, err := tx.Exec(`INSERT INTO collection_shares(collection_id,user_id,created_at) VALUES(?,?,?)`, collectionID, userID, time.Now().UTC().Format(time.RFC3339)); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) CollectionByID(id string) (core.Collection, error) {
	items, err := s.ListCollections()
	if err != nil {
		return core.Collection{}, err
	}
	for _, item := range items {
		if item.ID == id {
			return item, nil
		}
	}
	return core.Collection{}, core.ErrNotFound
}

func (s *Store) CreateQuestion(q core.Question) error {
	queryIR, chart, params, err := encodeQuestion(q)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO questions(id, collection_id, dashboard_id, name, description, queryir, native_sql, chartspec, query_type, database_id, created_by, created_at, archived_at, parameters) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		q.ID, nullString(q.CollectionID), nullString(q.DashboardID), q.Name, q.Description, queryIR, q.NativeSQL, chart, q.QueryType, nullString(q.DatabaseID), nullString(q.CreatedBy), q.CreatedAt.UTC().Format(time.RFC3339), nil, nullString(params))
	return err
}

func (s *Store) UpdateQuestion(q core.Question) error {
	queryIR, chart, params, err := encodeQuestion(q)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`UPDATE questions SET collection_id=?, dashboard_id=?, name=?, description=?, queryir=?, native_sql=?, chartspec=?, query_type=?, database_id=?, parameters=? WHERE id=?`,
		nullString(q.CollectionID), nullString(q.DashboardID), q.Name, q.Description, queryIR, q.NativeSQL, chart, q.QueryType, nullString(q.DatabaseID), nullString(params), q.ID)
	return err
}

func (s *Store) SetQuestionArchived(id string, archivedAt *time.Time) error {
	var value any
	if archivedAt != nil {
		value = archivedAt.UTC().Format(time.RFC3339)
	}
	_, err := s.db.Exec(`UPDATE questions SET archived_at=? WHERE id=?`, value, id)
	return err
}

func (s *Store) QuestionByID(id string) (core.Question, error) {
	row := s.db.QueryRow(`SELECT id, collection_id, dashboard_id, name, description, queryir, native_sql, chartspec, query_type, database_id, created_by, created_at, archived_at, parameters FROM questions WHERE id = ?`, id)
	q, err := scanQuestion(row)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Question{}, core.ErrNotFound
	}
	return q, err
}

func (s *Store) ListQuestions(includeArchived bool) ([]core.Question, error) {
	sqlText := `SELECT id, collection_id, dashboard_id, name, description, queryir, native_sql, chartspec, query_type, database_id, created_by, created_at, archived_at, parameters FROM questions`
	if !includeArchived {
		sqlText += ` WHERE archived_at IS NULL`
	}
	sqlText += ` ORDER BY created_at DESC`
	rows, err := s.db.Query(sqlText)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []core.Question{}
	for rows.Next() {
		q, err := scanQuestionRows(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, q)
	}
	return items, rows.Err()
}

func (s *Store) List() ([]core.Database, error) {
	rows, err := s.db.Query(`SELECT id, name, engine, host, status, created_at FROM catalog_databases ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []core.Database{}
	for rows.Next() {
		var item core.Database
		var created string
		if err := rows.Scan(&item.ID, &item.Name, &item.Engine, &item.Host, &item.Status, &created); err != nil {
			return nil, err
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339, created)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) Save(database core.Database) error {
	_, err := s.db.Exec(`INSERT INTO catalog_databases(id, name, engine, host, status, created_at) VALUES(?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET name=excluded.name, engine=excluded.engine, host=excluded.host, status=excluded.status`,
		database.ID, database.Name, database.Engine, database.Host, database.Status, database.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) Delete(id string) error {
	_, err := s.db.Exec(`DELETE FROM catalog_databases WHERE id = ?`, id)
	return err
}

func (s *Store) GetTableAnnotation(databaseID, schema, table string) (core.TableAnnotation, error) {
	var note core.TableAnnotation
	var fieldTypes sql.NullString
	err := s.db.QueryRow(`SELECT display_name, description, user_note, hidden, field_types FROM table_annotations WHERE database_id=? AND schema_name=? AND table_name=?`, databaseID, schema, table).
		Scan(&note.DisplayName, &note.Description, &note.UserNote, &note.Hidden, &fieldTypes)
	if errors.Is(err, sql.ErrNoRows) {
		return core.TableAnnotation{FieldTypes: map[string]string{}}, nil
	}
	if err != nil {
		return core.TableAnnotation{}, err
	}
	if note.UserNote == "" {
		note.UserNote = note.Description
	}
	note.FieldTypes = map[string]string{}
	if fieldTypes.String != "" {
		_ = json.Unmarshal([]byte(fieldTypes.String), &note.FieldTypes)
	}
	return note, nil
}

func (s *Store) SaveTableAnnotation(databaseID, schema, table string, annotation core.TableAnnotation) error {
	if annotation.FieldTypes == nil {
		annotation.FieldTypes = map[string]string{}
	}
	raw, _ := json.Marshal(annotation.FieldTypes)
	_, err := s.db.Exec(`INSERT INTO table_annotations(database_id, schema_name, table_name, display_name, description, user_note, hidden, field_types) VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(database_id, schema_name, table_name) DO UPDATE SET display_name=excluded.display_name, description=excluded.description, user_note=excluded.user_note, hidden=excluded.hidden, field_types=excluded.field_types`,
		databaseID, schema, table, annotation.DisplayName, annotation.Description, annotation.UserNote, annotation.Hidden, string(raw))
	return err
}

func (s *Store) ImportCatalog(items []core.Database) error {
	existing, err := s.List()
	if err != nil || len(existing) > 0 {
		return err
	}
	for _, item := range items {
		if err := s.Save(item); err != nil {
			return err
		}
	}
	return nil
}

type userAdapter struct{ *Store }

func (s *Store) Users() core.UserStore { return userAdapter{s} }

func (a userAdapter) Create(user core.User) error { return a.Store.Create(user) }
func (a userAdapter) ByEmail(email string) (core.User, error) {
	return a.Store.ByEmail(email)
}
func (a userAdapter) ByID(id string) (core.User, error) { return a.Store.ByID(id) }
func (a userAdapter) List() ([]core.User, error)        { return a.ListUsers() }
func (a userAdapter) SetActive(id string, active bool) error {
	return a.Store.SetActive(id, active)
}
func (a userAdapter) SetPassword(id, passwordHash string) error {
	return a.Store.SetPassword(id, passwordHash)
}
func (a userAdapter) UpdateProfile(id, name, email, locale, theme, avatarURL string) error {
	return a.Store.UpdateUserProfile(id, name, email, locale, theme, avatarURL)
}

type groupAdapter struct{ *Store }

func (s *Store) Groups() core.GroupStore { return groupAdapter{s} }

func (g groupAdapter) Create(group core.Group) error { return g.CreateGroup(group) }

type sessionAdapter struct{ *Store }

func (s *Store) Sessions() core.SessionStore { return sessionAdapter{s} }

func (a sessionAdapter) Create(session core.Session) error { return a.CreateSession(session) }
func (a sessionAdapter) ByID(id string) (core.Session, error) {
	return a.SessionByID(id)
}
func (a sessionAdapter) Delete(id string) error { return a.DeleteSession(id) }
func (a sessionAdapter) DeleteExpired(now time.Time) error {
	return a.Store.DeleteExpired(now)
}

type collectionAdapter struct{ *Store }

func (s *Store) Collections() core.CollectionStore { return collectionAdapter{s} }

func (a collectionAdapter) Create(c core.Collection) error { return a.CreateCollection(c) }
func (a collectionAdapter) Update(c core.Collection) error { return a.UpdateCollection(c) }
func (a collectionAdapter) List() ([]core.Collection, error) {
	return a.ListCollections()
}
func (a collectionAdapter) ByID(id string) (core.Collection, error) {
	return a.CollectionByID(id)
}
func (a collectionAdapter) Delete(id string) error { return a.DeleteCollection(id) }

type questionAdapter struct{ *Store }

func (s *Store) Questions() core.QuestionStore { return questionAdapter{s} }

func (a questionAdapter) Create(q core.Question) error { return a.CreateQuestion(q) }
func (a questionAdapter) Update(q core.Question) error { return a.UpdateQuestion(q) }
func (a questionAdapter) ByID(id string) (core.Question, error) {
	return a.QuestionByID(id)
}
func (a questionAdapter) List(includeArchived bool) ([]core.Question, error) {
	return a.ListQuestions(includeArchived)
}
func (a questionAdapter) SetArchived(id string, archivedAt *time.Time) error {
	return a.SetQuestionArchived(id, archivedAt)
}

type scanner interface {
	Scan(dest ...any) error
}

func scanQuestion(row scanner) (core.Question, error) {
	var q core.Question
	var collection, dashboard, desc, queryIR, native, chart, dbID, createdBy, created, archived, params sql.NullString
	if err := row.Scan(&q.ID, &collection, &dashboard, &q.Name, &desc, &queryIR, &native, &chart, &q.QueryType, &dbID, &createdBy, &created, &archived, &params); err != nil {
		return core.Question{}, err
	}
	q.CollectionID, q.DashboardID, q.Description, q.NativeSQL, q.DatabaseID, q.CreatedBy = collection.String, dashboard.String, desc.String, native.String, dbID.String, createdBy.String
	q.CreatedAt, _ = time.Parse(time.RFC3339, created.String)
	if archived.Valid {
		t, _ := time.Parse(time.RFC3339, archived.String)
		q.ArchivedAt = &t
	}
	if queryIR.String != "" {
		var parsed queryir.Query
		if err := json.Unmarshal([]byte(queryIR.String), &parsed); err == nil {
			q.QueryIR = &parsed
		}
	}
	if chart.String != "" {
		var spec core.ChartSpec
		if err := json.Unmarshal([]byte(chart.String), &spec); err == nil {
			q.ChartSpec = &spec
		}
	}
	if params.String != "" {
		_ = json.Unmarshal([]byte(params.String), &q.Parameters)
	}
	return q, nil
}

func scanQuestionRows(rows *sql.Rows) (core.Question, error) {
	return scanQuestion(rows)
}

func encodeQuestion(q core.Question) (queryIR string, chart string, params string, err error) {
	if q.QueryIR != nil {
		raw, e := json.Marshal(q.QueryIR)
		if e != nil {
			return "", "", "", e
		}
		queryIR = string(raw)
	}
	if q.ChartSpec != nil {
		raw, e := json.Marshal(q.ChartSpec)
		if e != nil {
			return "", "", "", e
		}
		chart = string(raw)
	}
	if len(q.Parameters) > 0 {
		raw, e := json.Marshal(q.Parameters)
		if e != nil {
			return "", "", "", e
		}
		params = string(raw)
	}
	return queryIR, chart, params, nil
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func nullString(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}
