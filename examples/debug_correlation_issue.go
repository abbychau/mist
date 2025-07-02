package main

import (
	"fmt"
	"log"

	"github.com/abbychau/mist"
)

func main() {
	// Create a new engine
	engine := mist.NewSQLEngine()

	// Create simple test tables
	_, err := engine.Execute("CREATE TABLE users (id INT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(100))")
	if err != nil {
		log.Fatal(err)
	}

	_, err = engine.Execute("CREATE TABLE orders (user_id INT, amount DECIMAL(10,2))")
	if err != nil {
		log.Fatal(err)
	}

	// Insert specific test data
	_, err = engine.Execute("INSERT INTO users (name) VALUES ('Alice')")  // ID will be 1
	if err != nil {
		log.Fatal(err)
	}
	_, err = engine.Execute("INSERT INTO users (name) VALUES ('Bob')")    // ID will be 2
	if err != nil {
		log.Fatal(err)
	}
	_, err = engine.Execute("INSERT INTO orders (user_id, amount) VALUES (1, 100.00)")  // Only Alice has an order
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Debug Correlation Issue ===")

	// Verify the basic data
	fmt.Println("\n--- Data verification ---")
	result, err := engine.Execute("SELECT id, name FROM users")
	if err != nil {
		log.Fatal(err)
	}
	selectResult := result.(*mist.SelectResult)
	fmt.Printf("Users:\n")
	for _, row := range selectResult.Rows {
		fmt.Printf("  ID: %v, Name: %v\n", row[0], row[1])
	}

	result, err = engine.Execute("SELECT user_id, amount FROM orders")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Orders:\n")
	for _, row := range selectResult.Rows {
		fmt.Printf("  UserID: %v, Amount: %v\n", row[0], row[1])
	}

	// Test scalar subquery correlation first
	fmt.Println("\n--- Scalar subquery correlation test ---")
	result, err = engine.Execute("SELECT u.name, (SELECT COUNT(*) FROM orders WHERE user_id = u.id) AS order_count FROM users u")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Order counts by user:\n")
	for _, row := range selectResult.Rows {
		fmt.Printf("  %v: %v orders\n", row[0], row[1])
	}

	// Expected: Alice: 1 orders, Bob: 0 orders
	// If this shows both users having the same count, correlation is broken

	// Test EXISTS correlation  
	fmt.Println("\n--- EXISTS correlation test ---")
	result, err = engine.Execute("SELECT u.name FROM users u WHERE EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id)")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Users with orders:\n")
	for _, row := range selectResult.Rows {
		fmt.Printf("  %v\n", row[0])
	}
	fmt.Printf("Expected: Alice only. Got %d users.\n", len(selectResult.Rows))

	// Test manual verification
	fmt.Println("\n--- Manual verification ---")
	result, err = engine.Execute("SELECT COUNT(*) FROM orders WHERE user_id = 1")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Orders for user_id=1: %v\n", selectResult.Rows[0][0])

	result, err = engine.Execute("SELECT COUNT(*) FROM orders WHERE user_id = 2")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Orders for user_id=2: %v\n", selectResult.Rows[0][0])
}