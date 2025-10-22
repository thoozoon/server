#!/bin/bash

# Test script for the file upload server

SERVER_URL="http://localhost:8080"
TEST_FILE="test.txt"

echo "=== File Upload Server Test Script ==="
echo

# Check if server is running
echo "1. Checking if server is running..."
if curl -s "$SERVER_URL" >/dev/null; then
    echo "✓ Server is running at $SERVER_URL"
else
    echo "✗ Server is not running. Please start it with: go run server.go"
    exit 1
fi

echo

# Test 1: Upload the test file
echo "2. Testing file upload..."
if [ -f "$TEST_FILE" ]; then
    echo "Uploading $TEST_FILE using Go client..."
    go run upload_local.go "$TEST_FILE"

    # Check if file was created (using the actual filename)
    if [ -f "uploads/$TEST_FILE" ]; then
        echo "✓ File successfully uploaded and saved to uploads/$TEST_FILE"
    else
        echo "✗ File was not found in uploads directory"
    fi
else
    echo "✗ Test file $TEST_FILE not found"
fi

echo

# Test 2: Try invalid method (GET)
echo "3. Testing invalid method (GET)..."
response=$(curl -s -w "%{http_code}" "$SERVER_URL/files/test.txt")
http_code="${response: -3}"
if [ "$http_code" = "405" ]; then
    echo "✓ Server correctly rejected GET request with 405 Method Not Allowed"
else
    echo "✗ Server did not handle invalid method correctly (got $http_code)"
fi

echo

# Test 3: Try empty filename
echo "4. Testing empty filename..."
response=$(curl -s -w "%{http_code}" -X PUT "$SERVER_URL/files/")
http_code="${response: -3}"
if [ "$http_code" = "400" ]; then
    echo "✓ Server correctly rejected empty filename with 400 Bad Request"
else
    echo "✗ Server did not handle empty filename correctly (got $http_code)"
fi

echo

# Test 4: Upload a simple text string
echo "5. Testing direct text upload..."
echo "Hello, World! This is a test." | curl -s -X PUT --data-binary @- "$SERVER_URL/files/hello.txt"
if [ -f "uploads/hello.txt" ]; then
    echo "✓ Direct text upload successful"
    echo "Content: $(cat uploads/hello.txt)"
else
    echo "✗ Direct text upload failed"
fi

echo

# List uploaded files
echo "6. Files in uploads directory:"
if [ -d "uploads" ]; then
    ls -la uploads/
else
    echo "No uploads directory found"
fi

echo
echo "=== Test completed ==="
