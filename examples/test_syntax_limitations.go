package main

import (
	"fmt"
	"log"

	"github.com/abbychau/mist"
)

func main() {
	fmt.Println("=== Syntax Limitations Demonstration for Mist Database ===")

	engine := mist.NewSQLEngine()

	// Create a test table
	createTableQuery := `CREATE TABLE users (
		id INT AUTO_INCREMENT PRIMARY KEY,
		first_name VARCHAR(50),
		last_name VARCHAR(50),
		age INT,
		salary DECIMAL(10,2),
		department VARCHAR(50),
		email VARCHAR(100)
	)`

	_, err := engine.Execute(createTableQuery)
	if err != nil {
		log.Fatalf("Error creating users table: %v", err)
	}
	fmt.Println("✅ Users table created successfully")

	// Insert sample data
	insertQueries := []string{
		`INSERT INTO users (first_name, last_name, age, salary, department, email) VALUES 
		 ('John', 'Doe', 30, 75000.00, 'Engineering', 'john.doe@company.com')`,
		`INSERT INTO users (first_name, last_name, age, salary, department, email) VALUES 
		 ('Jane', 'Smith', 25, 65000.00, 'Marketing', 'jane.smith@company.com')`,
		`INSERT INTO users (first_name, last_name, age, salary, department, email) VALUES 
		 ('Bob', 'Johnson', 35, 85000.00, 'Engineering', 'bob.johnson@company.com')`,
		`INSERT INTO users (first_name, last_name, age, salary, department, email) VALUES 
		 ('Alice', 'Brown', 28, 70000.00, 'Sales', 'alice.brown@company.com')`,
	}

	for i, query := range insertQueries {
		_, err := engine.Execute(query)
		if err != nil {
			log.Printf("Error inserting user %d: %v", i+1, err)
		} else {
			fmt.Printf("✅ User %d inserted successfully\n", i+1)
		}
	}

	fmt.Println("\n=== WHAT WORKS: Basic SELECT operations ===")

	// Test 1: Basic SELECT with column names and aliases
	fmt.Println("\n1. Basic column selection with aliases:")
	result, err := engine.Execute("SELECT first_name, last_name, age FROM users")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Println("✅ Works:")
		mist.PrintResult(result)
	}

	// Test 2: Aggregate functions (what works)
	fmt.Println("\n2. Aggregate functions:")
	result, err = engine.Execute("SELECT COUNT(*), AVG(salary), MAX(age) FROM users")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Println("✅ Works:")
		mist.PrintResult(result)
	}

	// Test 3: Basic WHERE conditions
	fmt.Println("\n3. Basic WHERE conditions:")
	result, err = engine.Execute("SELECT first_name, age FROM users WHERE age > 30")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Println("✅ Works:")
		mist.PrintResult(result)
	}

	fmt.Println("\n=== WHAT DOESN'T WORK: Advanced SELECT features ===")

	// Test 4: Functions in SELECT (should fail)
	fmt.Println("\n4. String functions in SELECT:")
	result, err = engine.Execute("SELECT UPPER(first_name), CONCAT(first_name, ' ', last_name) AS full_name FROM users")
	if err != nil {
		fmt.Printf("❌ Expected limitation: %v\n", err)
	} else {
		fmt.Println("✅ Unexpected success:")
		mist.PrintResult(result)
	}

	// Test 5: Calculated columns (should fail)
	fmt.Println("\n5. Calculated columns:")
	result, err = engine.Execute("SELECT first_name, salary, salary * 12 AS annual_salary FROM users")
	if err != nil {
		fmt.Printf("❌ Expected limitation: %v\n", err)
	} else {
		fmt.Println("✅ Unexpected success:")
		mist.PrintResult(result)
	}

	// Test 6: IF function (should fail)
	fmt.Println("\n6. IF function:")
	result, err = engine.Execute("SELECT first_name, IF(age >= 30, 'Senior', 'Junior') AS category FROM users")
	if err != nil {
		fmt.Printf("❌ Expected limitation: %v\n", err)
	} else {
		fmt.Println("✅ Unexpected success:")
		mist.PrintResult(result)
	}

	// Test 7: CASE expression (should fail)
	fmt.Println("\n7. CASE expressions:")
	result, err = engine.Execute("SELECT first_name, CASE WHEN age < 30 THEN 'Young' WHEN age < 35 THEN 'Middle' ELSE 'Senior' END AS age_group FROM users")
	if err != nil {
		fmt.Printf("❌ Expected limitation: %v\n", err)
	} else {
		fmt.Println("✅ Unexpected success:")
		mist.PrintResult(result)
	}

	fmt.Println("\n=== WHAT DOESN'T WORK: Advanced WHERE features ===")

	// Test 8: LIKE pattern matching (should fail)
	fmt.Println("\n8. LIKE pattern matching:")
	result, err = engine.Execute("SELECT first_name, email FROM users WHERE email LIKE '%@company.com'")
	if err != nil {
		fmt.Printf("❌ Expected limitation: %v\n", err)
	} else {
		fmt.Println("✅ Unexpected success:")
		mist.PrintResult(result)
	}

	// Test 9: IN operator (should fail)
	fmt.Println("\n9. IN operator:")
	result, err = engine.Execute("SELECT first_name, department FROM users WHERE department IN ('Engineering', 'Marketing')")
	if err != nil {
		fmt.Printf("❌ Expected limitation: %v\n", err)
	} else {
		fmt.Println("✅ Unexpected success:")
		mist.PrintResult(result)
	}

	// Test 10: BETWEEN operator (should fail)
	fmt.Println("\n10. BETWEEN operator:")
	result, err = engine.Execute("SELECT first_name, age FROM users WHERE age BETWEEN 25 AND 32")
	if err != nil {
		fmt.Printf("❌ Expected limitation: %v\n", err)
	} else {
		fmt.Println("✅ Unexpected success:")
		mist.PrintResult(result)
	}

	// Test 11: IS NULL (should fail)
	fmt.Println("\n11. IS NULL testing:")
	result, err = engine.Execute("SELECT first_name FROM users WHERE department IS NOT NULL")
	if err != nil {
		fmt.Printf("❌ Expected limitation: %v\n", err)
	} else {
		fmt.Println("✅ Unexpected success:")
		mist.PrintResult(result)
	}

	// Test 12: Subqueries in SELECT (should fail)
	fmt.Println("\n12. Scalar subqueries:")
	result, err = engine.Execute("SELECT first_name, (SELECT MAX(salary) FROM users) AS max_salary FROM users")
	if err != nil {
		fmt.Printf("❌ Expected limitation: %v\n", err)
	} else {
		fmt.Println("✅ Unexpected success:")
		mist.PrintResult(result)
	}

	fmt.Println("\n=== WORKAROUNDS: How to achieve similar results ===")

	// Workaround 1: Instead of CONCAT, use application logic
	fmt.Println("\n13. Workaround for CONCAT - fetch separately and combine in application:")
	result, err = engine.Execute("SELECT first_name, last_name FROM users")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Println("✅ Fetch separately, combine in application:")
		if selectResult, ok := result.(*mist.SelectResult); ok {
			fmt.Println("| full_name    |")
			fmt.Println("|--------------|")
			for _, row := range selectResult.Rows {
				firstName := row[0].(string)
				lastName := row[1].(string)
				fullName := firstName + " " + lastName
				fmt.Printf("| %-12s |\n", fullName)
			}
		}
	}

	// Workaround 2: Instead of LIKE, use exact matches or application filtering
	fmt.Println("\n14. Workaround for LIKE - use exact domain match:")
	result, err = engine.Execute("SELECT first_name, email FROM users") // Get all, filter in app
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Println("✅ Filter @company.com emails in application:")
		if selectResult, ok := result.(*mist.SelectResult); ok {
			fmt.Println("| first_name   | email                      |")
			fmt.Println("|--------------|----------------------------|")
			for _, row := range selectResult.Rows {
				firstName := row[0].(string)
				email := row[1].(string)
				if len(email) > 12 && email[len(email)-12:] == "@company.com" {
					fmt.Printf("| %-12s | %-26s |\n", firstName, email)
				}
			}
		}
	}

	// Workaround 3: Instead of IN, use multiple queries
	fmt.Println("\n15. Workaround for IN - use separate queries:")
	fmt.Println("✅ Query Engineering department:")
	result, err = engine.Execute("SELECT first_name, department FROM users WHERE department = 'Engineering'")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		mist.PrintResult(result)
	}
	
	fmt.Println("✅ Query Marketing department:")
	result, err = engine.Execute("SELECT first_name, department FROM users WHERE department = 'Marketing'")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		mist.PrintResult(result)
	}

	fmt.Println("\n=== Summary ===")
	fmt.Println("✅ What works well in Mist:")
	fmt.Println("   • Basic SELECT with column names and aliases")
	fmt.Println("   • Aggregate functions (COUNT, SUM, AVG, MIN, MAX)")
	fmt.Println("   • Simple WHERE conditions (=, !=, <, <=, >, >=)")
	fmt.Println("   • Logical operators (AND, OR)")
	fmt.Println("   • JOINs with equality conditions")
	fmt.Println("   • UNION operations")
	fmt.Println("   • GROUP BY and HAVING clauses")

	fmt.Println("\n❌ Current limitations:")
	fmt.Println("   • No functions in SELECT (UPPER, CONCAT, IF, etc.)")
	fmt.Println("   • No calculated expressions (math operations)")
	fmt.Println("   • No CASE expressions")
	fmt.Println("   • No LIKE pattern matching")
	fmt.Println("   • No IN, BETWEEN, IS NULL operators")
	fmt.Println("   • No scalar subqueries")

	fmt.Println("\n💡 Recommended approach:")
	fmt.Println("   • Keep queries simple with basic operations")
	fmt.Println("   • Handle complex logic in application code")
	fmt.Println("   • Use multiple simple queries instead of complex ones")
	fmt.Println("   • Filter and transform data in your application")
}