package mist

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/abbychau/mysql-parser/ast"
	"github.com/abbychau/mysql-parser/opcode"
)

// AggregateType represents different types of aggregate functions
type AggregateType int

const (
	AggCount AggregateType = iota
	AggSum
	AggAvg
	AggMin
	AggMax
)

func (at AggregateType) String() string {
	switch at {
	case AggCount:
		return "COUNT"
	case AggSum:
		return "SUM"
	case AggAvg:
		return "AVG"
	case AggMin:
		return "MIN"
	case AggMax:
		return "MAX"
	default:
		return "UNKNOWN"
	}
}

// AggregateFunction represents an aggregate function in a query
type AggregateFunction struct {
	Type       AggregateType
	Column     string
	IsDistinct bool
	IsStar     bool // for COUNT(*)
}

// AggregateResult holds the result of aggregate computation
type AggregateResult struct {
	Functions []AggregateFunction
	Values    []interface{}
}

// detectAggregateFunction checks if a field contains an aggregate function
func detectAggregateFunction(field *ast.SelectField) (*AggregateFunction, error) {
	if funcCall, ok := field.Expr.(*ast.AggregateFuncExpr); ok {
		aggFunc := &AggregateFunction{
			IsDistinct: funcCall.Distinct,
		}

		// Determine function type
		switch strings.ToUpper(funcCall.F) {
		case "COUNT":
			aggFunc.Type = AggCount
		case "SUM":
			aggFunc.Type = AggSum
		case "AVG":
			aggFunc.Type = AggAvg
		case "MIN":
			aggFunc.Type = AggMin
		case "MAX":
			aggFunc.Type = AggMax
		default:
			return nil, fmt.Errorf("unsupported aggregate function: %s", funcCall.F)
		}

		// Handle arguments
		if len(funcCall.Args) == 0 {
			return nil, fmt.Errorf("aggregate function %s requires arguments", funcCall.F)
		}

		// Check for COUNT(*) - special case
		if aggFunc.Type == AggCount {
			// For COUNT(*), the argument might be represented differently
			// Let's check multiple ways to detect the wildcard
			firstArg := funcCall.Args[0]

			// Method 1: Check if it's a ColumnNameExpr with name "*"
			if colExpr, ok := firstArg.(*ast.ColumnNameExpr); ok {
				if colExpr.Name.Name.String() == "*" {
					aggFunc.IsStar = true
					return aggFunc, nil
				}
			}

			// Method 2: Check the string representation
			argStr := fmt.Sprintf("%v", firstArg)
			if strings.Contains(argStr, "*") {
				aggFunc.IsStar = true
				return aggFunc, nil
			}
		}

		// Get column name for other functions
		if colExpr, ok := funcCall.Args[0].(*ast.ColumnNameExpr); ok {
			aggFunc.Column = colExpr.Name.Name.String()
			return aggFunc, nil
		} else {
			// For COUNT(*), we might reach here, so let's handle it
			if aggFunc.Type == AggCount {
				aggFunc.IsStar = true
				return aggFunc, nil
			}
			return nil, fmt.Errorf("aggregate function %s only supports column references", funcCall.F)
		}
	}

	return nil, nil
}

// hasAggregateFunction checks if any field contains aggregate functions
func hasAggregateFunction(fields []*ast.SelectField) bool {
	for _, field := range fields {
		if aggFunc, _ := detectAggregateFunction(field); aggFunc != nil {
			return true
		}
	}
	return false
}

// executeAggregateQuery processes a SELECT query with aggregate functions
func executeAggregateQuery(table *Table, fields []*ast.SelectField, whereExpr ast.ExprNode, groupBy *ast.GroupByClause, having *ast.HavingClause, limit *ast.Limit) (*SelectResult, error) {
	// Get all rows and apply WHERE filter
	rows := table.GetRows()
	var filteredRows []Row

	if whereExpr != nil {
		for _, row := range rows {
			match, err := evaluateWhereCondition(whereExpr, table, row)
			if err != nil {
				return nil, fmt.Errorf("error evaluating WHERE clause: %v", err)
			}
			if match {
				filteredRows = append(filteredRows, row)
			}
		}
	} else {
		filteredRows = rows
	}

	// Check if we have GROUP BY
	if groupBy != nil && len(groupBy.Items) > 0 {
		return executeGroupByQuery(table, fields, filteredRows, groupBy, having, limit)
	}

	// No GROUP BY - process as simple aggregate query
	// All fields must be aggregates
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

	// Compute aggregate values
	values, err := computeAggregates(table, aggregates, filteredRows)
	if err != nil {
		return nil, err
	}

	// Create result with single row
	resultRows := [][]interface{}{values}

	// Apply LIMIT clause if present (though unusual for aggregates)
	if limit != nil {
		resultRows = applyLimitToRows(resultRows, limit)
	}

	// Return aggregate results
	return &SelectResult{
		Columns: columnNames,
		Rows:    resultRows,
	}, nil
}

