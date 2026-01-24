package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	_ "ignis-wasmtime/sdk/http"
)

func main() {
	req, err := http.NewRequest("GET", "https://icanhazdadjoke.com/", nil)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Standard Go Program (Running in WASM)")
	client := http.Client{
		Transport: http.DefaultTransport,
		Timeout:   30,
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to get joke: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
	}

	// Unmarshal and pretty-print the JSON.
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		log.Fatalf("Error decoding JSON: %v. Body was: %s", err, body)
	}

	prettyJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatalf("Error marshalling JSON: %v", err)
	}

	fmt.Println(string(prettyJSON))
}
