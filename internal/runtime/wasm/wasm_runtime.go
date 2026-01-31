package wasm

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/bytecodealliance/wasmtime-go/v41"
	"github.com/google/uuid"
	"github.com/ignis-runtime/ignis-wasmtime/internal/models"
	"github.com/ignis-runtime/ignis-wasmtime/internal/runtime"
)

type WasmRuntime struct {
	session runtime.Session
}

// runtimeConfig handles the configuration of a WasmRuntime
type runtimeConfig struct {
	id               uuid.UUID
	serializedModule []byte
	preopenedDir     string
	args             []string
	hash             string
	err              error
}

// NewRuntimeConfig initializes a new builder with the required ID
func NewRuntimeConfig(id uuid.UUID) *runtimeConfig {
	if id == uuid.Nil {
		return &runtimeConfig{err: fmt.Errorf("invalid UUID")}
	}
	return &runtimeConfig{id: id}
}

// WithSerializedModule provides pre-compiled bytes (e.g., fetched from Redis)
func (b *runtimeConfig) WithSerializedModule(data []byte) *runtimeConfig {
	b.serializedModule = data
	return b
}

func (b *runtimeConfig) WithPreopenedDir(dir string) *runtimeConfig {
	b.preopenedDir = dir
	return b
}

func (b *runtimeConfig) WithArgs(args []string) *runtimeConfig {
	b.args = args
	return b
}

func (b *runtimeConfig) Type() models.RuntimeType {
	return models.RuntimeTypeWASM
}

// Instantiate finalizes the construction and performs the heavy initialization
func (b *runtimeConfig) Instantiate() (runtime.Runtime, error) {
	if b.err != nil {
		return nil, b.err
	}

	engine := wasmtime.NewEngine()
	var module *wasmtime.Module
	var err error

	// 1. Attempt to use Serialized Module first (Cache Hit path)
	if len(b.serializedModule) > 0 {
		module, err = wasmtime.NewModuleDeserialize(engine, b.serializedModule)
		if err != nil {
			// If deserialization fails, we don't return error yet,
			// we try to fall back to raw compilation if available.
			module = nil
		}
	}

	// Setup IO descriptors in /dev/shm
	stdinFile, stdoutFile, err := runtime.CreateIoDescriptors(b.id)
	if err != nil {
		return nil, fmt.Errorf("io setup failed: %w", err)
	}

	return &WasmRuntime{
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
	_ = r.session.Stdin.Sync()

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
