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

	fmt.Println("=== Testing Enhanced Mist Features ===")
	fmt.Println()

	// Test 1: DATE type
	fmt.Println("1. Testing DATE type...")
	_, err := engine.Execute(`CREATE TABLE events (
		id INT AUTO_INCREMENT PRIMARY KEY,
		event_name VARCHAR(100) NOT NULL,
		event_date DATE NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		log.Fatalf("Error creating events table: %v", err)
	}

	_, err = engine.Execute("INSERT INTO events (event_name, event_date) VALUES ('Birthday Party', '2024-06-25')")
	if err != nil {
		log.Fatalf("Error inserting event: %v", err)
	}

	result, _ := engine.Execute("SELECT * FROM events")
	fmt.Println("Events table:")
	mist.PrintResult(result)
	fmt.Println()

	// Test 2: UNIQUE constraint
	fmt.Println("2. Testing UNIQUE constraints...")
	_, err = engine.Execute(`CREATE TABLE users_unique (
		id INT AUTO_INCREMENT PRIMARY KEY,
		email VARCHAR(100) NOT NULL UNIQUE,
		username VARCHAR(50) UNIQUE,
		name VARCHAR(100) NOT NULL
	)`)
	if err != nil {
		log.Fatalf("Error creating users_unique table: %v", err)
	}

	// Insert valid data
	_, err = engine.Execute("INSERT INTO users_unique (email, username, name) VALUES ('john@example.com', 'john123', 'John Doe')")
	if err != nil {
		log.Fatalf("Error inserting first user: %v", err)
	}

	// Try to insert duplicate email (should fail)
	_, err = engine.Execute("INSERT INTO users_unique (email, username, name) VALUES ('john@example.com', 'john456', 'Jane Doe')")
	if err != nil {
		fmt.Printf("✓ UNIQUE constraint working - duplicate email rejected: %v\n", err)
	} else {
		fmt.Println("✗ UNIQUE constraint failed - duplicate email was accepted")
	}

	result, _ = engine.Execute("SELECT * FROM users_unique")
	fmt.Println("Users table:")
	mist.PrintResult(result)
	fmt.Println()

	// Test 3: ENUM type
	fmt.Println("3. Testing ENUM type...")
	_, err = engine.Execute(`CREATE TABLE orders (
		id INT AUTO_INCREMENT PRIMARY KEY,
		product_name VARCHAR(100) NOT NULL,
		status ENUM('pending', 'processing', 'shipped', 'delivered') NOT NULL DEFAULT 'pending',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		log.Fatalf("Error creating orders table: %v", err)
	}

	// Insert valid enum values
	_, err = engine.Execute("INSERT INTO orders (product_name, status) VALUES ('Laptop', 'processing')")
	if err != nil {
		log.Fatalf("Error inserting order: %v", err)
	}

	// Insert with default enum value
	_, err = engine.Execute("INSERT INTO orders (product_name) VALUES ('Mouse')")
	if err != nil {
		log.Fatalf("Error inserting order with default enum: %v", err)
	}

	// Try to insert invalid enum value (should fail)
	_, err = engine.Execute("INSERT INTO orders (product_name, status) VALUES ('Keyboard', 'invalid_status')")
	if err != nil {
		fmt.Printf("✓ ENUM constraint working - invalid value rejected: %v\n", err)
	} else {
		fmt.Println("✗ ENUM constraint failed - invalid value was accepted")
	}

	result, _ = engine.Execute("SELECT * FROM orders")
	fmt.Println("Orders table:")
	mist.PrintResult(result)
	fmt.Println()

	// Test 4: ON UPDATE CURRENT_TIMESTAMP
	fmt.Println("4. Testing ON UPDATE CURRENT_TIMESTAMP...")
	_, err = engine.Execute(`CREATE TABLE products (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		price DECIMAL(10,2) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	)`)
	if err != nil {
		log.Fatalf("Error creating products table: %v", err)
	}

	// Insert a product
	_, err = engine.Execute("INSERT INTO products (name, price) VALUES ('Test Product', 19.99)")
	if err != nil {
		log.Fatalf("Error inserting product: %v", err)
	}

	fmt.Println("Product after insert:")
	result, _ = engine.Execute("SELECT * FROM products")
	mist.PrintResult(result)

	// Wait a moment and update the product
	fmt.Println("Updating product price...")
	_, err = engine.Execute("UPDATE products SET price = 24.99 WHERE id = 1")
	if err != nil {
		log.Fatalf("Error updating product: %v", err)
	}

	fmt.Println("Product after update (updated_at should be newer):")
	result, _ = engine.Execute("SELECT * FROM products")
	mist.PrintResult(result)
	fmt.Println()

	// Test 5: Combined features - realistic table
	fmt.Println("5. Testing combined features with realistic table...")
	_, err = engine.Execute(`CREATE TABLE user_profiles (
		id INT AUTO_INCREMENT PRIMARY KEY,
		email VARCHAR(255) NOT NULL UNIQUE,
		username VARCHAR(50) NOT NULL UNIQUE,
		full_name VARCHAR(100) NOT NULL,
		birth_date DATE,
		status ENUM('active', 'inactive', 'suspended') NOT NULL DEFAULT 'active',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	)`)
	if err != nil {
		log.Fatalf("Error creating user_profiles table: %v", err)
	}

	// Insert test data
	_, err = engine.Execute(`INSERT INTO user_profiles (email, username, full_name, birth_date, status) 
		VALUES ('alice@example.com', 'alice_smith', 'Alice Smith', '1990-05-15', 'active')`)
	if err != nil {
		log.Fatalf("Error inserting user profile: %v", err)
	}

	result, _ = engine.Execute("SELECT * FROM user_profiles")
	fmt.Println("User profiles table:")
	mist.PrintResult(result)

	fmt.Println("\n=== All tests completed successfully! ===")
}
