package types

import (
	"context"
)

// Runtime defines the interface for execution environments
type Runtime interface {
	// Execute runs the runtime with the given HTTP request and returns the raw output
	Execute(ctx context.Context, fdRequest interface{}) ([]byte, error)

	// Close cleans up resources used by the runtime
	Close() error
}