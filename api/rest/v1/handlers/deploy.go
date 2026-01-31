package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	v1 "github.com/ignis-runtime/ignis-wasmtime/api/rest/v1"
	"github.com/ignis-runtime/ignis-wasmtime/api/rest/v1/schemas"
	"github.com/ignis-runtime/ignis-wasmtime/internal/services"
)

type DeployHandler struct {
	service services.DeploymentService
}

func NewDeployHandler(service services.DeploymentService) *DeployHandler {
	return &DeployHandler{
		service: service,
	}
}

func (d *DeployHandler) HandleCreateDeployment(c *gin.Context) error {
	var req schemas.DeployRequest
	if err := c.ShouldBind(&req); err != nil {
		return v1.APIError{
			Code: http.StatusBadRequest,
			Err:  "Bad Request",
		}
	}

	// Delegate the business logic to the service layer
	result, err := d.service.CreateDeployment(c.Request.Context(), req)
	if err != nil {
		// Handle specific error types
		var invalidRuntimeTypeError *services.InvalidRuntimeTypeError
		if errors.As(err, &invalidRuntimeTypeError) {
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
		Data: result,
	}
}

func (d *DeployHandler) HandleListDeployments(c *gin.Context) error {
	deployments, err := d.service.ListAllDeployments(c.Request.Context())
	if err != nil {
		return v1.APIError{
			Code: http.StatusInternalServerError,
			Err:  "Failed to retrieve runtimes from database",
		}
	}

	return v1.APIResponse{
		Code: http.StatusOK,
		Msg:  "Successfully retrieved runtimes",
		Data: deployments,
	}
}
