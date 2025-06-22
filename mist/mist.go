// package mist provides an in-memory MySQL-compatible database engine.
//
// Mist is a lightweight, thread-safe SQL database that supports basic SQL operations
// including CREATE TABLE, INSERT, SELECT, UPDATE, DELETE, JOIN, and more.
// It uses the TiDB parser for MySQL-compatible SQL parsing.
//
// Example usage:
//
//	engine := mist.NewSQLEngine()
//
//	// Create a table
//	_, err := engine.Execute("CREATE TABLE users (id INT, name VARCHAR(50), age INT)")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Insert data
//	_, err = engine.Execute("INSERT INTO users VALUES (1, 'Alice', 30)")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Query data
//	result, err := engine.Execute("SELECT * FROM users WHERE age > 25")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Print results
//	mist.PrintResult(result)
//
//	// Import SQL files
//	results, err := engine.ImportSQLFile("schema.sql")
//	if err != nil {
//		log.Fatal(err)
//	}
package mist

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/pingcap/tidb/pkg/parser"
	"github.com/pingcap/tidb/pkg/parser/ast"
	_ "github.com/pingcap/tidb/pkg/types/parser_driver"
)

// Version returns the current version of the Mist database engine
func Version() string {
	return "1.0.0"
}

// parse parses a SQL statement and returns the AST
func parse(sql string) (*ast.StmtNode, error) {
	p := parser.New()

	stmtNodes, _, err := p.ParseSQL(sql)
	if err != nil {
		return nil, err
	}

	return &stmtNodes[0], nil
}

// PrintResult prints the result of a SQL execution in a user-friendly format
func PrintResult(result interface{}) {
	switch r := result.(type) {
	case *SelectResult:
		PrintSelectResult(r)
	case string:
		fmt.Println(r)
	default:
		fmt.Printf("Result: %v\n", r)
	}
}

// PrintSelectResult prints a SELECT result in a formatted table
func PrintSelectResult(result *SelectResult) {
	if len(result.Columns) == 0 {
		fmt.Println("No columns in result")
		return
	}

	// Calculate column widths
	colWidths := make([]int, len(result.Columns))
	for i, col := range result.Columns {
		colWidths[i] = len(col)
	}

	// Check data widths
	for _, row := range result.Rows {
		for i, val := range row {
			if i < len(colWidths) {
				valStr := fmt.Sprintf("%v", val)
				if len(valStr) > colWidths[i] {
					colWidths[i] = len(valStr)
				}
			}
		}
	}

	// Ensure minimum width
	for i := range colWidths {
		if colWidths[i] < 8 {
			colWidths[i] = 8
		}
	}

	// Print header
	fmt.Print("|")
	for i, col := range result.Columns {
		fmt.Printf(" %-*s |", colWidths[i], col)
	}
	fmt.Println()

	// Print separator
	fmt.Print("|")
	for i := range result.Columns {
		fmt.Print(strings.Repeat("-", colWidths[i]+2))
		fmt.Print("|")
	}
	fmt.Println()

	// Print rows
	for _, row := range result.Rows {
		fmt.Print("|")
		for i, val := range row {
			if i < len(colWidths) {
				valStr := fmt.Sprintf("%v", val)
				if val == nil {
					valStr = "NULL"
				}
				fmt.Printf(" %-*s |", colWidths[i], valStr)
			}
		}
		fmt.Println()
	}
}

