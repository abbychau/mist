package mist

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/pingcap/tidb/pkg/parser/ast"
)

// SQLEngine represents the main SQL execution engine
type SQLEngine struct {
	database        *Database
	recording       bool
	recordedQueries []string
	recordingMutex  sync.RWMutex
}

// NewSQLEngine creates a new SQL engine with an empty database
func NewSQLEngine() *SQLEngine {
	return &SQLEngine{
		database:        NewDatabase(),
		recording:       false,
		recordedQueries: make([]string, 0),
	}
}

// Execute executes a SQL statement and returns the result
func (engine *SQLEngine) Execute(sql string) (interface{}, error) {
	// Record query if recording is enabled
	engine.recordingMutex.RLock()
	if engine.recording {
		engine.recordingMutex.RUnlock()
		engine.recordingMutex.Lock()
		engine.recordedQueries = append(engine.recordedQueries, sql)
		engine.recordingMutex.Unlock()
	} else {
		engine.recordingMutex.RUnlock()
	}

	// Trim whitespace and ensure statement ends with semicolon for parsing
	sql = strings.TrimSpace(sql)
	if !strings.HasSuffix(sql, ";") {
		sql += ";"
	}

	// Handle special cases that might not parse well with TiDB parser
	if isCreateIndexStatement(sql) {
		err := parseCreateIndexSQL(engine.database, sql)
		if err != nil {
			return nil, err
		}
		return "Index created successfully", nil
	}

	if isDropIndexStatement(sql) {
		err := parseDropIndexSQL(engine.database, sql)
		if err != nil {
			return nil, err
		}
		return "Index dropped successfully", nil
	}

	if isShowIndexStatement(sql) {
		result, err := parseShowIndexSQL(engine.database, sql)
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	// Parse the SQL statement
	astNode, err := parse(sql)
	if err != nil {
		return nil, fmt.Errorf("parse error: %v", err)
	}

	// Route to appropriate handler based on statement type
	switch stmt := (*astNode).(type) {
	case *ast.CreateTableStmt:
		err := ExecuteCreateTable(engine.database, stmt)
		if err != nil {
			return nil, err
		}
		return fmt.Sprintf("Table %s created successfully", stmt.Table.Name.String()), nil

	case *ast.InsertStmt:
		err := ExecuteInsert(engine.database, stmt)
		if err != nil {
			return nil, err
		}
		return "Insert successful", nil

	case *ast.SelectStmt:
		// Check if this is a JOIN query
		if engine.isJoinQuery(stmt) {
			result, err := ExecuteSelectWithJoin(engine.database, stmt)
			if err != nil {
				return nil, err
			}
			return result, nil
		} else {
			result, err := ExecuteSelect(engine.database, stmt)
			if err != nil {
				return nil, err
			}
			return result, nil
		}

	case *ast.UpdateStmt:
		count, err := ExecuteUpdate(engine.database, stmt)
		if err != nil {
			return nil, err
		}
		return fmt.Sprintf("Updated %d row(s)", count), nil

	case *ast.DeleteStmt:
		count, err := ExecuteDelete(engine.database, stmt)
		if err != nil {
			return nil, err
		}
		return fmt.Sprintf("Deleted %d row(s)", count), nil

	case *ast.AlterTableStmt:
		err := ExecuteAlterTable(engine.database, stmt)
		if err != nil {
			return nil, err
		}
		return fmt.Sprintf("Table %s altered successfully", stmt.Table.Name.String()), nil

	case *ast.ShowStmt:
		return engine.executeShow(stmt)

	case *ast.CreateIndexStmt:
		err := ExecuteCreateIndex(engine.database, stmt)
		if err != nil {
			return nil, err
		}
		return "Index created successfully", nil

	case *ast.DropIndexStmt:
		err := ExecuteDropIndex(engine.database, stmt)
		if err != nil {
			return nil, err
		}
		return "Index dropped successfully", nil

	default:
		return nil, fmt.Errorf("unsupported statement type: %T", stmt)
	}
}

// isJoinQuery checks if a SELECT statement contains a JOIN
func (engine *SQLEngine) isJoinQuery(stmt *ast.SelectStmt) bool {
	if stmt.From == nil || stmt.From.TableRefs == nil {
		return false
	}

	// Check if TableRefs has a Right side (indicating a JOIN)
	if stmt.From.TableRefs.Right != nil {
		return true
	}

	// Check for comma-separated tables (cross join)
	// In this case, Left is a Join with Tp=0 and Right is nil
	if join, ok := stmt.From.TableRefs.Left.(*ast.Join); ok {
		if join.Tp == 0 && join.Right == nil {
			return true
		}
	}

	return false
}

// executeShow handles SHOW statements
func (engine *SQLEngine) executeShow(stmt *ast.ShowStmt) (interface{}, error) {
	switch stmt.Tp {
	case ast.ShowTables:
		tables := engine.database.ListTables()
		result := &SelectResult{
			Columns: []string{"Tables"},
			Rows:    make([][]interface{}, len(tables)),
		}
		for i, table := range tables {
			result.Rows[i] = []interface{}{table}
		}
		return result, nil

	default:
		return nil, fmt.Errorf("unsupported SHOW statement type: %v", stmt.Tp)
	}
}

// ExecuteMultiple executes multiple SQL statements separated by semicolons
func (engine *SQLEngine) ExecuteMultiple(sql string) ([]interface{}, error) {
	// Split by semicolon and execute each statement
	statements := strings.Split(sql, ";")
	results := make([]interface{}, 0)

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		result, err := engine.Execute(stmt)
		if err != nil {
			return results, err
		}
		results = append(results, result)
	}

	return results, nil
}

