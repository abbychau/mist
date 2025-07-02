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

	// Insert data
	_, err = engine.Execute("INSERT INTO users (id, name) VALUES (1, 'Alice')")
	if err != nil {
		log.Fatal(err)
	}
	_, err = engine.Execute("INSERT INTO orders (user_id) VALUES (1)")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Detailed Column Resolution Debug ===")

	// Test EXISTS first (we know this works)
	fmt.Println("\n--- EXISTS test (working baseline) ---")
	result, err := engine.Execute("SELECT name FROM users WHERE EXISTS (SELECT 1 FROM orders WHERE user_id = id)")
	if err != nil {
		fmt.Printf("EXISTS failed: %v\n", err)
	} else {
		selectResult := result.(*mist.SelectResult)
		fmt.Printf("EXISTS works: %v\n", selectResult.Rows)
	}

	// Now test scalar subquery with the EXACT same correlation pattern
	fmt.Println("\n--- Scalar subquery test (failing) ---")
	result, err = engine.Execute("SELECT name, (SELECT COUNT(*) FROM orders WHERE user_id = id) FROM users")
	if err != nil {
		fmt.Printf("Scalar subquery failed: %v\n", err)
	} else {
		selectResult := result.(*mist.SelectResult)
		fmt.Printf("Scalar subquery works: %v\n", selectResult.Rows)
	}

	// Test with qualified reference
	fmt.Println("\n--- Scalar subquery with qualified reference ---")
	result, err = engine.Execute("SELECT name, (SELECT COUNT(*) FROM orders WHERE user_id = users.id) FROM users")
	if err != nil {
		fmt.Printf("Qualified scalar subquery failed: %v\n", err)
	} else {
		selectResult := result.(*mist.SelectResult)
		fmt.Printf("Qualified scalar subquery works: %v\n", selectResult.Rows)
	}

	// Test with alias qualified reference
	fmt.Println("\n--- Scalar subquery with alias ---")
	result, err = engine.Execute("SELECT u.name, (SELECT COUNT(*) FROM orders WHERE user_id = u.id) FROM users u")
	if err != nil {
		fmt.Printf("Alias scalar subquery failed: %v\n", err)
	} else {
		selectResult := result.(*mist.SelectResult)
		fmt.Printf("Alias scalar subquery works: %v\n", selectResult.Rows)
	}

	fmt.Println("\n--- Analysis ---")
	fmt.Println("If EXISTS works but scalar subquery fails with the same correlation pattern,")
	fmt.Println("the issue is in the scalar subquery execution path, not the column resolution logic.")
}