package mist

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pingcap/tidb/pkg/parser/ast"
)

// ExecuteInsert processes an INSERT statement
func ExecuteInsert(db *Database, stmt *ast.InsertStmt) error {
	tableName := stmt.Table.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName).Name.String()

	table, err := db.GetTable(tableName)
	if err != nil {
		return err
	}

	// Handle different types of INSERT statements
	if len(stmt.Lists) > 0 {
		// INSERT INTO table VALUES (...), (...), ...
		return executeInsertValues(db, table, stmt)
	}

	return fmt.Errorf("unsupported INSERT statement type")
}

// executeInsertValues handles INSERT ... VALUES statements
func executeInsertValues(db *Database, table *Table, stmt *ast.InsertStmt) error {
	// Get column names if specified
	var targetColumns []string
	if len(stmt.Columns) > 0 {
		for _, col := range stmt.Columns {
			targetColumns = append(targetColumns, col.Name.String())
		}
	} else {
		// If no columns specified, use all columns in order
		for _, col := range table.Columns {
			targetColumns = append(targetColumns, col.Name)
		}
	}

	// Validate that all target columns exist
	columnIndexes := make([]int, len(targetColumns))
	for i, colName := range targetColumns {
		index := table.GetColumnIndex(colName)
		if index == -1 {
			return fmt.Errorf("column %s does not exist in table %s", colName, table.Name)
		}
		columnIndexes[i] = index
	}

	// Process each row of values
	for _, valueList := range stmt.Lists {
		if len(valueList) != len(targetColumns) {
			return fmt.Errorf("column count mismatch: expected %d, got %d", len(targetColumns), len(valueList))
		}

		// Create a row with default values
		rowValues := make([]interface{}, len(table.Columns))

		// Fill in the specified values
		for i, expr := range valueList {
			colIndex := columnIndexes[i]
			value, err := evaluateExpression(expr, table.Columns[colIndex].Type)
			if err != nil {
				return fmt.Errorf("error evaluating value for column %s: %v", table.Columns[colIndex].Name, err)
			}
			rowValues[colIndex] = value
		}

		// Add the row to the table with index updates
		if err := table.AddRowWithIndexManager(rowValues, db.IndexManager); err != nil {
			return err
		}
	}

	return nil
}

// evaluateExpression converts an AST expression to a Go value
func evaluateExpression(expr ast.ExprNode, expectedType ColumnType) (interface{}, error) {
	switch e := expr.(type) {
	case ast.ValueExpr:
		return evaluateValueExpr(e, expectedType)
	case *ast.UnaryOperationExpr:
		// Handle negative numbers
		if e.Op == '-' {
			val, err := evaluateExpression(e.V, expectedType)
			if err != nil {
				return nil, err
			}
			switch v := val.(type) {
			case int64:
				return -v, nil
			case float64:
				return -v, nil
			default:
				return nil, fmt.Errorf("cannot apply unary minus to %T", v)
			}
		}
		return nil, fmt.Errorf("unsupported unary operation: %v", e.Op)
	default:
		return nil, fmt.Errorf("unsupported expression type: %T", expr)
	}
}

// evaluateValueExpr converts a ValueExpr to a Go value
func evaluateValueExpr(expr ast.ValueExpr, expectedType ColumnType) (interface{}, error) {
	// For now, let's use a simpler approach and get the raw value
	// The exact API depends on the TiDB version, so we'll use reflection-like approach
	value := expr.GetValue()

	if value == nil {
		return nil, nil
	}

	switch expectedType {
	case TypeInt:
		switch v := value.(type) {
		case int64:
			return v, nil
		case int:
			return int64(v), nil
		case int32:
			return int64(v), nil
		case string:
			return strconv.ParseInt(v, 10, 64)
		case float64:
			return int64(v), nil
		case float32:
			return int64(v), nil
		default:
			// Try to convert using string representation for unknown types
			str := fmt.Sprintf("%v", v)
			if i, err := strconv.ParseInt(str, 10, 64); err == nil {
				return i, nil
			}
			return nil, fmt.Errorf("cannot convert %T to int", v)
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
			// Try to convert using string representation for unknown types (like MyDecimal)
			str := fmt.Sprintf("%v", v)
			if f, err := strconv.ParseFloat(str, 64); err == nil {
				return f, nil
			}
			return nil, fmt.Errorf("cannot convert %T to float", v)
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
		case string:
			str := strings.ToLower(v)
			return str == "true" || str == "1", nil
		default:
			return false, nil
		}

	default:
		return value, nil
	}
}

// parseInsertSQL is a helper function to parse and execute INSERT
func parseInsertSQL(db *Database, sql string) error {
	astNode, err := parse(sql)
	if err != nil {
		return fmt.Errorf("parse error: %v", err)
	}

	stmt, ok := (*astNode).(*ast.InsertStmt)
	if !ok {
		return fmt.Errorf("not an INSERT statement")
	}

	return ExecuteInsert(db, stmt)
}
