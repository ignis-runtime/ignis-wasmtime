package services

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/ignis-runtime/ignis-wasmtime/api/rest/server"
	"github.com/ignis-runtime/ignis-wasmtime/api/rest/v1/schemas"
	"github.com/ignis-runtime/ignis-wasmtime/internal/config"
	"github.com/ignis-runtime/ignis-wasmtime/internal/models"
	"github.com/ignis-runtime/ignis-wasmtime/internal/storage"
	"github.com/ignis-runtime/ignis-wasmtime/internal/utils"
)

// DeployResult represents the result of a deployment operation
type DeployResult struct {
	Response   *schemas.DeployResponse
	IsExisting bool // True if the deployment was an existing runtime with the same hash
}

// DeployService defines the interface for deployment operations
type DeployService interface {
	Deploy(req schemas.DeployRequest) (*DeployResult, error)
}

// deployService implements the DeployService interface
type deployService struct {
	server    *server.Server
	config    *config.Config
	s3Storage storage.S3Storage
}

// NewDeployService creates a new instance of DeployService
func NewDeployService(server *server.Server, config *config.Config) DeployService {
	return &deployService{
		server:    server,
		config:    config,
		s3Storage: server.S3Storage,
	}
}

// Deploy handles the deployment logic
func (ds *deployService) Deploy(req schemas.DeployRequest) (*DeployResult, error) {
	// Validate that only one file is provided
	if req.File == nil {
		return nil, fmt.Errorf("no file provided")
	}

	// Validate file extension matches runtime type
	expectedExt := getFileExtensionForRuntimeType(req.RuntimeType)
	actualExt := filepath.Ext(req.File.Filename)
	if actualExt != expectedExt {
		return nil, fmt.Errorf("file extension mismatch: expected %s for %s runtime, got %s", expectedExt, req.RuntimeType, actualExt)
	}

	// Open the uploaded file
	file, err := req.File.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read the file data
	filedata, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	// Calculate the hash based on the file data
	targetHash := utils.GetHash(filedata)

	// Check if a runtime with the same hash already exists in the database
	existingRuntime, err := ds.server.RuntimeRepo.FindByHash(targetHash)
	if err == nil && existingRuntime != nil {
		// Runtime with same hash already exists
		return &DeployResult{
			Response: &schemas.DeployResponse{
				ID: existingRuntime.ID.String(),
			},
			IsExisting: true,
		}, nil
	}

	// Create new runtime with a new UUID
	id, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}

	// Use S3 storage
	key := id.String() + expectedExt

	// Upload to S3
	if err := ds.s3Storage.UploadFile(context.Background(), "", key, filedata); err != nil {
		return nil, fmt.Errorf("failed to upload file to S3: %w", err)
	}

	// Create the runtime record in the database
	runtimeRecord := &models.Runtime{
		ID:          id,
		RuntimeType: req.RuntimeType,
		Hash:        targetHash,
		S3FilePath:  key, // Store the S3 key in the database
	}

	if err := ds.server.RuntimeRepo.Create(runtimeRecord); err != nil {
		// Clean up the file if DB insertion fails
		// Delete from S3 if DB insertion fails
		ds.s3Storage.DeleteFile(context.Background(), "", key)
		return nil, fmt.Errorf("failed to save runtime to database: %w", err)
	}

	return &DeployResult{
		Response: &schemas.DeployResponse{
			ID: id.String(),
		},
		IsExisting: false,
	}, nil
}

// getFileExtensionForRuntimeType returns the expected file extension for a given runtime type
func getFileExtensionForRuntimeType(runtimeType string) string {
	switch runtimeType {
	case "js":
		return ".js"
	case "wasm":
		return ".wasm"
	default:
		return ".bin" // fallback for unknown types
	}
}

// InvalidRuntimeTypeError represents an error for invalid runtime types
type InvalidRuntimeTypeError struct {
	RuntimeType string
}

func (e *InvalidRuntimeTypeError) Error() string {
	return "Invalid runtime type: " + e.RuntimeType
}