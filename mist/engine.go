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

// TransactionData holds the state of an active transaction
type TransactionData struct {
	// Snapshot of the database state when transaction started
	originalTables map[string]*Table
	// Changes made during the transaction (for rollback)
	changes []TransactionChange
	// Nested transaction support
	level      int                   // Transaction nesting level (0 = outermost)
	parent     *TransactionData      // Parent transaction (nil for outermost)
	savepoints map[string]*Savepoint // Named savepoints within this transaction
}

// Savepoint represents a savepoint within a transaction
type Savepoint struct {
	name           string
	snapshotTables map[string]*Table
	level          int // Transaction level when savepoint was created
}

// TransactionChange represents a change made during a transaction
type TransactionChange struct {
	Type      string // "INSERT", "UPDATE", "DELETE", "CREATE_TABLE", "ALTER_TABLE"
	TableName string
	// For rollback purposes
	OldRow   *Row // for UPDATE and DELETE
	NewRow   *Row // for INSERT and UPDATE
	RowIndex int  // for UPDATE and DELETE
}

// SQLEngine represents the main SQL execution engine
type SQLEngine struct {
	database        *Database
	recording       bool
	recordedQueries []string
	recordingMutex  sync.RWMutex
	// Transaction support
	inTransaction    bool
	transactionData  *TransactionData
	transactionLevel int // Current nesting level (0 = no transaction)
	transactionMutex sync.RWMutex
}

