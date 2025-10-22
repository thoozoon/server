# Quick Start Guide - File Upload Server

This directory contains a complete file upload system with both client and server components.

## What's Here

- `server.go` - Go web server that receives file uploads
- `upload.go` - Client that uploads to the remote server (comp3007-f25.scs.carleton.ca)
- `upload_local.go` - Client that uploads to the local server for testing
- `test.txt` - Sample file for testing
- `uploads/` - Directory where uploaded files are stored
- `test_server.sh` - Automated test script

## Quick Start (Local Testing)

1. **Start the server** (in one terminal):

   ```bash
   go run server.go
   ```

   You should see:

   ```
   Starting server on port :8080
   Upload directory: ./uploads
   Send PUT requests to: http://localhost:8080/files/<filename>
   ```

2. **Test the upload** (in another terminal):

   ```bash
   go run upload_local.go test.txt
   ```

3. **Check the result**:
   ```bash
   ls -la uploads/
   cat uploads/test.txt
   ```

## Testing with curl

```bash
# Upload a file
curl -X PUT --data-binary @test.txt http://localhost:8080/files/example.txt

# Upload text directly
echo "Hello World" | curl -X PUT --data-binary @- http://localhost:8080/files/hello.txt
```

## Running Automated Tests

```bash
./test_server.sh
```

## Remote Server Upload

To upload to the actual course server:

```bash
go run upload.go test.txt
```

## How It Works

1. **Server** (`server.go`):
   - Listens on port 8080
   - Accepts PUT requests to `/files/<filename>`
   - Saves files to `uploads/` directory
   - Returns JSON response with upload status

2. **Client** (`upload_local.go`):
   - Takes filename as argument
   - Uses the base filename for the server endpoint
   - Sends PUT request with file content
   - Displays server response

## Troubleshooting

- **Port in use**: Change the port in `server.go` if 8080 is busy
- **Permission denied**: Make sure you have write permissions in this directory
- **File not found**: Check that the file you're uploading exists

## Files Generated

All uploaded files appear in the `uploads/` directory with the filename specified in the URL path.
