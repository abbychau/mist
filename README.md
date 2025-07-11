# Mist - In-Memory MySQL Database

Mist is a lightweight, in-memory SQL database engine that supports MySQL-compatible syntax. It's built using the TiDB parser and provides a simple way to run SQL queries in memory without requiring a full database server.

## Features

- **MySQL-compatible SQL syntax** using TiDB parser
- **In-memory storage** for fast operations
- **Basic SQL operations**: CREATE TABLE, INSERT, SELECT, UPDATE, DELETE
- **Transaction support**: START TRANSACTION, BEGIN, COMMIT, ROLLBACK with nested transactions and savepoints
- **Scalar subqueries**: Support for single-value subqueries in SELECT and WHERE clauses
- **WHERE clauses** with comparison operators and pattern matching (LIKE, NOT LIKE)
- **JOIN operations** between tables (including comma-separated table joins)
- **Aggregate functions**: COUNT, SUM, AVG, MIN, MAX
- **LIMIT clause** with offset support
- **Subqueries** in FROM clause and EXISTS/NOT EXISTS conditions
- **ALTER TABLE** operations (ADD/DROP/MODIFY columns)
- **Index support** for query optimization
- **Auto increment ID columns** for primary keys
- **Interactive mode** for testing queries
- **Daemon mode**: MySQL-compatible server that listens on port 3306
- **Thread-safe operations**
- **Library support** for embedding in Go applications
- **Enhanced MySQL compatibility** with graceful handling of ENUM, FOREIGN KEY, and UNIQUE constraints

## Installation

### As a Standalone Application
```bash
git clone <repository-url>
cd mist
go mod tidy
go build -o mistdb .
# Note: Use -o mistdb to avoid naming conflict with the mist/ folder
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

#### Auto Increment Example
```go
package main

import (
    "fmt"
    "log"

    "github.com/abbychau/mist"
)

func main() {
    engine := mist.NewSQLEngine()

    // Create a table with auto increment primary key
    _, err := engine.Execute(`CREATE TABLE products (
        id INT AUTO_INCREMENT PRIMARY KEY,
        name VARCHAR(100),
        price FLOAT
    )`)
    if err != nil {
        log.Fatal(err)
    }

    // Insert data without specifying ID (auto increment will handle it)
    insertQueries := []string{
        "INSERT INTO products (name, price) VALUES ('Laptop', 999.99)",
        "INSERT INTO products (name, price) VALUES ('Mouse', 29.99)",
        "INSERT INTO products (name, price) VALUES ('Keyboard', 79.99)",
    }

    for _, query := range insertQueries {
        result, err := engine.Execute(query)
        if err != nil {
            log.Fatal(err)
        }
        fmt.Printf("Inserted record with ID: %v\n", result)
    }

    // Query all products
    result, err := engine.Execute("SELECT * FROM products ORDER BY id")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("\nAll products:")
    mist.PrintResult(result)
}
```

#### Advanced Library Usage
```go
package main

