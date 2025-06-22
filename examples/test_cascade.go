//go:build ignore
// +build ignore

package main

import (
	"fmt"

	"github.com/abbychau/mist/mist"
)

func main() {
	engine := mist.NewSQLEngine()

	fmt.Println("=== Testing Foreign Key CASCADE Actions ===")

	// Create parent table
	fmt.Println("1. Creating parent table...")
	_, err := engine.Execute("CREATE TABLE categories (id INT PRIMARY KEY, name VARCHAR(100))")
	if err != nil {
		fmt.Printf("Error creating parent table: %v\n", err)
		return
	}

	// Create child table with CASCADE delete
	fmt.Println("2. Creating child table with CASCADE delete...")
	_, err = engine.Execute("CREATE TABLE products (id INT PRIMARY KEY, name VARCHAR(100), category_id INT, FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE)")
	if err != nil {
		fmt.Printf("Error creating child table: %v\n", err)
		return
	}

	// Insert test data
	fmt.Println("3. Inserting test data...")
	_, err = engine.Execute("INSERT INTO categories VALUES (1, 'Electronics'), (2, 'Books')")
	if err != nil {
		fmt.Printf("Error inserting categories: %v\n", err)
		return
	}

	_, err = engine.Execute("INSERT INTO products VALUES (1, 'Laptop', 1), (2, 'Phone', 1), (3, 'Novel', 2)")
	if err != nil {
		fmt.Printf("Error inserting products: %v\n", err)
		return
	}

	// Show initial data
	fmt.Println("4. Initial data:")
	result, err := engine.Execute("SELECT * FROM categories")
	if err != nil {
		fmt.Printf("Error querying categories: %v\n", err)
		return
	}
	fmt.Println("Categories:")
	mist.PrintResult(result)

	result, err = engine.Execute("SELECT * FROM products")
	if err != nil {
		fmt.Printf("Error querying products: %v\n", err)
		return
	}
	fmt.Println("Products:")
	mist.PrintResult(result)

	// Test CASCADE delete
	fmt.Println("5. Testing CASCADE delete (deleting Electronics category)...")
	_, err = engine.Execute("DELETE FROM categories WHERE id = 1")
	if err != nil {
		fmt.Printf("Error with CASCADE delete: %v\n", err)
		return
	}
	fmt.Println("✓ CASCADE delete succeeded")

	// Show final data
	fmt.Println("6. Final data after CASCADE delete:")
	result, err = engine.Execute("SELECT * FROM categories")
	if err != nil {
		fmt.Printf("Error querying categories: %v\n", err)
		return
	}
	fmt.Println("Categories:")
	mist.PrintResult(result)

	result, err = engine.Execute("SELECT * FROM products")
	if err != nil {
		fmt.Printf("Error querying products: %v\n", err)
		return
	}
	fmt.Println("Products (should only have Books products):")
	mist.PrintResult(result)

	fmt.Println("\n=== CASCADE Test Complete ===")
}
