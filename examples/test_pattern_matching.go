package main

import (
	"fmt"
	"log"

	"github.com/abbychau/mist/mist"
)

func main() {
	// Create a new engine
	engine := mist.NewSQLEngine()

	// Create test tables
	err := createTestTables(engine)
	if err != nil {
		log.Fatal(err)
	}

	// Test LIKE and NOT LIKE pattern matching
	fmt.Println("=== Testing LIKE and NOT LIKE Pattern Matching ===")
	testLikePatterns(engine)

	// Test logical NOT operator
	fmt.Println("\n=== Testing Logical NOT Operator ===")
	testLogicalNot(engine)

	// Test EXISTS and NOT EXISTS subqueries
	fmt.Println("\n=== Testing EXISTS and NOT EXISTS Subqueries ===")
	testExistsSubqueries(engine)

	// Test complex combinations
	fmt.Println("\n=== Testing Complex Combinations ===")
	testComplexCombinations(engine)

	fmt.Println("\n=== All pattern matching tests completed successfully! ===")
}

func createTestTables(engine *mist.SQLEngine) error {
	// Create products table
	_, err := engine.Execute("CREATE TABLE products (id INT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(255), description TEXT, price FLOAT, category_id INT)")
	if err != nil {
		return err
	}

	// Create categories table
	_, err = engine.Execute("CREATE TABLE categories (id INT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(255), active BOOL)")
	if err != nil {
		return err
	}

	// Insert test data
	testData := []string{
		// Products
		"INSERT INTO products (name, description, price, category_id) VALUES ('Apple iPhone 15', 'Latest smartphone from Apple', 999.99, 1)",
		"INSERT INTO products (name, description, price, category_id) VALUES ('Samsung Galaxy S24', 'Android flagship phone', 849.99, 1)",
		"INSERT INTO products (name, description, price, category_id) VALUES ('MacBook Pro', 'Professional laptop computer', 2499.99, 2)",
		"INSERT INTO products (name, description, price, category_id) VALUES ('Dell XPS 13', 'Ultrabook laptop', 1299.99, 2)",
		"INSERT INTO products (name, description, price, category_id) VALUES ('iPad Air', 'Tablet device from Apple', 599.99, 3)",
		"INSERT INTO products (name, description, price, category_id) VALUES ('Microsoft Surface', 'Windows tablet computer', 799.99, 3)",
		"INSERT INTO products (name, description, price, category_id) VALUES ('Apple Watch', 'Smartwatch from Apple', 299.99, 4)",

		// Categories
		"INSERT INTO categories (name, active) VALUES ('Smartphones', true)",
		"INSERT INTO categories (name, active) VALUES ('Laptops', true)",
		"INSERT INTO categories (name, active) VALUES ('Tablets', true)",
		"INSERT INTO categories (name, active) VALUES ('Wearables', false)",
		"INSERT INTO categories (name, active) VALUES ('Accessories', true)",
	}

	for _, query := range testData {
		_, err := engine.Execute(query)
		if err != nil {
			return fmt.Errorf("error inserting test data: %v", err)
		}
	}

	return nil
}

func testLikePatterns(engine *mist.SQLEngine) {
	tests := []struct {
		name  string
		query string
		desc  string
	}{
		{
			"Basic LIKE with %",
			"SELECT name FROM products WHERE name LIKE 'Apple%'",
			"Find products starting with 'Apple'",
		},
		{
			"LIKE with % at beginning",
			"SELECT name FROM products WHERE name LIKE '%Pro'",
			"Find products ending with 'Pro'",
		},
		{
			"LIKE with % in middle",
			"SELECT name FROM products WHERE name LIKE '%Galaxy%'",
			"Find products containing 'Galaxy'",
		},
		{
			"LIKE with underscore",
			"SELECT name FROM products WHERE name LIKE 'iPad _ir'",
			"Find products matching 'iPad _ir' pattern",
		},
		{
			"NOT LIKE pattern",
			"SELECT name FROM products WHERE name NOT LIKE 'Apple%'",
			"Find products NOT starting with 'Apple'",
		},
		{
			"LIKE with case sensitivity",
			"SELECT name FROM products WHERE UPPER(name) LIKE 'APPLE%'",
			"Case-insensitive search using UPPER function",
		},
		{
			"Complex LIKE in WHERE",
			"SELECT name, price FROM products WHERE description LIKE '%computer%' AND price > 1000",
			"Find expensive computer products",
		},
	}

	for _, test := range tests {
		fmt.Printf("\n--- %s ---\n", test.name)
		fmt.Printf("Description: %s\n", test.desc)
		fmt.Printf("Query: %s\n", test.query)
		
		result, err := engine.Execute(test.query)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		selectResult, ok := result.(*mist.SelectResult)
		if !ok {
			fmt.Printf("Unexpected result type: %T\n", result)
			continue
		}

		fmt.Printf("Results (%d rows):\n", len(selectResult.Rows))
		for _, row := range selectResult.Rows {
			fmt.Printf("  %v\n", row)
		}
	}
}

