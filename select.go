package mist

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/abbychau/mysql-parser/ast"
	"github.com/abbychau/mysql-parser/opcode"
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

	// If not SELECT *, process individual columns/expressions
	var expressions []ast.ExprNode
	if len(selectedColumns) == 0 {
		for _, field := range stmt.Fields.Fields {
			// Generate column name (use alias if present, otherwise infer from expression)
			var colName string
			if field.AsName.L != "" {
				colName = field.AsName.L
			} else {
				colName = inferColumnNameFromExpression(field.Expr)
			}
			
			selectedColumns = append(selectedColumns, colName)
			expressions = append(expressions, field.Expr)
		}
	}

	// Build result rows
	var resultRows [][]interface{}
	for _, row := range rows {
		var resultRow []interface{}
		
		if len(expressions) > 0 {
			// Evaluate expressions for each column
			for _, expr := range expressions {
				value, err := evaluateExpressionInRowWithDB(expr, db, table, row)
				if err != nil {
					return nil, fmt.Errorf("error evaluating SELECT expression: %v", err)
				}
				resultRow = append(resultRow, value)
			}
		} else {
			// Use column indexes for SELECT *
			resultRow = make([]interface{}, len(columnIndexes))
			for i, colIndex := range columnIndexes {
				resultRow[i] = row.Values[colIndex]
			}
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

// evaluateWhereConditionWithDB evaluates a WHERE condition with database context for EXISTS
func evaluateWhereConditionWithDB(expr ast.ExprNode, db *Database, table *Table, row Row) (bool, error) {
	switch e := expr.(type) {
	case *ast.BinaryOperationExpr:
		return evaluateBinaryOperationWithDB(e, db, table, row)
	case *ast.ColumnNameExpr:
		// Column reference - treat as boolean
		colIndex := table.GetColumnIndex(e.Name.Name.String())
		if colIndex == -1 {
			return false, fmt.Errorf("column %s does not exist", e.Name.Name.String())
		}
		value := row.Values[colIndex]
		return isTruthy(value), nil
	case *ast.IsNullExpr:
		return evaluateIsNullExpression(e, table, row)
	case *ast.BetweenExpr:
		return evaluateBetweenExpression(e, table, row)
	case *ast.PatternInExpr:
		return evaluateInExpression(e, table, row)
	case *ast.ParenthesesExpr:
		// Handle parentheses by evaluating the inner expression
		return evaluateWhereConditionWithDB(e.Expr, db, table, row)
	case *ast.PatternLikeOrIlikeExpr:
		return evaluateLikeExpression(e, table, row)
	case *ast.PatternRegexpExpr:
		return evaluateRegexpExpression(e, table, row)
	case *ast.ExistsSubqueryExpr:
		return evaluateExistsExpression(e, db, table, row)
	case *ast.UnaryOperationExpr:
		// Handle logical NOT
		if e.Op == opcode.Not {
			return evaluateNotExpressionWithDB(e, db, table, row)
		}
		return false, fmt.Errorf("unsupported unary operator in WHERE: %v", e.Op)
	default:
		return false, fmt.Errorf("unsupported WHERE expression type: %T", expr)
	}
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
	case *ast.IsNullExpr:
		return evaluateIsNullExpression(e, table, row)
	case *ast.BetweenExpr:
		return evaluateBetweenExpression(e, table, row)
	case *ast.PatternInExpr:
		return evaluateInExpression(e, table, row)
	case *ast.ParenthesesExpr:
		// Handle parentheses by evaluating the inner expression
		return evaluateWhereCondition(e.Expr, table, row)
	case *ast.PatternLikeOrIlikeExpr:
		return evaluateLikeExpression(e, table, row)
	case *ast.PatternRegexpExpr:
		return evaluateRegexpExpression(e, table, row)
	case *ast.ExistsSubqueryExpr:
		// Need access to database for EXISTS subqueries
		return false, fmt.Errorf("EXISTS subqueries require database context - use ExecuteSelectWithDatabase")
	case *ast.UnaryOperationExpr:
		// Handle logical NOT
		if e.Op == opcode.Not {
			return evaluateNotExpression(e, table, row)
		}
		return false, fmt.Errorf("unsupported unary operator in WHERE: %v", e.Op)
	default:
		return false, fmt.Errorf("unsupported WHERE expression type: %T", expr)
	}
}

// evaluateBinaryOperation evaluates binary operations like =, >, <, etc.
func evaluateBinaryOperation(expr *ast.BinaryOperationExpr, table *Table, row Row) (bool, error) {
	// Handle logical operators differently - they need boolean evaluation
	switch expr.Op {
	case opcode.LogicAnd:
		leftResult, err := evaluateWhereCondition(expr.L, table, row)
		if err != nil {
			return false, err
		}
		rightResult, err := evaluateWhereCondition(expr.R, table, row)
		if err != nil {
			return false, err
		}
		return leftResult && rightResult, nil
		
	case opcode.LogicOr:
		leftResult, err := evaluateWhereCondition(expr.L, table, row)
		if err != nil {
			return false, err
		}
		rightResult, err := evaluateWhereCondition(expr.R, table, row)
		if err != nil {
			return false, err
		}
		return leftResult || rightResult, nil
	}
	
	// For comparison operators, evaluate as values
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
	case opcode.Regexp:
		return evaluateRegexpOperation(leftVal, rightVal)
	default:
		return false, fmt.Errorf("unsupported binary operator: %v", expr.Op)
	}
}

