package main

import (
	"fmt"
	"log"

	"github.com/abbychau/mist"
)

func main() {
	// Create a new engine
	engine := mist.NewSQLEngine()

	// Create EXACT same table structure as comprehensive test
	_, err := engine.Execute("CREATE TABLE users (id INT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(100), age INT)")
	if err != nil {
		log.Fatal(err)
	}

	_, err = engine.Execute("CREATE TABLE orders (id INT PRIMARY KEY AUTO_INCREMENT, user_id INT, amount DECIMAL(10,2))")
	if err != nil {
		log.Fatal(err)
	}

	// Insert EXACT same data as comprehensive test
	testData := []string{
		// Users
		"INSERT INTO users (name, age) VALUES ('Alice', 30)",
		"INSERT INTO users (name, age) VALUES ('Bob', 25)",
		"INSERT INTO users (name, age) VALUES ('Charlie', 35)",
		"INSERT INTO users (name, age) VALUES ('Diana', 28)",
		"INSERT INTO users (name, age) VALUES ('Adam', 32)",

		// Orders (Alice, Charlie, and Adam have orders)
		"INSERT INTO orders (user_id, amount) VALUES (1, 150.00)",  // Alice
		"INSERT INTO orders (user_id, amount) VALUES (1, 75.00)",   // Alice
		"INSERT INTO orders (user_id, amount) VALUES (3, 200.00)",  // Charlie
		"INSERT INTO orders (user_id, amount) VALUES (3, 25.00)",   // Charlie
		"INSERT INTO orders (user_id, amount) VALUES (5, 300.00)",  // Adam
	}

	for _, query := range testData {
		_, err := engine.Execute(query)
		if err != nil {
			log.Fatal(fmt.Errorf("error inserting test data: %v", err))
		}
	}

	fmt.Println("=== Exact Comprehensive Test Reproduction ===")

	// Test the EXACT same queries as the comprehensive test
	fmt.Println("\n--- Correlated EXISTS (exact same query) ---")
	result, err := engine.Execute("SELECT u.name FROM users u WHERE EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id)")
	if err != nil {
		log.Fatal(err)
	}
	selectResult := result.(*mist.SelectResult)
	fmt.Printf("Results: %d users (should be 3)\n", len(selectResult.Rows))
	for _, row := range selectResult.Rows {
		fmt.Printf("  %v\n", row[0])
	}

	fmt.Println("\n--- Scalar subquery in SELECT (exact same query) ---")
	result, err = engine.Execute("SELECT u.name, (SELECT COUNT(*) FROM orders WHERE user_id = u.id) AS order_count FROM users u")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Results:\n")
	for _, row := range selectResult.Rows {
		fmt.Printf("  %v: %v orders\n", row[0], row[1])
	}
}