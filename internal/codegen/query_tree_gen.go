package codegen

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bilalabdelkadir/prim/internal/schema"
)

// GeneratePrimQuery generates Go code for a Prisma-style nested query.
// It returns the structs code and method code separately.
func GeneratePrimQuery(q *PrimQuery, s *schema.Schema) (code string, structs string, err error) {
	model := findModel(s, q.ModelName)
	if model == nil {
		return "", "", fmt.Errorf("codegen: model %q not found in schema", q.ModelName)
	}

	// Validate includes.
	if err := validateIncludes(q.Include, s); err != nil {
		return "", "", err
	}

	var structBuf strings.Builder
	var methodBuf strings.Builder

	prefix := q.Name

	// Generate result structs (delete has no result struct).
	if q.Operation != QueryOpDelete {
		generateResultStruct(&structBuf, prefix, "", model, q.Select, q.Include, s, "")
	}

	// Generate method.
	if err := generateMethod(&methodBuf, q, model, prefix, s); err != nil {
		return "", "", err
	}

	return methodBuf.String(), structBuf.String(), nil
}

func validateIncludes(includes []IncludeNode, s *schema.Schema) error {
	for _, inc := range includes {
		if findModel(s, inc.ModelName) == nil {
			return fmt.Errorf("codegen: included model %q not found in schema", inc.ModelName)
		}
		if err := validateIncludes(inc.Include, s); err != nil {
			return err
		}
	}
	return nil
}

// generateResultStruct writes a result struct for a model at a given nesting level.
func generateResultStruct(b *strings.Builder, prefix string, suffix string, model *schema.Model, sel []string, includes []IncludeNode, s *schema.Schema, fkField string) {
	structName := prefix + suffix + "Result"
	selected := makeFieldSet(sel)

	b.WriteString("type ")
	b.WriteString(structName)
	b.WriteString(" struct {\n")

	for _, f := range model.Fields {
		if !isColumnField(f) {
			continue
		}
		if len(selected) > 0 && !selected[f.Name] {
			// Always include the id field for mapping.
			isID := false
			for _, attr := range f.Attributes {
				if attr.Name == "id" {
					isID = true
					break
				}
			}
			// Always include FK field for parent-child mapping.
			if f.Name == fkField {
				// include it
			} else if !isID && f.Name != "id" {
				continue
			}
		}
		b.WriteString("\t")
		b.WriteString(goName(f.Name))
		b.WriteString(" ")
		b.WriteString(goType(f))
		b.WriteString("\n")
	}

	// Add slice fields for includes.
	for _, inc := range includes {
		childStructName := prefix + suffix + goName(inc.RelationName) + "Result"
		b.WriteString("\t")
		b.WriteString(goName(inc.RelationName))
		b.WriteString(" ")
		if inc.IsArray {
			b.WriteString("[]")
		} else {
			b.WriteString("*")
		}
		b.WriteString(childStructName)
		b.WriteString("\n")
	}

	b.WriteString("}\n\n")

	// Recurse for each include.
	for _, inc := range includes {
		childModel := findModel(s, inc.ModelName)
		if childModel == nil {
			continue
		}
		childSuffix := suffix + goName(inc.RelationName)
		// For has-many (IsArray), the FK lives on the child side so include it.
		// For belongs-to (!IsArray), the FK lives on the parent side; don't add it to child struct.
		childFKField := ""
		if inc.IsArray {
			childFKField = inc.ForeignKey
		}
		generateResultStruct(b, prefix, childSuffix, childModel, inc.Select, inc.Include, s, childFKField)
	}
}

