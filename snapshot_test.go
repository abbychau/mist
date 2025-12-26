package mist

import (
	"os"
	"testing"
)

func TestSnapshotting(t *testing.T) {
	aofPath := "test_snapshot.aof"
	snapshotPath := "test_snapshot.bin"
	defer os.Remove(aofPath)
	defer os.Remove(snapshotPath)

	options := PersistenceOptions{
		Enabled:      true,
		AofPath:      aofPath,
		SnapshotPath: snapshotPath,
	}
	engine := NewSQLEngineWithOptions(options)
	defer engine.Close()

	// 1. Setup some data
	_, err := engine.Execute("CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50))")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	_, err = engine.Execute("INSERT INTO users VALUES (1, 'Alice')")
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}
	_, err = engine.Execute("CREATE INDEX idx_name ON users (name)")
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	// 2. Save snapshot
	err = engine.SaveSnapshot()
	if err != nil {
		t.Fatalf("Failed to save snapshot: %v", err)
	}

	// 3. Verify AOF is truncated
	aofInfo, err := os.Stat(aofPath)
	if err != nil {
		t.Fatalf("Failed to stat AOF: %v", err)
	}
	if aofInfo.Size() != 0 {
		t.Errorf("Expected AOF to be truncated after snapshot, but size is %d", aofInfo.Size())
	}

	// 4. Close and reload from snapshot
	engine.Close()
	engine2 := NewSQLEngineWithOptions(options)
	defer engine2.Close()

	// 5. Verify data and index recovery
	result, err := engine2.Execute("SELECT * FROM users WHERE id = 1")
	if err != nil {
		t.Fatalf("Failed to query after snapshot recovery: %v", err)
	}
	selectResult := result.(*SelectResult)
	if len(selectResult.Rows) != 1 || selectResult.Rows[0][1] != "Alice" {
		t.Errorf("Data mismatch after snapshot recovery: %v", selectResult.Rows)
	}

	// Check index
	result, err = engine2.Execute("SHOW INDEX FROM users")
	if err != nil {
		t.Fatalf("Failed to show index: %v", err)
	}
	selectResult = result.(*SelectResult)
	foundIdx := false
	for _, row := range selectResult.Rows {
		if row[0] == "idx_name" {
			foundIdx = true
			break
		}
	}
	if !foundIdx {
		t.Errorf("Index 'idx_name' not found after recovery")
	}

	// 6. Test AOF replay ON TOP of snapshot
	_, err = engine2.Execute("INSERT INTO users VALUES (2, 'Bob')")
	if err != nil {
		t.Fatalf("Failed to insert after recovery: %v", err)
	}

	engine2.Close()
	engine3 := NewSQLEngineWithOptions(options)
	defer engine3.Close()

	result, err = engine3.Execute("SELECT COUNT(*) FROM users")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	count := result.(*SelectResult).Rows[0][0].(int64)
	if count != 2 {
		t.Errorf("Expected 2 users after snapshot + AOF recovery, got %d", count)
	}
}
