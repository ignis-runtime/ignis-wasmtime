//go:build !wasip1

package http

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/bytecodealliance/wasmtime-go/v41"
)

// HostRequest is the structure received from the guest.
type HostRequest struct {
	Method  string      `json:"method"`
	URL     string      `json:"url"`
	Headers http.Header `json:"headers"`
	Body    []byte      `json:"body"`
}

// HostResponse is the structure sent back to the guest.
type HostResponse struct {
	StatusCode int         `json:"status_code"`
	Headers    http.Header `json:"headers"`
	Body       []byte      `json:"body"`
}

// Link attaches the SDK's full HTTP host functions to the Wasmtime linker.
func Link(store *wasmtime.Store, linker *wasmtime.Linker) error {
	return linker.DefineFunc(store, "env", "host_http_request", func(caller *wasmtime.Caller, reqPtr, reqLen, respPtr, respLen int32) int32 {
		memory := caller.GetExport("memory").Memory()
		if memory == nil {
			log.Println("host_http_request: failed to get memory export")
			return 0
		}

		// Read the request JSON from guest memory.
		reqBytes := memory.UnsafeData(store)[reqPtr : reqPtr+reqLen]

		// Unmarshal the request.
		var hostReq HostRequest
		if err := json.Unmarshal(reqBytes, &hostReq); err != nil {
			log.Printf("host_http_request: failed to unmarshal request JSON: %v\n", err)
			return 0
		}

		// Create the request on the host.
		req, err := http.NewRequest(hostReq.Method, hostReq.URL, bytes.NewReader(hostReq.Body))
		if err != nil {
			log.Printf("host_http_request: failed to create request: %v\n", err)
			return 0
		}
		req.Header = hostReq.Headers

		// Make the request.
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("host_http_request: failed to execute request: %v\n", err)
			return 0
		}
		defer resp.Body.Close()

		// Read the response body.
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("host_http_request: failed to read response body: %v\n", err)
			return 0
		}

		// Create the response structure.
		hostResp := HostResponse{
			StatusCode: resp.StatusCode,
			Headers:    resp.Header,
			Body:       body,
		}

		// Marshal the response to JSON.
		respBytes, err := json.Marshal(hostResp)
		if err != nil {
			log.Printf("host_http_request: failed to marshal response JSON: %v\n", err)
			return 0
		}

		// Check if the result buffer is large enough.
		if int32(len(respBytes)) > respLen {
			log.Printf("host_http_request: result buffer too small. Needed %d, have %d\n", len(respBytes), respLen)
			return 0
		}

		// Write the JSON response back into the guest's memory buffer.
		copy(memory.UnsafeData(store)[respPtr:respPtr+int32(len(respBytes))], respBytes)

		// Return the length of the JSON response written.
		return int32(len(respBytes))
	})
}
