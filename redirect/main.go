package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://comp3007-f25.scs.carleton.ca", http.StatusPermanentRedirect)
}

func main() {
	http.HandleFunc("/", redirectHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Starting redirect server on port %s\n", port)
	fmt.Println("All requests will be redirected to: https://comp3007-f25.scs.carleton.ca")

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
