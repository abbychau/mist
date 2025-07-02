package main

import (
	"fmt"
	"log"

	"github.com/abbychau/mist"
)

func main() {
	// Create a new engine
	engine := mist.NewSQLEngine()

	// Create the exact same tables and data as the comprehensive test
	_, err := engine.Execute("CREATE TABLE users (id INT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(100), age INT)")
	if err != nil {
		log.Fatal(err)
	}

	_, err = engine.Execute("CREATE TABLE orders (id INT PRIMARY KEY AUTO_INCREMENT, user_id INT, amount DECIMAL(10,2))")
	if err != nil {
		log.Fatal(err)
	}

	// Insert exact same test data
	testData := []string{
		// Users
		"INSERT INTO users (name, age) VALUES ('Alice', 30)",   // ID 1
		"INSERT INTO users (name, age) VALUES ('Bob', 25)",     // ID 2
		"INSERT INTO users (name, age) VALUES ('Charlie', 35)", // ID 3
		"INSERT INTO users (name, age) VALUES ('Diana', 28)",   // ID 4
		"INSERT INTO users (name, age) VALUES ('Adam', 32)",    // ID 5

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

	fmt.Println("=== Debug Comprehensive Data ===")

	// Verify data
	fmt.Println("\n--- Data verification ---")
	result, err := engine.Execute("SELECT id, name, age FROM users")
	if err != nil {
		log.Fatal(err)
	}
	selectResult := result.(*mist.SelectResult)
	fmt.Printf("Users:\n")
	for _, row := range selectResult.Rows {
		fmt.Printf("  ID: %v, Name: %v, Age: %v\n", row[0], row[1], row[2])
	}

	result, err = engine.Execute("SELECT user_id, amount FROM orders")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Orders:\n")
	for _, row := range selectResult.Rows {
		fmt.Printf("  UserID: %v, Amount: %v\n", row[0], row[1])
	}

	// Test each user's order count individually
	fmt.Println("\n--- Individual order counts ---")
	for i := 1; i <= 5; i++ {
		result, err := engine.Execute(fmt.Sprintf("SELECT COUNT(*) FROM orders WHERE user_id = %d", i))
		if err != nil {
			log.Fatal(err)
		}
		selectResult := result.(*mist.SelectResult)
		
		nameResult, err := engine.Execute(fmt.Sprintf("SELECT name FROM users WHERE id = %d", i))
		if err != nil {
			log.Fatal(err)
		}
		nameSelectResult := nameResult.(*mist.SelectResult)
		userName := nameSelectResult.Rows[0][0].(string)
		
		fmt.Printf("User %s (ID %d): %v orders\n", userName, i, selectResult.Rows[0][0])
	}

	// Test scalar subquery correlation
	fmt.Println("\n--- Scalar subquery correlation ---")
	result, err = engine.Execute("SELECT u.name, (SELECT COUNT(*) FROM orders WHERE user_id = u.id) AS order_count FROM users u")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Scalar subquery results:\n")
	for _, row := range selectResult.Rows {
		fmt.Printf("  %v: %v orders\n", row[0], row[1])
	}

	// Test EXISTS correlation
	fmt.Println("\n--- EXISTS correlation ---")
	result, err = engine.Execute("SELECT u.name FROM users u WHERE EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id)")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("EXISTS results: %d users (expected: 3 - Alice, Charlie, Adam)\n", len(selectResult.Rows))
	for _, row := range selectResult.Rows {
		fmt.Printf("  %v\n", row[0])
	}
}