package mist

import (
	"fmt"
	"strings"

	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/opcode"
)

// JoinResult represents the result of a JOIN operation
type JoinResult struct {
	Columns    []string
	TableNames []string // Which table each column comes from
	Rows       [][]interface{}
}

// ExecuteSelectWithJoin processes a SELECT statement with JOIN
func ExecuteSelectWithJoin(db *Database, stmt *ast.SelectStmt) (*SelectResult, error) {
	// Parse the JOIN structure
	joinInfo, err := parseJoinStructure(db, stmt.From)
	if err != nil {
		return nil, err
	}

	// Perform the JOIN operation
	joinResult, err := performJoin(joinInfo)
	if err != nil {
		return nil, err
	}

	// Apply WHERE clause if present
	if stmt.Where != nil {
		filteredRows, err := filterJoinedRows(stmt.Where, joinResult)
		if err != nil {
			return nil, fmt.Errorf("error evaluating WHERE clause: %v", err)
		}
		joinResult.Rows = filteredRows
	}

	// Select specific columns
	result, err := selectColumnsFromJoin(stmt.Fields.Fields, joinResult)
	if err != nil {
		return nil, err
	}

	// Apply LIMIT clause if present
	if stmt.Limit != nil {
		result.Rows = applyLimitToJoinRows(result.Rows, stmt.Limit)
	}

	return result, nil
}

// JoinInfo contains information about tables and join conditions
type JoinInfo struct {
	LeftTable   *Table
	RightTable  *Table
	LeftAlias   string
	RightAlias  string
	JoinType    string
	OnCondition ast.ExprNode
}

// parseJoinStructure extracts join information from the FROM clause
func parseJoinStructure(db *Database, from *ast.TableRefsClause) (*JoinInfo, error) {
	if from == nil || from.TableRefs == nil {
		return nil, fmt.Errorf("no tables specified")
	}

	join := from.TableRefs
	if join == nil {
		return nil, fmt.Errorf("not a JOIN operation")
	}

	// Check if this is a comma-separated table join (cross join)
	if innerJoin, ok := join.Left.(*ast.Join); ok && innerJoin.Tp == 0 && innerJoin.Right == nil {
		// This is comma-separated tables: FROM table1, table2
		// Left table is in innerJoin.Left, right table is in join.Right

		// Get left table
		leftSource, ok := innerJoin.Left.(*ast.TableSource)
		if !ok {
			return nil, fmt.Errorf("complex left table not supported")
		}

		leftTableName, ok := leftSource.Source.(*ast.TableName)
		if !ok {
			return nil, fmt.Errorf("subqueries not supported in JOIN")
		}

		leftTable, err := db.GetTable(leftTableName.Name.String())
		if err != nil {
			return nil, fmt.Errorf("left table error: %v", err)
		}

		// Get right table
		rightSource, ok := join.Right.(*ast.TableSource)
		if !ok {
			return nil, fmt.Errorf("complex right table not supported")
		}

		rightTableName, ok := rightSource.Source.(*ast.TableName)
		if !ok {
			return nil, fmt.Errorf("subqueries not supported in JOIN")
		}

		rightTable, err := db.GetTable(rightTableName.Name.String())
		if err != nil {
			return nil, fmt.Errorf("right table error: %v", err)
		}

		// Get aliases
		leftAlias := leftTableName.Name.String()
		if leftSource.AsName.String() != "" {
			leftAlias = leftSource.AsName.String()
		}

		rightAlias := rightTableName.Name.String()
		if rightSource.AsName.String() != "" {
			rightAlias = rightSource.AsName.String()
		}

		return &JoinInfo{
			LeftTable:   leftTable,
			RightTable:  rightTable,
			LeftAlias:   leftAlias,
			RightAlias:  rightAlias,
			JoinType:    "CROSS", // Comma-separated tables are cross joins
			OnCondition: nil,     // No ON condition for comma-separated tables
		}, nil
	}

	// Handle explicit JOIN syntax
	// Get left table
	leftSource, ok := join.Left.(*ast.TableSource)
	if !ok {
		return nil, fmt.Errorf("complex left table not supported")
	}

	leftTableName, ok := leftSource.Source.(*ast.TableName)
	if !ok {
		return nil, fmt.Errorf("subqueries not supported in JOIN")
	}

	leftTable, err := db.GetTable(leftTableName.Name.String())
	if err != nil {
		return nil, fmt.Errorf("left table error: %v", err)
	}

	// Get right table
	rightSource, ok := join.Right.(*ast.TableSource)
	if !ok {
		return nil, fmt.Errorf("complex right table not supported")
	}

	rightTableName, ok := rightSource.Source.(*ast.TableName)
	if !ok {
		return nil, fmt.Errorf("subqueries not supported in JOIN")
	}

	rightTable, err := db.GetTable(rightTableName.Name.String())
	if err != nil {
		return nil, fmt.Errorf("right table error: %v", err)
	}

	// Get aliases
	leftAlias := leftTableName.Name.String()
	if leftSource.AsName.String() != "" {
		leftAlias = leftSource.AsName.String()
	}

	rightAlias := rightTableName.Name.String()
	if rightSource.AsName.String() != "" {
		rightAlias = rightSource.AsName.String()
	}

	// Determine join type
	joinType := "INNER"
	switch join.Tp {
	case ast.LeftJoin:
		joinType = "LEFT"
	case ast.RightJoin:
		joinType = "RIGHT"
	case ast.CrossJoin:
		joinType = "CROSS"
	}

	var onCondition ast.ExprNode
	if join.On != nil {
		onCondition = join.On.Expr
	}

	return &JoinInfo{
		LeftTable:   leftTable,
		RightTable:  rightTable,
		LeftAlias:   leftAlias,
		RightAlias:  rightAlias,
		JoinType:    joinType,
		OnCondition: onCondition,
	}, nil
}

