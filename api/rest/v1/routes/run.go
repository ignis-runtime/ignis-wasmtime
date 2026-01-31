package routes

import (
	"github.com/gin-gonic/gin"
	v1 "github.com/ignis-runtime/ignis-wasmtime/api/rest/v1"
	"github.com/ignis-runtime/ignis-wasmtime/api/rest/v1/handlers"
	"github.com/ignis-runtime/ignis-wasmtime/api/rest/v1/middleware"
	"github.com/ignis-runtime/ignis-wasmtime/internal/cache"
	"github.com/ignis-runtime/ignis-wasmtime/internal/services"
)

func runRoutes(deployService services.DeploymentService, cache *cache.RedisCache, router gin.IRoutes) {
	runHandlers := handlers.NewRunHandlers(cache, deployService)
	router.Any("/run/:uuid/*path", middleware.UUIDValidator(), v1.ErrorHandler(runHandlers.HandleWasmRequest))
}
