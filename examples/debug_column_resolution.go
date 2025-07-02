package main

import (
	"fmt"
	"log"

	"github.com/abbychau/mist"
)

func main() {
	// Create a new engine
	engine := mist.NewSQLEngine()

	// Create test tables
	_, err := engine.Execute("CREATE TABLE users (id INT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(100))")
	if err != nil {
		log.Fatal(err)
	}

	_, err = engine.Execute("CREATE TABLE orders (id INT PRIMARY KEY AUTO_INCREMENT, user_id INT)")
	if err != nil {
		log.Fatal(err)
	}

	// Insert minimal test data  
	_, err = engine.Execute("INSERT INTO users (name) VALUES ('Alice')")
	if err != nil {
		log.Fatal(err)
	}
	_, err = engine.Execute("INSERT INTO orders (user_id) VALUES (1)")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Debug Column Resolution ===")

	// Test the subquery part separately to see what's happening
	fmt.Println("\n--- Test subquery alone ---")
	result, err := engine.Execute("SELECT 1 FROM orders o WHERE o.user_id = 1")
	if err != nil {
		log.Fatal(err)
	}
	selectResult := result.(*mist.SelectResult)
	fmt.Printf("SELECT 1 FROM orders o WHERE o.user_id = 1: %d rows\n", len(selectResult.Rows))

	result, err = engine.Execute("SELECT 1 FROM orders o WHERE o.user_id = 2")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("SELECT 1 FROM orders o WHERE o.user_id = 2: %d rows\n", len(selectResult.Rows))

	fmt.Println("\n--- Test basic EXISTS ---")
	result, err = engine.Execute("SELECT name FROM users WHERE EXISTS (SELECT 1 FROM orders WHERE user_id = 1)")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("EXISTS with user_id = 1: %d rows\n", len(selectResult.Rows))
	for _, row := range selectResult.Rows {
		fmt.Printf("  %v\n", row)
	}

	fmt.Println("\n--- Test with unqualified column reference ---")
	result, err = engine.Execute("SELECT u.name FROM users u WHERE EXISTS (SELECT 1 FROM orders WHERE user_id = u.id)")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("EXISTS with unqualified correlation: %d rows\n", len(selectResult.Rows))
	for _, row := range selectResult.Rows {
		fmt.Printf("  %v\n", row)
	}

	fmt.Println("\n--- Test with qualified column reference ---")
	result, err = engine.Execute("SELECT u.name FROM users u WHERE EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id)")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("EXISTS with qualified correlation: %d rows\n", len(selectResult.Rows))
	for _, row := range selectResult.Rows {
		fmt.Printf("  %v\n", row)
	}
}