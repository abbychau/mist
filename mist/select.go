package mist

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/opcode"
)

// SelectResult represents the result of a SELECT query
type SelectResult struct {
	Columns []string
	Rows    [][]interface{}
}

// ExecuteSelect processes a SELECT statement
func ExecuteSelect(db *Database, stmt *ast.SelectStmt) (*SelectResult, error) {
	// Handle simple SELECT from single table
	if stmt.From == nil {
		return nil, fmt.Errorf("SELECT without FROM is not supported yet")
	}

	// Get the table name - handle different table reference types
	table, err := resolveTableReference(db, stmt.From.TableRefs.Left)
	if err != nil {
		return nil, err
	}

	// Check if this is an aggregate query
	if hasAggregateFunction(stmt.Fields.Fields) {
		return executeAggregateQuery(table, stmt.Fields.Fields, stmt.Where, stmt.GroupBy, stmt.Having, stmt.Limit)
	}

	// Get rows from the table, potentially using indexes
	rows, err := getRowsWithOptimization(db, table, stmt.Where)
	if err != nil {
		return nil, err
	}

	// Determine which columns to select
	var selectedColumns []string
	var columnIndexes []int

	// Check for SELECT *
	if len(stmt.Fields.Fields) == 1 {
		field := stmt.Fields.Fields[0]
		if field.WildCard != nil {
			// This is SELECT *
			for i, col := range table.Columns {
				selectedColumns = append(selectedColumns, col.Name)
				columnIndexes = append(columnIndexes, i)
			}
		} else if colExpr, ok := field.Expr.(*ast.ColumnNameExpr); ok {
			if colExpr.Name.Name.String() == "*" {
				// Alternative way to detect SELECT *
				for i, col := range table.Columns {
					selectedColumns = append(selectedColumns, col.Name)
					columnIndexes = append(columnIndexes, i)
				}
			}
		}
	}

	// If not SELECT *, process individual columns
	if len(selectedColumns) == 0 {
		for _, field := range stmt.Fields.Fields {
			if colExpr, ok := field.Expr.(*ast.ColumnNameExpr); ok {
				colName := colExpr.Name.Name.String()
				colIndex := table.GetColumnIndex(colName)
				if colIndex == -1 {
					return nil, fmt.Errorf("column %s does not exist", colName)
				}
				selectedColumns = append(selectedColumns, colName)
				columnIndexes = append(columnIndexes, colIndex)
			} else {
				return nil, fmt.Errorf("complex expressions in SELECT not supported yet: %T", field.Expr)
			}
		}
	}

	// Build result rows
	var resultRows [][]interface{}
	for _, row := range rows {
		resultRow := make([]interface{}, len(columnIndexes))
		for i, colIndex := range columnIndexes {
			resultRow[i] = row.Values[colIndex]
		}
		resultRows = append(resultRows, resultRow)
	}

	// Apply LIMIT clause if present
	if stmt.Limit != nil {
		resultRows = applyLimit(resultRows, stmt.Limit)
	}

	// Build final result
	result := &SelectResult{
		Columns: selectedColumns,
		Rows:    resultRows,
	}

	return result, nil
}

// evaluateWhereCondition evaluates a WHERE condition against a row
func evaluateWhereCondition(expr ast.ExprNode, table *Table, row Row) (bool, error) {
	switch e := expr.(type) {
	case *ast.BinaryOperationExpr:
		return evaluateBinaryOperation(e, table, row)
	case *ast.ColumnNameExpr:
		// Column reference - treat as boolean
		colIndex := table.GetColumnIndex(e.Name.Name.String())
		if colIndex == -1 {
			return false, fmt.Errorf("column %s does not exist", e.Name.Name.String())
		}
		value := row.Values[colIndex]
		return isTruthy(value), nil
	default:
		return false, fmt.Errorf("unsupported WHERE expression type: %T", expr)
	}
}

// evaluateBinaryOperation evaluates binary operations like =, >, <, etc.
func evaluateBinaryOperation(expr *ast.BinaryOperationExpr, table *Table, row Row) (bool, error) {
	leftVal, err := evaluateExpressionInRow(expr.L, table, row)
	if err != nil {
		return false, err
	}

	rightVal, err := evaluateExpressionInRow(expr.R, table, row)
	if err != nil {
		return false, err
	}

	switch expr.Op {
	case opcode.EQ:
		return compareValues(leftVal, rightVal) == 0, nil
	case opcode.NE:
		return compareValues(leftVal, rightVal) != 0, nil
	case opcode.LT:
		return compareValues(leftVal, rightVal) < 0, nil
	case opcode.LE:
		return compareValues(leftVal, rightVal) <= 0, nil
	case opcode.GT:
		return compareValues(leftVal, rightVal) > 0, nil
	case opcode.GE:
		return compareValues(leftVal, rightVal) >= 0, nil
	case opcode.LogicAnd:
		return isTruthy(leftVal) && isTruthy(rightVal), nil
	case opcode.LogicOr:
		return isTruthy(leftVal) || isTruthy(rightVal), nil
	default:
		return false, fmt.Errorf("unsupported binary operator: %v", expr.Op)
	}
}

