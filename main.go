package main

import (
	"fmt"
	"os"

	"github.com/abbychau/mist/mist"
)

func main() {
	// Create a new SQL engine
	engine := mist.NewSQLEngine()

	// Check if user wants interactive mode
	if len(os.Args) > 1 && os.Args[1] == "-i" {
		mist.Interactive(engine)
		return
	}

	// Run demo by default
	runDemo(engine)

	// Demo the recording functionality
	runRecordingDemo(engine)
}

func runDemo(engine *mist.SQLEngine) {
	// Example usage
	fmt.Println("=== Mist In-Memory MySQL Database Demo ===")
	fmt.Println("Now with ALTER TABLE, LIMIT, SUBQUERIES, AGGREGATES, and INDEXES!")
	fmt.Println()

	// Create tables
	fmt.Println("Creating tables...")
	tables := []string{
		"CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50), age INT, department_id INT, salary FLOAT)",
		"CREATE TABLE departments (id INT PRIMARY KEY, name VARCHAR(50), budget FLOAT)",
	}

	for _, query := range tables {
		result, err := engine.Execute(query)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Println(result)
	}
	fmt.Println()

	// Create indexes for performance
	fmt.Println("Creating indexes...")
	indexes := []string{
		"CREATE INDEX idx_age ON users (age)",
		"CREATE INDEX idx_dept ON users (department_id)",
	}

	for _, query := range indexes {
		result, err := engine.Execute(query)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Println(result)
	}
	fmt.Println()

	// Insert data
	fmt.Println("Inserting data...")
	queries := []string{
		"INSERT INTO departments VALUES (1, 'Engineering', 100000.0)",
		"INSERT INTO departments VALUES (2, 'Marketing', 75000.0)",
		"INSERT INTO users VALUES (1, 'Alice', 30, 1, 85000.0)",
		"INSERT INTO users VALUES (2, 'Bob', 25, 2, 65000.0)",
		"INSERT INTO users VALUES (3, 'Charlie', 35, 1, 95000.0)",
		"INSERT INTO users VALUES (4, 'Diana', 28, 2, 70000.0)",
		"INSERT INTO users VALUES (5, 'Eve', 32, 1, 90000.0)",
	}

	for _, query := range queries {
		result, err := engine.Execute(query)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		fmt.Println(result)
	}
	fmt.Println()

	// Demonstrate AGGREGATE functions
	fmt.Println("AGGREGATES - Database statistics:")
	aggregateQueries := []string{
		"SELECT COUNT(*) FROM users",
		"SELECT AVG(age) FROM users",
		"SELECT SUM(salary) FROM users",
		"SELECT MIN(salary), MAX(salary) FROM users",
	}

	for _, query := range aggregateQueries {
		fmt.Printf("Query: %s\n", query)
		result, err := engine.Execute(query)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		mist.PrintResult(result)
		fmt.Println()
	}

	// Demonstrate INDEX usage (optimized query)
	fmt.Println("INDEX-OPTIMIZED QUERY - Users over 30 (using idx_age):")
	result, err := engine.Execute("SELECT name, age FROM users WHERE age > 30")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	mist.PrintResult(result)
	fmt.Println()

	// Demonstrate JOIN with aggregates
	fmt.Println("JOIN + AGGREGATES - Count of users by department:")
	result, err = engine.Execute("SELECT COUNT(*) FROM users JOIN departments ON users.department_id = departments.id")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	mist.PrintResult(result)
	fmt.Println()

	// Regular JOIN without aggregates
	fmt.Println("JOIN - Users with their departments:")
	result, err = engine.Execute("SELECT users.name, departments.name FROM users JOIN departments ON users.department_id = departments.id")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	mist.PrintResult(result)
	fmt.Println()

	// Show indexes
	fmt.Println("SHOW INDEXES for users table:")
	result, err = engine.Execute("SHOW INDEX FROM users")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	mist.PrintResult(result)
	fmt.Println()

	// Demonstrate UPDATE with arithmetic
	fmt.Println("UPDATE - 10% salary increase for Engineering:")
	result, err = engine.Execute("UPDATE users SET salary = salary * 1.1 WHERE department_id = 1")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println(result)

	// check if update was successful
	fmt.Println("Checking updated salaries:")
	result, err = engine.Execute("SELECT name, salary FROM users WHERE department_id = 1")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	//salary should of Alice should be 85000*1.1 = 93500.0 now
	aliceSalaryFromDB := result.(*mist.SelectResult).Rows[0][1]
	if aliceSalaryFromDB != 93500.00000000001 {
		// 93500.00000000001 is the expected value due to floating point precision
		//red emoji
		fmt.Println("❌")
		fmt.Println("Error: Alice's salary was not updated correctly!")
		fmt.Println("Expected: 93500.00000000001, Got: ", aliceSalaryFromDB)
		return
	}

	// Show updated salaries with LIMIT
	fmt.Println("Updated salaries (top 2):")
	result, err = engine.Execute("SELECT name, salary FROM users WHERE department_id = 1 LIMIT 2")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	mist.PrintResult(result)
	fmt.Println()

	// Demonstrate ALTER TABLE
	fmt.Println("ALTER TABLE - Adding email column:")
	result, err = engine.Execute("ALTER TABLE users ADD COLUMN email VARCHAR(100)")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println(result)

	// Update with email data
	fmt.Println("Adding email data:")
	emailUpdates := []string{
		"UPDATE users SET email = 'alice@company.com' WHERE name = 'Alice'",
		"UPDATE users SET email = 'charlie@company.com' WHERE name = 'Charlie'",
		"UPDATE users SET email = 'eve@company.com' WHERE name = 'Eve'",
	}

	for _, query := range emailUpdates {
		_, err = engine.Execute(query)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
	}

	// Demonstrate subquery
	fmt.Println("SUBQUERY - High earners from Engineering:")
	result, err = engine.Execute("SELECT name, salary FROM (SELECT * FROM users WHERE department_id = 1) AS eng_users WHERE salary > 90000")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	mist.PrintResult(result)
	fmt.Println()

	// Demonstrate LIMIT with offset
	fmt.Println("LIMIT with offset - Users 2-3:")
	result, err = engine.Execute("SELECT name, email FROM users LIMIT 1, 2")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	mist.PrintResult(result)
	fmt.Println()

	// Demonstrate query join without JOIN keyword
	fmt.Println("Query join without JOIN keyword - Users and departments:")
	result, err = engine.Execute("SELECT users.name, departments.name FROM users, departments WHERE users.department_id = departments.id")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	mist.PrintResult(result)
	fmt.Println()

	fmt.Println("Demo completed!")
	fmt.Println("Features demonstrated:")
	fmt.Println("??CREATE TABLE with multiple data types")
	fmt.Println("??ALTER TABLE (ADD COLUMN, DROP COLUMN, MODIFY COLUMN)")
	fmt.Println("??CREATE INDEX for query optimization")
	fmt.Println("??INSERT with data validation")
	fmt.Println("??SELECT with WHERE clauses (index-optimized)")
	fmt.Println("??LIMIT clause with offset and count")
	fmt.Println("??SUBQUERIES in FROM clause")
	fmt.Println("??AGGREGATE functions (COUNT, SUM, AVG, MIN, MAX)")
	fmt.Println("??JOIN operations between tables")
	fmt.Println("??UPDATE with arithmetic expressions")
	fmt.Println("??SHOW TABLES and SHOW INDEX commands")
	fmt.Println()
	fmt.Println("Run with '-i' flag for interactive mode: go run . -i")
}

