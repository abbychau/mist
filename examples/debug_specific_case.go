package main

import (
	"fmt"
	"log"

	"github.com/abbychau/mist"
)

func main() {
	// Create a new engine
	engine := mist.NewSQLEngine()

	// Create test tables - exactly the same as debug_correlated.go
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

	fmt.Println("=== Specific Case Debug ===")

	// Test each user individually
	for i := 1; i <= 4; i++ {
		fmt.Printf("\n--- Testing user id %d ---\n", i)
		
		// Get the user name
		result, err := engine.Execute(fmt.Sprintf("SELECT name FROM users WHERE id = %d", i))
		if err != nil {
			log.Fatal(err)
		}
		selectResult := result.(*mist.SelectResult)
		userName := ""
		if len(selectResult.Rows) > 0 {
			userName = selectResult.Rows[0][0].(string)
		}
		fmt.Printf("User: %s\n", userName)
		
		// Check if they have orders
		result, err = engine.Execute(fmt.Sprintf("SELECT COUNT(*) FROM orders WHERE user_id = %d", i))
		if err != nil {
			log.Fatal(err)
		}
		selectResult = result.(*mist.SelectResult)
		orderCount := selectResult.Rows[0][0]
		fmt.Printf("Order count: %v\n", orderCount)
		
		// Test EXISTS for this specific user
		result, err = engine.Execute(fmt.Sprintf("SELECT name FROM users u WHERE u.id = %d AND EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id)", i))
		if err != nil {
			log.Fatal(err)
		}
		selectResult = result.(*mist.SelectResult)
		fmt.Printf("EXISTS result: %d rows\n", len(selectResult.Rows))
		if len(selectResult.Rows) > 0 {
			fmt.Printf("  %v\n", selectResult.Rows[0])
		}
	}

	fmt.Println("\n--- Full correlated EXISTS ---")
	result, err := engine.Execute("SELECT u.name FROM users u WHERE EXISTS (SELECT 1 FROM orders o WHERE o.user_id = u.id)")
	if err != nil {
		log.Fatal(err)
	}
	selectResult := result.(*mist.SelectResult)
	fmt.Printf("Full correlated EXISTS: %d rows\n", len(selectResult.Rows))
	for _, row := range selectResult.Rows {
		fmt.Printf("  %v\n", row)
	}
}