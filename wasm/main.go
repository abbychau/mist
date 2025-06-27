package main

import (
	"fmt"
	"syscall/js"
)

var engine *SQLEngine

func main() {
	// Add console logging for debugging
	js.Global().Get("console").Call("log", "Mist WASM: Starting initialization...")
	
	// Initialize the WASM SQL engine
	engine = NewSQLEngine()
	js.Global().Get("console").Call("log", "Mist WASM: Engine created")
	
	// Register JavaScript functions
	js.Global().Set("mistExecute", js.FuncOf(mistExecute))
	js.Global().Get("console").Call("log", "Mist WASM: mistExecute registered")
	
	js.Global().Set("mistStartRecording", js.FuncOf(mistStartRecording))
	js.Global().Set("mistEndRecording", js.FuncOf(mistEndRecording))
	js.Global().Set("mistGetRecordedQueries", js.FuncOf(mistGetRecordedQueries))
	js.Global().Set("mistShowTables", js.FuncOf(mistShowTables))
	js.Global().Get("console").Call("log", "Mist WASM: All functions registered")
	
	// Signal that Mist is ready
	js.Global().Set("mistReady", true)
	js.Global().Get("console").Call("log", "Mist WASM: Ready flag set, initialization complete")
	
	// Keep the Go runtime alive
	select {}
}

// mistExecute executes a SQL query and returns the result as JSON
func mistExecute(this js.Value, args []js.Value) interface{} {
	// Use defer to catch any panics
	defer func() {
		if r := recover(); r != nil {
			js.Global().Get("console").Call("log", "Mist WASM: PANIC in mistExecute:", r)
		}
	}()
	
	js.Global().Get("console").Call("log", "Mist WASM: mistExecute called with", len(args), "arguments")
	
	if len(args) != 1 {
		errorResult := map[string]interface{}{
			"error": "Expected 1 argument (SQL query string)",
		}
		js.Global().Get("console").Call("log", "Mist WASM: Returning argument error")
		return errorResult
	}
	
	query := args[0].String()
	js.Global().Get("console").Call("log", "Mist WASM: Executing query:", query)
	
	// Execute the query
	result, err := engine.Execute(query)
	if err != nil {
		js.Global().Get("console").Call("log", "Mist WASM: Query executed with error:", err.Error())
	} else {
		js.Global().Get("console").Call("log", "Mist WASM: Query executed successfully")
	}
	// Don't log the result directly as it might cause ValueOf issues
	
	if err != nil {
		errorResult := map[string]interface{}{
			"error": err.Error(),
		}
		js.Global().Get("console").Call("log", "Mist WASM: Returning error result")
		return errorResult
	}
	
	// Handle nil result
	if result == nil {
		nilResult := map[string]interface{}{
			"type":    "message", 
			"message": "Query completed successfully (no result)",
		}
		js.Global().Get("console").Call("log", "Mist WASM: Returning nil result")
		return nilResult
	}
	
	// Convert result to JSON-serializable format
	js.Global().Get("console").Call("log", "Mist WASM: Converting result of type:", fmt.Sprintf("%T", result))
	
	switch r := result.(type) {
	case *SelectResult:
		js.Global().Get("console").Call("log", "Mist WASM: Processing SelectResult - columns:", len(r.Columns), "rows:", len(r.Rows))
		
		// Build the result step by step to avoid ValueOf issues
		js.Global().Get("console").Call("log", "Mist WASM: Building result object...")
		
		// Create result using JavaScript objects directly
		result := js.Global().Get("Object").New()
		result.Set("type", "select")
		
		// Create columns array
		columnsArray := js.Global().Get("Array").New()
		for i, col := range r.Columns {
			columnsArray.SetIndex(i, fmt.Sprintf("%v", col))
		}
		result.Set("columns", columnsArray)
		
		// Create rows array
		rowsArray := js.Global().Get("Array").New()
		for i, row := range r.Rows {
			rowArray := js.Global().Get("Array").New()
			for j, cell := range row {
				if cell == nil {
					rowArray.SetIndex(j, js.Null())
				} else {
					rowArray.SetIndex(j, fmt.Sprintf("%v", cell))
				}
			}
			rowsArray.SetIndex(i, rowArray)
		}
		result.Set("rows", rowsArray)
		js.Global().Get("console").Call("log", "Mist WASM: Returning SelectResult with", len(r.Columns), "columns and", len(r.Rows), "rows")
		return result
		
	case string:
		js.Global().Get("console").Call("log", "Mist WASM: Processing string result:", r)
		stringResult := map[string]interface{}{
			"type":    "message",
			"message": r,
		}
		js.Global().Get("console").Call("log", "Mist WASM: Returning string result")
		return stringResult
		
	default:
		js.Global().Get("console").Call("log", "Mist WASM: Processing default case")
		defaultResult := map[string]interface{}{
			"type":    "message",
			"message": fmt.Sprintf("%v", result),
		}
		js.Global().Get("console").Call("log", "Mist WASM: Returning default result")
		return defaultResult
	}
}

// mistStartRecording starts query recording
func mistStartRecording(this js.Value, args []js.Value) interface{} {
	engine.StartRecording()
	return map[string]interface{}{
		"message": "Recording started",
	}
}

// mistEndRecording stops query recording
func mistEndRecording(this js.Value, args []js.Value) interface{} {
	engine.EndRecording()
	return map[string]interface{}{
		"message": "Recording stopped",
	}
}

// mistGetRecordedQueries returns recorded queries as JSON
func mistGetRecordedQueries(this js.Value, args []js.Value) interface{} {
	queries := engine.GetRecordedQueries()
	return map[string]interface{}{
		"queries": queries,
	}
}

// mistShowTables returns list of tables
func mistShowTables(this js.Value, args []js.Value) interface{} {
	result, err := engine.Execute("SHOW TABLES")
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}
	}
	
	if selectResult, ok := result.(*SelectResult); ok {
		var tables []string
		for _, row := range selectResult.Rows {
			if len(row) > 0 {
				if tableName, ok := row[0].(string); ok {
					tables = append(tables, tableName)
				}
			}
		}
		return map[string]interface{}{
			"tables": tables,
		}
	}
	
	return map[string]interface{}{
		"tables": []string{},
	}
}