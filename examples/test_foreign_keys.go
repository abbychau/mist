//go:build ignore
// +build ignore

package main

import (
	"fmt"

	"github.com/abbychau/mist/mist"
)

func main() {
	engine := mist.NewSQLEngine()

	fmt.Println("=== Testing Foreign Key Constraints ===")

	// Create parent table
	fmt.Println("1. Creating parent table...")
	_, err := engine.Execute("CREATE TABLE categories (id INT PRIMARY KEY, name VARCHAR(100))")
	if err != nil {
		fmt.Printf("Error creating parent table: %v\n", err)
		return
	}

	// Create child table with foreign key
	fmt.Println("2. Creating child table with foreign key...")
	_, err = engine.Execute("CREATE TABLE products (id INT PRIMARY KEY, name VARCHAR(100), category_id INT, FOREIGN KEY (category_id) REFERENCES categories(id))")
	if err != nil {
		fmt.Printf("Error creating child table: %v\n", err)
		return
	}

	// Insert data into parent table
	fmt.Println("3. Inserting data into parent table...")
	_, err = engine.Execute("INSERT INTO categories VALUES (1, 'Electronics'), (2, 'Books')")
	if err != nil {
		fmt.Printf("Error inserting into parent table: %v\n", err)
		return
	}

	// Test valid foreign key insert
	fmt.Println("4. Testing valid foreign key insert...")
	_, err = engine.Execute("INSERT INTO products VALUES (1, 'Laptop', 1)")
	if err != nil {
		fmt.Printf("Error with valid foreign key insert: %v\n", err)
		return
	}
	fmt.Println("✓ Valid foreign key insert succeeded")

	// Test invalid foreign key insert (should fail)
	fmt.Println("5. Testing invalid foreign key insert (should fail)...")
	_, err = engine.Execute("INSERT INTO products VALUES (2, 'Phone', 99)")
	if err != nil {
		fmt.Printf("✓ Invalid foreign key insert correctly failed: %v\n", err)
	} else {
		fmt.Printf("✗ Invalid foreign key insert should have failed but didn't\n")
	}

	// Test NULL foreign key insert (should be allowed)
	fmt.Println("6. Testing NULL foreign key insert (should be allowed)...")
	_, err = engine.Execute("INSERT INTO products VALUES (3, 'Generic Product', NULL)")
	if err != nil {
		fmt.Printf("Error with NULL foreign key insert: %v\n", err)
		return
	}
	fmt.Printf("✓ NULL foreign key insert succeeded\n")

	// Test delete with foreign key constraint (should fail)
	fmt.Println("7. Testing delete with foreign key constraint (should fail)...")
	_, err = engine.Execute("DELETE FROM categories WHERE id = 1")
	if err != nil {
		fmt.Printf("✓ Delete correctly failed due to foreign key constraint: %v\n", err)
	} else {
		fmt.Printf("✗ Delete should have failed due to foreign key constraint but didn't\n")
	}

	// Show final data
	fmt.Println("8. Final data verification...")
	result, err := engine.Execute("SELECT * FROM products")
	if err != nil {
		fmt.Printf("Error querying products: %v\n", err)
		return
	}
	fmt.Println("Products:")
	mist.PrintResult(result)

	fmt.Println("\n=== Foreign Key Test Complete ===")
}
