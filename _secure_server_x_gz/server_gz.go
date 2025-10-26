package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	SERVER_GZ_PORT = ":8081"
	// Secret header to verify requests come from server s
	INTERNAL_SECRET_HEADER = "X-Internal-Secret"
	INTERNAL_SECRET_VALUE  = "s-to-gz-internal-token-12345"
)

func main() {
	mux := http.NewServeMux()

	// Middleware to check authentication
	authenticatedHandler := func(handler http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Check for the secret header
			secretHeader := r.Header.Get(INTERNAL_SECRET_HEADER)
			if secretHeader != INTERNAL_SECRET_VALUE {
				log.Printf("Unauthorized request from %s: missing or invalid secret header", r.RemoteAddr)
				http.Error(w, "Forbidden: This server only accepts requests from server s", http.StatusForbidden)
				return
			}

			log.Printf("Authenticated request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
			handler(w, r)
		}
	}

	// Root handler for /
	mux.HandleFunc("/", authenticatedHandler(func(w http.ResponseWriter, r *http.Request) {
		handleGzRoot(w, r)
	}))

	// Hello endpoint
	mux.HandleFunc("/hello", authenticatedHandler(func(w http.ResponseWriter, r *http.Request) {
		handleGzHello(w, r)
	}))

	// Status endpoint
	mux.HandleFunc("/status", authenticatedHandler(func(w http.ResponseWriter, r *http.Request) {
		handleGzStatus(w, r)
	}))

	// Health check endpoint
	mux.HandleFunc("/health", authenticatedHandler(func(w http.ResponseWriter, r *http.Request) {
		handleGzHealth(w, r)
	}))

	// API endpoint for grading data (example of what this server might do)
	mux.HandleFunc("/api/grade", authenticatedHandler(func(w http.ResponseWriter, r *http.Request) {
		handleGradeAPI(w, r)
	}))

	log.Printf("Server 'gz' starting on port %s", SERVER_GZ_PORT)
	log.Printf("Server 'gz' only accepts requests with valid internal secret header")

	if err := http.ListenAndServe(SERVER_GZ_PORT, mux); err != nil {
		log.Fatal("GZ Server failed to start:", err)
	}
}

func handleGzRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	response := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Server GZ</title>
</head>
<body>
    <h1>Welcome to Server GZ</h1>
    <p><strong>⚠️ Internal Server Only ⚠️</strong></p>
    <p>This server only accepts authenticated requests from server 's'.</p>

    <p><strong>Method:</strong> %s</p>
    <p><strong>Path:</strong> %s</p>
    <p><strong>Host:</strong> %s</p>
    <p><strong>Remote Address:</strong> %s</p>

    <h2>Available Endpoints:</h2>
    <ul>
        <li><a href="/gz">/ - This page</a></li>
        <li><a href="/gz/hello">/hello - Greeting endpoint</a></li>
        <li><a href="/gz/status">/status - Server status</a></li>
        <li><a href="/gz/health">/health - Health check</a></li>
        <li><a href="/gz/api/grade">/api/grade - Grade processing API</a></li>
    </ul>

    <h2>Request Headers:</h2>
    <ul>`, r.Method, r.URL.Path, r.Host, r.RemoteAddr)

	for name, values := range r.Header {
		for _, value := range values {
			response += fmt.Sprintf("<li><strong>%s:</strong> %s</li>", name, value)
		}
	}

	response += `
    </ul>
</body>
</html>`

	io.WriteString(w, response)
}

func handleGzHello(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"message":   "Hello from GZ server!",
		"server":    "gz",
		"timestamp": time.Now().Unix(),
		"method":    r.Method,
		"path":      r.URL.Path,
	}

	json.NewEncoder(w).Encode(response)
}

func handleGzStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"status":    "running",
		"server":    "gz",
		"uptime":    "unknown", // In a real implementation, track actual uptime
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
	}

	json.NewEncoder(w).Encode(response)
}

func handleGzHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"healthy":   true,
		"server":    "gz",
		"timestamp": time.Now().Unix(),
	}

	json.NewEncoder(w).Encode(response)
}

func handleGradeAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		// Return mock grading data
		response := map[string]interface{}{
			"grades": []map[string]interface{}{
				{"student_id": "101295001", "name": "Test Student1", "grade": 85, "max_points": 100},
				{"student_id": "101295002", "name": "Test Student2", "grade": 92, "max_points": 100},
			},
			"total_submissions": 2,
			"average_grade":     88.5,
			"timestamp":         time.Now().Unix(),
		}
		json.NewEncoder(w).Encode(response)

	case "POST":
		// Process a new grade submission
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusBadRequest)
			return
		}

		response := map[string]interface{}{
			"message":   "Grade processed successfully",
			"server":    "gz",
			"received":  string(body),
			"timestamp": time.Now().Unix(),
			"status":    "processed",
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