// evaluateExpressionInRow evaluates an expression in the context of a row
func evaluateExpressionInRow(expr ast.ExprNode, table *Table, row Row) (interface{}, error) {
	switch e := expr.(type) {
	case *ast.ColumnNameExpr:
		colIndex := table.GetColumnIndex(e.Name.Name.String())
		if colIndex == -1 {
			return nil, fmt.Errorf("column %s does not exist", e.Name.Name.String())
		}
		return row.Values[colIndex], nil
	case ast.ValueExpr:
		return e.GetValue(), nil
	default:
		return nil, fmt.Errorf("unsupported expression type in WHERE: %T", expr)
	}
}

// compareValues compares two values and returns -1, 0, or 1
func compareValues(left, right interface{}) int {
	// Handle null values
	if left == nil && right == nil {
		return 0
	}
	if left == nil {
		return -1
	}
	if right == nil {
		return 1
	}

	// Convert to comparable types
	leftStr := fmt.Sprintf("%v", left)
	rightStr := fmt.Sprintf("%v", right)

	// Try numeric comparison first
	if leftNum, err1 := strconv.ParseFloat(leftStr, 64); err1 == nil {
		if rightNum, err2 := strconv.ParseFloat(rightStr, 64); err2 == nil {
			if leftNum < rightNum {
				return -1
			} else if leftNum > rightNum {
				return 1
			}
			return 0
		}
	}

	// Fall back to string comparison
	if leftStr < rightStr {
		return -1
	} else if leftStr > rightStr {
		return 1
	}
	return 0
}

// isTruthy determines if a value is "truthy"
func isTruthy(value interface{}) bool {
	if value == nil {
		return false
	}

	switch v := value.(type) {
	case bool:
		return v
	case int, int32, int64:
		return reflect.ValueOf(v).Int() != 0
	case float32, float64:
		return reflect.ValueOf(v).Float() != 0
	case string:
		return v != "" && strings.ToLower(v) != "false"
	default:
		return true
	}
}

// getRowsWithOptimization gets rows from table, using indexes when possible
func getRowsWithOptimization(db *Database, table *Table, whereExpr ast.ExprNode) ([]Row, error) {
	// If no WHERE clause, return all rows
	if whereExpr == nil {
		return table.GetRows(), nil
	}

	// Try to use index optimization for simple equality conditions
	if indexedRows, used := tryIndexOptimization(db, table, whereExpr); used {
		return indexedRows, nil
	}

	// Fall back to full table scan
	allRows := table.GetRows()
	var filteredRows []Row

	for _, row := range allRows {
		match, err := evaluateWhereCondition(whereExpr, table, row)
		if err != nil {
			return nil, fmt.Errorf("error evaluating WHERE clause: %v", err)
		}
		if match {
			filteredRows = append(filteredRows, row)
		}
	}

	return filteredRows, nil
}

// tryIndexOptimization attempts to use indexes for WHERE clause optimization
func tryIndexOptimization(db *Database, table *Table, whereExpr ast.ExprNode) ([]Row, bool) {
	// Only handle simple binary operations for now
	binOp, ok := whereExpr.(*ast.BinaryOperationExpr)
	if !ok {
		return nil, false
	}

	// Only handle equality operations
	if binOp.Op != opcode.EQ {
		return nil, false
	}

	// Check if left side is a column and right side is a value
	var columnName string
	var value interface{}

	if colExpr, ok := binOp.L.(*ast.ColumnNameExpr); ok {
		if valExpr, ok := binOp.R.(ast.ValueExpr); ok {
			columnName = colExpr.Name.Name.String()
			value = valExpr.GetValue()
		}
	} else if colExpr, ok := binOp.R.(*ast.ColumnNameExpr); ok {
		if valExpr, ok := binOp.L.(ast.ValueExpr); ok {
			columnName = colExpr.Name.Name.String()
			value = valExpr.GetValue()
		}
	}

	if columnName == "" {
		return nil, false
	}

	// Look for an index on this column
	indexes := db.IndexManager.GetIndexesForTable(table.Name, columnName)
	if len(indexes) == 0 {
		return nil, false
	}

	// Use the first available index
	index := indexes[0]
	rowIndexes := index.Lookup(value)

	if rowIndexes == nil {
		return []Row{}, true // No matching rows, but we used the index
	}

	// Get the actual rows
	allRows := table.GetRows()
	var result []Row

	for _, rowIndex := range rowIndexes {
		if rowIndex < len(allRows) {
			result = append(result, allRows[rowIndex])
		}
	}

	return result, true
}

