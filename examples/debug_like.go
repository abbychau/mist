package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/abbychau/mist/mist"
)

func convertLikePatternToRegex(likePattern string) string {
	// First handle escaped wildcards (literal % and _ in the pattern)
	// In SQL, \% matches literal %, \_ matches literal _
	// Replace them with placeholders to avoid confusion
	escaped := strings.ReplaceAll(likePattern, "\\%", "__LITERAL_PERCENT__")
	escaped = strings.ReplaceAll(escaped, "\\_", "__LITERAL_UNDERSCORE__")
	
	// Replace SQL wildcards with special placeholders that won't be escaped
	escaped = strings.ReplaceAll(escaped, "%", "__WILDCARD_PERCENT__")
	escaped = strings.ReplaceAll(escaped, "_", "__WILDCARD_UNDERSCORE__")
	
	// Now escape all special regex characters
	escaped = regexp.QuoteMeta(escaped)
	
	// Replace placeholders with actual regex patterns
	escaped = strings.ReplaceAll(escaped, "__WILDCARD_PERCENT__", ".*")
	escaped = strings.ReplaceAll(escaped, "__WILDCARD_UNDERSCORE__", ".")
	
	// Restore escaped wildcards as literal characters
	escaped = strings.ReplaceAll(escaped, "__LITERAL_PERCENT__", "%")
	escaped = strings.ReplaceAll(escaped, "__LITERAL_UNDERSCORE__", "_")
	
	// Anchor the pattern to match the entire string
	return "^" + escaped + "$"
}

func main() {
	engine := mist.NewSQLEngine()

	// Create test table and data
	_, err := engine.Execute("CREATE TABLE test (name VARCHAR(255))")
	if err != nil {
		log.Fatal(err)
	}

	_, err = engine.Execute("INSERT INTO test VALUES ('Apple iPhone 15')")
	if err != nil {
		log.Fatal(err)
	}

	// Test manual pattern conversion
	testPattern := "Apple%"
	regexPattern := convertLikePatternToRegex(testPattern)
	fmt.Printf("Original pattern: %s\n", testPattern)
	fmt.Printf("Regex pattern: %s\n", regexPattern)

	// Test matching
	testString := "Apple iPhone 15"
	matched, err := regexp.MatchString(regexPattern, testString)
	if err != nil {
		fmt.Printf("Error matching: %v\n", err)
	} else {
		fmt.Printf("Does '%s' match pattern '%s'? %v\n", testString, testPattern, matched)
	}

	// Test with database
	fmt.Println("\nTesting with database:")
	result, err := engine.Execute("SELECT name FROM test WHERE name LIKE 'Apple%'")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		selectResult := result.(*mist.SelectResult)
		fmt.Printf("Query result: %d rows\n", len(selectResult.Rows))
		for _, row := range selectResult.Rows {
			fmt.Printf("  %v\n", row)
		}
	}

	// Test simple case
	fmt.Println("\nTesting simple pattern:")
	result, err = engine.Execute("SELECT name FROM test WHERE name = 'Apple iPhone 15'")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		selectResult := result.(*mist.SelectResult)
		fmt.Printf("Exact match result: %d rows\n", len(selectResult.Rows))
		for _, row := range selectResult.Rows {
			fmt.Printf("  %v\n", row)
		}
	}
}