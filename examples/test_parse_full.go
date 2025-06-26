package main

import (
	"fmt"

	"github.com/abbychau/mist/mist"
)

func main() {
	engine := mist.NewSQLEngine()

	// Create some test tables first
	fmt.Println("=== Setting up test tables ===")
	_, err := engine.Execute("CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))")
	if err != nil {
		fmt.Printf("Error creating users table: %v\n", err)
		return
	}
	
	_, err = engine.Execute("CREATE TABLE products (id INT PRIMARY KEY, name VARCHAR(50), price DECIMAL(10,2))")
	if err != nil {
		fmt.Printf("Error creating products table: %v\n", err)
		return
	}
	fmt.Println("Tables created successfully")

	// Test all the new parse-only statements
	fmt.Println("\n=== Testing Parse-Only Statement Support ===")
	
	testStatements := []struct {
		name string
		sql  string
	}{
		{"SET Transaction Isolation (Session)", "SET SESSION TRANSACTION ISOLATION LEVEL READ UNCOMMITTED;"},
		{"SET Transaction Isolation (Global)", "SET GLOBAL TRANSACTION ISOLATION LEVEL SERIALIZABLE;"},
		{"SET Transaction Isolation (Default)", "SET TRANSACTION ISOLATION LEVEL REPEATABLE READ;"},
		{"SET autocommit", "SET autocommit = 0;"},
		{"SET sql_mode", "SET sql_mode = 'STRICT_TRANS_TABLES';"},
		{"LOCK TABLES (Single, READ)", "LOCK TABLES users READ;"},
		{"LOCK TABLES (Single, WRITE)", "LOCK TABLES products WRITE;"},
		{"LOCK TABLES (Multiple)", "LOCK TABLES users READ, products WRITE;"},
		{"UNLOCK TABLES", "UNLOCK TABLES;"},
		{"CREATE TABLE with CHECK", "CREATE TABLE orders (id INT, amount DECIMAL(10,2), CHECK (amount > 0));"},
		{"CREATE TABLE with multiple CHECKs", "CREATE TABLE accounts (id INT, balance DECIMAL(10,2), status VARCHAR(20), CHECK (balance >= 0), CHECK (status IN ('active', 'inactive')));"},
	}

	for _, test := range testStatements {
		fmt.Printf("\n--- %s ---\n", test.name)
		fmt.Printf("SQL: %s\n", test.sql)
		
		result, err := engine.Execute(test.sql)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Printf("Result: %v\n", result)
		}
	}

	fmt.Println("\n=== Migration Compatibility Test ===")
	// Test a typical MySQL dump-style sequence
	migrationSQL := []string{
		"SET SESSION sql_mode = 'NO_AUTO_VALUE_ON_ZERO';",
		"SET SESSION foreign_key_checks = 0;",
		"SET SESSION unique_checks = 0;",
		"LOCK TABLES employees WRITE;",
		"CREATE TABLE employees (id INT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(100) NOT NULL, salary DECIMAL(10,2) CHECK (salary > 0));",
		"INSERT INTO employees (name, salary) VALUES ('John Doe', 50000.00);",
		"UNLOCK TABLES;",
		"SET SESSION foreign_key_checks = 1;",
		"SET SESSION unique_checks = 1;",
	}

	fmt.Println("Simulating MySQL dump migration:")
	for i, sql := range migrationSQL {
		fmt.Printf("%d. %s\n", i+1, sql)
		result, err := engine.Execute(sql)
		if err != nil {
			fmt.Printf("   ❌ Error: %v\n", err)
		} else {
			fmt.Printf("   ✅ %v\n", result)
		}
	}

	fmt.Println("\n=== Verification ===")
	// Verify the table was created and data inserted
	result, err := engine.Execute("SELECT * FROM employees;")
	if err != nil {
		fmt.Printf("Error querying employees: %v\n", err)
	} else {
		fmt.Printf("Employees table content: %v\n", result)
	}
}