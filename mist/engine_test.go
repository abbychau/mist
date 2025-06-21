package mist

import (
	"fmt"
	"testing"
)

func TestCreateTable(t *testing.T) {
	engine := NewSQLEngine()

	// Test basic CREATE TABLE
	result, err := engine.Execute("CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50), age INT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	if result != "Table users created successfully" {
		t.Errorf("Unexpected result: %v", result)
	}

	// Test table exists
	_, err = engine.GetDatabase().GetTable("users")
	if err != nil {
		t.Errorf("Table was not created: %v", err)
	}

	// Test duplicate table creation should fail
	_, err = engine.Execute("CREATE TABLE users (id INT)")
	if err == nil {
		t.Error("Expected error for duplicate table creation")
	}

	// Test IF NOT EXISTS
	result, err = engine.Execute("CREATE TABLE IF NOT EXISTS users (id INT)")
	if err != nil {
		t.Errorf("IF NOT EXISTS should not fail: %v", err)
	}
}

func TestInsertAndSelect(t *testing.T) {
	engine := NewSQLEngine()

	// Create table
	_, err := engine.Execute("CREATE TABLE test_table (id INT, name VARCHAR(30), score FLOAT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert data
	_, err = engine.Execute("INSERT INTO test_table VALUES (1, 'Alice', 95.5)")
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	_, err = engine.Execute("INSERT INTO test_table VALUES (2, 'Bob', 87.2)")
	if err != nil {
		t.Fatalf("Failed to insert second row: %v", err)
	}

	// Test SELECT *
	result, err := engine.Execute("SELECT * FROM test_table")
	if err != nil {
		t.Fatalf("Failed to select data: %v", err)
	}

	selectResult, ok := result.(*SelectResult)
	if !ok {
		t.Fatalf("Expected SelectResult, got %T", result)
	}

	if len(selectResult.Rows) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(selectResult.Rows))
	}

	if len(selectResult.Columns) != 3 {
		t.Errorf("Expected 3 columns, got %d", len(selectResult.Columns))
	}

	// Test SELECT specific columns
	result, err = engine.Execute("SELECT name, score FROM test_table")
	if err != nil {
		t.Fatalf("Failed to select specific columns: %v", err)
	}

	selectResult = result.(*SelectResult)
	if len(selectResult.Columns) != 2 {
		t.Errorf("Expected 2 columns, got %d", len(selectResult.Columns))
	}

	if selectResult.Columns[0] != "name" || selectResult.Columns[1] != "score" {
		t.Errorf("Unexpected column names: %v", selectResult.Columns)
	}
}

func TestWhereClause(t *testing.T) {
	engine := NewSQLEngine()

	// Create and populate table
	_, err := engine.Execute("CREATE TABLE products (id INT, name VARCHAR(50), price FLOAT, in_stock INT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	queries := []string{
		"INSERT INTO products VALUES (1, 'Laptop', 999.99, 1)",
		"INSERT INTO products VALUES (2, 'Mouse', 29.99, 1)",
		"INSERT INTO products VALUES (3, 'Keyboard', 79.99, 0)",
		"INSERT INTO products VALUES (4, 'Monitor', 299.99, 1)",
	}

	for _, query := range queries {
		_, err = engine.Execute(query)
		if err != nil {
			t.Fatalf("Failed to insert data: %v", err)
		}
	}

	// Test WHERE with equality
	result, err := engine.Execute("SELECT name FROM products WHERE price = 29.99")
	if err != nil {
		t.Fatalf("Failed to execute WHERE query: %v", err)
	}

	selectResult := result.(*SelectResult)
	if len(selectResult.Rows) != 1 {
		t.Errorf("Expected 1 row, got %d", len(selectResult.Rows))
	}

	if selectResult.Rows[0][0] != "Mouse" {
		t.Errorf("Expected 'Mouse', got %v", selectResult.Rows[0][0])
	}

	// Test WHERE with greater than
	result, err = engine.Execute("SELECT name FROM products WHERE price > 100")
	if err != nil {
		t.Fatalf("Failed to execute WHERE > query: %v", err)
	}

	selectResult = result.(*SelectResult)
	if len(selectResult.Rows) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(selectResult.Rows))
	}

	// Test WHERE with boolean condition
	result, err = engine.Execute("SELECT name FROM products WHERE in_stock = 1")
	if err != nil {
		t.Fatalf("Failed to execute WHERE boolean query: %v", err)
	}

	selectResult = result.(*SelectResult)
	if len(selectResult.Rows) != 3 {
		t.Errorf("Expected 3 rows, got %d", len(selectResult.Rows))
	}
}

