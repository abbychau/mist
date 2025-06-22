package mist

import (
	"fmt"
	"strings"

	"github.com/pingcap/tidb/pkg/parser/ast"
)

// ExecuteAlterTable processes an ALTER TABLE statement
func ExecuteAlterTable(db *Database, stmt *ast.AlterTableStmt) error {
	tableName := stmt.Table.Name.String()

	// Get the table
	table, err := db.GetTable(tableName)
	if err != nil {
		return err
	}

	// Process each ALTER specification
	for _, spec := range stmt.Specs {
		switch spec.Tp {
		case ast.AlterTableAddColumns:
			err = executeAddColumn(db, table, spec)
		case ast.AlterTableDropColumn:
			err = executeDropColumn(db, table, spec)
		case ast.AlterTableModifyColumn:
			err = executeModifyColumn(db, table, spec)
		case ast.AlterTableChangeColumn:
			err = executeChangeColumn(db, table, spec)
		default:
			return fmt.Errorf("unsupported ALTER TABLE operation: %v", spec.Tp)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

// executeAddColumn adds a new column to the table
func executeAddColumn(db *Database, table *Table, spec *ast.AlterTableSpec) error {
	if len(spec.NewColumns) == 0 {
		return fmt.Errorf("no columns specified for ADD COLUMN")
	}

	table.mutex.Lock()
	defer table.mutex.Unlock()

	for _, colDef := range spec.NewColumns {
		// Parse the new column
		colType, length, err := parseColumnType(colDef)
		if err != nil {
			return fmt.Errorf("error parsing new column %s: %v", colDef.Name.Name.String(), err)
		}

		notNull, primary, autoIncr := parseColumnConstraints(colDef)

		newColumn := Column{
			Name:     colDef.Name.Name.String(),
			Type:     colType,
			Length:   length,
			NotNull:  notNull,
			Primary:  primary,
			AutoIncr: autoIncr,
		}

		// Check if column already exists
		if table.GetColumnIndex(newColumn.Name) != -1 {
			return fmt.Errorf("column %s already exists", newColumn.Name)
		}

		// Add the column to the table schema
		table.Columns = append(table.Columns, newColumn)

		// Add default value to all existing rows
		defaultValue := getDefaultValue(newColumn)
		for i := range table.Rows {
			table.Rows[i].Values = append(table.Rows[i].Values, defaultValue)
		}
	}

	return nil
}

// executeDropColumn removes a column from the table
func executeDropColumn(db *Database, table *Table, spec *ast.AlterTableSpec) error {
	if spec.OldColumnName == nil {
		return fmt.Errorf("no column specified for DROP COLUMN")
	}

	columnName := spec.OldColumnName.Name.String()
	colIndex := table.GetColumnIndex(columnName)
	if colIndex == -1 {
		return fmt.Errorf("column %s does not exist", columnName)
	}

	table.mutex.Lock()
	defer table.mutex.Unlock()

	// Remove column from schema
	table.Columns = append(table.Columns[:colIndex], table.Columns[colIndex+1:]...)

	// Remove column data from all rows
	for i := range table.Rows {
		table.Rows[i].Values = append(table.Rows[i].Values[:colIndex], table.Rows[i].Values[colIndex+1:]...)
	}

	// Update any indexes that reference this column
	indexesToDrop := make([]string, 0)
	for _, indexName := range db.IndexManager.ListIndexes() {
		if index, exists := db.IndexManager.GetIndex(indexName); exists {
			if strings.EqualFold(index.TableName, table.Name) && strings.EqualFold(index.ColumnName, columnName) {
				indexesToDrop = append(indexesToDrop, indexName)
			}
		}
	}

	// Drop affected indexes
	for _, indexName := range indexesToDrop {
		_ = db.IndexManager.DropIndex(indexName)
	}

	return nil
}

// executeModifyColumn modifies an existing column
func executeModifyColumn(db *Database, table *Table, spec *ast.AlterTableSpec) error {
	if len(spec.NewColumns) == 0 {
		return fmt.Errorf("no column specified for MODIFY COLUMN")
	}

	colDef := spec.NewColumns[0]
	columnName := colDef.Name.Name.String()
	colIndex := table.GetColumnIndex(columnName)
	if colIndex == -1 {
		return fmt.Errorf("column %s does not exist", columnName)
	}

	// Parse the new column definition
	colType, length, err := parseColumnType(colDef)
	if err != nil {
		return fmt.Errorf("error parsing modified column %s: %v", columnName, err)
	}

	notNull, primary, autoIncr := parseColumnConstraints(colDef)

	table.mutex.Lock()
	defer table.mutex.Unlock()

	// Update the column definition
	table.Columns[colIndex] = Column{
		Name:     columnName,
		Type:     colType,
		Length:   length,
		NotNull:  notNull,
		Primary:  primary,
		AutoIncr: autoIncr,
	}

	// Convert existing data to new type if possible
	for i := range table.Rows {
		if colIndex < len(table.Rows[i].Values) {
			convertedValue, err := convertValueToColumnType(table.Rows[i].Values[colIndex], colType)
			if err != nil {
				return fmt.Errorf("cannot convert existing data in row %d: %v", i, err)
			}
			table.Rows[i].Values[colIndex] = convertedValue
		}
	}

	return nil
}

// executeChangeColumn renames and/or modifies a column
func executeChangeColumn(db *Database, table *Table, spec *ast.AlterTableSpec) error {
	if spec.OldColumnName == nil || len(spec.NewColumns) == 0 {
		return fmt.Errorf("invalid CHANGE COLUMN specification")
	}

	oldColumnName := spec.OldColumnName.Name.String()
	colIndex := table.GetColumnIndex(oldColumnName)
	if colIndex == -1 {
		return fmt.Errorf("column %s does not exist", oldColumnName)
	}

	colDef := spec.NewColumns[0]
	newColumnName := colDef.Name.Name.String()

	// Check if new name conflicts with existing columns (unless it's the same column)
	if !strings.EqualFold(oldColumnName, newColumnName) {
		if table.GetColumnIndex(newColumnName) != -1 {
			return fmt.Errorf("column %s already exists", newColumnName)
		}
	}

	// Parse the new column definition
	colType, length, err := parseColumnType(colDef)
	if err != nil {
		return fmt.Errorf("error parsing changed column %s: %v", newColumnName, err)
	}

	notNull, primary, autoIncr := parseColumnConstraints(colDef)

	table.mutex.Lock()
	defer table.mutex.Unlock()

	// Update the column definition
	table.Columns[colIndex] = Column{
		Name:     newColumnName,
		Type:     colType,
		Length:   length,
		NotNull:  notNull,
		Primary:  primary,
		AutoIncr: autoIncr,
	}

	// Convert existing data to new type if possible
	for i := range table.Rows {
		if colIndex < len(table.Rows[i].Values) {
			convertedValue, err := convertValueToColumnType(table.Rows[i].Values[colIndex], colType)
			if err != nil {
				return fmt.Errorf("cannot convert existing data in row %d: %v", i, err)
			}
			table.Rows[i].Values[colIndex] = convertedValue
		}
	}

	// Update indexes that reference the old column name
	for _, indexName := range db.IndexManager.ListIndexes() {
		if index, exists := db.IndexManager.GetIndex(indexName); exists {
			if strings.EqualFold(index.TableName, table.Name) && strings.EqualFold(index.ColumnName, oldColumnName) {
				// Update the index column name
				index.ColumnName = newColumnName
				// Rebuild the index with the new column name
				_ = index.RebuildIndex(table)
			}
		}
	}

	return nil
}

// getDefaultValue returns an appropriate default value for a column type
func getDefaultValue(column Column) interface{} {
	if !column.NotNull {
		return nil
	}

	switch column.Type {
	case TypeInt:
		return int64(0)
	case TypeFloat:
		return float64(0)
	case TypeVarchar, TypeText:
		return ""
	case TypeBool:
		return false
	default:
		return nil
	}
}