// performJoin executes the actual join operation
func performJoin(joinInfo *JoinInfo) (*JoinResult, error) {
	// Create column mapping
	var columns []string
	var tableNames []string

	// Add left table columns
	for _, col := range joinInfo.LeftTable.Columns {
		columns = append(columns, fmt.Sprintf("%s.%s", joinInfo.LeftAlias, col.Name))
		tableNames = append(tableNames, joinInfo.LeftAlias)
	}

	// Add right table columns
	for _, col := range joinInfo.RightTable.Columns {
		columns = append(columns, fmt.Sprintf("%s.%s", joinInfo.RightAlias, col.Name))
		tableNames = append(tableNames, joinInfo.RightAlias)
	}

	result := &JoinResult{
		Columns:    columns,
		TableNames: tableNames,
		Rows:       make([][]interface{}, 0),
	}

	leftRows := joinInfo.LeftTable.GetRows()
	rightRows := joinInfo.RightTable.GetRows()

	// Perform INNER JOIN (can be extended for other join types)
	for _, leftRow := range leftRows {
		for _, rightRow := range rightRows {
			// Check join condition
			if joinInfo.OnCondition != nil {
				match, err := evaluateJoinCondition(joinInfo.OnCondition, joinInfo, leftRow, rightRow)
				if err != nil {
					return nil, fmt.Errorf("error evaluating join condition: %v", err)
				}
				if !match {
					continue
				}
			}

			// Combine rows
			combinedRow := make([]interface{}, 0, len(leftRow.Values)+len(rightRow.Values))
			combinedRow = append(combinedRow, leftRow.Values...)
			combinedRow = append(combinedRow, rightRow.Values...)

			result.Rows = append(result.Rows, combinedRow)
		}
	}

	return result, nil
}

// evaluateJoinCondition evaluates the ON condition for a join
func evaluateJoinCondition(expr ast.ExprNode, joinInfo *JoinInfo, leftRow, rightRow Row) (bool, error) {
	switch e := expr.(type) {
	case *ast.BinaryOperationExpr:
		leftVal, err := evaluateJoinExpression(e.L, joinInfo, leftRow, rightRow)
		if err != nil {
			return false, err
		}

		rightVal, err := evaluateJoinExpression(e.R, joinInfo, leftRow, rightRow)
		if err != nil {
			return false, err
		}

		// For now, only support equality joins
		return compareValues(leftVal, rightVal) == 0, nil

	default:
		return false, fmt.Errorf("unsupported join condition type: %T", expr)
	}
}

