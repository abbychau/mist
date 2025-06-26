package main

import (
	"fmt"
	"regexp"
	"strings"
)

func main() {
	pattern := "Apple%"
	
	fmt.Printf("Step 1 - Original: %s\n", pattern)
	
	// Step 1: Handle escaped wildcards
	escaped := strings.ReplaceAll(pattern, "\\%", "〈LITERAL-PERCENT〉")
	escaped = strings.ReplaceAll(escaped, "\\_", "〈LITERAL-UNDERSCORE〉")
	fmt.Printf("Step 2 - After literal escapes: %s\n", escaped)
	
	// Step 2: Replace wildcards with placeholders
	escaped = strings.ReplaceAll(escaped, "%", "〈WILDCARD-PERCENT〉")
	escaped = strings.ReplaceAll(escaped, "_", "〈WILDCARD-UNDERSCORE〉")
	fmt.Printf("Step 3 - After wildcard placeholders: %s\n", escaped)
	
	// Step 3: Escape regex characters
	escaped = regexp.QuoteMeta(escaped)
	fmt.Printf("Step 4 - After QuoteMeta: %s\n", escaped)
	
	// Step 4: Replace placeholders with regex
	escaped = strings.ReplaceAll(escaped, "〈WILDCARD-PERCENT〉", ".*")
	escaped = strings.ReplaceAll(escaped, "〈WILDCARD-UNDERSCORE〉", ".")
	fmt.Printf("Step 5 - After regex replacement: %s\n", escaped)
	
	// Step 5: Restore literals
	escaped = strings.ReplaceAll(escaped, "〈LITERAL-PERCENT〉", "%")
	escaped = strings.ReplaceAll(escaped, "〈LITERAL-UNDERSCORE〉", "_")
	fmt.Printf("Step 6 - After literal restoration: %s\n", escaped)
	
	// Step 6: Anchor
	result := "^" + escaped + "$"
	fmt.Printf("Step 7 - Final regex: %s\n", result)
	
	// Test it
	testString := "Apple iPhone 15"
	matched, err := regexp.MatchString(result, testString)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Does '%s' match? %v\n", testString, matched)
	}
}