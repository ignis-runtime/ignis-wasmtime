package js

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"

	"github.com/bytecodealliance/wasmtime-go/v41"
	"github.com/google/uuid"

	"github.com/ignis-runtime/ignis-wasmtime/internal/runtime"
)

//go:embed qjs.wasm
var QJSWasm []byte

const (
	DefaultModulesDir = "./internal/runtime/js/modules"
)

// RuntimeJS implements the Runtime interface for JavaScript execution using QuickJS
type RuntimeJS struct {
	session runtime.Session
}

// runtimeBuilder handles the configuration for JS execution
type runtimeBuilder struct {
	id     uuid.UUID
	jsFile []byte
	// args         []string
	// preopenedDir string
	err error
}

func NewJsRuntime(id uuid.UUID, jsFile []byte) *runtimeBuilder {
	b := &runtimeBuilder{id: id}
	if id == uuid.Nil {
		b.err = fmt.Errorf("invalid UUID for JS runtime")
	}
	if jsFile == nil {
		b.err = fmt.Errorf("no js file provided")
	}
	b.jsFile = jsFile

	return b
}

// Build compiles the QuickJS module and initializes the session
func (b *runtimeBuilder) Build() (runtime.Runtime, error) {
	if b.err != nil {
		return nil, b.err
	}

	engine := wasmtime.NewEngine()

	// Create deterministic IO descriptors
	stdin, stdout, err := runtime.CreateIoDescriptors(b.id)
	if err != nil {
		return nil, err
	}

	// Compile the embedded QuickJS WASM
	module, err := wasmtime.NewModule(engine, QJSWasm)
	if err != nil {
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("failed to compile QuickJS: %w", err)
	}
	args := []string{" ", "-e", string(b.jsFile)}
	return &RuntimeJS{
		session: runtime.Session{
			ID:           b.id,
			Args:         args,
			Engine:       engine,
			Module:       module,
			Stdin:        stdin,
			Stdout:       stdout,
			PreOpenedDir: DefaultModulesDir,
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
	r.session.Stdin.Sync()

	// Setup Store & Linker (utilizing the logic in runtime/session.go)
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