// evaluateJoinExpression evaluates an expression in the context of a join
func evaluateJoinExpression(expr ast.ExprNode, joinInfo *JoinInfo, leftRow, rightRow Row) (interface{}, error) {
	switch e := expr.(type) {
	case *ast.ColumnNameExpr:
		// Determine which table the column belongs to
		colName := e.Name.Name.String()
		tableName := ""

		if e.Name.Table.String() != "" {
			tableName = e.Name.Table.String()
		}

		// Try to find the column in left table
		if tableName == "" || tableName == joinInfo.LeftAlias {
			colIndex := joinInfo.LeftTable.GetColumnIndex(colName)
			if colIndex != -1 {
				return leftRow.Values[colIndex], nil
			}
		}

		// Try to find the column in right table
		if tableName == "" || tableName == joinInfo.RightAlias {
			colIndex := joinInfo.RightTable.GetColumnIndex(colName)
			if colIndex != -1 {
				return rightRow.Values[colIndex], nil
			}
		}

		return nil, fmt.Errorf("column %s not found in joined tables", colName)

	case ast.ValueExpr:
		return e.GetValue(), nil

	default:
		return nil, fmt.Errorf("unsupported expression type in join: %T", expr)
	}
}

// filterJoinedRows applies WHERE clause to joined results
func filterJoinedRows(whereExpr ast.ExprNode, joinResult *JoinResult) ([][]interface{}, error) {
	var filteredRows [][]interface{}

	for _, row := range joinResult.Rows {
		match, err := evaluateWhereConditionOnJoinResult(whereExpr, joinResult, row)
		if err != nil {
			return nil, fmt.Errorf("error evaluating WHERE clause on join result: %v", err)
		}
		if match {
			filteredRows = append(filteredRows, row)
		}
	}

	return filteredRows, nil
}

// evaluateWhereConditionOnJoinResult evaluates a WHERE condition on a joined row
func evaluateWhereConditionOnJoinResult(expr ast.ExprNode, joinResult *JoinResult, row []interface{}) (bool, error) {
	switch e := expr.(type) {
	case *ast.BinaryOperationExpr:
		// Handle logical operators differently - they need boolean evaluation
		switch e.Op {
		case opcode.LogicAnd:
			leftResult, err := evaluateWhereConditionOnJoinResult(e.L, joinResult, row)
			if err != nil {
				return false, err
			}
			rightResult, err := evaluateWhereConditionOnJoinResult(e.R, joinResult, row)
			if err != nil {
				return false, err
			}
			return leftResult && rightResult, nil
		case opcode.LogicOr:
			leftResult, err := evaluateWhereConditionOnJoinResult(e.L, joinResult, row)
			if err != nil {
				return false, err
			}
			rightResult, err := evaluateWhereConditionOnJoinResult(e.R, joinResult, row)
			if err != nil {
				return false, err
			}
			return leftResult || rightResult, nil
		}
		
		// For comparison operators, evaluate as values
		leftVal, err := evaluateExpressionOnJoinResult(e.L, joinResult, row)
		if err != nil {
			return false, err
		}

		rightVal, err := evaluateExpressionOnJoinResult(e.R, joinResult, row)
		if err != nil {
			return false, err
		}

		switch e.Op {
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
		default:
			return false, fmt.Errorf("unsupported binary operator in WHERE clause: %v", e.Op)
		}

	case *ast.ColumnNameExpr:
		// For single column expressions, just check if they're truthy
		val, err := evaluateExpressionOnJoinResult(e, joinResult, row)
		if err != nil {
			return false, err
		}
		return isTruthy(val), nil

	case ast.ValueExpr:
		val := e.GetValue()
		return isTruthy(val), nil
		
	case *ast.IsNullExpr:
		return evaluateIsNullExpressionOnJoinResult(e, joinResult, row)
		
	case *ast.BetweenExpr:
		return evaluateBetweenExpressionOnJoinResult(e, joinResult, row)
		
	case *ast.PatternInExpr:
		return evaluateInExpressionOnJoinResult(e, joinResult, row)
		
	case *ast.ParenthesesExpr:
		// Handle parentheses by evaluating the inner expression
		return evaluateWhereConditionOnJoinResult(e.Expr, joinResult, row)

	default:
		return false, fmt.Errorf("unsupported expression type in WHERE clause: %T", expr)
	}
}

