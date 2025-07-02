package main

import (
	"fmt"
	"log"

	"github.com/abbychau/mist"
)

func main() {
	fmt.Println("=== Testing Column Name Conflict ===")

	// Test with same column name 'id' in both tables (failing case)
	fmt.Println("\n--- Test 1: Same column name 'id' in both tables ---")
	engine1 := mist.NewSQLEngine()
	
	_, err := engine1.Execute("CREATE TABLE users (id INT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(100))")
	if err != nil {
		log.Fatal(err)
	}
	_, err = engine1.Execute("CREATE TABLE orders (id INT PRIMARY KEY AUTO_INCREMENT, user_id INT)")
	if err != nil {
		log.Fatal(err)
	}
	_, err = engine1.Execute("INSERT INTO users (name) VALUES ('Alice'), ('Bob')")
	if err != nil {
		log.Fatal(err)
	}
	_, err = engine1.Execute("INSERT INTO orders (user_id) VALUES (1)")
	if err != nil {
		log.Fatal(err)
	}

	result, err := engine1.Execute("SELECT u.name, (SELECT COUNT(*) FROM orders WHERE user_id = u.id) AS count FROM users u")
	if err != nil {
		log.Fatal(err)
	}
	selectResult := result.(*mist.SelectResult)
	for _, row := range selectResult.Rows {
		fmt.Printf("  %v: %v orders\n", row[0], row[1])
	}

	// Test with different column names (should work)
	fmt.Println("\n--- Test 2: Different column names ---")
	engine2 := mist.NewSQLEngine()
	
	_, err = engine2.Execute("CREATE TABLE users (user_id INT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(100))")
	if err != nil {
		log.Fatal(err)
	}
	_, err = engine2.Execute("CREATE TABLE orders (order_id INT PRIMARY KEY AUTO_INCREMENT, user_id INT)")
	if err != nil {
		log.Fatal(err)
	}
	_, err = engine2.Execute("INSERT INTO users (name) VALUES ('Alice'), ('Bob')")
	if err != nil {
		log.Fatal(err)
	}
	_, err = engine2.Execute("INSERT INTO orders (user_id) VALUES (1)")
	if err != nil {
		log.Fatal(err)
	}

	result, err = engine2.Execute("SELECT u.name, (SELECT COUNT(*) FROM orders WHERE user_id = u.user_id) AS count FROM users u")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	for _, row := range selectResult.Rows {
		fmt.Printf("  %v: %v orders\n", row[0], row[1])
	}

	fmt.Println("\nIf Test 1 fails and Test 2 works, then the column name conflict is the issue.")
}