func TestShowTables(t *testing.T) {
	engine := NewSQLEngine()

	// Create multiple tables
	tables := []string{"users", "products", "orders"}
	for _, table := range tables {
		_, err := engine.Execute(fmt.Sprintf("CREATE TABLE %s (id INT)", table))
		if err != nil {
			t.Fatalf("Failed to create table %s: %v", table, err)
		}
	}

	// Test SHOW TABLES
	result, err := engine.Execute("SHOW TABLES")
	if err != nil {
		t.Fatalf("Failed to show tables: %v", err)
	}

	selectResult := result.(*SelectResult)
	if len(selectResult.Rows) != 3 {
		t.Errorf("Expected 3 tables, got %d", len(selectResult.Rows))
	}
}

func TestColumnTypes(t *testing.T) {
	engine := NewSQLEngine()

	// Create table with various column types
	_, err := engine.Execute("CREATE TABLE type_test (id INT, name VARCHAR(100), description TEXT, price FLOAT, active BOOL)")
	if err != nil {
		t.Fatalf("Failed to create table with various types: %v", err)
	}

	// Insert data with different types
	_, err = engine.Execute("INSERT INTO type_test VALUES (42, 'Test Product', 'A test description', 19.99, 1)")
	if err != nil {
		t.Fatalf("Failed to insert typed data: %v", err)
	}

	// Verify data
	result, err := engine.Execute("SELECT * FROM type_test")
	if err != nil {
		t.Fatalf("Failed to select typed data: %v", err)
	}

	selectResult := result.(*SelectResult)
	if len(selectResult.Rows) != 1 {
		t.Errorf("Expected 1 row, got %d", len(selectResult.Rows))
	}

	row := selectResult.Rows[0]
	if len(row) != 5 {
		t.Errorf("Expected 5 columns, got %d", len(row))
	}
}

func TestErrorHandling(t *testing.T) {
	engine := NewSQLEngine()

	// Test selecting from non-existent table
	_, err := engine.Execute("SELECT * FROM non_existent")
	if err == nil {
		t.Error("Expected error for non-existent table")
	}

	// Test inserting into non-existent table
	_, err = engine.Execute("INSERT INTO non_existent VALUES (1)")
	if err == nil {
		t.Error("Expected error for inserting into non-existent table")
	}

	// Create table for further tests
	_, err = engine.Execute("CREATE TABLE error_test (id INT, name VARCHAR(10))")
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Test column count mismatch
	_, err = engine.Execute("INSERT INTO error_test VALUES (1)")
	if err == nil {
		t.Error("Expected error for column count mismatch")
	}

	// Test selecting non-existent column
	_, err = engine.Execute("SELECT non_existent_column FROM error_test")
	if err == nil {
		t.Error("Expected error for non-existent column")
	}
}

func TestUpdateStatement(t *testing.T) {
	engine := NewSQLEngine()

	// Create and populate table
	_, err := engine.Execute("CREATE TABLE update_test (id INT, name VARCHAR(50), age INT, salary FLOAT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	queries := []string{
		"INSERT INTO update_test VALUES (1, 'Alice', 30, 50000.0)",
		"INSERT INTO update_test VALUES (2, 'Bob', 25, 45000.0)",
		"INSERT INTO update_test VALUES (3, 'Charlie', 35, 60000.0)",
	}

	for _, query := range queries {
		_, err = engine.Execute(query)
		if err != nil {
			t.Fatalf("Failed to insert data: %v", err)
		}
	}

	// Test UPDATE with WHERE clause
	result, err := engine.Execute("UPDATE update_test SET age = 31 WHERE name = 'Alice'")
	if err != nil {
		t.Fatalf("Failed to execute UPDATE: %v", err)
	}

	if result != "Updated 1 row(s)" {
		t.Errorf("Expected 'Updated 1 row(s)', got %v", result)
	}

	// Verify the update
	selectResult, err := engine.Execute("SELECT age FROM update_test WHERE name = 'Alice'")
	if err != nil {
		t.Fatalf("Failed to verify update: %v", err)
	}

	sr := selectResult.(*SelectResult)
	if len(sr.Rows) != 1 || sr.Rows[0][0] != int64(31) {
		t.Errorf("Update verification failed: expected age 31, got %v", sr.Rows[0][0])
	}

	// Test UPDATE with arithmetic
	_, err = engine.Execute("UPDATE update_test SET salary = salary * 1.1 WHERE age > 30")
	if err != nil {
		t.Fatalf("Failed to execute arithmetic UPDATE: %v", err)
	}

	// Test UPDATE all rows
	result, err = engine.Execute("UPDATE update_test SET age = age + 1")
	if err != nil {
		t.Fatalf("Failed to execute UPDATE all: %v", err)
	}

	if result != "Updated 3 row(s)" {
		t.Errorf("Expected 'Updated 3 row(s)', got %v", result)
	}
}

