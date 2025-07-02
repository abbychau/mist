package main

import (
	"fmt"
	"log"

	"github.com/abbychau/mist"
)

func main() {
	// Create a new engine
	engine := mist.NewSQLEngine()

	// Create AUTO_INCREMENT tables like the failing case
	_, err := engine.Execute("CREATE TABLE users (id INT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(100))")
	if err != nil {
		log.Fatal(err)
	}

	_, err = engine.Execute("CREATE TABLE orders (user_id INT, amount DECIMAL(10,2))")
	if err != nil {
		log.Fatal(err)
	}

	// Insert specific data: User 1 has orders, User 2 doesn't
	_, err = engine.Execute("INSERT INTO users (name) VALUES ('Alice')")  // Will get ID 1
	if err != nil {
		log.Fatal(err)
	}
	_, err = engine.Execute("INSERT INTO users (name) VALUES ('Bob')")    // Will get ID 2
	if err != nil {
		log.Fatal(err)
	}
	_, err = engine.Execute("INSERT INTO orders (user_id, amount) VALUES (1, 100.00)")  // Only Alice
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Minimal Reproduction ===")

	// Verify basic data
	fmt.Println("\n--- Basic verification ---")
	result, err := engine.Execute("SELECT id, name FROM users")
	if err != nil {
		log.Fatal(err)
	}
	selectResult := result.(*mist.SelectResult)
	for _, row := range selectResult.Rows {
		fmt.Printf("User ID %v: %v\n", row[0], row[1])
	}

	result, err = engine.Execute("SELECT user_id, amount FROM orders")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	for _, row := range selectResult.Rows {
		fmt.Printf("Order for user %v: %v\n", row[0], row[1])
	}

	// Manual verification
	fmt.Println("\n--- Manual verification ---")
	result, err = engine.Execute("SELECT COUNT(*) FROM orders WHERE user_id = 1")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Orders for user 1: %v\n", selectResult.Rows[0][0])

	result, err = engine.Execute("SELECT COUNT(*) FROM orders WHERE user_id = 2")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Orders for user 2: %v\n", selectResult.Rows[0][0])

	// Test scalar subquery
	fmt.Println("\n--- Scalar subquery test ---")
	result, err = engine.Execute("SELECT u.name, (SELECT COUNT(*) FROM orders WHERE user_id = u.id) AS count FROM users u")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Scalar subquery results:\n")
	for _, row := range selectResult.Rows {
		fmt.Printf("  %v: %v orders (Alice should have 1, Bob should have 0)\n", row[0], row[1])
	}

	// Test EXISTS
	fmt.Println("\n--- EXISTS test ---")
	result, err = engine.Execute("SELECT u.name FROM users u WHERE EXISTS (SELECT 1 FROM orders WHERE user_id = u.id)")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("EXISTS results: %d users (should be 1: Alice)\n", len(selectResult.Rows))
	for _, row := range selectResult.Rows {
		fmt.Printf("  %v\n", row[0])
	}
}