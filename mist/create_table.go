package mist

import (
	"fmt"
	"strings"

	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/mysql"
)

// parseColumnType converts TiDB column type to our internal type
func parseColumnType(colDef *ast.ColumnDef) (ColumnType, int, error) {
	tp := colDef.Tp

	switch tp.GetType() {
	case mysql.TypeTiny, mysql.TypeShort, mysql.TypeLong, mysql.TypeLonglong, mysql.TypeInt24:
		return TypeInt, 0, nil
	case mysql.TypeVarchar:
		length := 255 // default
		if tp.GetFlen() > 0 {
			length = tp.GetFlen()
		}
		return TypeVarchar, length, nil
	case mysql.TypeString, mysql.TypeVarString:
		length := 255 // default
		if tp.GetFlen() > 0 {
			length = tp.GetFlen()
		}
		return TypeVarchar, length, nil
	case mysql.TypeBlob, mysql.TypeTinyBlob, mysql.TypeMediumBlob, mysql.TypeLongBlob:
		return TypeText, 0, nil
	case mysql.TypeFloat, mysql.TypeDouble, mysql.TypeNewDecimal:
		return TypeFloat, 0, nil
	case mysql.TypeBit:
		return TypeBool, 0, nil
	default:
		return TypeText, 0, fmt.Errorf("unsupported column type: %v", tp.GetType())
	}
}

// parseColumnConstraints extracts constraints from column definition
func parseColumnConstraints(colDef *ast.ColumnDef) (notNull, primary, autoIncr bool) {
	for _, option := range colDef.Options {
		switch option.Tp {
		case ast.ColumnOptionNotNull:
			notNull = true
		case ast.ColumnOptionPrimaryKey:
			primary = true
		case ast.ColumnOptionAutoIncrement:
			autoIncr = true
		}
	}
	return
}

// ExecuteCreateTable processes a CREATE TABLE statement
func ExecuteCreateTable(db *Database, stmt *ast.CreateTableStmt) error {
	tableName := stmt.Table.Name.String()

	// Check if IF NOT EXISTS is specified
	if stmt.IfNotExists {
		if _, err := db.GetTable(tableName); err == nil {
			// Table exists, but IF NOT EXISTS was specified, so this is not an error
			return nil
		}
	}

	var columns []Column

	// Process column definitions
	for _, col := range stmt.Cols {
		colType, length, err := parseColumnType(col)
		if err != nil {
			return fmt.Errorf("error parsing column %s: %v", col.Name.Name.String(), err)
		}

		notNull, primary, autoIncr := parseColumnConstraints(col)

		column := Column{
			Name:     col.Name.Name.String(),
			Type:     colType,
			Length:   length,
			NotNull:  notNull,
			Primary:  primary,
			AutoIncr: autoIncr,
		}

		columns = append(columns, column)
	}

	// Process table constraints (like PRIMARY KEY)
	for _, constraint := range stmt.Constraints {
		switch constraint.Tp {
		case ast.ConstraintPrimaryKey:
			// Mark columns as primary key
			for _, key := range constraint.Keys {
				colName := key.Column.Name.String()
				for i := range columns {
					if strings.EqualFold(columns[i].Name, colName) {
						columns[i].Primary = true
						columns[i].NotNull = true // Primary keys are implicitly NOT NULL
						break
					}
				}
			}
		}
	}

	// Create the table
	return db.CreateTable(tableName, columns)
}
