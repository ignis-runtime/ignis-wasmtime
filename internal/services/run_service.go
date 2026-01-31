package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bytecodealliance/wasmtime-go/v41"
	"github.com/google/uuid"
	"github.com/ignis-runtime/ignis-wasmtime/internal/cache"
	"github.com/ignis-runtime/ignis-wasmtime/internal/runtime"
	"github.com/ignis-runtime/ignis-wasmtime/internal/runtime/js"
	"github.com/ignis-runtime/ignis-wasmtime/internal/runtime/wasm"
	"github.com/ignis-runtime/ignis-wasmtime/types"
	"google.golang.org/protobuf/proto"
)

// RunService defines the interface for running deployments
type RunService interface {
	ExecuteDeployment(ctx context.Context, id uuid.UUID, request *types.FDRequest) (*types.FDResponse, error)
}

// runService implements the RunService interface
type runService struct {
	cache             *cache.RedisCache
	deploymentService DeploymentService
}

// NewRunService creates a new RunService instance
func NewRunService(cache *cache.RedisCache, deploymentService DeploymentService) RunService {
	return &runService{
		cache:             cache,
		deploymentService: deploymentService,
	}
}

// ExecuteDeployment executes a deployment by UUID with the given HTTP request context
func (s *runService) ExecuteDeployment(ctx context.Context, id uuid.UUID, request *types.FDRequest) (*types.FDResponse, error) {
	deployment, err := s.deploymentService.GetDeploymentByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	var config runtime.RuntimeConfig

	switch strings.ToLower(deployment.RuntimeType) {
	case "js":
		// 1. Get/Compile QuickJS Engine
		engineBytes, err := s.getSerializedModule(ctx, "qjs-serialized", func() ([]byte, error) {
			return js.QJSWasm, nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get JS engine: %w", err)
		}

		// 2. Get JS Script Source (Directly from cache or DB)
		var jsFile []byte
		if cached, exists := s.cache.Get(ctx, deployment.Hash); exists {
			jsFile = cached.Data
		} else {
			jsFile, err = s.deploymentService.GetDeploymentFileContentByHash(ctx, deployment.Hash)
			if err != nil {
				return nil, fmt.Errorf("failed to get JS file content: %w", err)
			}
			_ = s.cache.Set(ctx, deployment.Hash, &types.Module{Hash: deployment.Hash, Data: jsFile}, time.Hour*2)
		}

		config = js.NewRuntimeConfig(id).WithSerializedModule(engineBytes).WithJSFile(jsFile)

	case "wasm":
		// Get/Compile the specific WASM deployment
		moduleBytes, err := s.getSerializedModule(ctx, deployment.Hash, func() ([]byte, error) {
			return s.deploymentService.GetDeploymentFileContentByUUID(ctx, id)
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get WASM module: %w", err)
		}
		config = wasm.NewRuntimeConfig(id).WithSerializedModule(moduleBytes)

	default:
		return nil, fmt.Errorf("invalid runtime type: %s", deployment.RuntimeType)
	}

	// Lifecycle execution
	rt, err := config.Instantiate()
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate runtime: %w", err)
	}
	defer rt.Close(ctx)

	// Create FDRequest from HTTP request context
	fdRequest := &types.FDRequest{
		Method:        request.Method,
		Body:          request.Body,
		ContentLength: request.ContentLength,
		Host:          request.Host,
		RemoteAddr:    request.RemoteAddr,
		RequestUri:    request.RequestUri,
		Header:        request.Header,
	}

	reqBytes, err := proto.Marshal(fdRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	respBytes, err := rt.Execute(ctx, reqBytes)
	if err != nil {
		return nil, fmt.Errorf("runtime execution failed: %w", err)
	}

	var fdResponse types.FDResponse
	err = proto.Unmarshal(respBytes, &fdResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &fdResponse, nil
}

// getSerializedModule abstracts the "Check Cache -> Compile -> Store Cache" workflow
func (s *runService) getSerializedModule(ctx context.Context, cacheKey string, loader func() ([]byte, error)) ([]byte, error) {
	if cached, exists := s.cache.Get(ctx, cacheKey); exists {
		return cached.Data, nil
	}

	// Cache miss: Load raw bytes
	raw, err := loader()
	if err != nil {
		return nil, fmt.Errorf("loader failed: %w", err)
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
