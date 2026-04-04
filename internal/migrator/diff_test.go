package migrator

import (
	"testing"

	"github.com/bilalabdelkadir/prim/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiff_AddTable(t *testing.T) {
	next := &schema.Schema{
		Models: []*schema.Model{
			{
				Name: "User",
				Fields: []*schema.Field{
					{Name: "id", Type: schema.FieldTypeInt, Attributes: []*schema.Attribute{{Name: "id"}}},
				},
			},
		},
	}

	ops := Diff(&schema.Schema{}, next)
	require.Len(t, ops, 1)
	assert.Equal(t, OpCreateTable, ops[0].Type)
	assert.Equal(t, "User", ops[0].TableName)
}

func TestDiff_DropTable(t *testing.T) {
	current := &schema.Schema{
		Models: []*schema.Model{
			{
				Name: "User",
				Fields: []*schema.Field{
					{Name: "id", Type: schema.FieldTypeInt},
				},
			},
		},
	}

	ops := Diff(current, &schema.Schema{})
	require.Len(t, ops, 1)
	assert.Equal(t, OpDropTable, ops[0].Type)
	assert.Equal(t, "User", ops[0].TableName)
}

func TestDiff_AddColumn(t *testing.T) {
	current := &schema.Schema{
		Models: []*schema.Model{
			{
				Name: "User",
				Fields: []*schema.Field{
					{Name: "id", Type: schema.FieldTypeInt},
				},
			},
		},
	}
	next := &schema.Schema{
		Models: []*schema.Model{
			{
				Name: "User",
				Fields: []*schema.Field{
					{Name: "id", Type: schema.FieldTypeInt},
					{Name: "email", Type: schema.FieldTypeString},
				},
			},
		},
	}

	ops := Diff(current, next)
	require.Len(t, ops, 1)
	assert.Equal(t, OpAddColumn, ops[0].Type)
	assert.Equal(t, "User", ops[0].TableName)
	assert.Equal(t, "email", ops[0].ColumnName)
}

func TestDiff_DropColumn(t *testing.T) {
	current := &schema.Schema{
		Models: []*schema.Model{
			{
				Name: "User",
				Fields: []*schema.Field{
					{Name: "id", Type: schema.FieldTypeInt},
					{Name: "email", Type: schema.FieldTypeString},
				},
			},
		},
	}
	next := &schema.Schema{
		Models: []*schema.Model{
			{
				Name: "User",
				Fields: []*schema.Field{
					{Name: "id", Type: schema.FieldTypeInt},
				},
			},
		},
	}

	ops := Diff(current, next)
	require.Len(t, ops, 1)
	assert.Equal(t, OpDropColumn, ops[0].Type)
	assert.Equal(t, "User", ops[0].TableName)
	assert.Equal(t, "email", ops[0].ColumnName)
}

func TestDiff_NoChanges(t *testing.T) {
	s := &schema.Schema{
		Models: []*schema.Model{
			{
				Name: "User",
				Fields: []*schema.Field{
					{Name: "id", Type: schema.FieldTypeInt},
					{Name: "email", Type: schema.FieldTypeString},
				},
			},
		},
	}

	ops := Diff(s, s)
	assert.Empty(t, ops)
}

func TestDiff_NilCurrent(t *testing.T) {
	next := &schema.Schema{
		Models: []*schema.Model{
			{
				Name: "Post",
				Fields: []*schema.Field{
					{Name: "id", Type: schema.FieldTypeInt},
				},
			},
		},
	}

	ops := Diff(nil, next)
	require.Len(t, ops, 1)
	assert.Equal(t, OpCreateTable, ops[0].Type)
	assert.Equal(t, "Post", ops[0].TableName)
}
