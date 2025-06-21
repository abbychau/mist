package mist

import (
	"fmt"
	"strconv"

	"github.com/pingcap/tidb/pkg/parser/ast"
)

// ExecuteUpdate processes an UPDATE statement
func ExecuteUpdate(db *Database, stmt *ast.UpdateStmt) (int, error) {
	// Get the table name from the first table reference
	tableRefs := stmt.TableRefs.TableRefs
	if tableRefs == nil {
		return 0, fmt.Errorf("no table specified in UPDATE statement")
	}

	tableSource, ok := tableRefs.Left.(*ast.TableSource)
	if !ok {
		return 0, fmt.Errorf("complex table references not supported in UPDATE")
	}

	tableName, ok := tableSource.Source.(*ast.TableName)
	if !ok {
		return 0, fmt.Errorf("subqueries not supported in UPDATE")
	}

	table, err := db.GetTable(tableName.Name.String())
	if err != nil {
		return 0, err
	}

	// Get all rows from the table
	rows := table.GetRows()
	updatedCount := 0

	// Process each row
	for i, row := range rows {
		// Check if row matches WHERE condition
		shouldUpdate := true
		if stmt.Where != nil {
			match, err := evaluateWhereCondition(stmt.Where, table, row)
			if err != nil {
				return 0, fmt.Errorf("error evaluating WHERE clause: %v", err)
			}
			shouldUpdate = match
		}

		if shouldUpdate {
			// Apply updates to this row
			newRow, err := applyUpdates(table, row, stmt.List)
			if err != nil {
				return 0, fmt.Errorf("error applying updates: %v", err)
			}

			// Update the row in place (thread-safe)
			table.mutex.Lock()
			table.Rows[i] = newRow
			table.mutex.Unlock()

			updatedCount++
		}
	}

	return updatedCount, nil
}

// applyUpdates applies the SET clauses to a row
func applyUpdates(table *Table, row Row, assignments []*ast.Assignment) (Row, error) {
	// Create a copy of the row values
	newValues := make([]interface{}, len(row.Values))
	copy(newValues, row.Values)

	// Apply each assignment
	for _, assignment := range assignments {
		colName := assignment.Column.Name.String()
		colIndex := table.GetColumnIndex(colName)
		if colIndex == -1 {
			return Row{}, fmt.Errorf("column %s does not exist", colName)
		}

		// Evaluate the new value
		newValue, err := evaluateUpdateExpression(assignment.Expr, table, row)
		if err != nil {
			return Row{}, fmt.Errorf("error evaluating expression for column %s: %v", colName, err)
		}

		// Convert the value to the appropriate type for the column
		convertedValue, err := convertValueToColumnType(newValue, table.Columns[colIndex].Type)
		if err != nil {
			return Row{}, fmt.Errorf("error converting value for column %s: %v", colName, err)
		}

		// Validate the converted value against column type
		if err := table.validateValue(colIndex, convertedValue); err != nil {
			return Row{}, err
		}

		newValues[colIndex] = convertedValue
	}

	return Row{Values: newValues}, nil
}

// evaluateUpdateExpression evaluates an expression in the context of an UPDATE
func evaluateUpdateExpression(expr ast.ExprNode, table *Table, row Row) (interface{}, error) {
	switch e := expr.(type) {
	case *ast.ColumnNameExpr:
		// Reference to another column in the same row
		colIndex := table.GetColumnIndex(e.Name.Name.String())
		if colIndex == -1 {
			return nil, fmt.Errorf("column %s does not exist", e.Name.Name.String())
		}
		return row.Values[colIndex], nil

	case ast.ValueExpr:
		// Literal value
		return e.GetValue(), nil

	case *ast.BinaryOperationExpr:
		// Arithmetic or other binary operations
		return evaluateBinaryExpressionForUpdate(e, table, row)

	case *ast.UnaryOperationExpr:
		// Unary operations like negation
		val, err := evaluateUpdateExpression(e.V, table, row)
		if err != nil {
			return nil, err
		}

		switch e.Op {
		case '-':
			return negateValue(val)
		default:
			return nil, fmt.Errorf("unsupported unary operator: %v", e.Op)
		}

	default:
		return nil, fmt.Errorf("unsupported expression type in UPDATE: %T", expr)
	}
}

