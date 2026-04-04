package migrator

import "github.com/bilalabdelkadir/prim/internal/schema"

// OpType describes the kind of schema change a MigrationOp represents.
type OpType string

const (
	OpCreateTable OpType = "CREATE_TABLE"
	OpDropTable   OpType = "DROP_TABLE"
	OpAddColumn   OpType = "ADD_COLUMN"
	OpDropColumn  OpType = "DROP_COLUMN"
	OpAlterColumn OpType = "ALTER_COLUMN"
)

// MigrationOp is a single migration operation produced by diffing two schemas.
type MigrationOp struct {
	Type       OpType
	TableName  string
	ColumnName string
	Field      *schema.Field
}

// Diff compares current and next schemas and returns the list of operations
// needed to migrate from current to next. A nil current is treated as an empty
// schema (i.e. a fresh database).
func Diff(current, next *schema.Schema) []MigrationOp {
	if current == nil {
		current = &schema.Schema{}
	}
	if next == nil {
		next = &schema.Schema{}
	}

	// Index current models by name for O(1) lookup.
	curModels := make(map[string]*schema.Model, len(current.Models))
	for _, m := range current.Models {
		curModels[m.Name] = m
	}

	// Index next models by name.
	nextModels := make(map[string]*schema.Model, len(next.Models))
	for _, m := range next.Models {
		nextModels[m.Name] = m
	}

	var ops []MigrationOp

	// Detect new tables and column-level changes for existing tables.
	for _, nm := range next.Models {
		cm, exists := curModels[nm.Name]
		if !exists {
			ops = append(ops, MigrationOp{
				Type:      OpCreateTable,
				TableName: nm.Name,
			})
			continue
		}

		// Table exists in both — diff columns.
		curFields := make(map[string]*schema.Field, len(cm.Fields))
		for _, f := range cm.Fields {
			if f.IsArray || f.IsRelation() {
				continue
			}
			curFields[f.Name] = f
		}

		nextFields := make(map[string]*schema.Field, len(nm.Fields))
		for _, f := range nm.Fields {
			if f.IsArray || f.IsRelation() {
				continue
			}
			nextFields[f.Name] = f
		}

		// New columns.
		for _, f := range nm.Fields {
			if f.IsArray || f.IsRelation() {
				continue
			}
			if _, found := curFields[f.Name]; !found {
				ops = append(ops, MigrationOp{
					Type:       OpAddColumn,
					TableName:  nm.Name,
					ColumnName: f.Name,
					Field:      f,
				})
			}
		}

		// Dropped columns.
		for _, f := range cm.Fields {
			if f.IsArray || f.IsRelation() {
				continue
			}
			if _, found := nextFields[f.Name]; !found {
				ops = append(ops, MigrationOp{
					Type:       OpDropColumn,
					TableName:  nm.Name,
					ColumnName: f.Name,
					Field:      f,
				})
			}
		}
	}

	// Detect dropped tables.
	for _, cm := range current.Models {
		if _, exists := nextModels[cm.Name]; !exists {
			ops = append(ops, MigrationOp{
				Type:      OpDropTable,
				TableName: cm.Name,
			})
		}
	}

	return ops
}
