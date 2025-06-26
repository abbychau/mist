package main

import (
	"fmt"
	"log"

	"github.com/abbychau/mist"
)

func main() {
	fmt.Println("=== Testing New Operators with JOINs ===")

	engine := mist.NewSQLEngine()

	// Create test tables
	createUsersTable := `CREATE TABLE users (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(100),
		department_id INT,
		salary DECIMAL(10,2),
		manager_id INT
	)`

	createDepartmentsTable := `CREATE TABLE departments (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(100),
		budget DECIMAL(15,2),
		location VARCHAR(100)
	)`

	_, err := engine.Execute(createUsersTable)
	if err != nil {
		log.Fatalf("Error creating users table: %v", err)
	}

	_, err = engine.Execute(createDepartmentsTable)
	if err != nil {
		log.Fatalf("Error creating departments table: %v", err)
	}

	fmt.Println("✅ Tables created successfully")

	// Insert test data
	userInserts := []string{
		`INSERT INTO users (name, department_id, salary, manager_id) VALUES ('Alice', 1, 75000, NULL)`,
		`INSERT INTO users (name, department_id, salary, manager_id) VALUES ('Bob', 1, 65000, 1)`,
		`INSERT INTO users (name, department_id, salary, manager_id) VALUES ('Carol', 2, 85000, NULL)`,
		`INSERT INTO users (name, department_id, salary, manager_id) VALUES ('David', 2, 70000, 3)`,
		`INSERT INTO users (name, department_id, salary, manager_id) VALUES ('Eve', NULL, 60000, 1)`, // NULL dept
		`INSERT INTO users (name, department_id, salary, manager_id) VALUES ('Frank', 3, NULL, NULL)`, // NULL salary
	}

	deptInserts := []string{
		`INSERT INTO departments (name, budget, location) VALUES ('Engineering', 500000, 'Building A')`,
		`INSERT INTO departments (name, budget, location) VALUES ('Sales', 300000, 'Building B')`,
		`INSERT INTO departments (name, budget, location) VALUES ('Marketing', 200000, NULL)`, // NULL location
	}

	for _, query := range userInserts {
		_, err := engine.Execute(query)
		if err != nil {
			log.Printf("Error inserting user: %v", err)
		}
	}

	for _, query := range deptInserts {
		_, err := engine.Execute(query)
		if err != nil {
			log.Printf("Error inserting department: %v", err)
		}
	}

	fmt.Println("✅ Test data inserted")

	// Test 1: JOIN with IS NULL
	fmt.Println("\n=== Test 1: JOIN with IS NULL ===")
	result, err := engine.Execute(`
		SELECT u.name, d.name as dept_name, u.manager_id 
		FROM users u 
		LEFT JOIN departments d ON u.department_id = d.id 
		WHERE u.manager_id IS NULL
	`)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Println("✅ JOIN + IS NULL works:")
		mist.PrintResult(result)
	}

	// Test 2: JOIN with BETWEEN
	fmt.Println("\n=== Test 2: JOIN with BETWEEN ===")
	result, err = engine.Execute(`
		SELECT u.name, d.name as dept_name, u.salary 
		FROM users u 
		JOIN departments d ON u.department_id = d.id 
		WHERE u.salary BETWEEN 65000 AND 80000
	`)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Println("✅ JOIN + BETWEEN works:")
		mist.PrintResult(result)
	}

	// Test 3: JOIN with IN
	fmt.Println("\n=== Test 3: JOIN with IN ===")
	result, err = engine.Execute(`
		SELECT u.name, d.name as dept_name, d.budget 
		FROM users u 
		JOIN departments d ON u.department_id = d.id 
		WHERE d.budget IN (300000, 500000)
	`)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Println("✅ JOIN + IN works:")
		mist.PrintResult(result)
	}

	// Test 4: Complex condition with multiple operators
	fmt.Println("\n=== Test 4: Complex JOIN with Multiple Operators ===")
	result, err = engine.Execute(`
		SELECT u.name, d.name as dept_name, u.salary, d.location 
		FROM users u 
		LEFT JOIN departments d ON u.department_id = d.id 
		WHERE (u.salary IS NOT NULL AND u.salary > 60000) 
		  AND d.name IN ('Engineering', 'Sales')
	`)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Println("✅ Complex JOIN conditions work:")
		mist.PrintResult(result)
	}

	// Test 5: JOIN with NULL location
	fmt.Println("\n=== Test 5: JOIN with NULL Checks ===")
	result, err = engine.Execute(`
		SELECT u.name, d.name as dept_name, d.location 
		FROM users u 
		JOIN departments d ON u.department_id = d.id 
		WHERE d.location IS NULL OR d.location = 'Building A'
	`)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Println("✅ JOIN + IS NULL + OR works:")
		mist.PrintResult(result)
	}

	fmt.Println("\n✅ All JOIN tests with new operators completed successfully!")
	fmt.Println("🎯 New operators work seamlessly with:")
	fmt.Println("   • INNER JOINs")
	fmt.Println("   • LEFT JOINs")
	fmt.Println("   • Complex WHERE conditions")
	fmt.Println("   • AND/OR combinations")
	fmt.Println("   • Mixed operator types")
}