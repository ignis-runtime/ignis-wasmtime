package wasm

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/ignis-runtime/ignis-wasmtime/internal/runtime"

	"github.com/bytecodealliance/wasmtime-go/v41"
	"github.com/google/uuid"
)

type WasmRuntime struct {
	session    runtime.Session
	wasmModule []byte
}

// RuntimeBuilder handles the configuration of a WasmRuntime
type runtimeBuilder struct {
	id           uuid.UUID
	wasmModule   []byte
	preopenedDir string
	args         []string
	err          error
}

// NewBuilder initializes a new builder with required fields
func NewWasmRuntime(id uuid.UUID, wasmModule []byte) *runtimeBuilder {
	b := &runtimeBuilder{id: id, wasmModule: wasmModule}
	if wasmModule == nil {
		b.err = fmt.Errorf("wasm module not provided")
	}
	if id == uuid.Nil {
		b.err = fmt.Errorf("invalid UUID")
	}
	return b
}

func (b *runtimeBuilder) WithPreopenedDir(dir string) *runtimeBuilder {
	b.preopenedDir = dir
	return b
}

func (b *runtimeBuilder) WithArgs(args []string) *runtimeBuilder {
	b.args = args
	return b
}

// Build finalizes the construction and performs the heavy initialization
func (b *runtimeBuilder) Build() (runtime.Runtime, error) {
	if b.err != nil {
		return nil, b.err
	}

	engine := wasmtime.NewEngine()

	// Create the IO files in /dev/shm
	stdinFile, stdoutFile, err := runtime.CreateIoDescriptors(b.id)
	if err != nil {
		return nil, fmt.Errorf("io setup failed: %w", err)
	}

	// Compile the module
	module, err := wasmtime.NewModule(engine, b.wasmModule)
	if err != nil {
		// Cleanup IO if module compilation fails
		stdinFile.Close()
		stdoutFile.Close()
		return nil, fmt.Errorf("module compilation failed: %w", err)
	}

	return &WasmRuntime{
		wasmModule: b.wasmModule,
		session: runtime.Session{
			ID:           b.id,
			Args:         b.args,
			Engine:       engine,
			Module:       module,
			Stdin:        stdinFile,
			Stdout:       stdoutFile,
			PreOpenedDir: b.preopenedDir,
		},
	}, nil
}

// Execute runs the WASM module
func (r *WasmRuntime) Execute(ctx context.Context, fdRequest any) ([]byte, error) {
	reqBytes, ok := fdRequest.([]byte)
	if !ok {
		return nil, fmt.Errorf("expected []byte for fdRequest")
	}

	// Reset Stdin
	if err := r.resetFile(r.session.Stdin); err != nil {
		return nil, err
	}

	if _, err := r.session.Stdin.Write(reqBytes); err != nil {
		return nil, fmt.Errorf("failed to write to WASM stdin: %w", err)
	}
	r.session.Stdin.Sync()

	// Setup Store & Linker
	store, linker, err := r.session.NewStore()
	if err != nil {
		return nil, err
	}
	defer store.Close()

	// Run
	if err := r.session.Run(store, linker); err != nil {
		return nil, err
	}

	// Read Output
	if _, err := r.session.Stdout.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek stdout: %w", err)
	}

	return io.ReadAll(r.session.Stdout)
}

func (r *WasmRuntime) resetFile(f *os.File) error {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seek failed: %w", err)
	}
	if err := f.Truncate(0); err != nil {
		return fmt.Errorf("truncate failed: %w", err)
	}
	return nil
}

func (r *WasmRuntime) Close(ctx context.Context) error {
	r.session.Cleanup()
	return nil
}
