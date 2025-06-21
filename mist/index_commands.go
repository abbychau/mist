package mist

import (
	"fmt"
	"strings"

	"github.com/pingcap/tidb/pkg/parser/ast"
)

// ExecuteCreateIndex processes a CREATE INDEX statement
func ExecuteCreateIndex(db *Database, stmt *ast.CreateIndexStmt) error {
	indexName := stmt.IndexName
	tableName := stmt.Table.Name.String()

	// Get the table
	table, err := db.GetTable(tableName)
	if err != nil {
		return err
	}

	// For simplicity, we only support single-column indexes
	if len(stmt.IndexPartSpecifications) != 1 {
		return fmt.Errorf("only single-column indexes are supported")
	}

	columnName := stmt.IndexPartSpecifications[0].Column.Name.String()

	// Verify column exists
	if table.GetColumnIndex(columnName) == -1 {
		return fmt.Errorf("column %s does not exist in table %s", columnName, tableName)
	}

	// Determine index type (default to hash for simplicity)
	indexType := HashIndex

	// Create the index
	return db.IndexManager.CreateIndex(indexName, tableName, columnName, indexType, table)
}

// ExecuteDropIndex processes a DROP INDEX statement
func ExecuteDropIndex(db *Database, stmt *ast.DropIndexStmt) error {
	indexName := stmt.IndexName

	// Drop the index
	return db.IndexManager.DropIndex(indexName)
}

// ExecuteShowIndexes shows all indexes for a table
func ExecuteShowIndexes(db *Database, tableName string) (*SelectResult, error) {
	// Get all indexes for the table
	indexes := db.IndexManager.GetIndexesForTable(tableName, "")

	result := &SelectResult{
		Columns: []string{"Index_Name", "Table", "Column", "Type"},
		Rows:    make([][]interface{}, len(indexes)),
	}

	for i, index := range indexes {
		result.Rows[i] = []interface{}{
			index.Name,
			index.TableName,
			index.ColumnName,
			index.Type.String(),
		}
	}

	return result, nil
}

// parseCreateIndexSQL is a helper function to parse and execute CREATE INDEX
func parseCreateIndexSQL(db *Database, sql string) error {
	// Simple parsing for CREATE INDEX index_name ON table_name (column_name)
	// Remove semicolon and normalize
	sql = strings.TrimSuffix(strings.TrimSpace(sql), ";")
	originalParts := strings.Fields(sql)
	upperParts := strings.Fields(strings.ToUpper(sql))

	if len(upperParts) < 6 {
		return fmt.Errorf("invalid CREATE INDEX syntax")
	}

	if upperParts[0] != "CREATE" || upperParts[1] != "INDEX" || upperParts[3] != "ON" {
		return fmt.Errorf("invalid CREATE INDEX syntax")
	}

	indexName := originalParts[2] // Keep original case
	tableName := originalParts[4] // Keep original case

	// Extract column name from (column_name)
	columnPart := strings.Join(originalParts[5:], " ")
	if !strings.HasPrefix(columnPart, "(") || !strings.HasSuffix(columnPart, ")") {
		return fmt.Errorf("invalid CREATE INDEX syntax: column must be in parentheses")
	}

	columnName := strings.Trim(columnPart, "()")

	// Get the table
	table, err := db.GetTable(tableName)
	if err != nil {
		return err
	}

	// Verify column exists
	if table.GetColumnIndex(columnName) == -1 {
		return fmt.Errorf("column %s does not exist in table %s", columnName, tableName)
	}

	// Create the index
	return db.IndexManager.CreateIndex(indexName, tableName, columnName, HashIndex, table)
}

// parseDropIndexSQL is a helper function to parse and execute DROP INDEX
func parseDropIndexSQL(db *Database, sql string) error {
	// Simple parsing for DROP INDEX index_name
	originalParts := strings.Fields(sql)
	upperParts := strings.Fields(strings.ToUpper(sql))

	if len(upperParts) < 3 {
		return fmt.Errorf("invalid DROP INDEX syntax")
	}

	if upperParts[0] != "DROP" || upperParts[1] != "INDEX" {
		return fmt.Errorf("invalid DROP INDEX syntax")
	}

	indexName := strings.TrimSuffix(originalParts[2], ";")
	return db.IndexManager.DropIndex(indexName)
}

// isCreateIndexStatement checks if a SQL statement is CREATE INDEX
func isCreateIndexStatement(sql string) bool {
	trimmed := strings.TrimSpace(strings.ToUpper(sql))
	return strings.HasPrefix(trimmed, "CREATE INDEX") || strings.HasPrefix(trimmed, "CREATE UNIQUE INDEX")
}

// isDropIndexStatement checks if a SQL statement is DROP INDEX
func isDropIndexStatement(sql string) bool {
	trimmed := strings.TrimSpace(strings.ToUpper(sql))
	return strings.HasPrefix(trimmed, "DROP INDEX")
}

// isShowIndexStatement checks if a SQL statement is SHOW INDEX
func isShowIndexStatement(sql string) bool {
	trimmed := strings.TrimSpace(strings.ToUpper(sql))
	return strings.HasPrefix(trimmed, "SHOW INDEX") || strings.HasPrefix(trimmed, "SHOW INDEXES")
}

// parseShowIndexSQL parses and executes SHOW INDEX statement
func parseShowIndexSQL(db *Database, sql string) (*SelectResult, error) {
	// Simple parsing for SHOW INDEX FROM table_name
	parts := strings.Fields(strings.ToUpper(sql))
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid SHOW INDEX syntax")
	}

	if parts[0] != "SHOW" || (parts[1] != "INDEX" && parts[1] != "INDEXES") || parts[2] != "FROM" {
		return nil, fmt.Errorf("invalid SHOW INDEX syntax")
	}

	tableName := strings.TrimSuffix(parts[3], ";")
	return ExecuteShowIndexes(db, tableName)
}
