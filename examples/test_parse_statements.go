package main

import (
	"fmt"

	"github.com/abbychau/mist/mist"
)

func main() {
	engine := mist.NewSQLEngine()

	// Test statements to see what AST types they parse to
	testStatements := []string{
		"SET SESSION TRANSACTION ISOLATION LEVEL READ UNCOMMITTED;",
		"SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;",
		"LOCK TABLES users READ;",
		"LOCK TABLES users WRITE, products READ;",
		"UNLOCK TABLES;",
		"CREATE TABLE test (id INT, name VARCHAR(50), CHECK (id > 0));",
	}

	for _, sql := range testStatements {
		fmt.Printf("Testing: %s\n", sql)
		result, err := engine.Execute(sql)
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
		} else {
			fmt.Printf("  Result: %v\n", result)
		}
		fmt.Println()
	}
}