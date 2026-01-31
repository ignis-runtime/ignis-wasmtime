package routes

import (
	"github.com/ignis-runtime/ignis-wasmtime/api/rest/server"
	"github.com/ignis-runtime/ignis-wasmtime/internal/cache"
	"github.com/ignis-runtime/ignis-wasmtime/internal/services"
)

func RegisterRoutes(server *server.Server, deployService services.DeploymentService, cache *cache.RedisCache) {
	apiV1 := server.Engine.Group("/api/v1")

	runRoutes(deployService, cache, apiV1)
	deploymentRoutes(deployService, apiV1)
}