func generateMethod(b *strings.Builder, q *PrimQuery, model *schema.Model, prefix string, s *schema.Schema) error {
	structName := prefix + "Result"

	// Method signature.
	b.WriteString("func (r *")
	b.WriteString(q.ModelName)
	b.WriteString("Repository) ")
	b.WriteString(q.Name)
	b.WriteString("(ctx context.Context")

	// For create/update, data params come first.
	for _, d := range q.Data {
		b.WriteString(", ")
		b.WriteString(d.ParamName)
		b.WriteString(" ")
		b.WriteString(d.ParamType)
	}
	for _, w := range q.Where {
		if w.Operator == "is_null" {
			continue
		}
		b.WriteString(", ")
		b.WriteString(w.ParamName)
		b.WriteString(" ")
		b.WriteString(w.ParamType)
	}
	// Also collect params from includes.
	collectIncludeParams(b, q.Include)
	b.WriteString(") ")

	switch q.Operation {
	case QueryOpCount:
		b.WriteString("(int, error)")
	case QueryOpFindOne:
		b.WriteString("(*")
		b.WriteString(structName)
		b.WriteString(", error)")
	case QueryOpFindMany:
		b.WriteString("([]*")
		b.WriteString(structName)
		b.WriteString(", error)")
	case QueryOpCreate:
		b.WriteString("(*")
		b.WriteString(structName)
		b.WriteString(", error)")
	case QueryOpUpdate:
		b.WriteString("(*")
		b.WriteString(structName)
		b.WriteString(", error)")
	case QueryOpDelete:
		b.WriteString("error")
	}
	b.WriteString(" {\n")

	// Build root SQL.
	rootTable := tableName(q.ModelName)
	rootCols := buildPrimSelectCols(model, q.Select, q.Include)

	switch q.Operation {
	case QueryOpCount:
		generateCountBody(b, rootTable, q.Where)
		b.WriteString("}\n")
		return nil
	case QueryOpFindOne:
		generateRootQuery(b, rootTable, rootCols, q, model, structName, s, prefix, true)
	case QueryOpFindMany:
		generateRootQuery(b, rootTable, rootCols, q, model, structName, s, prefix, false)
	case QueryOpCreate:
		generateCreateBody(b, rootTable, rootCols, q, model, structName, s, prefix)
	case QueryOpUpdate:
		generateUpdateBody(b, rootTable, rootCols, q, model, structName, s, prefix)
	case QueryOpDelete:
		generateDeleteBody(b, rootTable, q)
	}

	b.WriteString("}\n")
	return nil
}

func generateCreateBody(b *strings.Builder, table string, cols []string, q *PrimQuery, model *schema.Model, structName string, s *schema.Schema, prefix string) {
	// Build INSERT columns and parameter placeholders from Data fields.
	var insertCols []string
	var insertParams []string
	var insertArgs []string
	for i, d := range q.Data {
		insertCols = append(insertCols, `"`+d.FieldName+`"`)
		insertParams = append(insertParams, "$"+strconv.Itoa(i+1))
		insertArgs = append(insertArgs, d.ParamName)
	}

	// Build RETURNING columns.
	var retCols []string
	for _, c := range cols {
		retCols = append(retCols, `"`+c+`"`)
	}

	b.WriteString("\tu := &")
	b.WriteString(structName)
	b.WriteString("{}\n")
	b.WriteString("\terr := r.db.QueryRowContext(ctx,\n")
	b.WriteString("\t\t`INSERT INTO \"")
	b.WriteString(table)
	b.WriteString("\" (")
	b.WriteString(strings.Join(insertCols, ", "))
	b.WriteString(") VALUES (")
	b.WriteString(strings.Join(insertParams, ", "))
	b.WriteString(") RETURNING ")
	b.WriteString(strings.Join(retCols, ", "))
	b.WriteString("`,\n")
	if len(insertArgs) > 0 {
		b.WriteString("\t\t")
		b.WriteString(strings.Join(insertArgs, ", "))
		b.WriteString(",\n")
	}
	b.WriteString("\t).Scan(")
	writeScanArgs(b, cols, "u")
	b.WriteString(")\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn nil, err\n")
	b.WriteString("\t}\n")

	// Handle nested creates for includes with CreateData.
	for _, inc := range q.Include {
		if len(inc.CreateData) == 0 {
			continue
		}
		childModel := findModel(s, inc.ModelName)
		if childModel == nil {
			continue
		}
		childTable := tableName(inc.ModelName)
		childCols := buildChildSelectCols(childModel, inc.Select, inc.ForeignKey)

		var childInsertCols []string
		var childInsertParams []string
		var childInsertArgs []string
		paramIdx := 1

		// FK column referencing parent.
		childInsertCols = append(childInsertCols, `"`+inc.ForeignKey+`"`)
		childInsertParams = append(childInsertParams, "$"+strconv.Itoa(paramIdx))
		childInsertArgs = append(childInsertArgs, "u.Id")
		paramIdx++

		for _, d := range inc.CreateData {
			childInsertCols = append(childInsertCols, `"`+d.FieldName+`"`)
			childInsertParams = append(childInsertParams, "$"+strconv.Itoa(paramIdx))
			childInsertArgs = append(childInsertArgs, d.ParamName)
			paramIdx++
		}

		var childRetCols []string
		for _, c := range childCols {
			childRetCols = append(childRetCols, `"`+c+`"`)
		}

		childStructName := prefix + goName(inc.RelationName) + "Result"
		childVar := strings.ToLower(inc.RelationName[:1]) + inc.RelationName[1:]

		b.WriteString("\t")
		b.WriteString(childVar)
		b.WriteString(" := &")
		b.WriteString(childStructName)
		b.WriteString("{}\n")
		b.WriteString("\terr = r.db.QueryRowContext(ctx,\n")
		b.WriteString("\t\t`INSERT INTO \"")
		b.WriteString(childTable)
		b.WriteString("\" (")
		b.WriteString(strings.Join(childInsertCols, ", "))
		b.WriteString(") VALUES (")
		b.WriteString(strings.Join(childInsertParams, ", "))
		b.WriteString(") RETURNING ")
		b.WriteString(strings.Join(childRetCols, ", "))
		b.WriteString("`,\n")
		b.WriteString("\t\t")
		b.WriteString(strings.Join(childInsertArgs, ", "))
		b.WriteString(",\n")
		b.WriteString("\t).Scan(")
		writeScanArgs(b, childCols, childVar)
		b.WriteString(")\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn nil, err\n")
		b.WriteString("\t}\n")

		// Attach child to parent.
		if inc.IsArray {
			b.WriteString("\tu.")
			b.WriteString(goName(inc.RelationName))
			b.WriteString(" = append(u.")
			b.WriteString(goName(inc.RelationName))
			b.WriteString(", ")
			b.WriteString(childVar)
			b.WriteString(")\n")
		} else {
			b.WriteString("\tu.")
			b.WriteString(goName(inc.RelationName))
			b.WriteString(" = ")
			b.WriteString(childVar)
			b.WriteString("\n")
		}
	}

	b.WriteString("\treturn u, nil\n")
}

