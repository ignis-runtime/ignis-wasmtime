package routes

import (
	"github.com/gin-gonic/gin"
	v1 "github.com/ignis-runtime/ignis-wasmtime/api/rest/v1"
	"github.com/ignis-runtime/ignis-wasmtime/api/rest/v1/handlers"
	"github.com/ignis-runtime/ignis-wasmtime/internal/services"
)

func deploymentRoutes(deployService services.DeploymentService, router gin.IRoutes) {
	deployHandler := handlers.NewDeployHandler(deployService)
	router.POST("/deploy", v1.ErrorHandler(deployHandler.HandleCreateDeployment))
	router.GET("/deploy", v1.ErrorHandler(deployHandler.HandleListDeployments))
}
