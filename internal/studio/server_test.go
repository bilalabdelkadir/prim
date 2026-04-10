package studio

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bilalabdelkadir/prim/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testSchema(models ...*schema.Model) *schema.Schema {
	return &schema.Schema{
		Datasource: &schema.Datasource{Provider: "sqlite", URL: "file:test.db"},
		Models:     models,
	}
}

func TestHandleSchema(t *testing.T) {
	s := testSchema(&schema.Model{
		Name: "User",
		Fields: []*schema.Field{
			{Name: "id", Type: schema.FieldTypeInt, Attributes: []*schema.Attribute{{Name: "id"}}},
			{Name: "name", Type: schema.FieldTypeString},
		},
	})
	srv := NewServer(nil, s)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/schema", nil)
	srv.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

	var result schema.Schema
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	require.Len(t, result.Models, 1)
	assert.Equal(t, "User", result.Models[0].Name)
	assert.Len(t, result.Models[0].Fields, 2)
}

func TestHandleTables(t *testing.T) {
	s := testSchema(
		&schema.Model{Name: "User"},
		&schema.Model{Name: "Post"},
	)
	srv := NewServer(nil, s)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/tables", nil)
	srv.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var entries []struct {
		Name      string `json:"name"`
		TableName string `json:"table_name"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &entries))
	require.Len(t, entries, 2)
	assert.Equal(t, "User", entries[0].Name)
	assert.Equal(t, "users", entries[0].TableName)
	assert.Equal(t, "Post", entries[1].Name)
	assert.Equal(t, "posts", entries[1].TableName)
}

func TestHandleTableByName(t *testing.T) {
	s := testSchema(
		&schema.Model{
			Name: "User",
			Fields: []*schema.Field{
				{Name: "id", Type: schema.FieldTypeInt},
				{Name: "email", Type: schema.FieldTypeString, IsOptional: true},
			},
		},
		&schema.Model{Name: "Post"},
	)
	srv := NewServer(nil, s)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/tables/User", nil)
	srv.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var model schema.Model
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &model))
	assert.Equal(t, "User", model.Name)
	assert.Len(t, model.Fields, 2)
}

func TestHandleTableNotFound(t *testing.T) {
	s := testSchema(&schema.Model{Name: "User"})
	srv := NewServer(nil, s)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/tables/Unknown", nil)
	srv.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandleModelFields(t *testing.T) {
	s := testSchema(&schema.Model{
		Name: "User",
		Fields: []*schema.Field{
			{Name: "id", Type: schema.FieldTypeInt, Attributes: []*schema.Attribute{{Name: "id"}}},
			{Name: "email", Type: schema.FieldTypeString},
			{Name: "name", Type: schema.FieldTypeString, IsOptional: true},
			{Name: "posts", Type: schema.FieldType("Post"), IsArray: true},
		},
	})
	srv := NewServer(nil, s)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/models/User/fields", nil)
	srv.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

	var fields []FieldInfo
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &fields))
	require.Len(t, fields, 4)

	assert.Equal(t, "id", fields[0].Name)
	assert.Equal(t, "Int", fields[0].Type)
	assert.True(t, fields[0].IsPrimary)
	assert.False(t, fields[0].IsOptional)

	assert.Equal(t, "email", fields[1].Name)
	assert.Equal(t, "String", fields[1].Type)
	assert.False(t, fields[1].IsOptional)

	assert.Equal(t, "name", fields[2].Name)
	assert.True(t, fields[2].IsOptional)
}

func TestHandleModelFieldsNotFound(t *testing.T) {
	s := testSchema(&schema.Model{Name: "User"})
	srv := NewServer(nil, s)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/models/Unknown/fields", nil)
	srv.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandleModelRelations(t *testing.T) {
	s := testSchema(&schema.Model{
		Name: "User",
		Fields: []*schema.Field{
			{Name: "id", Type: schema.FieldTypeInt},
			{Name: "email", Type: schema.FieldTypeString},
			{Name: "posts", Type: schema.FieldType("Post"), IsArray: true},
			{Name: "profile", Type: schema.FieldType("Profile")},
		},
	})
	srv := NewServer(nil, s)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/models/User/relations", nil)
	srv.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

	var relations []RelationInfo
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &relations))
	require.Len(t, relations, 2)

	assert.Equal(t, "posts", relations[0].Name)
	assert.Equal(t, "Post", relations[0].Model)

	assert.Equal(t, "profile", relations[1].Name)
	assert.Equal(t, "Profile", relations[1].Model)
}

func TestHandleModelRelationsNotFound(t *testing.T) {
	s := testSchema(&schema.Model{Name: "User"})
	srv := NewServer(nil, s)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/models/Unknown/relations", nil)
	srv.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

