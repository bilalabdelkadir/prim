package codegen

import (
	"strings"
	"testing"

	"github.com/bilalabdelkadir/prim/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func primTestSchema() *schema.Schema {
	return &schema.Schema{
		Models: []*schema.Model{
			{
				Name: "User",
				Fields: []*schema.Field{
					{Name: "id", Type: schema.FieldTypeInt, Attributes: []*schema.Attribute{{Name: "id"}}},
					{Name: "email", Type: schema.FieldTypeString},
					{Name: "name", Type: schema.FieldTypeString, IsOptional: true},
					{Name: "status", Type: schema.FieldTypeString},
					{Name: "createdAt", Type: schema.FieldTypeDateTime},
					{Name: "posts", Type: "Post", IsArray: true},
				},
			},
			{
				Name: "Post",
				Fields: []*schema.Field{
					{Name: "id", Type: schema.FieldTypeInt, Attributes: []*schema.Attribute{{Name: "id"}}},
					{Name: "title", Type: schema.FieldTypeString},
					{Name: "content", Type: schema.FieldTypeString, IsOptional: true},
					{Name: "published", Type: schema.FieldTypeBool},
					{Name: "authorId", Type: schema.FieldTypeInt},
					{Name: "createdAt", Type: schema.FieldTypeDateTime},
					{Name: "author", Type: "User"},
					{Name: "comments", Type: "Comment", IsArray: true},
				},
			},
			{
				Name: "Comment",
				Fields: []*schema.Field{
					{Name: "id", Type: schema.FieldTypeInt, Attributes: []*schema.Attribute{{Name: "id"}}},
					{Name: "body", Type: schema.FieldTypeString},
					{Name: "postId", Type: schema.FieldTypeInt},
					{Name: "createdAt", Type: schema.FieldTypeDateTime},
					{Name: "post", Type: "Post"},
				},
			},
		},
	}
}

func TestPrimQuery_SimpleFind(t *testing.T) {
	s := primTestSchema()
	q := &PrimQuery{
		Name:      "FindActiveUsers",
		ModelName: "User",
		Operation: QueryOpFindMany,
		Where: []WhereClause{
			{Field: "status", Operator: "eq", ParamName: "status", ParamType: "string"},
		},
		OrderBy: []OrderClause{
			{Field: "createdAt", Direction: "DESC"},
		},
		Limit: 10,
	}

	code, structs, err := GeneratePrimQuery(q, s)
	require.NoError(t, err)

	// Should have result struct.
	assert.Contains(t, structs, "type FindActiveUsersResult struct")
	assert.Contains(t, structs, "Id int")
	assert.Contains(t, structs, "Email string")

	// Should have method.
	assert.Contains(t, code, "func (r *UserRepository) FindActiveUsers(ctx context.Context, status string)")
	assert.Contains(t, code, `FROM "users"`)
	assert.Contains(t, code, `WHERE "status" = $1`)
	assert.Contains(t, code, `ORDER BY "createdAt" DESC`)
	assert.Contains(t, code, "LIMIT 10")

	// Should NOT have any pq.Array or include logic.
	assert.NotContains(t, code, "pq.Array")
	assert.NotContains(t, code, "ANY")
}

func TestPrimQuery_WithOneInclude(t *testing.T) {
	s := primTestSchema()
	q := &PrimQuery{
		Name:      "FindUsersWithPosts",
		ModelName: "User",
		Operation: QueryOpFindMany,
		Include: []IncludeNode{
			{
				RelationName: "posts",
				ModelName:    "Post",
				IsArray:      true,
				ForeignKey:   "authorId",
				ReferenceKey: "id",
			},
		},
	}

	code, structs, err := GeneratePrimQuery(q, s)
	require.NoError(t, err)

	// Should have parent and child structs.
	assert.Contains(t, structs, "type FindUsersWithPostsResult struct")
	assert.Contains(t, structs, "type FindUsersWithPostsPostsResult struct")
	assert.Contains(t, structs, "Posts []FindUsersWithPostsPostsResult")

	// Should have two queries.
	assert.Contains(t, code, `FROM "users"`)
	assert.Contains(t, code, `FROM "posts" WHERE "authorId" = ANY($1)`)
	assert.Contains(t, code, "pq.Array(")
}