func generateUpdateBody(b *strings.Builder, table string, cols []string, q *PrimQuery, model *schema.Model, structName string, s *schema.Schema, prefix string) {
	// Build SET clause from Data fields.
	var setClauses []string
	var setArgs []string
	paramIdx := 1
	for _, d := range q.Data {
		setClauses = append(setClauses, `"`+d.FieldName+`" = $`+strconv.Itoa(paramIdx))
		setArgs = append(setArgs, d.ParamName)
		paramIdx++
	}

	// Build RETURNING columns.
	var retCols []string
	for _, c := range cols {
		retCols = append(retCols, `"`+c+`"`)
	}

	b.WriteString("\tu := &")
	b.WriteString(structName)
	b.WriteString("{}\n")
	b.WriteString("\terr := r.db.QueryRowContext(ctx,\n")
	b.WriteString("\t\t`UPDATE \"")
	b.WriteString(table)
	b.WriteString("\" SET ")
	b.WriteString(strings.Join(setClauses, ", "))

	// WHERE clause.
	if len(q.Where) > 0 {
		b.WriteString(" WHERE ")
		for i, w := range q.Where {
			if i > 0 {
				b.WriteString(" AND ")
			}
			b.WriteString(`"`)
			b.WriteString(w.Field)
			b.WriteString(`" `)
			b.WriteString(sqlOperator(w.Operator))
			if w.Operator != "is_null" {
				b.WriteString(" $")
				b.WriteString(strconv.Itoa(paramIdx))
				paramIdx++
			}
		}
	}

	b.WriteString(" RETURNING ")
	b.WriteString(strings.Join(retCols, ", "))
	b.WriteString("`,\n")

	// Args: data args then where args.
	var allArgs []string
	allArgs = append(allArgs, setArgs...)
	for _, w := range q.Where {
		if w.Operator != "is_null" {
			allArgs = append(allArgs, w.ParamName)
		}
	}
	if len(allArgs) > 0 {
		b.WriteString("\t\t")
		b.WriteString(strings.Join(allArgs, ", "))
		b.WriteString(",\n")
	}

	b.WriteString("\t).Scan(")
	writeScanArgs(b, cols, "u")
	b.WriteString(")\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn nil, err\n")
	b.WriteString("\t}\n")

	// If includes exist, fetch related data (same as find_one).
	if len(q.Include) > 0 {
		b.WriteString("\tparentIDs := map[int]int{u.Id: 0}\n")
		b.WriteString("\tresults := []*")
		b.WriteString(structName)
		b.WriteString("{u}\n")
		generateIncludeQueries(b, q.Include, s, prefix, "", "results", "parentIDs", "Id")
		b.WriteString("\treturn results[0], nil\n")
	} else {
		b.WriteString("\treturn u, nil\n")
	}
}

