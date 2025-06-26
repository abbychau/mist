//go:build ignore
// +build ignore

package main

import (
	"fmt"

	"github.com/abbychau/mist"
)

func main() {
	// Create a new SQL engine
	engine := mist.NewSQLEngine()

	fmt.Println("=== Testing Original Schema Files ===")

	// Test the original CREATE TABLE statements
	fmt.Println("1. Importing original schema file...")
	results, err := engine.ImportSQLFile("test_data_real/001_create_tables.sql")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("✓ Successfully executed %d statements\n", len(results))
	}

	// Test the original data file
	fmt.Println("\n2. Importing original data file...")
	results, err = engine.ImportSQLFile("test_data_real/002_insert_sample_data.sql")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("✓ Successfully executed %d statements\n", len(results))
	}

	// Check the data
	fmt.Println("\n3. Checking imported data...")

	// Companies
	result, err := engine.Execute("SELECT COUNT(*) as company_count FROM companies")
	if err != nil {
		fmt.Printf("Error querying companies: %v\n", err)
	} else {
		fmt.Println("Companies:")
		mist.PrintResult(result)
	}

	// Users
	result, err = engine.Execute("SELECT COUNT(*) as user_count FROM users")
	if err != nil {
		fmt.Printf("Error querying users: %v\n", err)
	} else {
		fmt.Println("Users:")
		mist.PrintResult(result)
	}

	// Invoices with ENUM status
	result, err = engine.Execute("SELECT status, COUNT(*) as count FROM invoices GROUP BY status")
	if err != nil {
		fmt.Printf("Error querying invoice status: %v\n", err)
	} else {
		fmt.Println("Invoice status distribution:")
		mist.PrintResult(result)
	}

	// Test DATE operations
	result, err = engine.Execute("SELECT issue_date, payment_due_date FROM invoices LIMIT 3")
	if err != nil {
		fmt.Printf("Error querying dates: %v\n", err)
	} else {
		fmt.Println("Sample invoice dates:")
		mist.PrintResult(result)
	}

	fmt.Println("\n=== Original Schema Test Complete ===")
}
