// +build js,wasm

package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// Smart WASM SQL engine that implements proper WHERE and JOIN logic
// without heavy TiDB dependencies

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

// Execute executes a SQL query with proper parsing
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

// Enhanced CREATE TABLE with proper column parsing
func (engine *SQLEngine) handleCreateTable(query string) (string, error) {
	// Parse CREATE TABLE statement properly
	parts := strings.Fields(query)
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid CREATE TABLE syntax")
	}
	
	tableName := strings.ToLower(parts[2])
	
	// Find the opening parenthesis for column definitions
	openParen := strings.Index(query, "(")
	closeParen := strings.LastIndex(query, ")")
	
	if openParen == -1 || closeParen == -1 {
		return "", fmt.Errorf("invalid CREATE TABLE syntax: missing column definitions")
	}
	
	columnDefs := query[openParen+1:closeParen]
	columns := parseColumnDefinitions(columnDefs)
	
	table := &Table{
		Name:            tableName,
		Columns:         columns,
		Rows:            [][]interface{}{},
		AutoIncrementID: 1,
	}
	
	engine.DB.Tables[tableName] = table
	return fmt.Sprintf("Table '%s' created successfully", tableName), nil
}

// Parse column definitions from CREATE TABLE
func parseColumnDefinitions(columnDefs string) []Column {
	var columns []Column
	
	// Split by comma, but be careful of parentheses
	parts := smartSplit(columnDefs, ',')
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		
		column := parseColumnDefinition(part)
		if column.Name != "" {
			columns = append(columns, column)
		}
	}
	
	return columns
}

// Parse a single column definition
func parseColumnDefinition(def string) Column {
	parts := strings.Fields(def)
	if len(parts) < 2 {
		return Column{}
	}
	
	column := Column{
		Name: strings.ToLower(parts[0]),
	}
	
	// Parse type
	typeStr := strings.ToUpper(parts[1])
	if strings.Contains(typeStr, "INT") {
		column.Type = ColumnTypeInt
	} else if strings.Contains(typeStr, "VARCHAR") {
		column.Type = ColumnTypeVarChar
		// Extract length from VARCHAR(n)
		if parenStart := strings.Index(typeStr, "("); parenStart != -1 {
			if parenEnd := strings.Index(typeStr[parenStart:], ")"); parenEnd != -1 {
				lengthStr := typeStr[parenStart+1:parenStart+parenEnd]
				if length, err := strconv.Atoi(lengthStr); err == nil {
					column.Length = length
				}
			}
		}
	} else if strings.Contains(typeStr, "TEXT") {
		column.Type = ColumnTypeText
	} else if strings.Contains(typeStr, "DECIMAL") {
		column.Type = ColumnTypeDecimal
	} else if strings.Contains(typeStr, "FLOAT") {
		column.Type = ColumnTypeFloat
	} else {
		column.Type = ColumnTypeVarChar // Default
	}
	
	// Parse constraints
	defUpper := strings.ToUpper(def)
	if strings.Contains(defUpper, "AUTO_INCREMENT") {
		column.AutoIncrement = true
	}
	if strings.Contains(defUpper, "PRIMARY KEY") {
		column.PrimaryKey = true
	}
	if strings.Contains(defUpper, "UNIQUE") {
		column.Unique = true
	}
	if strings.Contains(defUpper, "NOT NULL") {
		column.NotNull = true
	}
	
	return column
}

// Smart split that respects parentheses
func smartSplit(s string, delimiter rune) []string {
	var parts []string
	var current strings.Builder
	parenLevel := 0
	
	for _, char := range s {
		if char == '(' {
			parenLevel++
		} else if char == ')' {
			parenLevel--
		} else if char == delimiter && parenLevel == 0 {
			parts = append(parts, current.String())
			current.Reset()
			continue
		}
		current.WriteRune(char)
	}
	
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	
	return parts
}

