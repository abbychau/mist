//go:build ignore
// +build ignore

package main

import (
	"fmt"

	"github.com/abbychau/mist"
)

func main() {
	engine := mist.NewSQLEngine()

	fmt.Println("=== Testing Original Schema with Foreign Key Enforcement ===")

	// Import original schema
	fmt.Println("1. Importing original schema...")
	results, err := engine.ImportSQLFile("test_data_real/001_create_tables.sql")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("✓ Successfully executed %d statements\n", len(results))

	// Import original data (should work since they respect foreign keys)
	fmt.Println("2. Importing original data...")
	results, err = engine.ImportSQLFile("test_data_real/002_insert_sample_data.sql")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("✓ Successfully executed %d statements\n", len(results))

	// Test inserting invalid foreign key data (should fail)
	fmt.Println("3. Testing invalid foreign key insert (should fail)...")
	_, err = engine.Execute("INSERT INTO users (company_id, full_name, email, password) VALUES (999, 'Test User', 'test@example.com', 'password')")
	if err != nil {
		fmt.Printf("✓ Invalid foreign key insert correctly failed: %v\n", err)
	} else {
		fmt.Printf("✗ Invalid foreign key insert should have failed\n")
	}

	// Test inserting invalid business partner (should fail)
	fmt.Println("4. Testing invalid business partner insert (should fail)...")
	_, err = engine.Execute("INSERT INTO business_partners (company_id, corporate_name, representative, phone_number, postal_code, address) VALUES (999, 'Test Corp', 'Test Rep', '123-456-7890', '12345', 'Test Address')")
	if err != nil {
		fmt.Printf("✓ Invalid business partner insert correctly failed: %v\n", err)
	} else {
		fmt.Printf("✗ Invalid business partner insert should have failed\n")
	}

	// Test deleting company with dependent records (should fail)
	fmt.Println("5. Testing delete of company with dependent records (should fail)...")
	_, err = engine.Execute("DELETE FROM companies WHERE id = 1")
	if err != nil {
		fmt.Printf("✓ Delete correctly failed due to foreign key constraint: %v\n", err)
	} else {
		fmt.Printf("✗ Delete should have failed due to foreign key constraint\n")
	}
	// Show some data to verify integrity
	fmt.Println("6. Verifying data integrity...")
	result, err := engine.Execute("SELECT corporate_name FROM companies")
	if err != nil {
		fmt.Printf("Error querying data: %v\n", err)
		return
	}
	fmt.Println("Companies:")
	mist.PrintResult(result)

	fmt.Println("\n=== Original Schema Foreign Key Test Complete ===")
}
