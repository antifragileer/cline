// Package state provides atomic file operations for state management.
// This ensures data integrity during concurrent access and system crashes.
package state

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// AtomicFileWriter provides atomic file write operations
type AtomicFileWriter struct {
	mu sync.Mutex
}

// NewAtomicFileWriter creates a new atomic file writer
func NewAtomicFileWriter() *AtomicFileWriter {
	return &AtomicFileWriter{}
}

// WriteFile writes data to a file atomically using a temporary file and rename
func (w *AtomicFileWriter) WriteFile(path string, data []byte, perm os.FileMode) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Ensure the directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Create a temporary file in the same directory
	tempFile, err := os.CreateTemp(dir, ".tmp-")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()

	// Clean up temp file if something goes wrong
	defer func() {
		if err != nil {
			os.Remove(tempPath)
		}
	}()

	// Write data to temp file
	if _, err = tempFile.Write(data); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	// Sync to ensure data is written to disk
	if err = tempFile.Sync(); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	// Close the file before renaming
	if err = tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Set permissions
	if err = os.Chmod(tempPath, perm); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Atomic rename
	if err = os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	// Sync the directory to ensure the rename is persisted
	if dirFile, err := os.Open(dir); err == nil {
		dirFile.Sync()
		dirFile.Close()
	}

	return nil
}

// CopyFile copies a file atomically
func (w *AtomicFileWriter) CopyFile(src, dst string, perm os.FileMode) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Open source file
	source, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer source.Close()

	// Ensure destination directory exists
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Create temp file
	tempFile, err := os.CreateTemp(dir, ".tmp-")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()

	// Clean up on error
	defer func() {
		if err != nil {
			os.Remove(tempPath)
		}
	}()

	// Copy data
	if _, err = io.Copy(tempFile, source); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to copy data: %w", err)
	}

	// Sync to disk
	if err = tempFile.Sync(); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	// Close before rename
	if err = tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Set permissions
	if err = os.Chmod(tempPath, perm); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Atomic rename
	if err = os.Rename(tempPath, dst); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// SafeAppend provides safe append operations for log files
func (w *AtomicFileWriter) SafeAppend(path string, data []byte, perm os.FileMode) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Open file for appending (create if doesn't exist)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, perm)
	if err != nil {
		return fmt.Errorf("failed to open file for append: %w", err)
	}
	defer f.Close()

	// Write data
	if _, err = f.Write(data); err != nil {
		return fmt.Errorf("failed to append data: %w", err)
	}

	// Sync to ensure durability
	if err = f.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	return nil
}

// FileLock provides advisory file locking for exclusive access
type FileLock struct {
	path string
	file *os.File
	mu   sync.Mutex
}

// NewFileLock creates a new file lock
func NewFileLock(path string) *FileLock {
	return &FileLock{
		path: path,
	}
}

// Lock acquires an exclusive lock on the file
func (fl *FileLock) Lock() error {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	if fl.file != nil {
		return fmt.Errorf("file already locked")
	}

	// Ensure directory exists
	dir := filepath.Dir(fl.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Open or create the lock file
	f, err := os.OpenFile(fl.path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed to open lock file: %w", err)
	}

	// Try to acquire exclusive lock (platform-specific)
	if err := fl.acquireLock(f); err != nil {
		f.Close()
		return err
	}

	fl.file = f
	return nil
}

// Unlock releases the file lock
func (fl *FileLock) Unlock() error {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	if fl.file == nil {
		return fmt.Errorf("file not locked")
	}

	// Release lock (platform-specific)
	if err := fl.releaseLock(fl.file); err != nil {
		fl.file.Close()
		fl.file = nil
		return err
	}

	// Close the file
	if err := fl.file.Close(); err != nil {
		fl.file = nil
		return fmt.Errorf("failed to close lock file: %w", err)
	}

	fl.file = nil
	return nil
}

// TryLock attempts to acquire the lock without blocking
func (fl *FileLock) TryLock() error {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	if fl.file != nil {
		return fmt.Errorf("file already locked")
	}

	// Ensure directory exists
	dir := filepath.Dir(fl.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Open or create the lock file
	f, err := os.OpenFile(fl.path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed to open lock file: %w", err)
	}

	// Try to acquire exclusive lock without blocking (platform-specific)
	if err := fl.tryAcquireLock(f); err != nil {
		f.Close()
		return err
	}

	fl.file = f
	return nil
}

// IsLocked returns true if the file is currently locked
func (fl *FileLock) IsLocked() bool {
	fl.mu.Lock()
	defer fl.mu.Unlock()
	return fl.file != nil
}