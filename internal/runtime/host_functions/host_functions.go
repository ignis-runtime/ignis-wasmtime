//go:build !wasip1

package host_functions

import (
	"github.com/bytecodealliance/wasmtime-go/v41"
)

// Link attaches all host functions to the Wasmtime linker.
func Link(store *wasmtime.Store, linker *wasmtime.Linker) error {
	// Add legacy WASI preview 1 socket functions that might be expected by some WASM modules
	if err := DefineLegacyWasiSockets(linker); err != nil {
		return err
	}

	// Link socket functions
	if err := LinkSocketFunctions(store, linker); err != nil {
		return err
	}

	// Link HTTP functions
	return LinkHTTPFunctions(store, linker)
}