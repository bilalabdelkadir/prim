package codegen

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bilalabdelkadir/prim/internal/schema"
)

// QueryDefinition describes a custom query built in the studio UI.
type QueryDefinition struct {
	Name      string        // method name, e.g. "FindUserWithPosts"
	ModelName string        // base model, e.g. "User"
	Operation QueryOp       // find_many, find_one, count, aggregate
	Fields    []string      // fields to select from base model
	Where     []WhereClause // filter conditions
	OrderBy   []OrderClause // sorting
	Limit     int           // 0 means no limit
	Joins     []JoinClause  // related models to join
}

// QueryOp represents the type of query operation.
type QueryOp string

const (
	QueryOpFindMany QueryOp = "find_many"
	QueryOpFindOne  QueryOp = "find_one"
	QueryOpCount    QueryOp = "count"
	QueryOpCreate   QueryOp = "create"
	QueryOpUpdate   QueryOp = "update"
	QueryOpDelete   QueryOp = "delete"
)

// WhereClause represents a single filter condition in a custom query.
type WhereClause struct {
	Field     string // e.g. "email"
	Operator  string // eq, neq, gt, lt, gte, lte, like, in, is_null
	ParamName string // Go parameter name
	ParamType string // Go type for the parameter
}

// OrderClause represents a sorting directive.
type OrderClause struct {
	Field     string
	Direction string // "ASC" or "DESC"
}

// JoinClause represents a related model to join.
type JoinClause struct {
	ModelName    string   // e.g. "Post"
	Fields       []string // fields to select from joined model
	ForeignKey   string   // e.g. "authorId"
	ReferenceKey string   // e.g. "id"
	Type         string   // "inner", "left"
}

// sqlOperator maps a WhereClause.Operator string to its SQL equivalent.
func sqlOperator(op string) string {
	switch op {
	case "eq":
		return "="
	case "neq":
		return "!="
	case "gt":
		return ">"
	case "lt":
		return "<"
	case "gte":
		return ">="
	case "lte":
		return "<="
	case "like":
		return "LIKE"
	case "in":
		return "IN"
	case "is_null":
		return "IS NULL"
	default:
		return "="
	}
}

// findModel looks up a model by name in the schema.
func findModel(s *schema.Schema, name string) *schema.Model {
	for _, m := range s.Models {
		if m.Name == name {
			return m
		}
	}
	return nil
}

// GenerateJoinResultStruct generates a combined result struct when joins are
// present. For example, a User query joining Post produces UserWithPosts.
func GenerateJoinResultStruct(def *QueryDefinition, s *schema.Schema) (string, error) {
	if len(def.Joins) == 0 {
		return "", nil
	}

	baseModel := findModel(s, def.ModelName)
	if baseModel == nil {
		return "", fmt.Errorf("codegen: model %q not found in schema", def.ModelName)
	}

	var b strings.Builder
	b.Grow(512)

	// Build struct name from joins.
	structName := def.ModelName
	for _, j := range def.Joins {
		structName += "With" + j.ModelName + "s"
	}

	b.WriteString("type ")
	b.WriteString(structName)
	b.WriteString(" struct {\n")

	// Base model fields.
	selectedBase := makeFieldSet(def.Fields)
	for _, f := range baseModel.Fields {
		if !isColumnField(f) {
			continue
		}
		if len(selectedBase) > 0 && !selectedBase[f.Name] {
			continue
		}
		b.WriteString("\t")
		b.WriteString(goName(f.Name))
		b.WriteString(" ")
		b.WriteString(goType(f))
		b.WriteString("\n")
	}

	// Joined model fields (prefixed with model name).
	for _, j := range def.Joins {
		joinModel := findModel(s, j.ModelName)
		if joinModel == nil {
			return "", fmt.Errorf("codegen: joined model %q not found in schema", j.ModelName)
		}
		selectedJoin := makeFieldSet(j.Fields)
		for _, f := range joinModel.Fields {
			if !isColumnField(f) {
				continue
			}
			if len(selectedJoin) > 0 && !selectedJoin[f.Name] {
				continue
			}
			b.WriteString("\t")
			b.WriteString(j.ModelName)
			b.WriteString(goName(f.Name))
			b.WriteString(" ")
			b.WriteString(goType(f))
			b.WriteString("\n")
		}
	}

	b.WriteString("}\n")
	return b.String(), nil
}

