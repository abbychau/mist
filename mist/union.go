package mist

import (
	"fmt"
	"strings"

	"github.com/pingcap/tidb/pkg/parser/ast"
)

// ExecuteSetOperation is a placeholder for UNION operations
// UNION operations are currently not supported due to TiDB parser limitations
func ExecuteSetOperation(db *Database, stmt *ast.SetOprStmt) (*SelectResult, error) {
	return nil, fmt.Errorf("UNION operations are not yet supported due to parser limitations")
}

// executeUnionSelect executes a single SELECT statement within a UNION
func executeUnionSelect(db *Database, stmt *ast.SelectStmt) (*SelectResult, error) {
	// Create a temporary engine instance to execute the SELECT
	tempEngine := &SQLEngine{database: db}
	
	// Check if this is a JOIN query
	if tempEngine.isJoinQuery(stmt) {
		return ExecuteSelectWithJoin(db, stmt)
	}
	
	// Check if this is an aggregate query
	if hasAggregateFunction(stmt.Fields.Fields) {
		table, err := resolveTableFromSelect(db, stmt)
		if err != nil {
			return nil, err
		}
		return executeAggregateQuery(table, stmt.Fields.Fields, stmt.Where, stmt.GroupBy, stmt.Having, stmt.Limit)
	}

	// Regular SELECT
	return ExecuteSelect(db, stmt)
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

// mergeUnionResults combines multiple SelectResults into one
func mergeUnionResults(results []*SelectResult, isDistinct bool) *SelectResult {
	if len(results) == 0 {
		return &SelectResult{Columns: []string{}, Rows: [][]interface{}{}}
	}

	// Use first result's columns as the base
	mergedResult := &SelectResult{
		Columns: results[0].Columns,
		Rows:    [][]interface{}{},
	}

	var seenRows map[string]bool
	if isDistinct {
		seenRows = make(map[string]bool)
	}

	// Combine all rows
	for _, result := range results {
		for _, row := range result.Rows {
			if isDistinct {
				rowKey := createRowKey(row)
				if !seenRows[rowKey] {
					mergedResult.Rows = append(mergedResult.Rows, row)
					seenRows[rowKey] = true
				}
			} else {
				mergedResult.Rows = append(mergedResult.Rows, row)
			}
		}
	}

	return mergedResult
}

// createRowKey creates a string key for a row for duplicate detection
func createRowKey(row []interface{}) string {
	var parts []string
	for _, value := range row {
		if value == nil {
			parts = append(parts, "NULL")
		} else {
			parts = append(parts, fmt.Sprintf("%v", value))
		}
	}
	return strings.Join(parts, "|")
}

// isJoinQuery checks if a SELECT statement contains a JOIN
func (engine *SQLEngine) isUnionJoinQuery(stmt *ast.SelectStmt) bool {
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