func generateDeleteBody(b *strings.Builder, table string, q *PrimQuery) {
	// If includes are present, delete children first.
	for _, inc := range q.Include {
		childTable := tableName(inc.ModelName)
		b.WriteString("\t_, err := r.db.ExecContext(ctx,\n")
		b.WriteString("\t\t`DELETE FROM \"")
		b.WriteString(childTable)
		b.WriteString("\" WHERE \"")
		b.WriteString(inc.ForeignKey)
		b.WriteString("\" IN (SELECT \"")
		b.WriteString(inc.ReferenceKey)
		b.WriteString("\" FROM \"")
		b.WriteString(table)
		b.WriteString("\"")

		paramIdx := 1
		if len(q.Where) > 0 {
			b.WriteString(" WHERE ")
			for i, w := range q.Where {
				if i > 0 {
					b.WriteString(" AND ")
				}
				b.WriteString(`"`)
				b.WriteString(w.Field)
				b.WriteString(`" `)
				b.WriteString(sqlOperator(w.Operator))
				if w.Operator != "is_null" {
					b.WriteString(" $")
					b.WriteString(strconv.Itoa(paramIdx))
					paramIdx++
				}
			}
		}

		b.WriteString(")`,\n")
		var args []string
		for _, w := range q.Where {
			if w.Operator != "is_null" {
				args = append(args, w.ParamName)
			}
		}
		if len(args) > 0 {
			b.WriteString("\t\t")
			b.WriteString(strings.Join(args, ", "))
			b.WriteString(",\n")
		}
		b.WriteString("\t)\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn err\n")
		b.WriteString("\t}\n")
	}

	// Delete the parent record.
	if len(q.Include) > 0 {
		// We already declared err above, use = instead of :=
		b.WriteString("\t_, err = r.db.ExecContext(ctx,\n")
	} else {
		b.WriteString("\t_, err := r.db.ExecContext(ctx,\n")
	}
	b.WriteString("\t\t`DELETE FROM \"")
	b.WriteString(table)
	b.WriteString("\"")

	paramIdx := 1
	if len(q.Where) > 0 {
		b.WriteString(" WHERE ")
		for i, w := range q.Where {
			if i > 0 {
				b.WriteString(" AND ")
			}
			b.WriteString(`"`)
			b.WriteString(w.Field)
			b.WriteString(`" `)
			b.WriteString(sqlOperator(w.Operator))
			if w.Operator != "is_null" {
				b.WriteString(" $")
				b.WriteString(strconv.Itoa(paramIdx))
				paramIdx++
			}
		}
	}

	b.WriteString("`,\n")
	var args []string
	for _, w := range q.Where {
		if w.Operator != "is_null" {
			args = append(args, w.ParamName)
		}
	}
	if len(args) > 0 {
		b.WriteString("\t\t")
		b.WriteString(strings.Join(args, ", "))
		b.WriteString(",\n")
	}
	b.WriteString("\t)\n")
	b.WriteString("\treturn err\n")
}

func collectIncludeParams(b *strings.Builder, includes []IncludeNode) {
	for _, inc := range includes {
		for _, d := range inc.CreateData {
			b.WriteString(", ")
			b.WriteString(d.ParamName)
			b.WriteString(" ")
			b.WriteString(d.ParamType)
		}
		for _, w := range inc.Where {
			if w.Operator == "is_null" {
				continue
			}
			b.WriteString(", ")
			b.WriteString(w.ParamName)
			b.WriteString(" ")
			b.WriteString(w.ParamType)
		}
		collectIncludeParams(b, inc.Include)
	}
}

