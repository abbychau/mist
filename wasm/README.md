# Mist WebAssembly Implementation

This directory contains the WebAssembly version of the Mist SQL engine for running in web browsers.

## Files

- `wasm_engine.go` - WASM wrapper around the main Mist engine with JavaScript bindings
- `go.mod` - Go module configuration with main Mist dependency

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

- `executeSQL(query)` - Execute SQL query
- `startRecording()` - Start query recording  
- `stopRecording()` - Stop query recording
- `getRecordedQueries()` - Get recorded queries
- `clearRecordedQueries()` - Clear recorded queries

## Features

Since this WASM engine now uses the main Mist engine, it supports **ALL** features available in the native version:

- **Complete SQL Support** - All DDL and DML operations
- **Transaction Support** - BEGIN, COMMIT, ROLLBACK, savepoints
- **Advanced Features** - JOINs, subqueries, aggregate functions, indexes
- **Data Types** - All supported types including DECIMAL, TIMESTAMP, ENUM
- **Query Recording** - Full recording and playback capabilities
- **Thread-safe Operations** - Concurrent query execution
- **MySQL Compatibility** - Uses the same mysql-parser as native version

## Architecture

This WASM engine is now a **thin wrapper** around the main Mist SQL engine. It:

1. Imports the main `github.com/abbychau/mist` package
2. Creates a `mist.SQLEngine` instance
3. Provides WASM-compatible result formatting
4. Exports JavaScript functions for web usage

### Benefits of Shared Engine

- **Feature Parity** - WASM automatically gets all new features added to main engine
- **Consistency** - Identical behavior between web playground and native Mist
- **Reduced Maintenance** - No duplicate code to maintain
- **Single Source of Truth** - All SQL logic centralized in main engine

The WASM binary size remains small (~2.5MB) while providing the full power of the Mist SQL engine in the browser.