# Auto Increment Example

This example demonstrates how to use AUTO_INCREMENT columns in Mist database engine.

## Features Demonstrated

- Creating tables with AUTO_INCREMENT PRIMARY KEY
- Inserting records without specifying ID values
- Automatic ID generation and incrementing
- Mixed insert approaches (explicit ID + auto increment)
- Querying tables with auto-generated IDs

## Running the Example

```bash
cd examples/auto_increment
go run main.go
```

## Key Points

1. **AUTO_INCREMENT must be used with PRIMARY KEY**: The column must be both AUTO_INCREMENT and PRIMARY KEY
2. **Automatic ID assignment**: When inserting without specifying the ID, Mist automatically assigns the next available ID
3. **Manual ID insertion**: You can still insert with explicit IDs, and auto increment will continue from the highest ID
4. **Thread-safe**: Auto increment operations are thread-safe and handle concurrent inserts properly

## Example Output

The example will show:
- Table creation with AUTO_INCREMENT
- Multiple insert operations with automatic ID generation
- Complete product catalog with sequential IDs
- Basic statistics and queries on the auto-generated data
