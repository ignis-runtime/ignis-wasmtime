//go:build linux

package runtime

import (
	"fmt"
	"os"

	"github.com/google/uuid"
)

const SharedMemDir = "/dev/shm"

// CreateIoDescriptors generates deterministic file paths in /dev/shm for a session.
func CreateIoDescriptors(id uuid.UUID) (stdin *os.File, stdout *os.File, err error) {
	// Pattern includes the UUID and a suffix placeholder
	stdinPattern := fmt.Sprintf("%s_stdin-*.tmp", id)
	stdoutPattern := fmt.Sprintf("%s_stdout-*.tmp", id)

	// Helper to clean up on failure
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

	// os.CreateTemp handles the "*" randomization automatically
	if stdin, err = os.CreateTemp(SharedMemDir, stdinPattern); err != nil {
		return nil, nil, fmt.Errorf("stdin creation: %w", err)
	}

	if stdout, err = os.CreateTemp(SharedMemDir, stdoutPattern); err != nil {
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
