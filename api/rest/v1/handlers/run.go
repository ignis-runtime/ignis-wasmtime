package handlers

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	v1 "github.com/ignis-runtime/ignis-wasmtime/api/rest/v1"
	"github.com/ignis-runtime/ignis-wasmtime/internal/cache"
	"github.com/ignis-runtime/ignis-wasmtime/internal/services"
	"github.com/ignis-runtime/ignis-wasmtime/types"
)

type RunHandlers struct {
	cache             *cache.RedisCache
	deploymentService services.DeploymentService
	runService        services.RunService
}

func NewRunHandlers(cache *cache.RedisCache, deploymentService services.DeploymentService) *RunHandlers {
	runService := services.NewRunService(cache, deploymentService)
	return &RunHandlers{
		cache:             cache,
		deploymentService: deploymentService,
		runService:        runService,
	}
}

func (s *RunHandlers) HandleWasmRequest(c *gin.Context) error {
	rawId := c.Param("uuid")
	id, err := uuid.Parse(rawId)
	if err != nil {
		return v1.APIError{Code: http.StatusBadRequest, Err: "invalid UUID"}
	}

	reqBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return v1.APIError{Code: http.StatusInternalServerError, Err: "failed to read request body"}
	}

	reqHeaders := make(map[string]*types.HeaderFields)
	for k, v := range c.Request.Header {
		reqHeaders[k] = &types.HeaderFields{
			Fields: v,
		}
	}

	fdResponse, err := s.runService.ExecuteDeployment(
		c.Request.Context(),
		id,
		&types.FDRequest{
			Method:           c.Request.Method,
			Header:           reqHeaders,
			Body:             reqBody,
			ContentLength:    c.Request.ContentLength,
			TransferEncoding: &types.StringSlice{Fields: c.Request.TransferEncoding},
			Host:             c.Request.Host,
			RemoteAddr:       c.Request.RemoteAddr,
			RequestUri:       c.Param("path"),
			Pattern:          c.Request.Pattern,
		},
	)
	if err != nil {
		return v1.APIError{Code: http.StatusInternalServerError, Err: err.Error()}
	}

	// Map FDResponse back to Gin Response
	return s.writeResponse(c, fdResponse)
}

func (s *RunHandlers) writeResponse(c *gin.Context, res *types.FDResponse) error {
	status := http.StatusOK
	if res.StatusCode != 0 {
		status = int(res.StatusCode)
	}

	for key, values := range res.Header {
		for _, value := range values.Fields {
			c.Writer.Header().Add(key, value)
		}
	}

	c.Data(status, c.Writer.Header().Get("Content-Type"), res.Body)
	return nil
}
