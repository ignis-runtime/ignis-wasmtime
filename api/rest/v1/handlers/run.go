package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bytecodealliance/wasmtime-go/v41"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	v1 "github.com/ignis-runtime/ignis-wasmtime/api/rest/v1"
	"github.com/ignis-runtime/ignis-wasmtime/internal/cache"
	"github.com/ignis-runtime/ignis-wasmtime/internal/runtime"
	"github.com/ignis-runtime/ignis-wasmtime/internal/runtime/js"
	"github.com/ignis-runtime/ignis-wasmtime/internal/runtime/wasm"
	"github.com/ignis-runtime/ignis-wasmtime/internal/services"
	"github.com/ignis-runtime/ignis-wasmtime/types"
	"google.golang.org/protobuf/proto"
)

type RunHandlers struct {
	cache             *cache.RedisCache
	deploymentService services.DeploymentService
}

func NewRunHandlers(cache *cache.RedisCache, deploymentService services.DeploymentService) *RunHandlers {
	return &RunHandlers{cache: cache, deploymentService: deploymentService}
}

// getSerializedModule abstracts the "Check Cache -> Compile -> Store Cache" workflow
func (s *RunHandlers) getSerializedModule(ctx context.Context, cacheKey string, loader func() ([]byte, error)) ([]byte, error) {
	if cached, exists := s.cache.Get(ctx, cacheKey); exists {
		return cached.Data, nil
	}

	// Cache miss: Load raw bytes
	raw, err := loader()
	if err != nil {
		return nil, err
	}

	// Compile and Serialize
	engine := wasmtime.NewEngine()
	module, err := wasmtime.NewModule(engine, raw)
	if err != nil {
		return nil, fmt.Errorf("compilation failed: %w", err)
	}

	serialized, err := module.Serialize()
	if err != nil {
		return nil, fmt.Errorf("serialization failed: %w", err)
	}

	// Async cache update to avoid blocking
	_ = s.cache.Set(ctx, cacheKey, &types.Module{Hash: cacheKey, Data: serialized}, time.Hour*2)
	return serialized, nil
}

func (s *RunHandlers) HandleWasmRequest(c *gin.Context) error {
	rawId := c.Param("uuid")
	id, err := uuid.Parse(rawId)
	if err != nil {
		return v1.APIError{Code: http.StatusBadRequest, Err: "invalid UUID"}
	}

	deployment, err := s.deploymentService.GetDeploymentByID(c.Request.Context(), id)
	if err != nil {
		return v1.APIError{Code: http.StatusInternalServerError, Err: err.Error()}
	}

	var config runtime.RuntimeConfig

	switch strings.ToLower(deployment.RuntimeType) {
	case "js":
		// 1. Get/Compile QuickJS Engine
		engineBytes, err := s.getSerializedModule(c, "qjs-serialized", func() ([]byte, error) {
			return js.QJSWasm, nil
		})
		if err != nil {
			return v1.APIError{Code: http.StatusInternalServerError, Err: err.Error()}
		}

		// 2. Get JS Script Source (Directly from cache or DB)
		var jsFile []byte
		if cached, exists := s.cache.Get(c, deployment.Hash); exists {
			jsFile = cached.Data
		} else {
			jsFile, err = s.deploymentService.GetDeploymentFileContentByHash(c, deployment.Hash)
			if err != nil {
				return v1.APIError{Code: http.StatusInternalServerError, Err: err.Error()}
			}
			_ = s.cache.Set(c, deployment.Hash, &types.Module{Hash: deployment.Hash, Data: jsFile}, time.Hour*2)
		}

		config = js.NewRuntimeConfig(id).WithSerializedModule(engineBytes).WithJSFile(jsFile)

	case "wasm":
		// Get/Compile the specific WASM deployment
		moduleBytes, err := s.getSerializedModule(c, deployment.Hash, func() ([]byte, error) {
			return s.deploymentService.GetDeploymentFileContentByUUID(c, id)
		})
		if err != nil {
			return v1.APIError{Code: http.StatusInternalServerError, Err: err.Error()}
		}
		config = wasm.NewRuntimeConfig(id).WithSerializedModule(moduleBytes)

	default:
		return v1.APIError{Code: http.StatusBadRequest, Err: "invalid runtime type"}
	}

	// Lifecycle execution
	rt, err := config.Instantiate()
	if err != nil {
		return v1.APIError{Code: http.StatusInternalServerError, Err: err.Error()}
	}
	defer rt.Close(c.Request.Context())

	fdResponse, err := s.executeRuntimeCycle(c, rt)
	if err != nil {
		return v1.APIError{Code: http.StatusInternalServerError, Err: err.Error()}
	}

	// Map FDResponse back to Gin Response
	return s.writeResponse(c, fdResponse)
}

func (s *RunHandlers) writeResponse(c *gin.Context, res *types.FDResponse) error {
	status := http.StatusOK
	if res.StatusCode != 0 {
		status = int(res.StatusCode)
	}

	for key, values := range res.Header {
		for _, value := range values.Fields {
			c.Writer.Header().Add(key, value)
		}
	}

	c.Data(status, c.Writer.Header().Get("Content-Type"), res.Body)
	return nil
}

func (s *RunHandlers) executeRuntimeCycle(c *gin.Context, rt runtime.Runtime) (*types.FDResponse, error) {
	path := c.Param("path")
	if path == "" {
		path = "/"
	}

	reqBody, _ := io.ReadAll(c.Request.Body)
	fdRequest := &types.FDRequest{
		Method:        c.Request.Method,
		Body:          reqBody,
		ContentLength: c.Request.ContentLength,
		Host:          c.Request.Host,
		RemoteAddr:    c.Request.RemoteAddr,
		RequestUri:    path,
		Header:        make(map[string]*types.HeaderFields),
	}

	for k, v := range c.Request.Header {
		fdRequest.Header[k] = &types.HeaderFields{Fields: v}
	}

	reqBytes, _ := proto.Marshal(fdRequest)
	respBytes, err := rt.Execute(c.Request.Context(), reqBytes)
	if err != nil {
		return nil, err
	}

	var fdResponse types.FDResponse
	return &fdResponse, proto.Unmarshal(respBytes, &fdResponse)
}
