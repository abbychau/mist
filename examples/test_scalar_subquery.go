package main

import (
	"fmt"

	"github.com/abbychau/mist"
)

func main() {
	engine := mist.NewSQLEngine()

	// Create test tables
	fmt.Println("=== Setting up test tables ===")
	_, err := engine.Execute("CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50), department_id INT)")
	if err != nil {
		fmt.Printf("Error creating users table: %v\n", err)
		return
	}
	
	_, err = engine.Execute("CREATE TABLE departments (id INT PRIMARY KEY, name VARCHAR(50), budget DECIMAL(10,2))")
	if err != nil {
		fmt.Printf("Error creating departments table: %v\n", err)
		return
	}

	// Insert test data
	_, err = engine.Execute("INSERT INTO departments VALUES (1, 'Engineering', 100000.00), (2, 'Sales', 50000.00)")
	if err != nil {
		fmt.Printf("Error inserting departments: %v\n", err)
		return
	}
	
	_, err = engine.Execute("INSERT INTO users VALUES (1, 'Alice', 1), (2, 'Bob', 2), (3, 'Charlie', 1)")
	if err != nil {
		fmt.Printf("Error inserting users: %v\n", err)
		return
	}

	fmt.Println("Tables created and populated successfully")

	// Test scalar subqueries that should be supported
	fmt.Println("\n=== Testing Scalar Subqueries ===")
	
	testQueries := []struct {
		name string
		sql  string
	}{
		{"Scalar subquery in SELECT", "SELECT name, (SELECT name FROM departments WHERE id = users.department_id) as dept_name FROM users;"},
		{"Scalar subquery in WHERE", "SELECT * FROM users WHERE department_id = (SELECT id FROM departments WHERE name = 'Engineering');"},
		{"Scalar subquery with aggregate", "SELECT name FROM users WHERE department_id = (SELECT id FROM departments WHERE budget = (SELECT MAX(budget) FROM departments));"},
		{"Multiple scalar subqueries", "SELECT name, (SELECT name FROM departments WHERE id = users.department_id) as dept, (SELECT budget FROM departments WHERE id = users.department_id) as budget FROM users;"},
	}

	for _, test := range testQueries {
		fmt.Printf("\n--- %s ---\n", test.name)
		fmt.Printf("SQL: %s\n", test.sql)
		
		result, err := engine.Execute(test.sql)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Printf("Result: %v\n", result)
		}
	}
}