// evaluateExpressionOnJoinResult evaluates an expression in the context of a joined row
func evaluateExpressionOnJoinResult(expr ast.ExprNode, joinResult *JoinResult, row []interface{}) (interface{}, error) {
	switch e := expr.(type) {
	case *ast.BinaryOperationExpr:
		// Handle binary operations by evaluating both sides and applying the operator
		leftVal, err := evaluateExpressionOnJoinResult(e.L, joinResult, row)
		if err != nil {
			return nil, err
		}

		rightVal, err := evaluateExpressionOnJoinResult(e.R, joinResult, row)
		if err != nil {
			return nil, err
		}

		// Check if this is an arithmetic operation
		switch e.Op {
		case opcode.Plus, opcode.Minus, opcode.Mul, opcode.Div, opcode.Mod:
			return evaluateBinaryOperationValue(e.Op, leftVal, rightVal)
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
			leftBool := isTruthy(leftVal)
			rightBool := isTruthy(rightVal)
			return leftBool && rightBool, nil
		case opcode.LogicOr:
			leftBool := isTruthy(leftVal)
			rightBool := isTruthy(rightVal)
			return leftBool || rightBool, nil
		default:
			return nil, fmt.Errorf("unsupported binary operator in expression evaluation: %v", e.Op)
		}

	case *ast.ColumnNameExpr:
		// Find the column in the join result
		colName := e.Name.Name.String()
		tableName := e.Name.Table.String()

		// Build the full column name
		fullColName := colName
		if tableName != "" {
			fullColName = fmt.Sprintf("%s.%s", tableName, colName)
		}

		// Find the column index
		colIndex := -1
		for i, col := range joinResult.Columns {
			if col == fullColName || (tableName == "" && strings.HasSuffix(col, "."+colName)) {
				colIndex = i
				break
			}
		}

		if colIndex == -1 {
			return nil, fmt.Errorf("column %s not found in join result", fullColName)
		}

		if colIndex >= len(row) {
			return nil, fmt.Errorf("column index %d out of range for row with %d columns", colIndex, len(row))
		}

		return row[colIndex], nil

	case ast.ValueExpr:
		return e.GetValue(), nil

	case *ast.FuncCallExpr:
		return evaluateFunctionCallOnJoinResult(e, joinResult, row)

	case *ast.CaseExpr:
		return evaluateCaseExpressionOnJoinResult(e, joinResult, row)

	case *ast.FuncCastExpr:
		return evaluateCastExpressionOnJoinResult(e, joinResult, row)

	case *ast.UnaryOperationExpr:
		return evaluateUnaryOperationOnJoinResult(e, joinResult, row)

	default:
		return nil, fmt.Errorf("unsupported expression type in join result evaluation: %T", expr)
	}
}

