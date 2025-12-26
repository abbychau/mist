package mist

import (
	"os"
	"testing"
)

func TestAofPersistence(t *testing.T) {
	aofPath := "test_persistence.aof"
	defer os.Remove(aofPath)

	// Step 1: Create engine with persistence
	options := PersistenceOptions{
		Enabled: true,
		AofPath: aofPath,
	}
	engine := NewSQLEngineWithOptions(options)
	defer engine.Close()

	// Step 2: Execute some mutations
	_, err := engine.Execute("CREATE TABLE users (id INT, name VARCHAR(50))")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	_, err = engine.Execute("INSERT INTO users VALUES (1, 'Alice')")
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	_, err = engine.Execute("INSERT INTO users VALUES (2, 'Bob')")
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Step 3: Verify AOF file exists and has content
	content, err := os.ReadFile(aofPath)
	if err != nil {
		t.Fatalf("Failed to read AOF file: %v", err)
	}

	expectedContent := "CREATE TABLE users (id INT, name VARCHAR(50));\nINSERT INTO users VALUES (1, 'Alice');\nINSERT INTO users VALUES (2, 'Bob');\n"
	if string(content) != expectedContent {
		t.Errorf("Unexpected AOF content.\nExpected: %q\nGot:      %q", expectedContent, string(content))
	}

	// Step 4: Close engine and create a new one to test recovery
	engine.Close()

	engine2 := NewSQLEngineWithOptions(options)
	defer engine2.Close()

	// Step 5: Verify data is recovered
	result, err := engine2.Execute("SELECT * FROM users")
	if err != nil {
		t.Fatalf("Failed to select from recovered database: %v", err)
	}

	selectResult, ok := result.(*SelectResult)
	if !ok {
		t.Fatalf("Expected SelectResult, got %T", result)
	}

	if len(selectResult.Rows) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(selectResult.Rows))
	}

	if selectResult.Rows[0][1] != "Alice" || selectResult.Rows[1][1] != "Bob" {
		t.Errorf("Data mismatch after recovery: %v", selectResult.Rows)
	}
}

func TestAofPersistenceWithTransactions(t *testing.T) {
	aofPath := "test_txn_persistence.aof"
	defer os.Remove(aofPath)

	options := PersistenceOptions{
		Enabled: true,
		AofPath: aofPath,
	}
	engine := NewSQLEngineWithOptions(options)
	defer engine.Close()

	// Mutations with transaction
	_, err := engine.Execute("CREATE TABLE items (id INT, name VARCHAR(50))")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	_, err = engine.Execute("BEGIN")
	if err != nil {
		t.Fatalf("Failed to BEGIN: %v", err)
	}

	_, err = engine.Execute("INSERT INTO items VALUES (1, 'Item 1')")
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	_, err = engine.Execute("COMMIT")
	if err != nil {
		t.Fatalf("Failed to COMMIT: %v", err)
	}

	engine.Close()

	// Recover
	engine2 := NewSQLEngineWithOptions(options)
	defer engine2.Close()

	result, err := engine2.Execute("SELECT COUNT(*) FROM items")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	selectResult := result.(*SelectResult)
	count := selectResult.Rows[0][0].(int64)
	if count != 1 {
		t.Errorf("Expected 1 item after txn recovery, got %d", count)
	}
}
