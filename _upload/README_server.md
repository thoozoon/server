# File Upload Server

A simple Go web server that receives file uploads via HTTP PUT requests and stores them locally.

## Overview

This server provides a REST API endpoint for file uploads. Files are uploaded using PUT requests to `/files/<filename>` and are stored in the local `uploads/` directory using the same filename as specified in the URL.

## Features

- **Simple REST API**: Upload files via PUT requests to `/files/<filename>`
- **Local Storage**: Files are saved to the `uploads/` directory
- **Security**: Basic filename sanitization to prevent directory traversal attacks
- **Logging**: Request logging with file sizes and paths
- **JSON Responses**: Structured JSON responses for upload results
- **Error Handling**: Proper HTTP status codes for various error conditions

## Getting Started

### Prerequisites

- Go 1.16 or later

### Running the Server

1. Navigate to the server directory:
   ```bash
   cd /path/to/upload/directory
   ```

2. Start the server:
   ```bash
   go run server.go
   ```

   The server will start on port 8080 and create an `uploads/` directory if it doesn't exist.

3. You should see output like:
   ```
   2024/12/02 10:30:00 Starting server on port :8080
   2024/12/02 10:30:00 Upload directory: ./uploads
   2024/12/02 10:30:00 Send PUT requests to: http://localhost:8080/files/<filename>
   ```

### Building the Server

To compile into an executable:

```bash
go build -o fileserver server.go
./fileserver
```

## API Endpoints

### Upload File
- **URL**: `/files/<filename>`
- **Method**: `PUT`
- **Body**: Raw file content (binary)
- **Response**: JSON with upload status

#### Example Request
```bash
curl -X PUT --data-binary @myfile.txt http://localhost:8080/files/myfile.txt
```

#### Example Response
```json
{
    "status": "success",
    "message": "File uploaded successfully",
    "filename": "myfile.txt",
    "bytes_written": 1234,
    "path": "./uploads/myfile.txt"
}
```

### Server Info
- **URL**: `/`
- **Method**: `GET`
- **Response**: HTML page with server information

## Usage Examples

### Using curl

```bash
# Upload a text file
curl -X PUT --data-binary @document.txt http://localhost:8080/files/document.txt

# Upload a binary file
curl -X PUT --data-binary @image.jpg http://localhost:8080/files/image.jpg

# Upload from stdin
echo "Hello World" | curl -X PUT --data-binary @- http://localhost:8080/files/hello.txt
```

### Using the Go upload client

If you have the companion upload client (`upload.go`):

```bash
# Upload to local server (modify the client to use localhost
