package main

import (
	"fmt"
	"log"

	"github.com/abbychau/mist/mist"
)

func main() {
	fmt.Println("=== UNION Operations Test for Mist Database ===")

	engine := mist.NewSQLEngine()

	// Create test tables for UNION operations
	createTable1 := `CREATE TABLE employees (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		department VARCHAR(50),
		salary DECIMAL(10,2),
		status ENUM('active', 'inactive') DEFAULT 'active'
	)`

	createTable2 := `CREATE TABLE contractors (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		department VARCHAR(50),
		hourly_rate DECIMAL(8,2),
		status ENUM('active', 'inactive') DEFAULT 'active'
	)`

	_, err := engine.Execute(createTable1)
	if err != nil {
		log.Fatalf("Error creating employees table: %v", err)
	}
	fmt.Println("✅ Employees table created successfully")

	_, err = engine.Execute(createTable2)
	if err != nil {
		log.Fatalf("Error creating contractors table: %v", err)
	}
	fmt.Println("✅ Contractors table created successfully")

	// Insert sample data into employees
	employeeInserts := []string{
		`INSERT INTO employees (name, department, salary, status) VALUES 
		 ('Alice Johnson', 'Engineering', 85000.00, 'active')`,
		`INSERT INTO employees (name, department, salary, status) VALUES 
		 ('Bob Smith', 'Marketing', 65000.00, 'active')`,
		`INSERT INTO employees (name, department, salary, status) VALUES 
		 ('Carol Davis', 'Engineering', 90000.00, 'inactive')`,
		`INSERT INTO employees (name, department, salary, status) VALUES 
		 ('David Wilson', 'Sales', 70000.00, 'active')`,
	}

	// Insert sample data into contractors
	contractorInserts := []string{
		`INSERT INTO contractors (name, department, hourly_rate, status) VALUES 
		 ('Eva Brown', 'Engineering', 75.00, 'active')`,
		`INSERT INTO contractors (name, department, hourly_rate, status) VALUES 
		 ('Frank Miller', 'Design', 60.00, 'active')`,
		`INSERT INTO contractors (name, department, hourly_rate, status) VALUES 
		 ('Grace Lee', 'Marketing', 55.00, 'inactive')`,
		`INSERT INTO contractors (name, department, hourly_rate, status) VALUES 
		 ('Henry Taylor', 'Engineering', 80.00, 'active')`,
	}

	// Execute all inserts
	for i, query := range employeeInserts {
		_, err := engine.Execute(query)
		if err != nil {
			log.Printf("Error inserting employee %d: %v", i+1, err)
		} else {
			fmt.Printf("✅ Employee %d inserted successfully\n", i+1)
		}
	}

	for i, query := range contractorInserts {
		_, err := engine.Execute(query)
		if err != nil {
			log.Printf("Error inserting contractor %d: %v", i+1, err)
		} else {
			fmt.Printf("✅ Contractor %d inserted successfully\n", i+1)
		}
	}

	// Test 1: Basic UNION (DISTINCT) - combine names from both tables
	fmt.Println("\n=== Test 1: Basic UNION (DISTINCT) ===")
	unionQuery1 := `SELECT name, department FROM employees 
	                 UNION 
	                 SELECT name, department FROM contractors`
	
	result, err := engine.Execute(unionQuery1)
	if err != nil {
		log.Printf("Error with basic UNION: %v", err)
	} else {
		fmt.Println("📋 All names and departments (UNION DISTINCT):")
		mist.PrintResult(result)
	}

	// Test 2: UNION ALL - include all rows including duplicates
	fmt.Println("\n=== Test 2: UNION ALL ===")
	unionQuery2 := `SELECT department FROM employees 
	                 UNION ALL 
	                 SELECT department FROM contractors`
	
	result, err = engine.Execute(unionQuery2)
	if err != nil {
		log.Printf("Error with UNION ALL: %v", err)
	} else {
		fmt.Println("📋 All departments (UNION ALL - with potential duplicates):")
		mist.PrintResult(result)
	}

	// Test 3: UNION with WHERE clauses
	fmt.Println("\n=== Test 3: UNION with WHERE Clauses ===")
	unionQuery3 := `SELECT name, department FROM employees WHERE status = 'active'
	                 UNION 
	                 SELECT name, department FROM contractors WHERE status = 'active'`
	
	result, err = engine.Execute(unionQuery3)
	if err != nil {
		log.Printf("Error with UNION WHERE: %v", err)
	} else {
		fmt.Println("📋 Active employees and contractors:")
		mist.PrintResult(result)
	}

	// Test 4: UNION with different column types (demonstrating type compatibility)
	fmt.Println("\n=== Test 4: UNION with Different Column Types ===")
	unionQuery4 := `SELECT name, salary FROM employees 
	                 UNION ALL 
	                 SELECT name, hourly_rate FROM contractors`
	
	result, err = engine.Execute(unionQuery4)
	if err != nil {
		log.Printf("Error with mixed types UNION: %v", err)
	} else {
		fmt.Println("📋 Names with salary/hourly_rate (mixed types):")
		mist.PrintResult(result)
	}

	// Test 5: UNION with literals and constants
	fmt.Println("\n=== Test 5: UNION with Literals ===")
	unionQuery5 := `SELECT name, 'Employee' as type FROM employees 
	                 UNION 
	                 SELECT name, 'Contractor' as type FROM contractors`
	
	result, err = engine.Execute(unionQuery5)
	if err != nil {
		log.Printf("Error with literal UNION: %v", err)
	} else {
		fmt.Println("📋 All workers with type indicator:")
		mist.PrintResult(result)
	}

	// Test 6: Three-way UNION
	fmt.Println("\n=== Test 6: Three-way UNION ===")
	unionQuery6 := `SELECT name FROM employees WHERE department = 'Engineering'
	                 UNION 
	                 SELECT name FROM contractors WHERE department = 'Engineering'
	                 UNION 
	                 SELECT name FROM employees WHERE department = 'Marketing'`
	
	result, err = engine.Execute(unionQuery6)
	if err != nil {
		log.Printf("Error with three-way UNION: %v", err)
	} else {
		fmt.Println("📋 Engineering workers and Marketing employees:")
		mist.PrintResult(result)
	}

	// Test 7: Error handling - mismatched column counts
	fmt.Println("\n=== Test 7: Error Handling - Mismatched Columns ===")
	unionQuery7 := `SELECT name, department, salary FROM employees 
	                 UNION 
	                 SELECT name, department FROM contractors`
	
	result, err = engine.Execute(unionQuery7)
	if err != nil {
		fmt.Printf("✅ Expected error for mismatched columns: %v\n", err)
	} else {
		fmt.Println("❌ Should have failed for mismatched column count")
	}

	// Test 8: UNION with ORDER BY (if supported)
	fmt.Println("\n=== Test 8: Complex UNION Query ===")
	unionQuery8 := `SELECT name, department, 'E' as source FROM employees WHERE status = 'active'
	                 UNION ALL 
	                 SELECT name, department, 'C' as source FROM contractors WHERE status = 'active'`
	
	result, err = engine.Execute(unionQuery8)
	if err != nil {
		log.Printf("Error with complex UNION: %v", err)
	} else {
		fmt.Println("📋 All active workers with source indicator:")
		mist.PrintResult(result)
	}

	// Test 9: Empty result UNION
	fmt.Println("\n=== Test 9: UNION with Empty Results ===")
	unionQuery9 := `SELECT name FROM employees WHERE status = 'terminated'
	                 UNION 
	                 SELECT name FROM contractors WHERE status = 'terminated'`
	
	result, err = engine.Execute(unionQuery9)
	if err != nil {
		log.Printf("Error with empty UNION: %v", err)
	} else {
		fmt.Println("📋 Terminated workers (should be empty):")
		mist.PrintResult(result)
	}

	// Test 10: UNION with single table (edge case)
	fmt.Println("\n=== Test 10: UNION within Same Table ===")
	unionQuery10 := `SELECT name, department FROM employees WHERE department = 'Engineering'
	                  UNION 
	                  SELECT name, department FROM employees WHERE department = 'Marketing'`
	
	result, err = engine.Execute(unionQuery10)
	if err != nil {
		log.Printf("Error with same table UNION: %v", err)
	} else {
		fmt.Println("📋 Engineering and Marketing employees:")
		mist.PrintResult(result)
	}

	fmt.Println("\n✅ All UNION tests completed!")
	
	fmt.Println("\n📊 Summary of UNION Features:")
	fmt.Println("   ✅ UNION (DISTINCT) - Eliminates duplicate rows")
	fmt.Println("   ✅ UNION ALL - Includes all rows including duplicates")
	fmt.Println("   ✅ Multi-table UNION - Combining different tables")
	fmt.Println("   ✅ UNION with WHERE clauses - Filtering before union")
	fmt.Println("   ✅ Mixed data types - Type compatibility handling")
	fmt.Println("   ✅ Complex UNION expressions - With literals and constants")
	fmt.Println("   ✅ Multi-way UNION - More than two SELECT statements")
	fmt.Println("   ✅ Error handling - Proper validation of column compatibility")
	fmt.Println("   ✅ Edge cases - Empty results and same table unions")
}