// evaluateBinaryExpressionForUpdate handles binary operations in UPDATE expressions
func evaluateBinaryExpressionForUpdate(expr *ast.BinaryOperationExpr, table *Table, row Row) (interface{}, error) {
	leftVal, err := evaluateUpdateExpression(expr.L, table, row)
	if err != nil {
		return nil, err
	}

	rightVal, err := evaluateUpdateExpression(expr.R, table, row)
	if err != nil {
		return nil, err
	}

	return performArithmetic(leftVal, rightVal, expr.Op.String())
}

// negateValue negates a numeric value
func negateValue(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case int64:
		return -v, nil
	case int:
		return -v, nil
	case float64:
		return -v, nil
	case float32:
		return -v, nil
	default:
		return nil, fmt.Errorf("cannot negate non-numeric value: %T", v)
	}
}

// performArithmetic performs arithmetic operations between two values
func performArithmetic(left, right interface{}, op string) (interface{}, error) {
	// Convert values to float64 for arithmetic
	leftFloat, err := toFloat64(left)
	if err != nil {
		return nil, fmt.Errorf("left operand not numeric: %v", err)
	}

	rightFloat, err := toFloat64(right)
	if err != nil {
		return nil, fmt.Errorf("right operand not numeric: %v", err)
	}

	switch op {
	case "+", "plus":
		return leftFloat + rightFloat, nil
	case "-", "minus":
		return leftFloat - rightFloat, nil
	case "*", "mul":
		return leftFloat * rightFloat, nil
	case "/", "div":
		if rightFloat == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return leftFloat / rightFloat, nil
	default:
		return nil, fmt.Errorf("unsupported arithmetic operator: %s", op)
	}
}

// toFloat64 converts various numeric types to float64
func toFloat64(value interface{}) (float64, error) {
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

// parseUpdateSQL is a helper function to parse and execute UPDATE
func parseUpdateSQL(db *Database, sql string) (int, error) {
	astNode, err := parse(sql)
	if err != nil {
		return 0, fmt.Errorf("parse error: %v", err)
	}

	stmt, ok := (*astNode).(*ast.UpdateStmt)
	if !ok {
		return 0, fmt.Errorf("not an UPDATE statement")
	}

	return ExecuteUpdate(db, stmt)
}

// convertValueToColumnType converts a value to match the expected column type
func convertValueToColumnType(value interface{}, colType ColumnType) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	switch colType {
	case TypeInt:
		switch v := value.(type) {
		case int64:
			return v, nil
		case int:
			return int64(v), nil
		case float64:
			return int64(v), nil
		case float32:
			return int64(v), nil
		case string:
			return strconv.ParseInt(v, 10, 64)
		default:
			// Try string conversion as fallback
			str := fmt.Sprintf("%v", v)
			return strconv.ParseInt(str, 10, 64)
		}

	case TypeFloat:
		switch v := value.(type) {
		case float64:
			return v, nil
		case float32:
			return float64(v), nil
		case int64:
			return float64(v), nil
		case int:
			return float64(v), nil
		case string:
			return strconv.ParseFloat(v, 64)
		default:
			// Try string conversion as fallback
			str := fmt.Sprintf("%v", v)
			return strconv.ParseFloat(str, 64)
		}

	case TypeVarchar, TypeText:
		return fmt.Sprintf("%v", value), nil

	case TypeBool:
		switch v := value.(type) {
		case bool:
			return v, nil
		case int64:
			return v != 0, nil
		case int:
			return v != 0, nil
		case float64:
			return v != 0, nil
		case string:
			return strconv.ParseBool(v)
		default:
			return false, nil
		}

	default:
		return value, nil
	}
}
