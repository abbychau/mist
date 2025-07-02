package mist

import (
	"fmt"
	"regexp"

	"github.com/abbychau/mysql-parser/ast"
	"github.com/abbychau/mysql-parser/opcode"
)

// Helper functions for correlated subquery support

// evaluateBinaryOperationWithCorrelatedContext evaluates binary operations with correlated context
func evaluateBinaryOperationWithCorrelatedContext(expr *ast.BinaryOperationExpr, db *Database, table *Table, row Row, outerTable *Table, outerRow Row) (bool, error) {
	// Handle logical operators differently - they need boolean evaluation
	switch expr.Op {
	case opcode.LogicAnd:
		leftResult, err := evaluateWhereConditionWithCorrelatedContext(expr.L, db, table, row, outerTable, outerRow)
		if err != nil {
			return false, err
		}
		rightResult, err := evaluateWhereConditionWithCorrelatedContext(expr.R, db, table, row, outerTable, outerRow)
		if err != nil {
			return false, err
		}
		return leftResult && rightResult, nil
		
	case opcode.LogicOr:
		leftResult, err := evaluateWhereConditionWithCorrelatedContext(expr.L, db, table, row, outerTable, outerRow)
		if err != nil {
			return false, err
		}
		rightResult, err := evaluateWhereConditionWithCorrelatedContext(expr.R, db, table, row, outerTable, outerRow)
		if err != nil {
			return false, err
		}
		return leftResult || rightResult, nil
	}
	
	// For comparison operators, evaluate as values
	leftVal, err := evaluateExpressionInRowWithCorrelatedContext(expr.L, db, table, row, outerTable, outerRow)
	if err != nil {
		return false, err
	}

	rightVal, err := evaluateExpressionInRowWithCorrelatedContext(expr.R, db, table, row, outerTable, outerRow)
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

// evaluateIsNullExpressionWithCorrelatedContext evaluates IS NULL expressions with correlated context
func evaluateIsNullExpressionWithCorrelatedContext(expr *ast.IsNullExpr, table *Table, row Row, outerTable *Table, outerRow Row) (bool, error) {
	// Evaluate the expression being tested for null
	value, err := evaluateExpressionInRowWithCorrelatedContext(expr.Expr, nil, table, row, outerTable, outerRow)
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

// evaluateBetweenExpressionWithCorrelatedContext evaluates BETWEEN expressions with correlated context
func evaluateBetweenExpressionWithCorrelatedContext(expr *ast.BetweenExpr, table *Table, row Row, outerTable *Table, outerRow Row) (bool, error) {
	// Evaluate the main expression
	value, err := evaluateExpressionInRowWithCorrelatedContext(expr.Expr, nil, table, row, outerTable, outerRow)
	if err != nil {
		return false, err
	}
	
	// Evaluate the lower bound
	leftValue, err := evaluateExpressionInRowWithCorrelatedContext(expr.Left, nil, table, row, outerTable, outerRow)
	if err != nil {
		return false, err
	}
	
	// Evaluate the upper bound
	rightValue, err := evaluateExpressionInRowWithCorrelatedContext(expr.Right, nil, table, row, outerTable, outerRow)
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

// evaluateInExpressionWithCorrelatedContext evaluates IN expressions with correlated context
func evaluateInExpressionWithCorrelatedContext(expr *ast.PatternInExpr, table *Table, row Row, outerTable *Table, outerRow Row) (bool, error) {
	// Evaluate the expression being tested
	value, err := evaluateExpressionInRowWithCorrelatedContext(expr.Expr, nil, table, row, outerTable, outerRow)
	if err != nil {
		return false, err
	}
	
	// Check if value matches any of the values in the list
	for _, listExpr := range expr.List {
		listValue, err := evaluateExpressionInRowWithCorrelatedContext(listExpr, nil, table, row, outerTable, outerRow)
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

// evaluateLikeExpressionWithCorrelatedContext evaluates LIKE expressions with correlated context
func evaluateLikeExpressionWithCorrelatedContext(expr *ast.PatternLikeOrIlikeExpr, table *Table, row Row, outerTable *Table, outerRow Row) (bool, error) {
	// Evaluate the expression being tested
	value, err := evaluateExpressionInRowWithCorrelatedContext(expr.Expr, nil, table, row, outerTable, outerRow)
	if err != nil {
		return false, err
	}
	
	// Evaluate the pattern
	pattern, err := evaluateExpressionInRowWithCorrelatedContext(expr.Pattern, nil, table, row, outerTable, outerRow)
	if err != nil {
		return false, err
	}
	
	// Handle NULL values - LIKE with NULL returns NULL (false in boolean context)
	if value == nil || pattern == nil {
		return false, nil
	}
	
	// Convert to strings
	valueStr := fmt.Sprintf("%v", value)
	patternStr := fmt.Sprintf("%v", pattern)
	
	// Convert SQL LIKE pattern to Go regex (using existing function)
	regexPattern := convertLikePatternToRegex(patternStr)
	
	// Compile and match
	matches, err := regexp.MatchString(regexPattern, valueStr)
	if err != nil {
		return false, fmt.Errorf("invalid LIKE pattern: %v", err)
	}
	
	// Handle NOT LIKE
	if expr.Not {
		return !matches, nil
	}
	
	return matches, nil
}

// evaluateRegexpExpressionWithCorrelatedContext evaluates REGEXP expressions with correlated context
func evaluateRegexpExpressionWithCorrelatedContext(expr *ast.PatternRegexpExpr, table *Table, row Row, outerTable *Table, outerRow Row) (bool, error) {
	// Evaluate the expression being tested
	value, err := evaluateExpressionInRowWithCorrelatedContext(expr.Expr, nil, table, row, outerTable, outerRow)
	if err != nil {
		return false, err
	}
	
	// Evaluate the pattern
	pattern, err := evaluateExpressionInRowWithCorrelatedContext(expr.Pattern, nil, table, row, outerTable, outerRow)
	if err != nil {
		return false, err
	}
	
	// Apply regex matching
	matches, err := evaluateRegexpOperation(value, pattern)
	if err != nil {
		return false, err
	}
	
	// Handle NOT REGEXP
	if expr.Not {
		return !matches, nil
	}
	
	return matches, nil
}

// evaluateNotExpressionWithCorrelatedContext evaluates logical NOT with correlated context
func evaluateNotExpressionWithCorrelatedContext(notExpr *ast.UnaryOperationExpr, db *Database, table *Table, row Row, outerTable *Table, outerRow Row) (bool, error) {
	// Evaluate the inner expression
	result, err := evaluateWhereConditionWithCorrelatedContext(notExpr.V, db, table, row, outerTable, outerRow)
	if err != nil {
		return false, err
	}
	
	// Return the logical negation
	return !result, nil
}

// evaluateScalarSubqueryWithCorrelatedContext evaluates scalar subqueries with correlated context
func evaluateScalarSubqueryWithCorrelatedContext(subqueryExpr *ast.SubqueryExpr, db *Database, outerTable *Table, outerRow Row, correlatedOuterTable *Table, correlatedOuterRow Row) (interface{}, error) {
	// Cast Query to SelectStmt
	subquery, ok := subqueryExpr.Query.(*ast.SelectStmt)
	if !ok {
		return nil, fmt.Errorf("scalar subquery must be a SELECT statement")
	}

	// Execute the subquery with correlated context
	result, err := ExecuteSelectWithCorrelatedContext(db, subquery, correlatedOuterTable, correlatedOuterRow)
	if err != nil {
		return nil, fmt.Errorf("error executing correlated scalar subquery: %v", err)
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

// executeAggregateQueryWithCorrelatedContext executes aggregate queries with correlated context
func executeAggregateQueryWithCorrelatedContext(table *Table, fields []*ast.SelectField, whereExpr ast.ExprNode, groupBy *ast.GroupByClause, having *ast.HavingClause, limit *ast.Limit, db *Database, outerTable *Table, outerRow Row) (*SelectResult, error) {
	// Get rows with correlated context for WHERE clause evaluation
	rows, err := getRowsWithOptimizationAndCorrelatedContext(db, table, whereExpr, outerTable, outerRow)
	if err != nil {
		return nil, err
	}

	// Process aggregate functions on the pre-filtered rows
	return processAggregateOnFilteredRows(table, fields, rows, groupBy, having, limit)
}

// processAggregateOnFilteredRows processes aggregate functions on pre-filtered rows
func processAggregateOnFilteredRows(table *Table, fields []*ast.SelectField, filteredRows []Row, groupBy *ast.GroupByClause, having *ast.HavingClause, limit *ast.Limit) (*SelectResult, error) {
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