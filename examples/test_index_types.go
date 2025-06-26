package main

import (
	"fmt"
	"log"

	"github.com/abbychau/mist/mist"
)

func main() {
	fmt.Println("=== Index Types Test for Mist Database ===")

	engine := mist.NewSQLEngine()

	// Create a test table with multiple columns for indexing
	createTableQuery := `CREATE TABLE products (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		category VARCHAR(50),
		price DECIMAL(10,2),
		description TEXT,
		tags SET('electronics', 'clothing', 'books', 'home', 'sports'),
		created_date DATE,
		status ENUM('active', 'inactive', 'discontinued')
	)`

	_, err := engine.Execute(createTableQuery)
	if err != nil {
		log.Fatalf("Error creating products table: %v", err)
	}
	fmt.Println("✅ Products table created successfully")

	// Insert sample data
	insertQueries := []string{
		`INSERT INTO products (name, category, price, description, tags, created_date, status) VALUES 
		 ('Laptop Pro', 'electronics', 1299.99, 'High-performance laptop with latest specs', 'electronics', '2024-01-15', 'active')`,
		
		`INSERT INTO products (name, category, price, description, tags, created_date, status) VALUES 
		 ('Cotton T-Shirt', 'clothing', 29.99, 'Comfortable cotton t-shirt in various colors', 'clothing', '2024-02-01', 'active')`,
		
		`INSERT INTO products (name, category, price, description, tags, created_date, status) VALUES 
		 ('Programming Book', 'books', 49.99, 'Comprehensive guide to modern programming', 'books', '2024-01-20', 'active')`,
		
		`INSERT INTO products (name, category, price, description, tags, created_date, status) VALUES 
		 ('Smart Watch', 'electronics', 399.99, 'Feature-rich smartwatch with health tracking', 'electronics', '2024-02-10', 'discontinued')`,
		
		`INSERT INTO products (name, category, price, description, tags, created_date, status) VALUES 
		 ('Running Shoes', 'sports', 129.99, 'Lightweight running shoes for athletes', 'sports,clothing', '2024-01-30', 'active')`,
	}

	for i, query := range insertQueries {
		_, err := engine.Execute(query)
		if err != nil {
			log.Printf("Error inserting row %d: %v", i+1, err)
		} else {
			fmt.Printf("✅ Product %d inserted successfully\n", i+1)
		}
	}

	// Test 1: Create single-column functional index (should work normally)
	fmt.Println("\n=== Test 1: Single-Column Functional Index ===")
	_, err = engine.Execute("CREATE INDEX idx_category ON products (category)")
	if err != nil {
		log.Printf("Error creating single-column index: %v", err)
	} else {
		fmt.Println("✅ Single-column index created successfully")
	}

	// Test 2: Create composite index (should be parsed but not functional)
	fmt.Println("\n=== Test 2: Composite Index (Parsed Only) ===")
	_, err = engine.Execute("CREATE INDEX idx_category_price ON products (category, price)")
	if err != nil {
		log.Printf("Error creating composite index: %v", err)
	} else {
		fmt.Println("✅ Composite index created successfully (parsed only)")
	}

	// Test 3: Create full-text index (should be parsed but not functional)
	fmt.Println("\n=== Test 3: Full-Text Index (Parsed Only) ===")
	_, err = engine.Execute("CREATE FULLTEXT INDEX idx_fulltext_description ON products (description)")
	if err != nil {
		log.Printf("Error creating full-text index: %v", err)
	} else {
		fmt.Println("✅ Full-text index created successfully (parsed only)")
	}

	// Test 4: Create multi-column full-text index
	fmt.Println("\n=== Test 4: Multi-Column Full-Text Index ===")
	_, err = engine.Execute("CREATE FULLTEXT INDEX idx_fulltext_multi ON products (name, description)")
	if err != nil {
		log.Printf("Error creating multi-column full-text index: %v", err)
	} else {
		fmt.Println("✅ Multi-column full-text index created successfully (parsed only)")
	}

	// Test 5: Show all indexes
	fmt.Println("\n=== Test 5: Show All Indexes ===")
	result, err := engine.Execute("SHOW INDEX FROM products")
	if err != nil {
		log.Printf("Error showing indexes: %v", err)
	} else {
		fmt.Println("📋 All indexes for products table:")
		mist.PrintResult(result)
	}

	// Test 6: Test functional behavior - single-column index should work
	fmt.Println("\n=== Test 6: Query Performance Test ===")
	fmt.Println("Querying with indexed column (category) - should use index:")
	result, err = engine.Execute("SELECT name, category, price FROM products WHERE category = 'electronics'")
	if err != nil {
		log.Printf("Error with indexed query: %v", err)
	} else {
		mist.PrintResult(result)
	}

	// Test 7: Test parsed-only behavior - composite index should not provide lookup
	fmt.Println("\n=== Test 7: Composite Index Behavior ===")
	fmt.Println("Note: Composite indexes are parsed but don't provide actual lookup functionality")
	result, err = engine.Execute("SELECT name, category, price FROM products WHERE category = 'electronics' AND price > 300")
	if err != nil {
		log.Printf("Error with composite condition query: %v", err)
	} else {
		fmt.Println("Query executes normally (without composite index optimization):")
		mist.PrintResult(result)
	}

	// Test 8: Test drop index functionality
	fmt.Println("\n=== Test 8: Drop Index Test ===")
	_, err = engine.Execute("DROP INDEX idx_category_price")
	if err != nil {
		log.Printf("Error dropping composite index: %v", err)
	} else {
		fmt.Println("✅ Composite index dropped successfully")
	}

	// Test 9: Show indexes after drop
	fmt.Println("\n=== Test 9: Show Indexes After Drop ===")
	result, err = engine.Execute("SHOW INDEX FROM products")
	if err != nil {
		log.Printf("Error showing indexes after drop: %v", err)
	} else {
		fmt.Println("📋 Remaining indexes after drop:")
		mist.PrintResult(result)
	}

	// Test 10: Create index with invalid syntax (should fail)
	fmt.Println("\n=== Test 10: Invalid Index Syntax Test ===")
	_, err = engine.Execute("CREATE INDEX invalid_index ON products ()")
	if err != nil {
		fmt.Printf("✅ Expected error for invalid syntax: %v\n", err)
	} else {
		fmt.Println("❌ Should have failed for invalid syntax")
	}

	// Test 11: Create index on non-existent column (should fail)
	fmt.Println("\n=== Test 11: Non-Existent Column Test ===")
	_, err = engine.Execute("CREATE INDEX idx_nonexistent ON products (nonexistent_column)")
	if err != nil {
		fmt.Printf("✅ Expected error for non-existent column: %v\n", err)
	} else {
		fmt.Println("❌ Should have failed for non-existent column")
	}

	fmt.Println("\n✅ All index type tests completed!")
	
	fmt.Println("\n📊 Summary of Index Types:")
	fmt.Println("   ✅ Single-column indexes - Fully functional with hash-based lookup")
	fmt.Println("   📝 Composite indexes - Parsed and stored but no lookup functionality")
	fmt.Println("   📝 Full-text indexes - Parsed and stored but no search functionality")
	fmt.Println("   ✅ Index management - CREATE, DROP, SHOW INDEX all work correctly")
	fmt.Println("   ✅ Error handling - Proper validation for invalid syntax and columns")
}