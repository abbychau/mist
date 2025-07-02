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

	fmt.Println("=== Testing Scalar Subqueries in SELECT Clauses ===")

	// Test basic scalar subquery in SELECT
	tests := []struct {
		name        string
		query       string
		description string
	}{
		{
			"Basic scalar subquery",
			"SELECT name, (SELECT MAX(salary) FROM employees) AS max_salary FROM departments",
			"Get department names with max salary from employees table",
		},
		{
			"Scalar subquery with WHERE",
			"SELECT name, (SELECT COUNT(*) FROM employees WHERE dept_id = 1) AS emp_count FROM departments WHERE id = 1",
			"Get department name with employee count for specific department",
		},
		{
			"Multiple scalar subqueries",
			"SELECT name, (SELECT MAX(salary) FROM employees) AS max_sal, (SELECT MIN(salary) FROM employees) AS min_sal FROM departments LIMIT 1",
			"Get department with max and min salaries",
		},
		{
			"Scalar subquery with aggregates",
			"SELECT name, (SELECT AVG(salary) FROM employees) AS avg_salary FROM departments",
			"Get departments with average salary",
		},
		{
			"NULL handling subquery",
			"SELECT name, (SELECT salary FROM employees WHERE name = 'NonExistent') AS missing_salary FROM departments LIMIT 1",
			"Test NULL handling when subquery returns no rows",
		},
		{
			"Complex scalar subquery",
			"SELECT name, (SELECT COUNT(*) FROM employees WHERE salary > 60000) AS high_earners FROM departments",
			"Count high earners across all departments",
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

	fmt.Println("\n=== Testing Subqueries with JOINs ===")

	joinTests := []struct {
		name        string
		query       string
		description string
	}{
		{
			"Scalar subquery in JOIN SELECT",
			"SELECT d.name, e.name, (SELECT MAX(salary) FROM employees) AS max_salary FROM departments d JOIN employees e ON d.id = e.dept_id",
			"JOIN with scalar subquery in SELECT clause",
		},
		{
			"Multiple subqueries in JOIN",
			"SELECT d.name, (SELECT COUNT(*) FROM employees) AS total_employees, (SELECT AVG(salary) FROM employees) AS avg_salary FROM departments d LIMIT 2",
			"Department info with employee statistics",
		},
	}

	for _, test := range joinTests {
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

	fmt.Println("\n=== Testing Error Cases ===")

	errorTests := []struct {
		name        string
		query       string
		description string
	}{
		{
			"Multiple rows error",
			"SELECT name, (SELECT salary FROM employees) AS all_salaries FROM departments LIMIT 1",
			"Should fail: subquery returns multiple rows",
		},
		{
			"Multiple columns error", 
			"SELECT name, (SELECT name, salary FROM employees LIMIT 1) AS emp_info FROM departments LIMIT 1",
			"Should fail: subquery returns multiple columns",
		},
	}

	for _, test := range errorTests {
		fmt.Printf("\n--- %s ---\n", test.name)
		fmt.Printf("Description: %s\n", test.description)
		fmt.Printf("Query: %s\n", test.query)
		
		result, err := engine.Execute(test.query)
		if err != nil {
			fmt.Printf("✅ Expected error: %v\n", err)
		} else {
			fmt.Printf("❌ Expected error but got result: %v\n", result)
		}
	}

	fmt.Println("\n=== SELECT Subquery Testing Complete! ===")
}

func createTestTables(engine *mist.SQLEngine) error {
	// Create departments table
	_, err := engine.Execute("CREATE TABLE departments (id INT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(255))")
	if err != nil {
		return err
	}

	// Create employees table
	_, err = engine.Execute("CREATE TABLE employees (id INT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(255), salary FLOAT, dept_id INT)")
	if err != nil {
		return err
	}

	// Insert test data
	testData := []string{
		// Departments
		"INSERT INTO departments (name) VALUES ('Engineering')",
		"INSERT INTO departments (name) VALUES ('Sales')",
		"INSERT INTO departments (name) VALUES ('Marketing')",

		// Employees
		"INSERT INTO employees (name, salary, dept_id) VALUES ('Alice', 75000, 1)",
		"INSERT INTO employees (name, salary, dept_id) VALUES ('Bob', 65000, 1)",
		"INSERT INTO employees (name, salary, dept_id) VALUES ('Charlie', 55000, 2)",
		"INSERT INTO employees (name, salary, dept_id) VALUES ('Diana', 70000, 2)",
		"INSERT INTO employees (name, salary, dept_id) VALUES ('Eve', 60000, 3)",
	}

	for _, query := range testData {
		_, err := engine.Execute(query)
		if err != nil {
			return fmt.Errorf("error inserting test data: %v", err)
		}
	}

	return nil
}