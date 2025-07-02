package main

import (
	"fmt"
	"log"

	"github.com/abbychau/mist"
)

func main() {
	// Create a new engine
	engine := mist.NewSQLEngine()

	// Create tables
	_, err := engine.Execute("CREATE TABLE users (id INT, name VARCHAR(100))")
	if err != nil {
		log.Fatal(err)
	}

	_, err = engine.Execute("CREATE TABLE orders (user_id INT)")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Debug Table Structure ===")

	// Check table structure
	fmt.Println("\n--- Table structures ---")
	result, err := engine.Execute("SHOW TABLES")
	if err != nil {
		log.Fatal(err)
	}
	selectResult := result.(*mist.SelectResult)
	fmt.Printf("Tables: %v\n", selectResult.Rows)

	// Let's try a simple query to see column resolution
	fmt.Println("\n--- Simple column resolution test ---")
	result, err = engine.Execute("SELECT id FROM users")
	if err != nil {
		fmt.Printf("Error selecting id from users: %v\n", err)
	} else {
		fmt.Printf("SELECT id FROM users works\n")
	}

	result, err = engine.Execute("SELECT user_id FROM orders")
	if err != nil {
		fmt.Printf("Error selecting user_id from orders: %v\n", err)
	} else {
		fmt.Printf("SELECT user_id FROM orders works\n")
	}

	// Insert some data for testing
	_, err = engine.Execute("INSERT INTO users (id, name) VALUES (1, 'Alice')")
	if err != nil {
		log.Fatal(err)
	}

	_, err = engine.Execute("INSERT INTO orders (user_id) VALUES (1)")
	if err != nil {
		log.Fatal(err)
	}

	// Test qualified references
	fmt.Println("\n--- Qualified reference test ---")
	result, err = engine.Execute("SELECT users.id FROM users")
	if err != nil {
		fmt.Printf("Error with users.id: %v\n", err)
	} else {
		selectResult := result.(*mist.SelectResult)
		fmt.Printf("users.id works: %v\n", selectResult.Rows)
	}

	result, err = engine.Execute("SELECT u.id FROM users u")
	if err != nil {
		fmt.Printf("Error with u.id: %v\n", err)
	} else {
		selectResult := result.(*mist.SelectResult)
		fmt.Printf("u.id works: %v\n", selectResult.Rows)
	}

	// Test the problematic query step by step
	fmt.Println("\n--- Step by step subquery test ---")
	
	// This should work (no correlation)
	result, err = engine.Execute("SELECT COUNT(*) FROM orders WHERE user_id = 1")
	if err != nil {
		fmt.Printf("Error with hardcoded subquery: %v\n", err)
	} else {
		selectResult := result.(*mist.SelectResult)
		fmt.Printf("Hardcoded subquery works: %v\n", selectResult.Rows)
	}

	// This is where it fails (correlation)
	result, err = engine.Execute("SELECT (SELECT COUNT(*) FROM orders WHERE user_id = id) FROM users")
	if err != nil {
		fmt.Printf("Error with correlated subquery: %v\n", err)
	} else {
		selectResult := result.(*mist.SelectResult)
		fmt.Printf("Correlated subquery works: %v\n", selectResult.Rows)
	}
}