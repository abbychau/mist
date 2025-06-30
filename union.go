package mist

import (
	"fmt"

	"github.com/abbychau/mysql-parser/ast"
)

// ExecuteUnion executes a UNION statement combining multiple SELECT results
func ExecuteUnion(db *Database, stmt *ast.SetOprStmt) (*SelectResult, error) {
	if stmt.SelectList == nil || len(stmt.SelectList.Selects) == 0 {
		return nil, fmt.Errorf("UNION statement must contain at least one SELECT")
	}

	// Execute all SELECT statements and collect results
	var allResults []*SelectResult
	var finalColumns []string
	
	// Process each SELECT statement in the UNION
	for i, selectNode := range stmt.SelectList.Selects {
		var result *SelectResult
		var err error
		
		switch selectStmt := selectNode.(type) {
		case *ast.SelectStmt:
			// Check if this is a JOIN query
			if isUnionJoinQuery(selectStmt) {
				result, err = ExecuteSelectWithJoin(db, selectStmt)
			} else {
				result, err = ExecuteSelect(db, selectStmt)
			}
			if err != nil {
				return nil, fmt.Errorf("error executing SELECT %d in UNION: %v", i+1, err)
			}
			
		case *ast.SetOprStmt:
			// Nested UNION operation
			result, err = ExecuteUnion(db, selectStmt)
			if err != nil {
				return nil, fmt.Errorf("error executing nested UNION %d: %v", i+1, err)
			}
			
		default:
			return nil, fmt.Errorf("unsupported statement type in UNION: %T", selectNode)
		}
		
		// Validate column compatibility
		if i == 0 {
			// First SELECT defines the column structure
			finalColumns = make([]string, len(result.Columns))
			copy(finalColumns, result.Columns)
		} else {
			// Subsequent SELECTs must have the same number of columns
			if len(result.Columns) != len(finalColumns) {
				return nil, fmt.Errorf("UNION SELECT %d has %d columns, expected %d", 
					i+1, len(result.Columns), len(finalColumns))
			}
		}
		
		allResults = append(allResults, result)
	}
	
	// Determine the UNION operation type for each SELECT
	unionTypes := getUnionTypes(stmt.SelectList.Selects)
	
	// Combine results according to UNION semantics
	return combineUnionResults(allResults, unionTypes, finalColumns)
}

// isUnionJoinQuery checks if a SELECT statement contains a JOIN (helper for UNION)
func isUnionJoinQuery(stmt *ast.SelectStmt) bool {
	if stmt.From == nil || stmt.From.TableRefs == nil {
		return false
	}

	// Check if TableRefs has a Right side (indicating a JOIN)
	if stmt.From.TableRefs.Right != nil {
		return true
	}

	// Check for comma-separated tables (cross join)
	if join, ok := stmt.From.TableRefs.Left.(*ast.Join); ok {
		if join.Tp == 0 && join.Right == nil {
			return true
		}
	}

	return false
}

// getUnionTypes extracts the union operation types for each SELECT
func getUnionTypes(selects []ast.Node) []ast.SetOprType {
	unionTypes := make([]ast.SetOprType, len(selects))
	
	for i, selectNode := range selects {
		if i == 0 {
			// First SELECT has no preceding operation
			unionTypes[i] = ast.Union // Default, but not used
			continue
		}
		
		switch selectStmt := selectNode.(type) {
		case *ast.SelectStmt:
			if selectStmt.AfterSetOperator != nil {
				unionTypes[i] = *selectStmt.AfterSetOperator
			} else {
				unionTypes[i] = ast.Union // Default to UNION DISTINCT
			}
			
		case *ast.SetOprSelectList:
			if selectStmt.AfterSetOperator != nil {
				unionTypes[i] = *selectStmt.AfterSetOperator
			} else {
				unionTypes[i] = ast.Union // Default to UNION DISTINCT
			}
			
		default:
			unionTypes[i] = ast.Union // Default fallback
		}
	}
	
	return unionTypes
}

// combineUnionResults combines multiple SELECT results according to UNION semantics
func combineUnionResults(results []*SelectResult, unionTypes []ast.SetOprType, finalColumns []string) (*SelectResult, error) {
	if len(results) == 0 {
		return &SelectResult{Columns: finalColumns, Rows: [][]interface{}{}}, nil
	}
	
	// Start with the first result
	combinedResult := &SelectResult{
		Columns: finalColumns,
		Rows:    make([][]interface{}, 0),
	}
	
	// Copy rows from first result
	for _, row := range results[0].Rows {
		combinedResult.Rows = append(combinedResult.Rows, row)
	}
	
	// Process each subsequent result
	for i := 1; i < len(results); i++ {
		unionType := unionTypes[i]
		
		switch unionType {
		case ast.Union:
			// UNION DISTINCT - add rows but eliminate duplicates
			for _, row := range results[i].Rows {
				if !containsRow(combinedResult.Rows, row) {
					combinedResult.Rows = append(combinedResult.Rows, row)
				}
			}
			
		case ast.UnionAll:
			// UNION ALL - add all rows without checking duplicates
			for _, row := range results[i].Rows {
				combinedResult.Rows = append(combinedResult.Rows, row)
			}
			
		default:
			return nil, fmt.Errorf("unsupported set operation type: %v", unionType)
		}
	}
	
	return combinedResult, nil
}

// containsRow checks if a slice of rows contains a specific row (for UNION DISTINCT)
func containsRow(rows [][]interface{}, targetRow []interface{}) bool {
	for _, row := range rows {
		if len(row) != len(targetRow) {
			continue
		}
		
		match := true
		for j := 0; j < len(row); j++ {
			if !valuesEqual(row[j], targetRow[j]) {
				match = false
				break
			}
		}
		
		if match {
			return true
		}
	}
	
	return false
}

// ExecuteSetOperation is the legacy function now redirecting to ExecuteUnion
func ExecuteSetOperation(db *Database, stmt *ast.SetOprStmt) (*SelectResult, error) {
	return ExecuteUnion(db, stmt)
}

// resolveTableFromSelect resolves the main table from a SELECT statement
func resolveTableFromSelect(db *Database, stmt *ast.SelectStmt) (*Table, error) {
	if stmt.From == nil || stmt.From.TableRefs == nil {
		return nil, fmt.Errorf("SELECT statement has no FROM clause")
	}

	// Handle different table reference types
	switch ref := stmt.From.TableRefs.Left.(type) {
	case *ast.TableSource:
		if tableName, ok := ref.Source.(*ast.TableName); ok {
			return db.GetTable(tableName.Name.String())
		}
	case *ast.TableName:
		return db.GetTable(ref.Name.String())
	}

	return nil, fmt.Errorf("could not resolve table from SELECT statement")
}