// applyLimit applies LIMIT clause to result rows
func applyLimit(rows [][]interface{}, limit *ast.Limit) [][]interface{} {
	if limit == nil {
		return rows
	}

	// Parse offset and count - try different approaches
	offset := int64(0)
	count := int64(len(rows)) // default to all rows

	// Try to extract offset
	if limit.Offset != nil {
		if offsetExpr, ok := limit.Offset.(ast.ValueExpr); ok {
			if offsetVal := offsetExpr.GetValue(); offsetVal != nil {
				switch v := offsetVal.(type) {
				case int64:
					offset = v
				case int:
					offset = int64(v)
				case uint64:
					offset = int64(v)
				case float64:
					offset = int64(v)
				}
			}
		} else {
			// Parse from string representation
			offsetStr := fmt.Sprintf("%v", limit.Offset)
			if strings.Contains(offsetStr, "KindUint64") {
				// Extract the number after "KindUint64 "
				parts := strings.Split(offsetStr, " ")
				for i, part := range parts {
					if part == "KindUint64" && i+1 < len(parts) {
						if val, err := strconv.ParseInt(parts[i+1], 10, 64); err == nil {
							offset = val
						}
						break
					}
				}
			}
		}
	}

	// Try to extract count
	if limit.Count != nil {
		if countExpr, ok := limit.Count.(ast.ValueExpr); ok {
			if countVal := countExpr.GetValue(); countVal != nil {
				switch v := countVal.(type) {
				case int64:
					count = v
				case int:
					count = int64(v)
				case uint64:
					count = int64(v)
				case float64:
					count = int64(v)
				}
			}
		}
	}

	// Apply offset and count
	start := int(offset)
	if start < 0 {
		start = 0
	}
	if start >= len(rows) {
		return [][]interface{}{} // empty result
	}

	end := start + int(count)
	if end > len(rows) {
		end = len(rows)
	}
	return rows[start:end]
}

// resolveTableReference resolves different types of table references
func resolveTableReference(db *Database, tableRef ast.ResultSetNode) (*Table, error) {
	switch ref := tableRef.(type) {
	case *ast.TableSource:
		// Handle table source with potential alias
		switch source := ref.Source.(type) {
		case *ast.TableName:
			// Simple table reference
			return db.GetTable(source.Name.String())
		case *ast.SelectStmt:
			// Subquery - execute it and create a virtual table
			return executeSubquery(db, source)
		default:
			return nil, fmt.Errorf("unsupported table source type: %T", source)
		}
	case *ast.TableName:
		// Direct table name reference
		return db.GetTable(ref.Name.String())
	default:
		return nil, fmt.Errorf("unsupported table reference type: %T", ref)
	}
}

// executeSubquery executes a subquery and returns a virtual table
func executeSubquery(db *Database, subquery *ast.SelectStmt) (*Table, error) {
	// Execute the subquery
	result, err := ExecuteSelect(db, subquery)
	if err != nil {
		return nil, fmt.Errorf("error executing subquery: %v", err)
	}

	// Create a virtual table from the result
	virtualTable := &Table{
		Name:    "subquery_result",
		Columns: make([]Column, len(result.Columns)),
		Rows:    make([]Row, len(result.Rows)),
	}

	// Create column definitions (infer types from data)
	for i, colName := range result.Columns {
		colType := TypeText // default type
		if len(result.Rows) > 0 && i < len(result.Rows[0]) {
			// Infer type from first non-null value
			for _, row := range result.Rows {
				if row[i] != nil {
					colType = inferColumnType(row[i])
					break
				}
			}
		}

		virtualTable.Columns[i] = Column{
			Name: colName,
			Type: colType,
		}
	}

	// Convert result rows to table rows
	for i, resultRow := range result.Rows {
		virtualTable.Rows[i] = Row{Values: resultRow}
	}

	return virtualTable, nil
}

// inferColumnType infers column type from a value
func inferColumnType(value interface{}) ColumnType {
	switch value.(type) {
	case int, int32, int64:
		return TypeInt
	case float32, float64:
		return TypeFloat
	case bool:
		return TypeBool
	case string:
		// Try to infer if it's a timestamp or date format
		if str := value.(string); str != "" {
			// Simple heuristic for timestamp/date detection
			if len(str) >= 10 && (str[4] == '-' || str[2] == '/') {
				if len(str) > 10 {
					return TypeTimestamp
				}
				return TypeDate
			}
		}
		return TypeVarchar
	default:
		return TypeText
	}
}
