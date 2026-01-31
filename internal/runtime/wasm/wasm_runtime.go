package wasm

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/ignis-runtime/ignis-wasmtime/internal/cache"
	"github.com/ignis-runtime/ignis-wasmtime/internal/models"
	"github.com/ignis-runtime/ignis-wasmtime/internal/runtime"
	"github.com/ignis-runtime/ignis-wasmtime/types"

	"github.com/bytecodealliance/wasmtime-go/v41"
	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"
)

const (
	cacheKeyFormat = "wasm:%s"
)

type WasmRuntime struct {
	session    runtime.Session
	wasmModule []byte
}

// RuntimeBuilder handles the configuration of a WasmRuntime
type runtimeConfig struct {
	id           uuid.UUID
	wasmModule   []byte
	preopenedDir string
	args         []string
	cache        *cache.RedisCache
	hash         string
	err          error
}

// NewBuilder initializes a new builder with required fields
func NewRuntimeConfig(id uuid.UUID, wasmModule []byte) *runtimeConfig {
	b := &runtimeConfig{id: id, wasmModule: wasmModule}
	if wasmModule == nil {
		b.err = fmt.Errorf("wasm module not provided")
	}
	if id == uuid.Nil {
		b.err = fmt.Errorf("invalid UUID")
	}
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

func (b *runtimeConfig) WithCache(cache *cache.RedisCache) *runtimeConfig {
	b.cache = cache
	return b
}

func (b *runtimeConfig) Type() models.RuntimeType {
	return models.RuntimeTypeWASM
}

func (b *runtimeConfig) GetHash() string {
	return b.hash
}

// getOrCompileModule attempts to fetch a pre-compiled WASM module from Redis
// or compiles it from source if not found or if the hash has changed.
func (b *runtimeConfig) getOrCompileModule(engine *wasmtime.Engine) (*wasmtime.Module, string, error) {
	// 1. Prepare hashing and keys
	rawHash := strconv.FormatUint(xxhash.Sum64(b.wasmModule), 16)
	cacheKey := fmt.Sprintf(cacheKeyFormat, b.id)

	// 2. Try Cache
	cached, found := b.cache.Get(context.Background(), cacheKey)
	if found && cached != nil {
		// Validate hash integrity
		if cached.Hash == rawHash {
			module, err := wasmtime.NewModuleDeserialize(engine, cached.Data)
			if err == nil {
				return module, rawHash, nil
			}
			fmt.Printf("Cache deserialize error: %v\n", err)
		} else {
			fmt.Printf("Cache hash mismatch. Expected: %s, Got: %s\n", rawHash, cached.Hash)
		}
	}

	// 3. Compile Fresh
	module, err := wasmtime.NewModule(engine, b.wasmModule)
	if err != nil {
		return nil, "", fmt.Errorf("module compilation failed: %w", err)
	}

	// 4. Update Cache
	if serialized, err := module.Serialize(); err == nil {
		_ = b.cache.Set(context.Background(), cacheKey, &types.Module{
			Hash: rawHash,
			Data: serialized,
		}, 24*time.Hour)
	}

	return module, rawHash, nil
}

// Instantiate finalizes the construction and performs the heavy initialization
func (b *runtimeConfig) Instantiate() (runtime.Runtime, error) {
	if b.err != nil {
		return nil, b.err
	}
	if b.cache == nil {
		return nil, fmt.Errorf("error: cache not provided when instantiating the runtime")
	}
	engine := wasmtime.NewEngine()

	// Resolve the WASM Module (Cache or Compile)
	module, moduleHash, err := b.getOrCompileModule(engine)
	if err != nil {
		return nil, err
	}
	b.hash = moduleHash // Update builder state

	// Create the IO files in /dev/shm
	stdinFile, stdoutFile, err := runtime.CreateIoDescriptors(b.id)
	if err != nil {
		return nil, fmt.Errorf("io setup failed: %w", err)
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
