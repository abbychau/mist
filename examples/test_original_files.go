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

	fmt.Println("=== Testing Original (Non-Compatible) SQL Files ===")
	fmt.Println()

	// Test the original CREATE TABLE statements with enhanced features
	fmt.Println("1. Testing original schema file...")
	results, err := engine.ImportSQLFile("test_data_real/001_create_tables.sql")
	if err != nil {
		fmt.Printf("Error (expected if some features not supported): %v\n", err)
		fmt.Println("Continuing...")
	} else {
		fmt.Printf("✓ Successfully executed %d statements from original schema file\n", len(results))
	}

	// Try a simpler test with just one table creation that should now work
	fmt.Println("\n2. Testing individual features that should now work...")

	// Test ENUM
	_, err = engine.Execute(`CREATE TABLE test_enum (
		id INT AUTO_INCREMENT PRIMARY KEY,
		status ENUM('active', 'inactive', 'pending') NOT NULL DEFAULT 'pending'
	)`)
	if err != nil {
		fmt.Printf("✗ ENUM test failed: %v\n", err)
	} else {
		fmt.Println("✓ ENUM type creation successful")

		_, err = engine.Execute("INSERT INTO test_enum (status) VALUES ('active')")
		if err != nil {
			fmt.Printf("✗ ENUM insert failed: %v\n", err)
		} else {
			fmt.Println("✓ ENUM insert successful")
		}
	}

	// Test DATE
	_, err = engine.Execute(`CREATE TABLE test_date (
		id INT AUTO_INCREMENT PRIMARY KEY,
		event_date DATE NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		fmt.Printf("✗ DATE test failed: %v\n", err)
	} else {
		fmt.Println("✓ DATE type creation successful")

		_, err = engine.Execute("INSERT INTO test_date (event_date) VALUES ('2024-12-25')")
		if err != nil {
			fmt.Printf("✗ DATE insert failed: %v\n", err)
		} else {
			fmt.Println("✓ DATE insert successful")
		}
	}

	// Test UNIQUE
	_, err = engine.Execute(`CREATE TABLE test_unique (
		id INT AUTO_INCREMENT PRIMARY KEY,
		email VARCHAR(255) NOT NULL UNIQUE,
		username VARCHAR(50) UNIQUE
	)`)
	if err != nil {
		fmt.Printf("✗ UNIQUE test failed: %v\n", err)
	} else {
		fmt.Println("✓ UNIQUE constraint creation successful")

		_, err = engine.Execute("INSERT INTO test_unique (email, username) VALUES ('test@example.com', 'testuser')")
		if err != nil {
			fmt.Printf("✗ UNIQUE insert failed: %v\n", err)
		} else {
			fmt.Println("✓ UNIQUE insert successful")

			// Try duplicate
			_, err = engine.Execute("INSERT INTO test_unique (email, username) VALUES ('test@example.com', 'testuser2')")
			if err != nil {
				fmt.Println("✓ UNIQUE constraint working - duplicate rejected")
			} else {
				fmt.Println("✗ UNIQUE constraint failed - duplicate accepted")
			}
		}
	}

	// Test ON UPDATE CURRENT_TIMESTAMP
	_, err = engine.Execute(`CREATE TABLE test_on_update (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	)`)
	if err != nil {
		fmt.Printf("✗ ON UPDATE test failed: %v\n", err)
	} else {
		fmt.Println("✓ ON UPDATE CURRENT_TIMESTAMP creation successful")

		_, err = engine.Execute("INSERT INTO test_on_update (name) VALUES ('Test Item')")
		if err != nil {
			fmt.Printf("✗ ON UPDATE insert failed: %v\n", err)
		} else {
			fmt.Println("✓ ON UPDATE insert successful")

			// Test update
			_, err = engine.Execute("UPDATE test_on_update SET name = 'Updated Item' WHERE id = 1")
			if err != nil {
				fmt.Printf("✗ ON UPDATE update failed: %v\n", err)
			} else {
				fmt.Println("✓ ON UPDATE update successful")
			}
		}
	}

	fmt.Println("\n=== Enhanced Mist Features Test Complete ===")
}
