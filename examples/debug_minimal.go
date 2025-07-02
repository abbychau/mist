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

	// Insert minimal data
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

	fmt.Println("=== Minimal Test ===")

	// Test the subquery part in isolation with different values
	fmt.Println("\n--- Subquery tests ---")
	result, err := engine.Execute("SELECT COUNT(*) FROM orders WHERE user_id = 1")
	if err != nil {
		log.Fatal(err)
	}
	selectResult := result.(*mist.SelectResult)
	fmt.Printf("Orders for user_id=1: %v\n", selectResult.Rows[0][0])

	result, err = engine.Execute("SELECT COUNT(*) FROM orders WHERE user_id = 2")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Orders for user_id=2: %v\n", selectResult.Rows[0][0])

	// Now test with EXISTS for each user individually
	fmt.Println("\n--- Individual EXISTS tests ---")
	result, err = engine.Execute("SELECT id FROM users WHERE id = 1 AND EXISTS (SELECT 1 FROM orders WHERE user_id = 1)")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("User 1 with hardcoded EXISTS: %d rows\n", len(selectResult.Rows))

	result, err = engine.Execute("SELECT id FROM users WHERE id = 2 AND EXISTS (SELECT 1 FROM orders WHERE user_id = 2)")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("User 2 with hardcoded EXISTS: %d rows\n", len(selectResult.Rows))

	// Now the correlated version
	fmt.Println("\n--- Correlated EXISTS tests ---")
	result, err = engine.Execute("SELECT id FROM users u WHERE u.id = 1 AND EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id)")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("User 1 with correlated EXISTS: %d rows\n", len(selectResult.Rows))

	result, err = engine.Execute("SELECT id FROM users u WHERE u.id = 2 AND EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id)")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("User 2 with correlated EXISTS: %d rows\n", len(selectResult.Rows))

	// Full correlated test
	fmt.Println("\n--- Full correlated EXISTS ---")
	result, err = engine.Execute("SELECT id FROM users u WHERE EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id)")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("All users with orders (correlated): %d rows\n", len(selectResult.Rows))
	for _, row := range selectResult.Rows {
		fmt.Printf("  %v\n", row)
	}
}