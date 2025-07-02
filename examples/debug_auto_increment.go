package main

import (
	"fmt"
	"log"

	"github.com/abbychau/mist"
)

func main() {
	// Create a new engine
	engine := mist.NewSQLEngine()

	// Create tables with AUTO_INCREMENT (like the failing case)
	_, err := engine.Execute("CREATE TABLE users (id INT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(100), age INT)")
	if err != nil {
		log.Fatal(err)
	}

	_, err = engine.Execute("CREATE TABLE orders (id INT PRIMARY KEY AUTO_INCREMENT, user_id INT, amount DECIMAL(10,2))")
	if err != nil {
		log.Fatal(err)
	}

	// Insert data using AUTO_INCREMENT
	_, err = engine.Execute("INSERT INTO users (name, age) VALUES ('Alice', 30)")
	if err != nil {
		log.Fatal(err)
	}
	_, err = engine.Execute("INSERT INTO users (name, age) VALUES ('Bob', 25)")
	if err != nil {
		log.Fatal(err)
	}
	_, err = engine.Execute("INSERT INTO orders (user_id, amount) VALUES (1, 150.00)")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Auto Increment Debug ===")

	// Check the data
	fmt.Println("\n--- Data check ---")
	result, err := engine.Execute("SELECT id, name, age FROM users")
	if err != nil {
		log.Fatal(err)
	}
	selectResult := result.(*mist.SelectResult)
	fmt.Printf("Users:\n")
	for _, row := range selectResult.Rows {
		fmt.Printf("  ID: %v, Name: %v, Age: %v\n", row[0], row[1], row[2])
	}

	result, err = engine.Execute("SELECT id, user_id, amount FROM orders")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Orders:\n")
	for _, row := range selectResult.Rows {
		fmt.Printf("  ID: %v, UserID: %v, Amount: %v\n", row[0], row[1], row[2])
	}

	// Test basic queries
	fmt.Println("\n--- Basic EXISTS tests ---")
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

	// Test the correlated EXISTS
	fmt.Println("\n--- Correlated EXISTS test ---")
	result, err = engine.Execute("SELECT u.name FROM users u WHERE EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id)")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Correlated EXISTS: %d rows (should be 1: Alice)\n", len(selectResult.Rows))
	for _, row := range selectResult.Rows {
		fmt.Printf("  Name: %v\n", row[0])
	}

	// Test individual users
	fmt.Println("\n--- Individual user tests ---")
	result, err = engine.Execute("SELECT u.name FROM users u WHERE u.id = 1 AND EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id)")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Alice: %d rows (should be 1)\n", len(selectResult.Rows))

	result, err = engine.Execute("SELECT u.name FROM users u WHERE u.id = 2 AND EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id)")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Bob: %d rows (should be 0)\n", len(selectResult.Rows))
}