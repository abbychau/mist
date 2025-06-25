package main

import (
	"fmt"
	"log"

	"github.com/abbychau/mist/mist"
)

func main() {
	fmt.Println("=== Testing New Data Types: TIME, YEAR, SET ===")
	
	engine := mist.NewSQLEngine()

	// Test TIME, YEAR, and SET data types
	testTimeDataType(engine)
	testYearDataType(engine)
	testSetDataType(engine)
	
	fmt.Println("\n✅ All new data types tested successfully!")
}

func testTimeDataType(engine *mist.SQLEngine) {
	fmt.Println("\n=== Testing TIME Data Type ===")
	
	// Create table with TIME column
	_, err := engine.Execute(`CREATE TABLE schedule (
		id INT AUTO_INCREMENT PRIMARY KEY,
		event_name VARCHAR(100),
		start_time TIME,
		end_time TIME NOT NULL
	)`)
	if err != nil {
		log.Printf("Error creating schedule table: %v", err)
		return
	}
	fmt.Println("✅ Table with TIME columns created successfully")
	
	// Test inserting TIME values
	testTimeInserts := []string{
		"INSERT INTO schedule (event_name, start_time, end_time) VALUES ('Morning Meeting', '09:00:00', '10:30:00')",
		"INSERT INTO schedule (event_name, start_time, end_time) VALUES ('Lunch Break', '12:00:00', '13:00:00')",
		"INSERT INTO schedule (event_name, start_time, end_time) VALUES ('Conference Call', '14:30:00', '15:45:00')",
		"INSERT INTO schedule (event_name, start_time, end_time) VALUES ('Late Work', '18:00:00', '22:30:15')",
		"INSERT INTO schedule (event_name, end_time) VALUES ('All Day Event', '23:59:59')", // start_time is NULL
	}
	
	for _, query := range testTimeInserts {
		result, err := engine.Execute(query)
		if err != nil {
			log.Printf("Error inserting TIME data: %v", err)
		} else {
			fmt.Printf("✅ TIME insert successful: %v\n", result)
		}
	}
	
	// Test invalid TIME values
	invalidTimeQueries := []string{
		"INSERT INTO schedule (event_name, start_time, end_time) VALUES ('Invalid Time', '25:00:00', '10:00:00')", // Invalid hour
		"INSERT INTO schedule (event_name, start_time, end_time) VALUES ('Invalid Time', '10:60:00', '11:00:00')", // Invalid minute
		"INSERT INTO schedule (event_name, start_time, end_time) VALUES ('Invalid Time', '10:00:60', '11:00:00')", // Invalid second
		"INSERT INTO schedule (event_name, start_time, end_time) VALUES ('Invalid Format', '10:00', '11:00:00')",   // Invalid format
	}
	
	fmt.Println("\nTesting invalid TIME values (should fail):")
	for _, query := range invalidTimeQueries {
		_, err := engine.Execute(query)
		if err != nil {
			fmt.Printf("✅ Expected error for invalid TIME: %v\n", err)
		} else {
			fmt.Printf("❌ Invalid TIME value was unexpectedly accepted\n")
		}
	}
	
	// Display all schedule data
	fmt.Println("\nSchedule table contents:")
	result, err := engine.Execute("SELECT * FROM schedule")
	if err != nil {
		log.Printf("Error selecting from schedule: %v", err)
	} else {
		mist.PrintResult(result)
	}
}

