package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	// Check if we have the correct number of arguments
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <filename>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example: %s myfile.txt\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "This version uploads to local server at localhost:8080\n")
		os.Exit(1)
	}

	filename := os.Args[1]
	urlString := filepath.Base(filename)

	// Check if file exists and can be opened
	file, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file '%s': %v\n", filename, err)
		os.Exit(1)
	}
	defer file.Close()

	// Get file info for logging
	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting file info: %v\n", err)
		os.Exit(1)
	}

	// Construct the URL for local server
	baseURL := "http://localhost:8080"
	url := fmt.Sprintf("%s/files/%s", baseURL, urlString)

	fmt.Printf("Uploading file '%s' (%d bytes) to %s\n", filename, fileInfo.Size(), url)

	// Create the PUT request
	req, err := http.NewRequest("PUT", url, file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		os.Exit(1)
	}

	// Set appropriate headers
	req.Header.Set("Content-Type", "application/octet-stream")
	req.ContentLength = fileInfo.Size()

	// Optional: Set additional headers that might be useful
	req.Header.Set("X-Filename", filepath.Base(filename))

	// Create HTTP client and send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending request: %v\n", err)
		fmt.Fprintf(os.Stderr, "Make sure the server is running with: go run server.go\n")
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Read and display the response
	fmt.Printf("Response Status: %s\n", resp.Status)
	fmt.Printf("Response Headers:\n")
	for key, values := range resp.Header {
		for _, value := range values {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response body: %v\n", err)
		os.Exit(1)
	}

	if len(body) > 0 {
		fmt.Printf("Response Body:\n%s\n", string(body))
	}

	// Check if the request was successful
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Printf("File uploaded successfully!\n")
	} else {
		fmt.Fprintf(os.Stderr, "Upload failed with status: %s\n", resp.Status)
		os.Exit(1)
	}
}
