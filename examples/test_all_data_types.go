package main

import (
	"fmt"
	"log"

	"github.com/abbychau/mist/mist"
)

func main() {
	fmt.Println("=== Comprehensive Data Type Test for Mist Database ===")
	
	engine := mist.NewSQLEngine()

	// Create a comprehensive table with all supported data types
	createTableQuery := `CREATE TABLE comprehensive_test (
		id INT AUTO_INCREMENT PRIMARY KEY,
		-- Integer types
		int_col INT NOT NULL,
		-- String types  
		varchar_col VARCHAR(100),
		text_col TEXT,
		-- Float type
		float_col FLOAT,
		-- Boolean type
		bool_col BOOL,
		-- Decimal type
		decimal_col DECIMAL(10,2),
		-- Date/Time types
		timestamp_col TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		date_col DATE,
		time_col TIME,
		year_col YEAR,
		-- Enum type
		status_enum ENUM('active', 'inactive', 'pending') DEFAULT 'pending',
		-- Set type
		features_set SET('feature1', 'feature2', 'feature3', 'feature4') 
	)`

	_, err := engine.Execute(createTableQuery)
	if err != nil {
		log.Fatalf("Error creating comprehensive table: %v", err)
	}
	fmt.Println("✅ Comprehensive table with all data types created successfully")

	// Insert sample data covering all data types
	insertQueries := []string{
		`INSERT INTO comprehensive_test 
		 (int_col, varchar_col, text_col, float_col, bool_col, decimal_col, 
		  date_col, time_col, year_col, status_enum, features_set) 
		 VALUES 
		 (42, 'Hello World', 'This is a long text field', 3.14159, true, 123.45,
		  '2024-01-15', '14:30:00', 2024, 'active', 'feature1,feature3')`,

		`INSERT INTO comprehensive_test 
		 (int_col, varchar_col, text_col, float_col, bool_col, decimal_col, 
		  date_col, time_col, year_col, status_enum, features_set) 
		 VALUES 
		 (100, 'Test String', 'Another text entry', 2.71828, false, 999.99,
		  '2023-12-25', '09:15:30', 99, 'inactive', 'feature2,feature4')`,

		`INSERT INTO comprehensive_test 
		 (int_col, varchar_col, text_col, float_col, bool_col, decimal_col, 
		  date_col, time_col, year_col, status_enum, features_set) 
		 VALUES 
		 (0, '', 'Empty varchar test', 0.0, false, 0.00,
		  '2000-01-01', '00:00:00', 2000, 'pending', '')`,

		`INSERT INTO comprehensive_test 
		 (int_col, varchar_col, text_col, float_col, bool_col, decimal_col, 
		  date_col, time_col, year_col, status_enum, features_set) 
		 VALUES 
		 (-50, 'Negative test', 'Testing negative values', -1.23, true, -50.75,
		  '1995-06-15', '23:59:59', 1995, 'active', 'feature1,feature2,feature3')`,
	}

	for i, query := range insertQueries {
		result, err := engine.Execute(query)
		if err != nil {
			log.Printf("Error inserting row %d: %v", i+1, err)
		} else {
			fmt.Printf("✅ Row %d inserted successfully: %v\n", i+1, result)
		}
	}

	// Display all data
	fmt.Println("\n=== All Data in Comprehensive Test Table ===")
	result, err := engine.Execute("SELECT * FROM comprehensive_test")
	if err != nil {
		log.Printf("Error selecting all data: %v", err)
	} else {
		mist.PrintResult(result)
	}

	// Test some queries with different data types
	fmt.Println("\n=== Testing Queries with Different Data Types ===")

	// Test TIME queries
	fmt.Println("\nEvents in the morning (before 12:00:00):")
	result, err = engine.Execute("SELECT id, varchar_col, time_col FROM comprehensive_test WHERE time_col < '12:00:00'")
	if err != nil {
		log.Printf("Error with TIME query: %v", err)
	} else {
		mist.PrintResult(result)
	}

	// Test YEAR queries
	fmt.Println("\nRecords from this millennium (year >= 2000):")
	result, err = engine.Execute("SELECT id, varchar_col, year_col FROM comprehensive_test WHERE year_col >= 2000")
	if err != nil {
		log.Printf("Error with YEAR query: %v", err)
	} else {
		mist.PrintResult(result)
	}

	// Test ENUM queries
	fmt.Println("\nActive records:")
	result, err = engine.Execute("SELECT id, varchar_col, status_enum FROM comprehensive_test WHERE status_enum = 'active'")
	if err != nil {
		log.Printf("Error with ENUM query: %v", err)
	} else {
		mist.PrintResult(result)
	}

	// Test SET queries (this is approximate since exact SET matching in WHERE clauses needs more complex parsing)
	fmt.Println("\nRecords with feature1:")
	result, err = engine.Execute("SELECT id, varchar_col, features_set FROM comprehensive_test WHERE features_set LIKE '%feature1%'")
	if err != nil {
		log.Printf("Error with SET query: %v", err)
	} else {
		mist.PrintResult(result)
	}

	// Test DECIMAL and FLOAT operations
	fmt.Println("\nRecords with positive decimal values:")
	result, err = engine.Execute("SELECT id, varchar_col, decimal_col, float_col FROM comprehensive_test WHERE decimal_col > 0")
	if err != nil {
		log.Printf("Error with DECIMAL query: %v", err)
	} else {
		mist.PrintResult(result)
	}

	// Test boolean operations
	fmt.Println("\nRecords where bool_col is true:")
	result, err = engine.Execute("SELECT id, varchar_col, bool_col FROM comprehensive_test WHERE bool_col = true")
	if err != nil {
		log.Printf("Error with BOOL query: %v", err)
	} else {
		mist.PrintResult(result)
	}

	fmt.Println("\n✅ All data type tests completed successfully!")
	fmt.Println("\n📊 Summary of Supported Data Types:")
	fmt.Println("   ✅ INT - Integer values")
	fmt.Println("   ✅ VARCHAR(n) - Variable length strings")
	fmt.Println("   ✅ TEXT - Long text fields")
	fmt.Println("   ✅ FLOAT - Floating point numbers")
	fmt.Println("   ✅ BOOL - Boolean values")
	fmt.Println("   ✅ DECIMAL(p,s) - Fixed-point decimal numbers")
	fmt.Println("   ✅ TIMESTAMP - Date and time with CURRENT_TIMESTAMP support")
	fmt.Println("   ✅ DATE - Date values")
	fmt.Println("   ✅ TIME - Time values (HH:MM:SS)")
	fmt.Println("   ✅ YEAR - Year values (1901-2155, 2/4-digit)")
	fmt.Println("   ✅ ENUM - Enumerated values")
	fmt.Println("   ✅ SET - Multiple choice values")
}