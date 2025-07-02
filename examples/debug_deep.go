package main

import (
	"fmt"
	"log"

	"github.com/abbychau/mist"
)

func main() {
	// Create a new engine
	engine := mist.NewSQLEngine()

	// Create tables
	_, err := engine.Execute("CREATE TABLE users (id INT, name VARCHAR(100))")
	if err != nil {
		log.Fatal(err)
	}

	_, err = engine.Execute("CREATE TABLE orders (user_id INT)")
	if err != nil {
		log.Fatal(err)
	}

	// Insert data - Alice has orders, Bob doesn't
	_, err = engine.Execute("INSERT INTO users (id, name) VALUES (1, 'Alice')")
	if err != nil {
		log.Fatal(err)
	}
	_, err = engine.Execute("INSERT INTO users (id, name) VALUES (2, 'Bob')")
	if err != nil {
		log.Fatal(err)
	}
	_, err = engine.Execute("INSERT INTO orders (user_id) VALUES (1)")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Deep Debug ===")

	// First verify the basic subquery works
	fmt.Println("\n--- Basic subquery verification ---")
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

	// Test EXISTS with hardcoded values
	fmt.Println("\n--- EXISTS with hardcoded values ---")
	result, err = engine.Execute("SELECT name FROM users WHERE EXISTS (SELECT 1 FROM orders WHERE user_id = 1)")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("EXISTS with user_id=1: %d rows (should be 2)\n", len(selectResult.Rows))

	result, err = engine.Execute("SELECT name FROM users WHERE EXISTS (SELECT 1 FROM orders WHERE user_id = 2)")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("EXISTS with user_id=2: %d rows (should be 0)\n", len(selectResult.Rows))

	// Test the actual correlated query for each user separately
	fmt.Println("\n--- Individual correlated tests ---")
	result, err = engine.Execute("SELECT name FROM users u WHERE u.id = 1 AND EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id)")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Alice with correlated EXISTS: %d rows (should be 1)\n", len(selectResult.Rows))

	result, err = engine.Execute("SELECT name FROM users u WHERE u.id = 2 AND EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id)")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Bob with correlated EXISTS: %d rows (should be 0)\n", len(selectResult.Rows))

	// Now the full query
	fmt.Println("\n--- Full correlated query ---")
	result, err = engine.Execute("SELECT name FROM users u WHERE EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id)")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Full correlated EXISTS: %d rows (should be 1: Alice)\n", len(selectResult.Rows))
	for _, row := range selectResult.Rows {
		fmt.Printf("  %v\n", row[0])
	}

	// Let's also test a very simple case to ensure the correlation itself works
	fmt.Println("\n--- Simple correlation test ---")
	result, err = engine.Execute("SELECT u.id, u.name FROM users u WHERE u.id = 1")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Simple WHERE u.id = 1: %d rows\n", len(selectResult.Rows))

	result, err = engine.Execute("SELECT u.id, u.name FROM users u WHERE u.id = 2")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Simple WHERE u.id = 2: %d rows\n", len(selectResult.Rows))
}