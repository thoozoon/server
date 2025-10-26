package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

const (
	SERVER_S_PORT  = ":8080"
	SERVER_GZ_PORT = ":8081"
	GZ_PREFIX      = "/gz"
	// Secret header to ensure requests to gz come from s
	INTERNAL_SECRET_HEADER = "X-Internal-Secret"
	INTERNAL_SECRET_VALUE  = "s-to-gz-internal-token-12345"
)

func main() {
	// Create reverse proxy to gz server
	gzURL, err := url.Parse("http://localhost" + SERVER_GZ_PORT)
	if err != nil {
		log.Fatal("Failed to parse gz server URL:", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(gzURL)

	// Modify the proxy to add authentication header
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		// Add secret header to authenticate with gz server
		req.Header.Set(INTERNAL_SECRET_HEADER, INTERNAL_SECRET_VALUE)
		// Remove the /gz prefix from the path when forwarding
		req.URL.Path = strings.TrimPrefix(req.URL.Path, GZ_PREFIX)
		if req.URL.Path == "" {
			req.URL.Path = "/"
		}
	}

	mux := http.NewServeMux()

	// Forward all /gz requests to gz server
	mux.HandleFunc("/gz/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Forwarding request to gz server: %s %s", r.Method, r.URL.Path)
		proxy.ServeHTTP(w, r)
	})

	// Handle all other requests directly
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, GZ_PREFIX) {
			// This handles /gz without trailing slash
			proxy.ServeHTTP(w, r)
			return
		}

		log.Printf("Handling request directly: %s %s", r.Method, r.URL.Path)
		handleDirectRequest(w, r)
	})

	log.Printf("Server 's' starting on port %s", SERVER_S_PORT)
	log.Printf("Forwarding %s* requests to gz server on port %s", GZ_PREFIX, SERVER_GZ_PORT)

	if err := http.ListenAndServe(SERVER_S_PORT, mux); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

func handleDirectRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	response := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Server S</title>
</head>
<body>
    <h1>Welcome to Server S</h1>
    <p><strong>Method:</strong> %s</p>
    <p><strong>Path:</strong> %s</p>
    <p><strong>Host:</strong> %s</p>

    <h2>Available Endpoints:</h2>
    <ul>
        <li><a href="/">/ - This page (handled by server s)</a></li>
        <li><a href="/health">/ health - Health check (handled by server s)</a></li>
        <li><a href="/gz">/gz - Forwarded to gz server</a></li>
        <li><a href="/gz/hello">/gz/hello - Forwarded to gz server</a></li>
        <li><a href="/gz/status">/gz/status - Forwarded to gz server</a></li>
    </ul>

    <h2>Request Headers:</h2>
    <ul>`, r.Method, r.URL.Path, r.Host)

	for name, values := range r.Header {
		for _, value := range values {
			response += fmt.Sprintf("<li><strong>%s:</strong> %s</li>", name, value)
		}
	}

	response += `
    </ul>
</body>
</html>`

	// Special handling for health check
	if r.URL.Path == "/health" {
		w.Header().Set("Content-Type", "application/json")
		response = `{"status":"ok","server":"s","timestamp":"` + fmt.Sprintf("%d",
			getCurrentTimestamp()) + `"}`
	}

	io.WriteString(w, response)
}

func getCurrentTimestamp() int64 {
	return 1234567890 // Simplified timestamp for demo
}
