package schemas

import (
	"mime/multipart"
	"time"
)

type DeployRequest struct {
	RuntimeType  string                `form:"runtime_type" binding:"required,oneof=js wasm"`
	File         *multipart.FileHeader `form:"file" binding:"required"`
	PreopenedDir string                `form:"preopened_dir"`
	Args         []string              `form:"args"`
}

type DeployResponse struct {
	ID          string    `json:"id"`
	IsExisting  bool      `json:"is_existing"`
	RuntimeType string    `json:"runtime_type"`
	Hash        string    `json:"hash"`
	S3FilePath  string    `json:"-"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
