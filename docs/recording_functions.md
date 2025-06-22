# Query Recording Functions

The Mist SQL Engine now includes three new functions for recording and retrieving SQL queries:

## Functions

### 1. `StartRecording()`
- **Purpose**: Begins recording all SQL queries executed by the engine
- **Thread-safety**: Yes, uses mutex protection
- **Behavior**: Clears any previously recorded queries when starting a new recording session
- **Usage**: Call this before executing queries you want to record

```go
engine := mist.NewSQLEngine()
engine.StartRecording()
```

### 2. `EndRecording()`
- **Purpose**: Stops recording SQL queries
- **Thread-safety**: Yes, uses mutex protection
- **Behavior**: Recording state is set to false, but recorded queries are preserved
- **Usage**: Call this when you want to stop recording queries

```go
engine.EndRecording()
```

### 3. `GetRecordedQueries()`
- **Purpose**: Returns all queries that were recorded between StartRecording and EndRecording
- **Return type**: `[]string` - a slice of SQL query strings
- **Thread-safety**: Yes, returns a copy of the recorded queries to prevent external modification
- **Usage**: Call this to retrieve all recorded queries

```go
recordedQueries := engine.GetRecordedQueries()
for i, query := range recordedQueries {
    fmt.Printf("%d. %s\n", i+1, query)
}
```

## Example Usage

```go
package main

import (
    "fmt"
    "github.com/abbychau/mist/mist"
)

func main() {
    engine := mist.NewSQLEngine()
    
    // Set up a table
    engine.Execute("CREATE TABLE users (id INT, name VARCHAR(50))")
    
    // Start recording
    engine.StartRecording()
    
    // Execute queries while recording
    engine.Execute("INSERT INTO users VALUES (1, 'Alice')")
    engine.Execute("INSERT INTO users VALUES (2, 'Bob')")
    engine.Execute("SELECT * FROM users")
    
    // Stop recording
    engine.EndRecording()
    
    // Get recorded queries
    queries := engine.GetRecordedQueries()
    fmt.Printf("Recorded %d queries:\n", len(queries))
    for i, query := range queries {
        fmt.Printf("%d. %s\n", i+1, query)
    }
}
```

## Key Features

- **Thread-safe**: All functions use proper mutex locking for concurrent access
- **Clean recording sessions**: Starting a new recording clears previous recordings
- **Immutable results**: GetRecordedQueries returns a copy to prevent external modification
- **No performance impact when not recording**: Recording only occurs when explicitly enabled
- **Original query preservation**: Queries are recorded exactly as passed to Execute()

## Use Cases

- **Debugging**: Track which queries are being executed in your application
- **Auditing**: Log all database operations for compliance or analysis
- **Testing**: Verify that your code executes the expected SQL queries
- **Performance analysis**: Analyze query patterns and frequency
- **Migration tools**: Capture queries for replay on different environments
