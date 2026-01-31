package js

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/bytecodealliance/wasmtime-go/v41"
	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"

	"github.com/ignis-runtime/ignis-wasmtime/internal/models"
	"github.com/ignis-runtime/ignis-wasmtime/internal/runtime"
)

//go:embed qjs.wasm
var QJSWasm []byte

const (
	defaultModulesDir = "./internal/runtime/js/modules"
)

// RuntimeJS implements the Runtime interface for JavaScript execution using QuickJS
type RuntimeJS struct {
	session runtime.Session
}

// runtimeConfig handles the configuration for JS execution
type runtimeConfig struct {
	id               uuid.UUID
	jsFile           []byte // The JavaScript source code
	serializedModule []byte // Pre-compiled QuickJS .cwasm bytes
	err              error
	hash             string
}

// NewRuntimeConfig initializes a new builder with the required ID
func NewRuntimeConfig(id uuid.UUID) *runtimeConfig {
	if id == uuid.Nil {
		return &runtimeConfig{err: fmt.Errorf("invalid UUID for JS runtime")}
	}
	// We default the rawModule to the embedded QJSWasm so it works out of the box
	return &runtimeConfig{id: id}
}

// WithJSFile provides the JavaScript source code to execute
func (b *runtimeConfig) WithJSFile(jsFile []byte) *runtimeConfig {
	b.jsFile = jsFile
	return b
}

// WithSerializedModule provides the pre-compiled QuickJS engine bytes
func (b *runtimeConfig) WithSerializedModule(data []byte) *runtimeConfig {
	b.serializedModule = data
	return b
}

func (b *runtimeConfig) Type() models.RuntimeType {
	return models.RuntimeTypeJS
}

// GetHash returns the hash of the JavaScript file (used for identifying the script)
func (b *runtimeConfig) GetHash() string {
	if b.hash == "" && b.jsFile != nil {
		b.hash = strconv.FormatUint(xxhash.Sum64(b.jsFile), 16)
	}
	return b.hash
}

// Instantiate initializes the QuickJS environment and prepares the session
func (b *runtimeConfig) Instantiate() (runtime.Runtime, error) {
	if b.err != nil {
		return nil, b.err
	}
	if len(b.jsFile) == 0 {
		return nil, fmt.Errorf("no javascript source provided")
	}

	engine := wasmtime.NewEngine()
	var module *wasmtime.Module
	var err error

	// 1. Attempt to use Serialized Module (QuickJS Engine Cache Hit)
	if len(b.serializedModule) > 0 {
		module, err = wasmtime.NewModuleDeserialize(engine, b.serializedModule)
		if err != nil {
			module = nil // Fallback to raw if deserialization fails
		}
	}

	stdin, stdout, err := runtime.CreateIoDescriptors(b.id)
	if err != nil {
		return nil, fmt.Errorf("io setup failed: %w", err)
	}

	// Prepare the command line arguments for QuickJS: ["qjs", "-e", "<script_content>"]
	args := []string{"qjs", "-e", string(b.jsFile)}

	return &RuntimeJS{
		session: runtime.Session{
			ID:           b.id,
			Args:         args,
			Engine:       engine,
			Module:       module,
			Stdin:        stdin,
			Stdout:       stdout,
			PreOpenedDir: defaultModulesDir,
		},
	}, nil
}

// Execute runs the JavaScript code
func (r *RuntimeJS) Execute(ctx context.Context, fdRequest any) ([]byte, error) {
	reqBytes, ok := fdRequest.([]byte)
	if !ok {
		return nil, fmt.Errorf("expected []byte for JavaScript input")
	}

	// Reset Stdin for fresh execution
	if err := r.resetFile(r.session.Stdin); err != nil {
		return nil, err
	}

	if _, err := r.session.Stdin.Write(reqBytes); err != nil {
		return nil, fmt.Errorf("failed to write to JS stdin: %w", err)
	}
	_ = r.session.Stdin.Sync()

	// Setup Store & Linker
	store, linker, err := r.session.NewStore()
	if err != nil {
		return nil, err
	}
	defer store.Close()

	// Run QuickJS
	if err := r.session.Run(store, linker); err != nil {
		return nil, fmt.Errorf("JS runtime error: %w", err)
	}

	// Read Output from deterministic stdout
	if _, err := r.session.Stdout.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek stdout: %w", err)
	}

	return io.ReadAll(r.session.Stdout)
}

func (r *RuntimeJS) resetFile(f *os.File) error {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	return f.Truncate(0)
}

// Close cleans up the /dev/shm files
func (r *RuntimeJS) Close(ctx context.Context) error {
	r.session.Cleanup()
	return nil
}
