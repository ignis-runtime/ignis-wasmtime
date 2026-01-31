package handlers

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ignis-runtime/ignis-wasmtime/api/rest/server"
	v1 "github.com/ignis-runtime/ignis-wasmtime/api/rest/v1"
	"github.com/ignis-runtime/ignis-wasmtime/api/rest/v1/schemas"
	"github.com/ignis-runtime/ignis-wasmtime/internal/runtime"
	"github.com/ignis-runtime/ignis-wasmtime/internal/runtime/js"
	"github.com/ignis-runtime/ignis-wasmtime/internal/runtime/wasm"
	"github.com/ignis-runtime/ignis-wasmtime/internal/utils"
)

type DeployHandler struct {
	server *server.Server
}

func NewDeployHandler(server *server.Server) *DeployHandler {
	return &DeployHandler{
		server: server,
	}
}
func (d *DeployHandler) HandleDeploy(c *gin.Context) error {
	var req schemas.DeployRequest
	if err := c.ShouldBind(&req); err != nil {
		return v1.APIError{
			Code: http.StatusBadRequest,
			Err:  "Bad Request",
		}
	}

	file, err := req.File.Open()
	if err != nil {
		return v1.APIError{
			Code: http.StatusBadRequest,
			Err:  err.Error(),
		}
	}
	defer file.Close()
	filedata, err := io.ReadAll(file)
	if err != nil {
		return v1.APIError{
			Code: http.StatusInternalServerError,
			Err:  err.Error(),
		}
	}

	// Calculate the hash based on the runtime type
	targetHash := utils.GetHash(filedata)

	// Check if a runtime with the same hash already exists
	existingID, exists := d.server.FindRuntimeByHash(targetHash)
	if exists {
		return v1.APIResponse{
			Code: http.StatusOK,
			Msg:  "Runtime with same hash already exists",
			Data: schemas.DeployResponse{
				ID: existingID.String(),
			},
		}
	}

	// Create new runtime with a new UUID
	id, err := uuid.NewUUID()
	if err != nil {
		return v1.APIError{
			Code: http.StatusInternalServerError,
			Err:  err.Error(),
		}
	}

	var config runtime.RuntimeConfig
	switch req.RuntimeType {
	case "js":
		config = js.NewRuntimeConfig(id, filedata).WithCache(d.server.Cache)
	case "wasm":
		config = wasm.NewRuntimeConfig(id, filedata).WithArgs(req.Args).WithCache(d.server.Cache).WithPreopenedDir(req.PreopenedDir)
	default:
		return v1.APIError{
			Code: http.StatusBadRequest,
			Err:  "Invalid runtime type",
		}
	}
	d.server.RegisterRuntime(id, config)
	return v1.APIResponse{
		Code: http.StatusOK,
		Msg:  "Successfully deployed",
		Data: schemas.DeployResponse{
			ID: id.String(),
		},
	}
}
