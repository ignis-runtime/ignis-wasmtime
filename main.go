package main

import (
	_ "embed"
	"io"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ignis-runtime/ignis-wasmtime/internal/runtime/js"
	"github.com/ignis-runtime/ignis-wasmtime/internal/runtime/wasm"
	"github.com/ignis-runtime/ignis-wasmtime/internal/server"
)

//go:embed internal/runtime/js/qjs.wasm
var qjsWasmBytes []byte

func main() {
	// Read the default JS content for the factory
	var jsContent []byte
	jsFile, err := os.Open("./example/js/index-page/example.js")
	if err == nil {
		jsContent, _ = io.ReadAll(jsFile)
		jsFile.Close()
	}
	var goContent []byte
	gofile, err := os.Open("./example/go/index-page/example.wasm")
	if err == nil {
		goContent, _ = io.ReadAll(gofile)
		gofile.Close()
	}
	jsRt, err := js.NewJsRuntime(uuid.MustParse("cdda4d36-8943-4033-9caa-e60f89574060"), jsContent).
		Build()
	if err != nil {
		log.Fatalf("error while creating js runtime: %v\n", err)
	}
	goRt, err := wasm.NewWasmRuntime(uuid.MustParse("cdda4d36-8943-4033-9caa-e60f89574061"), goContent).
		WithPreopenedDir("./example").
		Build()
	if err != nil {
		log.Fatalf("error while creating go runtime: %v\n", err)
	}

	server.RegisterRuntime(uuid.MustParse("cdda4d36-8943-4033-9caa-e60f89574060"), jsRt)
	server.RegisterRuntime(uuid.MustParse("cdda4d36-8943-4033-9caa-e60f89574061"), goRt)

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
			// If not a valid UUID, return 404 to avoid processing non-runtime requests
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
