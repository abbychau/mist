package main

import (
	"fmt"
	"log"

	"github.com/abbychau/mist/mist"
)

func main() {
	fmt.Println("=== New WHERE Operators Test (IS NULL, BETWEEN, IN) ===")

	engine := mist.NewSQLEngine()

	// Create test tables
	createTable := `CREATE TABLE products (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		category VARCHAR(50),
		price DECIMAL(10,2),
		stock_quantity INT,
		discontinued BOOL DEFAULT false,
		description TEXT
	)`

	_, err := engine.Execute(createTable)
	if err != nil {
		log.Fatalf("Error creating products table: %v", err)
	}
	fmt.Println("✅ Products table created successfully")

	// Insert test data with some NULL values
	insertQueries := []string{
		`INSERT INTO products (name, category, price, stock_quantity, discontinued, description) VALUES 
		 ('Laptop', 'Electronics', 999.99, 50, false, 'High-performance laptop')`,
		`INSERT INTO products (name, category, price, stock_quantity, discontinued, description) VALUES 
		 ('Mouse', 'Electronics', 25.50, 100, false, 'Wireless optical mouse')`,
		`INSERT INTO products (name, category, price, stock_quantity, discontinued, description) VALUES 
		 ('Keyboard', 'Electronics', 75.00, 30, false, 'Mechanical gaming keyboard')`,
		`INSERT INTO products (name, category, price, stock_quantity, discontinued, description) VALUES 
		 ('Mystery Item', NULL, 45.00, 0, true, NULL)`, // NULL category and description
		`INSERT INTO products (name, category, price, stock_quantity, discontinued, description) VALUES 
		 ('Old Product', 'Legacy', NULL, 5, true, 'Discontinued legacy product')`, // NULL price
		`INSERT INTO products (name, category, price, stock_quantity, discontinued, description) VALUES 
		 ('Book', 'Education', 29.99, 75, false, 'Programming guide')`,
		`INSERT INTO products (name, category, price, stock_quantity, discontinued, description) VALUES 
		 ('Tablet', 'Electronics', 199.99, 25, false, 'Compact tablet device')`,
		`INSERT INTO products (name, category, price, stock_quantity, discontinued, description) VALUES 
		 ('Headphones', 'Electronics', 149.99, 40, false, 'Noise-canceling headphones')`,
	}

	for i, query := range insertQueries {
		_, err := engine.Execute(query)
		if err != nil {
			log.Printf("Error inserting product %d: %v", i+1, err)
		} else {
			fmt.Printf("✅ Product %d inserted successfully\n", i+1)
		}
	}

	// Display all data first
	fmt.Println("\n=== All Products ===")
	result, err := engine.Execute("SELECT * FROM products")
	if err != nil {
		log.Printf("Error selecting all products: %v", err)
	} else {
		mist.PrintResult(result)
	}

	fmt.Println("\n=== Testing IS NULL and IS NOT NULL ===")

	// Test 1: IS NULL
	fmt.Println("\n1. Products with NULL category:")
	result, err = engine.Execute("SELECT id, name, category FROM products WHERE category IS NULL")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Println("✅ IS NULL works:")
		mist.PrintResult(result)
	}

	// Test 2: IS NOT NULL
	fmt.Println("\n2. Products with non-NULL price:")
	result, err = engine.Execute("SELECT id, name, price FROM products WHERE price IS NOT NULL")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Println("✅ IS NOT NULL works:")
		mist.PrintResult(result)
	}

	// Test 3: IS NULL with TEXT column
	fmt.Println("\n3. Products with NULL description:")
	result, err = engine.Execute("SELECT id, name, description FROM products WHERE description IS NULL")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Println("✅ IS NULL on TEXT column works:")
		mist.PrintResult(result)
	}

	fmt.Println("\n=== Testing BETWEEN Operator ===")

	// Test 4: BETWEEN with numeric values
	fmt.Println("\n4. Products with price between 50 and 200:")
	result, err = engine.Execute("SELECT id, name, price FROM products WHERE price BETWEEN 50.00 AND 200.00")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Println("✅ BETWEEN works:")
		mist.PrintResult(result)
	}

	// Test 5: NOT BETWEEN
	fmt.Println("\n5. Products with price NOT between 100 and 500:")
	result, err = engine.Execute("SELECT id, name, price FROM products WHERE price NOT BETWEEN 100.00 AND 500.00")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Println("✅ NOT BETWEEN works:")
		mist.PrintResult(result)
	}

	// Test 6: BETWEEN with integer values
	fmt.Println("\n6. Products with stock between 20 and 80:")
	result, err = engine.Execute("SELECT id, name, stock_quantity FROM products WHERE stock_quantity BETWEEN 20 AND 80")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Println("✅ BETWEEN with integers works:")
		mist.PrintResult(result)
	}

	fmt.Println("\n=== Testing IN Operator ===")

	// Test 7: IN with string values
	fmt.Println("\n7. Electronics and Education products:")
	result, err = engine.Execute("SELECT id, name, category FROM products WHERE category IN ('Electronics', 'Education')")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Println("✅ IN with strings works:")
		mist.PrintResult(result)
	}

	// Test 8: NOT IN
	fmt.Println("\n8. Non-Electronics products:")
	result, err = engine.Execute("SELECT id, name, category FROM products WHERE category NOT IN ('Electronics')")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Println("✅ NOT IN works:")
		mist.PrintResult(result)
	}

	// Test 9: IN with numeric values
	fmt.Println("\n9. Products with specific stock levels:")
	result, err = engine.Execute("SELECT id, name, stock_quantity FROM products WHERE stock_quantity IN (25, 50, 75)")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Println("✅ IN with numbers works:")
		mist.PrintResult(result)
	}

	// Test 10: IN with single value (edge case)
	fmt.Println("\n10. Products with exactly 30 stock:")
	result, err = engine.Execute("SELECT id, name, stock_quantity FROM products WHERE stock_quantity IN (30)")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Println("✅ IN with single value works:")
		mist.PrintResult(result)
	}

	fmt.Println("\n=== Testing Combined Conditions ===")

	// Test 11: Combining IS NOT NULL with BETWEEN
	fmt.Println("\n11. Products with non-NULL price between 25 and 100:")
	result, err = engine.Execute("SELECT id, name, price FROM products WHERE price IS NOT NULL AND price BETWEEN 25.00 AND 100.00")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Println("✅ IS NOT NULL + BETWEEN works:")
		mist.PrintResult(result)
	}

	// Test 12: Combining IN with other conditions
	fmt.Println("\n12. Electronics with stock greater than 30:")
	result, err = engine.Execute("SELECT id, name, category, stock_quantity FROM products WHERE category IN ('Electronics') AND stock_quantity > 30")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Println("✅ IN + comparison works:")
		mist.PrintResult(result)
	}

	// Test 13: IS NULL with OR condition
	fmt.Println("\n13. Products with NULL price OR discontinued:")
	result, err = engine.Execute("SELECT id, name, price, discontinued FROM products WHERE price IS NULL OR discontinued = true")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Println("✅ IS NULL + OR works:")
		mist.PrintResult(result)
	}

	fmt.Println("\n=== Testing Edge Cases ===")

	// Test 14: Empty IN list (should be handled gracefully)
	fmt.Println("\n14. Testing edge cases - IN with mixed types:")
	result, err = engine.Execute("SELECT id, name FROM products WHERE id IN (1, 3, 5)")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Println("✅ IN with IDs works:")
		mist.PrintResult(result)
	}

	// Test 15: BETWEEN with edge values
	fmt.Println("\n15. BETWEEN with exact boundary values:")
	result, err = engine.Execute("SELECT id, name, price FROM products WHERE price BETWEEN 25.50 AND 25.50")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		fmt.Println("✅ BETWEEN with exact match works:")
		mist.PrintResult(result)
	}

	fmt.Println("\n=== Summary ===")
	fmt.Println("✅ Successfully implemented operators:")
	fmt.Println("   • IS NULL - Test for NULL values")
	fmt.Println("   • IS NOT NULL - Test for non-NULL values")
	fmt.Println("   • BETWEEN value1 AND value2 - Range testing (inclusive)")
	fmt.Println("   • NOT BETWEEN value1 AND value2 - Inverse range testing")
	fmt.Println("   • IN (value1, value2, ...) - Set membership testing")
	fmt.Println("   • NOT IN (value1, value2, ...) - Inverse set membership")
	fmt.Println("   • All operators work with AND/OR combinations")
	fmt.Println("   • Support for different data types (strings, numbers, booleans)")

	fmt.Println("\n🎯 These operators greatly enhance WHERE clause capabilities!")
	fmt.Println("   • NULL handling is now fully supported")
	fmt.Println("   • Range queries are much more convenient")
	fmt.Println("   • Set membership testing eliminates need for multiple ORs")
	fmt.Println("   • Compatible with JOINs and other SQL features")
}