// evaluateIsNullExpression evaluates IS NULL and IS NOT NULL expressions
func evaluateIsNullExpression(expr *ast.IsNullExpr, table *Table, row Row) (bool, error) {
	// Evaluate the expression being tested for null
	value, err := evaluateExpressionInRow(expr.Expr, table, row)
	if err != nil {
		return false, err
	}
	
	isNull := (value == nil)
	
	// Return based on whether it's IS NULL or IS NOT NULL
	if expr.Not {
		return !isNull, nil // IS NOT NULL
	}
	return isNull, nil // IS NULL
}

// evaluateBetweenExpression evaluates BETWEEN expressions
func evaluateBetweenExpression(expr *ast.BetweenExpr, table *Table, row Row) (bool, error) {
	// Evaluate the main expression
	value, err := evaluateExpressionInRow(expr.Expr, table, row)
	if err != nil {
		return false, err
	}
	
	// Evaluate the lower bound
	leftValue, err := evaluateExpressionInRow(expr.Left, table, row)
	if err != nil {
		return false, err
	}
	
	// Evaluate the upper bound
	rightValue, err := evaluateExpressionInRow(expr.Right, table, row)
	if err != nil {
		return false, err
	}
	
	// Compare: value >= leftValue AND value <= rightValue
	leftComparison := compareValues(value, leftValue)
	rightComparison := compareValues(value, rightValue)
	
	isBetween := leftComparison >= 0 && rightComparison <= 0
	
	// Handle NOT BETWEEN
	if expr.Not {
		return !isBetween, nil
	}
	return isBetween, nil
}

// evaluateInExpression evaluates IN expressions
func evaluateInExpression(expr *ast.PatternInExpr, table *Table, row Row) (bool, error) {
	// Evaluate the expression being tested
	value, err := evaluateExpressionInRow(expr.Expr, table, row)
	if err != nil {
		return false, err
	}
	
	// Check if value matches any of the values in the list
	for _, listExpr := range expr.List {
		listValue, err := evaluateExpressionInRow(listExpr, table, row)
		if err != nil {
			return false, err
		}
		
		if compareValues(value, listValue) == 0 {
			// Found a match
			if expr.Not {
				return false, nil // NOT IN - match found, return false
			}
			return true, nil // IN - match found, return true
		}
	}
	
	// No match found
	if expr.Not {
		return true, nil // NOT IN - no match, return true
	}
	return false, nil // IN - no match, return false
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
	case *ast.FuncCallExpr:
		return evaluateFunctionCall(e, table, row)
	case *ast.CaseExpr:
		return evaluateCaseExpression(e, table, row)
	case *ast.FuncCastExpr:
		return evaluateCastExpression(e, table, row)
	case *ast.UnaryOperationExpr:
		return evaluateUnaryOperation(e, table, row)
	case *ast.BinaryOperationExpr:
		// Handle binary operations (both arithmetic and comparison)
		leftVal, err := evaluateExpressionInRow(e.L, table, row)
		if err != nil {
			return nil, err
		}
		rightVal, err := evaluateExpressionInRow(e.R, table, row)
		if err != nil {
			return nil, err
		}
		return evaluateBinaryOperationValue(e.Op, leftVal, rightVal)
	case *ast.SubqueryExpr:
		// Scalar subqueries need database context - fall back to non-DB version will fail
		return nil, fmt.Errorf("scalar subqueries require database context - use evaluateExpressionInRowWithDB")
	default:
		return nil, fmt.Errorf("unsupported expression type in evaluation: %T", expr)
	}
}

