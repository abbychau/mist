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

	fmt.Println("=== Testing Correlated EXISTS Subqueries ===")

	// Test correlated EXISTS subqueries
	tests := []struct {
		name        string
		query       string
		description string
	}{
		{
			"Basic correlated EXISTS",
			"SELECT u.name FROM users u WHERE EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id)",
			"Get users who have orders (correlated subquery)",
		},
		{
			"Correlated NOT EXISTS",
			"SELECT u.name FROM users u WHERE NOT EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id)",
			"Get users who have no orders (correlated subquery)",
		},
		{
			"Complex correlated EXISTS",
			"SELECT u.name FROM users u WHERE u.age > 25 AND EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id AND o.amount > 100)",
			"Get users over 25 with orders over $100",
		},
		{
			"Non-correlated EXISTS (should still work)",
			"SELECT u.name FROM users u WHERE EXISTS (SELECT 1 FROM orders)",
			"Get all users if any orders exist (non-correlated)",
		},
	}

	for _, test := range tests {
		fmt.Printf("\n--- %s ---\n", test.name)
		fmt.Printf("Description: %s\n", test.description)
		fmt.Printf("Query: %s\n", test.query)
		
		result, err := engine.Execute(test.query)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			continue
		}

		selectResult, ok := result.(*mist.SelectResult)
		if !ok {
			fmt.Printf("❌ Unexpected result type: %T\n", result)
			continue
		}

		fmt.Printf("✅ Results (%d rows):\n", len(selectResult.Rows))
		fmt.Printf("Columns: %v\n", selectResult.Columns)
		for _, row := range selectResult.Rows {
			fmt.Printf("  %v\n", row)
		}
	}

	fmt.Println("\n=== Correlated EXISTS Testing Complete! ===")
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