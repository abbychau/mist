package main

import (
	"fmt"
	"strings"

	"github.com/abbychau/mist"
)

func main() {
	fmt.Println("=== Scalar Subquery Unit Tests ===")
	
	passed := 0
	total := 0
	
	// Test 1: Basic scalar subquery in WHERE
	total++
	if testBasicScalarWhere() {
		fmt.Println("✅ Test 1: Basic scalar subquery in WHERE - PASSED")
		passed++
	} else {
		fmt.Println("❌ Test 1: Basic scalar subquery in WHERE - FAILED")
	}
	
	// Test 2: Scalar subquery in SELECT
	total++
	if testScalarInSelect() {
		fmt.Println("✅ Test 2: Scalar subquery in SELECT - PASSED")
		passed++
	} else {
		fmt.Println("❌ Test 2: Scalar subquery in SELECT - FAILED")
	}
	
	// Test 3: Scalar subquery with aggregates
	total++
	if testScalarWithAggregates() {
		fmt.Println("✅ Test 3: Scalar subquery with aggregates - PASSED")
		passed++
	} else {
		fmt.Println("❌ Test 3: Scalar subquery with aggregates - FAILED")
	}
	
	// Test 4: NULL handling
	total++
	if testScalarNullHandling() {
		fmt.Println("✅ Test 4: Scalar subquery NULL handling - PASSED")
		passed++
	} else {
		fmt.Println("❌ Test 4: Scalar subquery NULL handling - FAILED")
	}
	
	// Test 5: Error cases
	total++
	if testScalarErrorCases() {
		fmt.Println("✅ Test 5: Scalar subquery error cases - PASSED")
		passed++
	} else {
		fmt.Println("❌ Test 5: Scalar subquery error cases - FAILED")
	}
	
	// Test 6: Nested scalar subqueries
	total++
	if testNestedScalarSubqueries() {
		fmt.Println("✅ Test 6: Nested scalar subqueries - PASSED")
		passed++
	} else {
		fmt.Println("❌ Test 6: Nested scalar subqueries - FAILED")
	}
	
	fmt.Printf("\n=== Test Summary ===\n")
	fmt.Printf("Passed: %d/%d\n", passed, total)
	if passed == total {
		fmt.Println("🎉 All tests passed!")
	} else {
		fmt.Printf("⚠️  %d test(s) failed\n", total-passed)
	}
}

func setupTestData() *mist.SQLEngine {
	engine := mist.NewSQLEngine()
	
	// Setup tables
	engine.Execute("CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50), department_id INT)")
	engine.Execute("CREATE TABLE departments (id INT PRIMARY KEY, name VARCHAR(50), budget DECIMAL(10,2))")
	
	// Insert data
	engine.Execute("INSERT INTO departments VALUES (1, 'Engineering', 100000.00), (2, 'Sales', 50000.00)")
	engine.Execute("INSERT INTO users VALUES (1, 'Alice', 1), (2, 'Bob', 2), (3, 'Charlie', 1)")
	
	return engine
}

func testBasicScalarWhere() bool {
	engine := setupTestData()
	
	result, err := engine.Execute("SELECT name FROM users WHERE department_id = (SELECT id FROM departments WHERE name = 'Engineering')")
	if err != nil {
		fmt.Printf("    Error: %v\n", err)
		return false
	}
	
	selectResult, ok := result.(*mist.SelectResult)
	if !ok {
		fmt.Printf("    Error: Expected SelectResult, got %T\n", result)
		return false
	}
	
	// Should return Alice and Charlie
	if len(selectResult.Rows) != 2 {
		fmt.Printf("    Error: Expected 2 rows, got %d\n", len(selectResult.Rows))
		return false
	}
	
	return true
}

