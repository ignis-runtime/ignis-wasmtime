package schemas

import (
	"mime/multipart"
	"time"
)

// DeployRequest represents the request body for creating a new deployment
// @Description Deployment creation request
type DeployRequest struct {
	RuntimeType  string                `form:"runtime_type" binding:"required,oneof=js wasm"` // Runtime type (js or wasm)
	File         *multipart.FileHeader `form:"file" binding:"required"`                       // Runtime file to deploy
	PreopenedDir string                `form:"preopened_dir"`                                 // Preopened directory for WASI
	Args         []string              `form:"args"`                                          // Arguments to pass to the runtime
}

// DeployResponse represents the response body for a deployment
// @Description Deployment response
type DeployResponse struct {
	ID          string    `json:"id"`           // Unique identifier for the deployment
	IsExisting  bool      `json:"is_existing"`  // Indicates if this was an existing runtime with the same hash
	RuntimeType string    `json:"runtime_type"` // Type of runtime (js or wasm)
	Hash        string    `json:"hash"`         // Hash of the deployed file
	S3FilePath  string    `json:"-"`            // Path to the file in S3 storage (not returned in API)
	CreatedAt   time.Time `json:"created_at"`   // Creation timestamp
	UpdatedAt   time.Time `json:"updated_at"`   // Last update timestamp
}
