package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	v1 "github.com/ignis-runtime/ignis-wasmtime/api/rest/v1"
	"github.com/ignis-runtime/ignis-wasmtime/internal/cache"
	"github.com/ignis-runtime/ignis-wasmtime/internal/repository"
	"github.com/ignis-runtime/ignis-wasmtime/internal/runtime"
	"github.com/ignis-runtime/ignis-wasmtime/internal/runtime/js"
	"github.com/ignis-runtime/ignis-wasmtime/internal/runtime/wasm"
	"github.com/ignis-runtime/ignis-wasmtime/internal/storage"
	"github.com/ignis-runtime/ignis-wasmtime/types"
	"google.golang.org/protobuf/proto"
)

type Server struct {
	Addr        string
	Engine      *gin.Engine
	Cache       *cache.RedisCache
	RuntimeRepo repository.RuntimeRepository
	S3Storage   storage.S3Storage
}

// RegisterRuntime is now handled by the repository layer
// This method can be removed or kept for compatibility with existing code
func (s *Server) RegisterRuntime(runtimeID uuid.UUID, runtimeConfig runtime.RuntimeConfig) {
	// This method is deprecated. Use the repository directly instead.
}

func (s *Server) FindRuntimeByHash(targetHash string) (uuid.UUID, bool) {
	runtime, err := s.RuntimeRepo.FindByHash(targetHash)
	if err != nil {
		return uuid.Nil, false
	}
	return runtime.ID, true
}

func (s *Server) UnregisterRuntime(runtimeID uuid.UUID) {
	// Delete from the database
	_ = s.RuntimeRepo.Delete(runtimeID)
}

func NewServer(addr string, cache *cache.RedisCache, runtimeRepo repository.RuntimeRepository, s3Storage storage.S3Storage) *Server {
	gin.SetMode(gin.ReleaseMode)
	return &Server{
		Addr:        addr,
		Engine:      gin.Default(),
		Cache:       cache,
		RuntimeRepo: runtimeRepo,
		S3Storage:   s3Storage,
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

	// Get the runtime record from the database
	runtimeRecord, err := s.RuntimeRepo.FindByID(id)
	if err != nil {
		return v1.APIError{
			Code: http.StatusNotFound,
			Err:  "specified runtime ID not found",
		}
	}

	// Get the runtime file from local storage (or cache)
	runtimeData, err := s.getRuntimeData(runtimeRecord.S3FilePath)
	if err != nil {
		return v1.APIError{
			Code: http.StatusInternalServerError,
			Err:  fmt.Sprintf("failed to get runtime data: %v", err),
		}
	}

	// Create the appropriate runtime config based on runtime type
	// Still need to provide cache for internal operations (like QJS module caching)
	var runtimeConfig runtime.RuntimeConfig
	switch runtimeRecord.RuntimeType {
	case "js":
		runtimeConfig = js.NewRuntimeConfig(id, runtimeData).WithCache(s.Cache)
	case "wasm":
		// For now, we'll create a basic config without args/preopened dir
		// In a real implementation, you might want to store these in the DB too
		runtimeConfig = wasm.NewRuntimeConfig(id, runtimeData).WithCache(s.Cache)
	default:
		return v1.APIError{
			Code: http.StatusBadRequest,
			Err:  "invalid runtime type",
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

// getRuntimeData retrieves runtime data from S3 storage
func (s *Server) getRuntimeData(filePath string) ([]byte, error) {
	// filePath is the S3 key
	data, err := s.S3Storage.DownloadFile(context.Background(), "", filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to download file from S3: %w", err)
	}
	return data, nil
}
