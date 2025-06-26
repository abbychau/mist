//go:build ignore
// +build ignore

package main

import (
	"fmt"

	"github.com/abbychau/mist"
)

func main() {
	engine := mist.NewSQLEngine()

	fmt.Println("=== Testing Foreign Key Action Error Conditions ===")

	// Test SET NULL on NOT NULL column (should fail)
	fmt.Println("1. Testing SET NULL on NOT NULL column (should fail)...")
	_, err := engine.Execute("CREATE TABLE parent1 (id INT PRIMARY KEY, name VARCHAR(100))")
	if err != nil {
		fmt.Printf("Error creating parent1: %v\n", err)
		return
	}

	_, err = engine.Execute("CREATE TABLE child1 (id INT PRIMARY KEY, parent_id INT NOT NULL, FOREIGN KEY (parent_id) REFERENCES parent1(id) ON DELETE SET NULL)")
	if err != nil {
		fmt.Printf("Error creating child1: %v\n", err)
		return
	}

	_, err = engine.Execute("INSERT INTO parent1 VALUES (1, 'Parent')")
	if err != nil {
		fmt.Printf("Error inserting parent1: %v\n", err)
		return
	}

	_, err = engine.Execute("INSERT INTO child1 VALUES (1, 1)")
	if err != nil {
		fmt.Printf("Error inserting child1: %v\n", err)
		return
	}

	_, err = engine.Execute("DELETE FROM parent1 WHERE id = 1")
	if err != nil {
		fmt.Printf("✓ SET NULL on NOT NULL column correctly failed: %v\n", err)
	} else {
		fmt.Printf("✗ SET NULL on NOT NULL column should have failed\n")
	}

	// Test SET DEFAULT on column with no default (should fail)
	fmt.Println("2. Testing SET DEFAULT on column with no default (should fail)...")
	engine2 := mist.NewSQLEngine() // Fresh engine

	_, err = engine2.Execute("CREATE TABLE parent2 (id INT PRIMARY KEY, name VARCHAR(100))")
	if err != nil {
		fmt.Printf("Error creating parent2: %v\n", err)
		return
	}

	_, err = engine2.Execute("CREATE TABLE child2 (id INT PRIMARY KEY, parent_id INT, FOREIGN KEY (parent_id) REFERENCES parent2(id) ON DELETE SET DEFAULT)")
	if err != nil {
		fmt.Printf("Error creating child2: %v\n", err)
		return
	}

	_, err = engine2.Execute("INSERT INTO parent2 VALUES (1, 'Parent')")
	if err != nil {
		fmt.Printf("Error inserting parent2: %v\n", err)
		return
	}

	_, err = engine2.Execute("INSERT INTO child2 VALUES (1, 1)")
	if err != nil {
		fmt.Printf("Error inserting child2: %v\n", err)
		return
	}

	_, err = engine2.Execute("DELETE FROM parent2 WHERE id = 1")
	if err != nil {
		fmt.Printf("✓ SET DEFAULT on column with no default correctly failed: %v\n", err)
	} else {
		fmt.Printf("✗ SET DEFAULT on column with no default should have failed\n")
	}

	// Test CASCADE with circular reference (should be handled)
	fmt.Println("3. Testing CASCADE with potential circular reference...")
	engine3 := mist.NewSQLEngine() // Fresh engine

	_, err = engine3.Execute("CREATE TABLE departments (id INT PRIMARY KEY, name VARCHAR(100), manager_id INT)")
	if err != nil {
		fmt.Printf("Error creating departments: %v\n", err)
		return
	}

	_, err = engine3.Execute("CREATE TABLE employees (id INT PRIMARY KEY, name VARCHAR(100), dept_id INT, FOREIGN KEY (dept_id) REFERENCES departments(id) ON DELETE CASCADE)")
	if err != nil {
		fmt.Printf("Error creating employees: %v\n", err)
		return
	}

	_, err = engine3.Execute("INSERT INTO departments VALUES (1, 'Engineering', NULL)")
	if err != nil {
		fmt.Printf("Error inserting department: %v\n", err)
		return
	}

	_, err = engine3.Execute("INSERT INTO employees VALUES (1, 'Alice', 1), (2, 'Bob', 1)")
	if err != nil {
		fmt.Printf("Error inserting employees: %v\n", err)
		return
	}

	_, err = engine3.Execute("DELETE FROM departments WHERE id = 1")
	if err != nil {
		fmt.Printf("Error with CASCADE delete: %v\n", err)
		return
	}
	fmt.Printf("✓ CASCADE delete with multiple dependent rows succeeded\n")

	result, err := engine3.Execute("SELECT COUNT(*) FROM employees")
	if err != nil {
		fmt.Printf("Error counting employees: %v\n", err)
		return
	}
	fmt.Println("Employees remaining after CASCADE delete:")
	mist.PrintResult(result)

	fmt.Println("\n=== Error Condition Tests Complete ===")
}
