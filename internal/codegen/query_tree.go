package codegen

// PrimQuery describes a Prisma-style nested query.
type PrimQuery struct {
	Name      string        // method name, e.g. "FindActiveUsersWithPosts"
	ModelName string        // root model
	Operation QueryOp       // find_one, find_many, count, create, update, delete
	Select    []string      // fields to select (empty = all scalar)
	Where     []WhereClause // reuse existing type
	OrderBy   []OrderClause // reuse existing type
	Limit     int
	Skip      int
	Include   []IncludeNode // nested relations
	Data      []DataField   // fields to set for create/update operations
}

// DataField describes a field value to set in create/update.
type DataField struct {
	FieldName string // e.g. "email"
	ParamName string // Go parameter name, e.g. "email"
	ParamType string // Go type, e.g. "string"
}

// IncludeNode represents a nested relation include with its own query options.
type IncludeNode struct {
	RelationName string        // field name in parent, e.g. "posts"
	ModelName    string        // target model, e.g. "Post"
	IsArray      bool          // true for one-to-many
	ForeignKey   string        // FK column on the child side, e.g. "authorId"
	ReferenceKey string        // PK on the parent side, e.g. "id"
	Select       []string      // fields from this model
	Where        []WhereClause
	OrderBy      []OrderClause
	Limit        int
	Include      []IncludeNode // deeper nesting (recursive)
	CreateData   []DataField   // fields to set when creating nested records
}