// NewSQLEngine creates a new SQL engine with an empty database
func NewSQLEngine() *SQLEngine {
	return &SQLEngine{
		database:         NewDatabase(),
		recording:        false,
		recordedQueries:  make([]string, 0),
		inTransaction:    false,
		transactionData:  nil,
		transactionLevel: 0,
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

	case *ast.BeginStmt:
		return engine.executeBegin()

	case *ast.CommitStmt:
		return engine.executeCommit()

	case *ast.RollbackStmt:
		return engine.executeRollback(stmt)

	case *ast.SavepointStmt:
		return engine.executeSavepoint(stmt)

	case *ast.ReleaseSavepointStmt:
		return engine.executeReleaseSavepoint(stmt)

	case *ast.DropTableStmt:
		err := ExecuteDropTable(engine.database, stmt)
		if err != nil {
			return nil, err
		}
		return "Table dropped successfully", nil

	case *ast.TruncateTableStmt:
		err := ExecuteTruncateTable(engine.database, stmt)
		if err != nil {
			return nil, err
		}
		return "Table truncated successfully", nil

	case *ast.SetOprStmt:
		// UNION operations are not fully supported due to parser limitations
		return nil, fmt.Errorf("UNION operations require more complex parser integration - not yet supported")

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

// executeBegin starts a new transaction (supports nesting)
func (engine *SQLEngine) executeBegin() (interface{}, error) {
	engine.transactionMutex.Lock()
	defer engine.transactionMutex.Unlock()

	// Increment transaction level
	engine.transactionLevel++

	if engine.transactionLevel == 1 {
		// First level transaction - create initial snapshot
		originalTables := make(map[string]*Table)
		engine.database.mutex.RLock()
		for name, table := range engine.database.Tables {
			// Create a deep copy of the table
			originalTables[name] = engine.copyTable(table)
		}
		engine.database.mutex.RUnlock()

		engine.transactionData = &TransactionData{
			originalTables: originalTables,
			changes:        make([]TransactionChange, 0),
			level:          0,
			parent:         nil,
			savepoints:     make(map[string]*Savepoint),
		}
		engine.inTransaction = true
		return "Transaction started", nil
	} else {
		// Nested transaction - create a savepoint-like behavior
		currentTables := make(map[string]*Table)
		engine.database.mutex.RLock()
		for name, table := range engine.database.Tables {
			currentTables[name] = engine.copyTable(table)
		}
		engine.database.mutex.RUnlock()

		// Create nested transaction data
		nestedTransaction := &TransactionData{
			originalTables: currentTables,
			changes:        make([]TransactionChange, 0),
			level:          engine.transactionLevel - 1,
			parent:         engine.transactionData,
			savepoints:     make(map[string]*Savepoint),
		}

		// Link to parent
		engine.transactionData = nestedTransaction
		return fmt.Sprintf("Nested transaction started (level %d)", engine.transactionLevel), nil
	}
}

// executeCommit commits the current transaction (supports nesting)
func (engine *SQLEngine) executeCommit() (interface{}, error) {
	engine.transactionMutex.Lock()
	defer engine.transactionMutex.Unlock()

	if !engine.inTransaction || engine.transactionLevel == 0 {
		return nil, fmt.Errorf("no transaction in progress")
	}

	if engine.transactionLevel == 1 {
		// Outermost transaction - commit all changes
		engine.inTransaction = false
		engine.transactionData = nil
		engine.transactionLevel = 0
		return "Transaction committed", nil
	} else {
		// Nested transaction - merge changes to parent and pop level
		if engine.transactionData.parent != nil {
			// Move to parent transaction
			engine.transactionData = engine.transactionData.parent
		}
		engine.transactionLevel--
		return fmt.Sprintf("Nested transaction committed (level %d)", engine.transactionLevel), nil
	}
}

// executeRollback rolls back the current transaction (supports nesting and savepoints)
func (engine *SQLEngine) executeRollback(stmt *ast.RollbackStmt) (interface{}, error) {
	engine.transactionMutex.Lock()
	defer engine.transactionMutex.Unlock()

	if !engine.inTransaction || engine.transactionLevel == 0 {
		return nil, fmt.Errorf("no transaction in progress")
	}

	// Check if this is a rollback to savepoint
	if stmt.SavepointName != "" {
		return engine.rollbackToSavepoint(stmt.SavepointName)
	}

	if engine.transactionLevel == 1 {
		// Outermost transaction - rollback to original state
		engine.database.mutex.Lock()
		engine.database.Tables = engine.transactionData.originalTables
		engine.database.mutex.Unlock()

		// Clear transaction state
		engine.inTransaction = false
		engine.transactionData = nil
		engine.transactionLevel = 0
		return "Transaction rolled back", nil
	} else {
		// Nested transaction - rollback to the state when this nested transaction started
		engine.database.mutex.Lock()
		engine.database.Tables = engine.transactionData.originalTables
		engine.database.mutex.Unlock()

		// Move to parent transaction
		if engine.transactionData.parent != nil {
			engine.transactionData = engine.transactionData.parent
		}
		engine.transactionLevel--
		return fmt.Sprintf("Nested transaction rolled back (level %d)", engine.transactionLevel), nil
	}
}

// copyTable creates a deep copy of a table for transaction snapshots
func (engine *SQLEngine) copyTable(original *Table) *Table {
	original.mutex.RLock()
	defer original.mutex.RUnlock()

	// Copy columns
	columns := make([]Column, len(original.Columns))
	copy(columns, original.Columns)

	// Copy rows
	rows := make([]Row, len(original.Rows))
	for i, row := range original.Rows {
		values := make([]interface{}, len(row.Values))
		copy(values, row.Values)
		rows[i] = Row{Values: values}
	}

	// Copy unique indexes
	uniqueIndexes := make(map[string]map[interface{}]bool)
	for colName, index := range original.UniqueIndexes {
		uniqueIndexes[colName] = make(map[interface{}]bool)
		for value, exists := range index {
			uniqueIndexes[colName][value] = exists
		}
	}

	// Copy foreign keys
	foreignKeys := make([]ForeignKey, len(original.ForeignKeys))
	copy(foreignKeys, original.ForeignKeys)

	return &Table{
		Name:            original.Name,
		Columns:         columns,
		Rows:            rows,
		AutoIncrCounter: original.AutoIncrCounter,
		UniqueIndexes:   uniqueIndexes,
		ForeignKeys:     foreignKeys,
	}
}

// executeSavepoint creates a savepoint within the current transaction
func (engine *SQLEngine) executeSavepoint(stmt *ast.SavepointStmt) (interface{}, error) {
	engine.transactionMutex.Lock()
	defer engine.transactionMutex.Unlock()

	if !engine.inTransaction || engine.transactionLevel == 0 {
		return nil, fmt.Errorf("no transaction in progress")
	}

	savepointName := stmt.Name
	if savepointName == "" {
		return nil, fmt.Errorf("savepoint name cannot be empty")
	}

	// Create a snapshot of the current database state
	currentTables := make(map[string]*Table)
	engine.database.mutex.RLock()
	for name, table := range engine.database.Tables {
		currentTables[name] = engine.copyTable(table)
	}
	engine.database.mutex.RUnlock()

	// Create savepoint
	savepoint := &Savepoint{
		name:           savepointName,
		snapshotTables: currentTables,
		level:          engine.transactionLevel,
	}

	// Add to current transaction's savepoints
	engine.transactionData.savepoints[savepointName] = savepoint

	return fmt.Sprintf("Savepoint %s created", savepointName), nil
}

// executeReleaseSavepoint releases a savepoint
func (engine *SQLEngine) executeReleaseSavepoint(stmt *ast.ReleaseSavepointStmt) (interface{}, error) {
	engine.transactionMutex.Lock()
	defer engine.transactionMutex.Unlock()

	if !engine.inTransaction || engine.transactionLevel == 0 {
		return nil, fmt.Errorf("no transaction in progress")
	}

	savepointName := stmt.Name
	if savepointName == "" {
		return nil, fmt.Errorf("savepoint name cannot be empty")
	}

	// Find savepoint in current transaction or parent transactions
	currentTxn := engine.transactionData
	for currentTxn != nil {
		if _, exists := currentTxn.savepoints[savepointName]; exists {
			delete(currentTxn.savepoints, savepointName)
			return fmt.Sprintf("Savepoint %s released", savepointName), nil
		}
		currentTxn = currentTxn.parent
	}

	return nil, fmt.Errorf("savepoint %s does not exist", savepointName)
}

// rollbackToSavepoint rolls back to a specific savepoint
func (engine *SQLEngine) rollbackToSavepoint(savepointName string) (interface{}, error) {
	if savepointName == "" {
		return nil, fmt.Errorf("savepoint name cannot be empty")
	}

	// Find savepoint in current transaction or parent transactions
	currentTxn := engine.transactionData
	for currentTxn != nil {
		if savepoint, exists := currentTxn.savepoints[savepointName]; exists {
			// Restore database to savepoint state
			engine.database.mutex.Lock()
			engine.database.Tables = savepoint.snapshotTables
			engine.database.mutex.Unlock()

			return fmt.Sprintf("Rolled back to savepoint %s", savepointName), nil
		}
		currentTxn = currentTxn.parent
	}

	return nil, fmt.Errorf("savepoint %s does not exist", savepointName)
}
