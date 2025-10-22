# File Upload Program

A Go program that uploads files to a web server using HTTP PUT requests.

## Description

This program takes a filename and a string as command-line arguments, then sends the specified file to `https://comp3007-f25.scs.carleton.ca` using a PUT HTTP request to the URL `/files/<string>` where `<string>` is the second argument provided.

## Usage

```bash
go run upload.go <filename> <string>
```

### Arguments

- `<filename>`: Path to the file you want to upload (the base filename will be used on the server)

### Examples

```bash
# Upload test.txt to /files/test.txt
go run upload.go test.txt

# Upload a document to /files/document.pdf
go run upload.go document.pdf

# Upload data.json to /files/data.json
go run upload.go data.json

# Upload from a subdirectory - will be stored as /files/file.txt
go run upload.go path/to/file.txt
```

## Building the Program

To compile the program into an executable:

```bash
go build -o upload upload.go
```

Then run it directly:

```bash
./upload test.txt
```

## Features

- **Error Handling**: Checks if file exists and can be read
- **File Information**: Displays file size and upload progress
- **Response Details**: Shows server response status, headers, and body
- **Content Headers**: Sets appropriate HTTP headers including Content-Type and Content-Length
- **Success/Failure Reporting**: Clearly indicates whether the upload succeeded or failed

## HTTP Details

- **Method**: PUT
- **URL Pattern**: `https://comp3007-f25.scs.carleton.ca/files/<filename>`
- **Content-Type**: `application/octet-stream`
- **Additional Headers**:
  - `Content-Length`: Automatically set to file size
  - `X-Filename`: Set to the base name of the uploaded file

## Sample Output

```
$ go run upload.go test.txt example123
Uploading file 'test.txt' (245 bytes) to https://comp3007-f25.scs.carleton.ca/files/example123
Response Status: 200 OK
Response Headers:
  Content-Type: application/json
  Date: Mon, 02 Dec 2024 15:30:45 GMT
Response Body:
{"status": "success", "message": "File uploaded successfully"}
File uploaded successfully!
```

## Error Handling

The program handles various error conditions:

- **File not found**: If the specified file doesn't exist
- **Permission errors**: If the file cannot be read
- **Network errors**: If the HTTP request fails
- **Server errors**: If the server returns an error status code

## Requirements

- Go 1.16 or later
- Internet connection to reach the target server
- Read permissions for the file being uploaded

## Testing

A sample test file (`test.txt`) is included for testing purposes. You can use it like this:

```bash
go run upload.go test.txt
```
