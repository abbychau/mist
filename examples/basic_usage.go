package main

import (
	"fmt"
	"log"

	"github.com/abbychau/mist/mist"
)

func main() {
	// Create a new SQL engine
	engine := mist.NewSQLEngine()

	fmt.Println("=== Mist Database Library Example ===")
	fmt.Printf("Version: %s\n\n", mist.Version())

	// Create tables
	fmt.Println("1. Creating tables...")
	_, err := engine.Execute("CREATE TABLE departments (id INT, name VARCHAR(50))")
	if err != nil {
		log.Fatal(err)
	}

	_, err = engine.Execute("CREATE TABLE users (id INT, name VARCHAR(50), dept_id INT, salary FLOAT)")
	if err != nil {
		log.Fatal(err)
	}

	// Create indexes for better performance
	fmt.Println("2. Creating indexes...")
	_, err = engine.Execute("CREATE INDEX idx_dept ON users (dept_id)")
	if err != nil {
		log.Fatal(err)
	}

	// Insert sample data
	fmt.Println("3. Inserting sample data...")
	sampleData := []string{
		"INSERT INTO departments VALUES (1, 'Engineering'), (2, 'Marketing'), (3, 'Sales')",
		"INSERT INTO users VALUES (1, 'Alice', 1, 75000), (2, 'Bob', 2, 65000), (3, 'Charlie', 1, 85000)",
		"INSERT INTO users VALUES (4, 'Diana', 2, 70000), (5, 'Eve', 3, 60000)",
	}

	for _, query := range sampleData {
		_, err := engine.Execute(query)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Demonstrate various query types
	fmt.Println("\n4. Running queries...")

	// Basic SELECT
	fmt.Println("\n--- All Users ---")
	result, err := engine.Execute("SELECT * FROM users")
	if err != nil {
		log.Fatal(err)
	}
	mist.PrintResult(result)

	// Aggregate functions
	fmt.Println("\n--- Database Statistics ---")
	queries := map[string]string{
		"Total Users":    "SELECT COUNT(*) FROM users",
		"Average Salary": "SELECT AVG(salary) FROM users",
		"Highest Salary": "SELECT MAX(salary) FROM users",
		"Lowest Salary":  "SELECT MIN(salary) FROM users",
	}

	for description, query := range queries {
		fmt.Printf("\n%s:\n", description)
		result, err := engine.Execute(query)
		if err != nil {
			log.Fatal(err)
		}
		mist.PrintResult(result)
	}

	// JOIN operations
	fmt.Println("\n--- JOIN Examples ---")

	// Explicit JOIN
	fmt.Println("\nExplicit JOIN:")
	result, err = engine.Execute(`
		SELECT u.name, d.name, u.salary 
		FROM users u 
		JOIN departments d ON u.dept_id = d.id 
		WHERE u.salary > 65000
	`)
	if err != nil {
		log.Fatal(err)
	}
	mist.PrintResult(result)

	// Comma-separated table join
	fmt.Println("\nComma-separated table join:")
	result, err = engine.Execute(`
		SELECT users.name, departments.name, users.salary 
		FROM users, departments 
		WHERE users.dept_id = departments.id AND users.salary > 70000
	`)
	if err != nil {
		log.Fatal(err)
	}
	mist.PrintResult(result)

	// LIMIT with offset
	fmt.Println("\n--- LIMIT Examples ---")
	result, err = engine.Execute("SELECT name, salary FROM users LIMIT 2, 2")
	if err != nil {
		log.Fatal(err)
	}
	mist.PrintResult(result)

	// Subquery
	fmt.Println("\n--- Subquery Example ---")
	result, err = engine.Execute(`
		SELECT name, salary 
		FROM (SELECT * FROM users WHERE dept_id = 1) AS eng_users 
		WHERE salary > 75000
	`)
	if err != nil {
		log.Fatal(err)
	}
	mist.PrintResult(result)

	// Batch operations
	fmt.Println("\n5. Batch operations...")
	batchSQL := `
		UPDATE users SET salary = salary * 1.1 WHERE dept_id = 1;
		SELECT name, salary FROM users WHERE dept_id = 1;
	`

	results, err := engine.ExecuteMultiple(batchSQL)
	if err != nil {
		log.Fatal(err)
	}

	for i, result := range results {
		fmt.Printf("\nBatch result %d:\n", i+1)
		mist.PrintResult(result)
	}

	// Show database metadata
	fmt.Println("\n6. Database metadata...")
	db := engine.GetDatabase()
	tables := db.ListTables()
	fmt.Printf("Available tables: %v\n", tables)

	// Show indexes
	fmt.Println("\nIndexes:")
	result, err = engine.Execute("SHOW INDEX FROM users")
	if err != nil {
		log.Fatal(err)
	}
	mist.PrintResult(result)

	fmt.Println("\n=== Example completed successfully! ===")
}
