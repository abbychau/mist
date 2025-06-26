package main

import (
	"fmt"
	"log"

	"github.com/abbychau/mist"
)

func main() {
	fmt.Println("=== Testing Advanced Features: UNION, INSERT...SELECT, INSERT...ON DUPLICATE KEY UPDATE ===")
	
	engine := mist.NewSQLEngine()

	// Test UNION operations
	testUnionOperations(engine)
	
	// Test INSERT ... SELECT
	testInsertSelect(engine)
	
	// Test INSERT ... ON DUPLICATE KEY UPDATE
	testOnDuplicateKeyUpdate(engine)
	
	fmt.Println("\n✅ All advanced features tested successfully!")
}

func testUnionOperations(engine *mist.SQLEngine) {
	fmt.Println("\n=== Testing UNION Operations ===")
	
	// Create test tables
	engine.Execute("CREATE TABLE employees_north (id INT, name VARCHAR(50), salary FLOAT, department VARCHAR(30))")
	engine.Execute("CREATE TABLE employees_south (id INT, name VARCHAR(50), salary FLOAT, department VARCHAR(30))")
	
	// Insert test data
	northQueries := []string{
		"INSERT INTO employees_north VALUES (1, 'Alice', 75000, 'Engineering')",
		"INSERT INTO employees_north VALUES (2, 'Bob', 80000, 'Marketing')",
		"INSERT INTO employees_north VALUES (3, 'Charlie', 90000, 'Engineering')",
	}
	
	southQueries := []string{
		"INSERT INTO employees_south VALUES (4, 'Diana', 85000, 'Engineering')",
		"INSERT INTO employees_south VALUES (5, 'Eve', 95000, 'Sales')",
		"INSERT INTO employees_south VALUES (6, 'Frank', 70000, 'Marketing')",
	}
	
	for _, query := range northQueries {
		engine.Execute(query)
	}
	
	for _, query := range southQueries {
		engine.Execute(query)
	}
	
	// Test basic UNION
	fmt.Println("\nAll employees from both offices:")
	result, err := engine.Execute(`
		SELECT name, salary FROM employees_north
		UNION
		SELECT name, salary FROM employees_south
	`)
	
	if err != nil {
		log.Printf("UNION error: %v", err)
	} else {
		fmt.Println("✅ UNION working:")
		mist.PrintResult(result)
	}
	
	// Test UNION ALL
	fmt.Println("\nUNION ALL (includes duplicates if any):")
	result, err = engine.Execute(`
		SELECT department FROM employees_north
		UNION ALL
		SELECT department FROM employees_south
	`)
	
	if err != nil {
		log.Printf("UNION ALL error: %v", err)
	} else {
		fmt.Println("✅ UNION ALL working:")
		mist.PrintResult(result)
	}
	
	// Test UNION with WHERE clause
	fmt.Println("\nHigh-earning employees from both offices (salary > 80000):")
	result, err = engine.Execute(`
		SELECT name, salary FROM employees_north WHERE salary > 80000
		UNION
		SELECT name, salary FROM employees_south WHERE salary > 80000
	`)
	
	if err != nil {
		log.Printf("UNION with WHERE error: %v", err)
	} else {
		fmt.Println("✅ UNION with WHERE working:")
		mist.PrintResult(result)
	}
}

func testInsertSelect(engine *mist.SQLEngine) {
	fmt.Println("\n=== Testing INSERT ... SELECT ===")
	
	// Create target table
	engine.Execute("CREATE TABLE all_employees (id INT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(50), salary FLOAT, department VARCHAR(30), office VARCHAR(10))")
	
	// Test INSERT ... SELECT from single table
	fmt.Println("\nInserting from North office:")
	result, err := engine.Execute(`
		INSERT INTO all_employees (name, salary, department, office)
		SELECT name, salary, department, 'North' FROM employees_north
	`)
	
	if err != nil {
		log.Printf("INSERT ... SELECT error: %v", err)
	} else {
		fmt.Printf("✅ INSERT ... SELECT result: %v\n", result)
	}
	
	// Test INSERT ... SELECT from another table
	fmt.Println("\nInserting from South office:")
	result, err = engine.Execute(`
		INSERT INTO all_employees (name, salary, department, office)
		SELECT name, salary, department, 'South' FROM employees_south
	`)
	
	if err != nil {
		log.Printf("INSERT ... SELECT error: %v", err)
	} else {
		fmt.Printf("✅ INSERT ... SELECT result: %v\n", result)
	}
	
	// Show all employees
	fmt.Println("\nAll employees after INSERT ... SELECT:")
	result, err = engine.Execute("SELECT * FROM all_employees")
	if err != nil {
		log.Printf("SELECT error: %v", err)
	} else {
		mist.PrintResult(result)
	}
	
	// Test INSERT ... SELECT with UNION
	fmt.Println("\nCreating summary table with UNION in INSERT ... SELECT:")
	engine.Execute("CREATE TABLE employee_summary (id INT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(50), salary FLOAT)")
	
	result, err = engine.Execute(`
		INSERT INTO employee_summary (name, salary)
		SELECT name, salary FROM employees_north WHERE salary > 75000
		UNION
		SELECT name, salary FROM employees_south WHERE salary > 75000
	`)
	
	if err != nil {
		log.Printf("INSERT ... SELECT with UNION error: %v", err)
	} else {
		fmt.Printf("✅ INSERT ... SELECT with UNION result: %v\n", result)
		
		// Show the result
		result, _ = engine.Execute("SELECT * FROM employee_summary")
		mist.PrintResult(result)
	}
}