func TestPrimQuery_NestedIncludes(t *testing.T) {
	s := primTestSchema()
	q := &PrimQuery{
		Name:      "FindUsersWithPostsAndComments",
		ModelName: "User",
		Operation: QueryOpFindMany,
		Include: []IncludeNode{
			{
				RelationName: "posts",
				ModelName:    "Post",
				IsArray:      true,
				ForeignKey:   "authorId",
				ReferenceKey: "id",
				Include: []IncludeNode{
					{
						RelationName: "comments",
						ModelName:    "Comment",
						IsArray:      true,
						ForeignKey:   "postId",
						ReferenceKey: "id",
					},
				},
			},
		},
	}

	code, structs, err := GeneratePrimQuery(q, s)
	require.NoError(t, err)

	// Should have three struct levels.
	assert.Contains(t, structs, "type FindUsersWithPostsAndCommentsResult struct")
	assert.Contains(t, structs, "type FindUsersWithPostsAndCommentsPostsResult struct")
	assert.Contains(t, structs, "type FindUsersWithPostsAndCommentsPostsCommentsResult struct")

	// Should have three queries: users, posts, comments.
	assert.Contains(t, code, `FROM "users"`)
	assert.Contains(t, code, `FROM "posts" WHERE "authorId" = ANY($1)`)
	assert.Contains(t, code, `FROM "comments" WHERE "postId" = ANY($1)`)
}

func TestPrimQuery_FindOne(t *testing.T) {
	s := primTestSchema()
	q := &PrimQuery{
		Name:      "FindUserWithPosts",
		ModelName: "User",
		Operation: QueryOpFindOne,
		Where: []WhereClause{
			{Field: "id", Operator: "eq", ParamName: "id", ParamType: "int"},
		},
		Include: []IncludeNode{
			{
				RelationName: "posts",
				ModelName:    "Post",
				IsArray:      true,
				ForeignKey:   "authorId",
				ReferenceKey: "id",
			},
		},
	}

	code, structs, err := GeneratePrimQuery(q, s)
	require.NoError(t, err)

	assert.Contains(t, structs, "type FindUserWithPostsResult struct")
	// Should return single pointer.
	assert.Contains(t, code, "(*FindUserWithPostsResult, error)")
	assert.Contains(t, code, "QueryRowContext")
	// Should still fetch includes.
	assert.Contains(t, code, `FROM "posts" WHERE "authorId" = ANY($1)`)
}

func TestPrimQuery_Count(t *testing.T) {
	s := primTestSchema()
	q := &PrimQuery{
		Name:      "CountActiveUsers",
		ModelName: "User",
		Operation: QueryOpCount,
		Where: []WhereClause{
			{Field: "status", Operator: "eq", ParamName: "status", ParamType: "string"},
		},
	}

	code, _, err := GeneratePrimQuery(q, s)
	require.NoError(t, err)

	assert.Contains(t, code, "(int, error)")
	assert.Contains(t, code, "COUNT(*)")
	assert.Contains(t, code, `FROM "users"`)
	assert.Contains(t, code, `WHERE "status" = $1`)
	// No include logic.
	assert.NotContains(t, code, "pq.Array")
}

func TestPrimQuery_ResultStructs(t *testing.T) {
	s := primTestSchema()
	q := &PrimQuery{
		Name:      "FindUsers",
		ModelName: "User",
		Operation: QueryOpFindMany,
		Include: []IncludeNode{
			{
				RelationName: "posts",
				ModelName:    "Post",
				IsArray:      true,
				ForeignKey:   "authorId",
				ReferenceKey: "id",
			},
		},
	}

	_, structs, err := GeneratePrimQuery(q, s)
	require.NoError(t, err)

	// Parent struct should have slice for posts.
	assert.Contains(t, structs, "Posts []FindUsersPostsResult")

	// Child struct should have scalar fields from Post.
	assert.Contains(t, structs, "Title string")
	assert.Contains(t, structs, "Content *string")
	assert.Contains(t, structs, "Published bool")
	assert.Contains(t, structs, "AuthorId int")
}

