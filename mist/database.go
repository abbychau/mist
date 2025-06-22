package mist

import (
	"fmt"
	"strings"
	"sync"
)

// ColumnType represents the data type of a column
type ColumnType int

const (
	TypeInt ColumnType = iota
	TypeVarchar
	TypeText
	TypeFloat
	TypeBool
	TypeDecimal
	TypeTimestamp
	TypeDate
)

func (ct ColumnType) String() string {
	switch ct {
	case TypeInt:
		return "INT"
	case TypeVarchar:
		return "VARCHAR"
	case TypeText:
		return "TEXT"
	case TypeFloat:
		return "FLOAT"
	case TypeBool:
		return "BOOL"
	case TypeDecimal:
		return "DECIMAL"
	case TypeTimestamp:
		return "TIMESTAMP"
	case TypeDate:
		return "DATE"
	default:
		return "UNKNOWN"
	}
}

// Column represents a table column definition
type Column struct {
	Name      string
	Type      ColumnType
	Length    int // for VARCHAR
	Precision int // for DECIMAL (total digits)
	Scale     int // for DECIMAL (digits after decimal point)
	NotNull   bool
	Primary   bool
	AutoIncr  bool
	Default   interface{} // default value for the column
}

// Row represents a single row of data
type Row struct {
	Values []interface{}
}

// Table represents a database table
type Table struct {
	Name            string
	Columns         []Column
	Rows            []Row
	AutoIncrCounter int64 // Counter for auto increment columns
	mutex           sync.RWMutex
}

// NewTable creates a new table with the given name and columns
func NewTable(name string, columns []Column) *Table {
	return &Table{
		Name:            name,
		Columns:         columns,
		Rows:            make([]Row, 0),
		AutoIncrCounter: 0, // Initialize auto increment counter
	}
}

// AddRow adds a new row to the table
func (t *Table) AddRow(values []interface{}) error {
	return t.AddRowWithIndexManager(values, nil)
}

// AddRowWithIndexManager adds a new row to the table and updates indexes
func (t *Table) AddRowWithIndexManager(values []interface{}, indexManager *IndexManager) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if len(values) != len(t.Columns) {
		return fmt.Errorf("column count mismatch: expected %d, got %d", len(t.Columns), len(values))
	}

	// Basic type validation
	for i, value := range values {
		if err := t.validateValue(i, value); err != nil {
			return err
		}
	}

	newRow := Row{Values: values}
	rowIndex := len(t.Rows)
	t.Rows = append(t.Rows, newRow)

	// Update indexes if index manager is provided
	if indexManager != nil {
		indexManager.AddRowToIndexes(t.Name, rowIndex, newRow, t)
	}

	return nil
}

// validateValue validates a value against the column type
func (t *Table) validateValue(colIndex int, value interface{}) error {
	col := t.Columns[colIndex]

	// Auto increment columns can be NULL during insert (they'll be auto-generated)
	if value == nil && col.NotNull && !col.AutoIncr {
		return fmt.Errorf("column %s cannot be null", col.Name)
	}

	if value == nil {
		return nil
	}

	switch col.Type {
	case TypeInt:
		switch value.(type) {
		case int, int32, int64:
			return nil
		default:
			return fmt.Errorf("invalid type for column %s: expected int, got %T", col.Name, value)
		}
	case TypeVarchar, TypeText:
		if str, ok := value.(string); ok {
			if col.Type == TypeVarchar && col.Length > 0 && len(str) > col.Length {
				return fmt.Errorf("string too long for column %s: max %d, got %d", col.Name, col.Length, len(str))
			}
			return nil
		}
		return fmt.Errorf("invalid type for column %s: expected string, got %T", col.Name, value)
	case TypeFloat:
		switch value.(type) {
		case float32, float64:
			return nil
		default:
			return fmt.Errorf("invalid type for column %s: expected float, got %T", col.Name, value)
		}
	case TypeBool:
		if _, ok := value.(bool); ok {
			return nil
		}
		return fmt.Errorf("invalid type for column %s: expected bool, got %T", col.Name, value)
	case TypeDecimal:
		// Accept various numeric types for DECIMAL
		switch value.(type) {
		case float32, float64, int, int32, int64, string:
			return nil
		default:
			// Check if it's a MyDecimal type from TiDB
			if fmt.Sprintf("%T", value) == "*types.MyDecimal" {
				return nil
			}
			return fmt.Errorf("invalid type for column %s: expected numeric value, got %T", col.Name, value)
		}
	case TypeTimestamp:
		// Accept time.Time or string for TIMESTAMP
		switch value.(type) {
		case string:
			return nil // Will be parsed later
		default:
			return fmt.Errorf("invalid type for column %s: expected timestamp, got %T", col.Name, value)
		}
	case TypeDate:
		// Accept time.Time or string for DATE
		switch value.(type) {
		case string:
			return nil // Will be parsed later
		default:
			return fmt.Errorf("invalid type for column %s: expected date, got %T", col.Name, value)
		}
	}

	return nil
}

// GetRows returns all rows (thread-safe)
func (t *Table) GetRows() []Row {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	// Return a copy to avoid race conditions
	rows := make([]Row, len(t.Rows))
	copy(rows, t.Rows)
	return rows
}

// GetColumnIndex returns the index of a column by name
func (t *Table) GetColumnIndex(name string) int {
	for i, col := range t.Columns {
		if strings.EqualFold(col.Name, name) {
			return i
		}
	}
	return -1
}

// GetNextAutoIncrementValue returns and increments the auto increment counter
func (t *Table) GetNextAutoIncrementValue() int64 {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.AutoIncrCounter++
	return t.AutoIncrCounter
}

// GetAutoIncrementColumn returns the index of the auto increment column, or -1 if none exists
func (t *Table) GetAutoIncrementColumn() int {
	for i, col := range t.Columns {
		if col.AutoIncr {
			return i
		}
	}
	return -1
}

// Database represents the in-memory database
type Database struct {
	Tables       map[string]*Table
	IndexManager *IndexManager
	mutex        sync.RWMutex
}

// NewDatabase creates a new database instance
func NewDatabase() *Database {
	return &Database{
		Tables:       make(map[string]*Table),
		IndexManager: NewIndexManager(),
	}
}

// CreateTable creates a new table in the database
func (db *Database) CreateTable(name string, columns []Column) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	// Check if table already exists
	if _, exists := db.Tables[strings.ToLower(name)]; exists {
		return fmt.Errorf("table %s already exists", name)
	}

	db.Tables[strings.ToLower(name)] = NewTable(name, columns)
	return nil
}

// GetTable retrieves a table by name
func (db *Database) GetTable(name string) (*Table, error) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	table, exists := db.Tables[strings.ToLower(name)]
	if !exists {
		return nil, fmt.Errorf("table %s does not exist", name)
	}
	return table, nil
}

// ListTables returns all table names
func (db *Database) ListTables() []string {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	tables := make([]string, 0, len(db.Tables))
	for name := range db.Tables {
		tables = append(tables, name)
	}
	return tables
}
