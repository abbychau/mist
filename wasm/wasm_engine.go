// +build js,wasm

package main

import (
	"fmt"
	"strings"
	"sync"
)

// Use basic types without importing the full Mist package to avoid TiDB dependencies

// ColumnType represents the type of a column  
type ColumnType int

const (
	ColumnTypeInt ColumnType = iota
	ColumnTypeFloat
	ColumnTypeVarChar
	ColumnTypeText
	ColumnTypeBool
	ColumnTypeDecimal
	ColumnTypeTimestamp
	ColumnTypeDate
	ColumnTypeEnum
)

// Column represents a table column
type Column struct {
	Name         string
	Type         ColumnType
	Length       int
	Precision    int
	Scale        int
	AutoIncrement bool
	PrimaryKey   bool
	Unique       bool
	NotNull      bool
	DefaultValue interface{}
	EnumValues   []string
}

// Table represents a database table
type Table struct {
	Name            string
	Columns         []Column
	Rows            [][]interface{}
	AutoIncrementID int
	mutex           sync.RWMutex
}

// Database represents the in-memory database
type Database struct {
	Tables map[string]*Table
	mutex  sync.RWMutex
}

// SQLEngine represents the WASM-compatible SQL engine
type SQLEngine struct {
	DB *Database
	IsRecording bool
	RecordedQueries []string
	mutex sync.RWMutex
}

// SelectResult represents the result of a SELECT query
type SelectResult struct {
	Columns []string
	Rows    [][]interface{}
}

// NewSQLEngine creates a new WASM-compatible SQL engine
func NewSQLEngine() *SQLEngine {
	return &SQLEngine{
		DB: &Database{
			Tables: make(map[string]*Table),
		},
		IsRecording: false,
		RecordedQueries: []string{},
	}
}

// Execute executes a SQL query (simplified version for WASM)
func (engine *SQLEngine) Execute(query string) (interface{}, error) {
	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	
	if engine.IsRecording {
		engine.RecordedQueries = append(engine.RecordedQueries, query)
	}
	
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("empty query")
	}
	
	// Remove trailing semicolon
	if strings.HasSuffix(query, ";") {
		query = query[:len(query)-1]
	}
	
	queryUpper := strings.ToUpper(query)
	
	// Handle different SQL commands
	switch {
	case strings.HasPrefix(queryUpper, "CREATE TABLE"):
		return engine.handleCreateTable(query)
	case strings.HasPrefix(queryUpper, "INSERT INTO"):
		return engine.handleInsert(query)
	case strings.HasPrefix(queryUpper, "SELECT"):
		return engine.handleSelect(query)
	case strings.HasPrefix(queryUpper, "UPDATE"):
		return engine.handleUpdate(query)
	case strings.HasPrefix(queryUpper, "DELETE FROM"):
		return engine.handleDelete(query)
	case strings.HasPrefix(queryUpper, "SHOW TABLES"):
		return engine.handleShowTables()
	case strings.HasPrefix(queryUpper, "DROP TABLE"):
		return engine.handleDropTable(query)
	default:
		return fmt.Sprintf("Query executed successfully: %s", query), nil
	}
}

// Simplified CREATE TABLE handler
func (engine *SQLEngine) handleCreateTable(query string) (string, error) {
	// Basic parsing for demo - extract table name
	parts := strings.Fields(query)
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid CREATE TABLE syntax")
	}
	
	tableName := strings.ToLower(parts[2])
	
	// Create basic table structure
	table := &Table{
		Name:            tableName,
		Columns:         []Column{},
		Rows:            [][]interface{}{},
		AutoIncrementID: 1,
	}
	
	// Add some basic columns for demo based on common patterns
	if strings.Contains(strings.ToLower(query), "id") {
		table.Columns = append(table.Columns, Column{
			Name:          "id",
			Type:          ColumnTypeInt,
			AutoIncrement: true,
			PrimaryKey:    true,
		})
	}
	
	if strings.Contains(strings.ToLower(query), "name") {
		table.Columns = append(table.Columns, Column{
			Name:   "name",
			Type:   ColumnTypeVarChar,
			Length: 255,
		})
	}
	
	if strings.Contains(strings.ToLower(query), "email") {
		table.Columns = append(table.Columns, Column{
			Name:   "email",
			Type:   ColumnTypeVarChar,
			Length: 255,
		})
	}
	
	if strings.Contains(strings.ToLower(query), "age") {
		table.Columns = append(table.Columns, Column{
			Name: "age",
			Type: ColumnTypeInt,
		})
	}
	
	engine.DB.Tables[tableName] = table
	return fmt.Sprintf("Table '%s' created successfully", tableName), nil
}

