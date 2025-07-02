package main

import (
	"fmt"
	"log"

	"github.com/abbychau/mist"
)

func main() {
	// Create a new engine
	engine := mist.NewSQLEngine()

	// Create very simple tables
	_, err := engine.Execute("CREATE TABLE users (id INT, name VARCHAR(100))")
	if err != nil {
		log.Fatal(err)
	}

	_, err = engine.Execute("CREATE TABLE orders (user_id INT)")
	if err != nil {
		log.Fatal(err)
	}

	// Insert data
	_, err = engine.Execute("INSERT INTO users (id, name) VALUES (1, 'Alice')")
	if err != nil {
		log.Fatal(err)
	}
	_, err = engine.Execute("INSERT INTO orders (user_id) VALUES (1)")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Simple Correlation Test ===")

	// Test unqualified column reference first
	fmt.Println("\n--- Test with unqualified column ---")
	result, err := engine.Execute("SELECT name, (SELECT COUNT(*) FROM orders WHERE user_id = id) AS order_count FROM users")
	if err != nil {
		fmt.Printf("Error with unqualified: %v\n", err)
	} else {
		selectResult := result.(*mist.SelectResult)
		fmt.Printf("Success: %v\n", selectResult.Rows)
	}

	// Test qualified column reference  
	fmt.Println("\n--- Test with qualified column (table name) ---")
	result, err = engine.Execute("SELECT name, (SELECT COUNT(*) FROM orders WHERE user_id = users.id) AS order_count FROM users")
	if err != nil {
		fmt.Printf("Error with qualified table name: %v\n", err)
	} else {
		selectResult := result.(*mist.SelectResult)
		fmt.Printf("Success: %v\n", selectResult.Rows)
	}

	// Test alias reference
	fmt.Println("\n--- Test with alias reference ---")
	result, err = engine.Execute("SELECT u.name, (SELECT COUNT(*) FROM orders WHERE user_id = u.id) AS order_count FROM users u")
	if err != nil {
		fmt.Printf("Error with alias: %v\n", err)
	} else {
		selectResult := result.(*mist.SelectResult)
		fmt.Printf("Success: %v\n", selectResult.Rows)
	}

	// Test EXISTS correlation as baseline
	fmt.Println("\n--- Test EXISTS correlation ---")
	result, err = engine.Execute("SELECT name FROM users u WHERE EXISTS (SELECT 1 FROM orders WHERE user_id = u.id)")
	if err != nil {
		fmt.Printf("Error with EXISTS: %v\n", err)
	} else {
		selectResult := result.(*mist.SelectResult)
		fmt.Printf("EXISTS Success: %v\n", selectResult.Rows)
	}
}