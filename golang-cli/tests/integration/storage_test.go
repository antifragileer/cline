// Package integration provides integration tests for the Go CLI.
// This package tests storage implementations.
package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/cline/cline/golang-cli/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFileStorage tests the file storage implementation
func TestFileStorage(t *testing.T) {
	t.Run("creates storage file on first write", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "storage-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		filePath := filepath.Join(tempDir, "test.json")
		store, err := storage.NewClineFileStorage(filePath, 0644)
		require.NoError(t, err)

		// File is not created until first write
		_, err = os.Stat(filePath)
		assert.True(t, os.IsNotExist(err), "Storage file should not exist until first write")

		// Write some data
		err = store.Set("key1", "value1")
		require.NoError(t, err)

		// Now file should exist
		_, err = os.Stat(filePath)
		assert.NoError(t, err, "Storage file should exist after first write")
	})

	t.Run("sets and gets values", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "storage-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		filePath := filepath.Join(tempDir, "test.json")
		store, err := storage.NewClineFileStorage(filePath, 0644)
		require.NoError(t, err)

		// Set a value
		err = store.Set("key1", "value1")
		require.NoError(t, err)

		// Get the value
		val, ok := store.Get("key1")
		require.True(t, ok, "Key should exist")
		assert.Equal(t, "value1", val)
	})

	t.Run("updates existing values", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "storage-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		filePath := filepath.Join(tempDir, "test.json")
		store, err := storage.NewClineFileStorage(filePath, 0644)
		require.NoError(t, err)

		// Set initial value
		err = store.Set("key1", "value1")
		require.NoError(t, err)

		// Update value
		err = store.Set("key1", "value2")
		require.NoError(t, err)

		// Verify update
		val, ok := store.Get("key1")
		require.True(t, ok, "Key should exist")
		assert.Equal(t, "value2", val)
	})

	t.Run("returns false for non-existent key", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "storage-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		filePath := filepath.Join(tempDir, "test.json")
		store, err := storage.NewClineFileStorage(filePath, 0644)
		require.NoError(t, err)

		_, ok := store.Get("nonexistent")
		assert.False(t, ok, "Non-existent key should return false")
	})

	t.Run("deletes values", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "storage-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		filePath := filepath.Join(tempDir, "test.json")
		store, err := storage.NewClineFileStorage(filePath, 0644)
		require.NoError(t, err)

		// Set and delete
		err = store.Set("key1", "value1")
		require.NoError(t, err)

		err = store.Delete("key1")
		require.NoError(t, err)

		// Verify deletion
		_, ok := store.Get("key1")
		assert.False(t, ok, "Deleted key should not exist")
	})

	t.Run("stores complex types", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "storage-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		filePath := filepath.Join(tempDir, "test.json")
		store, err := storage.NewClineFileStorage(filePath, 0644)
		require.NoError(t, err)

		// Store a complex struct
		type TestStruct struct {
			Name    string   `json:"name"`
			Count   int      `json:"count"`
			Enabled bool     `json:"enabled"`
			Tags    []string `json:"tags"`
		}

		data := TestStruct{
			Name:    "Test",
			Count:   42,
			Enabled: true,
			Tags:    []string{"a", "b", "c"},
		}

		err = store.Set("struct", data)
		require.NoError(t, err)

		// Retrieve and verify
		val, ok := store.Get("struct")
		require.True(t, ok, "Key should exist")

		// The value should be stored as JSON, so we need to unmarshal
		var result TestStruct
		switch v := val.(type) {
		case map[string]interface{}:
			jsonData, _ := json.Marshal(v)
			json.Unmarshal(jsonData, &result)
		case TestStruct:
			result = v
		}

		assert.Equal(t, "Test", result.Name)
		assert.Equal(t, 42, result.Count)
	})

	t.Run("persists across instances", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "storage-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		filePath := filepath.Join(tempDir, "test.json")

		// First instance
		store1, err := storage.NewClineFileStorage(filePath, 0644)
		require.NoError(t, err)
		err = store1.Set("key1", "persisted_value")
		require.NoError(t, err)
		store1.Close()

		// Second instance
		store2, err := storage.NewClineFileStorage(filePath, 0644)
		require.NoError(t, err)
		defer store2.Close()

		val, ok := store2.Get("key1")
		require.True(t, ok, "Key should exist")
		assert.Equal(t, "persisted_value", val)
	})

	t.Run("handles concurrent access", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "storage-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		filePath := filepath.Join(tempDir, "test.json")
		store, err := storage.NewClineFileStorage(filePath, 0644)
		require.NoError(t, err)
		defer store.Close()

		// Concurrent writes
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func(n int) {
				key := fmt.Sprintf("key%d", n)
				val := fmt.Sprintf("value%d", n)
				err := store.Set(key, val)
				done <- err == nil
			}(i)
		}

		// Wait for all goroutines
		successCount := 0
		for i := 0; i < 10; i++ {
			if <-done {
				successCount++
			}
		}

		assert.Equal(t, 10, successCount, "All concurrent writes should succeed")

		// Verify all values
		for i := 0; i < 10; i++ {
			key := fmt.Sprintf("key%d", i)
			expected := fmt.Sprintf("value%d", i)
			val, ok := store.Get(key)
			require.True(t, ok, "Key %s should exist", key)
			assert.Equal(t, expected, val)
		}
	})
}