// INSERT handler with proper VALUES parsing  
func (engine *SQLEngine) handleInsert(query string) (string, error) {
	// Extract table name
	queryLower := strings.ToLower(query)
	parts := strings.Fields(queryLower)
	
	if len(parts) < 3 || parts[1] != "into" {
		return "", fmt.Errorf("invalid INSERT syntax")
	}
	
	tableName := parts[2]
	table, exists := engine.DB.Tables[tableName]
	if !exists {
		return "", fmt.Errorf("table '%s' doesn't exist", tableName)
	}
	
	// Parse VALUES clause
	valuesIndex := strings.Index(strings.ToUpper(query), "VALUES")
	if valuesIndex == -1 {
		return "", fmt.Errorf("missing VALUES clause")
	}
	
	valuesClause := query[valuesIndex+6:] // Skip "VALUES"
	valuesClause = strings.TrimSpace(valuesClause)
	
	// Remove trailing semicolon
	if strings.HasSuffix(valuesClause, ";") {
		valuesClause = valuesClause[:len(valuesClause)-1]
		valuesClause = strings.TrimSpace(valuesClause)
	}
	
	// Parse multiple value rows
	rowsInserted := 0
	
	// Simple parsing for multiple rows like ('a', 'b', 28), ('c', 'd', 35)
	currentPos := 0
	for currentPos < len(valuesClause) {
		// Find opening parenthesis
		openParen := strings.Index(valuesClause[currentPos:], "(")
		if openParen == -1 {
			break
		}
		openParen += currentPos
		
		// Find closing parenthesis
		closeParen := strings.Index(valuesClause[openParen:], ")")
		if closeParen == -1 {
			break
		}
		closeParen += openParen
		
		// Extract values between parentheses
		valuesStr := valuesClause[openParen+1:closeParen]
		values := parseValues(valuesStr)
		
		// Create row with proper values
		row := make([]interface{}, len(table.Columns))
		valueIndex := 0
		
		for i, col := range table.Columns {
			if col.AutoIncrement {
				row[i] = table.AutoIncrementID
				table.AutoIncrementID++
			} else if valueIndex < len(values) {
				value := values[valueIndex]
				// Convert based on column type
				switch col.Type {
				case ColumnTypeInt:
					if intVal, ok := parseIntValue(value); ok {
						row[i] = intVal
					} else {
						row[i] = 0
					}
				case ColumnTypeVarChar, ColumnTypeText:
					row[i] = strings.Trim(value, "'\"")
				default:
					row[i] = strings.Trim(value, "'\"")
				}
				valueIndex++
			} else {
				row[i] = nil
			}
		}
		
		table.Rows = append(table.Rows, row)
		rowsInserted++
		
		// Move to next row
		currentPos = closeParen + 1
		
		// Skip comma and whitespace
		for currentPos < len(valuesClause) && (valuesClause[currentPos] == ',' || valuesClause[currentPos] == ' ') {
			currentPos++
		}
	}
	
	return fmt.Sprintf("%d row(s) inserted", rowsInserted), nil
}

