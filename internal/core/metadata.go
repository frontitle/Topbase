package core

type TableAnnotation struct {
	DisplayName string            `json:"display_name"`
	Description string            `json:"description"`
	UserNote    string            `json:"user_note"`
	Hidden      bool              `json:"hidden"`
	FieldTypes  map[string]string `json:"field_types"`
}

type MetadataStore interface {
	GetTableAnnotation(databaseID, schema, table string) (TableAnnotation, error)
	SaveTableAnnotation(databaseID, schema, table string, annotation TableAnnotation) error
}