// TestSecretsStorage tests the secrets storage implementation
func TestSecretsStorage(t *testing.T) {
	t.Run("stores and retrieves secrets", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "secrets-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		filePath := filepath.Join(tempDir, "secrets.json")
		store, err := storage.NewClineFileStorage(filePath, 0600)
		require.NoError(t, err)
		defer store.Close()

		// Store a secret
		secret := "super-secret-api-key-12345"
		err = store.Set("api_key", secret)
		require.NoError(t, err)

		// Retrieve
		val, ok := store.Get("api_key")
		require.True(t, ok, "Key should exist")
		assert.Equal(t, secret, val)
	})

	t.Run("uses restricted permissions", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "secrets-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		filePath := filepath.Join(tempDir, "secrets.json")
		store, err := storage.NewClineFileStorage(filePath, 0600)
		require.NoError(t, err)

		err = store.Set("key", "value")
		require.NoError(t, err)
		store.Close()

		// Check file permissions
		info, err := os.Stat(filePath)
		require.NoError(t, err)
		// Note: On Windows, permissions may not be exactly 0600
		mode := info.Mode().Perm()
		assert.Equal(t, os.FileMode(0600), mode, "Secret file should have 0600 permissions")
	})
}

// TestStorageMigration tests storage migration scenarios
func TestStorageMigration(t *testing.T) {
	t.Run("migrates from old format", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "migration-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		filePath := filepath.Join(tempDir, "storage.json")

		// Create old format data
		oldData := map[string]interface{}{
			"old_key": "old_value",
		}
		data, _ := json.Marshal(oldData)
		err = os.WriteFile(filePath, data, 0644)
		require.NoError(t, err)

		// Initialize storage
		store, err := storage.NewClineFileStorage(filePath, 0644)
		require.NoError(t, err)
		defer store.Close()

		// Verify old data is preserved
		val, ok := store.Get("old_key")
		require.True(t, ok, "Old key should exist")
		assert.Equal(t, "old_value", val)
	})

	t.Run("handles corrupted files", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "migration-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		filePath := filepath.Join(tempDir, "storage.json")

		// Write corrupted data
		err = os.WriteFile(filePath, []byte("not valid json"), 0644)
		require.NoError(t, err)

		// Initialize should handle this gracefully - currently returns error
		_, err = storage.NewClineFileStorage(filePath, 0644)
		// The current implementation returns an error for corrupted files
		// This is acceptable behavior
		assert.Error(t, err, "Should return error for corrupted file")
	})
}

// TestStoragePerformance benchmarks storage operations
func BenchmarkStorageSet(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "bench-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "bench.json")
	store, err := storage.NewClineFileStorage(filePath, 0644)
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i)
		val := fmt.Sprintf("value%d", i)
		if err := store.Set(key, val); err != nil {
			b.Errorf("Set failed: %v", err)
		}
	}
}

func BenchmarkStorageGet(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "bench-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "bench.json")
	store, err := storage.NewClineFileStorage(filePath, 0644)
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key%d", i)
		val := fmt.Sprintf("value%d", i)
		store.Set(key, val)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i%1000)
		if _, ok := store.Get(key); !ok {
			b.Errorf("Get failed for key: %s", key)
		}
	}
}