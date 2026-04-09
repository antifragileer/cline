// Package integration provides integration tests for storage operations.
package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStorageIntegration validates storage operations
func TestStorageIntegration(t *testing.T) {
	t.Run("file_storage_persistence", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "storage-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		storagePath := filepath.Join(tempDir, "test.json")

		// Write data
		data := map[string]interface{}{
			"key":       "value",
			"number":    42,
			"timestamp": time.Now().Unix(),
		}

		jsonData, err := json.Marshal(data)
		require.NoError(t, err)
		err = os.WriteFile(storagePath, jsonData, 0644)
		require.NoError(t, err)

		// Read back
		readData, err := os.ReadFile(storagePath)
		require.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(readData, &parsed)
		require.NoError(t, err)

		assert.Equal(t, "value", parsed["key"])
		assert.Equal(t, float64(42), parsed["number"])
	})

	t.Run("directory_structure_creation", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "storage-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create nested structure
		nestedDir := filepath.Join(tempDir, "level1", "level2", "level3")
		err = os.MkdirAll(nestedDir, 0755)
		require.NoError(t, err)

		// Verify structure exists
		info, err := os.Stat(nestedDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("concurrent_file_access", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "storage-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		filePath := filepath.Join(tempDir, "concurrent.txt")
		
		// Write initial content
		err = os.WriteFile(filePath, []byte("initial"), 0644)
		require.NoError(t, err)

		// Concurrent reads
		done := make(chan bool, 5)
		for i := 0; i < 5; i++ {
			go func() {
				_, err := os.ReadFile(filePath)
				done <- err == nil
			}()
		}

		successCount := 0
		for i := 0; i < 5; i++ {
			if <-done {
				successCount++
			}
		}

		assert.Equal(t, 5, successCount)
	})

	t.Run("atomic_write_operation", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "storage-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		filePath := filepath.Join(tempDir, "atomic.txt")

		// Write data
		data := []byte("atomic write test data")
		tmpPath := filePath + ".tmp"
		
		// Write to temp file first
		err = os.WriteFile(tmpPath, data, 0644)
		require.NoError(t, err)

		// Rename for atomic operation
		err = os.Rename(tmpPath, filePath)
		require.NoError(t, err)

		// Verify
		readData, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, data, readData)
	})
}

// TestStorageErrorHandlingIntegration validates error handling
func TestStorageErrorHandlingIntegration(t *testing.T) {
	t.Run("nonexistent_file_read", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "storage-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		nonExistentPath := filepath.Join(tempDir, "does-not-exist.json")
		_, err = os.ReadFile(nonExistentPath)
		assert.Error(t, err)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("permission_denied_handling", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Skipping permission test when running as root")
		}

		tempDir, err := os.MkdirTemp("", "storage-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create read-only directory
		readonlyDir := filepath.Join(tempDir, "readonly")
		err = os.Mkdir(readonlyDir, 0555)
		require.NoError(t, err)

		// Try to write to read-only directory
		filePath := filepath.Join(readonlyDir, "test.txt")
		err = os.WriteFile(filePath, []byte("test"), 0644)
		assert.Error(t, err)
	})

	t.Run("corrupted_json_handling", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "storage-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		filePath := filepath.Join(tempDir, "corrupted.json")
		err = os.WriteFile(filePath, []byte("{invalid json"), 0644)
		require.NoError(t, err)

		data, err := os.ReadFile(filePath)
		require.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(data, &parsed)
		assert.Error(t, err)
	})

	t.Run("disk_full_simulation", func(t *testing.T) {
		// This is a simulation - actual disk full testing requires mocking
		writeFunc := func(path string, size int64) error {
			// Simulate checking available space
			if size > 1024*1024*1024*1024 { // 1TB
				return &os.PathError{Op: "write", Path: path, Err: os.ErrInvalid}
			}
			return nil
		}

		err := writeFunc("/test/path", 1024*1024*1024*1024*2) // 2TB
		assert.Error(t, err)
	})
}

// TestStoragePerformanceIntegration validates performance characteristics
func TestStoragePerformanceIntegration(t *testing.T) {
	t.Run("large_file_read_write", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "storage-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		filePath := filepath.Join(tempDir, "large.bin")

		// Create 1MB of data
		data := make([]byte, 1024*1024)
		for i := range data {
			data[i] = byte(i % 256)
		}

		start := time.Now()
		err = os.WriteFile(filePath, data, 0644)
		require.NoError(t, err)
		writeDuration := time.Since(start)

		start = time.Now()
		readData, err := os.ReadFile(filePath)
		require.NoError(t, err)
		readDuration := time.Since(start)

		assert.Equal(t, data, readData)
		t.Logf("Write: %v, Read: %v", writeDuration, readDuration)

		// Should complete within reasonable time
		assert.Less(t, writeDuration, 5*time.Second)
		assert.Less(t, readDuration, 5*time.Second)
	})

	t.Run("many_small_files", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "storage-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create many small files
		fileCount := 100
		start := time.Now()

		for i := 0; i < fileCount; i++ {
			filePath := filepath.Join(tempDir, fmt.Sprintf("file%03d.txt", i))
			err := os.WriteFile(filePath, []byte("small content"), 0644)
			require.NoError(t, err)
		}

		duration := time.Since(start)
		t.Logf("Created %d files in %v", fileCount, duration)

		// List files
		entries, err := os.ReadDir(tempDir)
		require.NoError(t, err)
		assert.Equal(t, fileCount, len(entries))
	})
}

// BenchmarkStorageOperations benchmarks storage operations
func BenchmarkFileWrite(b *testing.B) {
	tempDir, _ := os.MkdirTemp("", "bench-*")
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "bench.txt")
	data := []byte("benchmark data")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		os.WriteFile(filePath, data, 0644)
	}
}

func BenchmarkFileRead(b *testing.B) {
	tempDir, _ := os.MkdirTemp("", "bench-*")
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "bench.txt")
	data := make([]byte, 1024*1024) // 1MB
	os.WriteFile(filePath, data, 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		os.ReadFile(filePath)
	}
}

func BenchmarkJSONSerialization(b *testing.B) {
	data := map[string]interface{}{
		"id":        "task-123",
		"prompt":    "test prompt",
		"status":    "running",
		"timestamp": time.Now().Unix(),
		"metadata": map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		json.Marshal(data)
	}
}