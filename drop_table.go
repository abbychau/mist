package mist

import (
	"fmt"
	"strings"

	"github.com/abbychau/mysql-parser/ast"
)

// ExecuteDropTable handles DROP TABLE statements
func ExecuteDropTable(db *Database, stmt *ast.DropTableStmt) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	for _, table := range stmt.Tables {
		tableName := strings.ToLower(table.Name.String())

		// Check if table exists
		if _, exists := db.Tables[tableName]; !exists {
			if stmt.IfExists {
				// IF EXISTS specified, don't error if table doesn't exist
				continue
			}
			return fmt.Errorf("table %s does not exist", table.Name.String())
		}

		// Check for foreign key constraints that reference this table
		err := db.validateDropTable(tableName)
		if err != nil {
			return err
		}

		// Remove the table
		delete(db.Tables, tableName)

		// Remove any indexes that reference this table
		db.IndexManager.DropTableIndexes(tableName)
	}

	return nil
}

// ExecuteTruncateTable handles TRUNCATE TABLE statements
func ExecuteTruncateTable(db *Database, stmt *ast.TruncateTableStmt) error {
	tableName := strings.ToLower(stmt.Table.Name.String())

	// Get the table
	table, err := db.GetTable(tableName)
	if err != nil {
		return err
	}

	// Check for foreign key constraints that reference this table
	err = db.validateTruncateTable(table)
	if err != nil {
		return err
	}

	// Clear all rows but keep table structure
	table.mutex.Lock()
	defer table.mutex.Unlock()

	// Reset rows
	table.Rows = make([]Row, 0)

	// Reset auto increment counter
	table.AutoIncrCounter = 0

	// Clear unique indexes but keep the structure
	for colName := range table.UniqueIndexes {
		table.UniqueIndexes[colName] = make(map[interface{}]bool)
	}

	// Clear table indexes in index manager
	db.IndexManager.ClearTableIndexes(tableName)

	return nil
}

// validateDropTable checks if a table can be safely dropped
func (db *Database) validateDropTable(tableName string) error {
	// Check all tables for foreign keys that reference this table
	for _, table := range db.Tables {
		for _, fk := range table.ForeignKeys {
			if strings.EqualFold(fk.RefTable, tableName) {
				return fmt.Errorf("cannot drop table %s: foreign key constraint exists in table %s", tableName, table.Name)
			}
		}
	}
	return nil
}

// validateTruncateTable checks if a table can be safely truncated
func (db *Database) validateTruncateTable(table *Table) error {
	// Check all tables for foreign keys that reference this table
	for _, otherTable := range db.Tables {
		if otherTable == table {
			continue // skip the same table
		}

		for _, fk := range otherTable.ForeignKeys {
			if strings.EqualFold(fk.RefTable, table.Name) {
				// Check if there are any rows in the referencing table
				otherTable.mutex.RLock()
				hasReferencingRows := len(otherTable.Rows) > 0
				otherTable.mutex.RUnlock()

				if hasReferencingRows {
					return fmt.Errorf("cannot truncate table %s: foreign key constraint exists and referencing table %s has data", table.Name, otherTable.Name)
				}
			}
		}
	}
	return nil
}