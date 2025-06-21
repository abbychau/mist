package mist

import (
	"fmt"

	"github.com/pingcap/tidb/pkg/parser/ast"
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
	deletedCount := 0

	// Process each row
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
			deletedCount++
		} else {
			remainingRows = append(remainingRows, row)
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

// parseDeleteSQL is a helper function to parse and execute DELETE
func parseDeleteSQL(db *Database, sql string) (int, error) {
	astNode, err := parse(sql)
	if err != nil {
		return 0, fmt.Errorf("parse error: %v", err)
	}

	stmt, ok := (*astNode).(*ast.DeleteStmt)
	if !ok {
		return 0, fmt.Errorf("not a DELETE statement")
	}

	return ExecuteDelete(db, stmt)
}