func generateCountBody(b *strings.Builder, table string, wheres []WhereClause) {
	b.WriteString("\tvar count int\n")
	b.WriteString("\terr := r.db.QueryRowContext(ctx,\n")
	b.WriteString("\t\t`SELECT COUNT(*) FROM \"")
	b.WriteString(table)
	b.WriteString("\"")

	paramIdx := 1
	if len(wheres) > 0 {
		b.WriteString(" WHERE ")
		for i, w := range wheres {
			if i > 0 {
				b.WriteString(" AND ")
			}
			b.WriteString("\"")
			b.WriteString(w.Field)
			b.WriteString("\" ")
			b.WriteString(sqlOperator(w.Operator))
			if w.Operator != "is_null" {
				b.WriteString(" $")
				b.WriteString(strconv.Itoa(paramIdx))
				paramIdx++
			}
		}
	}
	b.WriteString("`,\n")

	var args []string
	for _, w := range wheres {
		if w.Operator != "is_null" {
			args = append(args, w.ParamName)
		}
	}
	if len(args) > 0 {
		b.WriteString("\t\t")
		b.WriteString(strings.Join(args, ", "))
		b.WriteString(",\n")
	}

	b.WriteString("\t).Scan(&count)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn 0, err\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn count, nil\n")
}

// buildPrimSelectCols returns the columns to SELECT for the root query.
// It always includes the id/PK field for parent-child mapping.
func buildPrimSelectCols(model *schema.Model, sel []string, includes []IncludeNode) []string {
	selected := makeFieldSet(sel)
	var cols []string
	for _, f := range model.Fields {
		if !isColumnField(f) {
			continue
		}
		if len(selected) > 0 && !selected[f.Name] {
			// Always include id for mapping.
			isID := false
			for _, attr := range f.Attributes {
				if attr.Name == "id" {
					isID = true
					break
				}
			}
			if !isID && f.Name != "id" {
				continue
			}
		}
		cols = append(cols, f.Name)
	}

	// For belongs-to includes (!IsArray), the FK lives on the parent side.
	// Ensure those FK columns are included in the parent SELECT.
	for _, inc := range includes {
		if !inc.IsArray {
			found := false
			for _, c := range cols {
				if c == inc.ForeignKey {
					found = true
					break
				}
			}
			if !found {
				cols = append(cols, inc.ForeignKey)
			}
		}
	}

	return cols
}

func generateRootQuery(b *strings.Builder, table string, cols []string, q *PrimQuery, model *schema.Model, structName string, s *schema.Schema, prefix string, findOne bool) {
	// Build SQL.
	var sqlParts []string
	for _, c := range cols {
		sqlParts = append(sqlParts, `"`+c+`"`)
	}
	sqlStr := strings.Join(sqlParts, ", ")

	if findOne {
		// find_one: use QueryRowContext
		b.WriteString("\tu := &")
		b.WriteString(structName)
		b.WriteString("{}\n")
		b.WriteString("\terr := r.db.QueryRowContext(ctx,\n")
		b.WriteString("\t\t`SELECT ")
		b.WriteString(sqlStr)
		b.WriteString(" FROM \"")
		b.WriteString(table)
		b.WriteString("\"")

		paramIdx := writeWhereClauses(b, q.Where)
		writeOrderBy(b, q.OrderBy)
		_ = paramIdx
		b.WriteString("`,\n")

		writeWhereArgs(b, q.Where)
		b.WriteString("\t).Scan(")
		writeScanArgs(b, cols, "u")
		b.WriteString(")\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn nil, err\n")
		b.WriteString("\t}\n")

		if len(q.Include) > 0 {
			// For find_one with includes, we need the parent ID.
			// Build a map with single entry.
			b.WriteString("\tparentIDs := map[int]int{u.Id: 0}\n")
			b.WriteString("\tresults := []*")
			b.WriteString(structName)
			b.WriteString("{u}\n")

			generateIncludeQueries(b, q.Include, s, prefix, "", "results", "parentIDs", "Id")
			b.WriteString("\treturn results[0], nil\n")
		} else {
			b.WriteString("\treturn u, nil\n")
		}
	} else {
		// find_many: use QueryContext
		b.WriteString("\trows, err := r.db.QueryContext(ctx,\n")
		b.WriteString("\t\t`SELECT ")
		b.WriteString(sqlStr)
		b.WriteString(" FROM \"")
		b.WriteString(table)
		b.WriteString("\"")

		writeWhereClauses(b, q.Where)
		writeOrderBy(b, q.OrderBy)
		if q.Limit > 0 {
			b.WriteString(" LIMIT ")
			b.WriteString(strconv.Itoa(q.Limit))
		}
		if q.Skip > 0 {
			b.WriteString(" OFFSET ")
			b.WriteString(strconv.Itoa(q.Skip))
		}
		b.WriteString("`,\n")

		writeWhereArgs(b, q.Where)
		b.WriteString("\t)\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn nil, err\n")
		b.WriteString("\t}\n")
		b.WriteString("\tdefer rows.Close()\n\n")

		b.WriteString("\tvar results []*")
		b.WriteString(structName)
		b.WriteString("\n")
		b.WriteString("\tparentIDs := make(map[int]int)\n")
		b.WriteString("\tfor rows.Next() {\n")
		b.WriteString("\t\tu := &")
		b.WriteString(structName)
		b.WriteString("{}\n")
		b.WriteString("\t\tif err := rows.Scan(")
		writeScanArgs(b, cols, "u")
		b.WriteString("); err != nil {\n")
		b.WriteString("\t\t\treturn nil, err\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t\tparentIDs[u.Id] = len(results)\n")
		b.WriteString("\t\tresults = append(results, u)\n")
		b.WriteString("\t}\n")
		b.WriteString("\tif err := rows.Err(); err != nil {\n")
		b.WriteString("\t\treturn nil, err\n")
		b.WriteString("\t}\n")
		b.WriteString("\tif len(results) == 0 {\n")
		b.WriteString("\t\treturn results, nil\n")
		b.WriteString("\t}\n\n")

		if len(q.Include) > 0 {
			generateIncludeQueries(b, q.Include, s, prefix, "", "results", "parentIDs", "Id")
		}

		b.WriteString("\treturn results, nil\n")
	}
}

