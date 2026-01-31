package main

import (
	_ "embed"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ignis-runtime/ignis-wasmtime/internal/cache"
	"github.com/ignis-runtime/ignis-wasmtime/internal/config"
	"github.com/ignis-runtime/ignis-wasmtime/internal/runtime/js"
	"github.com/ignis-runtime/ignis-wasmtime/internal/runtime/wasm"
	"github.com/ignis-runtime/ignis-wasmtime/internal/server"
)

//go:embed internal/runtime/js/qjs.wasm
var qjsWasmBytes []byte

func main() {
	// Initialize Redis cache
	cfg := config.GetConfig()
	redisCache := cache.NewRedisCache(cfg.RedisAddr)

	var jsContent []byte
	jsContent, err := os.ReadFile("./example/js/index-page/example.js")
	if err != nil {
		log.Println("failed to read js file: ", err)
	}

	var goContent []byte
	goContent, err = os.ReadFile("./example/go/index-page/example.wasm")
	if err != nil {
		log.Println("failed to read wasm file: ", err)
	}
	jsRuntimeConfig := js.NewRuntimeConfig(uuid.MustParse("cdda4d36-8943-4033-9caa-e60f89574060"), jsContent).
		WithCache(redisCache)
	goRuntimeConfig := wasm.NewRuntimeConfig(uuid.MustParse("cdda4d36-8943-4033-9caa-e60f89574061"), goContent).
		WithPreopenedDir("./example").
		WithCache(redisCache)

	server.RegisterRuntimeConfigs(uuid.MustParse("cdda4d36-8943-4033-9caa-e60f89574060"), jsRuntimeConfig)
	server.RegisterRuntimeConfigs(uuid.MustParse("cdda4d36-8943-4033-9caa-e60f89574061"), goRuntimeConfig)

	// Setup Gin router
	router := gin.Default()
	srv := server.NewServer()

	// Define the route that captures UUID and path
	router.Any("/:uuid/*path", func(c *gin.Context) {
		// Check if the first segment looks like a UUID before processing
		runtimeID := c.Param("uuid")

		// Validate UUID format before calling HandleWasmRequest
		parsedUUID, err := uuid.Parse(runtimeID)
		if err != nil {
			c.JSON(404, gin.H{"error": "Not Found"})
			return
		}

		// Store the parsed UUID in the context for HandleWasmRequest to use
		c.Set("parsed_uuid", parsedUUID)
		srv.HandleWasmRequest(c) // Use the single Server instance
	})

	port := ":8080"
	log.Printf("Starting Gin HTTP server on port %s", port)
	if err := router.Run(port); err != nil {
		log.Fatalf("Gin HTTP server failed: %s", err)
	}
}
