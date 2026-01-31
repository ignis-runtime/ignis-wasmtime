//go:build darwin || netbsd || openbsd

package runtime

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"github.com/google/uuid"
)

// shmOpen is a Go wrapper for shm_open.
func shmOpen(regionName string, flags int, perm os.FileMode) (*os.File, error) {
	name, err := syscall.BytePtrFromString(regionName)
	if err != nil {
		return nil, err
	}
	fd, _, errno := syscall.Syscall(syscall.SYS_SHM_OPEN,
		uintptr(unsafe.Pointer(name)),
		uintptr(flags), uintptr(perm),
	)
	if errno != 0 {
		return nil, errno
	}
	return os.NewFile(fd, regionName), nil
}

// shmUnlink is a Go wrapper for shm_unlink.
func shmUnlink(regionName string) error {
	name, err := syscall.BytePtrFromString(regionName)
	if err != nil {
		return err
	}
	if _, _, errno := syscall.Syscall(syscall.SYS_SHM_UNLINK,
		uintptr(unsafe.Pointer(name)), 0, 0,
	); errno != 0 {
		return errno
	}
	return nil
}

// CreateIoDescriptors generates deterministic shared memory regions for a session on Darwin.
func CreateIoDescriptors(id uuid.UUID) (stdin *os.File, stdout *os.File, err error) {
	// POSIX shared memory names must start with a slash and not contain other slashes.
	stdinName := fmt.Sprintf("/%s_stdin", id)
	stdoutName := fmt.Sprintf("/%s_stdout", id)

	cleanup := func() {
		if stdin != nil {
			stdin.Close()
			shmUnlink(stdinName)
		}
		if stdout != nil {
			stdout.Close()
			shmUnlink(stdoutName)
		}
	}

	// O_CREAT | O_RDWR for creating a new read-write region.
	// 0600 permissions: user can read/write.
	flags := syscall.O_RDWR | syscall.O_CREAT
	perm := os.FileMode(0600)

	if stdin, err = shmOpen(stdinName, flags, perm); err != nil {
		return nil, nil, fmt.Errorf("stdin shm_open: %w", err)
	}

	if stdout, err = shmOpen(stdoutName, flags, perm); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("stdout shm_open: %w", err)
	}

	return stdin, stdout, nil
}

func cleanupSessionDescriptors(s *Session) {
	if s.Stdin != nil {
		s.Stdin.Close()
		shmUnlink(s.Stdin.Name())
	}
	if s.Stdout != nil {
		s.Stdout.Close()
		shmUnlink(s.Stdout.Name())
	}
}