func TestPrimQuery_SelectFields(t *testing.T) {
	s := primTestSchema()
	q := &PrimQuery{
		Name:      "FindUserEmails",
		ModelName: "User",
		Operation: QueryOpFindMany,
		Select:    []string{"id", "email"},
	}

	code, structs, err := GeneratePrimQuery(q, s)
	require.NoError(t, err)

	// Struct should only have selected fields (+ id always).
	assert.Contains(t, structs, "Id int")
	assert.Contains(t, structs, "Email string")
	assert.NotContains(t, structs, "Name ")
	assert.NotContains(t, structs, "Status ")

	// SQL should only select those columns.
	assert.Contains(t, code, `"id"`)
	assert.Contains(t, code, `"email"`)
	assert.NotContains(t, code, `"name"`)
	assert.NotContains(t, code, `"status"`)
}

func TestPrimQuery_ModelNotFound(t *testing.T) {
	s := primTestSchema()
	q := &PrimQuery{
		Name:      "FindBad",
		ModelName: "NonExistent",
		Operation: QueryOpFindMany,
	}

	_, _, err := GeneratePrimQuery(q, s)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "NonExistent")
}

func TestPrimQuery_IncludeModelNotFound(t *testing.T) {
	s := primTestSchema()
	q := &PrimQuery{
		Name:      "FindBad",
		ModelName: "User",
		Operation: QueryOpFindMany,
		Include: []IncludeNode{
			{
				RelationName: "widgets",
				ModelName:    "Widget",
				IsArray:      true,
				ForeignKey:   "userId",
				ReferenceKey: "id",
			},
		},
	}

	_, _, err := GeneratePrimQuery(q, s)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Widget")
}

func TestPrimQuery_IncludeWithWhere(t *testing.T) {
	s := primTestSchema()
	q := &PrimQuery{
		Name:      "FindUsersWithPublishedPosts",
		ModelName: "User",
		Operation: QueryOpFindMany,
		Include: []IncludeNode{
			{
				RelationName: "posts",
				ModelName:    "Post",
				IsArray:      true,
				ForeignKey:   "authorId",
				ReferenceKey: "id",
				Where: []WhereClause{
					{Field: "published", Operator: "eq", ParamName: "published", ParamType: "bool"},
				},
				OrderBy: []OrderClause{
					{Field: "createdAt", Direction: "DESC"},
				},
			},
		},
	}

	code, _, err := GeneratePrimQuery(q, s)
	require.NoError(t, err)

	assert.Contains(t, code, `"published" = $2`)
	assert.Contains(t, code, `ORDER BY "createdAt" DESC`)

	// The method should accept published param.
	assert.Contains(t, code, "published bool")
}

func TestPrimQuery_SelectFieldsOnInclude(t *testing.T) {
	s := primTestSchema()
	q := &PrimQuery{
		Name:      "FindUsersWithPostTitles",
		ModelName: "User",
		Operation: QueryOpFindMany,
		Include: []IncludeNode{
			{
				RelationName: "posts",
				ModelName:    "Post",
				IsArray:      true,
				ForeignKey:   "authorId",
				ReferenceKey: "id",
				Select:       []string{"title"},
			},
		},
	}

	_, structs, err := GeneratePrimQuery(q, s)
	require.NoError(t, err)

	// The child struct should have title, id (always), and authorId (FK always).
	lines := strings.Split(structs, "\n")
	var childStructLines []string
	inChild := false
	for _, line := range lines {
		if strings.Contains(line, "FindUsersWithPostTitlesPostsResult struct") {
			inChild = true
			continue
		}
		if inChild {
			if strings.Contains(line, "}") {
				break
			}
			childStructLines = append(childStructLines, strings.TrimSpace(line))
		}
	}

	// Should have Id, Title, AuthorId.
	childStruct := strings.Join(childStructLines, " ")
	assert.Contains(t, childStruct, "Id int")
	assert.Contains(t, childStruct, "Title string")
	assert.Contains(t, childStruct, "AuthorId int")
	// Should NOT have Content or Published.
	assert.NotContains(t, childStruct, "Content")
	assert.NotContains(t, childStruct, "Published")
}

