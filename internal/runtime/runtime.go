package runtime

import (
	"errors"
	"os"

	"ignis-wasmtime/sdk/http" // To link our custom http host functions

	"github.com/bytecodealliance/wasmtime-go/v41"
)

// Runtime encapsulates the Wasmtime environment for running a specific module.
type Runtime struct {
	engine *wasmtime.Engine
	linker *wasmtime.Linker
	store  *wasmtime.Store
	module *wasmtime.Module
}

// Config holds configuration for the runtime.
type Config struct {
	WasmPath string
}

// NewRuntime creates and initializes a new custom runtime.
func NewRuntime(config Config) (*Runtime, error) {
	// 1. Basic Wasmtime and WASI setup
	engine := wasmtime.NewEngine()
	store := wasmtime.NewStore(engine)
	linker := wasmtime.NewLinker(engine)

	wasiConfig := wasmtime.NewWasiConfig()
	wasiConfig.InheritStdout()
	wasiConfig.InheritStderr()
	store.SetWasi(wasiConfig)

	// 2. Link standard WASI imports
	if err := linker.DefineWasi(); err != nil {
		return nil, err
	}

	// 3. Link our custom SDK's host functions
	if err := http.Link(store, linker); err != nil {
		return nil, err
	}

	// 4. Load and compile the guest WASM module
	wasmBytes, err := os.ReadFile(config.WasmPath)
	if err != nil {
		return nil, err
	}
	module, err := wasmtime.NewModule(engine, wasmBytes)
	if err != nil {
		return nil, err
	}

	// 5. Return the configured runtime instance
	return &Runtime{
		engine: engine,
		linker: linker,
		store:  store,
		module: module,
	}, nil
}

// Run executes the '_start' function of the loaded WASM module.
func (r *Runtime) Run() error {
	// Instantiate the module with the linker.
	instance, err := r.linker.Instantiate(r.store, r.module)
	if err != nil {
		return err
	}

	// Find the _start function.
	startFunc := instance.GetFunc(r.store, "_start")
	if startFunc == nil {
		return errors.New("_start function not found in module")
	}

	// Call the _start function.
	_, err = startFunc.Call(r.store)

	// Handle the clean exit case for WASI command modules.
	if err != nil {
		if wasmtimeErr, ok := err.(*wasmtime.Error); ok {
			if exitStatus, hasStatus := wasmtimeErr.ExitStatus(); hasStatus && exitStatus == 0 {
				return nil
			}
		}
	}
	return err // Return the actual error if it wasn't a clean exit
}
