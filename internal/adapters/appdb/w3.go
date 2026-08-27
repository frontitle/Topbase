package appdb

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/topbase/topbase/internal/core"
	"github.com/topbase/topbase/internal/core/queryir"
)

func (s *Store) SaveField(field core.FieldMeta) error {
	if !core.ValidSemanticType(field.SemanticType) {
		return errors.New("unsupported semantic type")
	}
	format, _ := json.Marshal(field.Format)
	var fkSchema, fkTable, fkField any
	if field.FKTarget != nil {
		fkSchema, fkTable, fkField = nullString(field.FKTarget.Schema), nullString(field.FKTarget.Table), nullString(field.FKTarget.Name)
	}
	_, err := s.db.Exec(`INSERT INTO field_metadata(database_id, schema_name, table_name, field_name, display_name, description, semantic_type, visibility, format, fk_schema, fk_table, fk_field)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(database_id, schema_name, table_name, field_name) DO UPDATE SET
		display_name=excluded.display_name, description=excluded.description, semantic_type=excluded.semantic_type,
		visibility=excluded.visibility, format=excluded.format, fk_schema=excluded.fk_schema, fk_table=excluded.fk_table, fk_field=excluded.fk_field`,
		field.DatabaseID, field.Schema, field.Table, field.Name, field.DisplayName, field.Description, field.SemanticType, field.Visibility, string(format), fkSchema, fkTable, fkField)
	return err
}

func (s *Store) ListFields(databaseID, schema, table string) ([]core.FieldMeta, error) {
	return s.queryFields(`SELECT database_id, schema_name, table_name, field_name, display_name, description, semantic_type, visibility, format, fk_schema, fk_table, fk_field FROM field_metadata WHERE database_id=? AND schema_name=? AND table_name=?`, databaseID, schema, table)
}

func (s *Store) ListDatabaseFields(databaseID string) ([]core.FieldMeta, error) {
	return s.queryFields(`SELECT database_id, schema_name, table_name, field_name, display_name, description, semantic_type, visibility, format, fk_schema, fk_table, fk_field FROM field_metadata WHERE database_id=?`, databaseID)
}

