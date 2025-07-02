package main

import (
	"fmt"
	"log"

	"github.com/abbychau/mist"
)

func main() {
	fmt.Println("=== Testing AUTO_INCREMENT Impact ===")

	// Test 1: Without AUTO_INCREMENT in orders table (should work)
	fmt.Println("\n--- Test 1: Orders table WITHOUT AUTO_INCREMENT ---")
	engine1 := mist.NewSQLEngine()
	
	_, err := engine1.Execute("CREATE TABLE users (id INT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(100))")
	if err != nil {
		log.Fatal(err)
	}
	_, err = engine1.Execute("CREATE TABLE orders (user_id INT, amount DECIMAL(10,2))")
	if err != nil {
		log.Fatal(err)
	}
	_, err = engine1.Execute("INSERT INTO users (name) VALUES ('Alice'), ('Bob')")
	if err != nil {
		log.Fatal(err)
	}
	_, err = engine1.Execute("INSERT INTO orders (user_id, amount) VALUES (1, 100.00)")
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

	// Test 2: With AUTO_INCREMENT in orders table (fails)
	fmt.Println("\n--- Test 2: Orders table WITH AUTO_INCREMENT ---")
	engine2 := mist.NewSQLEngine()
	
	_, err = engine2.Execute("CREATE TABLE users (id INT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(100))")
	if err != nil {
		log.Fatal(err)
	}
	_, err = engine2.Execute("CREATE TABLE orders (id INT PRIMARY KEY AUTO_INCREMENT, user_id INT, amount DECIMAL(10,2))")
	if err != nil {
		log.Fatal(err)
	}
	_, err = engine2.Execute("INSERT INTO users (name) VALUES ('Alice'), ('Bob')")
	if err != nil {
		log.Fatal(err)
	}
	_, err = engine2.Execute("INSERT INTO orders (user_id, amount) VALUES (1, 100.00)")
	if err != nil {
		log.Fatal(err)
	}

	result, err = engine2.Execute("SELECT u.name, (SELECT COUNT(*) FROM orders WHERE user_id = u.id) AS count FROM users u")
	if err != nil {
		log.Fatal(err)
	}
	selectResult = result.(*mist.SelectResult)
	for _, row := range selectResult.Rows {
		fmt.Printf("  %v: %v orders\n", row[0], row[1])
	}

	fmt.Println("\nIf Test 1 shows Alice: 1, Bob: 0 and Test 2 shows Alice: 1, Bob: 1,")
	fmt.Println("then the AUTO_INCREMENT column in orders table is causing the issue.")
}