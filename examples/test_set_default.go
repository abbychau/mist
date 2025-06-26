//go:build ignore
// +build ignore

package main

import (
	"fmt"

	"github.com/abbychau/mist"
)

func main() {
	engine := mist.NewSQLEngine()

	fmt.Println("=== Testing Foreign Key SET DEFAULT Actions ===")

	// Create parent table
	fmt.Println("1. Creating parent table...")
	_, err := engine.Execute("CREATE TABLE status_types (id INT PRIMARY KEY, name VARCHAR(100))")
	if err != nil {
		fmt.Printf("Error creating parent table: %v\n", err)
		return
	}

	// Create child table with SET DEFAULT delete and a default value
	fmt.Println("2. Creating child table with SET DEFAULT delete...")
	_, err = engine.Execute("CREATE TABLE tasks (id INT PRIMARY KEY, name VARCHAR(100), status_id INT DEFAULT 1, FOREIGN KEY (status_id) REFERENCES status_types(id) ON DELETE SET DEFAULT)")
	if err != nil {
		fmt.Printf("Error creating child table: %v\n", err)
		return
	}

	// Insert test data
	fmt.Println("3. Inserting test data...")
	_, err = engine.Execute("INSERT INTO status_types VALUES (1, 'Pending'), (2, 'In Progress'), (3, 'Completed')")
	if err != nil {
		fmt.Printf("Error inserting status types: %v\n", err)
		return
	}

	_, err = engine.Execute("INSERT INTO tasks VALUES (1, 'Task A', 2), (2, 'Task B', 3), (3, 'Task C', 2)")
	if err != nil {
		fmt.Printf("Error inserting tasks: %v\n", err)
		return
	}

	// Show initial data
	fmt.Println("4. Initial data:")
	result, err := engine.Execute("SELECT * FROM status_types")
	if err != nil {
		fmt.Printf("Error querying status types: %v\n", err)
		return
	}
	fmt.Println("Status Types:")
	mist.PrintResult(result)

	result, err = engine.Execute("SELECT * FROM tasks")
	if err != nil {
		fmt.Printf("Error querying tasks: %v\n", err)
		return
	}
	fmt.Println("Tasks:")
	mist.PrintResult(result)

	// Test SET DEFAULT delete
	fmt.Println("5. Testing SET DEFAULT delete (deleting 'In Progress' status)...")
	_, err = engine.Execute("DELETE FROM status_types WHERE id = 2")
	if err != nil {
		fmt.Printf("Error with SET DEFAULT delete: %v\n", err)
		return
	}
	fmt.Println("✓ SET DEFAULT delete succeeded")

	// Show final data
	fmt.Println("6. Final data after SET DEFAULT delete:")
	result, err = engine.Execute("SELECT * FROM status_types")
	if err != nil {
		fmt.Printf("Error querying status types: %v\n", err)
		return
	}
	fmt.Println("Status Types:")
	mist.PrintResult(result)

	result, err = engine.Execute("SELECT * FROM tasks")
	if err != nil {
		fmt.Printf("Error querying tasks: %v\n", err)
		return
	}
	fmt.Println("Tasks ('In Progress' tasks should now have status_id = 1):")
	mist.PrintResult(result)

	fmt.Println("\n=== SET DEFAULT Test Complete ===")
}