func writeWhereClauses(b *strings.Builder, wheres []WhereClause) int {
	paramIdx := 1
	if len(wheres) > 0 {
		b.WriteString(" WHERE ")
		for i, w := range wheres {
			if i > 0 {
				b.WriteString(" AND ")
			}
			b.WriteString("\"")
			b.WriteString(w.Field)
			b.WriteString("\" ")
			b.WriteString(sqlOperator(w.Operator))
			if w.Operator != "is_null" {
				b.WriteString(" $")
				b.WriteString(strconv.Itoa(paramIdx))
				paramIdx++
			}
		}
	}
	return paramIdx
}

func writeOrderBy(b *strings.Builder, orders []OrderClause) {
	if len(orders) > 0 {
		b.WriteString(" ORDER BY ")
		for i, o := range orders {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString("\"")
			b.WriteString(o.Field)
			b.WriteString("\" ")
			b.WriteString(o.Direction)
		}
	}
}

func writeWhereArgs(b *strings.Builder, wheres []WhereClause) {
	var args []string
	for _, w := range wheres {
		if w.Operator != "is_null" {
			args = append(args, w.ParamName)
		}
	}
	if len(args) > 0 {
		b.WriteString("\t\t")
		b.WriteString(strings.Join(args, ", "))
		b.WriteString(",\n")
	}
}

func writeScanArgs(b *strings.Builder, cols []string, varName string) {
	for i, c := range cols {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString("&")
		b.WriteString(varName)
		b.WriteString(".")
		b.WriteString(goName(c))
	}
}