// GenerateCustomQuery generates a Go method for the given query definition.
func GenerateCustomQuery(def *QueryDefinition, s *schema.Schema) (string, error) {
	baseModel := findModel(s, def.ModelName)
	if baseModel == nil {
		return "", fmt.Errorf("codegen: model %q not found in schema", def.ModelName)
	}

	var b strings.Builder
	b.Grow(1024)

	hasJoins := len(def.Joins) > 0

	// Determine return type and struct name.
	structName := def.ModelName
	if hasJoins {
		for _, j := range def.Joins {
			structName += "With" + j.ModelName + "s"
		}
	}

	// Write method signature.
	b.WriteString("func (r *")
	b.WriteString(def.ModelName)
	b.WriteString("Repository) ")
	b.WriteString(def.Name)
	b.WriteString("(ctx context.Context")
	for _, w := range def.Where {
		if w.Operator == "is_null" {
			continue
		}
		b.WriteString(", ")
		b.WriteString(w.ParamName)
		b.WriteString(" ")
		b.WriteString(w.ParamType)
	}
	b.WriteString(") ")

	switch def.Operation {
	case QueryOpFindOne:
		b.WriteString("(*")
		b.WriteString(structName)
		b.WriteString(", error)")
	case QueryOpFindMany:
		b.WriteString("([]*")
		b.WriteString(structName)
		b.WriteString(", error)")
	case QueryOpCount:
		b.WriteString("(int, error)")
	}

	b.WriteString(" {\n")

	// Build SQL query.
	baseTable := tableName(def.ModelName)
	baseAlias := "t0"

	var sqlBuf strings.Builder
	sqlBuf.Grow(256)

	switch def.Operation {
	case QueryOpCount:
		sqlBuf.WriteString("SELECT COUNT(*) FROM \"")
		sqlBuf.WriteString(baseTable)
		sqlBuf.WriteString("\"")
		if hasJoins {
			sqlBuf.WriteString(" ")
			sqlBuf.WriteString(baseAlias)
		}
	default:
		sqlBuf.WriteString("SELECT ")
		// Collect columns.
		cols := buildSelectColumns(def, baseModel, s, hasJoins, baseAlias)
		sqlBuf.WriteString(cols)
		sqlBuf.WriteString(" FROM \"")
		sqlBuf.WriteString(baseTable)
		sqlBuf.WriteString("\"")
		if hasJoins {
			sqlBuf.WriteString(" ")
			sqlBuf.WriteString(baseAlias)
		}
	}

	// JOINs.
	for i, j := range def.Joins {
		joinAlias := "t" + strconv.Itoa(i+1)
		joinTable := tableName(j.ModelName)
		joinType := "INNER"
		if j.Type == "left" {
			joinType = "LEFT"
		}
		sqlBuf.WriteString(" ")
		sqlBuf.WriteString(joinType)
		sqlBuf.WriteString(" JOIN \"")
		sqlBuf.WriteString(joinTable)
		sqlBuf.WriteString("\" ")
		sqlBuf.WriteString(joinAlias)
		sqlBuf.WriteString(" ON ")
		sqlBuf.WriteString(joinAlias)
		sqlBuf.WriteString(".\"")
		sqlBuf.WriteString(j.ForeignKey)
		sqlBuf.WriteString("\" = ")
		sqlBuf.WriteString(baseAlias)
		sqlBuf.WriteString(".\"")
		sqlBuf.WriteString(j.ReferenceKey)
		sqlBuf.WriteString("\"")
	}

	// WHERE clauses.
	paramIdx := 1
	if len(def.Where) > 0 {
		sqlBuf.WriteString(" WHERE ")
		for i, w := range def.Where {
			if i > 0 {
				sqlBuf.WriteString(" AND ")
			}
			if hasJoins {
				sqlBuf.WriteString(baseAlias)
				sqlBuf.WriteString(".")
			}
			sqlBuf.WriteString("\"")
			sqlBuf.WriteString(w.Field)
			sqlBuf.WriteString("\" ")
			sqlBuf.WriteString(sqlOperator(w.Operator))
			if w.Operator != "is_null" {
				sqlBuf.WriteString(" $")
				sqlBuf.WriteString(strconv.Itoa(paramIdx))
				paramIdx++
			}
		}
	}

	// ORDER BY.
	if len(def.OrderBy) > 0 && def.Operation != QueryOpCount {
		sqlBuf.WriteString(" ORDER BY ")
		for i, o := range def.OrderBy {
			if i > 0 {
				sqlBuf.WriteString(", ")
			}
			if hasJoins {
				sqlBuf.WriteString(baseAlias)
				sqlBuf.WriteString(".")
			}
			sqlBuf.WriteString("\"")
			sqlBuf.WriteString(o.Field)
			sqlBuf.WriteString("\" ")
			sqlBuf.WriteString(o.Direction)
		}
	}

	// LIMIT.
	if def.Limit > 0 && def.Operation != QueryOpCount {
		sqlBuf.WriteString(" LIMIT ")
		sqlBuf.WriteString(strconv.Itoa(def.Limit))
	}

	sql := sqlBuf.String()

	// Build parameter args string for the Go call.
	var args []string
	for _, w := range def.Where {
		if w.Operator == "is_null" {
			continue
		}
		args = append(args, w.ParamName)
	}

	// Generate method body.
	switch def.Operation {
	case QueryOpCount:
		b.WriteString("\tvar count int\n")
		b.WriteString("\terr := r.db.QueryRowContext(ctx,\n")
		b.WriteString("\t\t`")
		b.WriteString(sql)
		b.WriteString("`,\n")
		if len(args) > 0 {
			b.WriteString("\t\t")
			b.WriteString(strings.Join(args, ", "))
			b.WriteString(",\n")
		}
		b.WriteString("\t).Scan(&count)\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn 0, err\n")
		b.WriteString("\t}\n")
		b.WriteString("\treturn count, nil\n")

	case QueryOpFindOne:
		b.WriteString("\tu := &")
		b.WriteString(structName)
		b.WriteString("{}\n")
		b.WriteString("\terr := r.db.QueryRowContext(ctx,\n")
		b.WriteString("\t\t`")
		b.WriteString(sql)
		b.WriteString("`,\n")
		if len(args) > 0 {
			b.WriteString("\t\t")
			b.WriteString(strings.Join(args, ", "))
			b.WriteString(",\n")
		}
		b.WriteString("\t).Scan(")
		b.WriteString(buildScanArgs(def, baseModel, s, hasJoins))
		b.WriteString(")\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn nil, err\n")
		b.WriteString("\t}\n")
		b.WriteString("\treturn u, nil\n")

	case QueryOpFindMany:
		b.WriteString("\trows, err := r.db.QueryContext(ctx,\n")
		b.WriteString("\t\t`")
		b.WriteString(sql)
		b.WriteString("`,\n")
		if len(args) > 0 {
			b.WriteString("\t\t")
			b.WriteString(strings.Join(args, ", "))
			b.WriteString(",\n")
		}
		b.WriteString("\t)\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn nil, err\n")
		b.WriteString("\t}\n")
		b.WriteString("\tdefer rows.Close()\n")
		b.WriteString("\tvar result []*")
		b.WriteString(structName)
		b.WriteString("\n")
		b.WriteString("\tfor rows.Next() {\n")
		b.WriteString("\t\tu := &")
		b.WriteString(structName)
		b.WriteString("{}\n")
		b.WriteString("\t\tif err := rows.Scan(")
		b.WriteString(buildScanArgs(def, baseModel, s, hasJoins))
		b.WriteString("); err != nil {\n")
		b.WriteString("\t\t\treturn nil, err\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t\tresult = append(result, u)\n")
		b.WriteString("\t}\n")
		b.WriteString("\tif err := rows.Err(); err != nil {\n")
		b.WriteString("\t\treturn nil, err\n")
		b.WriteString("\t}\n")
		b.WriteString("\treturn result, nil\n")
	}

	b.WriteString("}\n")
	return b.String(), nil
}

