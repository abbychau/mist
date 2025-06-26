package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/abbychau/mist"
)

func main() {
	fmt.Println("=== Testing Mist Daemon Mode ===")

	// Start daemon in background
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	port := 3308 // Use non-standard port to avoid conflicts
	
	fmt.Printf("Starting Mist daemon on port %d...\n", port)
	
	// Start daemon in a goroutine
	daemonDone := make(chan error, 1)
	go func() {
		err := mist.StartDaemonWithContext(ctx, port)
		daemonDone <- err
	}()

	// Give daemon time to start
	time.Sleep(2 * time.Second)

	// Test client connection
	fmt.Printf("Connecting to daemon on port %d...\n", port)
	
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}
	defer conn.Close()

	// Read welcome message
	reader := bufio.NewReader(conn)
	
	fmt.Println("\n--- Reading welcome message ---")
	for i := 0; i < 3; i++ { // Read a few lines of welcome
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		fmt.Printf("Server: %s", line)
		if strings.Contains(line, "mist>") {
			break
		}
	}

	// Test SQL commands
	testQueries := []string{
		"CREATE TABLE test_users (id INT PRIMARY KEY, name VARCHAR(50));",
		"INSERT INTO test_users VALUES (1, 'Alice');",
		"INSERT INTO test_users VALUES (2, 'Bob');",
		"SELECT * FROM test_users;",
		"SELECT COUNT(*) FROM test_users;",
		"SELECT (SELECT MAX(id) FROM test_users) as max_id;", // Test scalar subquery
		"quit",
	}

	fmt.Println("\n--- Testing SQL commands ---")
	
	for _, query := range testQueries {
		fmt.Printf("\nSending: %s\n", query)
		
		// Send query
		_, err := conn.Write([]byte(query + "\n"))
		if err != nil {
			fmt.Printf("Error sending query: %v\n", err)
			break
		}

		if query == "quit" {
			// Read final response and exit
			line, _ := reader.ReadString('\n')
			fmt.Printf("Server: %s", line)
			break
		}

		// Read response until we see the prompt again
		fmt.Println("Response:")
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				fmt.Printf("Error reading response: %v\n", err)
				break
			}
			
			fmt.Printf("  %s", line)
			
			// Stop reading when we see the prompt
			if strings.Contains(line, "mist>") {
				break
			}
		}
	}

	fmt.Println("\n--- Test completed ---")
	
	// Cancel context to stop daemon
	cancel()
	
	// Wait for daemon to finish
	select {
	case err := <-daemonDone:
		if err != nil && err != context.Canceled {
			fmt.Printf("Daemon error: %v\n", err)
		} else {
			fmt.Println("Daemon stopped cleanly")
		}
	case <-time.After(5 * time.Second):
		fmt.Println("Daemon shutdown timeout")
	}
	
	fmt.Println("Test completed successfully!")
}