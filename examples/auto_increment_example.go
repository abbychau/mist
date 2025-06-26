// Auto Increment Example
// This example demonstrates how to use AUTO_INCREMENT columns in Mist
// Run with: go run auto_increment_example.go

//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"log"

	"github.com/abbychau/mist"
)

func main() {
	fmt.Println("=== Mist Auto Increment Example ===")

	// Create a new SQL engine
	engine := mist.NewSQLEngine()

	// Create a table with auto increment primary key
	fmt.Println("\n1. Creating table with AUTO_INCREMENT primary key...")
	_, err := engine.Execute(`CREATE TABLE products (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		price FLOAT NOT NULL,
		category VARCHAR(50)
	)`)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}
	fmt.Println("✓ Table 'products' created successfully")

	// Insert data without specifying ID
	fmt.Println("\n2. Inserting products without specifying ID...")
	insertQueries := []string{
		"INSERT INTO products (name, price, category) VALUES ('Gaming Laptop', 1299.99, 'Electronics')",
		"INSERT INTO products (name, price, category) VALUES ('Wireless Mouse', 29.99, 'Electronics')",
		"INSERT INTO products (name, price, category) VALUES ('Mechanical Keyboard', 149.99, 'Electronics')",
		"INSERT INTO products (name, price, category) VALUES ('Office Chair', 299.99, 'Furniture')",
		"INSERT INTO products (name, price, category) VALUES ('Standing Desk', 599.99, 'Furniture')",
	}

	for i, query := range insertQueries {
		result, err := engine.Execute(query)
		if err != nil {
			log.Fatalf("Failed to insert product %d: %v", i+1, err)
		}
		fmt.Printf("✓ Inserted product with auto-generated ID: %v\n", result)
	}

	// Query all products to show auto-generated IDs
	fmt.Println("\n3. Querying all products...")
	result, err := engine.Execute("SELECT * FROM products ORDER BY id")
	if err != nil {
		log.Fatal("Failed to query products:", err)
	}

	fmt.Println("\nProducts with auto-generated IDs:")
	mist.PrintResult(result)

	// Demonstrate inserting with mixed approaches
	fmt.Println("\n4. Mixed insert approaches...")

	// Insert with explicit ID (should work if ID doesn't conflict)
	_, err = engine.Execute("INSERT INTO products (id, name, price, category) VALUES (10, 'Premium Monitor', 799.99, 'Electronics')")
	if err != nil {
		log.Printf("Explicit ID insert failed (expected if ID conflicts): %v", err)
	} else {
		fmt.Println("✓ Inserted product with explicit ID: 10")
	}

	// Continue with auto increment after explicit ID
	result, err = engine.Execute("INSERT INTO products (name, price, category) VALUES ('Coffee Mug', 12.99, 'Kitchen')")
	if err != nil {
		log.Fatal("Failed to insert after explicit ID:", err)
	}
	fmt.Printf("✓ Auto increment continued with ID: %v\n", result)

	// Final query to show all products
	fmt.Println("\n5. Final product listing...")
	result, err = engine.Execute("SELECT id, name, price, category FROM products ORDER BY id")
	if err != nil {
		log.Fatal("Failed to query final products:", err)
	}

	fmt.Println("\nFinal product catalog:")
	mist.PrintResult(result)

	// Show some statistics
	fmt.Println("\n6. Product statistics...")
	queries := map[string]string{
		"Total Products":    "SELECT COUNT(*) as total_products FROM products",
		"Average Price":     "SELECT AVG(price) as avg_price FROM products",
		"Most Expensive":    "SELECT name, price FROM products WHERE price = (SELECT MAX(price) FROM products)",
		"Electronics Count": "SELECT COUNT(*) as electronics_count FROM products WHERE category = 'Electronics'",
	}

	for description, query := range queries {
		result, err := engine.Execute(query)
		if err != nil {
			fmt.Printf("Error executing %s: %v\n", description, err)
			continue
		}
		fmt.Printf("\n%s:\n", description)
		mist.PrintResult(result)
	}

	fmt.Println("\n=== Auto Increment Demo Complete ===")
}