// buildSelectColumns builds the SELECT column list.
func buildSelectColumns(def *QueryDefinition, baseModel *schema.Model, s *schema.Schema, hasJoins bool, baseAlias string) string {
	var cols []string
	selectedBase := makeFieldSet(def.Fields)

	for _, f := range baseModel.Fields {
		if !isColumnField(f) {
			continue
		}
		if len(selectedBase) > 0 && !selectedBase[f.Name] {
			continue
		}
		if hasJoins {
			cols = append(cols, baseAlias+".\""+f.Name+"\"")
		} else {
			cols = append(cols, "\""+f.Name+"\"")
		}
	}

	for i, j := range def.Joins {
		joinModel := findModel(s, j.ModelName)
		if joinModel == nil {
			continue
		}
		joinAlias := "t" + strconv.Itoa(i+1)
		selectedJoin := makeFieldSet(j.Fields)
		for _, f := range joinModel.Fields {
			if !isColumnField(f) {
				continue
			}
			if len(selectedJoin) > 0 && !selectedJoin[f.Name] {
				continue
			}
			cols = append(cols, joinAlias+".\""+f.Name+"\"")
		}
	}

	return strings.Join(cols, ", ")
}

// buildScanArgs builds the Scan arguments for a query result.
func buildScanArgs(def *QueryDefinition, baseModel *schema.Model, s *schema.Schema, hasJoins bool) string {
	var args []string
	selectedBase := makeFieldSet(def.Fields)

	for _, f := range baseModel.Fields {
		if !isColumnField(f) {
			continue
		}
		if len(selectedBase) > 0 && !selectedBase[f.Name] {
			continue
		}
		args = append(args, "&u."+goName(f.Name))
	}

	for _, j := range def.Joins {
		joinModel := findModel(s, j.ModelName)
		if joinModel == nil {
			continue
		}
		selectedJoin := makeFieldSet(j.Fields)
		for _, f := range joinModel.Fields {
			if !isColumnField(f) {
				continue
			}
			if len(selectedJoin) > 0 && !selectedJoin[f.Name] {
				continue
			}
			args = append(args, "&u."+j.ModelName+goName(f.Name))
		}
	}

	return strings.Join(args, ", ")
}

// makeFieldSet builds a set from a slice of field names. Returns nil if the
// slice is empty (meaning "select all").
func makeFieldSet(fields []string) map[string]bool {
	if len(fields) == 0 {
		return nil
	}
	m := make(map[string]bool, len(fields))
	for _, f := range fields {
		m[f] = true
	}
	return m
}

// AppendToRepoFile appends generated method code to an existing Go repository
// file. It inserts the code at the end of the file with proper spacing.
func AppendToRepoFile(filePath string, code string) error {
	// Create parent directories if they don't exist.
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("codegen: create dir: %w", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist — create it with a package header.
			data = []byte("package db\n")
		} else {
			return fmt.Errorf("codegen: read repo file: %w", err)
		}
	}

	content := string(data)

	// Ensure proper spacing: two newlines before the new method.
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if !strings.HasSuffix(content, "\n\n") {
		content += "\n"
	}
	content += code

	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("codegen: create repo file: %w", err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	if _, err := w.WriteString(content); err != nil {
		return fmt.Errorf("codegen: write repo file: %w", err)
	}
	return w.Flush()
}
