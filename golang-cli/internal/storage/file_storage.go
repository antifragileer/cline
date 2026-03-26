package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gofrs/flock"
)

// ClineFileStorage implements the Storage interface with atomic file operations
// and file locking for thread-safe and process-safe access to JSON storage files.
type ClineFileStorage struct {
	// filePath is the path to the JSON storage file
	filePath string

	// permissions is the file permission mode for the storage file
	permissions os.FileMode

	// data holds the in-memory cache of all key-value pairs
	data map[string]interface{}

	// fileLock provides process-level locking using flock
	fileLock *flock.Flock

	// mu provides thread-level locking for the data map
	mu sync.RWMutex

	// dirty tracks whether the in-memory data has been modified
	dirty bool

	// closed tracks whether the storage has been closed
	closed bool
}

// NewClineFileStorage creates a new ClineFileStorage instance.
// It creates the storage directory if it doesn't exist and loads existing data.
func NewClineFileStorage(filePath string, permissions os.FileMode) (*ClineFileStorage, error) {
	// Create parent directories if they don't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Initialize file lock
	lockPath := filePath + ".lock"
	fileLock := flock.New(lockPath)

	storage := &ClineFileStorage{
		filePath:    filePath,
		permissions: permissions,
		data:        make(map[string]interface{}),
		fileLock:    fileLock,
	}

	// Load existing data if file exists
	if _, err := os.Stat(filePath); err == nil {
		if err := storage.load(); err != nil {
			return nil, fmt.Errorf("failed to load storage from %s: %w", filePath, err)
		}
	}

	return storage, nil
}

// Get retrieves a value by key from storage.
// It returns the value and true if found, or nil and false if not found.
func (s *ClineFileStorage) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, false
	}

	val, ok := s.data[key]
	if !ok {
		return nil, false
	}

	// Return a deep copy to prevent external modification
	cloned, err := CloneValue(val)
	if err != nil {
		// If cloning fails, return the original value
		// This is a fallback that maintains backward compatibility
		return val, true
	}

	return cloned, true
}

// Set stores a value for the given key.
// The write is atomic and protected by both thread and file locks.
func (s *ClineFileStorage) Set(key string, value interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("storage is closed")
	}

	// Store a deep copy to prevent external modification affecting our data
	cloned, err := CloneValue(value)
	if err != nil {
		// If cloning fails, store the original value
		// This is a fallback that maintains backward compatibility
		cloned = value
	}

	s.data[key] = cloned
	s.dirty = true

	return s.flush()
}

// SetBatch stores multiple key-value pairs atomically.
// All pairs are written in a single atomic operation.
func (s *ClineFileStorage) SetBatch(pairs map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("storage is closed")
	}

	// Clone all values before storing
	for key, value := range pairs {
		cloned, err := CloneValue(value)
		if err != nil {
			cloned = value
		}
		s.data[key] = cloned
	}

	s.dirty = true

	return s.flush()
}

// Delete removes a key from storage.
func (s *ClineFileStorage) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("storage is closed")
	}

	delete(s.data, key)
	s.dirty = true

	return s.flush()
}

// GetAll returns all key-value pairs as a map.
// The returned map is a deep copy and can be safely modified.
func (s *ClineFileStorage) GetAll() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return make(map[string]interface{})
	}

	// Create a deep copy of all data
	result := make(map[string]interface{}, len(s.data))
	for key, value := range s.data {
		cloned, err := CloneValue(value)
		if err != nil {
			result[key] = value
		} else {
			result[key] = cloned
		}
	}

	return result
}

// Close releases all resources held by the storage.
// It ensures any pending writes are flushed and locks are released.
func (s *ClineFileStorage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true

	// Flush any pending changes
	var flushErr error
	if s.dirty {
		flushErr = s.flush()
	}

	// Release file lock
	if s.fileLock != nil {
		if err := s.fileLock.Unlock(); err != nil {
			// Log error but don't fail close
			// In production, this could use a proper logger
			_ = err
		}
	}

	return flushErr
}

// load reads the storage file from disk and populates the in-memory cache.
// This method must be called with the lock held or during initialization.
func (s *ClineFileStorage) load() error {
	// Acquire file lock for reading with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	locked, err := s.fileLock.TryLockContext(ctx, 100*time.Millisecond)
	if err != nil {
		return fmt.Errorf("failed to acquire file lock: %w", err)
	}
	if !locked {
		return fmt.Errorf("failed to acquire file lock: timeout")
	}
	defer s.fileLock.Unlock()

	file, err := os.Open(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet, start with empty data
			return nil
		}
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	if len(data) == 0 {
		// Empty file, start with empty data
		return nil
	}

	var storedData map[string]interface{}
	if err := json.Unmarshal(data, &storedData); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	s.data = storedData
	s.dirty = false

	return nil
}

// flush writes the in-memory data to disk atomically.
// This method must be called with the write lock held.
func (s *ClineFileStorage) flush() error {
	// Acquire file lock for writing
	locked, err := s.fileLock.TryLock()
	if err != nil {
		return fmt.Errorf("failed to acquire file lock: %w", err)
	}
	if !locked {
		return fmt.Errorf("failed to acquire file lock: already locked")
	}
	defer s.fileLock.Unlock()

	// Marshal data to JSON with indentation for readability
	jsonData, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write to a temporary file in the same directory
	// This ensures atomic writes (rename is atomic on most filesystems)
	dir := filepath.Dir(s.filePath)
	tempFile, err := os.CreateTemp(dir, ".cline-storage-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()

	// Clean up temp file on error
	defer func() {
		if err != nil {
			os.Remove(tempPath)
		}
	}()

	// Write JSON data to temp file
	if _, err := tempFile.Write(jsonData); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	// Ensure data is written to disk
	if err := tempFile.Sync(); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Set appropriate permissions on temp file before rename
	if err := os.Chmod(tempPath, s.permissions); err != nil {
		return fmt.Errorf("failed to set permissions on temp file: %w", err)
	}

	// Atomic rename: this is the critical operation that makes writes atomic
	if err := os.Rename(tempPath, s.filePath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	// Sync the directory to ensure the rename is persisted
	dirFile, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("failed to open directory for sync: %w", err)
	}
	if err := dirFile.Sync(); err != nil {
		dirFile.Close()
		return fmt.Errorf("failed to sync directory: %w", err)
	}
	dirFile.Close()

	s.dirty = false

	return nil
}

// FilePath returns the path to the storage file.
func (s *ClineFileStorage) FilePath() string {
	return s.filePath
}

// IsDirty returns true if there are unflushed changes.
func (s *ClineFileStorage) IsDirty() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dirty
}

// Reload reloads the data from disk, discarding any in-memory changes.
func (s *ClineFileStorage) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("storage is closed")
	}

	return s.load()
}