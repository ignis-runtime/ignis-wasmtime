package main

import (
	"fmt"
	"log"

	"ignis-wasmtime/internal/runtime"
)

func main() {
	// 1. Define the configuration for our custom runtime.
	config := runtime.Config{
		WasmPath: "example/example_post.wasm",
	}

	// 2. Create a new custom runtime instance.
	//    All the Wasmtime setup and host function linking happens here.
	rt, err := runtime.NewRuntime(config)
	if err != nil {
		log.Fatalf("Failed to create runtime: ", err)
	}

	// 3. Run the WASM module.
	fmt.Println("--- Running WASM module via Custom Runtime ---")
	if err := rt.Run(); err != nil {
		log.Fatalf("Runtime error: %v", err)
	}
	fmt.Println("--- WASM module finished ---")
}