// evaluateExpressionInRowWithDB evaluates an expression in the context of a row with database access
func evaluateExpressionInRowWithDB(expr ast.ExprNode, db *Database, table *Table, row Row) (interface{}, error) {
	switch e := expr.(type) {
	case *ast.ColumnNameExpr:
		colIndex := table.GetColumnIndex(e.Name.Name.String())
		if colIndex == -1 {
			return nil, fmt.Errorf("column %s does not exist", e.Name.Name.String())
		}
		return row.Values[colIndex], nil
	case ast.ValueExpr:
		return e.GetValue(), nil
	case *ast.FuncCallExpr:
		return evaluateFunctionCall(e, table, row)
	case *ast.CaseExpr:
		return evaluateCaseExpression(e, table, row)
	case *ast.FuncCastExpr:
		return evaluateCastExpression(e, table, row)
	case *ast.UnaryOperationExpr:
		return evaluateUnaryOperation(e, table, row)
	case *ast.BinaryOperationExpr:
		// Handle binary operations (both arithmetic and comparison)
		leftVal, err := evaluateExpressionInRowWithDB(e.L, db, table, row)
		if err != nil {
			return nil, err
		}
		rightVal, err := evaluateExpressionInRowWithDB(e.R, db, table, row)
		if err != nil {
			return nil, err
		}
		return evaluateBinaryOperationValue(e.Op, leftVal, rightVal)
	case *ast.SubqueryExpr:
		// Handle scalar subqueries
		return evaluateScalarSubquery(e, db, table, row)
	default:
		return nil, fmt.Errorf("unsupported expression type in evaluation: %T", expr)
	}
}

// evaluateScalarSubquery evaluates a scalar subquery and returns a single value
func evaluateScalarSubquery(subqueryExpr *ast.SubqueryExpr, db *Database, outerTable *Table, outerRow Row) (interface{}, error) {
	// Cast Query to SelectStmt
	subquery, ok := subqueryExpr.Query.(*ast.SelectStmt)
	if !ok {
		return nil, fmt.Errorf("scalar subquery must be a SELECT statement")
	}

	// Execute the subquery with correlated context if outer context is provided
	var result *SelectResult
	var err error
	if outerTable != nil {
		result, err = ExecuteSelectWithCorrelatedContext(db, subquery, outerTable, outerRow)
	} else {
		result, err = ExecuteSelect(db, subquery)
	}
	if err != nil {
		return nil, fmt.Errorf("error executing scalar subquery: %v", err)
	}

	// Scalar subquery must return exactly one row and one column
	if len(result.Rows) == 0 {
		return nil, nil // Returns NULL if no rows
	}
	
	if len(result.Rows) > 1 {
		return nil, fmt.Errorf("scalar subquery returned more than one row")
	}
	
	if len(result.Rows[0]) == 0 {
		return nil, fmt.Errorf("scalar subquery returned no columns")
	}
	
	if len(result.Rows[0]) > 1 {
		return nil, fmt.Errorf("scalar subquery returned more than one column")
	}
	
	// Return the single value
	return result.Rows[0][0], nil
}

