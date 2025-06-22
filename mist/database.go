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
	TypeEnum
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
	case TypeEnum:
		return "ENUM"
	default:
		return "UNKNOWN"
	}
}

// Column represents a table column definition
type Column struct {
	Name       string
	Type       ColumnType
	Length     int // for VARCHAR
	Precision  int // for DECIMAL (total digits)
	Scale      int // for DECIMAL (digits after decimal point)
	NotNull    bool
	Primary    bool
	Unique     bool // UNIQUE constraint
	AutoIncr   bool
	Default    interface{} // default value for the column
	OnUpdate   interface{} // ON UPDATE value (e.g., CURRENT_TIMESTAMP)
	EnumValues []string    // for ENUM type
	ForeignKey *ForeignKey // foreign key constraint, if any
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
	AutoIncrCounter int64                           // Counter for auto increment columns
	UniqueIndexes   map[string]map[interface{}]bool // column name -> value -> exists
	ForeignKeys     []ForeignKey                    // foreign key constraints
	mutex           sync.RWMutex
}

// NewTable creates a new table with the given name and columns
func NewTable(name string, columns []Column) *Table {
	table := &Table{
		Name:            name,
		Columns:         columns,
		Rows:            make([]Row, 0),
		AutoIncrCounter: 0, // Initialize auto increment counter
		UniqueIndexes:   make(map[string]map[interface{}]bool),
		ForeignKeys:     make([]ForeignKey, 0),
	}

	// Create unique indexes for columns with unique constraints
	for _, col := range columns {
		if col.Unique || col.Primary {
			table.UniqueIndexes[col.Name] = make(map[interface{}]bool)
		}
	}

	return table
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

	// Check unique constraints
	for i, value := range values {
		col := t.Columns[i]
		if (col.Unique || col.Primary) && value != nil {
			if uniqueIndex, exists := t.UniqueIndexes[col.Name]; exists {
				if _, duplicate := uniqueIndex[value]; duplicate {
					return fmt.Errorf("duplicate entry '%v' for unique column %s", value, col.Name)
				}
			}
		}
	}

	newRow := Row{Values: values}
	rowIndex := len(t.Rows)
	t.Rows = append(t.Rows, newRow)

	// Update unique indexes
	for i, value := range values {
		col := t.Columns[i]
		if (col.Unique || col.Primary) && value != nil {
			if uniqueIndex, exists := t.UniqueIndexes[col.Name]; exists {
				uniqueIndex[value] = true
			}
		}
	}

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
	case TypeEnum:
		if str, ok := value.(string); ok {
			// Check if the value is one of the allowed enum values
			for _, enumValue := range col.EnumValues {
				if str == enumValue {
					return nil
				}
			}
			return fmt.Errorf("invalid enum value for column %s: %s (allowed: %v)", col.Name, str, col.EnumValues)
		}
		return fmt.Errorf("invalid type for column %s: expected string for enum, got %T", col.Name, value)
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

// ForeignKeyAction represents the action to take on foreign key violations
type ForeignKeyAction int

const (
	FKActionNoAction ForeignKeyAction = iota
	FKActionRestrict
	FKActionCascade
	FKActionSetNull
	FKActionSetDefault
)

func (fka ForeignKeyAction) String() string {
	switch fka {
	case FKActionNoAction:
		return "NO ACTION"
	case FKActionRestrict:
		return "RESTRICT"
	case FKActionCascade:
		return "CASCADE"
	case FKActionSetNull:
		return "SET NULL"
	case FKActionSetDefault:
		return "SET DEFAULT"
	default:
		return "NO ACTION"
	}
}

// ForeignKey represents a foreign key constraint
type ForeignKey struct {
	Name         string           // constraint name
	LocalColumns []string         // local column names
	RefTable     string           // referenced table name
	RefColumns   []string         // referenced column names
	OnUpdate     ForeignKeyAction // action on update
	OnDelete     ForeignKeyAction // action on delete
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

// AddForeignKey adds a foreign key constraint to a table
func (t *Table) AddForeignKey(fk ForeignKey) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// Validate that local columns exist
	for _, colName := range fk.LocalColumns {
		found := false
		for _, col := range t.Columns {
			if strings.EqualFold(col.Name, colName) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("local column %s not found in table %s", colName, t.Name)
		}
	}

	t.ForeignKeys = append(t.ForeignKeys, fk)
	return nil
}

// ValidateForeignKeys validates foreign key constraints for a row
func (db *Database) ValidateForeignKeys(table *Table, values []interface{}) error {
	for _, fk := range table.ForeignKeys {
		if err := db.validateForeignKey(table, fk, values); err != nil {
			return err
		}
	}
	return nil
}

// validateForeignKey validates a single foreign key constraint
func (db *Database) validateForeignKey(table *Table, fk ForeignKey, values []interface{}) error {
	// Get referenced table
	refTable, err := db.GetTable(fk.RefTable)
	if err != nil {
		return fmt.Errorf("foreign key reference table %s not found", fk.RefTable)
	}

	// Build foreign key values from the current row
	fkValues := make([]interface{}, len(fk.LocalColumns))
	for i, colName := range fk.LocalColumns {
		colIndex := table.GetColumnIndex(colName)
		if colIndex == -1 {
			return fmt.Errorf("foreign key column %s not found", colName)
		}
		fkValues[i] = values[colIndex]
	}

	// Check if any foreign key value is NULL - NULL values are allowed in foreign keys
	hasNull := false
	for _, val := range fkValues {
		if val == nil {
			hasNull = true
			break
		}
	}
	if hasNull {
		return nil // NULL foreign key values are allowed
	}

	// Check if referenced values exist in the referenced table
	refColumnIndexes := make([]int, len(fk.RefColumns))
	for i, colName := range fk.RefColumns {
		colIndex := refTable.GetColumnIndex(colName)
		if colIndex == -1 {
			return fmt.Errorf("referenced column %s not found in table %s", colName, fk.RefTable)
		}
		refColumnIndexes[i] = colIndex
	}

	// Search for matching row in referenced table
	refTable.mutex.RLock()
	defer refTable.mutex.RUnlock()

	for _, row := range refTable.Rows {
		match := true
		for i, refColIndex := range refColumnIndexes {
			if !valuesEqual(fkValues[i], row.Values[refColIndex]) {
				match = false
				break
			}
		}
		if match {
			return nil // Found matching referenced row
		}
	}

	return fmt.Errorf("foreign key constraint violation: referenced row not found in table %s", fk.RefTable)
}

// ValidateForeignKeyDeletion validates that a row can be deleted without violating foreign key constraints
func (db *Database) ValidateForeignKeyDeletion(table *Table, row Row) error {
	// Check all tables for foreign keys that reference this table
	for _, otherTable := range db.Tables {
		if otherTable == table {
			continue // skip the same table
		}

		for _, fk := range otherTable.ForeignKeys {
			if strings.EqualFold(fk.RefTable, table.Name) {
				if err := db.validateReferencingRows(table, otherTable, fk, row); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// ExecuteForeignKeyDeletionActions executes the foreign key actions (CASCADE, SET NULL, SET DEFAULT) when deleting a row
func (db *Database) ExecuteForeignKeyDeletionActions(table *Table, row Row) error {
	// Check all tables for foreign keys that reference this table
	for _, otherTable := range db.Tables {
		if otherTable == table {
			continue // skip the same table
		}

		for _, fk := range otherTable.ForeignKeys {
			if strings.EqualFold(fk.RefTable, table.Name) {
				if err := db.executeForeignKeyAction(table, otherTable, fk, row); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// executeForeignKeyAction executes the foreign key action (CASCADE, SET NULL, SET DEFAULT) for a deleted row
func (db *Database) executeForeignKeyAction(refTable *Table, referencingTable *Table, fk ForeignKey, deletedRow Row) error {
	// Only execute actions for CASCADE, SET NULL, and SET DEFAULT
	if fk.OnDelete == FKActionRestrict || fk.OnDelete == FKActionNoAction {
		return nil // These are handled in validation phase
	}

	// Get the values being deleted from the referenced columns
	deletedValues := make([]interface{}, len(fk.RefColumns))
	for i, colName := range fk.RefColumns {
		colIndex := refTable.GetColumnIndex(colName)
		if colIndex == -1 {
			return fmt.Errorf("referenced column %s not found", colName)
		}
		deletedValues[i] = deletedRow.Values[colIndex]
	}

	// Get indexes of local columns in the referencing table
	localColumnIndexes := make([]int, len(fk.LocalColumns))
	for i, colName := range fk.LocalColumns {
		colIndex := referencingTable.GetColumnIndex(colName)
		if colIndex == -1 {
			return fmt.Errorf("local column %s not found in referencing table", colName)
		}
		localColumnIndexes[i] = colIndex
	}

	// Find and process matching rows
	referencingTable.mutex.Lock()
	defer referencingTable.mutex.Unlock()

	var indicesToDelete []int
	var rowsToUpdate []struct {
		index int
		row   Row
	}

	for i, row := range referencingTable.Rows {
		match := true
		hasNull := false

		// Check if foreign key values match the deleted values
		for j, localColIndex := range localColumnIndexes {
			localVal := row.Values[localColIndex]
			if localVal == nil {
				hasNull = true
				break
			}
			if !valuesEqual(localVal, deletedValues[j]) {
				match = false
				break
			}
		}

		if match && !hasNull {
			// Found a referencing row, apply the ON DELETE action
			switch fk.OnDelete {
			case FKActionCascade:
				// Mark row for deletion
				indicesToDelete = append(indicesToDelete, i)

			case FKActionSetNull:
				// Set foreign key columns to NULL
				newRow := Row{Values: make([]interface{}, len(row.Values))}
				copy(newRow.Values, row.Values)

				// Validate that foreign key columns can be NULL
				for _, localColIndex := range localColumnIndexes {
					col := referencingTable.Columns[localColIndex]
					if col.NotNull {
						return fmt.Errorf("cannot SET NULL on NOT NULL column %s", col.Name)
					}
					newRow.Values[localColIndex] = nil
				}

				rowsToUpdate = append(rowsToUpdate, struct {
					index int
					row   Row
				}{i, newRow})

			case FKActionSetDefault:
				// Set foreign key columns to their default values
				newRow := Row{Values: make([]interface{}, len(row.Values))}
				copy(newRow.Values, row.Values)

				for _, localColIndex := range localColumnIndexes {
					col := referencingTable.Columns[localColIndex]
					if col.Default == nil {
						return fmt.Errorf("cannot SET DEFAULT on column %s with no default value", col.Name)
					}
					newRow.Values[localColIndex] = col.Default
				}

				rowsToUpdate = append(rowsToUpdate, struct {
					index int
					row   Row
				}{i, newRow})
			}
		}
	}

	// Execute CASCADE deletions (delete rows from back to front to maintain indexes)
	if fk.OnDelete == FKActionCascade {
		// Before deleting, check if cascade delete would create more foreign key violations
		for _, index := range indicesToDelete {
			rowToDelete := referencingTable.Rows[index]
			if err := db.ValidateForeignKeyDeletion(referencingTable, rowToDelete); err != nil {
				return fmt.Errorf("cascade delete failed: %v", err)
			}
		}

		// Execute cascade deletions on dependent rows first
		for _, index := range indicesToDelete {
			rowToDelete := referencingTable.Rows[index]
			if err := db.ExecuteForeignKeyDeletionActions(referencingTable, rowToDelete); err != nil {
				return fmt.Errorf("cascade delete failed: %v", err)
			}
		}

		// Remove rows from back to front to maintain correct indexes
		for i := len(indicesToDelete) - 1; i >= 0; i-- {
			index := indicesToDelete[i]
			referencingTable.Rows = append(referencingTable.Rows[:index], referencingTable.Rows[index+1:]...)
		}
	}

	// Execute SET NULL and SET DEFAULT updates
	for _, update := range rowsToUpdate {
		// Validate foreign keys for the updated row
		if err := db.ValidateForeignKeys(referencingTable, update.row.Values); err != nil {
			return fmt.Errorf("foreign key action failed: %v", err)
		}
		referencingTable.Rows[update.index] = update.row
	}

	return nil
}

// validateReferencingRows checks if any rows in a referencing table would be violated by deleting a row (for RESTRICT/NO ACTION)
func (db *Database) validateReferencingRows(refTable *Table, referencingTable *Table, fk ForeignKey, deletedRow Row) error {
	// Only validate for RESTRICT and NO ACTION - other actions will be executed later
	if fk.OnDelete != FKActionRestrict && fk.OnDelete != FKActionNoAction {
		return nil // No validation needed for actions that will be executed
	}

	// Get the values being deleted from the referenced columns
	deletedValues := make([]interface{}, len(fk.RefColumns))
	for i, colName := range fk.RefColumns {
		colIndex := refTable.GetColumnIndex(colName)
		if colIndex == -1 {
			return fmt.Errorf("referenced column %s not found", colName)
		}
		deletedValues[i] = deletedRow.Values[colIndex]
	}

	// Get indexes of local columns in the referencing table
	localColumnIndexes := make([]int, len(fk.LocalColumns))
	for i, colName := range fk.LocalColumns {
		colIndex := referencingTable.GetColumnIndex(colName)
		if colIndex == -1 {
			return fmt.Errorf("local column %s not found in referencing table", colName)
		}
		localColumnIndexes[i] = colIndex
	}

	// Check if any row in the referencing table references the row being deleted
	referencingTable.mutex.RLock()
	defer referencingTable.mutex.RUnlock()

	for _, row := range referencingTable.Rows {
		match := true
		hasNull := false

		// Check if foreign key values match the deleted values
		for i, localColIndex := range localColumnIndexes {
			localVal := row.Values[localColIndex]
			if localVal == nil {
				hasNull = true
				break
			}
			if !valuesEqual(localVal, deletedValues[i]) {
				match = false
				break
			}
		}

		if match && !hasNull {
			// Found a referencing row with RESTRICT/NO ACTION
			return fmt.Errorf("foreign key constraint violation: cannot delete referenced row (table: %s)", refTable.Name)
		}
	}

	return nil
}

// valuesEqual compares two values for equality, handling different types appropriately
func valuesEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Convert both values to strings for comparison if they're not the same type
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)
	return aStr == bStr
}
