package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	fmt.Println("=== Simple Daemon Connection Test ===")
	
	// Test basic connection to daemon
	// (Assumes daemon is running on port 3307)
	
	fmt.Println("Testing connection to Mist daemon...")
	fmt.Println("Note: Start daemon first with: go run . -d --port 3307")
	
	conn, err := net.DialTimeout("tcp", "localhost:3307", 5*time.Second)
	if err != nil {
		fmt.Printf("Could not connect to daemon (is it running?): %v\n", err)
		fmt.Println("Start daemon with: go run . -d --port 3307")
		return
	}
	defer conn.Close()
	
	fmt.Println("✅ Successfully connected to Mist daemon!")
	
	// Send a simple test
	fmt.Println("Sending test command...")
	conn.Write([]byte("help\n"))
	
	// Read a bit of response
	buffer := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
	} else {
		fmt.Printf("Received %d bytes from server\n", n)
		fmt.Printf("First part of response: %s...\n", string(buffer[:min(100, n)]))
	}
	
	// Send quit command
	conn.Write([]byte("quit\n"))
	
	fmt.Println("✅ Test completed successfully!")
	fmt.Println()
	fmt.Println("To test manually:")
	fmt.Println("  1. Start daemon: go run . -d --port 3307")
	fmt.Println("  2. Connect: telnet localhost 3307")
	fmt.Println("  3. Try SQL: CREATE TABLE test (id INT); INSERT INTO test VALUES (1); SELECT * FROM test;")
}