package server

import (
	"fmt"
	"io"
	"log"
	"net/http" // Still needed for http.ResponseWriter and http.Request types

	"github.com/gin-gonic/gin" // Gin import
	"github.com/google/uuid"

	"github.com/ignis-runtime/ignis-wasmtime/types"

	"github.com/ignis-runtime/ignis-wasmtime/internal/runtime"

	"google.golang.org/protobuf/proto" // For protobuf serialization
)

type registeredRuntimesMap = map[uuid.UUID]runtime.Runtime

var (
	registeredRuntimes = make(registeredRuntimesMap)
)

func RegisterRuntime(id uuid.UUID, rt runtime.Runtime) {
	registeredRuntimes[id] = rt
}

// Server holds the HTTP server and a reference to the runtime factory.
type Server struct{}

// NewServer creates a new HTTP server instance with a runtime factory.
func NewServer() *Server {
	return &Server{}
}

// HandleWasmRequest handles incoming HTTP requests, serializes them to protobuf,
// passes them to the appropriate WASM module via the runtime factory, and returns
// the runtime's response as an HTTP response.
func (s *Server) HandleWasmRequest(c *gin.Context) {
	// Retrieve the parsed UUID from the context (validated in the route handler)
	runtimeIdInterface, exists := c.Get("parsed_uuid")
	if !exists {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("runtime ID not found in context"))
		return
	}

	runtimeIdUuid, ok := runtimeIdInterface.(uuid.UUID)
	if !ok {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("runtime ID is not a valid UUID"))
		return
	}

	// Get the remaining path after the UUID
	strippedPath := c.Param("path")
	if strippedPath == "" {
		strippedPath = "/"
	}

	log.Printf("Incoming path: %s, Runtime ID: %s, Forwarded path: %s", c.Request.URL.Path, runtimeIdUuid.String(), strippedPath)

	// Get the appropriate runtime from the factory
	runtime, exists := registeredRuntimes[runtimeIdUuid]
	if !exists {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("failed to get runtime"))
		return
	}

	// 1. Prepare the FDRequest from the incoming HTTP request
	r := c.Request

	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("Failed to read request body: %v", err))
		return
	}

	fdRequest := &types.FDRequest{
		Method:        r.Method,
		Body:          reqBody,
		ContentLength: r.ContentLength,
		Host:          r.Host,
		RemoteAddr:    r.RemoteAddr,
		RequestUri:    strippedPath, // Use the stripped path without the UUID prefix
		Pattern:       strippedPath, // Use the stripped path without the UUID prefix
	}

	// Populate headers
	fdRequest.Header = make(map[string]*types.HeaderFields)
	for key, values := range r.Header {
		fdRequest.Header[key] = &types.HeaderFields{Fields: values}
	}

	// Populate Transfer-Encoding if present
	if len(r.TransferEncoding) > 0 {
		fdRequest.TransferEncoding = &types.StringSlice{Fields: r.TransferEncoding}
	}

	// 2. Serialize FDRequest to protobuf and pass to runtime
	reqBytes, err := proto.Marshal(fdRequest)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("Failed to marshal FDRequest: %v", err))
		return
	}

	// Execute the runtime
	respBytes, err := runtime.Execute(c.Request.Context(), reqBytes)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("Runtime execution error: %v", err))
		return
	}

	var fdResponse types.FDResponse
	if err := proto.Unmarshal(respBytes, &fdResponse); err != nil {
		// If response is not a valid protobuf, treat as plain text
		log.Printf("Failed to unmarshal FDResponse from runtime, treating as plain text: %v", err)
		c.Data(http.StatusOK, "text/plain", respBytes)
		return
	}

	// Set HTTP status code
	if fdResponse.StatusCode != 0 {
		c.Status(int(fdResponse.StatusCode))
	} else {
		c.Status(http.StatusOK) // Default to 200 OK
	}

	// Set HTTP headers
	for key, values := range fdResponse.Header {
		for _, value := range values.Fields {
			c.Writer.Header().Add(key, value)
		}
	}

	// Write response body
	if len(fdResponse.Body) > 0 {
		c.Writer.Write(fdResponse.Body)
	}
}