func generateIncludeQueries(b *strings.Builder, includes []IncludeNode, s *schema.Schema, prefix string, parentSuffix string, parentResultsVar string, parentIDsVar string, parentKeyField string) {
	for _, inc := range includes {
		childModel := findModel(s, inc.ModelName)
		if childModel == nil {
			continue
		}

		childSuffix := parentSuffix + goName(inc.RelationName)
		childStructName := prefix + childSuffix + "Result"
		childTable := tableName(inc.ModelName)

		// For has-many (IsArray): FK is on the child side.
		//   Query: WHERE "fk" = ANY($1), collect parent IDs, match c.FK to parent ID.
		// For belongs-to (!IsArray): FK is on the parent side.
		//   Query: WHERE "referenceKey" = ANY($1), collect parent FK values, match c.ReferenceKey to parent FK.
		var childWhereCol string   // column used in WHERE ... = ANY($1)
		var childMatchField string // Go field on child used to match back to parent
		if inc.IsArray {
			childWhereCol = inc.ForeignKey
			childMatchField = goName(inc.ForeignKey)
		} else {
			childWhereCol = inc.ReferenceKey
			childMatchField = goName(inc.ReferenceKey)
		}

		// Build child select columns.
		// For has-many, FK must be in child columns. For belongs-to, FK is not on child.
		childFKForSelect := ""
		if inc.IsArray {
			childFKForSelect = inc.ForeignKey
		}
		childCols := buildChildSelectCols(childModel, inc.Select, childFKForSelect)

		var sqlColParts []string
		for _, c := range childCols {
			sqlColParts = append(sqlColParts, `"`+c+`"`)
		}
		childSQL := strings.Join(sqlColParts, ", ")

		// Unique variable names based on relation.
		rowsVar := strings.ToLower(inc.RelationName) + "Rows"
		idCollectVar := strings.ToLower(inc.RelationName) + "IDs"
		childMapVar := strings.ToLower(inc.RelationName) + "Map"

		if inc.IsArray {
			// has-many: collect parent IDs (the map keys are parent IDs).
			b.WriteString("\t")
			b.WriteString(idCollectVar)
			b.WriteString(" := make([]int, 0, len(")
			b.WriteString(parentIDsVar)
			b.WriteString("))\n")
			b.WriteString("\tfor k := range ")
			b.WriteString(parentIDsVar)
			b.WriteString(" {\n")
			b.WriteString("\t\t")
			b.WriteString(idCollectVar)
			b.WriteString(" = append(")
			b.WriteString(idCollectVar)
			b.WriteString(", k)\n")
			b.WriteString("\t}\n")
		} else {
			// belongs-to: collect parent FK values from the parent results.
			// We also need a map from parent FK value -> parent index for attaching.
			b.WriteString("\t")
			b.WriteString(idCollectVar)
			b.WriteString(" := make([]int, 0, len(")
			b.WriteString(parentResultsVar)
			b.WriteString("))\n")
			b.WriteString("\t")
			b.WriteString(idCollectVar)
			b.WriteString("Map := make(map[int]int)\n")
			b.WriteString("\tfor i, p := range ")
			b.WriteString(parentResultsVar)
			b.WriteString(" {\n")
			b.WriteString("\t\t")
			b.WriteString(idCollectVar)
			b.WriteString(" = append(")
			b.WriteString(idCollectVar)
			b.WriteString(", p.")
			b.WriteString(goName(inc.ForeignKey))
			b.WriteString(")\n")
			b.WriteString("\t\t")
			b.WriteString(idCollectVar)
			b.WriteString("Map[p.")
			b.WriteString(goName(inc.ForeignKey))
			b.WriteString("] = i\n")
			b.WriteString("\t}\n")
		}

		// Build child query.
		b.WriteString("\t")
		b.WriteString(rowsVar)
		b.WriteString(", err := r.db.QueryContext(ctx,\n")
		b.WriteString("\t\t`SELECT ")
		b.WriteString(childSQL)
		b.WriteString(" FROM \"")
		b.WriteString(childTable)
		b.WriteString("\" WHERE \"")
		b.WriteString(childWhereCol)
		b.WriteString("\" = ANY($1)")

		// Additional WHERE conditions on the child.
		childParamIdx := 2
		for _, w := range inc.Where {
			b.WriteString(" AND \"")
			b.WriteString(w.Field)
			b.WriteString("\" ")
			b.WriteString(sqlOperator(w.Operator))
			if w.Operator != "is_null" {
				b.WriteString(" $")
				b.WriteString(strconv.Itoa(childParamIdx))
				childParamIdx++
			}
		}

		// ORDER BY for child.
		writeOrderBy(b, inc.OrderBy)

		if inc.Limit > 0 {
			b.WriteString(" LIMIT ")
			b.WriteString(strconv.Itoa(inc.Limit))
		}

		b.WriteString("`,\n")
		b.WriteString("\t\tpq.Array(")
		b.WriteString(idCollectVar)
		b.WriteString(")")
		for _, w := range inc.Where {
			if w.Operator != "is_null" {
				b.WriteString(", ")
				b.WriteString(w.ParamName)
			}
		}
		b.WriteString(",\n")
		b.WriteString("\t)\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn nil, err\n")
		b.WriteString("\t}\n")
		b.WriteString("\tdefer ")
		b.WriteString(rowsVar)
		b.WriteString(".Close()\n\n")

		if len(inc.Include) > 0 {
			// Need a map for grandchild lookups.
			b.WriteString("\t")
			b.WriteString(childMapVar)
			b.WriteString(" := make(map[int]int)\n")
			b.WriteString("\tvar ")
			b.WriteString(strings.ToLower(inc.RelationName))
			b.WriteString("All []*")
			b.WriteString(childStructName)
			b.WriteString("\n")
		}

		b.WriteString("\tfor ")
		b.WriteString(rowsVar)
		b.WriteString(".Next() {\n")
		b.WriteString("\t\tc := &")
		b.WriteString(childStructName)
		b.WriteString("{}\n")
		b.WriteString("\t\tif err := ")
		b.WriteString(rowsVar)
		b.WriteString(".Scan(")
		writeScanArgs(b, childCols, "c")
		b.WriteString("); err != nil {\n")
		b.WriteString("\t\t\treturn nil, err\n")
		b.WriteString("\t\t}\n")

		// Attach to parent.
		// For has-many: match c.FK to parentIDsVar (map of parent ID -> index).
		// For belongs-to: match c.ReferenceKey to the FK-based map we built above.
		if inc.IsArray {
			b.WriteString("\t\tif idx, ok := ")
			b.WriteString(parentIDsVar)
			b.WriteString("[c.")
			b.WriteString(childMatchField)
			b.WriteString("]; ok {\n")
			b.WriteString("\t\t\t")
			b.WriteString(parentResultsVar)
			b.WriteString("[idx].")
			b.WriteString(goName(inc.RelationName))
			b.WriteString(" = append(")
			b.WriteString(parentResultsVar)
			b.WriteString("[idx].")
			b.WriteString(goName(inc.RelationName))
			b.WriteString(", c)\n")
		} else {
			b.WriteString("\t\tif idx, ok := ")
			b.WriteString(idCollectVar)
			b.WriteString("Map[c.")
			b.WriteString(childMatchField)
			b.WriteString("]; ok {\n")
			b.WriteString("\t\t\t")
			b.WriteString(parentResultsVar)
			b.WriteString("[idx].")
			b.WriteString(goName(inc.RelationName))
			b.WriteString(" = c\n")
		}

		if len(inc.Include) > 0 {
			b.WriteString("\t\t\t")
			b.WriteString(childMapVar)
			b.WriteString("[c.Id] = len(")
			b.WriteString(strings.ToLower(inc.RelationName))
			b.WriteString("All)\n")
			b.WriteString("\t\t\t")
			b.WriteString(strings.ToLower(inc.RelationName))
			b.WriteString("All = append(")
			b.WriteString(strings.ToLower(inc.RelationName))
			b.WriteString("All, c)\n")
		}

		b.WriteString("\t\t}\n")
		b.WriteString("\t}\n")
		b.WriteString("\tif err := ")
		b.WriteString(rowsVar)
		b.WriteString(".Err(); err != nil {\n")
		b.WriteString("\t\treturn nil, err\n")
		b.WriteString("\t}\n\n")

		// Recurse for nested includes.
		if len(inc.Include) > 0 {
			generateIncludeQueries(b, inc.Include, s, prefix, childSuffix, strings.ToLower(inc.RelationName)+"All", childMapVar, "Id")
		}
	}
}

func buildChildSelectCols(model *schema.Model, sel []string, fkField string) []string {
	selected := makeFieldSet(sel)
	var cols []string
	fkIncluded := false
	idIncluded := false

	for _, f := range model.Fields {
		if !isColumnField(f) {
			continue
		}
		isID := false
		for _, attr := range f.Attributes {
			if attr.Name == "id" {
				isID = true
				break
			}
		}
		if f.Name == "id" {
			isID = true
		}

		if len(selected) > 0 && !selected[f.Name] {
			// Always include FK and id for mapping.
			if f.Name == fkField {
				cols = append(cols, f.Name)
				fkIncluded = true
				continue
			}
			if isID {
				cols = append(cols, f.Name)
				idIncluded = true
				continue
			}
			continue
		}
		if f.Name == fkField {
			fkIncluded = true
		}
		if isID {
			idIncluded = true
		}
		cols = append(cols, f.Name)
	}

	// Ensure FK is in the list.
	if !fkIncluded {
		cols = append(cols, fkField)
	}
	_ = idIncluded

	return cols
}