import (
    "fmt"
    "log"

    "github.com/abbychau/mist"
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

    "github.com/abbychau/mist"
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

Daemon mode (MySQL-compatible server):
```bash
# Run on default port 3306
go run . -d

# Run on custom port
go run . -d --port 3307

# Show help
go run . --help
```

### Daemon Mode

Mist can run as a MySQL-compatible daemon server, allowing you to connect with standard MySQL clients or tools. The daemon uses a simplified text protocol that supports all Mist SQL features.

#### Starting the Daemon

The daemon can be started with several command-line options:

```bash
# Start on default MySQL port (3306)
go run . -d
# or
go run . --daemon

# Start on custom port
go run . -d --port 3307
go run . --daemon --port 8080

# Show all available options
go run . --help
```

**Command-line flags:**
- `-d, --daemon`: Enable daemon mode
- `--port`: Specify port number (default: 3306)
- `-i`: Interactive mode (cannot be used with daemon mode)


#### Protocol Details

The daemon uses a **simplified text-based protocol**:

1. **Input**: Each line sent to the server is treated as a SQL command
2. **Output**: Results are formatted as readable tables with timing information
3. **Termination**: Commands should end with semicolon (`;`) but it's optional
4. **Error handling**: Invalid queries return error messages instead of crashing

**Protocol Flow:**
```
Client connects -> Welcome message
Client sends: CREATE TABLE test (id INT);
Server responds: Table test created successfully
                Query OK (123.4µs)
                mist>

Client sends: SELECT * FROM test;
Server responds: Empty set (45.2µs)
                mist>

Client sends: quit
Server responds: Bye!
Connection closes
```



#### Production Usage

For production-like usage, you can:

```bash
# Build the binary first
go build -o mistdb .

# Run daemon in background
./mistdb -d --port 3306 &

# Or run with nohup for persistent operation
nohup ./mistdb -d --port 3306 > mist.log 2>&1 &

# Stop daemon gracefully
kill -TERM <pid>
# or use Ctrl+C if running in foreground
```

#### Example Session

```
$ telnet localhost 3306
Trying 127.0.0.1...
Connected to localhost.
Escape character is '^]'.
Welcome to Mist MySQL-compatible database (Connection #1)
Type 'help' for commands, 'quit' to exit
mist> CREATE TABLE products (id INT, name VARCHAR(50), price DECIMAL(10,2));
Table products created successfully
Query OK (234.5µs)

mist> INSERT INTO products VALUES (1, 'Laptop', 999.99), (2, 'Mouse', 29.99);
Insert successful
Query OK (123.2µs)

mist> SELECT * FROM products;
+----+--------+--------+
| id | name   | price  |
+----+--------+--------+
| 1  | Laptop | 999.99 |
| 2  | Mouse  | 29.99  |
+----+--------+--------+
2 rows in set (89.3µs)

mist> SELECT name FROM products WHERE price > (SELECT AVG(price) FROM products);
+--------+
| name   |
+--------+
| Laptop |
+--------+
1 row in set (156.7µs)

mist> help
Available SQL commands:
- CREATE TABLE, ALTER TABLE, DROP TABLE
- INSERT, SELECT, UPDATE, DELETE  
- START TRANSACTION, COMMIT, ROLLBACK
- CREATE INDEX, DROP INDEX, SHOW INDEX
- SHOW TABLES
Type 'quit' to exit

mist> quit
Bye!
Connection closed by foreign host.
```

#### Integration Examples

The daemon can be used with various tools and scripts:

```bash
# Simple automation script
echo "SELECT COUNT(*) FROM users;" | nc localhost 3306

# Batch operations
cat << EOF | nc localhost 3306
CREATE TABLE test (id INT, value VARCHAR(50));
INSERT INTO test VALUES (1, 'hello'), (2, 'world');
SELECT * FROM test;
quit
EOF

# Using with expect scripts for automation
expect << 'EOF'
spawn telnet localhost 3306
expect "mist>"
send "CREATE TABLE automation (id INT);\r"
expect "mist>"
send "quit\r"
expect eof
EOF
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

// ImportSQLFile reads a .sql file and executes all SQL statements in it
func (engine *SQLEngine) ImportSQLFile(filename string) ([]interface{}, error)

// ImportSQLFileFromReader reads SQL statements from an io.Reader and executes them
func (engine *SQLEngine) ImportSQLFileFromReader(reader io.Reader) ([]interface{}, error)

// ImportSQLFileWithProgress reads a .sql file and executes statements with progress reporting
func (engine *SQLEngine) ImportSQLFileWithProgress(filename string, progressCallback func(current, total int, statement string)) ([]interface{}, error)

// GetDatabase returns the underlying database (for advanced usage)
func (engine *SQLEngine) GetDatabase() *Database

// Version returns the current version
func Version() string

// PrintResult prints query results in a formatted table
func PrintResult(result interface{})

// Interactive starts an interactive SQL session
func Interactive(engine *SQLEngine)
```

### SQL File Import

Mist supports importing SQL files containing multiple statements. This is useful for:
- Setting up database schemas
- Loading sample data
- Running migration scripts
- Batch operations

#### Basic SQL File Import

```go
package main

import (
    "fmt"
    "log"
    "github.com/abbychau/mist"
)

func main() {
    engine := mist.NewSQLEngine()

    // Import a SQL file
    results, err := engine.ImportSQLFile("schema.sql")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Executed %d statements\n", len(results))
}
```

#### Import from String or Reader

```go
// Import from string
sqlContent := `
    CREATE TABLE users (id INT, name VARCHAR(50));
    INSERT INTO users VALUES (1, 'Alice');
    INSERT INTO users VALUES (2, 'Bob');
`

results, err := engine.ImportSQLFileFromReader(strings.NewReader(sqlContent))
if err != nil {
    log.Fatal(err)
}
```

#### Import with Progress Reporting

```go
// Progress callback function
progressCallback := func(current, total int, statement string) {
    fmt.Printf("Executing %d/%d: %s\n", current, total, statement)
}

results, err := engine.ImportSQLFileWithProgress("large_dataset.sql", progressCallback)
if err != nil {
    log.Fatal(err)
}
```

#### Features

- **Automatic statement separation**: Handles multiple SQL statements separated by semicolons
- **Comment filtering**: Ignores SQL comments (-- and #)
- **Error handling**: Provides detailed error messages with statement numbers
- **Progress reporting**: Optional progress callbacks for large files
- **Flexible input**: Support for files, strings, and io.Reader interfaces

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

-- Create table with auto increment primary key
CREATE TABLE products (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100),
    price FLOAT,
    category_id INT
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

-- Insert with auto increment (ID will be automatically assigned)
INSERT INTO products (name, price, category_id) VALUES ('Laptop', 999.99, 1);
INSERT INTO products (name, price, category_id) VALUES ('Mouse', 29.99, 2);

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

#### Transaction Support
```sql
-- Basic transactions
START TRANSACTION;
-- or alternatively
BEGIN;

-- Perform multiple operations
INSERT INTO users VALUES (1, 'Alice', 30);
UPDATE users SET age = 31 WHERE name = 'Alice';
DELETE FROM users WHERE age < 25;

-- Commit the transaction (make changes permanent)
COMMIT;

-- Or rollback to undo all changes since transaction started
-- ROLLBACK;

-- Nested transactions (supported)
START TRANSACTION;
  INSERT INTO users VALUES (2, 'Bob', 25);

  BEGIN; -- Start nested transaction
    INSERT INTO users VALUES (3, 'Charlie', 35);
    UPDATE users SET age = 26 WHERE name = 'Bob';
  ROLLBACK; -- Rollback nested transaction only

COMMIT; -- Commit outer transaction (Bob remains, Charlie is gone)

-- Savepoints within transactions
START TRANSACTION;
  INSERT INTO users VALUES (4, 'David', 40);

  SAVEPOINT sp1; -- Create savepoint
    INSERT INTO users VALUES (5, 'Eve', 28);
    UPDATE users SET age = 41 WHERE name = 'David';
  ROLLBACK TO SAVEPOINT sp1; -- Rollback to savepoint (Eve is gone, David age restored)

  RELEASE SAVEPOINT sp1; -- Release savepoint
COMMIT; -- Commit transaction
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
- `DECIMAL(precision, scale)` - Fixed-point decimal numbers
- `TIMESTAMP` - Date and time values
- `DATE` - Date values
- `ENUM` - Enumerated values (stored as VARCHAR for compatibility)

## Column Constraints

- `PRIMARY KEY` - Designates a column as the primary key
- `AUTO_INCREMENT` - Automatically generates sequential integer values (must be used with PRIMARY KEY)
- `NOT NULL` - Ensures column values cannot be null

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
- **Limited SQL features**: Subset of MySQL functionality
- **No user management**: No authentication or authorization
- **Single-node**: No distributed or clustering support
- **FOREIGN KEY constraints**: Parsed but not enforced (for compatibility)
- **UNIQUE constraints**: Parsed but not enforced (for compatibility)
- **ON UPDATE triggers**: Parsed but not executed (for compatibility)


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

