package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ignis-runtime/ignis-wasmtime/api/rest/server"
	v1 "github.com/ignis-runtime/ignis-wasmtime/api/rest/v1"
	"github.com/ignis-runtime/ignis-wasmtime/api/rest/v1/schemas"
	"github.com/ignis-runtime/ignis-wasmtime/internal/config"
	"github.com/ignis-runtime/ignis-wasmtime/internal/services"
)

type DeployHandler struct {
	server *server.Server
	service services.DeployService
}

func NewDeployHandler(server *server.Server) *DeployHandler {
	// Get the config instance
	config := config.GetConfig()
	return &DeployHandler{
		server: server,
		service: services.NewDeployService(server, config),
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

	// Delegate the business logic to the service layer
	result, err := d.service.Deploy(req)
	if err != nil {
		// Handle specific error types
		if _, ok := err.(*services.InvalidRuntimeTypeError); ok {
			return v1.APIError{
				Code: http.StatusBadRequest,
				Err:  err.Error(),
			}
		}

		// Generic error handling for other errors
		return v1.APIError{
			Code: http.StatusInternalServerError,
			Err:  err.Error(),
		}
	}

	// The service layer returns information about whether this was an existing runtime
	var msg string
	if result.IsExisting {
		msg = "Runtime with same hash already exists"
	} else {
		msg = "Successfully deployed"
	}

	return v1.APIResponse{
		Code: http.StatusOK,
		Msg:  msg,
		Data: *result.Response,
	}
}
