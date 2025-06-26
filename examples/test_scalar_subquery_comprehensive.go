package main

import (
	"fmt"

	"github.com/abbychau/mist/mist"
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

	_, err = engine.Execute("CREATE TABLE salaries (user_id INT, amount DECIMAL(10,2))")
	if err != nil {
		fmt.Printf("Error creating salaries table: %v\n", err)
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

	_, err = engine.Execute("INSERT INTO salaries VALUES (1, 75000.00), (2, 60000.00), (3, 80000.00)")
	if err != nil {
		fmt.Printf("Error inserting salaries: %v\n", err)
		return
	}

	fmt.Println("Tables created and populated successfully")

	// Test scalar subqueries - focusing on non-correlated ones that should work
	fmt.Println("\n=== Testing Scalar Subqueries (Non-correlated) ===")
	
	nonCorrelatedTests := []struct {
		name string
		sql  string
	}{
		{"Simple scalar in WHERE", "SELECT * FROM users WHERE department_id = (SELECT id FROM departments WHERE name = 'Engineering');"},
		{"Scalar with aggregate in WHERE", "SELECT name FROM users WHERE department_id = (SELECT id FROM departments WHERE budget = (SELECT MAX(budget) FROM departments));"},
		{"Scalar subquery in SELECT (constant)", "SELECT name, (SELECT MAX(budget) FROM departments) as max_budget FROM users;"},
		{"Scalar subquery with COUNT", "SELECT name FROM users WHERE id <= (SELECT COUNT(*) FROM departments);"},
		{"Nested scalar subqueries", "SELECT * FROM users WHERE department_id = (SELECT id FROM departments WHERE budget > (SELECT AVG(budget) FROM departments));"},
		{"Scalar returning NULL", "SELECT * FROM users WHERE department_id = (SELECT id FROM departments WHERE name = 'NonExistent');"},
	}

	for _, test := range nonCorrelatedTests {
		fmt.Printf("\n--- %s ---\n", test.name)
		fmt.Printf("SQL: %s\n", test.sql)
		
		result, err := engine.Execute(test.sql)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Printf("Result: %v\n", result)
		}
	}

	// Test error cases
	fmt.Println("\n=== Testing Error Cases ===")
	
	errorTests := []struct {
		name string
		sql  string
	}{
		{"Multiple rows returned", "SELECT * FROM users WHERE department_id = (SELECT id FROM departments);"},
		{"Multiple columns returned", "SELECT * FROM users WHERE department_id = (SELECT id, name FROM departments WHERE id = 1);"},
	}

	for _, test := range errorTests {
		fmt.Printf("\n--- %s ---\n", test.name)
		fmt.Printf("SQL: %s\n", test.sql)
		
		result, err := engine.Execute(test.sql)
		if err != nil {
			fmt.Printf("Expected Error: %v\n", err)
		} else {
			fmt.Printf("Unexpected Success: %v\n", result)
		}
	}
}