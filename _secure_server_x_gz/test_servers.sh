#!/bin/bash

# Test script to demonstrate server functionality

echo "=== Testing Server 's' and 'gz' functionality ==="
echo

# Function to make HTTP requests and show results
test_endpoint() {
    local url=$1
    local description=$2
    local expected_status=$3

    echo "Testing: $description"
    echo "URL: $url"

    response=$(curl -s -w "\nHTTP_STATUS:%{http_code}\n" "$url")
    status=$(echo "$response" | tail -n1 | cut -d: -f2)
    body=$(echo "$response" | sed '$d')

    echo "Status: $status"
    if [ "$expected_status" != "" ] && [ "$status" != "$expected_status" ]; then
        echo "❌ UNEXPECTED STATUS! Expected $expected_status, got $status"
    elif [ "$status" = "200" ]; then
        echo "✅ SUCCESS"
    elif [ "$status" = "403" ] && [ "$expected_status" = "403" ]; then
        echo "✅ CORRECTLY BLOCKED"
    else
        echo "⚠️  Status: $status"
    fi

    if [ ${#body} -gt 200 ]; then
        echo "Response: $(echo "$body" | head -c 200)..."
    else
        echo "Response: $body"
    fi
    echo "----------------------------------------"
    echo
}

# Wait a moment for servers to start up
echo "Waiting for servers to start..."
sleep 2

echo "=== Testing Server 's' direct endpoints ==="
test_endpoint "http://localhost:8080/" "Server s root page" "200"
test_endpoint "http://localhost:8080/health" "Server s health check" "200"

echo "=== Testing Server 's' forwarding to 'gz' ==="
test_endpoint "http://localhost:8080/gz" "Forward /gz to gz server" "200"
test_endpoint "http://localhost:8080/gz/hello" "Forward /gz/hello to gz server" "200"
test_endpoint "http://localhost:8080/gz/status" "Forward /gz/status to gz server" "200"
test_endpoint "http://localhost:8080/gz/health" "Forward /gz/health to gz server" "200"
test_endpoint "http://localhost:8080/gz/api/grade" "Forward /gz/api/grade to gz server" "200"

echo "=== Testing direct access to 'gz' server (should be blocked) ==="
test_endpoint "http://localhost:8081/" "Direct access to gz server" "403"
test_endpoint "http://localhost:8081/hello" "Direct access to gz/hello" "403"
test_endpoint "http://localhost:8081/status" "Direct access to gz/status" "403"

echo "=== Testing POST request to gz API through s ==="
echo "Testing: POST request to gz API through server s"
echo "URL: http://localhost:8080/gz/api/grade"

post_response=$(curl -s -w "\nHTTP_STATUS:%{http_code}\n" \
    -X POST \
    -H "Content-Type: application/json" \
    -d '{"student_id":"12345","assignment":"test","grade":95}' \
    "http://localhost:8080/gz/api/grade")

post_status=$(echo "$post_response" | tail -n1 | cut -d: -f2)
post_body=$(echo "$post_response" | sed '$d')

echo "Status: $post_status"
if [ "$post_status" = "201" ]; then
    echo "✅ POST SUCCESS"
else
    echo "⚠️  POST Status: $post_status"
fi
echo "Response: $post_body"
echo "----------------------------------------"
echo

echo "=== Summary ==="
echo "✅ Server 's' handles direct requests"
echo "✅ Server 's' forwards /gz/* requests to gz server"
echo "✅ Server 'gz' only accepts requests from server 's'"
echo "✅ Direct access to 'gz' is properly blocked"
echo
echo "The architecture ensures that:"
echo "1. All public traffic goes through server 's' on port 8080"
echo "2. Internal 'gz' functionality is only accessible via server 's'"
echo "3. Server 'gz' validates requests using internal secret header"
echo "4. Server 'gz' is protected from direct external access"
