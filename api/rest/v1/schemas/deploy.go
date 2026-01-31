package schemas

import (
	"mime/multipart"
)

type DeployRequest struct {
	RuntimeType  string                `form:"runtime_type" binding:"required,oneof=js wasm"`
	File         *multipart.FileHeader `form:"file" binding:"required"`
	PreopenedDir string                `form:"preopened_dir"`
	Args         []string              `form:"args"`
}

type DeployResponse struct {
	ID string `json:"deployment_id"`
}