func testLogicalNot(engine *mist.SQLEngine) {
	tests := []struct {
		name  string
		query string
		desc  string
	}{
		{
			"Simple NOT",
			"SELECT name FROM categories WHERE NOT active",
			"Find inactive categories",
		},
		{
			"NOT with AND",
			"SELECT name, price FROM products WHERE NOT (price > 1000 AND category_id = 1)",
			"Find products that are NOT (expensive smartphones)",
		},
		{
			"NOT with OR",
			"SELECT name FROM products WHERE NOT (name LIKE 'Apple%' OR name LIKE 'Samsung%')",
			"Find products NOT from Apple or Samsung",
		},
		{
			"Double NOT",
			"SELECT name FROM categories WHERE NOT (NOT active)",
			"Find active categories using double negation",
		},
		{
			"NOT with IS NULL",
			"SELECT name FROM products WHERE NOT (description IS NULL)",
			"Find products with non-null descriptions",
		},
		{
			"NOT with BETWEEN",
			"SELECT name, price FROM products WHERE NOT (price BETWEEN 500 AND 1000)",
			"Find products NOT in the $500-$1000 range",
		},
		{
			"NOT with IN",
			"SELECT name FROM products WHERE NOT (category_id IN (1, 2))",
			"Find products NOT in categories 1 or 2",
		},
	}

	for _, test := range tests {
		fmt.Printf("\n--- %s ---\n", test.name)
		fmt.Printf("Description: %s\n", test.desc)
		fmt.Printf("Query: %s\n", test.query)
		
		result, err := engine.Execute(test.query)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		selectResult, ok := result.(*mist.SelectResult)
		if !ok {
			fmt.Printf("Unexpected result type: %T\n", result)
			continue
		}

		fmt.Printf("Results (%d rows):\n", len(selectResult.Rows))
		for _, row := range selectResult.Rows {
			fmt.Printf("  %v\n", row)
		}
	}
}

func testExistsSubqueries(engine *mist.SQLEngine) {
	tests := []struct {
		name  string
		query string
		desc  string
	}{
		{
			"Basic EXISTS",
			"SELECT name FROM categories WHERE EXISTS (SELECT 1 FROM products WHERE products.category_id = categories.id)",
			"Find categories that have products",
		},
		{
			"NOT EXISTS",
			"SELECT name FROM categories WHERE NOT EXISTS (SELECT 1 FROM products WHERE products.category_id = categories.id)",
			"Find categories with no products",
		},
		{
			"EXISTS with conditions",
			"SELECT name FROM categories WHERE EXISTS (SELECT 1 FROM products WHERE products.category_id = categories.id AND products.price > 1000)",
			"Find categories with expensive products",
		},
		{
			"EXISTS in complex query",
			"SELECT p.name, p.price FROM products p WHERE p.price > 500 AND EXISTS (SELECT 1 FROM categories c WHERE c.id = p.category_id AND c.active = true)",
			"Find expensive products in active categories",
		},
		{
			"NOT EXISTS with LIKE",
			"SELECT name FROM categories WHERE NOT EXISTS (SELECT 1 FROM products WHERE products.category_id = categories.id AND products.name LIKE 'Apple%')",
			"Find categories without Apple products",
		},
	}

	for _, test := range tests {
		fmt.Printf("\n--- %s ---\n", test.name)
		fmt.Printf("Description: %s\n", test.desc)
		fmt.Printf("Query: %s\n", test.query)
		
		result, err := engine.Execute(test.query)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		selectResult, ok := result.(*mist.SelectResult)
		if !ok {
			fmt.Printf("Unexpected result type: %T\n", result)
			continue
		}

		fmt.Printf("Results (%d rows):\n", len(selectResult.Rows))
		for _, row := range selectResult.Rows {
			fmt.Printf("  %v\n", row)
		}
	}
}

func testComplexCombinations(engine *mist.SQLEngine) {
	tests := []struct {
		name  string
		query string
		desc  string
	}{
		{
			"LIKE + NOT + EXISTS",
			"SELECT p.name FROM products p WHERE p.name LIKE '%Apple%' AND NOT EXISTS (SELECT 1 FROM categories c WHERE c.id = p.category_id AND NOT c.active)",
			"Find Apple products in active categories",
		},
		{
			"Complex pattern with functions",
			"SELECT UPPER(name) as product_name, ROUND(price, 2) as rounded_price FROM products WHERE name NOT LIKE 'Apple%' AND price BETWEEN 500 AND 1500",
			"Find non-Apple products in mid-price range with formatting",
		},
		{
			"Nested NOT with LIKE",
			"SELECT name FROM products WHERE NOT (NOT (name LIKE '%Pro%' OR name LIKE '%Air%'))",
			"Find Pro or Air products using double negation",
		},
		{
			"EXISTS with LIKE in subquery",
			"SELECT c.name FROM categories c WHERE EXISTS (SELECT 1 FROM products p WHERE p.category_id = c.id AND p.description LIKE '%computer%')",
			"Find categories containing computer products",
		},
		{
			"Complex WHERE with all operators",
			"SELECT p.name, p.price FROM products p WHERE (p.name LIKE 'Apple%' OR p.name LIKE 'Samsung%') AND NOT (p.price < 300) AND EXISTS (SELECT 1 FROM categories c WHERE c.id = p.category_id AND c.active = true)",
			"Find Apple/Samsung products over $300 in active categories",
		},
	}

	for _, test := range tests {
		fmt.Printf("\n--- %s ---\n", test.name)
		fmt.Printf("Description: %s\n", test.desc)
		fmt.Printf("Query: %s\n", test.query)
		
		result, err := engine.Execute(test.query)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		selectResult, ok := result.(*mist.SelectResult)
		if !ok {
			fmt.Printf("Unexpected result type: %T\n", result)
			continue
		}

		fmt.Printf("Results (%d rows):\n", len(selectResult.Rows))
		for _, row := range selectResult.Rows {
			fmt.Printf("  %v\n", row)
		}
	}
}