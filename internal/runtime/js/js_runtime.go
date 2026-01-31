package js

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/cespare/xxhash/v2"

	"github.com/bytecodealliance/wasmtime-go/v41"
	"github.com/google/uuid"

	"github.com/ignis-runtime/ignis-wasmtime/internal/cache"
	"github.com/ignis-runtime/ignis-wasmtime/internal/models"
	"github.com/ignis-runtime/ignis-wasmtime/internal/runtime"
	"github.com/ignis-runtime/ignis-wasmtime/types"
)

//go:embed qjs.wasm
var QJSWasm []byte

const (
	cacheKeyFormat    = "js:%s"
	defaultModulesDir = "./internal/runtime/js/modules"
)

// RuntimeJS implements the Runtime interface for JavaScript execution using QuickJS
type RuntimeJS struct {
	session runtime.Session
}

// runtimeConfig handles the configuration for JS execution
type runtimeConfig struct {
	id     uuid.UUID
	jsFile []byte
	cache  *cache.RedisCache
	err    error
}

func NewRuntimeConfig(id uuid.UUID, jsFile []byte) *runtimeConfig {
	b := &runtimeConfig{id: id}
	if id == uuid.Nil {
		b.err = fmt.Errorf("invalid UUID for JS runtime")
	}
	if jsFile == nil {
		b.err = fmt.Errorf("no js file provided")
	}
	b.jsFile = jsFile

	return b
}

func (b *runtimeConfig) WithCache(cache *cache.RedisCache) *runtimeConfig {
	b.cache = cache
	return b
}

func (b *runtimeConfig) Type() models.RuntimeType {
	return models.RuntimeTypeJS
}

// Instantiate compiles the QuickJS module and initializes the session
func (b *runtimeConfig) Instantiate() (runtime.Runtime, error) {
	if b.err != nil {
		return nil, b.err
	}

	engine := wasmtime.NewEngine()

	// Create deterministic IO descriptors
	stdin, stdout, err := runtime.CreateIoDescriptors(b.id)
	if err != nil {
		return nil, err
	}

	// Compile the embedded QuickJS WASM with caching
	var module *wasmtime.Module
	var cacheKey string
	var hash string

	if b.cache != nil {
		hash = strconv.FormatUint(xxhash.Sum64(QJSWasm), 16)
		cacheKey = fmt.Sprintf(cacheKeyFormat, b.id) // Use session ID as cache key

		cachedModule, err := b.cache.Get(context.Background(), cacheKey)
		if err != nil {
			fmt.Printf("Cache get error: %v\n", err)
		}

		if cachedModule != nil {
			// Check if the cached module hash matches the current module hash
			if cachedModule.Hash == hash {
				module, err = wasmtime.NewModuleDeserialize(engine, cachedModule.Data)
				if err != nil {
					fmt.Printf("Cache deserialize error: %v\n", err)
				} else {
					fmt.Printf("Using cached module for hash: %s\n", hash)
				}
			} else {
				fmt.Printf("Cached module hash mismatch. Cached: %s, Current: %s\n", cachedModule.Hash, hash)
			}
		}
	}

	if module == nil {
		var err error
		module, err = wasmtime.NewModule(engine, QJSWasm)
		if err != nil {
			stdin.Close()
			stdout.Close()
			return nil, fmt.Errorf("failed to compile QuickJS: %w", err)
		}

		if b.cache != nil {
			serializedModule, err := module.Serialize()
			if err != nil {
				fmt.Printf("Module serialize error: %v\n", err)
			} else {
				err := b.cache.Set(context.Background(), cacheKey, &types.Module{
					Hash: hash,
					Data: serializedModule,
				}, 24*time.Hour)
				if err != nil {
					fmt.Printf("Cache set error: %v\n", err)
				} else {
					fmt.Printf("Cached module with hash: %s\n", hash)
				}
			}
		}
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
