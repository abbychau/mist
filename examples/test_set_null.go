//go:build ignore
// +build ignore

package main

import (
	"fmt"

	"github.com/abbychau/mist"
)

func main() {
	engine := mist.NewSQLEngine()

	fmt.Println("=== Testing Foreign Key SET NULL Actions ===")

	// Create parent table
	fmt.Println("1. Creating parent table...")
	_, err := engine.Execute("CREATE TABLE departments (id INT PRIMARY KEY, name VARCHAR(100))")
	if err != nil {
		fmt.Printf("Error creating parent table: %v\n", err)
		return
	}

	// Create child table with SET NULL delete (note: category_id is nullable)
	fmt.Println("2. Creating child table with SET NULL delete...")
	_, err = engine.Execute("CREATE TABLE employees (id INT PRIMARY KEY, name VARCHAR(100), department_id INT, FOREIGN KEY (department_id) REFERENCES departments(id) ON DELETE SET NULL)")
	if err != nil {
		fmt.Printf("Error creating child table: %v\n", err)
		return
	}

	// Insert test data
	fmt.Println("3. Inserting test data...")
	_, err = engine.Execute("INSERT INTO departments VALUES (1, 'Engineering'), (2, 'Marketing')")
	if err != nil {
		fmt.Printf("Error inserting departments: %v\n", err)
		return
	}

	_, err = engine.Execute("INSERT INTO employees VALUES (1, 'Alice', 1), (2, 'Bob', 1), (3, 'Charlie', 2)")
	if err != nil {
		fmt.Printf("Error inserting employees: %v\n", err)
		return
	}

	// Show initial data
	fmt.Println("4. Initial data:")
	result, err := engine.Execute("SELECT * FROM departments")
	if err != nil {
		fmt.Printf("Error querying departments: %v\n", err)
		return
	}
	fmt.Println("Departments:")
	mist.PrintResult(result)

	result, err = engine.Execute("SELECT * FROM employees")
	if err != nil {
		fmt.Printf("Error querying employees: %v\n", err)
		return
	}
	fmt.Println("Employees:")
	mist.PrintResult(result)

	// Test SET NULL delete
	fmt.Println("5. Testing SET NULL delete (deleting Engineering department)...")
	_, err = engine.Execute("DELETE FROM departments WHERE id = 1")
	if err != nil {
		fmt.Printf("Error with SET NULL delete: %v\n", err)
		return
	}
	fmt.Println("✓ SET NULL delete succeeded")

	// Show final data
	fmt.Println("6. Final data after SET NULL delete:")
	result, err = engine.Execute("SELECT * FROM departments")
	if err != nil {
		fmt.Printf("Error querying departments: %v\n", err)
		return
	}
	fmt.Println("Departments:")
	mist.PrintResult(result)

	result, err = engine.Execute("SELECT * FROM employees")
	if err != nil {
		fmt.Printf("Error querying employees: %v\n", err)
		return
	}
	fmt.Println("Employees (Engineering employees should have NULL department_id):")
	mist.PrintResult(result)

	fmt.Println("\n=== SET NULL Test Complete ===")
}