func TestDeleteStatement(t *testing.T) {
	engine := NewSQLEngine()

	// Create and populate table
	_, err := engine.Execute("CREATE TABLE delete_test (id INT, name VARCHAR(50), active BOOL)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	queries := []string{
		"INSERT INTO delete_test VALUES (1, 'Alice', 1)",
		"INSERT INTO delete_test VALUES (2, 'Bob', 0)",
		"INSERT INTO delete_test VALUES (3, 'Charlie', 1)",
		"INSERT INTO delete_test VALUES (4, 'David', 0)",
	}

	for _, query := range queries {
		_, err = engine.Execute(query)
		if err != nil {
			t.Fatalf("Failed to insert data: %v", err)
		}
	}

	// Test DELETE with WHERE clause
	result, err := engine.Execute("DELETE FROM delete_test WHERE active = 0")
	if err != nil {
		t.Fatalf("Failed to execute DELETE: %v", err)
	}

	if result != "Deleted 2 row(s)" {
		t.Errorf("Expected 'Deleted 2 row(s)', got %v", result)
	}

	// Verify the deletion
	selectResult, err := engine.Execute("SELECT COUNT(*) FROM delete_test")
	if err != nil {
		// COUNT is not implemented yet, so let's check differently
		selectResult, err = engine.Execute("SELECT * FROM delete_test")
		if err != nil {
			t.Fatalf("Failed to verify deletion: %v", err)
		}

		sr := selectResult.(*SelectResult)
		if len(sr.Rows) != 2 {
			t.Errorf("Expected 2 remaining rows, got %d", len(sr.Rows))
		}
	}

	// Test DELETE specific row
	result, err = engine.Execute("DELETE FROM delete_test WHERE name = 'Alice'")
	if err != nil {
		t.Fatalf("Failed to execute specific DELETE: %v", err)
	}

	if result != "Deleted 1 row(s)" {
		t.Errorf("Expected 'Deleted 1 row(s)', got %v", result)
	}
}

func TestJoinOperations(t *testing.T) {
	engine := NewSQLEngine()

	// Create users table
	_, err := engine.Execute("CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50), department_id INT)")
	if err != nil {
		t.Fatalf("Failed to create users table: %v", err)
	}

	// Create departments table
	_, err = engine.Execute("CREATE TABLE departments (id INT PRIMARY KEY, name VARCHAR(50))")
	if err != nil {
		t.Fatalf("Failed to create departments table: %v", err)
	}

	// Insert test data
	userQueries := []string{
		"INSERT INTO users VALUES (1, 'Alice', 1)",
		"INSERT INTO users VALUES (2, 'Bob', 2)",
		"INSERT INTO users VALUES (3, 'Charlie', 1)",
	}

	for _, query := range userQueries {
		_, err = engine.Execute(query)
		if err != nil {
			t.Fatalf("Failed to insert user data: %v", err)
		}
	}

	deptQueries := []string{
		"INSERT INTO departments VALUES (1, 'Engineering')",
		"INSERT INTO departments VALUES (2, 'Marketing')",
	}

	for _, query := range deptQueries {
		_, err = engine.Execute(query)
		if err != nil {
			t.Fatalf("Failed to insert department data: %v", err)
		}
	}

	// Test INNER JOIN
	result, err := engine.Execute("SELECT users.name, departments.name FROM users JOIN departments ON users.department_id = departments.id")
	if err != nil {
		t.Fatalf("Failed to execute JOIN: %v", err)
	}

	selectResult := result.(*SelectResult)
	if len(selectResult.Rows) != 3 {
		t.Errorf("Expected 3 joined rows, got %d", len(selectResult.Rows))
	}

	if len(selectResult.Columns) != 2 {
		t.Errorf("Expected 2 columns, got %d", len(selectResult.Columns))
	}

	// Verify specific join results
	expectedResults := map[string]string{
		"Alice":   "Engineering",
		"Bob":     "Marketing",
		"Charlie": "Engineering",
	}

	for _, row := range selectResult.Rows {
		userName := row[0].(string)
		deptName := row[1].(string)

		if expectedDept, exists := expectedResults[userName]; !exists || expectedDept != deptName {
			t.Errorf("Unexpected join result: %s -> %s", userName, deptName)
		}
	}
}

