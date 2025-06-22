//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"log"

	"github.com/abbychau/mist/mist"
)

func recordingDemo() {
	// Create a new SQL engine
	engine := mist.NewSQLEngine()

	// Set up a simple table
	_, err := engine.Execute("CREATE TABLE users (id INT, name VARCHAR(50), age INT)")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Query Recording Example ===")
	fmt.Println()

	// Start recording
	fmt.Println("1. Starting query recording...")
	engine.StartRecording()

	// Execute multiple queries while recording
	queries := []string{
		"INSERT INTO users VALUES (1, 'Alice', 25)",
		"INSERT INTO users VALUES (2, 'Bob', 30)",
		"INSERT INTO users VALUES (3, 'Charlie', 35)",
		"SELECT * FROM users WHERE age > 28",
		"UPDATE users SET age = age + 1 WHERE name = 'Alice'",
		"DELETE FROM users WHERE age > 35",
	}

	fmt.Println("2. Executing queries while recording is active:")
	for i, query := range queries {
		fmt.Printf("   %d. %s\n", i+1, query)
		result, err := engine.Execute(query)
		if err != nil {
			fmt.Printf("      Error: %v\n", err)
		} else {
			// Show abbreviated results
			switch r := result.(type) {
			case *mist.SelectResult:
				fmt.Printf("      -> %d rows returned\n", len(r.Rows))
			default:
				fmt.Printf("      -> %v\n", result)
			}
		}
	}
	fmt.Println()

	// Stop recording
	fmt.Println("3. Stopping query recording...")
	engine.EndRecording()

	// Get recorded queries
	fmt.Println("4. Getting recorded queries:")
	recordedQueries := engine.GetRecordedQueries()
	fmt.Printf("   Total recorded queries: %d\n", len(recordedQueries))
	for i, query := range recordedQueries {
		fmt.Printf("   %d. %s\n", i+1, query)
	}
	fmt.Println()

	// Execute more queries after recording stopped
	fmt.Println("5. Executing queries after recording stopped:")
	fmt.Println("   INSERT INTO users VALUES (4, 'David', 28)")
	_, err = engine.Execute("INSERT INTO users VALUES (4, 'David', 28)")
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
	} else {
		fmt.Println("   -> Insert successful")
	}

	// Verify recording count hasn't changed
	finalQueries := engine.GetRecordedQueries()
	fmt.Printf("   Final recorded query count: %d (should be same as before)\n", len(finalQueries))
	fmt.Println()

	// Demonstrate starting a new recording session
	fmt.Println("6. Starting a new recording session (clears previous recordings):")
	engine.StartRecording()
	engine.Execute("SELECT COUNT(*) FROM users")
	engine.Execute("SELECT * FROM users ORDER BY age")
	engine.EndRecording()

	newRecordedQueries := engine.GetRecordedQueries()
	fmt.Printf("   New recording session captured %d queries:\n", len(newRecordedQueries))
	for i, query := range newRecordedQueries {
		fmt.Printf("   %d. %s\n", i+1, query)
	}

	fmt.Println()
	fmt.Println("=== Recording Example Complete ===")
	fmt.Println("Key features demonstrated:")
	fmt.Println("• StartRecording() - begins capturing queries")
	fmt.Println("• EndRecording() - stops capturing queries")
	fmt.Println("• GetRecordedQueries() - retrieves all captured queries")
	fmt.Println("• Thread-safe recording with mutex protection")
	fmt.Println("• Each new recording session clears previous recordings")
}

func main() {
	recordingDemo()
}
