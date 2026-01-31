package services

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/ignis-runtime/ignis-wasmtime/api/rest/v1/schemas"
	"github.com/ignis-runtime/ignis-wasmtime/internal/config"
	"github.com/ignis-runtime/ignis-wasmtime/internal/models"
	"github.com/ignis-runtime/ignis-wasmtime/internal/repository"
	"github.com/ignis-runtime/ignis-wasmtime/internal/storage"
	"github.com/ignis-runtime/ignis-wasmtime/internal/utils"
)

const (
	s3PathFormat = "%s/%s.%s"
)

// DeploymentService defines the interface for deployment operations
type DeploymentService interface {
	CreateDeployment(context context.Context, req schemas.DeployRequest) (*schemas.DeployResponse, error)
	GetDeploymentByID(context context.Context, id uuid.UUID) (*schemas.DeployResponse, error)
	GetDeploymentByHash(context context.Context, hash string) (*schemas.DeployResponse, error)
	ListAllDeployments(context.Context) ([]*schemas.DeployResponse, error)
	GetDeploymentFileContentByUUID(context context.Context, id uuid.UUID) ([]byte, error)
	GetDeploymentFileContentByHash(context context.Context, hash string) ([]byte, error)
}

// deploymentService implements the DeploymentService interface
type deploymentService struct {
	deploymentRepo repository.DeploymentRepository
	config         *config.Config
	s3Storage      storage.S3Storage
}

// NewDeploymentService creates a new instance of DeploymentService
func NewDeploymentService(runtimeRepo repository.DeploymentRepository, s3Storage storage.S3Storage, config *config.Config) DeploymentService {
	return &deploymentService{
		deploymentRepo: runtimeRepo,
		config:         config,
		s3Storage:      s3Storage,
	}
}

// Deploy handles the deployment logic
func (ds *deploymentService) CreateDeployment(context context.Context, req schemas.DeployRequest) (*schemas.DeployResponse, error) {
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
	existingDeployment, err := ds.deploymentRepo.FindByHash(context, targetHash)
	if err == nil && existingDeployment != nil {
		// Runtime with same hash already exists
		return &schemas.DeployResponse{
			ID:          existingDeployment.ID.String(),
			IsExisting:  true,
			RuntimeType: existingDeployment.RuntimeType,
			Hash:        existingDeployment.Hash,
			S3FilePath:  existingDeployment.S3FilePath,
			CreatedAt:   existingDeployment.CreatedAt,
			UpdatedAt:   existingDeployment.UpdatedAt,
		}, nil
	}

	// Create new runtime with a new UUID
	id, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}

	// Use S3 storage
	key := fmt.Sprintf(s3PathFormat, req.RuntimeType, id, expectedExt)

	// Upload to S3
	if err := ds.s3Storage.UploadFile(context, key, filedata); err != nil {
		return nil, fmt.Errorf("failed to upload file to S3: %w", err)
	}

	// Create the runtime record in the database
	runtimeRecord := &models.Runtime{
		ID:          id,
		RuntimeType: req.RuntimeType,
		Hash:        targetHash,
		S3FilePath:  key, // Store the S3 key in the database
	}

	createdRecord, err := ds.deploymentRepo.Create(context, runtimeRecord)
	if err != nil {
		// Clean up the file if DB insertion fails
		// Delete from S3 if DB insertion fails
		err = ds.s3Storage.DeleteFile(context, key)
		return nil, fmt.Errorf("failed to save runtime to database: %w", err)
	}

	return &schemas.DeployResponse{
		ID:          createdRecord.ID.String(),
		IsExisting:  false,
		RuntimeType: createdRecord.RuntimeType,
		Hash:        createdRecord.Hash,
		CreatedAt:   createdRecord.CreatedAt,
		UpdatedAt:   createdRecord.UpdatedAt,
	}, nil
}

func (ds *deploymentService) GetDeploymentByID(context context.Context, id uuid.UUID) (*schemas.DeployResponse, error) {
	runtimeRecord, err := ds.deploymentRepo.FindByID(context, id)
	if err != nil {
		return nil, err
	}
	return &schemas.DeployResponse{
		ID:          runtimeRecord.ID.String(),
		RuntimeType: runtimeRecord.RuntimeType,
		Hash:        runtimeRecord.Hash,
		S3FilePath:  runtimeRecord.S3FilePath,
		CreatedAt:   runtimeRecord.CreatedAt,
		UpdatedAt:   runtimeRecord.UpdatedAt,
	}, nil
}
func (ds *deploymentService) GetDeploymentByHash(context context.Context, hash string) (*schemas.DeployResponse, error) {
	runtimeRecord, err := ds.deploymentRepo.FindByHash(context, hash)
	if err != nil {
		return nil, err
	}
	return &schemas.DeployResponse{
		ID:          runtimeRecord.ID.String(),
		RuntimeType: runtimeRecord.RuntimeType,
		Hash:        runtimeRecord.Hash,
		S3FilePath:  runtimeRecord.S3FilePath,
		CreatedAt:   runtimeRecord.CreatedAt,
		UpdatedAt:   runtimeRecord.UpdatedAt,
	}, nil
}
func (ds *deploymentService) GetDeploymentFileContentByUUID(context context.Context, id uuid.UUID) ([]byte, error) {
	res, err := ds.deploymentRepo.FindByID(context, id)
	if err != nil {
		return nil, err
	}
	return ds.s3Storage.DownloadFile(context, res.S3FilePath)
}
func (ds *deploymentService) ListAllDeployments(context context.Context) ([]*schemas.DeployResponse, error) {
	var res []*schemas.DeployResponse
	records, err := ds.deploymentRepo.GetAll(context)
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		res = append(res, &schemas.DeployResponse{
			ID:          record.ID.String(),
			RuntimeType: record.RuntimeType,
			Hash:        record.Hash,
			S3FilePath:  record.S3FilePath,
			CreatedAt:   record.CreatedAt,
			UpdatedAt:   record.UpdatedAt,
		})
	}
	return res, nil
}
func (ds *deploymentService) GetDeploymentFileContentByHash(context context.Context, hash string) ([]byte, error) {
	res, err := ds.deploymentRepo.FindByHash(context, hash)
	if err != nil {
		return nil, err
	}
	return ds.s3Storage.DownloadFile(context, res.S3FilePath)
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