// Interactive starts an interactive SQL session with the given engine
func Interactive(engine *SQLEngine) {
	fmt.Println("Mist In-Memory MySQL Database")
	fmt.Println("Type 'exit' or 'quit' to exit")
	fmt.Println("Type 'help' for help")
	fmt.Println("End statements with semicolon (;)")
	fmt.Println()

	var inputBuffer strings.Builder
	reader := bufio.NewReader(os.Stdin)

	for {
		if inputBuffer.Len() == 0 {
			fmt.Print("mist> ")
		} else {
			fmt.Print("   -> ")
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for special commands
		if inputBuffer.Len() == 0 {
			switch strings.ToLower(line) {
			case "exit", "quit":
				fmt.Println("Goodbye!")
				return
			case "help":
				printHelp()
				continue
			case "clear":
				fmt.Print("\033[2J\033[H") // Clear screen
				continue
			}
		}

		// Add line to buffer
		if inputBuffer.Len() > 0 {
			inputBuffer.WriteString(" ")
		}
		inputBuffer.WriteString(line)

		// Check if statement is complete (ends with semicolon)
		if strings.HasSuffix(line, ";") {
			input := inputBuffer.String()
			inputBuffer.Reset()

			result, err := engine.Execute(input)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				PrintResult(result)
			}
			fmt.Println()
		}
	}
}

// printHelp prints help information
func printHelp() {
	fmt.Println("Supported SQL statements:")
	fmt.Println("  CREATE TABLE table_name (column_name column_type, ...);")
	fmt.Println("  ALTER TABLE table_name ADD COLUMN column_name column_type;")
	fmt.Println("  ALTER TABLE table_name DROP COLUMN column_name;")
	fmt.Println("  ALTER TABLE table_name MODIFY COLUMN column_name new_type;")
	fmt.Println("  INSERT INTO table_name VALUES (value1, value2, ...);")
	fmt.Println("  INSERT INTO table_name (col1, col2) VALUES (val1, val2);")
	fmt.Println("  SELECT * FROM table_name;")
	fmt.Println("  SELECT col1, col2 FROM table_name WHERE condition LIMIT 10;")
	fmt.Println("  SELECT col1, col2 FROM table_name LIMIT 5, 10;")
	fmt.Println("  SELECT COUNT(*), SUM(col), AVG(col) FROM table_name;")
	fmt.Println("  SELECT * FROM table1 JOIN table2 ON condition;")
	fmt.Println("  SELECT * FROM table1, table2 WHERE table1.id = table2.foreign_id;")
	fmt.Println("  SELECT * FROM (SELECT * FROM table1) AS subquery;")
	fmt.Println("  UPDATE table_name SET col1 = value1 WHERE condition;")
	fmt.Println("  DELETE FROM table_name WHERE condition;")
	fmt.Println("  CREATE INDEX index_name ON table_name (column_name);")
	fmt.Println("  DROP INDEX index_name;")
	fmt.Println("  SHOW TABLES;")
	fmt.Println("  SHOW INDEX FROM table_name;")
	fmt.Println()
	fmt.Println("Supported column types:")
	fmt.Println("  INT, VARCHAR(length), TEXT, FLOAT, BOOL")
	fmt.Println()
	fmt.Println("Supported aggregate functions:")
	fmt.Println("  COUNT(*), COUNT(column), SUM(column), AVG(column), MIN(column), MAX(column)")
	fmt.Println()
	fmt.Println("LIMIT clause:")
	fmt.Println("  LIMIT count - limit to 'count' rows")
	fmt.Println("  LIMIT offset, count - skip 'offset' rows, then return 'count' rows")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50), age INT);")
	fmt.Println("  ALTER TABLE users ADD COLUMN email VARCHAR(100);")
	fmt.Println("  CREATE INDEX idx_age ON users (age);")
	fmt.Println("  INSERT INTO users VALUES (1, 'Alice', 30, 'alice@example.com');")
	fmt.Println("  SELECT * FROM users WHERE age > 25 LIMIT 5;")
	fmt.Println("  SELECT COUNT(*) FROM users WHERE age > 25;")
	fmt.Println("  SELECT AVG(age) FROM users;")
	fmt.Println("  UPDATE users SET age = age + 1 WHERE name = 'Alice';")
	fmt.Println("  DELETE FROM users WHERE age < 18;")
	fmt.Println("  SELECT u.name, p.title FROM users u JOIN posts p ON u.id = p.user_id LIMIT 10;")
}
