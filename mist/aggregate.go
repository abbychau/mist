package mist

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pingcap/tidb/pkg/parser/ast"
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
func executeAggregateQuery(table *Table, fields []*ast.SelectField, whereExpr ast.ExprNode, limit *ast.Limit) (*SelectResult, error) {
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

	// Detect aggregate functions
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
