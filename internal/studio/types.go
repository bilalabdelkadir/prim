package studio

// QueryRequest is the JSON request body for query builder endpoints.
type QueryRequest struct {
	Name       string        `json:"name"`
	ModelName  string        `json:"modelName"`
	Operation  string        `json:"operation"`
	Fields     []string      `json:"fields"`
	Where      []WhereClause `json:"where"`
	OrderBy    []OrderClause `json:"orderBy"`
	Limit      int           `json:"limit"`
	Joins      []JoinClause  `json:"joins"`
	OutputPath string        `json:"outputPath,omitempty"`
}

// WhereClause represents a single WHERE condition in a query definition.
type WhereClause struct {
	Field     string `json:"field"`
	Operator  string `json:"operator"`
	ParamName string `json:"paramName"`
	ParamType string `json:"paramType"`
}

// OrderClause represents an ORDER BY clause in a query definition.
type OrderClause struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

// JoinClause represents a JOIN in a query definition.
type JoinClause struct {
	ModelName    string   `json:"modelName"`
	Fields       []string `json:"fields"`
	ForeignKey   string   `json:"foreignKey"`
	ReferenceKey string   `json:"referenceKey"`
	Type         string   `json:"type"`
}

// FieldInfo describes a single model field for the UI field picker.
type FieldInfo struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	ColumnName   string   `json:"column_name"`
	IsOptional   bool     `json:"is_optional"`
	IsPrimary    bool     `json:"is_primary"`
	IsUnique     bool     `json:"is_unique"`
	DefaultValue string   `json:"default_value"`
	Attributes   []string `json:"attributes"`
}

// RelationInfo describes a relation field for the UI join builder.
type RelationInfo struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Model      string `json:"model"`
	ForeignKey string `json:"foreign_key"`
	References string `json:"references"`
}

// DataFieldRequest represents a field value to set in create/update operations.
type DataFieldRequest struct {
	FieldName string `json:"fieldName"`
	ParamName string `json:"paramName"`
	ParamType string `json:"paramType"`
}

// PrimQueryRequest is the JSON request body for the nested query builder endpoint.
type PrimQueryRequest struct {
	Name       string               `json:"name"`
	ModelName  string               `json:"model"`
	Operation  string               `json:"operation"`
	Select     []string             `json:"select"`
	Where      []WhereClause        `json:"where"`
	OrderBy    []OrderClause        `json:"orderBy"`
	Limit      int                  `json:"limit"`
	Skip       int                  `json:"skip"`
	Include    []IncludeNodeRequest `json:"include"`
	Data       []DataFieldRequest   `json:"data"`
	OutputPath string               `json:"outputPath,omitempty"`
}

// IncludeNodeRequest represents a nested relation include in the API request.
type IncludeNodeRequest struct {
	RelationName string               `json:"relationName"`
	ModelName    string               `json:"modelName"`
	IsArray      bool                 `json:"isArray"`
	ForeignKey   string               `json:"foreignKey"`
	ReferenceKey string               `json:"referenceKey"`
	Select       []string             `json:"select"`
	Where        []WhereClause        `json:"where"`
	OrderBy      []OrderClause        `json:"orderBy"`
	Limit        int                  `json:"limit"`
	Include      []IncludeNodeRequest `json:"include"`
	CreateData   []DataFieldRequest   `json:"createData"`
}
