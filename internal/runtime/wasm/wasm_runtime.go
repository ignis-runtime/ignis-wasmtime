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

// Instantiate finalizes the construction and performs the heavy initialization
func (b *runtimeConfig) Instantiate() (runtime.Runtime, error) {
	if b.err != nil {
		return nil, b.err
	}

	providedFileHash := xxhash.Sum64(b.wasmModule)
	b.hash = strconv.FormatUint(providedFileHash, 32)

	engine := wasmtime.NewEngine()

	// Create the IO files in /dev/shm
	stdinFile, stdoutFile, err := runtime.CreateIoDescriptors(b.id)
	if err != nil {
		return nil, fmt.Errorf("io setup failed: %w", err)
	}
	var module *wasmtime.Module
	var cacheKey string
	var hash string
	if b.cache != nil {
		hash = strconv.FormatUint(xxhash.Sum64(b.wasmModule), 16)
		cacheKey = fmt.Sprintf(cacheKeyFormat, b.id) // Use session ID as cache key

		cachedModule, err := b.cache.Get(context.Background(), cacheKey)
		if err != nil {
			// Log the error but proceed to compile
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
		module, err = wasmtime.NewModule(engine, b.wasmModule)
		if err != nil {
			stdinFile.Close()
			stdoutFile.Close()
			return nil, fmt.Errorf("module compilation failed: %w", err)
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
