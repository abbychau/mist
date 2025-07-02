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
	_, err = engine.Execute("INSERT INTO users (name) VALUES ('Bob')")
	if err != nil {
		log.Fatal(err)
	}

	// Alice has an order, Bob doesn't
	_, err = engine.Execute("INSERT INTO orders (user_id) VALUES (1)")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Simple Correlated EXISTS Test ===")

	fmt.Println("\n--- Data Check ---")
	result, err := engine.Execute("SELECT id, name FROM users")
	if err != nil {
		log.Fatal(err)
	}
	selectResult := result.(*mist.SelectResult)
	fmt.Printf("Users: %v\n", selectResult.Columns)
	for _, row := range selectResult.Rows {
		fmt.Printf("  %v\n", row)
	}

	result, err = engine.Execute("SELECT id, user_id FROM orders")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Orders: %v\n", selectResult.Columns)
	for _, row := range selectResult.Rows {
		fmt.Printf("  %v\n", row)
	}

	fmt.Println("\n--- Non-correlated EXISTS (should return both users) ---")
	result, err = engine.Execute("SELECT name FROM users WHERE EXISTS (SELECT 1 FROM orders)")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Result: %v\n", selectResult.Columns)
	for _, row := range selectResult.Rows {
		fmt.Printf("  %v\n", row)
	}

	fmt.Println("\n--- Correlated EXISTS (should return only Alice) ---")
	result, err = engine.Execute("SELECT name FROM users u WHERE EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id)")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Result: %v\n", selectResult.Columns)
	for _, row := range selectResult.Rows {
		fmt.Printf("  %v\n", row)
	}

	fmt.Println("\n--- Manual test with specific user_id ---")
	result, err = engine.Execute("SELECT name FROM users WHERE EXISTS (SELECT 1 FROM orders WHERE user_id = 1)")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Result for user_id=1: %v\n", selectResult.Columns)
	for _, row := range selectResult.Rows {
		fmt.Printf("  %v\n", row)
	}

	result, err = engine.Execute("SELECT name FROM users WHERE EXISTS (SELECT 1 FROM orders WHERE user_id = 2)")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Result for user_id=2: %v\n", selectResult.Columns)
	for _, row := range selectResult.Rows {
		fmt.Printf("  %v\n", row)
	}
}