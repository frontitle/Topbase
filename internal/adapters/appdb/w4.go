package appdb

import (
	"database/sql"
	"errors"
	"time"

	"github.com/topbase/topbase/internal/core"
)

func (s *Store) CreateSchedule(item core.Schedule) error {
	_, err := s.db.Exec(`INSERT INTO schedules(id, name, question_id, database_id, cron, timezone, materialize_to, strategy, watermark_field, model_id, confirm_source_write, enabled, last_run_at, created_by, created_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		item.ID, item.Name, item.QuestionID, nullString(item.DatabaseID), item.Cron, item.Timezone, item.MaterializeTo, item.Strategy,
		nullString(item.WatermarkField), nullString(item.ModelID), boolInt(item.ConfirmSourceWrite),
		boolInt(item.Enabled), timePtr(item.LastRunAt), nullString(item.CreatedBy), item.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) UpdateSchedule(item core.Schedule) error {
	_, err := s.db.Exec(`UPDATE schedules SET name=?, question_id=?, database_id=?, cron=?, timezone=?, materialize_to=?, strategy=?, watermark_field=?, model_id=?, confirm_source_write=?, enabled=?, last_run_at=? WHERE id=?`,
		item.Name, item.QuestionID, nullString(item.DatabaseID), item.Cron, item.Timezone, item.MaterializeTo, item.Strategy,
		nullString(item.WatermarkField), nullString(item.ModelID), boolInt(item.ConfirmSourceWrite),
		boolInt(item.Enabled), timePtr(item.LastRunAt), item.ID)
	return err
}

func (s *Store) ScheduleByID(id string) (core.Schedule, error) {
	row := s.db.QueryRow(`SELECT id, name, question_id, database_id, cron, timezone, materialize_to, strategy, watermark_field, model_id, confirm_source_write, enabled, last_run_at, created_by, created_at FROM schedules WHERE id=?`, id)
	item, err := scanSchedule(row)
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

func scanSchedule(row scanner) (core.Schedule, error) {
	var item core.Schedule
	var dbID, last, createdBy, created, watermark, modelID sql.NullString
	var enabled, confirm int
	if err := row.Scan(&item.ID, &item.Name, &item.QuestionID, &dbID, &item.Cron, &item.Timezone, &item.MaterializeTo, &item.Strategy, &watermark, &modelID, &confirm, &enabled, &last, &createdBy, &created); err != nil {
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

func (s *Store) CreateRun(item core.Run) error {
	_, err := s.db.Exec(`INSERT INTO schedule_runs(id, schedule_id, status, sql_compiled, row_count, error, started_at, finished_at) VALUES(?,?,?,?,?,?,?,?)`,
		item.ID, item.ScheduleID, item.Status, item.SQLCompiled, item.RowCount, nullString(item.Error), item.StartedAt.UTC().Format(time.RFC3339), timePtr(item.FinishedAt))
	return err
}

func (s *Store) UpdateRun(item core.Run) error {
	_, err := s.db.Exec(`UPDATE schedule_runs SET status=?, sql_compiled=?, row_count=?, error=?, finished_at=? WHERE id=?`,
		item.Status, item.SQLCompiled, item.RowCount, nullString(item.Error), timePtr(item.FinishedAt), item.ID)
	return err
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
	_, err := s.db.Exec(`INSERT INTO materialized_tables(id, database_id, schema_name, table_name, schedule_id, question_id, last_run_at, last_status, row_count, watermark)
		VALUES(?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(database_id, schema_name, table_name) DO UPDATE SET
		schedule_id=excluded.schedule_id, question_id=excluded.question_id, last_run_at=excluded.last_run_at, last_status=excluded.last_status, row_count=excluded.row_count, watermark=excluded.watermark`,
		item.ID, item.DatabaseID, item.Schema, item.Name, nullString(item.ScheduleID), nullString(item.QuestionID), timePtr(item.LastRunAt), item.LastStatus, item.RowCount, nullString(item.Watermark))
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
	_, err := s.db.Exec(`INSERT INTO lineage_edges(from_type, from_id, to_type, to_id) VALUES(?,?,?,?)`, edge.FromType, edge.FromID, edge.ToType, edge.ToID)
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