func runRecordingDemo(engine *mist.SQLEngine) {
	fmt.Println("=== Recording Demo ===")
	fmt.Println("Demonstrating query recording functionality...")
	fmt.Println()

	// Start recording
	fmt.Println("Starting query recording...")
	engine.StartRecording()

	// Execute some queries while recording
	queries := []string{
		"CREATE TABLE test_recording (id INT, message VARCHAR(100))",
		"INSERT INTO test_recording VALUES (1, 'First recorded query')",
		"INSERT INTO test_recording VALUES (2, 'Second recorded query')",
		"SELECT * FROM test_recording",
		"UPDATE test_recording SET message = 'Updated message' WHERE id = 1",
		"SELECT * FROM test_recording WHERE id = 1",
	}

	fmt.Println("Executing queries while recording:")
	for _, query := range queries {
		fmt.Printf("  %s\n", query)
		result, err := engine.Execute(query)
		if err != nil {
			fmt.Printf("    Error: %v\n", err)
		} else {
			// Don't print full results to keep output clean
			switch r := result.(type) {
			case *mist.SelectResult:
				fmt.Printf("    -> %d rows returned\n", len(r.Rows))
			default:
				fmt.Printf("    -> %v\n", result)
			}
		}
	}
	fmt.Println()

	// Stop recording
	fmt.Println("Stopping query recording...")
	engine.EndRecording()

	// Get recorded queries
	recordedQueries := engine.GetRecordedQueries()
	fmt.Printf("Recorded %d queries:\n", len(recordedQueries))
	for i, query := range recordedQueries {
		fmt.Printf("  %d. %s\n", i+1, query)
	}
	fmt.Println()

	// Execute a query after recording stopped (should not be recorded)
	fmt.Println("Executing query after recording stopped (should not be recorded):")
	_, _ = engine.Execute("SELECT COUNT(*) FROM test_recording")

	// Check recorded queries again
	finalRecordedQueries := engine.GetRecordedQueries()
	fmt.Printf("Final count of recorded queries: %d (should be same as before)\n", len(finalRecordedQueries))
	fmt.Println()
}
