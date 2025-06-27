# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Mist is an in-memory MySQL-compatible database engine written in Go. It provides a lightweight SQL database that supports basic SQL operations without requiring a full database server installation. The project uses the TiDB parser for MySQL-compatible SQL parsing.

## Development Commands

### Building and Running
```bash
# Build the project
go build .

# Run the demo (default mode)
go run .

# Run in interactive mode
go run . -i

# Install dependencies
go mod tidy
```

### Testing
```bash
# Run all tests
go test -v ./mist

# Run specific tests
go test -v -run TestCreateTable ./mist

# Run tests with race detection
go test -race ./mist
```

### WebAssembly Build
```bash
# Build WASM version for web playground
./build-wasm.sh

# Manual build (from project root)
cd wasm && GOOS=js GOARCH=wasm go build -o ../docs/mist.wasm .
```

## Architecture Overview

### Core Components

**SQLEngine** (`mist/engine.go`)
- Main execution engine that orchestrates SQL parsing and execution
- Handles transaction management including nested transactions and savepoints
- Manages query recording functionality
- Thread-safe with mutex protection for concurrent operations

**Database** (`mist/database.go`)
- In-memory storage manager for tables and data
- Handles table creation, retrieval, and foreign key constraints
- Manages unique indexes and validation
- Thread-safe operations with RWMutex

**Table Structure**
- Tables store column definitions and row data in memory
- Support for auto-increment columns with thread-safe counters
- Unique constraints and primary key enforcement
- Foreign key support with CASCADE, SET NULL, SET DEFAULT actions

**Query Execution Modules**
- `create_table.go` - CREATE TABLE statement execution
- `insert.go` - INSERT statement with auto-increment support
- `select.go` - SELECT with WHERE, LIMIT, aggregate functions, subqueries
- `update.go` - UPDATE statements with expression evaluation
- `delete.go` - DELETE with foreign key cascade handling
- `join.go` - JOIN operations (INNER, LEFT, RIGHT, CROSS)
- `alter_table.go` - ALTER TABLE operations (ADD/DROP/MODIFY columns)

**Index System** (`mist/index.go`, `mist/index_commands.go`)
- Hash-based indexing for query optimization
- Support for CREATE INDEX and DROP INDEX operations
- Automatic index usage in SELECT queries for performance

**Expression Evaluation**
- WHERE clause evaluation with comparison operators
- Arithmetic expressions in UPDATE statements
- Aggregate functions: COUNT, SUM, AVG, MIN, MAX

### Data Types Support
- INT, VARCHAR(length), TEXT, FLOAT, BOOL
- DECIMAL(precision, scale), TIMESTAMP, DATE
- ENUM (stored as VARCHAR for compatibility)
- Auto-increment support for primary keys

### Transaction System
- Full nested transaction support with BEGIN/COMMIT/ROLLBACK
- Savepoint functionality (SAVEPOINT, ROLLBACK TO SAVEPOINT, RELEASE SAVEPOINT)
- Thread-safe transaction management
- Deep copying of table states for rollback capabilities

### SQL Compatibility
The engine supports a significant subset of MySQL syntax:
- Table operations: CREATE TABLE, ALTER TABLE
- Data operations: INSERT, SELECT, UPDATE, DELETE
- Join operations: explicit JOINs and comma-separated tables
- Aggregate functions and subqueries
- Index management: CREATE INDEX, DROP INDEX, SHOW INDEX
- Transaction control: START TRANSACTION, BEGIN, COMMIT, ROLLBACK
- Utility commands: SHOW TABLES

### WebAssembly Playground
Mist includes a web-based SQL playground that runs the engine compiled to WebAssembly:

**Features:**
- Full MySQL-compatible SQL engine running in the browser
- Real-time query execution without server backend
- CodeMirror SQL editor with syntax highlighting
- Interactive query results display
- Pre-built example queries for common operations

**Files:**
- `docs/playground.html` - Web playground interface
- `docs/mist.wasm` - Compiled WebAssembly binary
- `wasm/` - WASM-specific Go source code
- `build-wasm.sh` - Build script for WASM compilation

**WASM Engine (`wasm/wasm_engine.go`):**
- Standalone SQL engine implementation (no external dependencies)
- Shares core SQL parsing logic with main engine but avoids TiDB dependencies
- Proper INSERT VALUES parsing for real data insertion
- Support for multiple row inserts in single statement
- Type conversion between SQL types and JavaScript values
- Thread-safe operations optimized for browser environment
- Small binary size (~2.5MB) for fast loading

### Key Design Patterns

**Thread Safety**
- All database operations use RWMutex for concurrent access
- Tables have individual mutexes for fine-grained locking
- Transaction state is protected with dedicated mutexes

**Parsing and Execution Flow**
1. SQL parsing using TiDB parser to generate AST
2. Statement type detection and routing
3. Execution by specialized handler functions
4. Result formatting and return

**Error Handling**
- Comprehensive validation for data types and constraints
- Foreign key constraint validation and enforcement
- Transaction rollback on errors
- Detailed error messages with context

## Important Implementation Notes

### Foreign Key Handling
- Foreign keys are parsed and stored but enforcement can be configured
- Support for all standard FK actions (CASCADE, SET NULL, SET DEFAULT, RESTRICT, NO ACTION)
- Validation occurs before data modification operations

### Auto-Increment Behavior
- Auto-increment values are thread-safe with atomic operations
- Counter continues from the highest explicitly inserted ID
- Works with both `INSERT INTO table VALUES` and `INSERT INTO table (cols) VALUES` syntax

### Memory Management
- All data is stored in memory - no persistence to disk
- Table snapshots for transaction rollback require memory for copies
- Consider memory usage for large datasets

### Performance Considerations
- Hash-based indexes provide O(1) lookup for equality operations
- Nested transactions create table copies - expensive for large tables
- JOIN operations perform nested loops - optimize with proper indexing

## Common Development Patterns

When extending functionality:

1. **Adding New SQL Statements**
   - Add AST case handling in `engine.go` Execute method
   - Create dedicated execution function in appropriate module
   - Handle transaction safety if data is modified

2. **Adding New Data Types**
   - Extend ColumnType enum in `database.go`
   - Add validation logic in Table.validateValue method
   - Update type conversion logic in relevant execution modules

3. **Extending Transaction Support**
   - Transaction data modifications should be tracked for rollback
   - Use existing mutex patterns for thread safety
   - Test nested transaction scenarios thoroughly

The codebase follows clean separation of concerns with each SQL operation having its dedicated module while sharing common database and table abstractions.