package mist

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/opcode"
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
	} else if stmt.Select != nil {
		// INSERT INTO table SELECT ...
		return executeInsertSelect(db, table, stmt)
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

	// Check if we're inserting into an auto increment column
	autoIncrColIndex := table.GetAutoIncrementColumn()
	hasAutoIncrInTarget := false
	if autoIncrColIndex != -1 {
		for _, colIndex := range columnIndexes {
			if colIndex == autoIncrColIndex {
				hasAutoIncrInTarget = true
				break
			}
		}
	}

	// Process each row of values
	for _, valueList := range stmt.Lists {
		if len(valueList) != len(targetColumns) {
			return fmt.Errorf("column count mismatch: expected %d, got %d", len(targetColumns), len(valueList))
		}

		// Create a row with default values
		rowValues := make([]interface{}, len(table.Columns))

		// First, fill in default values for all columns
		for i, col := range table.Columns {
			if col.Default != nil {
				if col.Default == "CURRENT_TIMESTAMP" {
					rowValues[i] = time.Now().Format("2006-01-02 15:04:05")
				} else {
					// Convert the default value to the appropriate type
					convertedDefault, err := convertValueToColumnType(col.Default, col.Type)
					if err != nil {
						return fmt.Errorf("error converting default value for column %s: %v", col.Name, err)
					}
					rowValues[i] = convertedDefault
				}
			} else if !col.NotNull {
				rowValues[i] = nil
			} else {
				// Set appropriate default for NOT NULL columns without explicit default
				switch col.Type {
				case TypeInt:
					rowValues[i] = int64(0)
				case TypeFloat:
					rowValues[i] = float64(0)
				case TypeVarchar, TypeText:
					rowValues[i] = ""
				case TypeBool:
					rowValues[i] = false
				case TypeDecimal:
					rowValues[i] = "0.00"
				case TypeTimestamp:
					rowValues[i] = time.Now().Format("2006-01-02 15:04:05")
				case TypeDate:
					rowValues[i] = time.Now().Format("2006-01-02")
				case TypeEnum:
					// Use the first enum value as default if available
					if len(col.EnumValues) > 0 {
						rowValues[i] = col.EnumValues[0]
					} else {
						rowValues[i] = ""
					}
				case TypeTime:
					rowValues[i] = "00:00:00"
				case TypeYear:
					rowValues[i] = "2000"
				case TypeSet:
					rowValues[i] = "" // Empty SET is valid
				default:
					rowValues[i] = nil
				}
			}
		}

		// Fill in the specified values
		for i, expr := range valueList {
			colIndex := columnIndexes[i]

			// Handle auto increment column
			if colIndex == autoIncrColIndex {
				// Check if the value is NULL or 0 (should be auto-generated)
				value, err := evaluateExpression(expr, table.Columns[colIndex].Type)
				if err != nil {
					return fmt.Errorf("error evaluating value for column %s: %v", table.Columns[colIndex].Name, err)
				}

				// If value is NULL or 0, auto-generate it
				if value == nil || (value != nil && value.(int64) == 0) {
					rowValues[colIndex] = table.GetNextAutoIncrementValue()
				} else {
					rowValues[colIndex] = value
					// Update the auto increment counter if the inserted value is larger
					if intVal, ok := value.(int64); ok && intVal > table.AutoIncrCounter {
						table.AutoIncrCounter = intVal
					}
				}
			} else {
				value, err := evaluateExpression(expr, table.Columns[colIndex].Type)
				if err != nil {
					return fmt.Errorf("error evaluating value for column %s: %v", table.Columns[colIndex].Name, err)
				}
				rowValues[colIndex] = value
			}
		}

		// If auto increment column is not in target columns, auto-generate it
		if autoIncrColIndex != -1 && !hasAutoIncrInTarget {
			rowValues[autoIncrColIndex] = table.GetNextAutoIncrementValue()
		}

		// Validate foreign key constraints
		if err := db.ValidateForeignKeys(table, rowValues); err != nil {
			return fmt.Errorf("foreign key constraint violation: %v", err)
		}

		// Handle ON DUPLICATE KEY UPDATE if specified
		if stmt.OnDuplicate != nil {
			err := handleOnDuplicateKeyUpdate(db, table, rowValues, stmt.OnDuplicate)
			if err != nil {
				return fmt.Errorf("error handling ON DUPLICATE KEY UPDATE: %v", err)
			}
		} else {
			// Add the row to the table with index updates
			if err := table.AddRowWithIndexManager(rowValues, db.IndexManager); err != nil {
				return err
			}
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

	case TypeTimestamp, TypeDate:
		// Convert to string representation for timestamps and dates
		switch v := value.(type) {
		case string:
			return v, nil
		default:
			return fmt.Sprintf("%v", v), nil
		}

	case TypeTime:
		// Convert to string representation for time
		switch v := value.(type) {
		case string:
			return v, nil
		default:
			return fmt.Sprintf("%v", v), nil
		}

	case TypeYear:
		// Convert to string representation for year
		switch v := value.(type) {
		case string:
			return v, nil
		case int, int32, int64:
			return fmt.Sprintf("%v", v), nil
		default:
			return fmt.Sprintf("%v", v), nil
		}

	case TypeSet:
		// Convert to string representation for set
		switch v := value.(type) {
		case string:
			return v, nil
		default:
			return fmt.Sprintf("%v", v), nil
		}

	default:
		return value, nil
	}
}

// executeInsertSelect handles INSERT ... SELECT statements
func executeInsertSelect(db *Database, table *Table, stmt *ast.InsertStmt) error {
	// Get target column names if specified
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

	// Execute the SELECT statement
	var selectResult *SelectResult
	var err error

	// Handle SELECT statement or UNION operation
	if selectStmt, ok := stmt.Select.(*ast.SelectStmt); ok {
		// Create a temporary engine to execute the SELECT
		tempEngine := &SQLEngine{database: db}

		// Route the SELECT appropriately
		if tempEngine.isJoinQuery(selectStmt) {
			selectResult, err = ExecuteSelectWithJoin(db, selectStmt)
		} else if hasAggregateFunction(selectStmt.Fields.Fields) {
			sourceTable, tableErr := resolveTableFromSelect(db, selectStmt)
			if tableErr != nil {
				return fmt.Errorf("error resolving source table: %v", tableErr)
			}
			selectResult, err = executeAggregateQuery(sourceTable, selectStmt.Fields.Fields, selectStmt.Where, selectStmt.GroupBy, selectStmt.Having, selectStmt.Limit)
		} else {
			selectResult, err = ExecuteSelect(db, selectStmt)
		}
	} else if _, ok := stmt.Select.(*ast.SetOprStmt); ok {
		// UNION operations in INSERT...SELECT are not supported yet
		return fmt.Errorf("UNION operations in INSERT ... SELECT are not yet supported")
	} else {
		return fmt.Errorf("unsupported SELECT statement type in INSERT ... SELECT")
	}

	if err != nil {
		return fmt.Errorf("error executing SELECT in INSERT ... SELECT: %v", err)
	}

	// Validate column count compatibility
	if len(selectResult.Columns) != len(targetColumns) {
		return fmt.Errorf("column count mismatch: SELECT returns %d columns, INSERT expects %d", len(selectResult.Columns), len(targetColumns))
	}

	// Insert each row from the SELECT result
	for rowIndex, selectRow := range selectResult.Rows {
		// Create a full row with default values
		fullRow := make([]interface{}, len(table.Columns))

		// Set default values for all columns
		for i, col := range table.Columns {
			if col.AutoIncr {
				// Auto increment columns will be handled later
				fullRow[i] = nil
			} else if col.Default != nil {
				fullRow[i] = col.Default
			} else {
				fullRow[i] = nil
			}
		}

		// Apply values from SELECT result to target columns
		for i, value := range selectRow {
			if i < len(columnIndexes) {
				fullRow[columnIndexes[i]] = value
			}
		}

		// Handle auto increment columns
		for i, col := range table.Columns {
			if col.AutoIncr && fullRow[i] == nil {
				fullRow[i] = table.GetNextAutoIncrementValue()
			}
		}

		// Handle ON UPDATE CURRENT_TIMESTAMP for new inserts
		for i, col := range table.Columns {
			if col.Type == TypeTimestamp {
				if col.Default != nil && fmt.Sprintf("%v", col.Default) == "CURRENT_TIMESTAMP" && fullRow[i] == nil {
					fullRow[i] = time.Now().Format("2006-01-02 15:04:05")
				}
			}
		}

		// Validate foreign keys before inserting
		if err := db.ValidateForeignKeys(table, fullRow); err != nil {
			return fmt.Errorf("foreign key constraint violation in INSERT ... SELECT row %d: %v", rowIndex+1, err)
		}

		// Handle ON DUPLICATE KEY UPDATE if specified
		if stmt.OnDuplicate != nil {
			err = handleOnDuplicateKeyUpdate(db, table, fullRow, stmt.OnDuplicate)
			if err != nil {
				return fmt.Errorf("error handling ON DUPLICATE KEY UPDATE for row %d: %v", rowIndex+1, err)
			}
		} else {
			// Regular insert
			err = table.AddRowWithIndexManager(fullRow, db.IndexManager)
			if err != nil {
				return fmt.Errorf("error inserting row %d: %v", rowIndex+1, err)
			}
		}
	}

	return nil
}

// handleOnDuplicateKeyUpdate handles INSERT ... ON DUPLICATE KEY UPDATE logic
func handleOnDuplicateKeyUpdate(db *Database, table *Table, newRow []interface{}, onDuplicate []*ast.Assignment) error {
	// Find any primary key or unique constraint violations
	duplicateFound := false
	duplicateRowIndex := -1

	table.mutex.RLock()
	existingRows := table.GetRows()
	table.mutex.RUnlock()

	// Check for primary key duplicates
	for i, col := range table.Columns {
		if col.Primary && newRow[i] != nil {
			// Find existing row with same primary key
			for rowIdx, existingRow := range existingRows {
				if existingRow.Values[i] != nil && compareValues(newRow[i], existingRow.Values[i]) == 0 {
					duplicateFound = true
					duplicateRowIndex = rowIdx
					break
				}
			}
			if duplicateFound {
				break
			}
		}
	}

	// Check for unique constraint duplicates if no primary key duplicate found
	if !duplicateFound {
		for i, col := range table.Columns {
			if col.Unique && newRow[i] != nil {
				for rowIdx, existingRow := range existingRows {
					if existingRow.Values[i] != nil && compareValues(newRow[i], existingRow.Values[i]) == 0 {
						duplicateFound = true
						duplicateRowIndex = rowIdx
						break
					}
				}
				if duplicateFound {
					break
				}
			}
		}
	}

	if duplicateFound {
		// Update the existing row
		table.mutex.Lock()
		defer table.mutex.Unlock()

		oldRow := table.Rows[duplicateRowIndex]
		updatedRow := Row{Values: make([]interface{}, len(oldRow.Values))}
		copy(updatedRow.Values, oldRow.Values)

		// Apply ON DUPLICATE KEY UPDATE assignments
		for _, assignment := range onDuplicate {
			colName := assignment.Column.Name.String()
			colIndex := table.GetColumnIndex(colName)
			if colIndex == -1 {
				return fmt.Errorf("column %s does not exist", colName)
			}

			// Evaluate the assignment expression
			newValue, err := evaluateOnDuplicateExpression(assignment.Expr, table, updatedRow, newRow)
			if err != nil {
				return fmt.Errorf("error evaluating ON DUPLICATE KEY UPDATE expression for column %s: %v", colName, err)
			}

			updatedRow.Values[colIndex] = newValue
		}

		// Handle ON UPDATE CURRENT_TIMESTAMP
		for i, col := range table.Columns {
			if col.Type == TypeTimestamp && col.OnUpdate != nil {
				if fmt.Sprintf("%v", col.OnUpdate) == "CURRENT_TIMESTAMP" {
					updatedRow.Values[i] = time.Now().Format("2006-01-02 15:04:05")
				}
			}
		}

		// Validate foreign keys
		if err := db.ValidateForeignKeys(table, updatedRow.Values); err != nil {
			return fmt.Errorf("foreign key constraint violation in ON DUPLICATE KEY UPDATE: %v", err)
		}

		// Update the row
		table.Rows[duplicateRowIndex] = updatedRow

		// Update indexes
		db.IndexManager.UpdateIndexes(table.Name, duplicateRowIndex, &oldRow, &updatedRow, table)
	} else {
		// No duplicate found, insert normally
		err := table.AddRowWithIndexManager(newRow, db.IndexManager)
		if err != nil {
			return err
		}
	}

	return nil
}

// evaluateOnDuplicateExpression evaluates an expression in the context of ON DUPLICATE KEY UPDATE
func evaluateOnDuplicateExpression(expr ast.ExprNode, table *Table, currentRow Row, newRow []interface{}) (interface{}, error) {
	switch e := expr.(type) {
	case ast.ValueExpr:
		return e.GetValue(), nil
	case *ast.ColumnNameExpr:
		colName := e.Name.Name.String()
		colIndex := table.GetColumnIndex(colName)
		if colIndex == -1 {
			return nil, fmt.Errorf("column %s does not exist", colName)
		}
		return currentRow.Values[colIndex], nil
	case *ast.ValuesExpr:
		// Handle VALUES(column) function which refers to the new row values
		if e.Column != nil {
			colName := e.Column.Name.Name.String()
			colIndex := table.GetColumnIndex(colName)
			if colIndex == -1 {
				return nil, fmt.Errorf("column %s does not exist in VALUES()", colName)
			}
			return newRow[colIndex], nil
		}
		return nil, fmt.Errorf("unsupported VALUES() expression format")
	case *ast.FuncCallExpr:
		// Handle VALUES() function which refers to the new row values
		if strings.ToUpper(e.FnName.L) == "VALUES" && len(e.Args) == 1 {
			if colExpr, ok := e.Args[0].(*ast.ColumnNameExpr); ok {
				colName := colExpr.Name.Name.String()
				colIndex := table.GetColumnIndex(colName)
				if colIndex == -1 {
					return nil, fmt.Errorf("column %s does not exist in VALUES()", colName)
				}
				return newRow[colIndex], nil
			}
		}
		return nil, fmt.Errorf("unsupported function %s in ON DUPLICATE KEY UPDATE", e.FnName.L)
	case *ast.BinaryOperationExpr:
		// Handle expressions like quantity + VALUES(quantity)
		left, err := evaluateOnDuplicateExpression(e.L, table, currentRow, newRow)
		if err != nil {
			return nil, err
		}
		right, err := evaluateOnDuplicateExpression(e.R, table, currentRow, newRow)
		if err != nil {
			return nil, err
		}
		
		// Perform the operation
		switch e.Op {
		case opcode.Plus:
			if leftNum, ok := left.(int64); ok {
				if rightNum, ok := right.(int64); ok {
					return leftNum + rightNum, nil
				}
			}
			if leftNum, ok := left.(float64); ok {
				if rightNum, ok := right.(float64); ok {
					return leftNum + rightNum, nil
				}
			}
			return nil, fmt.Errorf("cannot add %T and %T", left, right)
		case opcode.Minus:
			if leftNum, ok := left.(int64); ok {
				if rightNum, ok := right.(int64); ok {
					return leftNum - rightNum, nil
				}
			}
			if leftNum, ok := left.(float64); ok {
				if rightNum, ok := right.(float64); ok {
					return leftNum - rightNum, nil
				}
			}
			return nil, fmt.Errorf("cannot subtract %T and %T", left, right)
		default:
			return nil, fmt.Errorf("unsupported binary operation %v in ON DUPLICATE KEY UPDATE", e.Op)
		}
	default:
		return nil, fmt.Errorf("unsupported expression type %T in ON DUPLICATE KEY UPDATE", expr)
	}
}