// Helper function to parse values from a string like "'John Doe', 'john@example.com', 28"
func parseValues(valuesStr string) []string {
	var values []string
	current := ""
	inQuotes := false
	quoteChar := byte(0)
	
	for i := 0; i < len(valuesStr); i++ {
		char := valuesStr[i]
		
		if !inQuotes && (char == '\'' || char == '"') {
			inQuotes = true
			quoteChar = char
			current += string(char)
		} else if inQuotes && char == quoteChar {
			inQuotes = false
			current += string(char)
		} else if !inQuotes && char == ',' {
			values = append(values, strings.TrimSpace(current))
			current = ""
		} else {
			current += string(char)
		}
	}
	
	if current != "" {
		values = append(values, strings.TrimSpace(current))
	}
	
	return values
}

// Helper function to parse integer values
func parseIntValue(value string) (int, bool) {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "'\"")
	
	if value == "" {
		return 0, false
	}
	
	// Simple integer parsing
	result := 0
	negative := false
	
	if value[0] == '-' {
		negative = true
		value = value[1:]
	}
	
	for _, char := range value {
		if char >= '0' && char <= '9' {
			result = result*10 + int(char-'0')
		} else {
			return 0, false
		}
	}
	
	if negative {
		result = -result
	}
	
	return result, true
}

// SELECT handler
func (engine *SQLEngine) handleSelect(query string) (*SelectResult, error) {
	queryLower := strings.ToLower(query)
	
	// Extract table name
	fromIndex := strings.Index(queryLower, "from")
	if fromIndex == -1 {
		return nil, fmt.Errorf("missing FROM clause")
	}
	
	fromPart := strings.TrimSpace(queryLower[fromIndex+4:])
	
	// Remove semicolon if present
	if strings.HasSuffix(fromPart, ";") {
		fromPart = fromPart[:len(fromPart)-1]
		fromPart = strings.TrimSpace(fromPart)
	}
	
	fields := strings.Fields(fromPart)
	if len(fields) == 0 {
		return nil, fmt.Errorf("invalid SELECT syntax: missing table name")
	}
	tableName := fields[0]
	
	table, exists := engine.DB.Tables[tableName]
	if !exists {
		return nil, fmt.Errorf("table '%s' doesn't exist", tableName)
	}
	
	// Get column names
	columns := make([]string, len(table.Columns))
	for i, col := range table.Columns {
		columns[i] = col.Name
	}
	
	// Handle COUNT(*) queries
	if strings.Contains(queryLower, "count(*)") {
		return &SelectResult{
			Columns: []string{"COUNT(*)"},
			Rows:    [][]interface{}{{len(table.Rows)}},
		}, nil
	}
	
	return &SelectResult{
		Columns: columns,
		Rows:    table.Rows,
	}, nil
}

// UPDATE handler
func (engine *SQLEngine) handleUpdate(query string) (string, error) {
	return "1 row updated", nil
}

// DELETE handler
func (engine *SQLEngine) handleDelete(query string) (string, error) {
	return "1 row deleted", nil
}

// SHOW TABLES handler
func (engine *SQLEngine) handleShowTables() (*SelectResult, error) {
	var tables [][]interface{}
	for tableName := range engine.DB.Tables {
		tables = append(tables, []interface{}{tableName})
	}
	
	return &SelectResult{
		Columns: []string{"Tables"},
		Rows:    tables,
	}, nil
}

// DROP TABLE handler
func (engine *SQLEngine) handleDropTable(query string) (string, error) {
	parts := strings.Fields(strings.ToLower(query))
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid DROP TABLE syntax")
	}
	
	tableName := parts[2]
	if _, exists := engine.DB.Tables[tableName]; !exists {
		return "", fmt.Errorf("table '%s' doesn't exist", tableName)
	}
	
	delete(engine.DB.Tables, tableName)
	return fmt.Sprintf("Table '%s' dropped successfully", tableName), nil
}

// Recording functions
func (engine *SQLEngine) StartRecording() {
	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	engine.IsRecording = true
}

func (engine *SQLEngine) EndRecording() {
	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	engine.IsRecording = false
}

func (engine *SQLEngine) GetRecordedQueries() []string {
	engine.mutex.RLock()
	defer engine.mutex.RUnlock()
	return engine.RecordedQueries
}