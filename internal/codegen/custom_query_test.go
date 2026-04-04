package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bilalabdelkadir/prim/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testSchema() *schema.Schema {
	return &schema.Schema{
		Models: []*schema.Model{
			{
				Name: "User",
				Fields: []*schema.Field{
					{Name: "id", Type: schema.FieldTypeInt, Attributes: []*schema.Attribute{{Name: "id"}}},
					{Name: "email", Type: schema.FieldTypeString},
					{Name: "name", Type: schema.FieldTypeString, IsOptional: true},
				},
			},
			{
				Name: "Post",
				Fields: []*schema.Field{
					{Name: "id", Type: schema.FieldTypeInt, Attributes: []*schema.Attribute{{Name: "id"}}},
					{Name: "title", Type: schema.FieldTypeString},
					{Name: "authorId", Type: schema.FieldTypeInt},
				},
			},
		},
	}
}

func TestGenerateCustomQuery_FindOne(t *testing.T) {
	def := &QueryDefinition{
		Name:      "FindByEmail",
		ModelName: "User",
		Operation: QueryOpFindOne,
		Where: []WhereClause{
			{Field: "email", Operator: "eq", ParamName: "email", ParamType: "string"},
		},
	}

	code, err := GenerateCustomQuery(def, testSchema())
	require.NoError(t, err)

	// Method signature.
	assert.Contains(t, code, "func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*User, error)")
	// SQL.
	assert.Contains(t, code, "SELECT")
	assert.Contains(t, code, `"email"`)
	assert.Contains(t, code, "WHERE")
	assert.Contains(t, code, "$1")
	// Uses QueryRowContext for find_one.
	assert.Contains(t, code, "QueryRowContext")
	// Returns pointer.
	assert.Contains(t, code, "return u, nil")
}

func TestGenerateCustomQuery_FindMany(t *testing.T) {
	def := &QueryDefinition{
		Name:      "FindRecentUsers",
		ModelName: "User",
		Operation: QueryOpFindMany,
		OrderBy: []OrderClause{
			{Field: "id", Direction: "DESC"},
		},
		Limit: 10,
	}

	code, err := GenerateCustomQuery(def, testSchema())
	require.NoError(t, err)

	// Returns slice.
	assert.Contains(t, code, "[]*User")
	// Uses QueryContext for find_many.
	assert.Contains(t, code, "QueryContext")
	// Has scan loop.
	assert.Contains(t, code, "rows.Next()")
	// ORDER BY and LIMIT.
	assert.Contains(t, code, "ORDER BY")
	assert.Contains(t, code, "DESC")
	assert.Contains(t, code, "LIMIT 10")
}

func TestGenerateCustomQuery_WithJoin(t *testing.T) {
	def := &QueryDefinition{
		Name:      "FindUserWithPosts",
		ModelName: "User",
		Operation: QueryOpFindOne,
		Where: []WhereClause{
			{Field: "id", Operator: "eq", ParamName: "userID", ParamType: "int"},
		},
		Joins: []JoinClause{
			{
				ModelName:    "Post",
				ForeignKey:   "authorId",
				ReferenceKey: "id",
				Type:         "left",
			},
		},
	}

	s := testSchema()

	// Test result struct generation.
	structCode, err := GenerateJoinResultStruct(def, s)
	require.NoError(t, err)
	assert.Contains(t, structCode, "type UserWithPosts struct")
	assert.Contains(t, structCode, "Email string")
	assert.Contains(t, structCode, "PostTitle string")
	assert.Contains(t, structCode, "PostAuthorId int")

	// Test query generation.
	code, err := GenerateCustomQuery(def, s)
	require.NoError(t, err)

	assert.Contains(t, code, "LEFT JOIN")
	assert.Contains(t, code, `"posts"`)
	assert.Contains(t, code, "t0")
	assert.Contains(t, code, "t1")
	assert.Contains(t, code, "UserWithPosts")
	assert.Contains(t, code, "ON t1.\"authorId\" = t0.\"id\"")
}

func TestGenerateCustomQuery_Count(t *testing.T) {
	def := &QueryDefinition{
		Name:      "CountActiveUsers",
		ModelName: "User",
		Operation: QueryOpCount,
		Where: []WhereClause{
			{Field: "email", Operator: "neq", ParamName: "excludeEmail", ParamType: "string"},
		},
	}

	code, err := GenerateCustomQuery(def, testSchema())
	require.NoError(t, err)

	// Returns int.
	assert.Contains(t, code, "(int, error)")
	// COUNT(*).
	assert.Contains(t, code, "COUNT(*)")
	// Scan into int.
	assert.Contains(t, code, "&count")
	assert.Contains(t, code, "return count, nil")
}

func TestAppendToRepoFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "user_repo.go")

	existing := `package db

type UserRepository struct {
	db *sql.DB
}

func (r *UserRepository) FindByID(ctx context.Context, id int) (*User, error) {
	return nil, nil
}
`
	err := os.WriteFile(filePath, []byte(existing), 0644)
	require.NoError(t, err)

	newMethod := `func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	return nil, nil
}
`

	err = AppendToRepoFile(filePath, newMethod)
	require.NoError(t, err)

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	content := string(data)

	// Original content is preserved.
	assert.Contains(t, content, "FindByID")
	// New method is appended.
	assert.Contains(t, content, "FindByEmail")
	// Proper spacing between methods.
	assert.Contains(t, content, "}\n\nfunc (r *UserRepository) FindByEmail")
}

func TestGenerateJoinResultStruct_NoJoins(t *testing.T) {
	def := &QueryDefinition{
		Name:      "FindByEmail",
		ModelName: "User",
		Operation: QueryOpFindOne,
	}

	code, err := GenerateJoinResultStruct(def, testSchema())
	require.NoError(t, err)
	assert.Empty(t, code)
}

func TestGenerateCustomQuery_ModelNotFound(t *testing.T) {
	def := &QueryDefinition{
		Name:      "FindStuff",
		ModelName: "NonExistent",
		Operation: QueryOpFindOne,
	}

	_, err := GenerateCustomQuery(def, testSchema())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
