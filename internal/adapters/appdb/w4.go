package appdb

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/topbase/topbase/internal/core"
	"github.com/topbase/topbase/internal/core/queryir"
)

func (s *Store) CreateSchedule(item core.Schedule) error {
	schema, table, err := core.ParseWarehouseTarget(item.MaterializeTo)
	if err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Claim first. The unique key serializes competing creators, including on
	// different nodes. A failed schedule insert rolls back its claim as well.
	key := sha256.Sum256([]byte(item.DatabaseID + "\x00" + schema + "\x00" + table))
	res, err := tx.Exec(`INSERT OR IGNORE INTO warehouse_target_claims(target_key, schedule_id) VALUES(?,?)`,
		hex.EncodeToString(key[:]), item.ID)
	if err != nil {
		return err
	}
	claimed, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if claimed == 0 {
		return core.ErrWarehouseTargetInUse
	}
	if err := checkExistingWarehouseTarget(tx, item.DatabaseID, schema, table); err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO schedules(id, name, question_id, database_id, cron, timezone, materialize_to, strategy, watermark_field, model_id, confirm_source_write, enabled, last_run_at, created_by, created_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		item.ID, item.Name, item.QuestionID, nullString(item.DatabaseID), item.Cron, item.Timezone, item.MaterializeTo, item.Strategy,
		nullString(item.WatermarkField), nullString(item.ModelID), boolInt(item.ConfirmSourceWrite),
		boolInt(item.Enabled), timePtr(item.LastRunAt), nullString(item.CreatedBy), item.CreatedAt.UTC().Format(time.RFC3339))
	if err != nil {
		return err
	}
	return tx.Commit()
}

func checkExistingWarehouseTarget(tx *transaction, databaseID, schema, table string) error {
	var count int
	err := tx.QueryRow(`SELECT COUNT(*) FROM materialized_tables WHERE database_id=? AND schema_name=? AND table_name=?`,
		databaseID, schema, table).Scan(&count)
	if err != nil {
		return err
	}
	if count != 0 {
		return core.ErrWarehouseTargetInUse
	}
	// Earlier schedules have no claim and may omit the target schema, wh_
	// prefix or connection. Preserve these rows, including pre-existing
	// duplicates, and resolve their effective target using the original source.
	rows, err := tx.Query(`SELECT s.materialize_to, COALESCE(NULLIF(s.database_id,''),
		CASE WHEN COALESCE(s.model_id,'')<>'' THEN m.database_id ELSE q.database_id END, ''),
		CASE WHEN COALESCE(s.model_id,'')<>'' THEN m.queryir ELSE q.queryir END
		FROM schedules s LEFT JOIN warehouse_target_claims c ON c.schedule_id=s.id
		LEFT JOIN models m ON m.id=s.model_id LEFT JOIN questions q ON q.id=s.question_id
		WHERE c.schedule_id IS NULL AND (s.database_id=? OR COALESCE(s.database_id,'')='')`, databaseID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var raw, effectiveDB string
		var query sql.NullString
		if err := rows.Scan(&raw, &effectiveDB, &query); err != nil {
			return err
		}
		oldSchema, oldTable, err := core.ParseWarehouseTarget(raw)
		if err != nil || oldSchema[:min(len(oldSchema), 63)] != schema || oldTable[:min(len(oldTable), 63)] != table {
			continue
		}
		if effectiveDB == "" && query.Valid {
			var source struct {
				Source queryir.Source `json:"source"`
			}
			if err := json.Unmarshal([]byte(query.String), &source); err != nil {
				return err
			}
			effectiveDB = source.Source.DatabaseID
		}
		if effectiveDB == databaseID {
			return core.ErrWarehouseTargetInUse
		}
	}
	return rows.Err()
}

func (s *Store) UpdateSchedule(item core.Schedule) error {
	_, err := s.db.Exec(`UPDATE schedules SET name=?, question_id=?, database_id=?, cron=?, timezone=?, materialize_to=?, strategy=?, watermark_field=?, model_id=?, confirm_source_write=?, enabled=?, last_run_at=? WHERE id=?`,
		item.Name, item.QuestionID, nullString(item.DatabaseID), item.Cron, item.Timezone, item.MaterializeTo, item.Strategy,
		nullString(item.WatermarkField), nullString(item.ModelID), boolInt(item.ConfirmSourceWrite),
		boolInt(item.Enabled), timePtr(item.LastRunAt), item.ID)
	return err
}

