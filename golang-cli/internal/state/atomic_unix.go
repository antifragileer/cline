//go:build !windows
// +build !windows

package state

import (
	"fmt"
	"os"
	"syscall"
)

// acquireLock acquires an exclusive lock on the file (Unix implementation)
func (fl *FileLock) acquireLock(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
}

// releaseLock releases the file lock (Unix implementation)
func (fl *FileLock) releaseLock(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}

// tryAcquireLock tries to acquire an exclusive lock without blocking (Unix implementation)
func (fl *FileLock) tryAcquireLock(f *os.File) error {
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		return fmt.Errorf("file already locked")
	}
	return nil
}