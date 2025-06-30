package mist

import (
	"fmt"
	"strings"

	"github.com/abbychau/mysql-parser/ast"
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

	// Extract column names
	var columnNames []string
	for _, spec := range stmt.IndexPartSpecifications {
		columnNames = append(columnNames, spec.Column.Name.String())
	}

	if len(columnNames) == 0 {
		return fmt.Errorf("index must specify at least one column")
	}

	// Determine index type based on statement properties
	var indexType IndexType
	
	// Use TiDB parser constants for proper detection
	// TODO: Check if IndexKeyTypeFullText exists in new parser
	// if stmt.KeyType == ast.IndexKeyTypeFullText { // FULLTEXT index
	//	indexType = FullTextIndex // Full-text parsed-only index
	// } else 
	if len(columnNames) == 1 {
		indexType = HashIndex // Single-column functional index
	} else {
		indexType = CompositeIndex // Multi-column parsed-only index
	}

	// Create the index
	return db.IndexManager.CreateCompositeIndex(indexName, tableName, columnNames, indexType, table)
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
		Columns: []string{"Index_Name", "Table", "Columns", "Type", "Functional"},
		Rows:    make([][]interface{}, len(indexes)),
	}

	for i, index := range indexes {
		columnsStr := strings.Join(index.ColumnNames, ", ")
		functional := "Yes"
		if index.IsParsedOnly {
			functional = "No (Parsed Only)"
		}
		
		result.Rows[i] = []interface{}{
			index.Name,
			index.TableName,
			columnsStr,
			index.Type.String(),
			functional,
		}
	}

	return result, nil
}

// parseCreateIndexSQL is a helper function to parse and execute CREATE INDEX
func parseCreateIndexSQL(db *Database, sql string) error {
	// Enhanced parsing for CREATE [FULLTEXT] INDEX index_name ON table_name (column1, column2, ...)
	sql = strings.TrimSuffix(strings.TrimSpace(sql), ";")
	upperSQL := strings.ToUpper(sql)
	
	// Check for FULLTEXT index
	isFullText := strings.Contains(upperSQL, "FULLTEXT")
	
	// Simple state machine for parsing
	var indexName, tableName string
	var columnNames []string
	
	// Split into tokens
	tokens := strings.Fields(sql)
	upperTokens := strings.Fields(upperSQL)
	
	// Find positions of key tokens
	createPos := -1
	indexPos := -1
	onPos := -1
	
	for i, token := range upperTokens {
		switch token {
		case "CREATE":
			createPos = i
		case "INDEX":
			indexPos = i
		case "ON":
			onPos = i
		}
	}
	
	if createPos == -1 || indexPos == -1 || onPos == -1 {
		return fmt.Errorf("invalid CREATE INDEX syntax")
	}
	
	// Extract index name (between INDEX and ON)
	if onPos <= indexPos+1 {
		return fmt.Errorf("missing index name")
	}
	indexName = tokens[indexPos+1]
	
	// Extract table name and column part (after ON)
	if len(tokens) <= onPos+1 {
		return fmt.Errorf("missing table name")
	}
	
	tableAndColumns := strings.Join(tokens[onPos+1:], " ")
	
	// Check if table name and columns are combined (e.g., "users(age)")
	parenPos := strings.Index(tableAndColumns, "(")
	if parenPos == -1 {
		return fmt.Errorf("invalid CREATE INDEX syntax: columns must be in parentheses")
	}
	
	tableName = tableAndColumns[:parenPos]
	columnPart := tableAndColumns[parenPos:]
	
	if !strings.HasPrefix(columnPart, "(") || !strings.HasSuffix(columnPart, ")") {
		return fmt.Errorf("invalid CREATE INDEX syntax: columns must be in parentheses")
	}
	
	// Parse column names (comma-separated inside parentheses)
	columnsStr := strings.Trim(columnPart, "()")
	for _, col := range strings.Split(columnsStr, ",") {
		columnName := strings.TrimSpace(col)
		if columnName != "" {
			columnNames = append(columnNames, columnName)
		}
	}
	
	if len(columnNames) == 0 {
		return fmt.Errorf("index must specify at least one column")
	}

	// Get the table
	table, err := db.GetTable(tableName)
	if err != nil {
		return err
	}

	// Determine index type
	var indexType IndexType
	if isFullText {
		indexType = FullTextIndex // Full-text parsed-only index
	} else if len(columnNames) == 1 {
		indexType = HashIndex // Single-column functional index
	} else {
		indexType = CompositeIndex // Multi-column parsed-only index
	}

	// Create the index
	return db.IndexManager.CreateCompositeIndex(indexName, tableName, columnNames, indexType, table)
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
	return strings.HasPrefix(trimmed, "CREATE INDEX") || 
		   strings.HasPrefix(trimmed, "CREATE UNIQUE INDEX") ||
		   strings.HasPrefix(trimmed, "CREATE FULLTEXT INDEX")
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
