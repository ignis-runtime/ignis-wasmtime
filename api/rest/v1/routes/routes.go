package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/ignis-runtime/ignis-wasmtime/api/rest/server"
	v1 "github.com/ignis-runtime/ignis-wasmtime/api/rest/v1"
	"github.com/ignis-runtime/ignis-wasmtime/api/rest/v1/handlers"
	"github.com/ignis-runtime/ignis-wasmtime/api/rest/v1/middleware"
)

func mainRoutes(server *server.Server, router gin.IRoutes) {
	router.Any("/run/:uuid/*path", middleware.UUIDValidator(), v1.ErrorHandler(server.HandleWasmRequest))
}

func deployRoutes(server *server.Server, router gin.IRoutes) {
	deployHandler := handlers.NewDeployHandler(server)
	router.POST("/deploy", v1.ErrorHandler(deployHandler.HandleDeploy))
}

func runtimeRoutes(server *server.Server, router gin.IRoutes) {
	runtimeHandler := handlers.NewRuntimeHandler(server)
	router.GET("/runtimes", v1.ErrorHandler(runtimeHandler.ListRuntimes))
}

func RegisterRoutes(server *server.Server) {
	apiv1 := server.Engine.Group("/api/v1")
	mainRoutes(server, apiv1)
	deployRoutes(server, apiv1)
	runtimeRoutes(server, apiv1)
}
