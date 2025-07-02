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
	err := createTestTables(engine)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Debug Correlated EXISTS ===")

	// Check the data first
	fmt.Println("\n--- Users ---")
	result, err := engine.Execute("SELECT id, name, age FROM users")
	if err != nil {
		log.Fatal(err)
	}
	selectResult := result.(*mist.SelectResult)
	fmt.Printf("Columns: %v\n", selectResult.Columns)
	for _, row := range selectResult.Rows {
		fmt.Printf("  %v\n", row)
	}

	fmt.Println("\n--- Orders ---")
	result, err = engine.Execute("SELECT id, user_id, amount FROM orders")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Columns: %v\n", selectResult.Columns)
	for _, row := range selectResult.Rows {
		fmt.Printf("  %v\n", row)
	}

	fmt.Println("\n--- Simple JOIN to verify expected results ---")
	result, err = engine.Execute("SELECT DISTINCT u.name FROM users u INNER JOIN orders o ON u.id = o.user_id")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Users with orders (via JOIN): %v\n", selectResult.Columns)
	for _, row := range selectResult.Rows {
		fmt.Printf("  %v\n", row)
	}

	fmt.Println("\n--- Test correlated EXISTS ---")
	result, err = engine.Execute("SELECT u.name FROM users u WHERE EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id)")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	fmt.Printf("Users with orders (via EXISTS): %v\n", selectResult.Columns)
	for _, row := range selectResult.Rows {
		fmt.Printf("  %v\n", row)
	}
}

func createTestTables(engine *mist.SQLEngine) error {
	// Create users table
	_, err := engine.Execute("CREATE TABLE users (id INT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(100), age INT)")
	if err != nil {
		return err
	}

	// Create orders table
	_, err = engine.Execute("CREATE TABLE orders (id INT PRIMARY KEY AUTO_INCREMENT, user_id INT, amount DECIMAL(10,2))")
	if err != nil {
		return err
	}

	// Insert test data
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
			return fmt.Errorf("error inserting test data: %v", err)
		}
	}

	return nil
}