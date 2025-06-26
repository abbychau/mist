package main

import (
	"fmt"
	"github.com/abbychau/mist"
)

func main() {
	engine := mist.NewSQLEngine()
	
	// Create and populate test tables
	engine.Execute("CREATE TABLE t1 (id INT, name VARCHAR(50))")
	engine.Execute("CREATE TABLE t2 (id INT, name VARCHAR(50))")
	engine.Execute("INSERT INTO t1 VALUES (1, 'Alice'), (2, 'Bob')")
	engine.Execute("INSERT INTO t2 VALUES (2, 'Bob'), (3, 'Carol')")
	
	// Test UNION DISTINCT
	fmt.Println("=== UNION (should eliminate duplicate Bob) ===")
	result, _ := engine.Execute("SELECT * FROM t1 UNION SELECT * FROM t2")
	mist.PrintResult(result)
	
	// Test UNION ALL
	fmt.Println("\n=== UNION ALL (should keep duplicate Bob) ===")
	result, _ = engine.Execute("SELECT * FROM t1 UNION ALL SELECT * FROM t2")
	mist.PrintResult(result)
}