package migrator

import (
	"testing"

	"github.com/bilalabdelkadir/prim/internal/schema"
	"github.com/stretchr/testify/assert"
)

func TestGenerate_CreateTable(t *testing.T) {
	s := &schema.Schema{
		Models: []*schema.Model{
			{
				Name: "User",
				Fields: []*schema.Field{
					{Name: "id", Type: schema.FieldTypeInt, Attributes: []*schema.Attribute{{Name: "id"}}},
					{Name: "email", Type: schema.FieldTypeString},
					{Name: "active", Type: schema.FieldTypeBool},
				},
			},
		},
	}
	ops := []MigrationOp{
		{Type: OpCreateTable, TableName: "User"},
	}

	sql := Generate(ops, s)
	assert.Contains(t, sql, `CREATE TABLE "users"`)
	assert.Contains(t, sql, `"id" SERIAL PRIMARY KEY`)
	assert.Contains(t, sql, `"email" TEXT NOT NULL`)
	assert.Contains(t, sql, `"active" BOOLEAN NOT NULL`)
}

func TestGenerate_AddColumn(t *testing.T) {
	s := &schema.Schema{}
	field := &schema.Field{Name: "email", Type: schema.FieldTypeString}
	ops := []MigrationOp{
		{Type: OpAddColumn, TableName: "User", ColumnName: "email", Field: field},
	}

	sql := Generate(ops, s)
	assert.Contains(t, sql, `ALTER TABLE "users" ADD COLUMN "email" TEXT NOT NULL`)
}

func TestGenerate_DropTable(t *testing.T) {
	s := &schema.Schema{}
	ops := []MigrationOp{
		{Type: OpDropTable, TableName: "User"},
	}

	sql := Generate(ops, s)
	assert.Contains(t, sql, `DROP TABLE IF EXISTS "users"`)
}

func TestGenerate_DropColumn(t *testing.T) {
	s := &schema.Schema{}
	ops := []MigrationOp{
		{Type: OpDropColumn, TableName: "User", ColumnName: "email"},
	}

	sql := Generate(ops, s)
	assert.Contains(t, sql, `ALTER TABLE "users" DROP COLUMN "email"`)
}

func TestGenerate_MultipleOps(t *testing.T) {
	s := &schema.Schema{
		Models: []*schema.Model{
			{
				Name: "Post",
				Fields: []*schema.Field{
					{Name: "id", Type: schema.FieldTypeInt, Attributes: []*schema.Attribute{{Name: "id"}}},
					{Name: "title", Type: schema.FieldTypeString},
				},
			},
		},
	}
	emailField := &schema.Field{Name: "email", Type: schema.FieldTypeString}
	ops := []MigrationOp{
		{Type: OpCreateTable, TableName: "Post"},
		{Type: OpAddColumn, TableName: "User", ColumnName: "email", Field: emailField},
		{Type: OpDropColumn, TableName: "User", ColumnName: "age"},
		{Type: OpDropTable, TableName: "Session"},
	}

	sql := Generate(ops, s)
	assert.Contains(t, sql, `CREATE TABLE "posts"`)
	assert.Contains(t, sql, `"title" TEXT NOT NULL`)
	assert.Contains(t, sql, `ALTER TABLE "users" ADD COLUMN "email" TEXT NOT NULL`)
	assert.Contains(t, sql, `ALTER TABLE "users" DROP COLUMN "age"`)
	assert.Contains(t, sql, `DROP TABLE IF EXISTS "sessions"`)
}
