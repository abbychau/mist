# Mist - In-Memory MySQL Database

Mist is a lightweight, in-memory SQL database engine that supports MySQL-compatible syntax. It's built using the TiDB parser and provides a simple way to run SQL queries in memory without requiring a full database server.

## Features

- **MySQL-compatible SQL syntax** using TiDB parser
- **In-memory storage** for fast operations
- **Basic SQL operations**: CREATE TABLE, INSERT, SELECT, UPDATE, DELETE
- **WHERE clauses** with comparison operators
- **JOIN operations** between tables (including comma-separated table joins)
- **Aggregate functions**: COUNT, SUM, AVG, MIN, MAX
- **LIMIT clause** with offset support
- **Subqueries** in FROM clause
- **ALTER TABLE** operations (ADD/DROP/MODIFY columns)
- **Index support** for query optimization
- **Interactive mode** for testing queries
- **Thread-safe operations**
- **Library support** for embedding in Go applications

## Installation

### As a Standalone Application
```bash
git clone <repository-url>
cd mist
go mod tidy
go build .
```

### As a Library
```bash
go get github.com/abbychau/mist
```

## Usage

### As a Library

Mist can be easily embedded in your Go applications as a library:

#### Basic Example
```go
package main

import (
    "fmt"
    "log"

    "github.com/abbychau/mist"
)

func main() {
    // Create a new SQL engine
    engine := mist.NewSQLEngine()

    // Create a table
    _, err := engine.Execute("CREATE TABLE users (id INT, name VARCHAR(50), age INT)")
    if err != nil {
        log.Fatal(err)
    }

    // Insert data
    _, err = engine.Execute("INSERT INTO users VALUES (1, 'Alice', 30)")
    if err != nil {
        log.Fatal(err)
    }

    // Query data
    result, err := engine.Execute("SELECT * FROM users WHERE age > 25")
    if err != nil {
        log.Fatal(err)
    }

    // Print results using the built-in formatter
    mist.PrintResult(result)

    // Or handle the result yourself
    if selectResult, ok := result.(*mist.SelectResult); ok {
        fmt.Printf("Found %d rows with %d columns\n",
            len(selectResult.Rows), len(selectResult.Columns))

        for i, row := range selectResult.Rows {
            fmt.Printf("Row %d: %v\n", i, row)
        }
    }
}
```

#### Advanced Library Usage
```go
package main

import (
    "fmt"
    "log"

    "github.com/abbychau/mist/mist"
)

func main() {
    engine := mist.NewSQLEngine()

    // Setup database schema
    setupQueries := []string{
        "CREATE TABLE departments (id INT, name VARCHAR(50))",
        "CREATE TABLE users (id INT, name VARCHAR(50), dept_id INT, salary FLOAT)",
        "CREATE INDEX idx_dept ON users (dept_id)",
        "CREATE INDEX idx_salary ON users (salary)",
    }

    for _, query := range setupQueries {
        if _, err := engine.Execute(query); err != nil {
            log.Fatalf("Setup failed: %v", err)
        }
    }

    // Insert sample data
    sampleData := []string{
        "INSERT INTO departments VALUES (1, 'Engineering'), (2, 'Marketing'), (3, 'Sales')",
        "INSERT INTO users VALUES (1, 'Alice', 1, 75000), (2, 'Bob', 2, 65000), (3, 'Charlie', 1, 85000)",
        "INSERT INTO users VALUES (4, 'Diana', 2, 70000), (5, 'Eve', 3, 60000)",
    }

    for _, query := range sampleData {
        if _, err := engine.Execute(query); err != nil {
            log.Fatalf("Data insertion failed: %v", err)
        }
    }

    // Execute multiple queries and handle results
    queries := map[string]string{
        "Total Users": "SELECT COUNT(*) FROM users",
        "Average Salary": "SELECT AVG(salary) FROM users",
        "High Earners": "SELECT name, salary FROM users WHERE salary > 70000",
        "Department Join": `SELECT u.name, d.name, u.salary
                           FROM users u
                           JOIN departments d ON u.dept_id = d.id
                           WHERE u.salary > 65000`,
        "Comma Join": `SELECT users.name, departments.name
                       FROM users, departments
                       WHERE users.dept_id = departments.id AND users.salary > 70000`,
    }

    for description, query := range queries {
        fmt.Printf("\n=== %s ===\n", description)
        result, err := engine.Execute(query)
        if err != nil {
            fmt.Printf("Error: %v\n", err)
            continue
        }
        mist.PrintResult(result)
    }

    // Execute multiple statements at once
    fmt.Println("\n=== Batch Operations ===")
    batchSQL := `
        UPDATE users SET salary = salary * 1.1 WHERE dept_id = 1;
        SELECT name, salary FROM users WHERE dept_id = 1;
    `

    results, err := engine.ExecuteMultiple(batchSQL)
    if err != nil {
        log.Fatalf("Batch execution failed: %v", err)
    }

    for i, result := range results {
        fmt.Printf("Result %d:\n", i+1)
        mist.PrintResult(result)
        fmt.Println()
    }

    // Access the underlying database for advanced operations
    db := engine.GetDatabase()
    tables := db.ListTables()
    fmt.Printf("Available tables: %v\n", tables)
}
```

