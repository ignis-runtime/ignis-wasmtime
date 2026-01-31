package main

import (
	_ "embed"
	"log"

	"github.com/ignis-runtime/ignis-wasmtime/api/rest/server"
	"github.com/ignis-runtime/ignis-wasmtime/api/rest/v1/routes"
	"github.com/ignis-runtime/ignis-wasmtime/internal/cache"
	"github.com/ignis-runtime/ignis-wasmtime/internal/config"
)

func main() {
	// Initialize Redis cache
	cfg := config.GetConfig()
	redisCache := cache.NewRedisCache(cfg.RedisAddr)

	addr := ":8080"
	srv := server.NewServer(addr, redisCache)
	routes.RegisterRoutes(srv)

	log.Printf("Starting Gin HTTP server on port %s", addr)
	if err := srv.Run(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
