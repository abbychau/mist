// +build js,wasm

package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"syscall/js"

	"github.com/abbychau/mist"
)

// WASMSQLEngine wraps the main MIST SQLEngine for WASM usage
type WASMSQLEngine struct {
	engine *mist.SQLEngine
	mutex  sync.Mutex
}

// NewWASMSQLEngine creates a new WASM SQL engine
func NewWASMSQLEngine() *WASMSQLEngine {
	return &WASMSQLEngine{
		engine: mist.NewSQLEngine(),
	}
}

// Execute runs SQL queries (supports multiple statements separated by semicolons) and returns results as JSON string
func (w *WASMSQLEngine) Execute(query string) (string, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	// Split multiple SQL statements by semicolon
	statements := w.splitSQLStatements(query)
	if len(statements) == 0 {
		errorResult := map[string]interface{}{
			"error": "No SQL statements found",
		}
		jsonBytes, _ := json.Marshal(errorResult)
		return string(jsonBytes), nil
	}

	// Execute multiple statements - return result of the last one that returns data
	var lastResult interface{}
	var resultMessages []string

	for i, stmt := range statements {
		if strings.TrimSpace(stmt) == "" {
			continue
		}

		result, err := w.engine.Execute(stmt)
		if err != nil {
			errorResult := map[string]interface{}{
				"error": fmt.Sprintf("Statement %d error: %s", i+1, err.Error()),
			}
			jsonBytes, _ := json.Marshal(errorResult)
			return string(jsonBytes), nil
		}

		// Collect results
		switch r := result.(type) {
		case *mist.SelectResult:
			// For SELECT results, use this as the final result
			lastResult = result
		case string:
			// For messages, collect them
			resultMessages = append(resultMessages, r)
		default:
			// For other types, collect as messages
			resultMessages = append(resultMessages, fmt.Sprintf("%v", result))
		}
	}

	// Determine what to return
	var finalResult interface{}
	if lastResult != nil {
		// If we have a SELECT result, return that
		finalResult = w.formatResultForWASM(lastResult)
	} else if len(resultMessages) > 0 {
		// If we only have messages, return them
		finalResult = map[string]interface{}{
			"type":     "message",
			"message":  strings.Join(resultMessages, "; "),
			"messages": resultMessages,
		}
	} else {
		finalResult = map[string]interface{}{
			"type":    "message",
			"message": "Statements executed successfully",
		}
	}

	// Convert result to JSON string
	jsonBytes, err := json.Marshal(finalResult)
	if err != nil {
		errorResult := map[string]interface{}{
			"error": "Failed to serialize result: " + err.Error(),
		}
		jsonBytes, _ := json.Marshal(errorResult)
		return string(jsonBytes), nil
	}

	return string(jsonBytes), nil
}

// splitSQLStatements splits a query string into individual SQL statements
func (w *WASMSQLEngine) splitSQLStatements(query string) []string {
	// Simple split on semicolon - could be improved to handle quoted strings
	parts := strings.Split(query, ";")
	var statements []string
	
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			statements = append(statements, trimmed)
		}
	}
	
	return statements
}

// formatResultForWASM converts MIST results to WASM-compatible format
func (w *WASMSQLEngine) formatResultForWASM(result interface{}) interface{} {
	switch r := result.(type) {
	case *mist.SelectResult:
		// Convert all row values to JavaScript-compatible types
		jsRows := make([][]interface{}, len(r.Rows))
		for i, row := range r.Rows {
			jsRow := make([]interface{}, len(row))
			for j, val := range row {
				jsRow[j] = convertToJSValue(val)
			}
			jsRows[i] = jsRow
		}
		
		return map[string]interface{}{
			"type":    "select",
			"columns": r.Columns,
			"rows":    jsRows,
		}
	case string:
		return map[string]interface{}{
			"type":    "message",
			"message": r,
		}
	case int:
		return map[string]interface{}{
			"type":         "affected_rows",
			"affectedRows": r,
		}
	default:
		return map[string]interface{}{
			"type":    "unknown",
			"result":  fmt.Sprintf("%v", result),
		}
	}
}

// convertToJSValue converts Go values to JavaScript-compatible values
func convertToJSValue(val interface{}) interface{} {
	if val == nil {
		return nil
	}
	
	switch v := val.(type) {
	case string, bool, int, int8, int16, int32, int64:
		return v
	case uint:
		return int64(v)
	case uint8:
		return int64(v)
	case uint16:
		return int64(v)
	case uint32:
		return int64(v)
	case uint64:
		return int64(v)
	case float32, float64:
		return v
	case []byte:
		return string(v)
	default:
		// Convert everything else to string for safety
		return fmt.Sprintf("%v", v)
	}
}

// StartRecording starts query recording
func (w *WASMSQLEngine) StartRecording() {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.engine.StartRecording()
}

// StopRecording stops query recording
func (w *WASMSQLEngine) StopRecording() {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.engine.EndRecording()
}

// GetRecordedQueries returns recorded queries
func (w *WASMSQLEngine) GetRecordedQueries() []string {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return w.engine.GetRecordedQueries()
}

// ClearRecordedQueries clears recorded queries  
func (w *WASMSQLEngine) ClearRecordedQueries() {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	// Note: Main MIST engine doesn't have ClearRecordedQueries method
	// Queries are cleared when StartRecording is called again
}

// Global engine instance
var globalEngine *WASMSQLEngine

// WASM exported functions
func executeSQL(this js.Value, p []js.Value) interface{} {
	if len(p) < 1 {
		errorResult := map[string]interface{}{
			"error": "Missing SQL query parameter",
		}
		jsonBytes, _ := json.Marshal(errorResult)
		return string(jsonBytes)
	}

	query := p[0].String()
	jsonResult, _ := globalEngine.Execute(query)
	
	return jsonResult
}

func startRecording(this js.Value, p []js.Value) interface{} {
	globalEngine.StartRecording()
	result := map[string]interface{}{
		"message": "Recording started",
	}
	jsonBytes, _ := json.Marshal(result)
	return string(jsonBytes)
}

func stopRecording(this js.Value, p []js.Value) interface{} {
	globalEngine.StopRecording()
	result := map[string]interface{}{
		"message": "Recording stopped",
	}
	jsonBytes, _ := json.Marshal(result)
	return string(jsonBytes)
}

func getRecordedQueries(this js.Value, p []js.Value) interface{} {
	queries := globalEngine.GetRecordedQueries()
	result := map[string]interface{}{
		"queries": queries,
	}
	jsonBytes, _ := json.Marshal(result)
	return string(jsonBytes)
}

func clearRecordedQueries(this js.Value, p []js.Value) interface{} {
	globalEngine.ClearRecordedQueries()
	result := map[string]interface{}{
		"message": "Recorded queries cleared",
	}
	jsonBytes, _ := json.Marshal(result)
	return string(jsonBytes)
}

func main() {
	// Initialize the global engine
	globalEngine = NewWASMSQLEngine()

	// Set up WASM bindings
	js.Global().Set("executeSQL", js.FuncOf(executeSQL))
	js.Global().Set("startRecording", js.FuncOf(startRecording))
	js.Global().Set("stopRecording", js.FuncOf(stopRecording))
	js.Global().Set("getRecordedQueries", js.FuncOf(getRecordedQueries))
	js.Global().Set("clearRecordedQueries", js.FuncOf(clearRecordedQueries))

	// Keep the program running
	select {}
}