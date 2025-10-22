package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	uploadDir = "./uploads"
	port      = ":8080"
)

func main() {
	// Create uploads directory if it doesn't exist
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Fatalf("Failed to create uploads directory: %v", err)
	}

	// Set up HTTP routes
	http.HandleFunc("/files/", handleFileUpload)
	http.HandleFunc("/", handleRoot)

	log.Printf("Starting server on port %s", port)
	log.Printf("Upload directory: %s", uploadDir)
	log.Printf("Send PUT requests to: http://localhost%s/files/<filename>", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head>
    <title>File Upload Server</title>
</head>
<body>
    <h1>File Upload Server</h1>
    <p>This server accepts file uploads via PUT requests.</p>
    <p>Send PUT requests to: <code>/files/&lt;filename&gt;</code></p>
    <p>Example: <code>curl -X PUT --data-binary @myfile.txt http://localhost%s/files/myfile.txt</code></p>
</body>
</html>
`, port)
}

func handleFileUpload(w http.ResponseWriter, r *http.Request) {
	// Only accept PUT requests
	if r.Method != http.MethodPut {
		w.Header().Set("Allow", "PUT")
		http.Error(w, "Method not allowed. Use PUT to upload files.", http.StatusMethodNotAllowed)
		return
	}

	// Extract filename from URL path
	path := strings.TrimPrefix(r.URL.Path, "/files/")
	if path == "" {
		http.Error(w, "Filename is required in URL path", http.StatusBadRequest)
		return
	}

	// Clean the filename to prevent directory traversal attacks
	filename := filepath.Base(path)
	if filename == "." || filename == ".." {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	// Create the full file path
	filePath := filepath.Join(uploadDir, filename)

	log.Printf("Receiving file upload: %s -> %s", path, filePath)

	// Create the file
	file, err := os.Create(filePath)
	if err != nil {
		log.Printf("Error creating file %s: %v", filePath, err)
		http.Error(w, "Failed to create file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Copy the request body to the file
	bytesWritten, err := io.Copy(file, r.Body)
	if err != nil {
		log.Printf("Error writing file %s: %v", filePath, err)
		http.Error(w, "Failed to write file", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully uploaded file: %s (%d bytes)", filename, bytesWritten)

	// Send success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{
	"status": "success",
	"message": "File uploaded successfully",
	"filename": "%s",
	"bytes_written": %d,
	"path": "%s"
}`, filename, bytesWritten, filePath)
}
