package mist

import (
	"fmt"
	"strings"

	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/model"
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
		// Handle ENUM type properly
		length := 255 // default max length for enum values
		return TypeEnum, length, 0, 0, nil
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
func parseColumnConstraints(colDef *ast.ColumnDef) (notNull, primary, unique, autoIncr bool, defaultValue, onUpdateValue interface{}, enumValues []string) {
	// Extract ENUM values if this is an enum column
	if colDef.Tp.GetType() == mysql.TypeEnum {
		for _, enumValue := range colDef.Tp.GetElems() {
			enumValues = append(enumValues, enumValue)
		}
	}

	for _, option := range colDef.Options {
		switch option.Tp {
		case ast.ColumnOptionNotNull:
			notNull = true
		case ast.ColumnOptionPrimaryKey:
			primary = true
		case ast.ColumnOptionAutoIncrement:
			autoIncr = true
		case ast.ColumnOptionUniqKey:
			// UNIQUE constraint
			unique = true
		case ast.ColumnOptionDefaultValue:
			if option.Expr != nil {
				// Handle CURRENT_TIMESTAMP and other default values
				if funcCall, ok := option.Expr.(*ast.FuncCallExpr); ok {
					funcNameUpper := strings.ToUpper(funcCall.FnName.L)
					if funcNameUpper == "CURRENT_TIMESTAMP" {
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
			// ON UPDATE CURRENT_TIMESTAMP
			if option.Expr != nil {
				if funcCall, ok := option.Expr.(*ast.FuncCallExpr); ok {
					if strings.ToUpper(funcCall.FnName.L) == "current_timestamp" {
						onUpdateValue = "CURRENT_TIMESTAMP"
					}
				} else if valueExpr, ok := option.Expr.(ast.ValueExpr); ok {
					onUpdateValue = valueExpr.GetValue()
				} else {
					onUpdateValue = fmt.Sprintf("%v", option.Expr)
				}
			}
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

		notNull, primary, unique, autoIncr, defaultValue, onUpdateValue, enumValues := parseColumnConstraints(col)

		column := Column{
			Name:       col.Name.Name.String(),
			Type:       colType,
			Length:     length,
			Precision:  precision,
			Scale:      scale,
			NotNull:    notNull,
			Primary:    primary,
			Unique:     unique,
			AutoIncr:   autoIncr,
			Default:    defaultValue,
			OnUpdate:   onUpdateValue,
			EnumValues: enumValues,
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
			// FOREIGN KEY constraints - now we'll process them
			if len(constraint.Keys) == 0 || constraint.Refer == nil || len(constraint.Refer.IndexPartSpecifications) == 0 {
				return fmt.Errorf("invalid foreign key constraint")
			}

			// Extract local column names
			var localColumns []string
			for _, key := range constraint.Keys {
				localColumns = append(localColumns, key.Column.Name.String())
			}

			// Extract referenced table and column names
			refTable := constraint.Refer.Table.Name.String()
			var refColumns []string
			for _, refCol := range constraint.Refer.IndexPartSpecifications {
				refColumns = append(refColumns, refCol.Column.Name.String())
			}

			if len(localColumns) != len(refColumns) {
				return fmt.Errorf("foreign key column count mismatch")
			}

			// Determine ON UPDATE and ON DELETE actions
			onUpdate := FKActionRestrict // default
			onDelete := FKActionRestrict // default

			if constraint.Refer.OnUpdate != nil {
				switch constraint.Refer.OnUpdate.ReferOpt {
				case model.ReferOptionCascade:
					onUpdate = FKActionCascade
				case model.ReferOptionSetNull:
					onUpdate = FKActionSetNull
				case model.ReferOptionSetDefault:
					onUpdate = FKActionSetDefault
				case model.ReferOptionNoAction:
					onUpdate = FKActionNoAction
				case model.ReferOptionRestrict:
					onUpdate = FKActionRestrict
				}
			}

			if constraint.Refer.OnDelete != nil {
				switch constraint.Refer.OnDelete.ReferOpt {
				case model.ReferOptionCascade:
					onDelete = FKActionCascade
				case model.ReferOptionSetNull:
					onDelete = FKActionSetNull
				case model.ReferOptionSetDefault:
					onDelete = FKActionSetDefault
				case model.ReferOptionNoAction:
					onDelete = FKActionNoAction
				case model.ReferOptionRestrict:
					onDelete = FKActionRestrict
				}
			}

			// Create foreign key constraint name
			constraintName := fmt.Sprintf("fk_%s_%s", tableName, strings.Join(localColumns, "_"))
			if constraint.Name != "" {
				constraintName = constraint.Name
			}

			// Store the foreign key info to be added after table creation
			fk := ForeignKey{
				Name:         constraintName,
				LocalColumns: localColumns,
				RefTable:     refTable,
				RefColumns:   refColumns,
				OnUpdate:     onUpdate,
				OnDelete:     onDelete,
			}

			// We'll add the foreign key after the table is created
			// Store it in a slice to be processed later
			defer func(database *Database, tablename string, foreignKey ForeignKey) {
				if table, err := database.GetTable(tablename); err == nil {
					table.AddForeignKey(foreignKey)
				}
			}(db, tableName, fk)
		case ast.ConstraintUniq, ast.ConstraintUniqKey, ast.ConstraintUniqIndex:
			// UNIQUE constraints - mark columns as unique
			for _, key := range constraint.Keys {
				colName := key.Column.Name.String()
				for i := range columns {
					if strings.EqualFold(columns[i].Name, colName) {
						columns[i].Unique = true
						break
					}
				}
			}
		}
	}

	// Create the table
	return db.CreateTable(tableName, columns)
}