func (s *Store) ScheduleByID(id string) (core.Schedule, error) {
	row := s.db.QueryRow(`SELECT id, name, question_id, database_id, cron, timezone, materialize_to, strategy, watermark_field, model_id, confirm_source_write, enabled, last_run_at, created_by, created_at,
		EXISTS(SELECT 1 FROM warehouse_target_claims c WHERE c.schedule_id=schedules.id) FROM schedules WHERE id=?`, id)
	var claimed bool
	item, err := scanSchedule(row, &claimed)
	item.ClaimedTarget = claimed
	if errors.Is(err, sql.ErrNoRows) {
		return core.Schedule{}, core.ErrNotFound
	}
	return item, err
}

func (s *Store) ListSchedules() ([]core.Schedule, error) {
	rows, err := s.db.Query(`SELECT id, name, question_id, database_id, cron, timezone, materialize_to, strategy, watermark_field, model_id, confirm_source_write, enabled, last_run_at, created_by, created_at FROM schedules ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []core.Schedule{}
	for rows.Next() {
		item, err := scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ListScheduleSummaries reads one indexed run per plan, including plans that
// have never produced a materialized table. Compiled SQL stays in run history.
func (s *Store) ListScheduleSummaries() ([]core.ScheduleSummary, error) {
	rows, err := s.db.Query(`SELECT s.id, s.name, s.question_id, s.database_id, s.cron, s.timezone, s.materialize_to, s.strategy,
		s.watermark_field, s.model_id, s.confirm_source_write, s.enabled, s.last_run_at, s.created_by, s.created_at,
		r.status, r.error FROM schedules s LEFT JOIN schedule_runs r ON r.id = (
			SELECT h.id FROM schedule_runs h WHERE h.schedule_id = s.id ORDER BY h.started_at DESC, h.id DESC LIMIT 1
		) ORDER BY s.created_at DESC, s.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []core.ScheduleSummary{}
	for rows.Next() {
		var status, message sql.NullString
		item, err := scanSchedule(rows, &status, &message)
		if err != nil {
			return nil, err
		}
		items = append(items, core.ScheduleSummary{Schedule: item, LastStatus: status.String, LastError: message.String})
	}
	return items, rows.Err()
}

func scanSchedule(row scanner, extra ...any) (core.Schedule, error) {
	var item core.Schedule
	var dbID, last, createdBy, created, watermark, modelID sql.NullString
	var enabled, confirm int
	fields := []any{&item.ID, &item.Name, &item.QuestionID, &dbID, &item.Cron, &item.Timezone, &item.MaterializeTo, &item.Strategy, &watermark, &modelID, &confirm, &enabled, &last, &createdBy, &created}
	if err := row.Scan(append(fields, extra...)...); err != nil {
		return core.Schedule{}, err
	}
	item.DatabaseID, item.CreatedBy, item.Enabled = dbID.String, createdBy.String, enabled == 1
	item.WatermarkField, item.ModelID, item.ConfirmSourceWrite = watermark.String, modelID.String, confirm == 1
	item.CreatedAt, _ = time.Parse(time.RFC3339, created.String)
	if last.String != "" {
		t, _ := time.Parse(time.RFC3339, last.String)
		item.LastRunAt = &t
	}
	return item, nil
}

const warehouseRunTimeLayout = "2006-01-02T15:04:05.000000000Z"

func runFinishedAt(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(warehouseRunTimeLayout)
}

func (s *Store) CreateRun(item core.Run) error {
	_, err := s.db.Exec(`INSERT INTO schedule_runs(id, schedule_id, status, sql_compiled, row_count, error, started_at, finished_at) VALUES(?,?,?,?,?,?,?,?)`,
		item.ID, item.ScheduleID, item.Status, item.SQLCompiled, item.RowCount, nullString(item.Error), item.StartedAt.UTC().Format(warehouseRunTimeLayout), runFinishedAt(item.FinishedAt))
	return err
}

func (s *Store) UpdateRun(item core.Run) error {
	if item.Status != "writing" {
		return updateRun(s.db, item)
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// A single indexed key per in-flight plan protects restarts and avoids
	// scanning historical successful runs to decide if a retry is safe.
	if _, err = tx.Exec(`INSERT INTO settings(key,value) VALUES(?,?)`, "warehouse_active_run:"+item.ScheduleID, item.ID); err != nil {
		return err
	}
	if err = updateRun(tx, item); err != nil {
		return err
	}
	return tx.Commit()
}

type warehouseExecer interface {
	Exec(string, ...any) (sql.Result, error)
}

func updateRun(db warehouseExecer, item core.Run) error {
	res, err := db.Exec(`UPDATE schedule_runs SET status=?, sql_compiled=?, row_count=?, error=?, finished_at=? WHERE id=?`,
		item.Status, item.SQLCompiled, item.RowCount, nullString(item.Error), runFinishedAt(item.FinishedAt), item.ID)
	if err == nil && item.Status == "writing" {
		changed, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if changed != 1 {
			return core.ErrNotFound
		}
	}
	return err
}

func (a runAdapter) Unconfirmed(scheduleID string) (core.Run, *core.MaterializedTable, error) {
	id, found, err := a.Get("warehouse_active_run:" + scheduleID)
	if err != nil {
		return core.Run{}, nil, err
	}
	if !found {
		return core.Run{}, nil, core.ErrNotFound
	}
	var item core.Run
	var message, compiled sql.NullString
	var started string
	err = a.db.QueryRow(`SELECT id,status,error,sql_compiled,started_at FROM schedule_runs WHERE id=? AND schedule_id=? AND status IN ('writing','needs_review')`,
		id, scheduleID).Scan(&item.ID, &item.Status, &message, &compiled, &started)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Run{}, nil, fmt.Errorf("运行 %s 的写入屏障与历史记录不一致，请核对数据", id)
	}
	if err != nil {
		return core.Run{}, nil, err
	}
	item.ScheduleID, item.Error, item.SQLCompiled = scheduleID, message.String, compiled.String
	item.StartedAt, err = time.Parse(time.RFC3339, started)
	if err != nil {
		return item, nil, err
	}
	raw, found, err := a.Get("warehouse_run_receipt:" + item.ID)
	if err != nil || !found {
		return item, nil, err
	}
	var table core.MaterializedTable
	if err := json.Unmarshal([]byte(raw), &table); err != nil {
		return item, nil, err
	}
	if table.ScheduleID != scheduleID || table.LastRunAt == nil {
		return item, nil, fmt.Errorf("invalid warehouse receipt for %s", item.ID)
	}
	return item, &table, nil
}

func (a runAdapter) MarkUncertain(item core.Run, completed *core.MaterializedTable) error {
	// A commit acknowledgement can be lost after the application database
	// committed successfully. Never downgrade an already-finalized run.
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`UPDATE schedule_runs SET status='needs_review',error=?,row_count=? WHERE id=? AND status='writing'`,
		item.Error, item.RowCount, item.ID)
	if err != nil {
		return err
	}
	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if count == 1 && completed != nil {
		raw, err := json.Marshal(completed)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(`INSERT INTO settings(key,value) VALUES(?,?)`, "warehouse_run_receipt:"+item.ID, string(raw)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (a runAdapter) Finish(run core.Run, table *core.MaterializedTable) error {
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if table != nil {
		if err := upsertMaterializedTable(tx, *table); err != nil {
			return err
		}
		for _, edge := range []core.LineageEdge{
			{FromType: "question", FromID: table.QuestionID},
			{FromType: "schedule", FromID: run.ScheduleID},
		} {
			edge.ToType, edge.ToID = "table", table.Schema+"."+table.Name
			if err := addLineage(tx, edge); err != nil {
				return err
			}
		}
	}
	res, err := tx.Exec(`UPDATE schedules SET last_run_at=? WHERE id=?`, runFinishedAt(run.FinishedAt), run.ScheduleID)
	if err != nil {
		return err
	}
	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return core.ErrNotFound
	}
	res, err = tx.Exec(`UPDATE schedule_runs SET status=?,sql_compiled=?,row_count=?,error=?,finished_at=? WHERE id=? AND schedule_id=? AND status IN ('running','writing','needs_review')`,
		run.Status, run.SQLCompiled, run.RowCount, nullString(run.Error), runFinishedAt(run.FinishedAt), run.ID, run.ScheduleID)
	if err != nil {
		return err
	}
	count, err = res.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return core.ErrNotFound
	}
	if _, err := tx.Exec(`DELETE FROM settings WHERE key=?`, "warehouse_run_receipt:"+run.ID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM settings WHERE key=? AND value=?`, "warehouse_active_run:"+run.ScheduleID, run.ID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ListRuns(scheduleID string) ([]core.Run, error) {
	rows, err := s.db.Query(`SELECT id, schedule_id, status, sql_compiled, row_count, error, started_at, finished_at FROM schedule_runs WHERE schedule_id=? OR ?='' ORDER BY started_at DESC LIMIT 100`, scheduleID, scheduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []core.Run{}
	for rows.Next() {
		var item core.Run
		var sqlText, errText, started, finished sql.NullString
		if err := rows.Scan(&item.ID, &item.ScheduleID, &item.Status, &sqlText, &item.RowCount, &errText, &started, &finished); err != nil {
			return nil, err
		}
		item.SQLCompiled, item.Error = sqlText.String, errText.String
		item.StartedAt, _ = time.Parse(time.RFC3339, started.String)
		if finished.String != "" {
			t, _ := time.Parse(time.RFC3339, finished.String)
			item.FinishedAt = &t
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) UpsertMaterializedTable(item core.MaterializedTable) error {
	return upsertMaterializedTable(s.db, item)
}

func upsertMaterializedTable(db warehouseExecer, item core.MaterializedTable) error {
	_, err := db.Exec(`INSERT INTO materialized_tables(id, database_id, schema_name, table_name, schedule_id, question_id, last_run_at, last_status, row_count, watermark)
		VALUES(?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(database_id, schema_name, table_name) DO UPDATE SET
		schedule_id=excluded.schedule_id, question_id=excluded.question_id, last_run_at=excluded.last_run_at, last_status=excluded.last_status, row_count=excluded.row_count, watermark=excluded.watermark`,
		item.ID, item.DatabaseID, item.Schema, item.Name, nullString(item.ScheduleID), nullString(item.QuestionID), runFinishedAt(item.LastRunAt), item.LastStatus, item.RowCount, nullString(item.Watermark))
	return err
}

func (s *Store) ListMaterializedTables() ([]core.MaterializedTable, error) {
	return s.queryMaterializedTables(`SELECT id, database_id, schema_name, table_name, schedule_id, question_id, last_run_at, last_status, row_count, watermark FROM materialized_tables ORDER BY table_name`)
}

func (s *Store) ListMaterializedTablesByDatabase(databaseID string) ([]core.MaterializedTable, error) {
	return s.queryMaterializedTables(`SELECT id, database_id, schema_name, table_name, schedule_id, question_id, last_run_at, last_status, row_count, watermark FROM materialized_tables WHERE database_id=? ORDER BY table_name`, databaseID)
}

func (s *Store) MaterializedTableByTarget(databaseID, schema, name string) (core.MaterializedTable, error) {
	items, err := s.queryMaterializedTables(`SELECT id, database_id, schema_name, table_name, schedule_id, question_id, last_run_at, last_status, row_count, watermark FROM materialized_tables WHERE database_id=? AND schema_name=? AND table_name=?`, databaseID, schema, name)
	if err != nil {
		return core.MaterializedTable{}, err
	}
	if len(items) == 0 {
		return core.MaterializedTable{}, core.ErrNotFound
	}
	return items[0], nil
}

func (s *Store) queryMaterializedTables(sqlText string, args ...any) ([]core.MaterializedTable, error) {
	rows, err := s.db.Query(sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []core.MaterializedTable{}
	for rows.Next() {
		var item core.MaterializedTable
		var scheduleID, questionID, last, status, watermark sql.NullString
		if err := rows.Scan(&item.ID, &item.DatabaseID, &item.Schema, &item.Name, &scheduleID, &questionID, &last, &status, &item.RowCount, &watermark); err != nil {
			return nil, err
		}
		item.ScheduleID, item.QuestionID, item.LastStatus, item.Warehouse, item.Watermark = scheduleID.String, questionID.String, status.String, true, watermark.String
		if last.String != "" {
			t, _ := time.Parse(time.RFC3339, last.String)
			item.LastRunAt = &t
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) AddLineage(edge core.LineageEdge) error {
	return addLineage(s.db, edge)
}

func addLineage(db warehouseExecer, edge core.LineageEdge) error {
	_, err := db.Exec(`INSERT INTO lineage_edges(from_type, from_id, to_type, to_id)
		SELECT ?,?,?,? WHERE NOT EXISTS
		(SELECT 1 FROM lineage_edges WHERE from_type=? AND from_id=? AND to_type=? AND to_id=?)`,
		edge.FromType, edge.FromID, edge.ToType, edge.ToID, edge.FromType, edge.FromID, edge.ToType, edge.ToID)
	return err
}

func (s *Store) ListLineage(entityType, id string) ([]core.LineageEdge, error) {
	rows, err := s.db.Query(`SELECT from_type, from_id, to_type, to_id FROM lineage_edges WHERE (from_type=? AND from_id=?) OR (to_type=? AND to_id=?)`, entityType, id, entityType, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []core.LineageEdge{}
	for rows.Next() {
		var edge core.LineageEdge
		if err := rows.Scan(&edge.FromType, &edge.FromID, &edge.ToType, &edge.ToID); err != nil {
			return nil, err
		}
		items = append(items, edge)
	}
	return items, rows.Err()
}

func timePtr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}

type scheduleAdapter struct{ *Store }

func (s *Store) Schedules() core.ScheduleStore { return scheduleAdapter{s} }

func (a scheduleAdapter) Create(item core.Schedule) error { return a.CreateSchedule(item) }
func (a scheduleAdapter) Update(item core.Schedule) error { return a.UpdateSchedule(item) }
func (a scheduleAdapter) ByID(id string) (core.Schedule, error) {
	return a.ScheduleByID(id)
}
func (a scheduleAdapter) List() ([]core.Schedule, error) { return a.ListSchedules() }

type runAdapter struct{ *Store }

func (s *Store) Runs() core.RunStore { return runAdapter{s} }

func (a runAdapter) Create(item core.Run) error { return a.CreateRun(item) }
func (a runAdapter) Update(item core.Run) error { return a.UpdateRun(item) }
func (a runAdapter) List(scheduleID string) ([]core.Run, error) {
	return a.ListRuns(scheduleID)
}

type materializedAdapter struct{ *Store }

func (s *Store) Materialized() core.MaterializedTableStore { return materializedAdapter{s} }

func (a materializedAdapter) Upsert(item core.MaterializedTable) error {
	return a.UpsertMaterializedTable(item)
}
func (a materializedAdapter) List() ([]core.MaterializedTable, error) {
	return a.ListMaterializedTables()
}
func (a materializedAdapter) ListByDatabase(databaseID string) ([]core.MaterializedTable, error) {
	return a.ListMaterializedTablesByDatabase(databaseID)
}
func (a materializedAdapter) ByTarget(databaseID, schema, name string) (core.MaterializedTable, error) {
	return a.MaterializedTableByTarget(databaseID, schema, name)
}

type lineageAdapter struct{ *Store }

func (s *Store) Lineage() core.LineageStore { return lineageAdapter{s} }

func (a lineageAdapter) Add(edge core.LineageEdge) error { return a.AddLineage(edge) }
func (a lineageAdapter) List(entityType, id string) ([]core.LineageEdge, error) {
	return a.ListLineage(entityType, id)
}
