package server

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	v1 "github.com/ignis-runtime/ignis-wasmtime/api/rest/v1"
	"github.com/ignis-runtime/ignis-wasmtime/internal/cache"
	"github.com/ignis-runtime/ignis-wasmtime/internal/runtime"
	"github.com/ignis-runtime/ignis-wasmtime/types"
	"google.golang.org/protobuf/proto"
)

type registeredRuntimesMap map[uuid.UUID]runtime.RuntimeConfig

type Server struct {
	Addr               string
	Engine             *gin.Engine
	Cache              *cache.RedisCache
	registeredRuntimes registeredRuntimesMap
	runtimeHashLookup  map[string]uuid.UUID  // hash -> UUID lookup for O(1) retrieval
}

func (s *Server) RegisterRuntime(runtimeID uuid.UUID, runtimeConfig runtime.RuntimeConfig) {
	s.registeredRuntimes[runtimeID] = runtimeConfig
	s.runtimeHashLookup[runtimeConfig.GetHash()] = runtimeID
}

func (s *Server) FindRuntimeByHash(targetHash string) (uuid.UUID, bool) {
	id, exists := s.runtimeHashLookup[targetHash]
	return id, exists
}

func (s *Server) UnregisterRuntime(runtimeID uuid.UUID) {
	if config, exists := s.registeredRuntimes[runtimeID]; exists {
		// Remove the corresponding hash from the lookup map
		delete(s.runtimeHashLookup, config.GetHash())
		// Remove the runtime from the main map
		delete(s.registeredRuntimes, runtimeID)
	}
}

func NewServer(addr string, cache *cache.RedisCache) *Server {
	gin.SetMode(gin.ReleaseMode)
	return &Server{
		Addr:                addr,
		Engine:              gin.Default(),
		Cache:               cache,
		registeredRuntimes:  make(registeredRuntimesMap),
		runtimeHashLookup:   make(map[string]uuid.UUID),
	}
}
func (s *Server) Run() error {
	return s.Engine.Run(s.Addr)
}

func (s *Server) HandleWasmRequest(c *gin.Context) error {
	rawId, exists := c.Params.Get("uuid")
	if !exists {
		return v1.APIError{
			Code: http.StatusBadRequest,
			Err:  "rt ID not found in context",
		}
	}
	id := uuid.MustParse(rawId)

	// Get the appropriate rt from the factory
	runtimeConfig, exists := s.registeredRuntimes[id]
	if !exists {
		return v1.APIError{
			Code: http.StatusNotFound,
			Err:  "specified runtime ID not found",
		}
	}

	rt, err := runtimeConfig.Instantiate()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	defer func() {
		err := rt.Close(c.Request.Context())
		if err != nil {
			log.Printf("error closing runtime: %v", err)
		}
	}()

	// Call the combined function
	fdResponse, err := s.executeRuntimeCycle(c, rt)
	if err != nil {
		return v1.APIError{
			Code: http.StatusInternalServerError,
			Err:  err.Error(),
		}
	}

	// Set HTTP status code
	if fdResponse.StatusCode != 0 {
		c.Status(int(fdResponse.StatusCode))
	} else {
		c.Status(http.StatusOK)
	}

	// Set HTTP headers and write body
	for key, values := range fdResponse.Header {
		for _, value := range values.Fields {
			c.Writer.Header().Add(key, value)
		}
	}

	if len(fdResponse.Body) > 0 {
		if _, err := c.Writer.Write(fdResponse.Body); err != nil {
			return v1.APIError{
				Code: http.StatusInternalServerError,
				Err:  err.Error(),
			}
		}
	}
	return nil
}

// ExecuteRuntimeCycle orchestrates the conversion of the Gin request to Protobuf,
// executes it within the provided runtime, and unmarshal the response.
func (s *Server) executeRuntimeCycle(c *gin.Context, rt runtime.Runtime) (*types.FDResponse, error) {

	strippedPath := c.Param("path")
	if strippedPath == "" {
		strippedPath = "/"
	}

	// 1. Prepare the FDRequest from the incoming Gin/HTTP request
	reqBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %v", err)
	}

	fdRequest := &types.FDRequest{
		Method:        c.Request.Method,
		Body:          reqBody,
		ContentLength: c.Request.ContentLength,
		Host:          c.Request.Host,
		RemoteAddr:    c.Request.RemoteAddr,
		RequestUri:    strippedPath,
		Pattern:       strippedPath,
		Header:        make(map[string]*types.HeaderFields),
	}

	// Populate headers and Transfer-Encoding
	for key, values := range c.Request.Header {
		fdRequest.Header[key] = &types.HeaderFields{Fields: values}
	}
	if len(c.Request.TransferEncoding) > 0 {
		fdRequest.TransferEncoding = &types.StringSlice{Fields: c.Request.TransferEncoding}
	}

	// 2. Serialize and Execute
	reqBytes, err := proto.Marshal(fdRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal FDRequest: %v", err)
	}

	respBytes, err := rt.Execute(c.Request.Context(), reqBytes)
	if err != nil {
		return nil, fmt.Errorf("runtime execution error: %v", err)
	}

	// 3. Unmarshal the result into an FDResponse
	var fdResponse types.FDResponse
	if err := proto.Unmarshal(respBytes, &fdResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal FDResponse: %v", err)
	}

	return &fdResponse, nil
}
