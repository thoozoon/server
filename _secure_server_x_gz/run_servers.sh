#!/bin/bash

# Script to run both server 's' and server 'gz'

echo "Starting both servers..."

# Function to cleanup background processes on exit
cleanup() {
    echo "Shutting down servers..."
    if [ ! -z "$GZ_PID" ]; then
        kill $GZ_PID 2>/dev/null
        echo "Stopped gz server (PID: $GZ_PID)"
    fi
    if [ ! -z "$S_PID" ]; then
        kill $S_PID 2>/dev/null
        echo "Stopped s server (PID: $S_PID)"
    fi
    exit 0
}

# Set up trap to cleanup on script exit
trap cleanup SIGINT SIGTERM EXIT

# Build the servers
echo "Building servers..."
go build -o server_s server_s.go
go build -o server_gz server_gz.go

if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi

# Start gz server in background
echo "Starting gz server on :8081..."
./server_gz &
GZ_PID=$!
sleep 1

# Start s server in background
echo "Starting s server on :8080..."
./server_s &
S_PID=$!
sleep 1

echo ""
echo "Both servers are now running:"
echo "  - Server 's' (main): http://localhost:8080"
echo "  - Server 'gz' (internal): http://localhost:8081 (only accessible via server s)"
echo ""
echo "Try these URLs:"
echo "  - http://localhost:8080/ (handled by s)"
echo "  - http://localhost:8080/health (handled by s)"
echo "  - http://localhost:8080/gz (forwarded to gz)"
echo "  - http://localhost:8080/gz/hello (forwarded to gz)"
echo "  - http://localhost:8080/gz/status (forwarded to gz)"
echo "  - http://localhost:8080/gz/api/grade (forwarded to gz)"
echo ""
echo "Direct access to gz server will be blocked:"
echo "  - http://localhost:8081/ (should return 403 Forbidden)"
echo ""
echo "Press Ctrl+C to stop both servers..."

# Wait for both processes
wait $S_PID $GZ_PID