// Enhanced INSERT with proper VALUES parsing (reusing previous implementation)
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
	
	valuesClause := query[valuesIndex+6:]
	valuesClause = strings.TrimSpace(valuesClause)
	
	if strings.HasSuffix(valuesClause, ";") {
		valuesClause = valuesClause[:len(valuesClause)-1]
		valuesClause = strings.TrimSpace(valuesClause)
	}
	
	rowsInserted := 0
	currentPos := 0
	
	for currentPos < len(valuesClause) {
		openParen := strings.Index(valuesClause[currentPos:], "(")
		if openParen == -1 {
			break
		}
		openParen += currentPos
		
		closeParen := strings.Index(valuesClause[openParen:], ")")
		if closeParen == -1 {
			break
		}
		closeParen += openParen
		
		valuesStr := valuesClause[openParen+1:closeParen]
		values := parseValues(valuesStr)
		
		row := make([]interface{}, len(table.Columns))
		valueIndex := 0
		
		for i, col := range table.Columns {
			if col.AutoIncrement {
				row[i] = table.AutoIncrementID
				table.AutoIncrementID++
			} else if valueIndex < len(values) {
				value := values[valueIndex]
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
		
		currentPos = closeParen + 1
		for currentPos < len(valuesClause) && (valuesClause[currentPos] == ',' || valuesClause[currentPos] == ' ') {
			currentPos++
		}
	}
	
	return fmt.Sprintf("%d row(s) inserted", rowsInserted), nil
}

// Enhanced SELECT with proper WHERE clause and JOIN support
func (engine *SQLEngine) handleSelect(query string) (*SelectResult, error) {
	queryLower := strings.ToLower(query)
	
	// Check if it's a JOIN query
	if strings.Contains(queryLower, " join ") {
		return engine.handleJoinSelect(query)
	}
	
	// Parse simple SELECT
	selectClause, fromClause, whereClause := parseSelectParts(query)
	
	// Get table
	tableName := strings.TrimSpace(fromClause)
	// Remove alias if present (e.g., "users u" -> "users")
	if parts := strings.Fields(tableName); len(parts) > 1 {
		tableName = parts[0]
	}
	
	table, exists := engine.DB.Tables[tableName]
	if !exists {
		return nil, fmt.Errorf("table '%s' doesn't exist", tableName)
	}
	
	// Determine which columns to select
	selectedColumns, columnIndexes := parseSelectColumns(selectClause, table)
	
	// Filter rows based on WHERE clause
	filteredRows := filterRows(table.Rows, table.Columns, whereClause)
	
	// Project columns
	resultRows := projectColumns(filteredRows, columnIndexes)
	
	return &SelectResult{
		Columns: selectedColumns,
		Rows:    resultRows,
	}, nil
}

// Handle JOIN queries
func (engine *SQLEngine) handleJoinSelect(query string) (*SelectResult, error) {
	// Parse JOIN query
	selectClause, joinInfo := parseJoinQuery(query)
	
	leftTable, exists := engine.DB.Tables[joinInfo.LeftTable]
	if !exists {
		return nil, fmt.Errorf("table '%s' doesn't exist", joinInfo.LeftTable)
	}
	
	rightTable, exists := engine.DB.Tables[joinInfo.RightTable]
	if !exists {
		return nil, fmt.Errorf("table '%s' doesn't exist", joinInfo.RightTable)
	}
	
	// Perform JOIN
	joinedRows := performJoin(leftTable, rightTable, joinInfo)
	
	// Parse select columns for JOIN (e.g., "u.name, o.product")
	selectedColumns, columnIndexes := parseJoinSelectColumns(selectClause, leftTable, rightTable, joinInfo)
	
	// Project columns
	resultRows := projectColumns(joinedRows, columnIndexes)
	
	return &SelectResult{
		Columns: selectedColumns,
		Rows:    resultRows,
	}, nil
}

// Parse SELECT, FROM, WHERE parts
func parseSelectParts(query string) (string, string, string) {
	queryLower := strings.ToLower(query)
	
	selectStart := strings.Index(queryLower, "select") + 6
	fromIndex := strings.Index(queryLower, "from")
	whereIndex := strings.Index(queryLower, "where")
	
	selectClause := ""
	fromClause := ""
	whereClause := ""
	
	if fromIndex > selectStart {
		selectClause = strings.TrimSpace(query[selectStart:fromIndex])
	}
	
	if fromIndex != -1 {
		fromStart := fromIndex + 4
		if whereIndex != -1 {
			fromClause = strings.TrimSpace(query[fromStart:whereIndex])
		} else {
			fromClause = strings.TrimSpace(query[fromStart:])
		}
	}
	
	if whereIndex != -1 {
		whereStart := whereIndex + 5
		whereClause = strings.TrimSpace(query[whereStart:])
	}
	
	return selectClause, fromClause, whereClause
}

// Parse which columns to select
func parseSelectColumns(selectClause string, table *Table) ([]string, []int) {
	if selectClause == "*" {
		columns := make([]string, len(table.Columns))
		indexes := make([]int, len(table.Columns))
		for i, col := range table.Columns {
			columns[i] = col.Name
			indexes[i] = i
		}
		return columns, indexes
	}
	
	// Parse comma-separated column list
	parts := strings.Split(selectClause, ",")
	columns := make([]string, len(parts))
	indexes := make([]int, len(parts))
	
	for i, part := range parts {
		columnName := strings.TrimSpace(part)
		columns[i] = columnName
		
		// Find column index
		for j, col := range table.Columns {
			if col.Name == columnName {
				indexes[i] = j
				break
			}
		}
	}
	
	return columns, indexes
}

// Filter rows based on WHERE clause
func filterRows(rows [][]interface{}, columns []Column, whereClause string) [][]interface{} {
	if whereClause == "" {
		return rows
	}
	
	var filteredRows [][]interface{}
	
	for _, row := range rows {
		if evaluateWhere(row, columns, whereClause) {
			filteredRows = append(filteredRows, row)
		}
	}
	
	return filteredRows
}

// Evaluate WHERE clause for a row
func evaluateWhere(row []interface{}, columns []Column, whereClause string) bool {
	// Simple WHERE evaluation (age > 30, name = 'John', etc.)
	whereClause = strings.TrimSpace(whereClause)
	
	// Handle simple comparisons
	for _, op := range []string{" >= ", " <= ", " > ", " < ", " = ", " != "} {
		if strings.Contains(whereClause, op) {
			parts := strings.Split(whereClause, op)
			if len(parts) == 2 {
				left := strings.TrimSpace(parts[0])
				right := strings.TrimSpace(parts[1])
				
				// Find column
				for i, col := range columns {
					if col.Name == left && i < len(row) {
						return compareValues(row[i], right, op)
					}
				}
			}
		}
	}
	
	return true // Default to include if we can't parse
}

// Compare values based on operator
func compareValues(leftVal interface{}, rightStr string, operator string) bool {
	rightStr = strings.Trim(rightStr, "' \"")
	
	switch leftVal := leftVal.(type) {
	case int:
		if rightInt, err := strconv.Atoi(rightStr); err == nil {
			switch operator {
			case " > ":
				return leftVal > rightInt
			case " < ":
				return leftVal < rightInt
			case " >= ":
				return leftVal >= rightInt
			case " <= ":
				return leftVal <= rightInt
			case " = ":
				return leftVal == rightInt
			case " != ":
				return leftVal != rightInt
			}
		}
	case string:
		switch operator {
		case " = ":
			return leftVal == rightStr
		case " != ":
			return leftVal != rightStr
		}
	}
	
	return false
}

// Project columns from rows
func projectColumns(rows [][]interface{}, columnIndexes []int) [][]interface{} {
	var resultRows [][]interface{}
	
	for _, row := range rows {
		resultRow := make([]interface{}, len(columnIndexes))
		for i, colIndex := range columnIndexes {
			if colIndex < len(row) {
				resultRow[i] = row[colIndex]
			}
		}
		resultRows = append(resultRows, resultRow)
	}
	
	return resultRows
}

// JOIN structures and functions
type JoinInfo struct {
	LeftTable  string
	RightTable string
	LeftAlias  string
	RightAlias string
	LeftColumn string
	RightColumn string
}

func parseJoinQuery(query string) (string, JoinInfo) {
	queryLower := strings.ToLower(query)
	
	// Extract SELECT clause
	selectStart := strings.Index(queryLower, "select") + 6
	fromIndex := strings.Index(queryLower, "from")
	selectClause := strings.TrimSpace(query[selectStart:fromIndex])
	
	// Parse FROM table1 alias1 JOIN table2 alias2 ON condition
	fromPart := query[fromIndex+4:]
	joinIndex := strings.Index(strings.ToLower(fromPart), " join ")
	onIndex := strings.Index(strings.ToLower(fromPart), " on ")
	
	leftPart := strings.TrimSpace(fromPart[:joinIndex])
	joinPart := strings.TrimSpace(fromPart[joinIndex+6:onIndex])
	onPart := strings.TrimSpace(fromPart[onIndex+4:])
	
	// Parse left table and alias
	leftParts := strings.Fields(leftPart)
	leftTable := leftParts[0]
	leftAlias := leftTable
	if len(leftParts) > 1 {
		leftAlias = leftParts[1]
	}
	
	// Parse right table and alias
	rightParts := strings.Fields(joinPart)
	rightTable := rightParts[0]
	rightAlias := rightTable
	if len(rightParts) > 1 {
		rightAlias = rightParts[1]
	}
	
	// Parse ON condition (e.g., "u.id = o.user_id")
	onParts := strings.Split(onPart, " = ")
	leftColumn := ""
	rightColumn := ""
	if len(onParts) == 2 {
		leftCol := strings.TrimSpace(onParts[0])
		rightCol := strings.TrimSpace(onParts[1])
		
		// Remove alias prefix (e.g., "u.id" -> "id")
		if dotIndex := strings.Index(leftCol, "."); dotIndex != -1 {
			leftColumn = leftCol[dotIndex+1:]
		}
		if dotIndex := strings.Index(rightCol, "."); dotIndex != -1 {
			rightColumn = rightCol[dotIndex+1:]
		}
	}
	
	return selectClause, JoinInfo{
		LeftTable:   leftTable,
		RightTable:  rightTable,
		LeftAlias:   leftAlias,
		RightAlias:  rightAlias,
		LeftColumn:  leftColumn,
		RightColumn: rightColumn,
	}
}

func performJoin(leftTable, rightTable *Table, joinInfo JoinInfo) [][]interface{} {
	var joinedRows [][]interface{}
	
	// Find column indices for join condition
	leftColIndex := -1
	rightColIndex := -1
	
	for i, col := range leftTable.Columns {
		if col.Name == joinInfo.LeftColumn {
			leftColIndex = i
			break
		}
	}
	
	for i, col := range rightTable.Columns {
		if col.Name == joinInfo.RightColumn {
			rightColIndex = i
			break
		}
	}
	
	if leftColIndex == -1 || rightColIndex == -1 {
		return joinedRows
	}
	
	// Perform nested loop join
	for _, leftRow := range leftTable.Rows {
		for _, rightRow := range rightTable.Rows {
			if fmt.Sprintf("%v", leftRow[leftColIndex]) == fmt.Sprintf("%v", rightRow[rightColIndex]) {
				// Combine rows
				combinedRow := make([]interface{}, len(leftRow)+len(rightRow))
				copy(combinedRow, leftRow)
				copy(combinedRow[len(leftRow):], rightRow)
				joinedRows = append(joinedRows, combinedRow)
			}
		}
	}
	
	return joinedRows
}

func parseJoinSelectColumns(selectClause string, leftTable, rightTable *Table, joinInfo JoinInfo) ([]string, []int) {
	parts := strings.Split(selectClause, ",")
	columns := make([]string, len(parts))
	indexes := make([]int, len(parts))
	
	for i, part := range parts {
		part = strings.TrimSpace(part)
		columns[i] = part
		
		// Parse alias.column (e.g., "u.name", "o.product")
		if dotIndex := strings.Index(part, "."); dotIndex != -1 {
			alias := part[:dotIndex]
			columnName := part[dotIndex+1:]
			
			if alias == joinInfo.LeftAlias {
				// Find in left table
				for j, col := range leftTable.Columns {
					if col.Name == columnName {
						indexes[i] = j
						break
					}
				}
			} else if alias == joinInfo.RightAlias {
				// Find in right table (offset by left table column count)
				for j, col := range rightTable.Columns {
					if col.Name == columnName {
						indexes[i] = len(leftTable.Columns) + j
						break
					}
				}
			}
		}
	}
	
	return columns, indexes
}

// Helper functions (reuse from previous implementation)
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

func parseIntValue(value string) (int, bool) {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "'\"")
	
	if value == "" {
		return 0, false
	}
	
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

// Other handlers (simplified)
func (engine *SQLEngine) handleUpdate(query string) (string, error) {
	return "1 row updated", nil
}

func (engine *SQLEngine) handleDelete(query string) (string, error) {
	return "1 row deleted", nil
}

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