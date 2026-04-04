package studio

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/bilalabdelkadir/prim/internal/codegen"
	"github.com/bilalabdelkadir/prim/internal/schema"
)

// Server is the HTTP server for the PrismaGo studio web UI.
type Server struct {
	db     *sql.DB
	schema *schema.Schema
	mux    *http.ServeMux
}

// NewServer creates a Server and registers all routes.
func NewServer(db *sql.DB, s *schema.Schema) *Server {
	srv := &Server{
		db:     db,
		schema: s,
		mux:    http.NewServeMux(),
	}
	srv.mux.HandleFunc("GET /api/schema", srv.handleSchema)
	srv.mux.HandleFunc("GET /api/tables", srv.handleTables)
	srv.mux.HandleFunc("GET /api/tables/{name}", srv.handleTableByName)
	srv.mux.HandleFunc("POST /api/query", srv.handleQuery)
	srv.mux.HandleFunc("POST /api/query/preview", srv.handleQueryPreview)
	srv.mux.HandleFunc("POST /api/query/save", srv.handleQuerySave)
	srv.mux.HandleFunc("GET /api/models/{name}/fields", srv.handleModelFields)
	srv.mux.HandleFunc("GET /api/models/{name}/relations", srv.handleModelRelations)
	srv.mux.HandleFunc("POST /api/query/build", srv.handleQueryBuild)
	srv.mux.HandleFunc("POST /api/query/build/save", srv.handleQueryBuildSave)
	return srv
}

// Start begins listening on the given port.
func (s *Server) Start(port int) error {
	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("prim studio running at http://localhost:%d\n", port)
	return http.ListenAndServe(addr, s)
}

// ServeHTTP delegates to the internal mux, applying CORS middleware.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	s.mux.ServeHTTP(w, r)
}

