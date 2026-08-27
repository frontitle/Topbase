package content

import (
	"fmt"
	"strings"
	"time"

	"github.com/topbase/topbase/internal/core"
)

func (s Service) SaveField(field core.FieldMeta) (core.FieldMeta, error) {
	if field.Name == "" || field.Table == "" || field.Schema == "" {
		return core.FieldMeta{}, fmt.Errorf("schema, table and field name are required")
	}
	if !core.ValidSemanticType(field.SemanticType) {
		return core.FieldMeta{}, fmt.Errorf("unsupported semantic type")
	}
	if field.SemanticType == "ForeignKey" && (field.FKTarget == nil || field.FKTarget.Table == "" || field.FKTarget.Name == "") {
		return core.FieldMeta{}, fmt.Errorf("foreign key target is required")
	}
	if err := s.Fields.SaveField(field); err != nil {
		return core.FieldMeta{}, err
	}
	return field, nil
}

func (s Service) ListFields(databaseID, schema, table string) ([]core.FieldMeta, error) {
	if s.Fields == nil {
		return []core.FieldMeta{}, nil
	}
	return s.Fields.ListFields(databaseID, schema, table)
}

func (s Service) CreateModel(m core.Model, userID string) (core.Model, error) {
	if strings.TrimSpace(m.Name) == "" || m.QueryIR == nil {
		return core.Model{}, fmt.Errorf("name and queryir are required")
	}
	if err := m.QueryIR.Validate(); err != nil {
		return core.Model{}, err
	}
	m.ID = core.NewID("mdl")
	m.CreatedBy = userID
	m.CreatedAt = time.Now().UTC()
	m.DatabaseID = m.QueryIR.Source.DatabaseID
	if err := s.Models.Create(m); err != nil {
		return core.Model{}, err
	}
	return m, nil
}

func (s Service) ListModels() ([]core.Model, error) {
	if s.Models == nil {
		return []core.Model{}, nil
	}
	return s.Models.List()
}

func (s Service) GetModel(id string) (core.Model, error) {
	return s.Models.ByID(id)
}

func (s Service) CreateMetric(m core.Metric) (core.Metric, error) {
	if strings.TrimSpace(m.Name) == "" || m.Table == "" {
		return core.Metric{}, fmt.Errorf("name and table are required")
	}
	m.ID = core.NewID("met")
	m.CreatedAt = time.Now().UTC()
	if err := s.Metrics.Create(m); err != nil {
		return core.Metric{}, err
	}
	return m, nil
}

func (s Service) ListMetrics() ([]core.Metric, error) {
	if s.Metrics == nil {
		return []core.Metric{}, nil
	}
	return s.Metrics.List()
}

func (s Service) CreateSegment(seg core.Segment) (core.Segment, error) {
	if strings.TrimSpace(seg.Name) == "" || len(seg.Filters) == 0 {
		return core.Segment{}, fmt.Errorf("name and filters are required")
	}
	seg.ID = core.NewID("seg")
	seg.CreatedAt = time.Now().UTC()
	if err := s.Segments.Create(seg); err != nil {
		return core.Segment{}, err
	}
	return seg, nil
}

func (s Service) ListSegments() ([]core.Segment, error) {
	if s.Segments == nil {
		return []core.Segment{}, nil
	}
	return s.Segments.List()
}

func (s Service) CreateGlossary(term core.GlossaryTerm) (core.GlossaryTerm, error) {
	if strings.TrimSpace(term.Name) == "" || strings.TrimSpace(term.Definition) == "" {
		return core.GlossaryTerm{}, fmt.Errorf("name and definition are required")
	}
	term.ID = core.NewID("gls")
	if err := s.Glossary.Create(term); err != nil {
		return core.GlossaryTerm{}, err
	}
	return term, nil
}

func (s Service) ListGlossary() ([]core.GlossaryTerm, error) {
	if s.Glossary == nil {
		return []core.GlossaryTerm{}, nil
	}
	return s.Glossary.List()
}

func (s Service) ImplicitJoin(databaseID, schema, table, targetTable string) (ok bool, join any) {
	if s.Fields == nil {
		return false, nil
	}
	fields, err := s.Fields.ListDatabaseFields(databaseID)
	if err != nil {
		return false, nil
	}
	for _, field := range fields {
		if field.Schema == schema && field.Table == table && field.FKTarget != nil && field.FKTarget.Table == targetTable {
			return true, field
		}
	}
	return false, nil
}
