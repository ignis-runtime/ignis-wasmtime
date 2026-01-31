package routes

import (
	"github.com/gin-gonic/gin"
	v1 "github.com/ignis-runtime/ignis-wasmtime/api/rest/v1"
	"github.com/ignis-runtime/ignis-wasmtime/api/rest/v1/handlers"
	"github.com/ignis-runtime/ignis-wasmtime/internal/services"
)

// @Summary Create a new deployment
// @Description Creates a new runtime deployment with the provided file and configuration
// @Tags Deployments
// @Accept multipart/form-data
// @Produce json
// @Param runtime_type formData string true "Runtime type (js or wasm)" Enums(js, wasm)
// @Param file formData file true "Runtime file to deploy"
// @Param preopened_dir formData string false "Preopened directory for WASI"
// @Param args[] formData string false "Arguments to pass to the runtime"
// @Success 200 {object} v1.APIResponse{data=schemas.DeployResponse}
// @Failure 400 {object} v1.APIError
// @Failure 500 {object} v1.APIError
// @Router /deploy [post]
func handleCreateDeployment(deployService services.DeploymentService, router gin.IRoutes) {
	deployHandler := handlers.NewDeployHandler(deployService)
	router.POST("/deploy", v1.ErrorHandler(deployHandler.HandleCreateDeployment))
}

// @Summary List all deployments
// @Description Retrieves a list of all deployed runtimes
// @Tags Deployments
// @Accept json
// @Produce json
// @Success 200 {object} v1.APIResponse{data=[]schemas.DeployResponse}
// @Failure 500 {object} v1.APIError
// @Router /deploy [get]
func handleListDeployments(deployService services.DeploymentService, router gin.IRoutes) {
	deployHandler := handlers.NewDeployHandler(deployService)
	router.GET("/deploy", v1.ErrorHandler(deployHandler.HandleListDeployments))
}

func deploymentRoutes(deployService services.DeploymentService, router gin.IRoutes) {
	handleCreateDeployment(deployService, router)
	handleListDeployments(deployService, router)
}