func testYearDataType(engine *mist.SQLEngine) {
	fmt.Println("\n=== Testing YEAR Data Type ===")
	
	// Create table with YEAR column
	_, err := engine.Execute(`CREATE TABLE documents (
		id INT AUTO_INCREMENT PRIMARY KEY,
		title VARCHAR(200),
		publication_year YEAR NOT NULL,
		revision_year YEAR
	)`)
	if err != nil {
		log.Printf("Error creating documents table: %v", err)
		return
	}
	fmt.Println("✅ Table with YEAR columns created successfully")
	
	// Test inserting YEAR values
	testYearInserts := []string{
		"INSERT INTO documents (title, publication_year, revision_year) VALUES ('Database Guide', 2023, 2024)",
		"INSERT INTO documents (title, publication_year, revision_year) VALUES ('Programming Manual', '2020', '2021')",
		"INSERT INTO documents (title, publication_year, revision_year) VALUES ('Legacy System', 1995, NULL)",
		"INSERT INTO documents (title, publication_year, revision_year) VALUES ('Y2K Report', 99, 00)", // 2-digit years
		"INSERT INTO documents (title, publication_year, revision_year) VALUES ('Millennium Bug', 70, 75)", // 1970, 1975
		"INSERT INTO documents (title, publication_year) VALUES ('Single Year Doc', 2025)", // revision_year is NULL
	}
	
	for _, query := range testYearInserts {
		result, err := engine.Execute(query)
		if err != nil {
			log.Printf("Error inserting YEAR data: %v", err)
		} else {
			fmt.Printf("✅ YEAR insert successful: %v\n", result)
		}
	}
	
	// Test invalid YEAR values
	invalidYearQueries := []string{
		"INSERT INTO documents (title, publication_year) VALUES ('Too Old', 1900)",    // Before 1901
		"INSERT INTO documents (title, publication_year) VALUES ('Too New', 2156)",    // After 2155
		"INSERT INTO documents (title, publication_year) VALUES ('Invalid', 12345)",   // 5 digits
		"INSERT INTO documents (title, publication_year) VALUES ('Invalid', 'ABCD')",  // Non-numeric
	}
	
	fmt.Println("\nTesting invalid YEAR values (should fail):")
	for _, query := range invalidYearQueries {
		_, err := engine.Execute(query)
		if err != nil {
			fmt.Printf("✅ Expected error for invalid YEAR: %v\n", err)
		} else {
			fmt.Printf("❌ Invalid YEAR value was unexpectedly accepted\n")
		}
	}
	
	// Display all documents data
	fmt.Println("\nDocuments table contents:")
	result, err := engine.Execute("SELECT * FROM documents")
	if err != nil {
		log.Printf("Error selecting from documents: %v", err)
	} else {
		mist.PrintResult(result)
	}
}

func testSetDataType(engine *mist.SQLEngine) {
	fmt.Println("\n=== Testing SET Data Type ===")
	
	// Create table with SET column
	_, err := engine.Execute(`CREATE TABLE user_permissions (
		id INT AUTO_INCREMENT PRIMARY KEY,
		username VARCHAR(50),
		permissions SET('read', 'write', 'delete', 'admin') NOT NULL,
		features SET('email', 'sms', 'push', 'newsletter')
	)`)
	if err != nil {
		log.Printf("Error creating user_permissions table: %v", err)
		return
	}
	fmt.Println("✅ Table with SET columns created successfully")
	
	// Test inserting SET values
	testSetInserts := []string{
		"INSERT INTO user_permissions (username, permissions, features) VALUES ('alice', 'read,write', 'email,sms')",
		"INSERT INTO user_permissions (username, permissions, features) VALUES ('bob', 'read,write,delete', 'email,push,newsletter')",
		"INSERT INTO user_permissions (username, permissions, features) VALUES ('admin', 'read,write,delete,admin', 'email,sms,push,newsletter')",
		"INSERT INTO user_permissions (username, permissions, features) VALUES ('guest', 'read', '')", // Empty SET for features
		"INSERT INTO user_permissions (username, permissions) VALUES ('limited', 'read')", // features is NULL
		"INSERT INTO user_permissions (username, permissions, features) VALUES ('single', 'write', 'email')", // Single values
	}
	
	for _, query := range testSetInserts {
		result, err := engine.Execute(query)
		if err != nil {
			log.Printf("Error inserting SET data: %v", err)
		} else {
			fmt.Printf("✅ SET insert successful: %v\n", result)
		}
	}
	
	// Test invalid SET values
	invalidSetQueries := []string{
		"INSERT INTO user_permissions (username, permissions) VALUES ('invalid1', 'read,invalid')",     // Invalid permission
		"INSERT INTO user_permissions (username, permissions) VALUES ('invalid2', 'read,read')",        // Duplicate values
		"INSERT INTO user_permissions (username, permissions) VALUES ('invalid3', 'execute')",          // Not in allowed values
		"INSERT INTO user_permissions (username, features) VALUES ('invalid4', 'chat,video')",          // Invalid features
	}
	
	fmt.Println("\nTesting invalid SET values (should fail):")
	for _, query := range invalidSetQueries {
		_, err := engine.Execute(query)
		if err != nil {
			fmt.Printf("✅ Expected error for invalid SET: %v\n", err)
		} else {
			fmt.Printf("❌ Invalid SET value was unexpectedly accepted\n")
		}
	}
	
	// Display all user permissions data
	fmt.Println("\nUser permissions table contents:")
	result, err := engine.Execute("SELECT * FROM user_permissions")
	if err != nil {
		log.Printf("Error selecting from user_permissions: %v", err)
	} else {
		mist.PrintResult(result)
	}
	
	// Test querying specific SET values
	fmt.Println("\nUsers with admin permissions:")
	result, err = engine.Execute("SELECT username, permissions FROM user_permissions WHERE permissions = 'read,write,delete,admin'")
	if err != nil {
		log.Printf("Error querying SET values: %v", err)
	} else {
		mist.PrintResult(result)
	}
}