// GetDatabase returns the underlying database (for testing)
func (engine *SQLEngine) GetDatabase() *Database {
	return engine.database
}

// StartRecording starts recording all SQL queries executed by this engine
func (engine *SQLEngine) StartRecording() {
	engine.recordingMutex.Lock()
	defer engine.recordingMutex.Unlock()

	engine.recording = true
	engine.recordedQueries = make([]string, 0) // Clear any previous recordings
}

// EndRecording stops recording SQL queries
func (engine *SQLEngine) EndRecording() {
	engine.recordingMutex.Lock()
	defer engine.recordingMutex.Unlock()

	engine.recording = false
}

// GetRecordedQueries returns all queries that were recorded between StartRecording and EndRecording
// Returns a copy of the recorded queries to prevent external modification
func (engine *SQLEngine) GetRecordedQueries() []string {
	engine.recordingMutex.RLock()
	defer engine.recordingMutex.RUnlock()

	// Return a copy to prevent external modification
	queries := make([]string, len(engine.recordedQueries))
	copy(queries, engine.recordedQueries)
	return queries
}

// ImportSQLFile reads a .sql file and executes all SQL statements in it
// The file should contain SQL statements separated by semicolons
// Returns a slice of results for each executed statement and any error encountered
func (engine *SQLEngine) ImportSQLFile(filename string) ([]interface{}, error) {
	// Open the file
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQL file %s: %v", filename, err)
	}
	defer file.Close()

	// Read the entire file content
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read SQL file %s: %v", filename, err)
	}

	// Execute the SQL content using ExecuteMultiple
	return engine.ExecuteMultiple(string(content))
}

// ImportSQLFileFromReader reads SQL statements from an io.Reader and executes them
// This is useful for reading from strings, network connections, or other sources
// Returns a slice of results for each executed statement and any error encountered
func (engine *SQLEngine) ImportSQLFileFromReader(reader io.Reader) ([]interface{}, error) {
	// Read the entire content
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read SQL content: %v", err)
	}

	// Execute the SQL content using ExecuteMultiple
	return engine.ExecuteMultiple(string(content))
}

// ImportSQLFileWithProgress reads a .sql file and executes all SQL statements with progress reporting
// The progressCallback function is called after each statement with the statement number and total count
// Returns a slice of results for each executed statement and any error encountered
func (engine *SQLEngine) ImportSQLFileWithProgress(filename string, progressCallback func(current, total int, statement string)) ([]interface{}, error) {
	// Open the file
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQL file %s: %v", filename, err)
	}
	defer file.Close()

	// Read and parse statements line by line to provide better progress reporting
	var sqlContent strings.Builder
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "--") && !strings.HasPrefix(line, "#") {
			sqlContent.WriteString(line)
			sqlContent.WriteString("\n")
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read SQL file %s: %v", filename, err)
	}

	// Split into statements and execute with progress
	return engine.executeWithProgress(sqlContent.String(), progressCallback)
}

// executeWithProgress executes SQL statements with progress reporting
func (engine *SQLEngine) executeWithProgress(sql string, progressCallback func(current, total int, statement string)) ([]interface{}, error) {
	// Split by semicolon to get individual statements
	statements := strings.Split(sql, ";")
	results := make([]interface{}, 0)

	// Filter out empty statements
	var validStatements []string
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt != "" {
			validStatements = append(validStatements, stmt)
		}
	}

	total := len(validStatements)

	// Execute each statement with progress reporting
	for i, stmt := range validStatements {
		if progressCallback != nil {
			progressCallback(i+1, total, stmt)
		}

		result, err := engine.Execute(stmt)
		if err != nil {
			return results, fmt.Errorf("error executing statement %d (%s): %v", i+1, stmt, err)
		}
		results = append(results, result)
	}

	return results, nil
}
