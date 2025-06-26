//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"log"

	"github.com/abbychau/mist"
)

func main() {
	// Create a new SQL engine
	engine := mist.NewSQLEngine()

	fmt.Println("=== Debug Default Values ===")

	// Create a simple table with default CURRENT_TIMESTAMP
	_, err := engine.Execute(`CREATE TABLE debug_table (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		log.Fatalf("Error creating table: %v", err)
	}

	// Get the table to inspect its columns
	db := engine.GetDatabase() // Assuming this method exists, or we'll need to add it
	if db != nil {
		table, err := db.GetTable("debug_table")
		if err == nil {
			fmt.Println("Table columns:")
			for i, col := range table.Columns {
				fmt.Printf("Column %d: Name=%s, Type=%s, Default=%v\n", i, col.Name, col.Type, col.Default)
			}
		}
	}

	// Try inserting without specifying created_at
	_, err = engine.Execute("INSERT INTO debug_table (name) VALUES ('Test')")
	if err != nil {
		log.Fatalf("Error inserting: %v", err)
	}

	result, _ := engine.Execute("SELECT * FROM debug_table")
	fmt.Println("Result:")
	mist.PrintResult(result)
}