func (s *Server) handleSchema(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(s.schema); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleTables(w http.ResponseWriter, r *http.Request) {
	type modelEntry struct {
		Name      string `json:"name"`
		TableName string `json:"table_name"`
	}
	entries := make([]modelEntry, len(s.schema.Models))
	for i, m := range s.schema.Models {
		entries[i] = modelEntry{
			Name:      m.Name,
			TableName: strings.ToLower(m.Name) + "s",
		}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(entries); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleTableByName(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	for _, m := range s.schema.Models {
		if strings.EqualFold(m.Name, name) {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(m); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
	}
	http.Error(w, "model not found", http.StatusNotFound)
}

type queryRequest struct {
	SQL string `json:"sql"`
}

func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	var req queryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.SQL == "" {
		http.Error(w, "sql field is required", http.StatusBadRequest)
		return
	}

	rows, err := s.db.Query(req.SQL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var results []map[string]interface{}

	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		row := make(map[string]interface{}, len(cols))
		for i, col := range cols {
			v := vals[i]
			// Convert []byte to string for JSON compatibility.
			if b, ok := v.([]byte); ok {
				v = string(b)
			}
			row[col] = v
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if results == nil {
		results = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) findModel(name string) *schema.Model {
	for _, m := range s.schema.Models {
		if strings.EqualFold(m.Name, name) {
			return m
		}
	}
	return nil
}

func (s *Server) handleModelFields(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	m := s.findModel(name)
	if m == nil {
		http.Error(w, "model not found", http.StatusNotFound)
		return
	}

	var fields []FieldInfo
	for _, f := range m.Fields {
		fi := FieldInfo{
			Name:       f.Name,
			Type:       string(f.Type),
			ColumnName: f.Name,
			IsOptional: f.IsOptional,
		}
		var attrs []string
		for _, a := range f.Attributes {
			attrs = append(attrs, "@"+a.Name)
			if a.Name == "id" {
				fi.IsPrimary = true
			}
			if a.Name == "unique" {
				fi.IsUnique = true
			}
			if a.Name == "default" && len(a.Args) > 0 {
				fi.DefaultValue = a.Args[0]
			}
		}
		if attrs == nil {
			attrs = []string{}
		}
		fi.Attributes = attrs
		fields = append(fields, fi)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(fields); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleModelRelations(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	m := s.findModel(name)
	if m == nil {
		http.Error(w, "model not found", http.StatusNotFound)
		return
	}

	var relations []RelationInfo
	for _, f := range m.Fields {
		if f.IsRelation() || f.IsArray {
			ri := RelationInfo{
				Name:  f.Name,
				Type:  string(f.Type),
				Model: string(f.Type),
			}
			// Extract foreign_key and references from @relation attribute args.
			for _, a := range f.Attributes {
				if a.Name == "relation" {
					for _, arg := range a.Args {
						arg = strings.TrimSpace(arg)
						if strings.HasPrefix(arg, "fields:") {
							fk := strings.TrimPrefix(arg, "fields:")
							fk = strings.Trim(fk, " []")
							ri.ForeignKey = fk
						}
						if strings.HasPrefix(arg, "references:") {
							ref := strings.TrimPrefix(arg, "references:")
							ref = strings.Trim(ref, " []")
							ri.References = ref
						}
					}
				}
			}
			relations = append(relations, ri)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(relations); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleQueryPreview(w http.ResponseWriter, r *http.Request) {
	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.ModelName == "" {
		http.Error(w, "name and modelName are required", http.StatusBadRequest)
		return
	}

	def := toCodegenDef(&req)
	code, err := codegen.GenerateCustomQuery(def, s.schema)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var structCode string
	if len(req.Joins) > 0 {
		structCode, err = codegen.GenerateJoinResultStruct(def, s.schema)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	resp := map[string]string{"code": code, "structCode": structCode}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleQuerySave(w http.ResponseWriter, r *http.Request) {
	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.ModelName == "" || req.OutputPath == "" {
		http.Error(w, "name, modelName, and outputPath are required", http.StatusBadRequest)
		return
	}

	def := toCodegenDef(&req)
	code, err := codegen.GenerateCustomQuery(def, s.schema)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If there are joins, prepend the result struct.
	if len(req.Joins) > 0 {
		structCode, err := codegen.GenerateJoinResultStruct(def, s.schema)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		code = structCode + "\n" + code
	}

	if err := codegen.AppendToRepoFile(req.OutputPath, code); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Added %s to %s", req.Name, req.OutputPath),
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleQueryBuild(w http.ResponseWriter, r *http.Request) {
	var req PrimQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.ModelName == "" {
		http.Error(w, "name and modelName are required", http.StatusBadRequest)
		return
	}

	pq := toPrimQuery(&req)
	code, structs, err := codegen.GeneratePrimQuery(pq, s.schema)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]string{"code": code, "structs": structs}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleQueryBuildSave(w http.ResponseWriter, r *http.Request) {
	var req PrimQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.ModelName == "" || req.OutputPath == "" {
		http.Error(w, "name, modelName, and outputPath are required", http.StatusBadRequest)
		return
	}

	pq := toPrimQuery(&req)
	code, structs, err := codegen.GeneratePrimQuery(pq, s.schema)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fullCode := structs + "\n" + code
	if err := codegen.AppendToRepoFile(req.OutputPath, fullCode); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Added %s to %s", req.Name, req.OutputPath),
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func mapOperation(op string) codegen.QueryOp {
	switch op {
	case "findOne":
		return codegen.QueryOpFindOne
	case "findMany":
		return codegen.QueryOpFindMany
	case "count":
		return codegen.QueryOpCount
	default:
		return codegen.QueryOp(op)
	}
}

func toPrimQuery(req *PrimQueryRequest) *codegen.PrimQuery {
	var wheres []codegen.WhereClause
	for _, w := range req.Where {
		wheres = append(wheres, codegen.WhereClause{
			Field:     w.Field,
			Operator:  w.Operator,
			ParamName: w.ParamName,
			ParamType: w.ParamType,
		})
	}
	var orders []codegen.OrderClause
	for _, o := range req.OrderBy {
		orders = append(orders, codegen.OrderClause{
			Field:     o.Field,
			Direction: o.Direction,
		})
	}
	return &codegen.PrimQuery{
		Name:      req.Name,
		ModelName: req.ModelName,
		Operation: mapOperation(req.Operation),
		Select:    req.Select,
		Where:     wheres,
		OrderBy:   orders,
		Limit:     req.Limit,
		Skip:      req.Skip,
		Include:   convertIncludes(req.Include),
	}
}

func convertIncludes(nodes []IncludeNodeRequest) []codegen.IncludeNode {
	var result []codegen.IncludeNode
	for _, n := range nodes {
		var wheres []codegen.WhereClause
		for _, w := range n.Where {
			wheres = append(wheres, codegen.WhereClause{
				Field:     w.Field,
				Operator:  w.Operator,
				ParamName: w.ParamName,
				ParamType: w.ParamType,
			})
		}
		var orders []codegen.OrderClause
		for _, o := range n.OrderBy {
			orders = append(orders, codegen.OrderClause{
				Field:     o.Field,
				Direction: o.Direction,
			})
		}
		result = append(result, codegen.IncludeNode{
			RelationName: n.RelationName,
			ModelName:    n.ModelName,
			IsArray:      n.IsArray,
			ForeignKey:   n.ForeignKey,
			ReferenceKey: n.ReferenceKey,
			Select:       n.Select,
			Where:        wheres,
			OrderBy:      orders,
			Limit:        n.Limit,
			Include:      convertIncludes(n.Include),
		})
	}
	return result
}

func toCodegenDef(req *QueryRequest) *codegen.QueryDefinition {
	var wheres []codegen.WhereClause
	for _, w := range req.Where {
		wheres = append(wheres, codegen.WhereClause{
			Field:     w.Field,
			Operator:  w.Operator,
			ParamName: w.ParamName,
			ParamType: w.ParamType,
		})
	}
	var orders []codegen.OrderClause
	for _, o := range req.OrderBy {
		orders = append(orders, codegen.OrderClause{
			Field:     o.Field,
			Direction: o.Direction,
		})
	}
	var joins []codegen.JoinClause
	for _, j := range req.Joins {
		joins = append(joins, codegen.JoinClause{
			ModelName:    j.ModelName,
			Fields:       j.Fields,
			ForeignKey:   j.ForeignKey,
			ReferenceKey: j.ReferenceKey,
			Type:         j.Type,
		})
	}
	return &codegen.QueryDefinition{
		Name:      req.Name,
		ModelName: req.ModelName,
		Operation: mapOperation(req.Operation),
		Fields:    req.Fields,
		Where:     wheres,
		OrderBy:   orders,
		Limit:     req.Limit,
		Joins:     joins,
	}
}