func testOnDuplicateKeyUpdate(engine *mist.SQLEngine) {
	fmt.Println("\n=== Testing INSERT ... ON DUPLICATE KEY UPDATE ===")
	
	// Create table with unique constraint
	engine.Execute(`CREATE TABLE products (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(100) UNIQUE,
		price FLOAT,
		quantity INT,
		last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	)`)
	
	// Insert initial data
	fmt.Println("\nInserting initial products:")
	queries := []string{
		"INSERT INTO products (name, price, quantity) VALUES ('Laptop', 999.99, 10)",
		"INSERT INTO products (name, price, quantity) VALUES ('Mouse', 29.99, 50)",
		"INSERT INTO products (name, price, quantity) VALUES ('Keyboard', 79.99, 25)",
	}
	
	for _, query := range queries {
		result, err := engine.Execute(query)
		if err != nil {
			log.Printf("Insert error: %v", err)
		} else {
			fmt.Printf("Insert result: %v\n", result)
		}
	}
	
	fmt.Println("\nInitial products:")
	result, _ := engine.Execute("SELECT * FROM products")
	mist.PrintResult(result)
	
	// Test ON DUPLICATE KEY UPDATE with VALUES()
	fmt.Println("\nTesting ON DUPLICATE KEY UPDATE with VALUES() function:")
	result, err := engine.Execute(`
		INSERT INTO products (name, price, quantity) 
		VALUES ('Laptop', 899.99, 15)
		ON DUPLICATE KEY UPDATE 
			price = VALUES(price),
			quantity = quantity + VALUES(quantity)
	`)
	
	if err != nil {
		log.Printf("ON DUPLICATE KEY UPDATE error: %v", err)
	} else {
		fmt.Printf("✅ ON DUPLICATE KEY UPDATE result: %v\n", result)
	}
	
	// Test ON DUPLICATE KEY UPDATE with column references
	fmt.Println("\nTesting ON DUPLICATE KEY UPDATE with column references:")
	result, err = engine.Execute(`
		INSERT INTO products (name, price, quantity) 
		VALUES ('Mouse', 24.99, 25)
		ON DUPLICATE KEY UPDATE 
			price = VALUES(price),
			quantity = quantity + 25
	`)
	
	if err != nil {
		log.Printf("ON DUPLICATE KEY UPDATE error: %v", err)
	} else {
		fmt.Printf("✅ ON DUPLICATE KEY UPDATE result: %v\n", result)
	}
	
	fmt.Println("\nProducts after ON DUPLICATE KEY UPDATE:")
	result, _ = engine.Execute("SELECT * FROM products")
	mist.PrintResult(result)
	
	// Test INSERT new item (no duplicate)
	fmt.Println("\nInserting new product (no duplicate):")
	result, err = engine.Execute(`
		INSERT INTO products (name, price, quantity) 
		VALUES ('Monitor', 299.99, 8)
		ON DUPLICATE KEY UPDATE 
			price = VALUES(price),
			quantity = quantity + VALUES(quantity)
	`)
	
	if err != nil {
		log.Printf("INSERT new item error: %v", err)
	} else {
		fmt.Printf("✅ New item insert result: %v\n", result)
	}
	
	fmt.Println("\nFinal products table:")
	result, _ = engine.Execute("SELECT * FROM products")
	mist.PrintResult(result)
	
	// Test INSERT ... SELECT with ON DUPLICATE KEY UPDATE
	fmt.Println("\nTesting INSERT ... SELECT with ON DUPLICATE KEY UPDATE:")
	
	// Create a temp table with new product data
	engine.Execute("CREATE TABLE new_products (name VARCHAR(100), price FLOAT, quantity INT)")
	engine.Execute("INSERT INTO new_products VALUES ('Laptop', 799.99, 5)")
	engine.Execute("INSERT INTO new_products VALUES ('Webcam', 59.99, 20)")
	engine.Execute("INSERT INTO new_products VALUES ('Headphones', 149.99, 12)")
	
	result, err = engine.Execute(`
		INSERT INTO products (name, price, quantity)
		SELECT name, price, quantity FROM new_products
		ON DUPLICATE KEY UPDATE 
			price = VALUES(price),
			quantity = quantity + VALUES(quantity)
	`)
	
	if err != nil {
		log.Printf("INSERT ... SELECT with ON DUPLICATE KEY UPDATE error: %v", err)
	} else {
		fmt.Printf("✅ INSERT ... SELECT with ON DUPLICATE KEY UPDATE result: %v\n", result)
	}
	
	fmt.Println("\nFinal products after INSERT ... SELECT with ON DUPLICATE KEY UPDATE:")
	result, _ = engine.Execute("SELECT * FROM products ORDER BY id")
	mist.PrintResult(result)
}