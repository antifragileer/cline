// Package storage provides file-based JSON storage for Cline CLI.
// This package implements atomic file operations with file locking and
// thread-safe read/write operations for persistent state management.
package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// FileStorage defines the interface for key-value storage operations.
// This interface is implemented by ClineFileStorage and provides
// a common contract for file-based JSON storage implementations.
type FileStorage interface {
	// Get retrieves a value by key. Returns the value and true if found,
	// or nil and false if not found.
	Get(key string) (interface{}, bool)

	// Set stores a value for the given key.
	Set(key string, value interface{}) error

	// SetBatch stores multiple key-value pairs atomically.
	SetBatch(pairs map[string]interface{}) error

	// Delete removes a key from storage.
	Delete(key string) error

	// GetAll returns all key-value pairs as a map.
	GetAll() map[string]interface{}

	// Close releases any resources held by the storage.
	Close() error
}

// StorageType represents the type of storage file.
type StorageType string

const (
	// GlobalStateType is for global settings and state.
	GlobalStateType StorageType = "globalState"
	// SecretsType is for sensitive data like API keys.
	SecretsType StorageType = "secrets"
	// WorkspaceStateType is for per-workspace settings.
	WorkspaceStateType StorageType = "workspaceState"
)

// StorageConfig holds configuration for storage initialization.
type StorageConfig struct {
	// Type is the kind of storage (globalState, secrets, workspaceState).
	Type StorageType

	// Dir is the base directory for storage files.
	// Defaults to ~/.cline/data
	Dir string

	// WorkspaceHash is the workspace identifier for workspace state.
	// Required when Type is WorkspaceStateType.
	WorkspaceHash string

	// Permissions for the storage file.
	// Defaults to 0644 for state, 0600 for secrets.
	Permissions os.FileMode
}

// StorageContext holds all storage instances for the application.
// It uses ClineFileStorage for all storage types to provide atomic
// file operations with file locking.
type StorageContext struct {
	GlobalState    *ClineFileStorage
	Secrets        *ClineFileStorage
	WorkspaceState *ClineFileStorage

	// mu protects the storage instances
	mu sync.RWMutex
}

// NewStorageContext creates a new StorageContext with initialized storage instances.
func NewStorageContext(baseDir string, workspaceHash string) (*StorageContext, error) {
	if baseDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		baseDir = filepath.Join(homeDir, ".cline", "data")
	}

	ctx := &StorageContext{}

	// Initialize global state storage
	globalStatePath := filepath.Join(baseDir, "globalState.json")
	globalStorage, err := NewClineFileStorage(globalStatePath, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create global state storage: %w", err)
	}
	ctx.GlobalState = globalStorage

	// Initialize secrets storage
	secretsPath := filepath.Join(baseDir, "secrets.json")
	secretsStorage, err := NewClineFileStorage(secretsPath, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to create secrets storage: %w", err)
	}
	ctx.Secrets = secretsStorage

	// Initialize workspace state storage
	if workspaceHash != "" {
		workspaceDir := filepath.Join(baseDir, "workspaces", workspaceHash)
		workspacePath := filepath.Join(workspaceDir, "workspaceState.json")
		workspaceStorage, err := NewClineFileStorage(workspacePath, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to create workspace state storage: %w", err)
		}
		ctx.WorkspaceState = workspaceStorage
	}

	return ctx, nil
}

// Close closes all storage instances in the context.
func (ctx *StorageContext) Close() error {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	var errs []string

	if ctx.GlobalState != nil {
		if err := ctx.GlobalState.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("global state: %v", err))
		}
	}

	if ctx.Secrets != nil {
		if err := ctx.Secrets.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("secrets: %v", err))
		}
	}

	if ctx.WorkspaceState != nil {
		if err := ctx.WorkspaceState.Close(); err != nil {
			errs = append(errs, fmt.Sprintf("workspace state: %v", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing storage: %s", strings.Join(errs, "; "))
	}

	return nil
}

// GetWorkspaceHash generates a hash for a workspace path.
// This is a simple implementation using the absolute path.
func GetWorkspaceHash(workspacePath string) (string, error) {
	absPath, err := filepath.Abs(workspacePath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}
	
	// Simple hash: replace path separators and encode
	// In production, this could use a proper hash function
	hash := strings.ReplaceAll(absPath, string(filepath.Separator), "_")
	hash = strings.ReplaceAll(hash, ":", "_") // For Windows
	
	// Sanitize for filesystem
	hash = sanitizeFilename(hash)
	
	return hash, nil
}

// sanitizeFilename removes or replaces characters that are invalid in filenames.
func sanitizeFilename(name string) string {
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := name
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}
	return result
}

// CloneValue creates a deep copy of a value that can be safely modified
// without affecting the stored value.
func CloneValue(v interface{}) (interface{}, error) {
	// Serialize to JSON and back to create a deep copy
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal value: %w", err)
	}
	
	var result interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal value: %w", err)
	}
	
	return result, nil
}

// GetTyped retrieves a value and unmarshals it into the provided type.
func GetTyped(s FileStorage, key string, dest interface{}) (bool, error) {
	val, ok := s.Get(key)
	if !ok {
		return false, nil
	}
	
	// Marshal and unmarshal to convert types
	data, err := json.Marshal(val)
	if err != nil {
		return false, fmt.Errorf("failed to marshal value: %w", err)
	}
	
	if err := json.Unmarshal(data, dest); err != nil {
		return false, fmt.Errorf("failed to unmarshal value: %w", err)
	}
	
	return true, nil
}