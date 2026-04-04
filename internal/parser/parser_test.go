package parser

import (
	"os"
	"testing"

	"github.com/bilalabdelkadir/prim/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_SingleModel(t *testing.T) {
	input := `
model User {
  id    Int    @id @default(autoincrement())
  email String @unique
  name  String?
}
`
	s, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, s.Models, 1)

	m := s.Models[0]
	assert.Equal(t, "User", m.Name)
	require.Len(t, m.Fields, 3)

	// id field
	assert.Equal(t, "id", m.Fields[0].Name)
	assert.Equal(t, schema.FieldTypeInt, m.Fields[0].Type)
	assert.False(t, m.Fields[0].IsOptional)
	require.Len(t, m.Fields[0].Attributes, 2)
	assert.Equal(t, "id", m.Fields[0].Attributes[0].Name)
	assert.Equal(t, "default", m.Fields[0].Attributes[1].Name)
	require.Len(t, m.Fields[0].Attributes[1].Args, 1)
	assert.Equal(t, "autoincrement()", m.Fields[0].Attributes[1].Args[0])

	// email field
	assert.Equal(t, "email", m.Fields[1].Name)
	assert.Equal(t, schema.FieldTypeString, m.Fields[1].Type)
	require.Len(t, m.Fields[1].Attributes, 1)
	assert.Equal(t, "unique", m.Fields[1].Attributes[0].Name)

	// name field
	assert.Equal(t, "name", m.Fields[2].Name)
	assert.Equal(t, schema.FieldTypeString, m.Fields[2].Type)
	assert.True(t, m.Fields[2].IsOptional)
}

func TestParser_Datasource(t *testing.T) {
	input := `
datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}
`
	s, err := Parse(input)
	require.NoError(t, err)
	require.NotNil(t, s.Datasource)
	assert.Equal(t, "postgresql", s.Datasource.Provider)
	assert.Equal(t, "DATABASE_URL", s.Datasource.URL)
}

func TestParser_RelationField(t *testing.T) {
	input := `
model Post {
  id       Int    @id @default(autoincrement())
  title    String
  authorId Int
  author   User   @relation(fields: [authorId], references: [id])
}
`
	s, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, s.Models, 1)

	m := s.Models[0]
	assert.Equal(t, "Post", m.Name)

	// The author field should be present with type "User".
	authorField := m.Fields[3]
	assert.Equal(t, "author", authorField.Name)
	assert.Equal(t, schema.FieldType("User"), authorField.Type)
	require.Len(t, authorField.Attributes, 1)
	assert.Equal(t, "relation", authorField.Attributes[0].Name)
	require.Len(t, authorField.Attributes[0].Args, 2)
}

func TestParser_MultipleModels(t *testing.T) {
	input := `
model User {
  id   Int    @id
  name String
}

model Post {
  id    Int    @id
  title String
}
`
	s, err := Parse(input)
	require.NoError(t, err)
	require.Len(t, s.Models, 2)
	assert.Equal(t, "User", s.Models[0].Name)
	assert.Equal(t, "Post", s.Models[1].Name)
}

func TestParser_InvalidSchema(t *testing.T) {
	input := `model { }`
	_, err := Parse(input)
	assert.Error(t, err, "missing model name should produce an error")
}

func TestParser_FullSchema(t *testing.T) {
	data, err := os.ReadFile("../../testdata/valid/basic.prisma")
	require.NoError(t, err)

	s, err := Parse(string(data))
	require.NoError(t, err)
	require.NotNil(t, s.Datasource)
	assert.Equal(t, "postgresql", s.Datasource.Provider)
	require.Len(t, s.Models, 2)
	assert.Equal(t, "User", s.Models[0].Name)
	assert.Equal(t, "Post", s.Models[1].Name)
}
