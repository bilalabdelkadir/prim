package schema

// Schema is the top-level AST node produced by parsing a .prisma file.
type Schema struct {
	Datasource *Datasource
	Models     []*Model
}

// Datasource holds the database connection configuration.
type Datasource struct {
	Provider string
	URL      string
}

// Model represents a Prisma model block.
type Model struct {
	Name   string
	Fields []*Field
}

// Field represents a single field inside a model.
type Field struct {
	Name       string
	Type       FieldType
	IsOptional bool
	IsArray    bool
	Attributes []*Attribute
}

// FieldType is the declared type of a field (may be a built-in scalar or a
// relation model name).
type FieldType string

const (
	FieldTypeInt      FieldType = "Int"
	FieldTypeString   FieldType = "String"
	FieldTypeBool     FieldType = "Boolean"
	FieldTypeFloat    FieldType = "Float"
	FieldTypeDateTime FieldType = "DateTime"
)

// Attribute represents a field-level attribute such as @id or @default(...).
type Attribute struct {
	Name string
	Args []string
}

// IsScalar reports whether the field type is a built-in scalar (not a relation).
func (t FieldType) IsScalar() bool {
	switch t {
	case FieldTypeInt, FieldTypeString, FieldTypeBool, FieldTypeFloat, FieldTypeDateTime:
		return true
	}
	return false
}

// IsRelation reports whether the field represents a relation to another model.
func (f *Field) IsRelation() bool {
	return !f.Type.IsScalar()
}
