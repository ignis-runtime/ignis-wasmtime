//go:build darwin || netbsd || openbsd

package runtime

import (
	"fmt"
	"os"

	"github.com/google/uuid"
)

// CreateIoDescriptors generates temporary files for a session on Darwin/macOS.
// Since macOS shm_open files don't support the file operations that WASI expects,
// we use temporary files in the system temp directory instead.
func CreateIoDescriptors(id uuid.UUID) (stdin *os.File, stdout *os.File, err error) {
	// Create a pattern that includes a portion of the UUID to prevent collisions
	// os.CreateTemp will add randomness to ensure uniqueness
	idStr := id.String()
	// Use first 16 characters of UUID for low collision probability while keeping filename reasonable
	stdinPattern := fmt.Sprintf("stdin_%s_*.tmp", idStr[:16])
	stdoutPattern := fmt.Sprintf("stdout_%s_*.tmp", idStr[:16])

	cleanup := func() {
		if stdin != nil {
			stdin.Close()
			os.Remove(stdin.Name())
		}
		if stdout != nil {
			stdout.Close()
			os.Remove(stdout.Name())
		}
	}

	// Create temporary files in the system temp directory
	if stdin, err = os.CreateTemp("", stdinPattern); err != nil {
		return nil, nil, fmt.Errorf("stdin creation: %w", err)
	}

	if stdout, err = os.CreateTemp("", stdoutPattern); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("stdout creation: %w", err)
	}

	return stdin, stdout, nil
}

func cleanupSessionDescriptors(s *Session) {
	if s.Stdin != nil {
		s.Stdin.Close()
		os.Remove(s.Stdin.Name())
	}
	if s.Stdout != nil {
		s.Stdout.Close()
		os.Remove(s.Stdout.Name())
	}
}
