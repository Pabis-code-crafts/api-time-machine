package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
)

const targetServer = "https://jsonplaceholder.typicode.com"

func proxyHandler(w http.ResponseWriter, r *http.Request) {

	// Read request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Restore body so it can be used again
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Build new request to real server
	req, err := http.NewRequest(
		r.Method,
		targetServer+r.URL.Path,
		bytes.NewBuffer(bodyBytes),
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Copy headers
	req.Header = r.Header.Clone()

	// HTTP client
	client := &http.Client{}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	log.Println("Captured Request:")
	log.Println("Method:", r.Method)
	log.Println("Path:", r.URL.Path)
	log.Println("Body:", string(bodyBytes))
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Write response status
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	io.Copy(w, resp.Body)
}

func main() {

	http.HandleFunc("/", proxyHandler)

	log.Println("Proxy server running on :8081")

	log.Fatal(http.ListenAndServe(":8081", nil))
}
