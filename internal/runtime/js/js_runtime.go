package js

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/bytecodealliance/wasmtime-go/v41"
	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
	"github.com/ignis-runtime/ignis-wasmtime/internal/utils"

	"github.com/ignis-runtime/ignis-wasmtime/internal/cache"
	"github.com/ignis-runtime/ignis-wasmtime/internal/models"
	"github.com/ignis-runtime/ignis-wasmtime/internal/runtime"
	"github.com/ignis-runtime/ignis-wasmtime/types"
)

//go:embed qjs.wasm
var QJSWasm []byte

const (
	cacheKeyFormat     = "js:%s"
	defaultModulesDir  = "./internal/runtime/js/modules"
	defaultQJSCacheKey = "quickjs-rt"
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
	hash   string
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

func (b *runtimeConfig) GetHash() string {
	return b.hash
}

// getQuickJSRuntime handles the caching logic for the QuickJS WASM module
func (b *runtimeConfig) getQuickJSRuntime(engine *wasmtime.Engine) (*wasmtime.Module, error) {
	var module *wasmtime.Module
	var err error
	// 1. Attempt Cache Retrieval
	cached, exists := b.cache.Get(context.Background(), defaultQJSCacheKey)
	if !exists {
		// 2. Compile Fresh
		module, err = wasmtime.NewModule(engine, QJSWasm)
		if err != nil {
			return nil, fmt.Errorf("failed to compile QuickJS: %w", err)
		}
		// 3. Update Cache
		serialized, err := module.Serialize()
		if err == nil {
			_ = b.cache.Set(context.Background(), defaultQJSCacheKey, &types.Module{
				Hash: utils.GetHash(serialized),
				Data: serialized,
			}, 24*time.Hour)
		}
	} else {
		module, err = wasmtime.NewModuleDeserialize(engine, cached.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize QuickJS: %w", err)
		}
	}

	return module, nil
}

// Instantiate compiles the QuickJS module and initializes the session
func (b *runtimeConfig) Instantiate() (runtime.Runtime, error) {
	if b.cache == nil {
		return nil, fmt.Errorf("error: cache not provided when instantiating the runtime")
	}
	if b.err != nil {
		return nil, b.err
	}

	engine := wasmtime.NewEngine()
	module, err := b.getQuickJSRuntime(engine)
	if err != nil {
		return nil, err
	}

	stdin, stdout, err := runtime.CreateIoDescriptors(b.id)
	if err != nil {
		return nil, err
	}
	b.hash = strconv.FormatUint(xxhash.Sum64(b.jsFile), 16)
	cacheKey := fmt.Sprintf(cacheKeyFormat, b.id)

	var jsFile []byte
	cached, exists := b.cache.Get(context.Background(), cacheKey)
	if !exists {
		jsFile = b.jsFile
		if err := b.cache.Set(context.Background(), cacheKey, &types.Module{Hash: utils.GetHash(jsFile), Data: jsFile}, 24*time.Hour); err != nil {
			return nil, err
		}
	} else {
		jsFile = cached.Data
	}

	return &RuntimeJS{
		session: runtime.Session{
			ID:           b.id,
			Args:         []string{" ", "-e", string(jsFile)},
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
