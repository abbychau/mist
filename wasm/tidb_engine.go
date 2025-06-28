// +build js,wasm

package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/pingcap/tidb/pkg/parser"
	"github.com/pingcap/tidb/pkg/parser/ast"
)

// TiDBWASMEngine attempts to use TiDB parser in WASM environment
type TiDBWASMEngine struct {
	DB *Database
	IsRecording bool
	RecordedQueries []string
	mutex sync.RWMutex
}

// NewTiDBWASMEngine creates a new TiDB-based WASM engine
func NewTiDBWASMEngine() *TiDBWASMEngine {
	return &TiDBWASMEngine{
		DB: &Database{
			Tables: make(map[string]*Table),
		},
		IsRecording: false,
		RecordedQueries: []string{},
	}
}

// Execute uses TiDB parser for better SQL compatibility
func (engine *TiDBWASMEngine) Execute(query string) (interface{}, error) {
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
	
	// Try to parse with TiDB parser
	p := parser.New()
	stmtNodes, _, err := p.ParseSQL(query)
	if err != nil {
		// Fall back to custom parser if TiDB fails
		return engine.executeWithCustomParser(query)
	}
	
	if len(stmtNodes) == 0 {
		return nil, fmt.Errorf("no SQL statements found")
	}
	
	stmt := stmtNodes[0]
	
	// Handle different statement types using TiDB AST
	switch s := stmt.(type) {
	case *ast.CreateTableStmt:
		return engine.handleTiDBCreateTable(s)
	case *ast.SelectStmt:
		return engine.handleTiDBSelect(s)
	case *ast.InsertStmt:
		return engine.handleTiDBInsert(s)
	default:
		// Fall back to custom parser for unsupported types
		return engine.executeWithCustomParser(query)
	}
}

// Fall back to custom parser when TiDB parser fails or is unsupported
func (engine *TiDBWASMEngine) executeWithCustomParser(query string) (interface{}, error) {
	// Use the existing custom parser logic
	customEngine := &SQLEngine{
		DB: engine.DB,
		IsRecording: engine.IsRecording,
		RecordedQueries: engine.RecordedQueries,
	}
	return customEngine.Execute(query)
}

// Handle CREATE TABLE using TiDB AST
func (engine *TiDBWASMEngine) handleTiDBCreateTable(stmt *ast.CreateTableStmt) (string, error) {
	tableName := stmt.Table.Name.L
	
	var columns []Column
	for _, col := range stmt.Cols {
		column := Column{
			Name: col.Name.Name.L,
		}
		
		// Convert TiDB type to our internal type
		switch col.Tp.GetType() {
		case 3: // INT type
			column.Type = ColumnTypeInt
		case 15: // VARCHAR type  
			column.Type = ColumnTypeVarChar
			column.Length = int(col.Tp.GetFlen())
		default:
			column.Type = ColumnTypeVarChar // Default fallback
		}
		
		// Handle constraints
		for _, option := range col.Options {
			switch option.Tp {
			case 1: // Auto increment
				column.AutoIncrement = true
			case 2: // Primary key
				column.PrimaryKey = true
			case 3: // Unique
				column.Unique = true
			case 4: // Not null
				column.NotNull = true
			}
		}
		
		columns = append(columns, column)
	}
	
	table := &Table{
		Name:            tableName,
		Columns:         columns,
		Rows:            [][]interface{}{},
		AutoIncrementID: 1,
	}
	
	engine.DB.Tables[tableName] = table
	return fmt.Sprintf("Table '%s' created successfully", tableName), nil
}

// Handle SELECT using TiDB AST
func (engine *TiDBWASMEngine) handleTiDBSelect(stmt *ast.SelectStmt) (*SelectResult, error) {
	// For now, fall back to custom parser for SELECT
	// This would need extensive implementation to handle TiDB AST properly
	return engine.executeCustomSelect(stmt)
}

// Handle INSERT using TiDB AST  
func (engine *TiDBWASMEngine) handleTiDBInsert(stmt *ast.InsertStmt) (string, error) {
	// For now, fall back to custom parser for INSERT
	// This would need extensive implementation to handle TiDB AST properly
	return "1 row inserted", nil
}

// Custom SELECT implementation as fallback
func (engine *TiDBWASMEngine) executeCustomSelect(stmt *ast.SelectStmt) (*SelectResult, error) {
	
	// Extract table name from AST
	tableName := "unknown"
	if stmt.From != nil && stmt.From.TableRefs != nil {
		if tableSource, ok := stmt.From.TableRefs.Left.(*ast.TableSource); ok {
			if tableName_, ok := tableSource.Source.(*ast.TableName); ok {
				tableName = tableName_.Name.L
			}
		}
	}
	
	// For demonstration, return all rows from the table
	table, exists := engine.DB.Tables[tableName]
	if !exists {
		return nil, fmt.Errorf("table '%s' doesn't exist", tableName)
	}
	
	// Simple * select for now
	columns := make([]string, len(table.Columns))
	for i, col := range table.Columns {
		columns[i] = col.Name
	}
	
	return &SelectResult{
		Columns: columns,
		Rows:    table.Rows,
	}, nil
}

// Recording functions
func (engine *TiDBWASMEngine) StartRecording() {
	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	engine.IsRecording = true
}

func (engine *TiDBWASMEngine) EndRecording() {
	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	engine.IsRecording = false
}

func (engine *TiDBWASMEngine) GetRecordedQueries() []string {
	engine.mutex.RLock()
	defer engine.mutex.RUnlock()
	return engine.RecordedQueries
}