func TestAggregateFunctions(t *testing.T) {
	engine := NewSQLEngine()

	// Create and populate table
	_, err := engine.Execute("CREATE TABLE sales (id INT, product VARCHAR(50), amount FLOAT, quantity INT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	queries := []string{
		"INSERT INTO sales VALUES (1, 'Laptop', 999.99, 2)",
		"INSERT INTO sales VALUES (2, 'Mouse', 29.99, 5)",
		"INSERT INTO sales VALUES (3, 'Keyboard', 79.99, 3)",
		"INSERT INTO sales VALUES (4, 'Monitor', 299.99, 1)",
	}

	for _, query := range queries {
		_, err = engine.Execute(query)
		if err != nil {
			t.Fatalf("Failed to insert data: %v", err)
		}
	}

	// Test COUNT(*)
	result, err := engine.Execute("SELECT COUNT(*) FROM sales")
	if err != nil {
		t.Fatalf("Failed to execute COUNT(*): %v", err)
	}

	selectResult := result.(*SelectResult)
	if len(selectResult.Rows) != 1 || selectResult.Rows[0][0] != int64(4) {
		t.Errorf("COUNT(*) failed: expected 4, got %v", selectResult.Rows[0][0])
	}

	// Test SUM
	result, err = engine.Execute("SELECT SUM(amount) FROM sales")
	if err != nil {
		t.Fatalf("Failed to execute SUM: %v", err)
	}

	selectResult = result.(*SelectResult)
	if len(selectResult.Rows) != 1 {
		t.Errorf("SUM failed: expected 1 row, got %d", len(selectResult.Rows))
	}

	sumValue := selectResult.Rows[0][0].(float64)
	expectedSum := 999.99 + 29.99 + 79.99 + 299.99
	if sumValue != expectedSum {
		t.Errorf("SUM failed: expected %f, got %f", expectedSum, sumValue)
	}

	// Test AVG
	result, err = engine.Execute("SELECT AVG(quantity) FROM sales")
	if err != nil {
		t.Fatalf("Failed to execute AVG: %v", err)
	}

	selectResult = result.(*SelectResult)
	avgValue := selectResult.Rows[0][0].(float64)
	expectedAvg := (2.0 + 5.0 + 3.0 + 1.0) / 4.0
	if avgValue != expectedAvg {
		t.Errorf("AVG failed: expected %f, got %f", expectedAvg, avgValue)
	}

	// Test MIN and MAX
	result, err = engine.Execute("SELECT MIN(amount), MAX(amount) FROM sales")
	if err != nil {
		t.Fatalf("Failed to execute MIN/MAX: %v", err)
	}

	selectResult = result.(*SelectResult)
	minValue := selectResult.Rows[0][0].(float64)
	maxValue := selectResult.Rows[0][1].(float64)

	if minValue != 29.99 {
		t.Errorf("MIN failed: expected 29.99, got %f", minValue)
	}
	if maxValue != 999.99 {
		t.Errorf("MAX failed: expected 999.99, got %f", maxValue)
	}
}

