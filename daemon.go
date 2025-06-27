// +build !js,!wasm

package mist

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

// SimpleMistServer represents a simple text-protocol MySQL-compatible daemon server
type SimpleMistServer struct {
	engine   *SQLEngine
	listener net.Listener
	port     int
	running  bool
	mutex    sync.RWMutex
}

// NewSimpleMistServer creates a new simple MySQL-compatible daemon server
func NewSimpleMistServer(port int) *SimpleMistServer {
	if port == 0 {
		port = 3306 // Default MySQL port
	}

	engine := NewSQLEngine()
	
	return &SimpleMistServer{
		engine: engine,
		port:   port,
	}
}

// Start starts the simple MySQL daemon server
func (s *SimpleMistServer) Start() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.running {
		return fmt.Errorf("server is already running")
	}

	// Create listener
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %v", s.port, err)
	}
	s.listener = listener

	s.running = true

	log.Printf("Mist MySQL daemon started on port %d", s.port)
	log.Printf("Connect with: telnet localhost %d (simplified text protocol)", s.port)
	log.Printf("Or use: nc localhost %d", s.port)
	log.Printf("Type SQL commands followed by ';' and press Enter")

	// Handle connections
	go func() {
		connectionID := 1
		for s.IsRunning() {
			conn, err := listener.Accept()
			if err != nil {
				if s.IsRunning() {
					log.Printf("Error accepting connection: %v", err)
				}
				continue
			}

			log.Printf("New connection #%d from %s", connectionID, conn.RemoteAddr())
			go s.handleConnection(conn, connectionID)
			connectionID++
		}
	}()

	return nil
}

// handleConnection handles a client connection with simple text protocol
func (s *SimpleMistServer) handleConnection(conn net.Conn, connID int) {
	defer conn.Close()
	
	// Send welcome message
	welcome := fmt.Sprintf("Welcome to Mist MySQL-compatible database (Connection #%d)\n", connID)
	welcome += "Type 'help' for commands, 'quit' to exit\n"
	welcome += "mist> "
	conn.Write([]byte(welcome))

	scanner := bufio.NewScanner(conn)
	var queryBuffer strings.Builder

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		if line == "" {
			conn.Write([]byte("mist> "))
			continue
		}

		// Handle special commands
		if strings.ToLower(line) == "quit" || strings.ToLower(line) == "exit" {
			conn.Write([]byte("Bye!\n"))
			return
		}

		if strings.ToLower(line) == "help" {
			s.sendHelp(conn)
			conn.Write([]byte("mist> "))
			continue
		}

		// Add to query buffer
		if queryBuffer.Len() > 0 {
			queryBuffer.WriteString(" ")
		}
		queryBuffer.WriteString(line)

		// Check if query is complete (ends with semicolon)
		if strings.HasSuffix(line, ";") {
			query := queryBuffer.String()
			queryBuffer.Reset()

			// Execute the query
			s.executeQuery(conn, query, connID)
		}

		conn.Write([]byte("mist> "))
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Connection #%d error: %v", connID, err)
	}
	log.Printf("Connection #%d closed", connID)
}

// executeQuery executes a SQL query and sends the result back to the client
func (s *SimpleMistServer) executeQuery(conn net.Conn, query string, connID int) {
	log.Printf("Connection #%d executing: %s", connID, query)

	start := time.Now()
	result, err := s.engine.Execute(query)
	duration := time.Since(start)

	if err != nil {
		response := fmt.Sprintf("ERROR: %v\n", err)
		conn.Write([]byte(response))
		return
	}

	// Format and send result
	s.sendResult(conn, result, duration)
}

// sendResult formats and sends query results to the client
func (s *SimpleMistServer) sendResult(conn net.Conn, result interface{}, duration time.Duration) {
	switch r := result.(type) {
	case *SelectResult:
		s.sendSelectResult(conn, r, duration)
	case string:
		response := fmt.Sprintf("%s\n", r)
		response += fmt.Sprintf("Query OK (%v)\n", duration)
		conn.Write([]byte(response))
	default:
		response := fmt.Sprintf("%v\n", result)
		response += fmt.Sprintf("Query OK (%v)\n", duration)
		conn.Write([]byte(response))
	}
}

