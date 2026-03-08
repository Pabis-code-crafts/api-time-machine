package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"strconv"
)

const targetServer = "https://jsonplaceholder.typicode.com"

type CapturedRequest struct {
	ID      int
	Method  string
	Path    string
	Headers map[string][]string
	Body    string
}

var requests []CapturedRequest
var requestID = 1

func proxyHandler(w http.ResponseWriter, r *http.Request) {

	// Read request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Restore body so it can be used again
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Capture request
	captured := CapturedRequest{
		ID:      requestID,
		Method:  r.Method,
		Path:    r.URL.Path,
		Headers: r.Header.Clone(),
		Body:    string(bodyBytes),
	}

	requests = append(requests, captured)
	requestID++

	log.Println("Captured Request:", captured.ID, captured.Method, captured.Path)

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

	client := &http.Client{}

	// Send request
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

	// Write response status
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	io.Copy(w, resp.Body)
}

func listRequestsHandler(w http.ResponseWriter, r *http.Request) {

	for _, req := range requests {

		line := "ID: " + strconv.Itoa(req.ID) +
			" | " + req.Method +
			" | " + req.Path

		w.Write([]byte(line + "\n"))
	}

}
func replayHandler(w http.ResponseWriter, r *http.Request) {

	idStr := r.URL.Path[len("/replay/"):]
	id, err := strconv.Atoi(idStr)

	if err != nil {
		http.Error(w, "Invalid request ID", http.StatusBadRequest)
		return
	}

	var storedRequest *CapturedRequest

	for i := range requests {
		if requests[i].ID == id {
			storedRequest = &requests[i]
			break
		}
	}

	if storedRequest == nil {
		http.Error(w, "Request not found", http.StatusNotFound)
		return
	}

	log.Println("Replaying request:", storedRequest.ID)

	req, err := http.NewRequest(
		storedRequest.Method,
		targetServer+storedRequest.Path,
		bytes.NewBuffer([]byte(storedRequest.Body)),
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.Header = storedRequest.Headers

	client := &http.Client{}

	resp, err := client.Do(req)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)

	io.Copy(w, resp.Body)
}

func main() {

	http.HandleFunc("/debug/requests", listRequestsHandler)
	http.HandleFunc("/replay/", replayHandler)
	http.HandleFunc("/", proxyHandler)

	log.Println("Proxy server running on :8081")

	log.Fatal(http.ListenAndServe(":8081", nil))
}
