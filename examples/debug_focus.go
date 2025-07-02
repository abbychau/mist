package main

import (
	"fmt"
	"log"

	"github.com/abbychau/mist"
)

func main() {
	// Create a new engine
	engine := mist.NewSQLEngine()

	// Create minimal tables
	_, err := engine.Execute("CREATE TABLE users (id INT)")
	if err != nil {
		log.Fatal(err)
	}

	_, err = engine.Execute("CREATE TABLE orders (user_id INT)")
	if err != nil {
		log.Fatal(err)
	}

	// Insert minimal data - only user 1 has an order
	_, err = engine.Execute("INSERT INTO users (id) VALUES (1)")
	if err != nil {
		log.Fatal(err)
	}
	_, err = engine.Execute("INSERT INTO users (id) VALUES (2)")
	if err != nil {
		log.Fatal(err)
	}
	_, err = engine.Execute("INSERT INTO orders (user_id) VALUES (1)") // Only user 1 has an order
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Focus Test ===")

	// Test non-correlated EXISTS first (baseline)
	fmt.Println("\n--- Non-correlated EXISTS ---")
	result, err := engine.Execute("SELECT id FROM users WHERE EXISTS (SELECT 1 FROM orders WHERE user_id = 1)")
	if err != nil {
		log.Fatal(err)
	}
	selectResult := result.(*mist.SelectResult)
	fmt.Printf("Non-correlated EXISTS with user_id=1: %d rows (should be 2)\n", len(selectResult.Rows))

	result, err = engine.Execute("SELECT id FROM users WHERE EXISTS (SELECT 1 FROM orders WHERE user_id = 2)")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Non-correlated EXISTS with user_id=2: %d rows (should be 0)\n", len(selectResult.Rows))

	// Test the subquery evaluation in isolation for each user context
	fmt.Println("\n--- Test subquery execution specifically ---")
	
	// This should test what happens when we run the subquery with user 1's context
	fmt.Println("What we want to achieve:")
	fmt.Println("  For user 1: EXISTS (SELECT 1 FROM orders WHERE user_id = 1) -> TRUE")
	fmt.Println("  For user 2: EXISTS (SELECT 1 FROM orders WHERE user_id = 2) -> FALSE")

	// Full correlated test
	fmt.Println("\n--- Correlated EXISTS ---")
	result, err = engine.Execute("SELECT id FROM users u WHERE EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id)")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Correlated EXISTS: %d rows (should be 1: user 1)\n", len(selectResult.Rows))
	for _, row := range selectResult.Rows {
		fmt.Printf("  User ID: %v\n", row[0])
	}

	// Test step by step
	fmt.Println("\n--- Debug step by step ---")
	fmt.Println("Expected behavior:")
	fmt.Println("1. For each user in outer query")
	fmt.Println("2. Execute subquery with that user's context")
	fmt.Println("3. If subquery returns rows, include user in result")
	
	// Let's test what the subquery returns for each user context
	// We can't directly test this, but we can infer from the final result
}