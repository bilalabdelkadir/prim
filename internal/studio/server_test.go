package studio

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
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

func TestHandleQueryPreview(t *testing.T) {
	s := testSchema(&schema.Model{
		Name: "User",
		Fields: []*schema.Field{
			{Name: "id", Type: schema.FieldTypeInt},
			{Name: "email", Type: schema.FieldTypeString},
			{Name: "status", Type: schema.FieldTypeString},
		},
	})
	srv := NewServer(nil, s)

	body := QueryRequest{
		Name:      "FindActiveUsers",
		ModelName: "User",
		Operation: "find_many",
		Fields:    []string{"id", "email"},
		Where: []WhereClause{
			{Field: "status", Operator: "eq", ParamName: "status", ParamType: "string"},
		},
		OrderBy: []OrderClause{{Field: "id", Direction: "DESC"}},
		Limit:   10,
	}
	b, err := json.Marshal(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/query/preview", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	srv.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

	var result map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	assert.Contains(t, result, "code")
	assert.Contains(t, result, "structCode")
	assert.Contains(t, result["code"], "FindActiveUsers")
}

func TestHandleQueryPreviewBadRequest(t *testing.T) {
	srv := NewServer(nil, testSchema())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/query/preview", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	srv.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleQuerySave(t *testing.T) {
	s := testSchema(&schema.Model{
		Name: "User",
		Fields: []*schema.Field{
			{Name: "id", Type: schema.FieldTypeInt},
			{Name: "email", Type: schema.FieldTypeString},
		},
	})
	srv := NewServer(nil, s)

	// Create a temp repo file for the save handler to append to.
	tmpFile, err := os.CreateTemp("", "repo_*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	_, err = tmpFile.WriteString("package db\n")
	require.NoError(t, err)
	tmpFile.Close()

	body := QueryRequest{
		Name:       "FindActiveUsers",
		ModelName:  "User",
		Operation:  "find_many",
		Fields:     []string{"id", "email"},
		OutputPath: tmpFile.Name(),
	}
	b, err := json.Marshal(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/query/save", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	srv.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	assert.Equal(t, true, result["success"])
	assert.Contains(t, result["message"], "FindActiveUsers")

	// Verify the method was actually appended to the file.
	content, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)
	assert.Contains(t, string(content), "FindActiveUsers")
}

func TestHandleQuerySaveMissingOutputPath(t *testing.T) {
	srv := NewServer(nil, testSchema())

	body := QueryRequest{
		Name:      "FindActiveUsers",
		ModelName: "User",
	}
	b, err := json.Marshal(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/query/save", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	srv.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestToCodegenDefConversion(t *testing.T) {
	req := &QueryRequest{
		Name:      "FindActiveUsers",
		ModelName: "User",
		Operation: "find_many",
		Fields:    []string{"id", "email", "name"},
		Where: []WhereClause{
			{Field: "status", Operator: "eq", ParamName: "status", ParamType: "string"},
		},
		OrderBy: []OrderClause{
			{Field: "createdAt", Direction: "DESC"},
		},
		Limit: 10,
		Joins: []JoinClause{
			{ModelName: "Post", Fields: []string{"id", "title"}, ForeignKey: "userId", ReferenceKey: "id", Type: "left"},
		},
	}

	// Verify the struct can be marshalled/unmarshalled correctly (round-trip).
	b, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded QueryRequest
	require.NoError(t, json.Unmarshal(b, &decoded))

	assert.Equal(t, req.Name, decoded.Name)
	assert.Equal(t, req.ModelName, decoded.ModelName)
	assert.Equal(t, req.Operation, decoded.Operation)
	assert.Equal(t, req.Fields, decoded.Fields)
	require.Len(t, decoded.Where, 1)
	assert.Equal(t, "status", decoded.Where[0].Field)
	assert.Equal(t, "eq", decoded.Where[0].Operator)
	require.Len(t, decoded.OrderBy, 1)
	assert.Equal(t, "DESC", decoded.OrderBy[0].Direction)
	assert.Equal(t, 10, decoded.Limit)
	require.Len(t, decoded.Joins, 1)
	assert.Equal(t, "Post", decoded.Joins[0].ModelName)
	assert.Equal(t, "left", decoded.Joins[0].Type)
}
