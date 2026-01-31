package runtime

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/ignis-runtime/ignis-wasmtime/internal/models"
	"github.com/ignis-runtime/ignis-wasmtime/internal/runtime/host_functions"

	"github.com/bytecodealliance/wasmtime-go/v41"
	"github.com/google/uuid"
)

type Runtime interface {
	Execute(ctx context.Context, fdrequest any) ([]byte, error)
	Close(ctx context.Context) error
}
type RuntimeConfig interface {
	Type() models.RuntimeType
	Instantiate() (Runtime, error)
}

// Session represents a single execution context.
type Session struct {
	ID           uuid.UUID
	Args         []string
	Engine       *wasmtime.Engine
	Module       *wasmtime.Module
	Stdin        *os.File
	Stdout       *os.File
	PreOpenedDir string
}

// NewSession is a constructor to ensure all resources are initialized correctly.
func NewSession(id uuid.UUID, engine *wasmtime.Engine, module *wasmtime.Module, args []string) (*Session, error) {
	stdin, stdout, err := CreateIoDescriptors(id)
	if err != nil {
		return nil, err
	}

	return &Session{
		ID:     id,
		Args:   args,
		Engine: engine,
		Module: module,
		Stdin:  stdin,
		Stdout: stdout,
	}, nil
}

// NewStore configures the WASI environment and links host functions.
func (s *Session) NewStore() (*wasmtime.Store, *wasmtime.Linker, error) {
	store := wasmtime.NewStore(s.Engine)
	linker := wasmtime.NewLinker(s.Engine)

	wasiConfig := wasmtime.NewWasiConfig()
	wasiConfig.SetStdinFile(s.Stdin.Name())
	wasiConfig.SetStdoutFile(s.Stdout.Name())
	wasiConfig.InheritStderr()
	wasiConfig.InheritEnv()
	wasiConfig.SetArgv(s.Args)

	dir, err := s.getPreopenDir()
	if err != nil {
		return nil, nil, err
	}
	wasiConfig.PreopenDir(dir, "/", wasmtime.DIR_READ, wasmtime.FILE_READ|wasmtime.FILE_WRITE)

	store.SetWasi(wasiConfig)

	if err := linker.DefineWasi(); err != nil {
		return nil, nil, fmt.Errorf("wasi link: %w", err)
	}

	if err := host_functions.Link(store, linker); err != nil {
		return nil, nil, fmt.Errorf("host functions link: %w", err)
	}

	return store, linker, nil
}

// Run executes the module. Note that for modern WASI, the Linker
// often handles finding '_start' automatically during instantiation.
func (s *Session) Run(store *wasmtime.Store, linker *wasmtime.Linker) error {
	instance, err := linker.Instantiate(store, s.Module)
	if err != nil {
		return fmt.Errorf("instantiation failed: %w", err)
	}

	// Modern WASI check: some modules use _start, some use a default linker entry
	start := instance.GetFunc(store, "_start")
	if start == nil {
		return errors.New("missing _start function")
	}

	_, err = start.Call(store)

	// Check for WASI exit code 0 (which Go treats as an error)
	if err != nil {
		if exitErr, ok := err.(*wasmtime.Error); ok {
			if code, ok := exitErr.ExitStatus(); ok && code == 0 {
				return nil
			}
		}
		return fmt.Errorf("execution error: %w", err)
	}

	return nil
}

func (s *Session) getPreopenDir() (string, error) {
	if s.PreOpenedDir != "" {
		return s.PreOpenedDir, nil
	}
	return os.Getwd()
}

// Cleanup ensures temporary resources are released.
func (s *Session) Cleanup() {
	cleanupSessionDescriptors(s)
	if s.Engine != nil {
		s.Engine.Close()
	}
	if s.Module != nil {
		s.Module.Close()
	}
}
