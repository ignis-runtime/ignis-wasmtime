package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ignis-runtime/ignis-wasmtime/api/rest/server"
	v1 "github.com/ignis-runtime/ignis-wasmtime/api/rest/v1"
	"github.com/ignis-runtime/ignis-wasmtime/api/rest/v1/schemas"
)

type RuntimeHandler struct {
	server *server.Server
}

func NewRuntimeHandler(server *server.Server) *RuntimeHandler {
	return &RuntimeHandler{
		server: server,
	}
}

func (rh *RuntimeHandler) ListRuntimes(c *gin.Context) error {
	// Get all runtimes from the database
	runtimes, err := rh.server.RuntimeRepo.GetAll()
	if err != nil {
		return v1.APIError{
			Code: http.StatusInternalServerError,
			Err:  "Failed to retrieve runtimes from database",
		}
	}

	// Convert to response format
	var responses []schemas.RuntimeResponse
	for _, runtime := range runtimes {
		responses = append(responses, schemas.RuntimeResponse{
			ID:          runtime.ID.String(),
			RuntimeType: runtime.RuntimeType,
			Hash:        runtime.Hash,
			S3FilePath:  runtime.S3FilePath,
			CreatedAt:   runtime.CreatedAt,
			UpdatedAt:   runtime.UpdatedAt,
		})
	}

	return v1.APIResponse{
		Code: http.StatusOK,
		Msg:  "Successfully retrieved runtimes",
		Data: responses,
	}
}