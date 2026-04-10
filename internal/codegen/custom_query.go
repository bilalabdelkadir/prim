package codegen

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bilalabdelkadir/prim/internal/schema"
)

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

// WhereClause represents a single filter condition.
type WhereClause struct {
	Field     string
	Operator  string
	ParamName string
	ParamType string
}

// OrderClause represents a sorting directive.
type OrderClause struct {
	Field     string
	Direction string
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
// file. Creates the file and parent directories if they don't exist.
func AppendToRepoFile(filePath string, code string) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("codegen: create dir: %w", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			data = []byte("package db\n")
		} else {
			return fmt.Errorf("codegen: read repo file: %w", err)
		}
	}

	content := string(data)
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
