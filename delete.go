package mist

import (
	"fmt"

	"github.com/abbychau/mysql-parser/ast"
)

// ExecuteDelete processes a DELETE statement
func ExecuteDelete(db *Database, stmt *ast.DeleteStmt) (int, error) {
	// Get the table name from the table references
	if stmt.TableRefs == nil || stmt.TableRefs.TableRefs == nil {
		return 0, fmt.Errorf("no table specified in DELETE statement")
	}

	tableSource, ok := stmt.TableRefs.TableRefs.Left.(*ast.TableSource)
	if !ok {
		return 0, fmt.Errorf("complex table references not supported in DELETE")
	}

	tableName, ok := tableSource.Source.(*ast.TableName)
	if !ok {
		return 0, fmt.Errorf("subqueries not supported in DELETE")
	}

	table, err := db.GetTable(tableName.Name.String())
	if err != nil {
		return 0, err
	}

	// Get all rows from the table
	rows := table.GetRows()
	var remainingRows []Row
	var rowsToDelete []Row
	deletedCount := 0

	// First pass: identify rows to delete and validate foreign key constraints
	for _, row := range rows {
		// Check if row matches WHERE condition
		shouldDelete := true
		if stmt.Where != nil {
			match, err := evaluateWhereCondition(stmt.Where, table, row)
			if err != nil {
				return 0, fmt.Errorf("error evaluating WHERE clause: %v", err)
			}
			shouldDelete = match
		}

		if shouldDelete {
			// Validate foreign key constraints before deletion
			if err := db.ValidateForeignKeyDeletion(table, row); err != nil {
				return 0, fmt.Errorf("cannot delete row: %v", err)
			}
			rowsToDelete = append(rowsToDelete, row)
			deletedCount++
		} else {
			remainingRows = append(remainingRows, row)
		}
	}

	// Second pass: execute foreign key actions for rows that will be deleted
	for _, row := range rowsToDelete {
		// Execute foreign key actions (CASCADE, SET NULL, SET DEFAULT)
		if err := db.ExecuteForeignKeyDeletionActions(table, row); err != nil {
			return 0, fmt.Errorf("foreign key action failed: %v", err)
		}
	}

	// Update the table with remaining rows (thread-safe)
	table.mutex.Lock()
	table.Rows = remainingRows
	table.mutex.Unlock()

	return deletedCount, nil
}

// ExecuteDeleteAll deletes all rows from a table (DELETE FROM table without WHERE)
func ExecuteDeleteAll(table *Table) int {
	table.mutex.Lock()
	defer table.mutex.Unlock()

	rowCount := len(table.Rows)
	table.Rows = make([]Row, 0)
	return rowCount
}