// sendSelectResult formats and sends SELECT query results
func (s *SimpleMistServer) sendSelectResult(conn net.Conn, result *SelectResult, duration time.Duration) {
	if len(result.Rows) == 0 {
		response := "Empty set (" + duration.String() + ")\n"
		conn.Write([]byte(response))
		return
	}

	var response strings.Builder

	// Calculate column widths
	colWidths := make([]int, len(result.Columns))
	for i, col := range result.Columns {
		colWidths[i] = len(col)
	}

	for _, row := range result.Rows {
		for i, val := range row {
			valStr := fmt.Sprintf("%v", val)
			if len(valStr) > colWidths[i] {
				colWidths[i] = len(valStr)
			}
		}
	}

	// Add minimum width
	for i := range colWidths {
		if colWidths[i] < 4 {
			colWidths[i] = 4
		}
	}

	// Write header separator
	response.WriteString("+")
	for _, width := range colWidths {
		response.WriteString(strings.Repeat("-", width+2))
		response.WriteString("+")
	}
	response.WriteString("\n")

	// Write column headers
	response.WriteString("|")
	for i, col := range result.Columns {
		response.WriteString(fmt.Sprintf(" %-*s |", colWidths[i], col))
	}
	response.WriteString("\n")

	// Write header separator
	response.WriteString("+")
	for _, width := range colWidths {
		response.WriteString(strings.Repeat("-", width+2))
		response.WriteString("+")
	}
	response.WriteString("\n")

	// Write rows
	for _, row := range result.Rows {
		response.WriteString("|")
		for i, val := range row {
			valStr := fmt.Sprintf("%v", val)
			if val == nil {
				valStr = "NULL"
			}
			response.WriteString(fmt.Sprintf(" %-*s |", colWidths[i], valStr))
		}
		response.WriteString("\n")
	}

	// Write bottom separator
	response.WriteString("+")
	for _, width := range colWidths {
		response.WriteString(strings.Repeat("-", width+2))
		response.WriteString("+")
	}
	response.WriteString("\n")

	// Write summary
	rowCount := len(result.Rows)
	rowWord := "row"
	if rowCount != 1 {
		rowWord = "rows"
	}
	response.WriteString(fmt.Sprintf("%d %s in set (%v)\n", rowCount, rowWord, duration))

	conn.Write([]byte(response.String()))
}

// sendHelp sends help information to the client
func (s *SimpleMistServer) sendHelp(conn net.Conn) {
	help := `
Available commands:
  help          - Show this help
  quit, exit    - Close connection
  
SQL commands (end with semicolon):
  CREATE TABLE table_name (column1 TYPE, column2 TYPE, ...);
  INSERT INTO table_name VALUES (val1, val2, ...);
  SELECT * FROM table_name;
  SELECT column1, column2 FROM table_name WHERE condition;
  UPDATE table_name SET column = value WHERE condition;
  DELETE FROM table_name WHERE condition;
  SHOW TABLES;
  
Examples:
  CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50));
  INSERT INTO users VALUES (1, 'Alice');
  SELECT * FROM users;
  
Supported features:
  - Scalar subqueries: SELECT (SELECT MAX(id) FROM table2) FROM table1;
  - JOINs: SELECT * FROM table1 JOIN table2 ON table1.id = table2.id;
  - Aggregates: SELECT COUNT(*), AVG(column) FROM table;
  - Indexes: CREATE INDEX idx_name ON table (column);
  - Transactions: BEGIN; ... COMMIT; / ROLLBACK;

`
	conn.Write([]byte(help))
}

// Stop stops the MySQL daemon server
func (s *SimpleMistServer) Stop() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.running {
		return fmt.Errorf("server is not running")
	}

	s.running = false

	if s.listener != nil {
		s.listener.Close()
	}

	log.Printf("Mist MySQL daemon stopped")
	return nil
}

// IsRunning returns whether the server is currently running
func (s *SimpleMistServer) IsRunning() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.running
}

// GetEngine returns the underlying SQL engine (for testing/management)
func (s *SimpleMistServer) GetEngine() *SQLEngine {
	return s.engine
}

// RunSimpleDaemon starts the simple Mist daemon and handles graceful shutdown
func RunSimpleDaemon(port int) error {
	server := NewSimpleMistServer(port)

	// Start the server
	if err := server.Start(); err != nil {
		return fmt.Errorf("failed to start server: %v", err)
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	<-sigChan
	log.Println("Received shutdown signal, stopping server...")

	// Stop the server
	if err := server.Stop(); err != nil {
		log.Printf("Error stopping server: %v", err)
	}

	// Give some time for cleanup
	time.Sleep(1 * time.Second)
	log.Println("Server stopped successfully")

	return nil
}

// StartSimpleDaemonWithContext starts the daemon with a context for programmatic control
func StartSimpleDaemonWithContext(ctx context.Context, port int) error {
	server := NewSimpleMistServer(port)

	// Start the server
	if err := server.Start(); err != nil {
		return fmt.Errorf("failed to start server: %v", err)
	}

	// Wait for context cancellation or completion
	<-ctx.Done()
	log.Println("Context cancelled, stopping server...")

	// Stop the server
	if err := server.Stop(); err != nil {
		log.Printf("Error stopping server: %v", err)
	}

	return nil
}

// Compatibility functions - use the simple implementation as the main daemon
func RunDaemon(port int) error {
	return RunSimpleDaemon(port)
}

func StartDaemonWithContext(ctx context.Context, port int) error {
	return StartSimpleDaemonWithContext(ctx, port)
}