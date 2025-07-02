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

	fmt.Println("=== Comprehensive Subquery Testing ===")

	// Test combinations of different subquery types
	tests := []struct {
		name        string
		query       string
		description string
	}{
		{
			"Correlated EXISTS",
			"SELECT u.name FROM users u WHERE EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id)",
			"Get users who have orders (correlated)",
		},
		{
			"Correlated NOT EXISTS",
			"SELECT u.name FROM users u WHERE NOT EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id)",
			"Get users who have no orders (correlated)",
		},
		{
			"Scalar subquery in SELECT",
			"SELECT u.name, (SELECT COUNT(*) FROM orders WHERE user_id = u.id) AS order_count FROM users u",
			"Get users with their order counts (correlated scalar)",
		},
		{
			"Combined EXISTS and scalar",
			"SELECT u.name, (SELECT MAX(amount) FROM orders) AS max_order FROM users u WHERE EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id)",
			"Users with orders plus global max order amount",
		},
		{
			"Complex correlated EXISTS with conditions",
			"SELECT u.name FROM users u WHERE u.age > 25 AND EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id AND o.amount > 100)",
			"Users over 25 with orders over $100",
		},
		{
			"Nested conditions",
			"SELECT u.name FROM users u WHERE EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id) AND u.age = (SELECT MAX(age) FROM users WHERE name LIKE 'A%')",
			"Users with orders and max age among names starting with A",
		},
		{
			"Multiple correlated subqueries",
			"SELECT u.name FROM users u WHERE EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id) AND NOT EXISTS (SELECT 1 FROM orders o2 WHERE o2.user_id = u.id AND o2.amount < 50)",
			"Users with orders but no orders under $50",
		},
		{
			"Scalar subquery with EXISTS in WHERE",
			"SELECT (SELECT COUNT(*) FROM users WHERE age > 30) AS older_users FROM users WHERE EXISTS (SELECT 1 FROM orders) LIMIT 1",
			"Count of older users if any orders exist",
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

	fmt.Println("\n=== Performance and Edge Cases ===")

	// Test performance and edge cases
	edgeCases := []struct {
		name  string
		query string
		desc  string
	}{
		{
			"Empty subquery result",
			"SELECT u.name FROM users u WHERE EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id AND o.amount > 10000)",
			"No orders over $10k",
		},
		{
			"NULL handling in correlation",
			"SELECT u.name FROM users u WHERE EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id AND o.amount IS NOT NULL)",
			"Users with non-NULL order amounts",
		},
		{
			"Multiple table correlation",
			"SELECT p.name FROM products p WHERE EXISTS (SELECT 1 FROM orders o JOIN order_items oi ON o.id = oi.order_id WHERE oi.product_id = p.id)",
			"Products that have been ordered (if tables existed)",
		},
	}

	for _, test := range edgeCases {
		fmt.Printf("\n--- %s ---\n", test.name)
		fmt.Printf("Description: %s\n", test.desc)
		fmt.Printf("Query: %s\n", test.query)
		
		result, err := engine.Execute(test.query)
		if err != nil {
			// Expected for the multi-table case since those tables don't exist
			fmt.Printf("⚠️ Expected error: %v\n", err)
			continue
		}

		selectResult, ok := result.(*mist.SelectResult)
		if !ok {
			fmt.Printf("❌ Unexpected result type: %T\n", result)
			continue
		}

		fmt.Printf("✅ Results (%d rows):\n", len(selectResult.Rows))
		if len(selectResult.Rows) <= 5 { // Only show first 5 rows
			for _, row := range selectResult.Rows {
				fmt.Printf("  %v\n", row)
			}
		} else {
			for i := 0; i < 5; i++ {
				fmt.Printf("  %v\n", selectResult.Rows[i])
			}
			fmt.Printf("  ... (%d more rows)\n", len(selectResult.Rows)-5)
		}
	}

	fmt.Println("\n=== Comprehensive Subquery Testing Complete! ===")
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
		"INSERT INTO users (name, age) VALUES ('Adam', 32)",

		// Orders (Alice, Charlie, and Adam have orders)
		"INSERT INTO orders (user_id, amount) VALUES (1, 150.00)",  // Alice
		"INSERT INTO orders (user_id, amount) VALUES (1, 75.00)",   // Alice
		"INSERT INTO orders (user_id, amount) VALUES (3, 200.00)",  // Charlie
		"INSERT INTO orders (user_id, amount) VALUES (3, 25.00)",   // Charlie (under $50)
		"INSERT INTO orders (user_id, amount) VALUES (5, 300.00)",  // Adam
	}

	for _, query := range testData {
		_, err := engine.Execute(query)
		if err != nil {
			return fmt.Errorf("error inserting test data: %v", err)
		}
	}

	return nil
}