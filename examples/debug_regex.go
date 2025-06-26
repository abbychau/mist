package main

import (
	"fmt"
	"regexp"
	"strings"
)

func main() {
	pattern := "Apple%"
	
	fmt.Printf("Original: %s\n", pattern)
	
	quoted := regexp.QuoteMeta(pattern)
	fmt.Printf("After QuoteMeta: %s\n", quoted)
	fmt.Printf("QuoteMeta bytes: %q\n", quoted)
	
	// Show what we need to replace
	fmt.Println("\nTesting replacements:")
	test1 := strings.ReplaceAll(quoted, "\\%", ".*")
	fmt.Printf("Replace \\\\%%: %s\n", test1)
	
	test2 := strings.ReplaceAll(quoted, "\\%", ".*")
	fmt.Printf("Replace \\%%: %s\n", test2)
}