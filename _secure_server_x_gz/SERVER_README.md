# Server Architecture: 's' and 'gz'

This module implements a dual-server architecture where server 's' acts as a public-facing proxy that forwards specific requests to an internal server 'gz'.

## Architecture Overview

```
[Client] → [Server 's'] → [Server 'gz']
         Port 8080      Port 8081
```

- **Server 's'**: Public-facing server on port 8080
- **Server 'gz'**: Internal server on port 8081, only accessible via server 's'

## Security Model

### Server 's' (Public)
- Accepts requests from any client
- Handles requests directly for most paths
- Forwards requests starting with `/gz` to server 'gz'
- Adds internal authentication header when forwarding

### Server 'gz' (Internal)
- Only accepts requests with valid internal secret header
- Rejects direct external requests with 403 Forbidden
- Only accessible through server 's' proxy

## Request Flow

1. **Direct requests to 's'**: `http://localhost:8080/` → Handled by server 's'
2. **Proxied requests**: `http://localhost:8080/gz/*` → Forwarded to server 'gz'
3. **Blocked requests**: `http://localhost:8081/*` → Returns 403 Forbidden

## Running the Servers

### Option 1: Using the run script
```bash
./run_servers.sh
```

### Option 2: Manual startup
```bash
# Terminal 1 - Start gz server
go run server_gz.go

# Terminal 2 - Start s server
go run server_s.go
```

### Option 3: Build and run binaries
```bash
go build server_s.go
go build server_gz.go

# Run in background
./server_gz &
./server_s &
```

## Testing

Run the test script to verify functionality:
```bash
./test_servers.sh
```

## Available Endpoints

### Server 's' Direct Endpoints
- `GET /` - Server 's' homepage
- `GET /health` - Server 's' health check

### Server 'gz' Endpoints (via '/gz' prefix)
- `GET /gz/` - Server 'gz' homepage
- `GET /gz/hello` - JSON greeting from gz
- `GET /gz/status` - Server status information
- `GET /gz/health` - Health check endpoint
- `GET /gz/api/grade` - Get grading data (mock)
- `POST /gz/api/grade` - Submit grade data

## Examples

### Valid Requests
```bash
# Direct to server 's'
curl http://localhost:8080/
curl http://localhost:8080/health

# Forwarded to server 'gz'
curl http://localhost:8080/gz/
curl http://localhost:8080/gz/hello
curl http://localhost:8080/gz/status

# POST to gz API through s
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"student":"123","grade":95}' \
  http://localhost:8080/gz/api/grade
```

### Blocked Requests (Direct to 'gz')
```bash
# These will return 403 Forbidden
curl http://localhost:8081/
curl http://localhost:8081/hello
curl http://localhost:8081/status
```

## Implementation Details

### Authentication Mechanism
- Server 's' adds internal secret header: `X-Internal-Secret: s-to-gz-internal-token-12345`
- Server 'gz' validates this header on all requests
- Invalid or missing header results in 403 Forbidden

### Path Rewriting
- `/gz/hello` on server 's' becomes `/hello` on server 'gz'
- `/gz/` becomes `/` on server 'gz'
- Original request path is preserved in logs

### Error Handling
- Invalid authentication → 403 Forbidden
- Server 'gz' unavailable → Proxy error from server 's'
- Invalid routes → 404 Not Found

## Configuration

Constants in both server files can be modified:
- `SERVER_S_PORT`: Port for server 's' (default: :8080)
- `SERVER_GZ_PORT`: Port for server 'gz' (default: :8081)
- `GZ_PREFIX`: URL prefix for forwarding (default: "/gz")
- `INTERNAL_SECRET_*`: Authentication token and header name

## Security Considerations

1. **Internal Token**: Change the default internal secret token in production
2. **Network Access**: In production, ensure server 'gz' is not directly accessible from external networks
3. **HTTPS**: Use HTTPS for production deployments
4. **Rate Limiting**: Consider adding rate limiting to server 's'
5. **Logging**: Monitor and log all authentication failures

## Use Cases

This architecture is useful when you need:
- Internal microservices that shouldn't be directly accessible
- API Gateway pattern with request routing
- Service isolation with controlled access
- Centralized authentication and request processing

## Extending the System

To add new internal services:
1. Create new server similar to 'gz'
2. Add forwarding rules to server 's'
3. Use the same authentication mechanism
4. Update the run and test scripts