func TestIndexFunctionality(t *testing.T) {
	engine := NewSQLEngine()

	// Create table
	_, err := engine.Execute("CREATE TABLE indexed_table (id INT, name VARCHAR(50), score INT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Create index
	result, err := engine.Execute("CREATE INDEX idx_score ON indexed_table (score)")
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	if result != "Index created successfully" {
		t.Errorf("Unexpected result: %v", result)
	}

	// Insert test data
	queries := []string{
		"INSERT INTO indexed_table VALUES (1, 'Alice', 95)",
		"INSERT INTO indexed_table VALUES (2, 'Bob', 87)",
		"INSERT INTO indexed_table VALUES (3, 'Charlie', 95)",
		"INSERT INTO indexed_table VALUES (4, 'Diana', 92)",
	}

	for _, query := range queries {
		_, err = engine.Execute(query)
		if err != nil {
			t.Fatalf("Failed to insert data: %v", err)
		}
	}

	// Test index-optimized query
	result, err = engine.Execute("SELECT name FROM indexed_table WHERE score = 95")
	if err != nil {
		t.Fatalf("Failed to execute indexed query: %v", err)
	}

	selectResult := result.(*SelectResult)
	if len(selectResult.Rows) != 2 {
		t.Errorf("Expected 2 rows with score 95, got %d", len(selectResult.Rows))
	}

	// Test SHOW INDEX
	result, err = engine.Execute("SHOW INDEX FROM indexed_table")
	if err != nil {
		t.Fatalf("Failed to show indexes: %v", err)
	}

	selectResult = result.(*SelectResult)
	if len(selectResult.Rows) != 1 {
		t.Errorf("Expected 1 index, got %d", len(selectResult.Rows))
	}

	if selectResult.Rows[0][0] != "idx_score" {
		t.Errorf("Expected index name 'idx_score', got %v", selectResult.Rows[0][0])
	}

	// Test DROP INDEX
	result, err = engine.Execute("DROP INDEX idx_score")
	if err != nil {
		t.Fatalf("Failed to drop index: %v", err)
	}

	if result != "Index dropped successfully" {
		t.Errorf("Unexpected result: %v", result)
	}
}

func TestAlterTable(t *testing.T) {
	engine := NewSQLEngine()

	// Create initial table
	_, err := engine.Execute("CREATE TABLE alter_test (id INT PRIMARY KEY, name VARCHAR(50))")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert initial data
	_, err = engine.Execute("INSERT INTO alter_test VALUES (1, 'Alice')")
	if err != nil {
		t.Fatalf("Failed to insert initial data: %v", err)
	}

	// Test ADD COLUMN
	result, err := engine.Execute("ALTER TABLE alter_test ADD COLUMN age INT")
	if err != nil {
		t.Fatalf("Failed to add column: %v", err)
	}

	if result != "Table alter_test altered successfully" {
		t.Errorf("Unexpected result: %v", result)
	}

	// Verify new column exists and has default value
	selectResult, err := engine.Execute("SELECT * FROM alter_test")
	if err != nil {
		t.Fatalf("Failed to select after ADD COLUMN: %v", err)
	}

	sr := selectResult.(*SelectResult)
	if len(sr.Columns) != 3 {
		t.Errorf("Expected 3 columns after ADD, got %d", len(sr.Columns))
	}

	// The new column should have a default value (nil for nullable columns)
	if sr.Rows[0][2] != nil {
		t.Errorf("Expected default value nil for new nullable column, got %v", sr.Rows[0][2])
	}

	// Test MODIFY COLUMN
	_, err = engine.Execute("ALTER TABLE alter_test MODIFY COLUMN age VARCHAR(10)")
	if err != nil {
		t.Fatalf("Failed to modify column: %v", err)
	}

	// Test DROP COLUMN
	_, err = engine.Execute("ALTER TABLE alter_test DROP COLUMN age")
	if err != nil {
		t.Fatalf("Failed to drop column: %v", err)
	}

	// Verify column was dropped
	selectResult, err = engine.Execute("SELECT * FROM alter_test")
	if err != nil {
		t.Fatalf("Failed to select after DROP COLUMN: %v", err)
	}

	sr = selectResult.(*SelectResult)
	if len(sr.Columns) != 2 {
		t.Errorf("Expected 2 columns after DROP, got %d", len(sr.Columns))
	}

	// Test CHANGE COLUMN (rename)
	_, err = engine.Execute("ALTER TABLE alter_test CHANGE COLUMN name full_name VARCHAR(100)")
	if err != nil {
		t.Fatalf("Failed to change column: %v", err)
	}

	// Verify column was renamed
	selectResult, err = engine.Execute("SELECT full_name FROM alter_test")
	if err != nil {
		t.Fatalf("Failed to select renamed column: %v", err)
	}

	sr = selectResult.(*SelectResult)
	if sr.Columns[0] != "full_name" {
		t.Errorf("Expected column name 'full_name', got %s", sr.Columns[0])
	}
}

func TestLimitClause(t *testing.T) {
	engine := NewSQLEngine()

	// Create and populate table
	_, err := engine.Execute("CREATE TABLE limit_test (id INT, value VARCHAR(10))")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert test data
	for i := 1; i <= 10; i++ {
		query := fmt.Sprintf("INSERT INTO limit_test VALUES (%d, 'value%d')", i, i)
		_, err = engine.Execute(query)
		if err != nil {
			t.Fatalf("Failed to insert data: %v", err)
		}
	}

	// Test LIMIT count
	result, err := engine.Execute("SELECT * FROM limit_test LIMIT 3")
	if err != nil {
		t.Fatalf("Failed to execute LIMIT query: %v", err)
	}

	selectResult := result.(*SelectResult)
	if len(selectResult.Rows) != 3 {
		t.Errorf("Expected 3 rows with LIMIT 3, got %d", len(selectResult.Rows))
	}

	// Test LIMIT offset, count
	result, err = engine.Execute("SELECT * FROM limit_test LIMIT 2, 3")
	if err != nil {
		t.Fatalf("Failed to execute LIMIT offset,count query: %v", err)
	}

	selectResult = result.(*SelectResult)
	if len(selectResult.Rows) != 3 {
		t.Errorf("Expected 3 rows with LIMIT 2,3, got %d", len(selectResult.Rows))
	}

	// Verify offset works correctly (should start from id=3)
	if selectResult.Rows[0][0] != int64(3) {
		t.Errorf("Expected first row id=3 with LIMIT 2,3, got %v", selectResult.Rows[0][0])
	}

	// Test LIMIT with WHERE clause
	result, err = engine.Execute("SELECT * FROM limit_test WHERE id > 5 LIMIT 2")
	if err != nil {
		t.Fatalf("Failed to execute LIMIT with WHERE: %v", err)
	}

	selectResult = result.(*SelectResult)
	if len(selectResult.Rows) != 2 {
		t.Errorf("Expected 2 rows with WHERE and LIMIT, got %d", len(selectResult.Rows))
	}

	// Test LIMIT with aggregate functions
	result, err = engine.Execute("SELECT COUNT(*) FROM limit_test LIMIT 1")
	if err != nil {
		t.Fatalf("Failed to execute LIMIT with aggregate: %v", err)
	}

	selectResult = result.(*SelectResult)
	if len(selectResult.Rows) != 1 {
		t.Errorf("Expected 1 row with aggregate LIMIT, got %d", len(selectResult.Rows))
	}
}

func TestSubqueries(t *testing.T) {
	engine := NewSQLEngine()

	// Create and populate table
	_, err := engine.Execute("CREATE TABLE subquery_test (id INT, category VARCHAR(20), value INT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	queries := []string{
		"INSERT INTO subquery_test VALUES (1, 'A', 100)",
		"INSERT INTO subquery_test VALUES (2, 'B', 200)",
		"INSERT INTO subquery_test VALUES (3, 'A', 150)",
		"INSERT INTO subquery_test VALUES (4, 'B', 250)",
	}

	for _, query := range queries {
		_, err = engine.Execute(query)
		if err != nil {
			t.Fatalf("Failed to insert data: %v", err)
		}
	}

	// Test simple subquery
	result, err := engine.Execute("SELECT * FROM (SELECT id, value FROM subquery_test WHERE category = 'A') AS sub")
	if err != nil {
		t.Fatalf("Failed to execute subquery: %v", err)
	}

	selectResult := result.(*SelectResult)
	if len(selectResult.Rows) != 2 {
		t.Errorf("Expected 2 rows from subquery, got %d", len(selectResult.Rows))
	}

	if len(selectResult.Columns) != 2 {
		t.Errorf("Expected 2 columns from subquery, got %d", len(selectResult.Columns))
	}

	// Test subquery with LIMIT
	result, err = engine.Execute("SELECT * FROM (SELECT * FROM subquery_test ORDER BY value) AS sub LIMIT 2")
	if err != nil {
		t.Fatalf("Failed to execute subquery with LIMIT: %v", err)
	}

	selectResult = result.(*SelectResult)
	if len(selectResult.Rows) != 2 {
		t.Errorf("Expected 2 rows from subquery with LIMIT, got %d", len(selectResult.Rows))
	}
}
