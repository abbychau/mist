#!/bin/bash

echo "=== Mist Daemon Mode Demo ==="
echo

echo "This script demonstrates how to use Mist in daemon mode."
echo
echo "1. Start the daemon:"
echo "   go run . -d --port 3307"
echo
echo "2. In another terminal, connect with:"
echo "   telnet localhost 3307"
echo "   # or"
echo "   nc localhost 3307"
echo
echo "3. Try these SQL commands:"
echo "   CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(50));"
echo "   INSERT INTO users VALUES (1, 'Alice');"
echo "   INSERT INTO users VALUES (2, 'Bob');"
echo "   SELECT * FROM users;"
echo "   SELECT (SELECT COUNT(*) FROM users) as total;"
echo "   help"
echo "   quit"
echo
echo "4. Stop the daemon with Ctrl+C"
echo

# Check if daemon is already running
if nc -z localhost 3307 2>/dev/null; then
    echo "✅ Daemon appears to be running on port 3307"
    echo "You can connect with: telnet localhost 3307"
else
    echo "ℹ️  No daemon detected on port 3307"
    echo "Start with: go run . -d --port 3307"
fi

echo
echo "Features supported in daemon mode:"
echo "  ✅ All SQL operations (CREATE, INSERT, SELECT, UPDATE, DELETE)"
echo "  ✅ Scalar subqueries"
echo "  ✅ JOINs and aggregates"
echo "  ✅ Transactions (BEGIN, COMMIT, ROLLBACK)"
echo "  ✅ Indexes and constraints"
echo "  ✅ Multiple concurrent connections"
echo "  ✅ Graceful shutdown"
echo