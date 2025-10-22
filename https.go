package main

import (
	"log"
	"net/http"

	"golang.org/x/crypto/acme/autocert"
)

func main_secondary() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, HTTPS!"))
	})

	// Autocert manager handles cert acquisition/renewal
	m := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		Cache:      autocert.DirCache("cert-cache"), // persist certs
		HostPolicy: autocert.HostWhitelist("example.com", "www.example.com"),
	}

	// HTTP (port 80) serves ACME http-01 challenges + redirects
	go func() {
		log.Println(http.ListenAndServe(":80", m.HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			target := "https://" + r.Host + r.URL.RequestURI()
			http.Redirect(w, r, target, http.StatusMovedPermanently)
		}))))
	}()

	// HTTPS (port 443) with TLS from autocert
	srv := &http.Server{
		Addr:      ":443",
		Handler:   mux,
		TLSConfig: m.TLSConfig(),
	}

	log.Fatal(srv.ListenAndServeTLS("", "")) // certs come from TLSConfig
}
