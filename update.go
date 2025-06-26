package mist

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pingcap/tidb/pkg/parser/ast"
)

// parseDate attempts to parse various date formats and return YYYY-MM-DD format
func parseDate(dateStr string) (string, error) {
	// Common date formats to try
	formats := []string{
		"2006-01-02",          // YYYY-MM-DD
		"2006/01/02",          // YYYY/MM/DD
		"01/02/2006",          // MM/DD/YYYY
		"02/01/2006",          // DD/MM/YYYY
		"2006-01-02 15:04:05", // YYYY-MM-DD HH:MM:SS
		"2006/01/02 15:04:05", // YYYY/MM/DD HH:MM:SS
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t.Format("2006-01-02"), nil
		}
	}

	return "", fmt.Errorf("unable to parse date: %s", dateStr)
}

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

			// Validate foreign key constraints for the updated row
			if err := db.ValidateForeignKeys(table, newRow.Values); err != nil {
				return 0, fmt.Errorf("foreign key constraint violation: %v", err)
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

	// First, handle ON UPDATE triggers for columns with ON UPDATE CURRENT_TIMESTAMP
	for i, col := range table.Columns {
		if col.OnUpdate == "CURRENT_TIMESTAMP" {
			// Check if this column is being explicitly updated in the SET clause
			isExplicitlyUpdated := false
			for _, assignment := range assignments {
				if assignment.Column.Name.String() == col.Name {
					isExplicitlyUpdated = true
					break
				}
			}

			// If not explicitly updated, apply the ON UPDATE trigger
			if !isExplicitlyUpdated {
				newValues[i] = time.Now().Format("2006-01-02 15:04:05")
			}
		}
	}

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

	case TypeDecimal:
		// Convert to string representation for DECIMAL
		switch v := value.(type) {
		case string:
			return v, nil
		case float64:
			return fmt.Sprintf("%.10f", v), nil
		case float32:
			return fmt.Sprintf("%.10f", v), nil
		case int64:
			return fmt.Sprintf("%d", v), nil
		case int:
			return fmt.Sprintf("%d", v), nil
		default:
			// Handle MyDecimal and other types by converting to string
			str := fmt.Sprintf("%v", v)
			// Clean up the string if it contains type information
			if strings.Contains(str, "KindMysqlDecimal") {
				// Extract just the numeric part
				parts := strings.Fields(str)
				if len(parts) > 1 {
					return parts[1], nil
				}
			}
			return str, nil
		}

	case TypeTimestamp:
		// Convert to string representation for timestamps
		switch v := value.(type) {
		case string:
			// Validate timestamp format if needed
			return v, nil
		default:
			return fmt.Sprintf("%v", v), nil
		}

	case TypeDate:
		// Convert to string representation for dates
		switch v := value.(type) {
		case string:
			// Validate date format (YYYY-MM-DD)
			if len(v) == 10 && v[4] == '-' && v[7] == '-' {
				return v, nil
			}
			// Try to parse and format the date
			if parsed, err := parseDate(v); err == nil {
				return parsed, nil
			}
			return v, nil
		default:
			str := fmt.Sprintf("%v", v)
			if parsed, err := parseDate(str); err == nil {
				return parsed, nil
			}
			return str, nil
		}

	case TypeEnum:
		// Convert to string for ENUM
		return fmt.Sprintf("%v", value), nil

	default:
		return value, nil
	}
}
