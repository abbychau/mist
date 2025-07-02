package main

import (
	"fmt"
	"log"

	"github.com/abbychau/mist"
)

func main() {
	// Create a new engine
	engine := mist.NewSQLEngine()

	// Create exact same tables as the failing test
	_, err := engine.Execute("CREATE TABLE users (id INT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(100), age INT)")
	if err != nil {
		log.Fatal(err)
	}

	_, err = engine.Execute("CREATE TABLE orders (id INT PRIMARY KEY AUTO_INCREMENT, user_id INT, amount DECIMAL(10,2))")
	if err != nil {
		log.Fatal(err)
	}

	// Insert exact same test data as failing test
	testData := []string{
		// Users
		"INSERT INTO users (name, age) VALUES ('Alice', 30)",
		"INSERT INTO users (name, age) VALUES ('Bob', 25)",
		"INSERT INTO users (name, age) VALUES ('Charlie', 35)",
		"INSERT INTO users (name, age) VALUES ('Diana', 28)",

		// Orders (Alice and Charlie have orders, Bob and Diana don't)
		"INSERT INTO orders (user_id, amount) VALUES (1, 150.00)",  // Alice
		"INSERT INTO orders (user_id, amount) VALUES (1, 75.00)",   // Alice
		"INSERT INTO orders (user_id, amount) VALUES (3, 200.00)",  // Charlie
		"INSERT INTO orders (user_id, amount) VALUES (3, 50.00)",   // Charlie
	}

	for _, query := range testData {
		_, err := engine.Execute(query)
		if err != nil {
			log.Fatal(fmt.Errorf("error inserting test data: %v", err))
		}
	}

	fmt.Println("=== Exact Case Debug ===")

	// Check data first
	fmt.Println("\n--- Verify data ---")
	result, err := engine.Execute("SELECT id, name FROM users")
	if err != nil {
		log.Fatal(err)
	}
	selectResult := result.(*mist.SelectResult)
	fmt.Printf("Users:\n")
	for _, row := range selectResult.Rows {
		fmt.Printf("  ID: %v, Name: %v\n", row[0], row[1])
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

	// Test the exact failing query
	fmt.Println("\n--- Test exact failing query ---")
	result, err = engine.Execute("SELECT u.name FROM users u WHERE EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id)")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Correlated EXISTS result: %d rows (should be 2: Alice, Charlie)\n", len(selectResult.Rows))
	for _, row := range selectResult.Rows {
		fmt.Printf("  Name: %v\n", row[0])
	}

	// Test each user individually 
	fmt.Println("\n--- Test each user individually ---")
	for i := 1; i <= 4; i++ {
		result, err = engine.Execute(fmt.Sprintf("SELECT name FROM users u WHERE u.id = %d AND EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id)", i))
		if err != nil {
			log.Fatal(err)
		}
		selectResult = result.(*mist.SelectResult)
		
		// Get user name
		nameResult, err := engine.Execute(fmt.Sprintf("SELECT name FROM users WHERE id = %d", i))
		if err != nil {
			log.Fatal(err)
		}
		nameSelectResult := nameResult.(*mist.SelectResult)
		userName := ""
		if len(nameSelectResult.Rows) > 0 {
			userName = nameSelectResult.Rows[0][0].(string)
		}
		
		fmt.Printf("User %s (ID %d): %d rows\n", userName, i, len(selectResult.Rows))
	}
}