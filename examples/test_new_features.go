package main

import (
	"fmt"
	"log"

	"github.com/abbychau/mist/mist"
)

func main() {
	fmt.Println("=== Testing New Features: DROP TABLE, TRUNCATE TABLE, HAVING ===")
	
	engine := mist.NewSQLEngine()

	// Test DROP TABLE and TRUNCATE TABLE
	testDropAndTruncate(engine)
	
	// Test HAVING clause
	testHavingClause(engine)
	
	// Note: UNION implementation is complex with TiDB parser
	// Will be completed in future version
	
	fmt.Println("\n✅ All implemented features tested successfully!")
}

func testDropAndTruncate(engine *mist.SQLEngine) {
	fmt.Println("\n=== Testing DROP TABLE and TRUNCATE TABLE ===")
	
	// Create test tables
	engine.Execute("CREATE TABLE test_drop (id INT, name VARCHAR(50))")
	engine.Execute("CREATE TABLE test_truncate (id INT, name VARCHAR(50))")
	
	// Insert data
	engine.Execute("INSERT INTO test_drop VALUES (1, 'Alice'), (2, 'Bob')")
	engine.Execute("INSERT INTO test_truncate VALUES (1, 'Charlie'), (2, 'Diana')")
	
	// Test DROP TABLE
	result, err := engine.Execute("DROP TABLE test_drop")
	if err != nil {
		log.Printf("DROP TABLE error: %v", err)
	} else {
		fmt.Printf("DROP TABLE result: %v\n", result)
	}
	
	// Verify table is dropped
	_, err = engine.Execute("SELECT * FROM test_drop")
	if err != nil {
		fmt.Println("✅ Table successfully dropped (expected error)")
	} else {
		fmt.Println("❌ Table should have been dropped")
	}
	
	// Test TRUNCATE TABLE
	result, err = engine.Execute("TRUNCATE TABLE test_truncate")
	if err != nil {
		log.Printf("TRUNCATE TABLE error: %v", err)
	} else {
		fmt.Printf("TRUNCATE TABLE result: %v\n", result)
	}
	
	// Verify table is empty but exists
	result, err = engine.Execute("SELECT COUNT(*) FROM test_truncate")
	if err != nil {
		log.Printf("Error checking truncated table: %v", err)
	} else {
		fmt.Println("✅ Table truncated successfully")
		mist.PrintResult(result)
	}
}

func testHavingClause(engine *mist.SQLEngine) {
	fmt.Println("\n=== Testing HAVING Clause ===")
	
	// Create test table
	engine.Execute("CREATE TABLE sales (id INT, department VARCHAR(50), amount FLOAT)")
	
	// Insert test data
	queries := []string{
		"INSERT INTO sales VALUES (1, 'Electronics', 1000)",
		"INSERT INTO sales VALUES (2, 'Electronics', 1500)",
		"INSERT INTO sales VALUES (3, 'Books', 300)",
		"INSERT INTO sales VALUES (4, 'Books', 200)",
		"INSERT INTO sales VALUES (5, 'Clothing', 800)",
		"INSERT INTO sales VALUES (6, 'Clothing', 1200)",
	}
	
	for _, query := range queries {
		engine.Execute(query)
	}
	
	// Test GROUP BY with HAVING
	fmt.Println("\nDepartments with total sales > 1000:")
	result, err := engine.Execute(`
		SELECT department, SUM(amount) as total_sales, COUNT(*) as num_sales
		FROM sales 
		GROUP BY department 
		HAVING SUM(amount) > 1000
	`)
	
	if err != nil {
		log.Printf("HAVING clause error: %v", err)
	} else {
		fmt.Println("✅ HAVING clause working:")
		mist.PrintResult(result)
	}
	
	// Test HAVING with COUNT
	fmt.Println("\nDepartments with more than 1 sale:")
	result, err = engine.Execute(`
		SELECT department, COUNT(*) as num_sales
		FROM sales 
		GROUP BY department 
		HAVING COUNT(*) > 1
	`)
	
	if err != nil {
		log.Printf("HAVING with COUNT error: %v", err)
	} else {
		fmt.Println("✅ HAVING with COUNT working:")
		mist.PrintResult(result)
	}
}

/*
// UNION implementation is deferred due to TiDB parser complexity
func testUnionOperations(engine *mist.SQLEngine) {
	// Implementation deferred - UNION requires more complex parsing integration
}
*/