package main

import (
	"io"
	"log"
	"net/http"
)

const targetServer = "https://jsonplaceholder.typicode.com"

func proxyHandler(w http.ResponseWriter, r *http.Request) {

	// Build new request to real server
	req, err := http.NewRequest(
		r.Method,
		targetServer+r.URL.Path,
		r.Body,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.Header = r.Header

	client := &http.Client{}

	resp, err := client.Do(req)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)

	io.Copy(w, resp.Body)
}

func main() {

	http.HandleFunc("/", proxyHandler)

	log.Println("Proxy server running on :8081")

	log.Fatal(http.ListenAndServe(":8081", nil))
}
