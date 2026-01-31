package server

import (
	"github.com/gin-gonic/gin"
	"github.com/ignis-runtime/ignis-wasmtime/internal/cache"
	"github.com/ignis-runtime/ignis-wasmtime/internal/services"
)

type Server struct {
	Addr   string
	Engine *gin.Engine
}

func NewServer(addr string, cache *cache.RedisCache, deploymentService services.DeploymentService) *Server {
	gin.SetMode(gin.ReleaseMode)
	s := &Server{
		Addr:   addr,
		Engine: gin.Default(),
	}

	return s
}

func (s *Server) Run() error {
	return s.Engine.Run(s.Addr)
}