// selectColumnsFromJoin selects specific columns from join result
func selectColumnsFromJoin(fields []*ast.SelectField, joinResult *JoinResult) (*SelectResult, error) {
	// Handle SELECT *
	if len(fields) == 1 && fields[0].WildCard != nil {
		return &SelectResult{
			Columns: joinResult.Columns,
			Rows:    joinResult.Rows,
		}, nil
	}

	// Check if this contains aggregate functions
	if hasAggregateFunction(fields) {
		return executeAggregateOnJoinResult(fields, joinResult)
	}

	// Handle specific columns and expressions
	var selectedColumns []string
	var expressions []ast.ExprNode

	for _, field := range fields {
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

	// Build result rows by evaluating expressions
	var resultRows [][]interface{}
	for _, row := range joinResult.Rows {
		var resultRow []interface{}
		for _, expr := range expressions {
			value, err := evaluateExpressionOnJoinResult(expr, joinResult, row)
			if err != nil {
				return nil, fmt.Errorf("error evaluating JOIN SELECT expression: %v", err)
			}
			resultRow = append(resultRow, value)
		}
		resultRows = append(resultRows, resultRow)
	}

	return &SelectResult{
		Columns: selectedColumns,
		Rows:    resultRows,
	}, nil
}

// executeAggregateOnJoinResult executes aggregate functions on join results
func executeAggregateOnJoinResult(fields []*ast.SelectField, joinResult *JoinResult) (*SelectResult, error) {
	// Convert JoinResult to a format that aggregate functions can work with
	var aggregates []AggregateFunction
	var columnNames []string

	for _, field := range fields {
		if aggFunc, err := detectAggregateFunction(field); err != nil {
			return nil, err
		} else if aggFunc != nil {
			aggregates = append(aggregates, *aggFunc)
			// Create column name for aggregate
			if aggFunc.IsStar {
				columnNames = append(columnNames, fmt.Sprintf("%s(*)", aggFunc.Type.String()))
			} else {
				columnNames = append(columnNames, fmt.Sprintf("%s(%s)", aggFunc.Type.String(), aggFunc.Column))
			}
		} else {
			return nil, fmt.Errorf("mixing aggregate and non-aggregate columns not supported without GROUP BY")
		}
	}

	// Compute aggregate values on join result
	values, err := computeAggregatesOnJoinResult(aggregates, joinResult)
	if err != nil {
		return nil, err
	}

	// Return single row with aggregate results
	return &SelectResult{
		Columns: columnNames,
		Rows:    [][]interface{}{values},
	}, nil
}

// computeAggregatesOnJoinResult calculates aggregate function values on join results
func computeAggregatesOnJoinResult(aggregates []AggregateFunction, joinResult *JoinResult) ([]interface{}, error) {
	results := make([]interface{}, len(aggregates))

	for i, aggFunc := range aggregates {
		switch aggFunc.Type {
		case AggCount:
			if aggFunc.IsStar {
				results[i] = int64(len(joinResult.Rows))
			} else {
				// Find column index in join result
				colIndex := findColumnInJoinResult(aggFunc.Column, joinResult)
				if colIndex == -1 {
					return nil, fmt.Errorf("column %s does not exist in join result", aggFunc.Column)
				}

				count := int64(0)
				seen := make(map[interface{}]bool)

				for _, row := range joinResult.Rows {
					value := row[colIndex]
					if value != nil {
						if aggFunc.IsDistinct {
							if !seen[value] {
								seen[value] = true
								count++
							}
						} else {
							count++
						}
					}
				}
				results[i] = count
			}

		case AggSum:
			colIndex := findColumnInJoinResult(aggFunc.Column, joinResult)
			if colIndex == -1 {
				return nil, fmt.Errorf("column %s does not exist in join result", aggFunc.Column)
			}

			sum := 0.0
			for _, row := range joinResult.Rows {
				value := row[colIndex]
				if value != nil {
					numValue, err := toFloat64Agg(value)
					if err != nil {
						return nil, fmt.Errorf("SUM requires numeric column: %v", err)
					}
					sum += numValue
				}
			}
			results[i] = sum

		case AggAvg:
			colIndex := findColumnInJoinResult(aggFunc.Column, joinResult)
			if colIndex == -1 {
				return nil, fmt.Errorf("column %s does not exist in join result", aggFunc.Column)
			}

			sum := 0.0
			count := 0
			for _, row := range joinResult.Rows {
				value := row[colIndex]
				if value != nil {
					numValue, err := toFloat64Agg(value)
					if err != nil {
						return nil, fmt.Errorf("AVG requires numeric column: %v", err)
					}
					sum += numValue
					count++
				}
			}

			if count == 0 {
				results[i] = nil
			} else {
				results[i] = sum / float64(count)
			}

		case AggMin:
			colIndex := findColumnInJoinResult(aggFunc.Column, joinResult)
			if colIndex == -1 {
				return nil, fmt.Errorf("column %s does not exist in join result", aggFunc.Column)
			}

			var minValue interface{}
			for _, row := range joinResult.Rows {
				value := row[colIndex]
				if value != nil {
					if minValue == nil || compareValues(value, minValue) < 0 {
						minValue = value
					}
				}
			}
			results[i] = minValue

		case AggMax:
			colIndex := findColumnInJoinResult(aggFunc.Column, joinResult)
			if colIndex == -1 {
				return nil, fmt.Errorf("column %s does not exist in join result", aggFunc.Column)
			}

			var maxValue interface{}
			for _, row := range joinResult.Rows {
				value := row[colIndex]
				if value != nil {
					if maxValue == nil || compareValues(value, maxValue) > 0 {
						maxValue = value
					}
				}
			}
			results[i] = maxValue
		}
	}

	return results, nil
}

// findColumnInJoinResult finds a column index in join result by name
func findColumnInJoinResult(columnName string, joinResult *JoinResult) int {
	for i, col := range joinResult.Columns {
		// Check exact match or suffix match (for qualified names)
		if strings.EqualFold(col, columnName) || strings.HasSuffix(strings.ToLower(col), "."+strings.ToLower(columnName)) {
			return i
		}
	}
	return -1
}

// applyLimitToJoinRows applies LIMIT clause to join result rows
func applyLimitToJoinRows(rows [][]interface{}, limit *ast.Limit) [][]interface{} {
	if limit == nil {
		return rows
	}

	// Parse offset and count
	offset := int64(0)
	count := int64(len(rows)) // default to all rows

	if limit.Offset != nil {
		if offsetExpr, ok := limit.Offset.(ast.ValueExpr); ok {
			if offsetVal := offsetExpr.GetValue(); offsetVal != nil {
				if offsetInt, ok := offsetVal.(int64); ok {
					offset = offsetInt
				}
			}
		}
	}

	if limit.Count != nil {
		if countExpr, ok := limit.Count.(ast.ValueExpr); ok {
			if countVal := countExpr.GetValue(); countVal != nil {
				if countInt, ok := countVal.(int64); ok {
					count = countInt
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

// evaluateIsNullExpressionOnJoinResult evaluates IS NULL expressions on joined rows
func evaluateIsNullExpressionOnJoinResult(expr *ast.IsNullExpr, joinResult *JoinResult, row []interface{}) (bool, error) {
	// Evaluate the expression being tested for null
	value, err := evaluateExpressionOnJoinResult(expr.Expr, joinResult, row)
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

// evaluateBetweenExpressionOnJoinResult evaluates BETWEEN expressions on joined rows
func evaluateBetweenExpressionOnJoinResult(expr *ast.BetweenExpr, joinResult *JoinResult, row []interface{}) (bool, error) {
	// Evaluate the main expression
	value, err := evaluateExpressionOnJoinResult(expr.Expr, joinResult, row)
	if err != nil {
		return false, err
	}
	
	// Evaluate the lower bound
	leftValue, err := evaluateExpressionOnJoinResult(expr.Left, joinResult, row)
	if err != nil {
		return false, err
	}
	
	// Evaluate the upper bound
	rightValue, err := evaluateExpressionOnJoinResult(expr.Right, joinResult, row)
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

// evaluateInExpressionOnJoinResult evaluates IN expressions on joined rows
func evaluateInExpressionOnJoinResult(expr *ast.PatternInExpr, joinResult *JoinResult, row []interface{}) (bool, error) {
	// Evaluate the expression being tested
	value, err := evaluateExpressionOnJoinResult(expr.Expr, joinResult, row)
	if err != nil {
		return false, err
	}
	
	// Check if value matches any of the values in the list
	for _, listExpr := range expr.List {
		listValue, err := evaluateExpressionOnJoinResult(listExpr, joinResult, row)
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

// evaluateCastExpressionOnJoinResult evaluates CAST expressions in JOIN context
func evaluateCastExpressionOnJoinResult(castExpr *ast.FuncCastExpr, joinResult *JoinResult, row []interface{}) (interface{}, error) {
	// Evaluate the expression being cast
	value, err := evaluateExpressionOnJoinResult(castExpr.Expr, joinResult, row)
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

// evaluateUnaryOperationOnJoinResult evaluates unary operations in JOIN context
func evaluateUnaryOperationOnJoinResult(unaryExpr *ast.UnaryOperationExpr, joinResult *JoinResult, row []interface{}) (interface{}, error) {
	// Evaluate the operand
	value, err := evaluateExpressionOnJoinResult(unaryExpr.V, joinResult, row)
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