// computeAggregates calculates the aggregate function values
func computeAggregates(table *Table, aggregates []AggregateFunction, rows []Row) ([]interface{}, error) {
	results := make([]interface{}, len(aggregates))

	for i, aggFunc := range aggregates {
		switch aggFunc.Type {
		case AggCount:
			if aggFunc.IsStar {
				results[i] = int64(len(rows))
			} else {
				// Count non-null values in the specified column
				colIndex := table.GetColumnIndex(aggFunc.Column)
				if colIndex == -1 {
					return nil, fmt.Errorf("column %s does not exist", aggFunc.Column)
				}

				count := int64(0)
				seen := make(map[interface{}]bool)

				for _, row := range rows {
					value := row.Values[colIndex]
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
			colIndex := table.GetColumnIndex(aggFunc.Column)
			if colIndex == -1 {
				return nil, fmt.Errorf("column %s does not exist", aggFunc.Column)
			}

			sum := 0.0
			for _, row := range rows {
				value := row.Values[colIndex]
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
			colIndex := table.GetColumnIndex(aggFunc.Column)
			if colIndex == -1 {
				return nil, fmt.Errorf("column %s does not exist", aggFunc.Column)
			}

			sum := 0.0
			count := 0
			for _, row := range rows {
				value := row.Values[colIndex]
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
			colIndex := table.GetColumnIndex(aggFunc.Column)
			if colIndex == -1 {
				return nil, fmt.Errorf("column %s does not exist", aggFunc.Column)
			}

			var minValue interface{}
			for _, row := range rows {
				value := row.Values[colIndex]
				if value != nil {
					if minValue == nil || compareValues(value, minValue) < 0 {
						minValue = value
					}
				}
			}
			results[i] = minValue

		case AggMax:
			colIndex := table.GetColumnIndex(aggFunc.Column)
			if colIndex == -1 {
				return nil, fmt.Errorf("column %s does not exist", aggFunc.Column)
			}

			var maxValue interface{}
			for _, row := range rows {
				value := row.Values[colIndex]
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

// toFloat64 converts various numeric types to float64 (reused from update.go)
func toFloat64Agg(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		// Try string conversion as fallback
		str := fmt.Sprintf("%v", v)
		return strconv.ParseFloat(str, 64)
	}
}

// applyLimitToRows applies LIMIT clause to result rows (shared function)
func applyLimitToRows(rows [][]interface{}, limit *ast.Limit) [][]interface{} {
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

// executeGroupByQuery processes a SELECT query with GROUP BY clause
func executeGroupByQuery(table *Table, fields []*ast.SelectField, rows []Row, groupBy *ast.GroupByClause, having *ast.HavingClause, limit *ast.Limit) (*SelectResult, error) {
	// Extract GROUP BY columns
	var groupByColumns []string
	var groupByIndexes []int

	for _, item := range groupBy.Items {
		if colExpr, ok := item.Expr.(*ast.ColumnNameExpr); ok {
			colName := colExpr.Name.Name.String()
			colIndex := table.GetColumnIndex(colName)
			if colIndex == -1 {
				return nil, fmt.Errorf("GROUP BY column %s does not exist", colName)
			}
			groupByColumns = append(groupByColumns, colName)
			groupByIndexes = append(groupByIndexes, colIndex)
		} else {
			return nil, fmt.Errorf("complex GROUP BY expressions not supported yet")
		}
	}

	// Group rows by the GROUP BY columns
	groups := make(map[string][]Row)

	for _, row := range rows {
		// Create group key from GROUP BY column values
		var keyParts []string
		for _, colIndex := range groupByIndexes {
			value := row.Values[colIndex]
			keyParts = append(keyParts, fmt.Sprintf("%v", value))
		}
		key := strings.Join(keyParts, "|")
		groups[key] = append(groups[key], row)
	}

	// Process SELECT fields to identify group columns and aggregates
	var resultColumns []string
	var groupColumnIndexes []int
	var aggregates []AggregateFunction
	var isAggregate []bool

	for _, field := range fields {
		if aggFunc, err := detectAggregateFunction(field); err != nil {
			return nil, err
		} else if aggFunc != nil {
			// This is an aggregate function
			aggregates = append(aggregates, *aggFunc)
			isAggregate = append(isAggregate, true)

			// Create column name for aggregate
			if aggFunc.IsStar {
				resultColumns = append(resultColumns, fmt.Sprintf("%s(*)", aggFunc.Type.String()))
			} else {
				resultColumns = append(resultColumns, fmt.Sprintf("%s(%s)", aggFunc.Type.String(), aggFunc.Column))
			}
		} else {
			// This is a regular column - must be in GROUP BY
			if colExpr, ok := field.Expr.(*ast.ColumnNameExpr); ok {
				colName := colExpr.Name.Name.String()
				colIndex := table.GetColumnIndex(colName)
				if colIndex == -1 {
					return nil, fmt.Errorf("column %s does not exist", colName)
				}

				// Check if this column is in GROUP BY
				found := false
				for _, groupCol := range groupByColumns {
					if groupCol == colName {
						found = true
						break
					}
				}

				if !found {
					return nil, fmt.Errorf("column %s must appear in GROUP BY clause", colName)
				}

				groupColumnIndexes = append(groupColumnIndexes, colIndex)
				isAggregate = append(isAggregate, false)
				resultColumns = append(resultColumns, colName)
			} else {
				return nil, fmt.Errorf("complex expressions in SELECT not supported yet")
			}
		}
	}

	// Build result rows
	var resultRows [][]interface{}

	for _, groupRows := range groups {
		if len(groupRows) == 0 {
			continue
		}

		resultRow := make([]interface{}, len(resultColumns))
		aggIndex := 0
		groupIndex := 0

		for i, isAgg := range isAggregate {
			if isAgg {
				// Compute aggregate for this group
				aggValues, err := computeAggregates(table, []AggregateFunction{aggregates[aggIndex]}, groupRows)
				if err != nil {
					return nil, err
				}
				resultRow[i] = aggValues[0]
				aggIndex++
			} else {
				// Use group column value (all rows in group have same value)
				colIndex := groupColumnIndexes[groupIndex]
				resultRow[i] = groupRows[0].Values[colIndex]
				groupIndex++
			}
		}

		resultRows = append(resultRows, resultRow)
	}

	// Apply HAVING clause if present
	if having != nil {
		filteredRows, err := applyHavingClause(table, resultRows, resultColumns, isAggregate, aggregates, having)
		if err != nil {
			return nil, err
		}
		resultRows = filteredRows
	}

	// Apply LIMIT clause if present
	if limit != nil {
		resultRows = applyLimitToRows(resultRows, limit)
	}

	return &SelectResult{
		Columns: resultColumns,
		Rows:    resultRows,
	}, nil
}

// applyHavingClause filters result rows based on HAVING clause conditions
func applyHavingClause(table *Table, resultRows [][]interface{}, resultColumns []string, isAggregate []bool, aggregates []AggregateFunction, having *ast.HavingClause) ([][]interface{}, error) {
	if having == nil {
		return resultRows, nil
	}

	var filteredRows [][]interface{}

	for _, row := range resultRows {
		// Create a virtual row context for HAVING evaluation
		virtualTable := &Table{
			Name:    "having_context",
			Columns: make([]Column, len(resultColumns)),
		}

		// Set up virtual columns for aggregate result evaluation
		for i, colName := range resultColumns {
			colType := TypeText // default
			if len(row) > i && row[i] != nil {
				colType = inferColumnType(row[i])
			}
			virtualTable.Columns[i] = Column{
				Name: colName,
				Type: colType,
			}
		}

		virtualRow := Row{Values: row}

		// Evaluate HAVING condition against the result row
		match, err := evaluateHavingCondition(having.Expr, virtualTable, virtualRow, table, isAggregate, aggregates)
		if err != nil {
			return nil, fmt.Errorf("error evaluating HAVING clause: %v", err)
		}

		if match {
			filteredRows = append(filteredRows, row)
		}
	}

	return filteredRows, nil
}

// evaluateHavingCondition evaluates a HAVING condition against a result row
func evaluateHavingCondition(expr ast.ExprNode, virtualTable *Table, resultRow Row, originalTable *Table, isAggregate []bool, aggregates []AggregateFunction) (bool, error) {
	switch e := expr.(type) {
	case *ast.BinaryOperationExpr:
		return evaluateHavingBinaryOperation(e, virtualTable, resultRow, originalTable, isAggregate, aggregates)
	case *ast.AggregateFuncExpr:
		// Direct aggregate function in HAVING - should be compared with a value
		return false, fmt.Errorf("aggregate function in HAVING must be part of a comparison")
	case *ast.ColumnNameExpr:
		// Column reference in HAVING - treat as boolean
		colIndex := virtualTable.GetColumnIndex(e.Name.Name.String())
		if colIndex == -1 {
			return false, fmt.Errorf("column %s does not exist in HAVING context", e.Name.Name.String())
		}
		value := resultRow.Values[colIndex]
		return isTruthy(value), nil
	default:
		return false, fmt.Errorf("unsupported HAVING expression type: %T", expr)
	}
}

// evaluateHavingBinaryOperation evaluates binary operations in HAVING clause
func evaluateHavingBinaryOperation(expr *ast.BinaryOperationExpr, virtualTable *Table, resultRow Row, originalTable *Table, isAggregate []bool, aggregates []AggregateFunction) (bool, error) {
	leftVal, err := evaluateHavingExpression(expr.L, virtualTable, resultRow, originalTable, isAggregate, aggregates)
	if err != nil {
		return false, err
	}

	rightVal, err := evaluateHavingExpression(expr.R, virtualTable, resultRow, originalTable, isAggregate, aggregates)
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
		return false, fmt.Errorf("unsupported binary operator in HAVING: %v", expr.Op)
	}
}

// evaluateHavingExpression evaluates an expression in HAVING context
func evaluateHavingExpression(expr ast.ExprNode, virtualTable *Table, resultRow Row, originalTable *Table, isAggregate []bool, aggregates []AggregateFunction) (interface{}, error) {
	switch e := expr.(type) {
	case *ast.ColumnNameExpr:
		// Look for the column in the result columns
		colIndex := virtualTable.GetColumnIndex(e.Name.Name.String())
		if colIndex != -1 {
			return resultRow.Values[colIndex], nil
		}
		
		// Check if this matches any aggregate function pattern
		colName := e.Name.Name.String()
		for i, isAgg := range isAggregate {
			if isAgg {
				aggFunc := aggregates[i]
				var expectedName string
				if aggFunc.IsStar {
					expectedName = fmt.Sprintf("%s(*)", aggFunc.Type.String())
				} else {
					expectedName = fmt.Sprintf("%s(%s)", aggFunc.Type.String(), aggFunc.Column)
				}
				if expectedName == colName {
					return resultRow.Values[i], nil
				}
			}
		}
		
		return nil, fmt.Errorf("column %s does not exist in HAVING context", colName)
		
	case *ast.AggregateFuncExpr:
		// Direct aggregate function - need to match it with our computed aggregates
		aggFunc, err := detectAggregateFunction(&ast.SelectField{Expr: e})
		if err != nil {
			return nil, err
		}
		
		// Find matching aggregate in our list
		for i, computedAgg := range aggregates {
			if aggFunc.Type == computedAgg.Type && 
			   aggFunc.Column == computedAgg.Column && 
			   aggFunc.IsStar == computedAgg.IsStar {
				return resultRow.Values[i], nil
			}
		}
		
		return nil, fmt.Errorf("aggregate function %s not found in SELECT list", aggFunc.Type.String())
		
	case ast.ValueExpr:
		return e.GetValue(), nil
		
	default:
		return nil, fmt.Errorf("unsupported expression type in HAVING: %T", expr)
	}
}