// evaluateBinaryOperationValue evaluates binary operations (arithmetic and comparison)
func evaluateBinaryOperationValue(op opcode.Op, left, right interface{}) (interface{}, error) {
	// Handle NULL values for arithmetic operations
	if (op == opcode.Plus || op == opcode.Minus || op == opcode.Mul || op == opcode.Div || op == opcode.Mod) &&
		(left == nil || right == nil) {
		return nil, nil
	}

	switch op {
	// Arithmetic operations
	case opcode.Plus, opcode.Minus, opcode.Mul, opcode.Div, opcode.Mod:
		// Convert to numeric values
		leftNum, err := toFloat64(left)
		if err != nil {
			return nil, fmt.Errorf("invalid numeric value in arithmetic operation: %v", err)
		}
		
		rightNum, err := toFloat64(right)
		if err != nil {
			return nil, fmt.Errorf("invalid numeric value in arithmetic operation: %v", err)
		}

		switch op {
		case opcode.Plus:
			return leftNum + rightNum, nil
		case opcode.Minus:
			return leftNum - rightNum, nil
		case opcode.Mul:
			return leftNum * rightNum, nil
		case opcode.Div:
			if rightNum == 0 {
				return nil, nil // Division by zero returns NULL in MySQL
			}
			return leftNum / rightNum, nil
		case opcode.Mod:
			if rightNum == 0 {
				return nil, nil // Modulo by zero returns NULL in MySQL
			}
			return float64(int64(leftNum) % int64(rightNum)), nil
		}

	// Comparison operations
	case opcode.EQ:
		return compareValues(left, right) == 0, nil
	case opcode.NE:
		return compareValues(left, right) != 0, nil
	case opcode.LT:
		return compareValues(left, right) < 0, nil
	case opcode.LE:
		return compareValues(left, right) <= 0, nil
	case opcode.GT:
		return compareValues(left, right) > 0, nil
	case opcode.GE:
		return compareValues(left, right) >= 0, nil

	// Logical operations
	case opcode.LogicAnd:
		return isTruthy(left) && isTruthy(right), nil
	case opcode.LogicOr:
		return isTruthy(left) || isTruthy(right), nil

	// Pattern matching operations
	case opcode.Regexp:
		return evaluateRegexpOperation(left, right)

	default:
		return nil, fmt.Errorf("unsupported binary operator: %v", op)
	}

	return nil, fmt.Errorf("unreachable code in binary operation")
}

// evaluateCastExpression evaluates CAST expressions like CAST(value AS type)
func evaluateCastExpression(castExpr *ast.FuncCastExpr, table *Table, row Row) (interface{}, error) {
	// Evaluate the expression being cast
	value, err := evaluateExpressionInRow(castExpr.Expr, table, row)
	if err != nil {
		return nil, fmt.Errorf("error evaluating CAST expression: %v", err)
	}

	if value == nil {
		return nil, nil
	}

	// Get the target type - use String() method of the FieldType
	targetType := strings.ToUpper(castExpr.Tp.String())

	// Handle common MySQL type names
	if strings.Contains(targetType, "CHAR") || strings.Contains(targetType, "TEXT") {
		return fmt.Sprintf("%v", value), nil
	}
	if strings.Contains(targetType, "INT") || strings.Contains(targetType, "BIGINT") {
		return toInt64(value)
	}
	if strings.Contains(targetType, "DECIMAL") || strings.Contains(targetType, "FLOAT") || strings.Contains(targetType, "DOUBLE") {
		return toFloat64(value)
	}
	if strings.Contains(targetType, "DATE") && !strings.Contains(targetType, "TIME") {
		dateStr := fmt.Sprintf("%v", value)
		t, err := parseDateTime(dateStr)
		if err != nil {
			return nil, fmt.Errorf("CAST: cannot convert to DATE: %v", err)
		}
		return t.Format("2006-01-02"), nil
	}
	if strings.Contains(targetType, "DATETIME") || strings.Contains(targetType, "TIMESTAMP") {
		dateStr := fmt.Sprintf("%v", value)
		t, err := parseDateTime(dateStr)
		if err != nil {
			return nil, fmt.Errorf("CAST: cannot convert to DATETIME: %v", err)
		}
		return t.Format("2006-01-02 15:04:05"), nil
	}

	// Default to string conversion
	return fmt.Sprintf("%v", value), nil
}