#### Using in Web Applications
```go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "log"

    "github.com/abbychau/mist/mist"
)

type QueryRequest struct {
    SQL string `json:"sql"`
}

type QueryResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
}

func main() {
    // Initialize the database engine
    engine := mist.NewSQLEngine()

    // Setup initial schema
    _, err := engine.Execute("CREATE TABLE users (id INT, name VARCHAR(50), email VARCHAR(100))")
    if err != nil {
        log.Fatal(err)
    }

    // HTTP handler for SQL queries
    http.HandleFunc("/query", func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
            return
        }

        var req QueryRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "Invalid JSON", http.StatusBadRequest)
            return
        }

        result, err := engine.Execute(req.SQL)

        var response QueryResponse
        if err != nil {
            response = QueryResponse{Success: false, Error: err.Error()}
        } else {
            response = QueryResponse{Success: true, Data: result}
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
    })

    fmt.Println("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### Command Line Interface

Run the demo:
```bash
go run .
```

Interactive mode:
```bash
go run . -i
```

### Library API Reference

#### Core Types

```go
// SQLEngine - Main database engine
type SQLEngine struct {
    // ... internal fields
}

// SelectResult - Result of SELECT queries
type SelectResult struct {
    Columns []string        `json:"columns"`
    Rows    [][]interface{} `json:"rows"`
}
```

#### Main Functions

```go
// NewSQLEngine creates a new database engine
func NewSQLEngine() *SQLEngine

// Execute runs a single SQL statement
func (engine *SQLEngine) Execute(sql string) (interface{}, error)

// ExecuteMultiple runs multiple SQL statements separated by semicolons
func (engine *SQLEngine) ExecuteMultiple(sql string) ([]interface{}, error)

// GetDatabase returns the underlying database (for advanced usage)
func (engine *SQLEngine) GetDatabase() *Database

// Version returns the current version
func Version() string

// PrintResult prints query results in a formatted table
func PrintResult(result interface{})

// Interactive starts an interactive SQL session
func Interactive(engine *SQLEngine)
```

### Supported SQL Statements

#### Table Operations
```sql
-- Create table
CREATE TABLE users (
    id INT PRIMARY KEY,
    name VARCHAR(50),
    age INT,
    salary FLOAT
);

-- Alter table
ALTER TABLE users ADD COLUMN email VARCHAR(100);
ALTER TABLE users DROP COLUMN email;
ALTER TABLE users MODIFY COLUMN name VARCHAR(100);
```

#### Data Operations
```sql
-- Insert data
INSERT INTO users VALUES (1, 'Alice', 30, 75000.0);
INSERT INTO users (name, age) VALUES ('Bob', 25);

-- Select data
SELECT * FROM users;
SELECT name, age FROM users WHERE age > 25;
SELECT * FROM users LIMIT 10;
SELECT * FROM users LIMIT 5, 10;  -- offset 5, limit 10

-- Update data
UPDATE users SET age = 31 WHERE name = 'Alice';
UPDATE users SET salary = salary * 1.1 WHERE age > 30;

-- Delete data
DELETE FROM users WHERE age < 18;
```

#### Advanced Features
```sql
-- Aggregate functions
SELECT COUNT(*) FROM users;
SELECT AVG(salary), MAX(age) FROM users;
SELECT SUM(salary) FROM users WHERE age > 30;

-- Explicit joins
SELECT u.name, d.name
FROM users u
JOIN departments d ON u.department_id = d.id;

-- Comma-separated table joins (cross join with WHERE)
SELECT users.name, departments.name
FROM users, departments
WHERE users.department_id = departments.id;

-- Subqueries
SELECT name, salary
FROM (SELECT * FROM users WHERE age > 25) AS young_users
WHERE salary > 50000;

-- Indexes
CREATE INDEX idx_age ON users (age);
DROP INDEX idx_age;
SHOW INDEX FROM users;
```

#### Utility Commands
```sql
SHOW TABLES;
SHOW INDEX FROM table_name;
```

## Supported Data Types

- `INT` - Integer numbers
- `VARCHAR(length)` - Variable-length strings
- `TEXT` - Text data
- `FLOAT` - Floating-point numbers
- `BOOL` - Boolean values

## Architecture

Mist consists of several key components:

- **SQL Parser**: Uses TiDB's parser for MySQL-compatible SQL parsing
- **Storage Engine**: In-memory table storage with row-based data
- **Query Executor**: Handles SELECT, INSERT, UPDATE, DELETE operations
- **Join Engine**: Supports INNER, LEFT, RIGHT, and CROSS joins (including comma-separated tables)
- **Aggregate Engine**: Processes COUNT, SUM, AVG, MIN, MAX functions
- **Index Engine**: Hash-based indexing for query optimization
- **Expression Evaluator**: Handles WHERE clauses and arithmetic operations

## Performance Considerations

- **Memory Usage**: All data is stored in memory, so consider available RAM
- **Query Optimization**: The engine performs basic optimizations like index usage
- **Concurrency**: The engine is designed to be thread-safe

## Limitations

- **In-memory only**: Data is not persisted to disk
- **No transactions**: Operations are not wrapped in transactions
- **Limited SQL features**: Subset of MySQL functionality
- **No user management**: No authentication or authorization
- **Single-node**: No distributed or clustering support


## Testing

Run the test suite:
```bash
go test -v ./mist
```

Run specific tests:
```bash
go test -v -run TestCreateTable ./mist
```

## License

MIT License
