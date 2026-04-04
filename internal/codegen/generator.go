package codegen

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/bilalabdelkadir/prim/internal/schema"
)

// goName capitalizes the first letter of s.
func goName(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// goType maps a schema field to its Go type string.
func goType(f *schema.Field) string {
	var base string
	switch f.Type {
	case schema.FieldTypeInt:
		base = "int"
	case schema.FieldTypeString:
		base = "string"
	case schema.FieldTypeBool:
		base = "bool"
	case schema.FieldTypeFloat:
		base = "float64"
	case schema.FieldTypeDateTime:
		base = "time.Time"
	default:
		base = string(f.Type)
	}
	if f.IsOptional {
		return "*" + base
	}
	return base
}

// tableName returns the lowercased model name with an "s" suffix.
func tableName(name string) string {
	return strings.ToLower(name) + "s"
}

// isColumnField returns true if the field maps to a database column.
func isColumnField(f *schema.Field) bool {
	return !f.IsArray && !f.IsRelation()
}

// isInsertableField returns true if the field should appear in INSERT statements
// (not an id field and not an array/relation field).
func isInsertableField(f *schema.Field) bool {
	if f.IsArray || f.IsRelation() {
		return false
	}
	for _, attr := range f.Attributes {
		if attr.Name == "id" {
			return false
		}
	}
	return f.Name != "id"
}

func selectCols(fields []*schema.Field) string {
	var cols []string
	for _, f := range fields {
		if isColumnField(f) {
			cols = append(cols, `"`+f.Name+`"`)
		}
	}
	return strings.Join(cols, ", ")
}

func scanArgs(fields []*schema.Field) string {
	var args []string
	for _, f := range fields {
		if isColumnField(f) {
			args = append(args, "&u."+goName(f.Name))
		}
	}
	return strings.Join(args, ", ")
}

func createArgs(fields []*schema.Field) string {
	var args []string
	for _, f := range fields {
		if isInsertableField(f) {
			args = append(args, f.Name+" "+goType(f))
		}
	}
	return strings.Join(args, ", ")
}

func createArgNames(fields []*schema.Field) string {
	var names []string
	for _, f := range fields {
		if isInsertableField(f) {
			names = append(names, f.Name)
		}
	}
	return strings.Join(names, ", ")
}

func insertCols(fields []*schema.Field) string {
	var cols []string
	for _, f := range fields {
		if isInsertableField(f) {
			cols = append(cols, `"`+f.Name+`"`)
		}
	}
	return strings.Join(cols, ", ")
}

func insertPlaceholders(fields []*schema.Field) string {
	var phs []string
	n := 1
	for _, f := range fields {
		if isInsertableField(f) {
			phs = append(phs, fmt.Sprintf("$%d", n))
			n++
		}
	}
	return strings.Join(phs, ", ")
}

func updateSetCols(fields []*schema.Field) string {
	var parts []string
	n := 1
	for _, f := range fields {
		if isInsertableField(f) {
			parts = append(parts, fmt.Sprintf(`"%s"=$%d`, f.Name, n))
			n++
		}
	}
	return strings.Join(parts, ", ")
}

func updateArgs(fields []*schema.Field) string {
	var names []string
	for _, f := range fields {
		if isInsertableField(f) {
			names = append(names, f.Name)
		}
	}
	names = append(names, "id")
	return strings.Join(names, ", ")
}

// updateIDPlaceholder returns the $N placeholder index for the id argument in
// an UPDATE query (one past the number of insertable fields).
func updateIDPlaceholder(fields []*schema.Field) int {
	n := 0
	for _, f := range fields {
		if isInsertableField(f) {
			n++
		}
	}
	return n + 1
}

// needsTimeImport reports whether any field in the model uses time.Time.
func needsTimeImport(fields []*schema.Field) bool {
	for _, f := range fields {
		if f.IsArray || f.IsRelation() {
			continue
		}
		if f.Type == schema.FieldTypeDateTime {
			return true
		}
	}
	return false
}

var funcMap = template.FuncMap{
	"goName":              goName,
	"goType":              goType,
	"tableName":           tableName,
	"selectCols":          selectCols,
	"scanArgs":            scanArgs,
	"createArgs":          createArgs,
	"createArgNames":      createArgNames,
	"insertCols":          insertCols,
	"insertPlaceholders":  insertPlaceholders,
	"updateSetCols":       updateSetCols,
	"updateArgs":          updateArgs,
	"updateIDPlaceholder": updateIDPlaceholder,
	"needsTimeImport":     needsTimeImport,
}

const modelTmpl = `package db
{{ if needsTimeImport .Fields }}
import "time"
{{ end }}
type {{ .Name }} struct {
{{- range .Fields }}
{{- if and (not .IsArray) (not .IsRelation) }}
	{{ goName .Name }} {{ goType . }}
{{- end }}
{{- end }}
}
`

const repoTmpl = `package db

import (
	"context"
	"database/sql"
{{- if needsTimeImport .Fields }}
	"time"
{{- end }}
)

type {{ .Name }}Repository struct {
	db *sql.DB
}

func New{{ .Name }}Repository(db *sql.DB) *{{ .Name }}Repository {
	return &{{ .Name }}Repository{db: db}
}

func (r *{{ .Name }}Repository) FindByID(ctx context.Context, id int) (*{{ .Name }}, error) {
	u := &{{ .Name }}{}
	err := r.db.QueryRowContext(ctx,
		` + "`" + `SELECT {{ selectCols .Fields }} FROM "{{ tableName .Name }}" WHERE "id"=$1` + "`" + `,
		id,
	).Scan({{ scanArgs .Fields }})
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *{{ .Name }}Repository) Create(ctx context.Context, {{ createArgs .Fields }}) (*{{ .Name }}, error) {
	u := &{{ .Name }}{}
	err := r.db.QueryRowContext(ctx,
		` + "`" + `INSERT INTO "{{ tableName .Name }}" ({{ insertCols .Fields }}) VALUES ({{ insertPlaceholders .Fields }}) RETURNING {{ selectCols .Fields }}` + "`" + `,
		{{ createArgNames .Fields }},
	).Scan({{ scanArgs .Fields }})
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *{{ .Name }}Repository) Update(ctx context.Context, id int, {{ createArgs .Fields }}) (*{{ .Name }}, error) {
	u := &{{ .Name }}{}
	err := r.db.QueryRowContext(ctx,
		` + "`" + `UPDATE "{{ tableName .Name }}" SET {{ updateSetCols .Fields }} WHERE "id"=${{ updateIDPlaceholder .Fields }} RETURNING {{ selectCols .Fields }}` + "`" + `,
		{{ updateArgs .Fields }},
	).Scan({{ scanArgs .Fields }})
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *{{ .Name }}Repository) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx,
		` + "`" + `DELETE FROM "{{ tableName .Name }}" WHERE "id"=$1` + "`" + `,
		id,
	)
	return err
}
`

var (
	parsedModelTmpl = template.Must(template.New("model").Funcs(funcMap).Parse(modelTmpl))
	parsedRepoTmpl  = template.Must(template.New("repo").Funcs(funcMap).Parse(repoTmpl))
)

// GenerateModel produces a Go struct definition for the given schema model.
func GenerateModel(m *schema.Model) (string, error) {
	var buf bytes.Buffer
	if err := parsedModelTmpl.Execute(&buf, m); err != nil {
		return "", fmt.Errorf("codegen: model template: %w", err)
	}
	return buf.String(), nil
}

// GenerateRepository produces a CRUD repository for the given schema model.
func GenerateRepository(m *schema.Model) (string, error) {
	var buf bytes.Buffer
	if err := parsedRepoTmpl.Execute(&buf, m); err != nil {
		return "", fmt.Errorf("codegen: repo template: %w", err)
	}
	return buf.String(), nil
}
