package routes

import (
	"github.com/gin-gonic/gin"
	v1 "github.com/ignis-runtime/ignis-wasmtime/api/rest/v1"
	"github.com/ignis-runtime/ignis-wasmtime/api/rest/v1/handlers"
	"github.com/ignis-runtime/ignis-wasmtime/api/rest/v1/middleware"
	"github.com/ignis-runtime/ignis-wasmtime/internal/services"
)

func runRoutes(runService services.RunService, deployService services.DeploymentService, router gin.IRoutes) {
	runHandlers := handlers.NewRunHandlers(runService, deployService)
	router.Any("/run/:uuid/*path", middleware.UUIDValidator(), v1.ErrorHandler(runHandlers.HandleWasmRequest))
}
