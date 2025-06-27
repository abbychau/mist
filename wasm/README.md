# Mist WebAssembly Implementation

This directory contains the WebAssembly version of the Mist SQL engine for running in web browsers.

## Files

- `main.go` - JavaScript bindings and WASM entry point
- `wasm_engine.go` - Standalone SQL engine optimized for WASM (no external dependencies)
- `go.mod` - Minimal Go module configuration

## Building

From the project root:
```bash
./build-wasm.sh
```

Or manually:
```bash
cd wasm && GOOS=js GOARCH=wasm go build -o ../docs/mist.wasm .
```

## Usage

The compiled WASM binary is used by the web playground at `docs/playground.html`. It provides these JavaScript functions:

- `mistExecute(query)` - Execute SQL query
- `mistStartRecording()` - Start query recording  
- `mistEndRecording()` - Stop query recording
- `mistGetRecordedQueries()` - Get recorded queries
- `mistShowTables()` - List tables

## Features

- **Proper SQL INSERT VALUES parsing** - Correctly handles real data values
- **Multiple row insertion support** - Single INSERT with multiple value tuples
- **Type conversion** - Seamless Go ↔ JavaScript data conversion
- **Thread-safe operations** - Concurrent query execution
- **Zero external dependencies** - Pure Go standard library only
- **Small binary size** - ~2.5MB WASM output
- **Browser-compatible** - No system calls or file I/O

## Architecture

This WASM engine is a **standalone implementation** that shares the same SQL parsing logic as the main Mist engine but avoids heavy dependencies like TiDB parser that don't compile to WASM. The engine provides:

- Basic SQL DDL (CREATE TABLE, DROP TABLE)  
- Full DML support (INSERT, SELECT, UPDATE, DELETE)
- Aggregate functions (COUNT, etc.)
- SHOW TABLES functionality
- Query recording capabilities