func TestPrimQuery_SimpleCreate(t *testing.T) {
	s := primTestSchema()
	q := &PrimQuery{
		Name:      "CreateUser",
		ModelName: "User",
		Operation: QueryOpCreate,
		Data: []DataField{
			{FieldName: "email", ParamName: "email", ParamType: "string"},
			{FieldName: "name", ParamName: "name", ParamType: "*string"},
		},
	}

	code, structs, err := GeneratePrimQuery(q, s)
	require.NoError(t, err)

	// Should have result struct.
	assert.Contains(t, structs, "type CreateUserResult struct")
	assert.Contains(t, structs, "Id int")
	assert.Contains(t, structs, "Email string")

	// Should have method with data params.
	assert.Contains(t, code, "func (r *UserRepository) CreateUser(ctx context.Context, email string, name *string)")
	assert.Contains(t, code, "(*CreateUserResult, error)")

	// Should have INSERT INTO with correct columns.
	assert.Contains(t, code, `INSERT INTO "users"`)
	assert.Contains(t, code, `"email"`)
	assert.Contains(t, code, `"name"`)
	assert.Contains(t, code, "VALUES ($1, $2)")
	assert.Contains(t, code, "RETURNING")

	// Should have Scan for returned fields.
	assert.Contains(t, code, "Scan(")
}

func TestPrimQuery_CreateWithNestedCreate(t *testing.T) {
	s := primTestSchema()
	q := &PrimQuery{
		Name:      "CreateUserWithPost",
		ModelName: "User",
		Operation: QueryOpCreate,
		Data: []DataField{
			{FieldName: "email", ParamName: "email", ParamType: "string"},
		},
		Include: []IncludeNode{
			{
				RelationName: "posts",
				ModelName:    "Post",
				IsArray:      true,
				ForeignKey:   "authorId",
				ReferenceKey: "id",
				CreateData: []DataField{
					{FieldName: "title", ParamName: "postTitle", ParamType: "string"},
				},
			},
		},
	}

	code, structs, err := GeneratePrimQuery(q, s)
	require.NoError(t, err)

	// Should have both parent and child structs.
	assert.Contains(t, structs, "type CreateUserWithPostResult struct")
	assert.Contains(t, structs, "type CreateUserWithPostPostsResult struct")

	// Should have two INSERTs.
	assert.Contains(t, code, `INSERT INTO "users"`)
	assert.Contains(t, code, `INSERT INTO "posts"`)

	// The child INSERT should reference the parent FK.
	assert.Contains(t, code, `"authorId"`)
	assert.Contains(t, code, "u.Id")

	// Method signature should include nested create params.
	assert.Contains(t, code, "postTitle string")
}

func TestPrimQuery_SimpleUpdate(t *testing.T) {
	s := primTestSchema()
	q := &PrimQuery{
		Name:      "UpdateUserEmail",
		ModelName: "User",
		Operation: QueryOpUpdate,
		Data: []DataField{
			{FieldName: "email", ParamName: "email", ParamType: "string"},
		},
		Where: []WhereClause{
			{Field: "id", Operator: "eq", ParamName: "id", ParamType: "int"},
		},
	}

	code, structs, err := GeneratePrimQuery(q, s)
	require.NoError(t, err)

	// Should have result struct.
	assert.Contains(t, structs, "type UpdateUserEmailResult struct")

	// Should have method.
	assert.Contains(t, code, "func (r *UserRepository) UpdateUserEmail(ctx context.Context, email string, id int)")
	assert.Contains(t, code, "(*UpdateUserEmailResult, error)")

	// Should have UPDATE SET with WHERE and RETURNING.
	assert.Contains(t, code, `UPDATE "users" SET "email" = $1`)
	assert.Contains(t, code, `WHERE "id" = $2`)
	assert.Contains(t, code, "RETURNING")
}

func TestPrimQuery_SimpleDelete(t *testing.T) {
	s := primTestSchema()
	q := &PrimQuery{
		Name:      "DeleteUser",
		ModelName: "User",
		Operation: QueryOpDelete,
		Where: []WhereClause{
			{Field: "id", Operator: "eq", ParamName: "id", ParamType: "int"},
		},
	}

	code, structs, err := GeneratePrimQuery(q, s)
	require.NoError(t, err)

	// Delete should have no result struct.
	assert.Empty(t, structs)

	// Should return error only.
	assert.Contains(t, code, "func (r *UserRepository) DeleteUser(ctx context.Context, id int) error")

	// Should have DELETE FROM with WHERE.
	assert.Contains(t, code, `DELETE FROM "users"`)
	assert.Contains(t, code, `WHERE "id" = $1`)

	// Should NOT have any Scan or result struct references.
	assert.NotContains(t, code, "Scan")
	assert.NotContains(t, code, "Result")
}
