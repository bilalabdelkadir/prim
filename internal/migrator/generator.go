package migrator

import (
	"strings"

	"github.com/bilalabdelkadir/prim/internal/schema"
)

// tableName converts a model name to a SQL table name (lowercase + "s").
func tableName(model string) string {
	return strings.ToLower(model) + "s"
}

// hasAttr reports whether the field has an attribute with the given name.
func hasAttr(f *schema.Field, name string) bool {
	for _, a := range f.Attributes {
		if a.Name == name {
			return true
		}
	}
	return false
}

// goTypeToSQL maps a schema field to its PostgreSQL column type.
func goTypeToSQL(f *schema.Field) string {
	switch f.Type {
	case schema.FieldTypeInt:
		if hasAttr(f, "id") {
			return "SERIAL"
		}
		return "INTEGER"
	case schema.FieldTypeString:
		return "TEXT"
	case schema.FieldTypeBool:
		return "BOOLEAN"
	case schema.FieldTypeFloat:
		return "DOUBLE PRECISION"
	case schema.FieldTypeDateTime:
		return "TIMESTAMP WITH TIME ZONE"
	default:
		return "TEXT"
	}
}

// columnDef builds a single column definition for a CREATE TABLE statement.
func columnDef(f *schema.Field) string {
	var b strings.Builder
	b.WriteString("\"")
	b.WriteString(f.Name)
	b.WriteString("\" ")
	b.WriteString(goTypeToSQL(f))
	if hasAttr(f, "id") {
		b.WriteString(" PRIMARY KEY")
	}
	if !f.IsOptional && !hasAttr(f, "id") {
		b.WriteString(" NOT NULL")
	}
	return b.String()
}

// Generate produces a SQL migration script for the given operations.
// The schema s is used to look up full model definitions for CREATE TABLE ops.
func Generate(ops []MigrationOp, s *schema.Schema) string {
	// Index models for O(1) lookup.
	models := make(map[string]*schema.Model, len(s.Models))
	for _, m := range s.Models {
		models[m.Name] = m
	}

	var b strings.Builder

	for _, op := range ops {
		tbl := tableName(op.TableName)

		switch op.Type {
		case OpCreateTable:
			m, ok := models[op.TableName]
			if !ok {
				continue
			}
			b.WriteString("CREATE TABLE \"")
			b.WriteString(tbl)
			b.WriteString("\" (\n")

			first := true
			for _, f := range m.Fields {
				if f.IsArray || f.IsRelation() {
					continue
				}
				if !first {
					b.WriteString(",\n")
				}
				b.WriteString("  ")
				b.WriteString(columnDef(f))
				first = false
			}
			b.WriteString("\n);\n")

		case OpDropTable:
			b.WriteString("DROP TABLE IF EXISTS \"")
			b.WriteString(tbl)
			b.WriteString("\";\n")

		case OpAddColumn:
			b.WriteString("ALTER TABLE \"")
			b.WriteString(tbl)
			b.WriteString("\" ADD COLUMN ")
			b.WriteString(columnDef(op.Field))
			b.WriteString(";\n")

		case OpDropColumn:
			b.WriteString("ALTER TABLE \"")
			b.WriteString(tbl)
			b.WriteString("\" DROP COLUMN \"")
			b.WriteString(op.ColumnName)
			b.WriteString("\";\n")
		}
	}

	return b.String()
}
