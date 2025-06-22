package mist

import (
	"fmt"
	"strings"

	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/mysql"
)

// parseColumnType converts TiDB column type to our internal type
func parseColumnType(colDef *ast.ColumnDef) (ColumnType, int, int, int, error) {
	tp := colDef.Tp

	switch tp.GetType() {
	case mysql.TypeTiny, mysql.TypeShort, mysql.TypeLong, mysql.TypeLonglong, mysql.TypeInt24:
		return TypeInt, 0, 0, 0, nil
	case mysql.TypeVarchar:
		length := 255 // default
		if tp.GetFlen() > 0 {
			length = tp.GetFlen()
		}
		return TypeVarchar, length, 0, 0, nil
	case mysql.TypeString, mysql.TypeVarString:
		length := 255 // default
		if tp.GetFlen() > 0 {
			length = tp.GetFlen()
		}
		return TypeVarchar, length, 0, 0, nil
	case mysql.TypeEnum:
		// Treat ENUM as VARCHAR for compatibility
		// ENUM values will be stored as strings
		return TypeVarchar, 255, 0, 0, nil
	case mysql.TypeBlob, mysql.TypeTinyBlob, mysql.TypeMediumBlob, mysql.TypeLongBlob:
		return TypeText, 0, 0, 0, nil
	case mysql.TypeFloat, mysql.TypeDouble:
		return TypeFloat, 0, 0, 0, nil
	case mysql.TypeNewDecimal:
		precision := 10 // default precision
		scale := 0      // default scale
		if tp.GetFlen() > 0 {
			precision = tp.GetFlen()
		}
		if tp.GetDecimal() >= 0 {
			scale = tp.GetDecimal()
		}
		return TypeDecimal, 0, precision, scale, nil
	case mysql.TypeTimestamp:
		return TypeTimestamp, 0, 0, 0, nil
	case mysql.TypeDate:
		return TypeDate, 0, 0, 0, nil
	case mysql.TypeDatetime:
		return TypeTimestamp, 0, 0, 0, nil
	case mysql.TypeBit:
		return TypeBool, 0, 0, 0, nil
	default:
		return TypeText, 0, 0, 0, fmt.Errorf("unsupported column type: %v", tp.GetType())
	}
}

// parseColumnConstraints extracts constraints from column definition
func parseColumnConstraints(colDef *ast.ColumnDef) (notNull, primary, autoIncr bool, defaultValue interface{}) {
	for _, option := range colDef.Options {
		switch option.Tp {
		case ast.ColumnOptionNotNull:
			notNull = true
		case ast.ColumnOptionPrimaryKey:
			primary = true
		case ast.ColumnOptionAutoIncrement:
			autoIncr = true
		case ast.ColumnOptionUniqKey:
			// UNIQUE constraint - for now, we'll ignore it but not error
			// In a full implementation, this would create a unique index
			continue
		case ast.ColumnOptionDefaultValue:
			if option.Expr != nil {
				// Handle CURRENT_TIMESTAMP and other default values
				if funcCall, ok := option.Expr.(*ast.FuncCallExpr); ok {
					if strings.ToUpper(funcCall.FnName.L) == "CURRENT_TIMESTAMP" {
						defaultValue = "CURRENT_TIMESTAMP"
					}
				} else if valueExpr, ok := option.Expr.(ast.ValueExpr); ok {
					// Handle literal default values
					defaultValue = valueExpr.GetValue()
				} else {
					// Fallback for other expression types
					defaultValue = fmt.Sprintf("%v", option.Expr)
				}
			}
		case ast.ColumnOptionOnUpdate:
			// ON UPDATE CURRENT_TIMESTAMP - ignore for now
			// In a full implementation, this would be handled during UPDATE operations
			continue
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
		colType, length, precision, scale, err := parseColumnType(col)
		if err != nil {
			return fmt.Errorf("error parsing column %s: %v", col.Name.Name.String(), err)
		}

		notNull, primary, autoIncr, defaultValue := parseColumnConstraints(col)

		column := Column{
			Name:      col.Name.Name.String(),
			Type:      colType,
			Length:    length,
			Precision: precision,
			Scale:     scale,
			NotNull:   notNull,
			Primary:   primary,
			AutoIncr:  autoIncr,
			Default:   defaultValue,
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
		case ast.ConstraintForeignKey:
			// FOREIGN KEY constraints - ignore for now but don't error
			// In a full implementation, this would enforce referential integrity
			continue
		case ast.ConstraintUniq, ast.ConstraintUniqKey, ast.ConstraintUniqIndex:
			// UNIQUE constraints - ignore for now but don't error
			// In a full implementation, this would create unique indexes
			continue
		}
	}

	// Create the table
	return db.CreateTable(tableName, columns)
}