func (s *Store) queryFields(sqlText string, args ...any) ([]core.FieldMeta, error) {
	rows, err := s.db.Query(sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []core.FieldMeta{}
	for rows.Next() {
		var item core.FieldMeta
		var display, desc, semantic, visibility, format, fkSchema, fkTable, fkField sql.NullString
		if err := rows.Scan(&item.DatabaseID, &item.Schema, &item.Table, &item.Name, &display, &desc, &semantic, &visibility, &format, &fkSchema, &fkTable, &fkField); err != nil {
			return nil, err
		}
		item.DisplayName, item.Description, item.SemanticType, item.Visibility = display.String, desc.String, semantic.String, visibility.String
		if format.String != "" {
			_ = json.Unmarshal([]byte(format.String), &item.Format)
		}
		if fkTable.String != "" {
			item.FKTarget = &core.FieldRef{Schema: fkSchema.String, Table: fkTable.String, Name: fkField.String}
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateModel(m core.Model) error {
	queryIR, columns, err := encodeModel(m)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO models(id, collection_id, name, database_id, queryir, native_sql, columns, created_by, created_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		m.ID, nullString(m.CollectionID), m.Name, m.DatabaseID, queryIR, m.NativeSQL, columns, nullString(m.CreatedBy), m.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) UpdateModel(m core.Model) error {
	queryIR, columns, err := encodeModel(m)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`UPDATE models SET collection_id=?, name=?, database_id=?, queryir=?, native_sql=?, columns=? WHERE id=?`,
		nullString(m.CollectionID), m.Name, m.DatabaseID, queryIR, m.NativeSQL, columns, m.ID)
	return err
}

func (s *Store) ModelByID(id string) (core.Model, error) {
	row := s.db.QueryRow(`SELECT id, collection_id, name, database_id, queryir, native_sql, columns, created_by, created_at FROM models WHERE id=?`, id)
	m, err := scanModel(row)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Model{}, core.ErrNotFound
	}
	return m, err
}

func (s *Store) ListModels() ([]core.Model, error) {
	rows, err := s.db.Query(`SELECT id, collection_id, name, database_id, queryir, native_sql, columns, created_by, created_at FROM models ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []core.Model{}
	for rows.Next() {
		item, err := scanModel(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func encodeModel(m core.Model) (queryIR string, columns string, err error) {
	if m.QueryIR != nil {
		raw, e := json.Marshal(m.QueryIR)
		if e != nil {
			return "", "", e
		}
		queryIR = string(raw)
	}
	if len(m.Columns) > 0 {
		raw, e := json.Marshal(m.Columns)
		if e != nil {
			return "", "", e
		}
		columns = string(raw)
	}
	return queryIR, columns, nil
}

func scanModel(row scanner) (core.Model, error) {
	var m core.Model
	var collection, queryIR, native, columns, createdBy, created sql.NullString
	if err := row.Scan(&m.ID, &collection, &m.Name, &m.DatabaseID, &queryIR, &native, &columns, &createdBy, &created); err != nil {
		return core.Model{}, err
	}
	m.CollectionID, m.NativeSQL, m.CreatedBy = collection.String, native.String, createdBy.String
	m.CreatedAt, _ = time.Parse(time.RFC3339, created.String)
	if queryIR.String != "" {
		var parsed queryir.Query
		if err := json.Unmarshal([]byte(queryIR.String), &parsed); err == nil {
			m.QueryIR = &parsed
		}
	}
	if columns.String != "" {
		_ = json.Unmarshal([]byte(columns.String), &m.Columns)
	}
	return m, nil
}

func (s *Store) CreateMetric(m core.Metric) error {
	agg, _ := json.Marshal(m.Aggregation)
	filters, _ := json.Marshal(m.Filters)
	_, err := s.db.Exec(`INSERT INTO metrics(id, name, database_id, schema_name, table_name, aggregation, filters, created_at) VALUES(?,?,?,?,?,?,?,?)`,
		m.ID, m.Name, m.DatabaseID, m.Schema, m.Table, string(agg), string(filters), m.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) MetricByID(id string) (core.Metric, error) {
	var m core.Metric
	var agg, filters, created sql.NullString
	err := s.db.QueryRow(`SELECT id, name, database_id, schema_name, table_name, aggregation, filters, created_at FROM metrics WHERE id=?`, id).
		Scan(&m.ID, &m.Name, &m.DatabaseID, &m.Schema, &m.Table, &agg, &filters, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Metric{}, core.ErrNotFound
	}
	if err != nil {
		return core.Metric{}, err
	}
	_ = json.Unmarshal([]byte(agg.String), &m.Aggregation)
	if filters.String != "" {
		_ = json.Unmarshal([]byte(filters.String), &m.Filters)
	}
	m.CreatedAt, _ = time.Parse(time.RFC3339, created.String)
	return m, nil
}

func (s *Store) ListMetrics() ([]core.Metric, error) {
	rows, err := s.db.Query(`SELECT id, name, database_id, schema_name, table_name, aggregation, filters, created_at FROM metrics ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []core.Metric{}
	for rows.Next() {
		var m core.Metric
		var agg, filters, created sql.NullString
		if err := rows.Scan(&m.ID, &m.Name, &m.DatabaseID, &m.Schema, &m.Table, &agg, &filters, &created); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(agg.String), &m.Aggregation)
		if filters.String != "" {
			_ = json.Unmarshal([]byte(filters.String), &m.Filters)
		}
		m.CreatedAt, _ = time.Parse(time.RFC3339, created.String)
		items = append(items, m)
	}
	return items, rows.Err()
}

func (s *Store) CreateSegment(seg core.Segment) error {
	filters, _ := json.Marshal(seg.Filters)
	_, err := s.db.Exec(`INSERT INTO segments(id, name, database_id, schema_name, table_name, filters, created_at) VALUES(?,?,?,?,?,?,?)`,
		seg.ID, seg.Name, seg.DatabaseID, seg.Schema, seg.Table, string(filters), seg.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *Store) SegmentByID(id string) (core.Segment, error) {
	var seg core.Segment
	var filters, created sql.NullString
	err := s.db.QueryRow(`SELECT id, name, database_id, schema_name, table_name, filters, created_at FROM segments WHERE id=?`, id).
		Scan(&seg.ID, &seg.Name, &seg.DatabaseID, &seg.Schema, &seg.Table, &filters, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Segment{}, core.ErrNotFound
	}
	if err != nil {
		return core.Segment{}, err
	}
	_ = json.Unmarshal([]byte(filters.String), &seg.Filters)
	seg.CreatedAt, _ = time.Parse(time.RFC3339, created.String)
	return seg, nil
}

func (s *Store) ListSegments() ([]core.Segment, error) {
	rows, err := s.db.Query(`SELECT id, name, database_id, schema_name, table_name, filters, created_at FROM segments ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []core.Segment{}
	for rows.Next() {
		var seg core.Segment
		var filters, created sql.NullString
		if err := rows.Scan(&seg.ID, &seg.Name, &seg.DatabaseID, &seg.Schema, &seg.Table, &filters, &created); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(filters.String), &seg.Filters)
		seg.CreatedAt, _ = time.Parse(time.RFC3339, created.String)
		items = append(items, seg)
	}
	return items, rows.Err()
}

func (s *Store) CreateGlossary(term core.GlossaryTerm) error {
	_, err := s.db.Exec(`INSERT INTO glossary_terms(id, name, definition) VALUES(?,?,?)`, term.ID, term.Name, term.Definition)
	return err
}

func (s *Store) ListGlossary() ([]core.GlossaryTerm, error) {
	rows, err := s.db.Query(`SELECT id, name, definition FROM glossary_terms ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []core.GlossaryTerm{}
	for rows.Next() {
		var term core.GlossaryTerm
		if err := rows.Scan(&term.ID, &term.Name, &term.Definition); err != nil {
			return nil, err
		}
		items = append(items, term)
	}
	return items, rows.Err()
}

type fieldAdapter struct{ *Store }

func (s *Store) Fields() core.FieldStore { return fieldAdapter{s} }

func (a fieldAdapter) SaveField(field core.FieldMeta) error { return a.Store.SaveField(field) }
func (a fieldAdapter) ListFields(databaseID, schema, table string) ([]core.FieldMeta, error) {
	return a.Store.ListFields(databaseID, schema, table)
}
func (a fieldAdapter) ListDatabaseFields(databaseID string) ([]core.FieldMeta, error) {
	return a.Store.ListDatabaseFields(databaseID)
}

type modelAdapter struct{ *Store }

func (s *Store) Models() core.ModelStore { return modelAdapter{s} }

func (a modelAdapter) Create(m core.Model) error { return a.CreateModel(m) }
func (a modelAdapter) Update(m core.Model) error { return a.UpdateModel(m) }
func (a modelAdapter) ByID(id string) (core.Model, error) {
	return a.ModelByID(id)
}
func (a modelAdapter) List() ([]core.Model, error) { return a.ListModels() }

type metricAdapter struct{ *Store }

func (s *Store) Metrics() core.MetricStore { return metricAdapter{s} }

func (a metricAdapter) Create(m core.Metric) error { return a.CreateMetric(m) }
func (a metricAdapter) ByID(id string) (core.Metric, error) {
	return a.MetricByID(id)
}
func (a metricAdapter) List() ([]core.Metric, error) { return a.ListMetrics() }

type segmentAdapter struct{ *Store }

func (s *Store) Segments() core.SegmentStore { return segmentAdapter{s} }

func (a segmentAdapter) Create(seg core.Segment) error { return a.CreateSegment(seg) }
func (a segmentAdapter) ByID(id string) (core.Segment, error) {
	return a.SegmentByID(id)
}
func (a segmentAdapter) List() ([]core.Segment, error) { return a.ListSegments() }

type glossaryAdapter struct{ *Store }

func (s *Store) Glossary() core.GlossaryStore { return glossaryAdapter{s} }

func (a glossaryAdapter) Create(term core.GlossaryTerm) error { return a.CreateGlossary(term) }
func (a glossaryAdapter) List() ([]core.GlossaryTerm, error)  { return a.ListGlossary() }
