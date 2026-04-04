package codegen

import (
	"testing"

	"github.com/bilalabdelkadir/prim/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func userModel() *schema.Model {
	return &schema.Model{
		Name: "User",
		Fields: []*schema.Field{
			{Name: "id", Type: schema.FieldTypeInt, Attributes: []*schema.Attribute{{Name: "id"}}},
			{Name: "email", Type: schema.FieldTypeString},
			{Name: "name", Type: schema.FieldTypeString, IsOptional: true},
		},
	}
}

func postModel() *schema.Model {
	return &schema.Model{
		Name: "Post",
		Fields: []*schema.Field{
			{Name: "id", Type: schema.FieldTypeInt, Attributes: []*schema.Attribute{{Name: "id"}}},
			{Name: "title", Type: schema.FieldTypeString},
		},
	}
}

func modelWithArray() *schema.Model {
	return &schema.Model{
		Name: "User",
		Fields: []*schema.Field{
			{Name: "id", Type: schema.FieldTypeInt, Attributes: []*schema.Attribute{{Name: "id"}}},
			{Name: "email", Type: schema.FieldTypeString},
			{Name: "posts", Type: "Post", IsArray: true},
		},
	}
}

func TestGenerate_StructFields(t *testing.T) {
	out, err := GenerateModel(userModel())
	require.NoError(t, err)

	assert.Contains(t, out, "type User struct")
	assert.Contains(t, out, "Id int")
	assert.Contains(t, out, "Email string")
	assert.Contains(t, out, "Name *string")
}

func TestGenerate_FindByID(t *testing.T) {
	out, err := GenerateRepository(userModel())
	require.NoError(t, err)

	assert.Contains(t, out, "FindByID")
	assert.Contains(t, out, "SELECT")
	assert.Contains(t, out, "QueryRowContext")
}

func TestGenerate_Create(t *testing.T) {
	out, err := GenerateRepository(postModel())
	require.NoError(t, err)

	assert.Contains(t, out, "INSERT INTO")
	assert.Contains(t, out, "RETURNING")
}

func TestGenerate_Update(t *testing.T) {
	out, err := GenerateRepository(postModel())
	require.NoError(t, err)

	assert.Contains(t, out, "UPDATE")
	assert.Contains(t, out, "SET")
	assert.Contains(t, out, "RETURNING")
}

func TestGenerate_Delete(t *testing.T) {
	out, err := GenerateRepository(postModel())
	require.NoError(t, err)

	assert.Contains(t, out, "DELETE FROM")
}

func TestGenerate_SkipsArrayFields(t *testing.T) {
	structOut, err := GenerateModel(modelWithArray())
	require.NoError(t, err)

	// The struct should not contain the array field.
	assert.NotContains(t, structOut, "Posts")
	assert.NotContains(t, structOut, "Post")
	assert.Contains(t, structOut, "Email string")

	repoOut, err := GenerateRepository(modelWithArray())
	require.NoError(t, err)

	// SQL should not reference the array field.
	assert.NotContains(t, repoOut, "posts")
	assert.NotContains(t, repoOut, "Posts")
}