func testScalarInSelect() bool {
	engine := setupTestData()
	
	result, err := engine.Execute("SELECT name, (SELECT MAX(budget) FROM departments) as max_budget FROM users LIMIT 1")
	if err != nil {
		fmt.Printf("    Error: %v\n", err)
		return false
	}
	
	selectResult, ok := result.(*mist.SelectResult)
	if !ok {
		fmt.Printf("    Error: Expected SelectResult, got %T\n", result)
		return false
	}
	
	// Should return 1 row with 2 columns
	if len(selectResult.Rows) != 1 || len(selectResult.Rows[0]) != 2 {
		fmt.Printf("    Error: Expected 1 row with 2 columns, got %d rows\n", len(selectResult.Rows))
		return false
	}
	
	// Check that max_budget is 100000.00
	if fmt.Sprintf("%v", selectResult.Rows[0][1]) != "100000.00" {
		fmt.Printf("    Error: Expected max_budget 100000.00, got %v\n", selectResult.Rows[0][1])
		return false
	}
	
	return true
}

func testScalarWithAggregates() bool {
	engine := setupTestData()
	
	result, err := engine.Execute("SELECT name FROM users WHERE id <= (SELECT COUNT(*) FROM departments)")
	if err != nil {
		fmt.Printf("    Error: %v\n", err)
		return false
	}
	
	selectResult, ok := result.(*mist.SelectResult)
	if !ok {
		fmt.Printf("    Error: Expected SelectResult, got %T\n", result)
		return false
	}
	
	// COUNT(*) FROM departments = 2, so users with id <= 2 (Alice, Bob)
	if len(selectResult.Rows) != 2 {
		fmt.Printf("    Error: Expected 2 rows, got %d\n", len(selectResult.Rows))
		return false
	}
	
	return true
}

func testScalarNullHandling() bool {
	engine := setupTestData()
	
	result, err := engine.Execute("SELECT * FROM users WHERE department_id = (SELECT id FROM departments WHERE name = 'NonExistent')")
	if err != nil {
		fmt.Printf("    Error: %v\n", err)
		return false
	}
	
	selectResult, ok := result.(*mist.SelectResult)
	if !ok {
		fmt.Printf("    Error: Expected SelectResult, got %T\n", result)
		return false
	}
	
	// Should return no rows (NULL comparison fails)
	if len(selectResult.Rows) != 0 {
		fmt.Printf("    Error: Expected 0 rows, got %d\n", len(selectResult.Rows))
		return false
	}
	
	return true
}

func testScalarErrorCases() bool {
	engine := setupTestData()
	
	// Test multiple rows error
	_, err := engine.Execute("SELECT * FROM users WHERE department_id = (SELECT id FROM departments)")
	if err == nil || !strings.Contains(err.Error(), "more than one row") {
		fmt.Printf("    Error: Expected 'more than one row' error, got: %v\n", err)
		return false
	}
	
	// Test multiple columns error
	_, err = engine.Execute("SELECT * FROM users WHERE department_id = (SELECT id, name FROM departments WHERE id = 1)")
	if err == nil || !strings.Contains(err.Error(), "more than one column") {
		fmt.Printf("    Error: Expected 'more than one column' error, got: %v\n", err)
		return false
	}
	
	return true
}

func testNestedScalarSubqueries() bool {
	engine := setupTestData()
	
	result, err := engine.Execute("SELECT * FROM users WHERE department_id = (SELECT id FROM departments WHERE budget > (SELECT AVG(budget) FROM departments))")
	if err != nil {
		fmt.Printf("    Error: %v\n", err)
		return false
	}
	
	selectResult, ok := result.(*mist.SelectResult)
	if !ok {
		fmt.Printf("    Error: Expected SelectResult, got %T\n", result)
		return false
	}
	
	// AVG(budget) = 75000, so Engineering (100000) > 75000
	// Should return Alice and Charlie
	if len(selectResult.Rows) != 2 {
		fmt.Printf("    Error: Expected 2 rows, got %d\n", len(selectResult.Rows))
		return false
	}
	
	return true
}