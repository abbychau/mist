package main

import (
	"fmt"
	"log"

	"github.com/abbychau/mist"
)

func main() {
	// Create a new engine
	engine := mist.NewSQLEngine()

	// Create AUTO_INCREMENT tables
	_, err := engine.Execute("CREATE TABLE users (id INT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(100))")
	if err != nil {
		log.Fatal(err)
	}

	_, err = engine.Execute("CREATE TABLE orders (user_id INT, amount DECIMAL(10,2))")
	if err != nil {
		log.Fatal(err)
	}

	// Insert 5 users like the failing case
	users := []string{"Alice", "Bob", "Charlie", "Diana", "Adam"}
	for _, name := range users {
		_, err = engine.Execute(fmt.Sprintf("INSERT INTO users (name) VALUES ('%s')", name))
		if err != nil {
			log.Fatal(err)
		}
	}

	// Insert orders for users 1, 3, 5 (Alice, Charlie, Adam) like the failing case
	orders := [][]interface{}{
		{1, 150.00}, // Alice
		{1, 75.00},  // Alice
		{3, 200.00}, // Charlie
		{3, 25.00},  // Charlie  
		{5, 300.00}, // Adam
	}
	for _, order := range orders {
		_, err = engine.Execute(fmt.Sprintf("INSERT INTO orders (user_id, amount) VALUES (%v, %v)", order[0], order[1]))
		if err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("=== Five Users Test ===")

	// Verify data
	fmt.Println("\n--- Data verification ---")
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

	// Test each user's order count
	fmt.Println("\n--- Individual counts ---")
	for i := 1; i <= 5; i++ {
		result, err := engine.Execute(fmt.Sprintf("SELECT COUNT(*) FROM orders WHERE user_id = %d", i))
		if err != nil {
			log.Fatal(err)
		}
		selectResult := result.(*mist.SelectResult)
		fmt.Printf("User %d: %v orders\n", i, selectResult.Rows[0][0])
	}

	// Test scalar subquery
	fmt.Println("\n--- Scalar subquery test ---")
	result, err = engine.Execute("SELECT u.name, (SELECT COUNT(*) FROM orders WHERE user_id = u.id) AS count FROM users u")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Scalar subquery results:\n")
	for _, row := range selectResult.Rows {
		fmt.Printf("  %v: %v orders\n", row[0], row[1])
	}

	// Test EXISTS
	fmt.Println("\n--- EXISTS test ---")
	result, err = engine.Execute("SELECT u.name FROM users u WHERE EXISTS (SELECT 1 FROM orders WHERE user_id = u.id)")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("EXISTS results: %d users (should be 3: Alice, Charlie, Adam)\n", len(selectResult.Rows))
	for _, row := range selectResult.Rows {
		fmt.Printf("  %v\n", row[0])
	}

	fmt.Println("\nExpected:")
	fmt.Println("  Alice: 2 orders")
	fmt.Println("  Bob: 0 orders")
	fmt.Println("  Charlie: 2 orders") 
	fmt.Println("  Diana: 0 orders")
	fmt.Println("  Adam: 1 order")
	fmt.Println("  EXISTS should return: Alice, Charlie, Adam")
}