// evaluateUnaryOperation evaluates unary operations like -value
func evaluateUnaryOperation(unaryExpr *ast.UnaryOperationExpr, table *Table, row Row) (interface{}, error) {
	// Evaluate the operand
	value, err := evaluateExpressionInRow(unaryExpr.V, table, row)
	if err != nil {
		return nil, err
	}

	if value == nil {
		return nil, nil
	}

	switch unaryExpr.Op {
	case opcode.Minus:
		// Unary minus
		num, err := toFloat64(value)
		if err != nil {
			return nil, fmt.Errorf("unary minus requires numeric value: %v", err)
		}
		return -num, nil
	case opcode.Plus:
		// Unary plus (no-op)
		num, err := toFloat64(value)
		if err != nil {
			return nil, fmt.Errorf("unary plus requires numeric value: %v", err)
		}
		return num, nil
	default:
		return nil, fmt.Errorf("unsupported unary operator: %v", unaryExpr.Op)
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
		match, err := evaluateWhereConditionWithDB(whereExpr, db, table, row)
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

// inferColumnNameFromExpression generates a column name from an expression
func inferColumnNameFromExpression(expr ast.ExprNode) string {
	switch e := expr.(type) {
	case *ast.ColumnNameExpr:
		return e.Name.Name.String()
	case *ast.FuncCallExpr:
		// Generate function call representation like "UPPER(name)"
		funcName := strings.ToUpper(e.FnName.L)
		if len(e.Args) == 0 {
			return fmt.Sprintf("%s()", funcName)
		}
		var argNames []string
		for _, arg := range e.Args {
			argNames = append(argNames, inferColumnNameFromExpression(arg))
		}
		return fmt.Sprintf("%s(%s)", funcName, strings.Join(argNames, ","))
	case *ast.BinaryOperationExpr:
		// Generate arithmetic expression representation like "(price * quantity)"
		leftName := inferColumnNameFromExpression(e.L)
		rightName := inferColumnNameFromExpression(e.R)
		return fmt.Sprintf("(%s %s %s)", leftName, e.Op.String(), rightName)
	case *ast.CaseExpr:
		// Generate CASE expression representation
		return "CASE"
	case ast.ValueExpr:
		// For literal values, use their string representation
		return fmt.Sprintf("%v", e.GetValue())
	default:
		return "expr"
	}
}

// evaluateBinaryOperationWithDB evaluates binary operations with database context
func evaluateBinaryOperationWithDB(expr *ast.BinaryOperationExpr, db *Database, table *Table, row Row) (bool, error) {
	// Handle logical operators differently - they need boolean evaluation
	switch expr.Op {
	case opcode.LogicAnd:
		leftResult, err := evaluateWhereConditionWithDB(expr.L, db, table, row)
		if err != nil {
			return false, err
		}
		rightResult, err := evaluateWhereConditionWithDB(expr.R, db, table, row)
		if err != nil {
			return false, err
		}
		return leftResult && rightResult, nil
		
	case opcode.LogicOr:
		leftResult, err := evaluateWhereConditionWithDB(expr.L, db, table, row)
		if err != nil {
			return false, err
		}
		rightResult, err := evaluateWhereConditionWithDB(expr.R, db, table, row)
		if err != nil {
			return false, err
		}
		return leftResult || rightResult, nil
	}
	
	// For comparison operators, evaluate as values
	leftVal, err := evaluateExpressionInRowWithDB(expr.L, db, table, row)
	if err != nil {
		return false, err
	}

	rightVal, err := evaluateExpressionInRowWithDB(expr.R, db, table, row)
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
	case opcode.Regexp:
		return evaluateRegexpOperation(leftVal, rightVal)
	default:
		return false, fmt.Errorf("unsupported binary operator: %v", expr.Op)
	}
}

// evaluateNotExpressionWithDB evaluates logical NOT with database context
func evaluateNotExpressionWithDB(notExpr *ast.UnaryOperationExpr, db *Database, table *Table, row Row) (bool, error) {
	// Evaluate the inner expression
	result, err := evaluateWhereConditionWithDB(notExpr.V, db, table, row)
	if err != nil {
		return false, err
	}
	
	// Return the logical negation
	return !result, nil
}

// ExecuteSelectWithCorrelatedContext executes a SELECT statement with access to outer table context for correlated subqueries
func ExecuteSelectWithCorrelatedContext(db *Database, stmt *ast.SelectStmt, outerTable *Table, outerRow Row) (*SelectResult, error) {
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
		return executeAggregateQueryWithCorrelatedContext(table, stmt.Fields.Fields, stmt.Where, stmt.GroupBy, stmt.Having, stmt.Limit, db, outerTable, outerRow)
	}

	// Get rows from the table, potentially using indexes
	rows, err := getRowsWithOptimizationAndCorrelatedContext(db, table, stmt.Where, outerTable, outerRow)
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

	// If not SELECT *, process individual columns/expressions
	var expressions []ast.ExprNode
	if len(selectedColumns) == 0 {
		for _, field := range stmt.Fields.Fields {
			// Generate column name (use alias if present, otherwise infer from expression)
			var colName string
			if field.AsName.L != "" {
				colName = field.AsName.L
			} else {
				colName = inferColumnNameFromExpression(field.Expr)
			}
			
			selectedColumns = append(selectedColumns, colName)
			expressions = append(expressions, field.Expr)
		}
	}

	// Build result rows
	var resultRows [][]interface{}
	for _, row := range rows {
		var resultRow []interface{}
		
		if len(expressions) > 0 {
			// Evaluate expressions for each column
			for _, expr := range expressions {
				value, err := evaluateExpressionInRowWithCorrelatedContext(expr, db, table, row, outerTable, outerRow)
				if err != nil {
					return nil, fmt.Errorf("error evaluating SELECT expression: %v", err)
				}
				resultRow = append(resultRow, value)
			}
		} else {
			// Use column indexes for SELECT *
			resultRow = make([]interface{}, len(columnIndexes))
			for i, colIndex := range columnIndexes {
				resultRow[i] = row.Values[colIndex]
			}
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

// getRowsWithOptimizationAndCorrelatedContext gets rows from table with correlated context
func getRowsWithOptimizationAndCorrelatedContext(db *Database, table *Table, whereExpr ast.ExprNode, outerTable *Table, outerRow Row) ([]Row, error) {
	// If no WHERE clause, return all rows
	if whereExpr == nil {
		return table.GetRows(), nil
	}

	// Try to use index optimization for simple equality conditions
	if indexedRows, used := tryIndexOptimization(db, table, whereExpr); used {
		return indexedRows, nil
	}

	// Fall back to full table scan with correlated context
	allRows := table.GetRows()
	var filteredRows []Row

	for _, row := range allRows {
		match, err := evaluateWhereConditionWithCorrelatedContext(whereExpr, db, table, row, outerTable, outerRow)
		if err != nil {
			return nil, fmt.Errorf("error evaluating WHERE clause: %v", err)
		}
		if match {
			filteredRows = append(filteredRows, row)
		}
	}

	return filteredRows, nil
}

// evaluateWhereConditionWithCorrelatedContext evaluates a WHERE condition with correlated context
func evaluateWhereConditionWithCorrelatedContext(expr ast.ExprNode, db *Database, table *Table, row Row, outerTable *Table, outerRow Row) (bool, error) {
	switch e := expr.(type) {
	case *ast.BinaryOperationExpr:
		return evaluateBinaryOperationWithCorrelatedContext(e, db, table, row, outerTable, outerRow)
	case *ast.ColumnNameExpr:
		// Column reference - treat as boolean
		colIndex := table.GetColumnIndex(e.Name.Name.String())
		if colIndex == -1 {
			return false, fmt.Errorf("column %s does not exist", e.Name.Name.String())
		}
		value := row.Values[colIndex]
		return isTruthy(value), nil
	case *ast.IsNullExpr:
		return evaluateIsNullExpressionWithCorrelatedContext(e, table, row, outerTable, outerRow)
	case *ast.BetweenExpr:
		return evaluateBetweenExpressionWithCorrelatedContext(e, table, row, outerTable, outerRow)
	case *ast.PatternInExpr:
		return evaluateInExpressionWithCorrelatedContext(e, table, row, outerTable, outerRow)
	case *ast.ParenthesesExpr:
		// Handle parentheses by evaluating the inner expression
		return evaluateWhereConditionWithCorrelatedContext(e.Expr, db, table, row, outerTable, outerRow)
	case *ast.PatternLikeOrIlikeExpr:
		return evaluateLikeExpressionWithCorrelatedContext(e, table, row, outerTable, outerRow)
	case *ast.PatternRegexpExpr:
		return evaluateRegexpExpressionWithCorrelatedContext(e, table, row, outerTable, outerRow)
	case *ast.ExistsSubqueryExpr:
		return evaluateExistsExpression(e, db, table, row)
	case *ast.UnaryOperationExpr:
		// Handle logical NOT
		if e.Op == opcode.Not {
			return evaluateNotExpressionWithCorrelatedContext(e, db, table, row, outerTable, outerRow)
		}
		return false, fmt.Errorf("unsupported unary operator in WHERE: %v", e.Op)
	default:
		return false, fmt.Errorf("unsupported WHERE expression type: %T", expr)
	}
}

// evaluateExpressionInRowWithCorrelatedContext evaluates an expression with correlated context
func evaluateExpressionInRowWithCorrelatedContext(expr ast.ExprNode, db *Database, table *Table, row Row, outerTable *Table, outerRow Row) (interface{}, error) {
	switch e := expr.(type) {
	case *ast.ColumnNameExpr:
		columnName := e.Name.Name.String()
		
		// Handle qualified column names first (e.g., "u.id", "table.column")
		if e.Name.Table.L != "" && outerTable != nil {
			// Qualified column - this likely refers to the outer table in a correlated subquery
			// In a correlated subquery, qualified references usually refer to the outer context
			outerColIndex := outerTable.GetColumnIndex(columnName)
			if outerColIndex != -1 {
				return outerRow.Values[outerColIndex], nil
			}
			// If not found in outer table, fall through to try inner table
		}
		
		// Try to resolve column in the current (inner) table
		colIndex := table.GetColumnIndex(columnName)
		if colIndex != -1 {
			return row.Values[colIndex], nil
		}
		
		// If not found in inner table and we have outer context, try outer table
		if outerTable != nil {
			outerColIndex := outerTable.GetColumnIndex(columnName)
			if outerColIndex != -1 {
				return outerRow.Values[outerColIndex], nil
			}
		}
		
		return nil, fmt.Errorf("column %s does not exist", columnName)
	case ast.ValueExpr:
		return e.GetValue(), nil
	case *ast.FuncCallExpr:
		return evaluateFunctionCall(e, table, row)
	case *ast.CaseExpr:
		return evaluateCaseExpression(e, table, row)
	case *ast.FuncCastExpr:
		return evaluateCastExpression(e, table, row)
	case *ast.UnaryOperationExpr:
		return evaluateUnaryOperation(e, table, row)
	case *ast.BinaryOperationExpr:
		// Handle binary operations with correlated context
		leftVal, err := evaluateExpressionInRowWithCorrelatedContext(e.L, db, table, row, outerTable, outerRow)
		if err != nil {
			return nil, err
		}
		rightVal, err := evaluateExpressionInRowWithCorrelatedContext(e.R, db, table, row, outerTable, outerRow)
		if err != nil {
			return nil, err
		}
		return evaluateBinaryOperationValue(e.Op, leftVal, rightVal)
	case *ast.SubqueryExpr:
		// Handle scalar subqueries with correlated context
		return evaluateScalarSubqueryWithCorrelatedContext(e, db, table, row, outerTable, outerRow)
	default:
		return nil, fmt.Errorf("unsupported expression type in evaluation: %T", expr)
	}
}
