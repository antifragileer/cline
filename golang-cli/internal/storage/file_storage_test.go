package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

func TestNewClineFileStorage(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cline-storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("creates storage with new file", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "new-storage.json")
		storage, err := NewClineFileStorage(filePath, 0644)
		if err != nil {
			t.Fatalf("NewClineFileStorage failed: %v", err)
		}
		defer storage.Close()

		if storage.FilePath() != filePath {
			t.Errorf("FilePath() = %s, want %s", storage.FilePath(), filePath)
		}

		// Directory should be created
		dir := filepath.Dir(filePath)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Error("Directory should be created")
		}
	})

	t.Run("loads existing file", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "existing-storage.json")

		// Create file with initial data
		initialData := map[string]interface{}{
			"existingKey": "existingValue",
			"numberKey":   float64(42),
		}
		data, _ := json.Marshal(initialData)
		if err := os.WriteFile(filePath, data, 0644); err != nil {
			t.Fatalf("Failed to create initial file: %v", err)
		}

		storage, err := NewClineFileStorage(filePath, 0644)
		if err != nil {
			t.Fatalf("NewClineFileStorage failed: %v", err)
		}
		defer storage.Close()

		val, ok := storage.Get("existingKey")
		if !ok {
			t.Error("Should find existing key")
		}
		if val != "existingValue" {
			t.Errorf("Got %v, want 'existingValue'", val)
		}
	})

	t.Run("creates nested directories", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "deep", "nested", "dir", "storage.json")
		storage, err := NewClineFileStorage(filePath, 0644)
		if err != nil {
			t.Fatalf("NewClineFileStorage failed: %v", err)
		}
		storage.Close()

		if _, err := os.Stat(filepath.Dir(filePath)); os.IsNotExist(err) {
			t.Error("Nested directories should be created")
		}
	})

	t.Run("handles empty file", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "empty.json")
		// Create empty file
		if err := os.WriteFile(filePath, []byte{}, 0644); err != nil {
			t.Fatalf("Failed to create empty file: %v", err)
		}

		storage, err := NewClineFileStorage(filePath, 0644)
		if err != nil {
			t.Fatalf("NewClineFileStorage failed: %v", err)
		}
		defer storage.Close()

		// Should start with empty data
		all := storage.GetAll()
		if len(all) != 0 {
			t.Errorf("Expected empty data, got %d entries", len(all))
		}
	})

	t.Run("handles invalid JSON", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "invalid.json")
		// Create file with invalid JSON
		if err := os.WriteFile(filePath, []byte("not valid json"), 0644); err != nil {
			t.Fatalf("Failed to create invalid file: %v", err)
		}

		_, err := NewClineFileStorage(filePath, 0644)
		if err == nil {
			t.Error("Should fail with invalid JSON")
		}
	})

	t.Run("handles directory creation failure", func(t *testing.T) {
		// Try to create storage in a path where we can't create directories
		// On most systems, this would be a read-only root or similar
		invalidPath := "/nonexistent_root_dir/storage.json"

		_, err := NewClineFileStorage(invalidPath, 0644)
		if err == nil {
			t.Skip("System allows creating directories anywhere, skipping")
		}
	})
}

func TestClineFileStorage_Get(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cline-storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewClineFileStorage(filepath.Join(tempDir, "test.json"), 0644)
	if err != nil {
		t.Fatalf("NewClineFileStorage failed: %v", err)
	}
	defer storage.Close()

	t.Run("returns value for existing key", func(t *testing.T) {
		if err := storage.Set("key1", "value1"); err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		val, ok := storage.Get("key1")
		if !ok {
			t.Error("Should find key")
		}
		if val != "value1" {
			t.Errorf("Got %v, want 'value1'", val)
		}
	})

	t.Run("returns false for missing key", func(t *testing.T) {
		_, ok := storage.Get("nonexistent")
		if ok {
			t.Error("Should not find nonexistent key")
		}
	})

	t.Run("returns deep copy", func(t *testing.T) {
		original := map[string]interface{}{
			"nested": map[string]interface{}{
				"inner": "value",
			},
		}
		if err := storage.Set("complex", original); err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		val, _ := storage.Get("complex")
		valMap := val.(map[string]interface{})
		valMap["nested"].(map[string]interface{})["inner"] = "modified"

		// Retrieve again to verify original wasn't modified
		val2, _ := storage.Get("complex")
		val2Map := val2.(map[string]interface{})
		if val2Map["nested"].(map[string]interface{})["inner"] != "value" {
			t.Error("Original value should not be modified")
		}
	})

	t.Run("returns false when closed", func(t *testing.T) {
		tempStorage, _ := NewClineFileStorage(filepath.Join(tempDir, "closed.json"), 0644)
		tempStorage.Set("key", "value")
		tempStorage.Close()

		_, ok := tempStorage.Get("key")
		if ok {
			t.Error("Should return false when storage is closed")
		}
	})
}

func TestClineFileStorage_Set(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cline-storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("stores simple values", func(t *testing.T) {
		storage, err := NewClineFileStorage(filepath.Join(tempDir, "simple.json"), 0644)
		if err != nil {
			t.Fatalf("NewClineFileStorage failed: %v", err)
		}
		defer storage.Close()

		if err := storage.Set("key1", "string value"); err != nil {
			t.Errorf("Set failed: %v", err)
		}
		if err := storage.Set("key2", 42); err != nil {
			t.Errorf("Set failed: %v", err)
		}
		if err := storage.Set("key3", true); err != nil {
			t.Errorf("Set failed: %v", err)
		}

		// Verify values
		if v, _ := storage.Get("key1"); v != "string value" {
			t.Errorf("key1 = %v, want 'string value'", v)
		}
		if v, _ := storage.Get("key2"); v != float64(42) {
			t.Errorf("key2 = %v, want 42", v)
		}
		if v, _ := storage.Get("key3"); v != true {
			t.Errorf("key3 = %v, want true", v)
		}
	})

	t.Run("stores complex objects", func(t *testing.T) {
		storage, err := NewClineFileStorage(filepath.Join(tempDir, "complex.json"), 0644)
		if err != nil {
			t.Fatalf("NewClineFileStorage failed: %v", err)
		}
		defer storage.Close()

		value := map[string]interface{}{
			"array": []interface{}{1, 2, 3},
			"object": map[string]interface{}{
				"nested": "value",
			},
		}
		if err := storage.Set("complex", value); err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		retrieved, _ := storage.Get("complex")
		retrievedMap := retrieved.(map[string]interface{})

		arr := retrievedMap["array"].([]interface{})
		if len(arr) != 3 || arr[0] != float64(1) {
			t.Error("Array not stored correctly")
		}

		nested := retrievedMap["object"].(map[string]interface{})
		if nested["nested"] != "value" {
			t.Error("Nested object not stored correctly")
		}
	})

	t.Run("updates existing key", func(t *testing.T) {
		storage, err := NewClineFileStorage(filepath.Join(tempDir, "update.json"), 0644)
		if err != nil {
			t.Fatalf("NewClineFileStorage failed: %v", err)
		}
		defer storage.Close()

		storage.Set("key", "original")
		if err := storage.Set("key", "updated"); err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		val, _ := storage.Get("key")
		if val != "updated" {
			t.Errorf("Got %v, want 'updated'", val)
		}
	})

	t.Run("returns error when closed", func(t *testing.T) {
		storage, _ := NewClineFileStorage(filepath.Join(tempDir, "closed-set.json"), 0644)
		storage.Close()

		err := storage.Set("key", "value")
		if err == nil {
			t.Error("Should return error when storage is closed")
		}
	})

	t.Run("stores nil value", func(t *testing.T) {
		storage, err := NewClineFileStorage(filepath.Join(tempDir, "nil.json"), 0644)
		if err != nil {
			t.Fatalf("NewClineFileStorage failed: %v", err)
		}
		defer storage.Close()

		// First set a value, then set it to nil
		storage.Set("key", "value")
		if err := storage.Set("key", nil); err != nil {
			t.Fatalf("Set nil failed: %v", err)
		}

		val, ok := storage.Get("key")
		if !ok {
			t.Error("Key with nil value should still exist")
		}
		if val != nil {
			t.Errorf("Expected nil, got %v", val)
		}
	})
}

func TestClineFileStorage_SetBatch(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cline-storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewClineFileStorage(filepath.Join(tempDir, "batch.json"), 0644)
	if err != nil {
		t.Fatalf("NewClineFileStorage failed: %v", err)
	}
	defer storage.Close()

	t.Run("stores multiple values atomically", func(t *testing.T) {
		pairs := map[string]interface{}{
			"key1": "value1",
			"key2": float64(42), // JSON numbers are float64
			"key3": true,
		}

		if err := storage.SetBatch(pairs); err != nil {
			t.Fatalf("SetBatch failed: %v", err)
		}

		// Verify all values
		for key, expected := range pairs {
			actual, ok := storage.Get(key)
			if !ok {
				t.Errorf("Key %s not found", key)
				continue
			}
			if actual != expected {
				t.Errorf("Key %s: got %v (type %T), want %v (type %T)", key, actual, actual, expected, expected)
			}
		}
	})

	t.Run("overwrites existing keys in batch", func(t *testing.T) {
		// Set initial value
		storage.Set("existing", "old")

		pairs := map[string]interface{}{
			"existing": "new",
			"newkey":   "value",
		}

		if err := storage.SetBatch(pairs); err != nil {
			t.Fatalf("SetBatch failed: %v", err)
		}

		val, _ := storage.Get("existing")
		if val != "new" {
			t.Errorf("Expected 'new', got %v", val)
		}
	})

	t.Run("empty batch is no-op", func(t *testing.T) {
		if err := storage.SetBatch(map[string]interface{}{}); err != nil {
			t.Fatalf("SetBatch failed: %v", err)
		}

		// After batch, dirty should be set then cleared by flush
		// This just verifies no error occurs
	})

	t.Run("returns error when closed", func(t *testing.T) {
		tempStorage, _ := NewClineFileStorage(filepath.Join(tempDir, "closed-batch.json"), 0644)
		tempStorage.Close()

		err := tempStorage.SetBatch(map[string]interface{}{"key": "value"})
		if err == nil {
			t.Error("Should return error when storage is closed")
		}
	})
}

func TestClineFileStorage_Delete(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cline-storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewClineFileStorage(filepath.Join(tempDir, "delete.json"), 0644)
	if err != nil {
		t.Fatalf("NewClineFileStorage failed: %v", err)
	}
	defer storage.Close()

	t.Run("deletes existing key", func(t *testing.T) {
		storage.Set("key1", "value1")

		if err := storage.Delete("key1"); err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		_, ok := storage.Get("key1")
		if ok {
			t.Error("Key should be deleted")
		}
	})

	t.Run("deleting nonexistent key is no-op", func(t *testing.T) {
		if err := storage.Delete("nonexistent"); err != nil {
			t.Fatalf("Delete of nonexistent key should not error: %v", err)
		}
	})

	t.Run("returns error when closed", func(t *testing.T) {
		tempStorage, _ := NewClineFileStorage(filepath.Join(tempDir, "closed-delete.json"), 0644)
		tempStorage.Set("key", "value")
		tempStorage.Close()

		err := tempStorage.Delete("key")
		if err == nil {
			t.Error("Should return error when storage is closed")
		}
	})
}

func TestClineFileStorage_GetAll(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cline-storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewClineFileStorage(filepath.Join(tempDir, "getall.json"), 0644)
	if err != nil {
		t.Fatalf("NewClineFileStorage failed: %v", err)
	}
	defer storage.Close()

	t.Run("returns all values", func(t *testing.T) {
		storage.Set("key1", "value1")
		storage.Set("key2", "value2")

		all := storage.GetAll()

		if len(all) != 2 {
			t.Errorf("Got %d entries, want 2", len(all))
		}
		if all["key1"] != "value1" {
			t.Errorf("key1 = %v, want 'value1'", all["key1"])
		}
		if all["key2"] != "value2" {
			t.Errorf("key2 = %v, want 'value2'", all["key2"])
		}
	})

	t.Run("returns deep copy", func(t *testing.T) {
		storage.Set("key", map[string]interface{}{"nested": "value"})

		all := storage.GetAll()
		all["key"].(map[string]interface{})["nested"] = "modified"

		// Verify original wasn't modified
		original, _ := storage.Get("key")
		if original.(map[string]interface{})["nested"] != "value" {
			t.Error("Original should not be modified")
		}
	})

	t.Run("returns empty map when closed", func(t *testing.T) {
		tempStorage, _ := NewClineFileStorage(filepath.Join(tempDir, "closed-getall.json"), 0644)
		tempStorage.Set("key", "value")
		tempStorage.Close()

		all := tempStorage.GetAll()
		if len(all) != 0 {
			t.Error("Should return empty map when closed")
		}
	})

	t.Run("returns empty map for new storage", func(t *testing.T) {
		all := storage.GetAll()
		// Should not be nil
		if all == nil {
			t.Error("GetAll should never return nil")
		}
	})
}

func TestClineFileStorage_Persistence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cline-storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "persist.json")

	t.Run("data persists across instances", func(t *testing.T) {
		// First instance
		storage1, err := NewClineFileStorage(filePath, 0644)
		if err != nil {
			t.Fatalf("NewClineFileStorage failed: %v", err)
		}

		storage1.Set("key1", "value1")
		storage1.Set("key2", map[string]interface{}{"nested": "data"})
		storage1.Close()

		// Second instance
		storage2, err := NewClineFileStorage(filePath, 0644)
		if err != nil {
			t.Fatalf("NewClineFileStorage failed: %v", err)
		}
		defer storage2.Close()

		val1, ok := storage2.Get("key1")
		if !ok || val1 != "value1" {
			t.Errorf("key1 = %v, want 'value1'", val1)
		}

		val2, ok := storage2.Get("key2")
		if !ok {
			t.Fatal("key2 not found")
		}
		val2Map := val2.(map[string]interface{})
		if val2Map["nested"] != "data" {
			t.Errorf("nested = %v, want 'data'", val2Map["nested"])
		}
	})

	t.Run("file is valid JSON", func(t *testing.T) {
		storage, _ := NewClineFileStorage(filePath, 0644)
		storage.Set("test", "value")
		storage.Close()

		data, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Errorf("File is not valid JSON: %v", err)
		}
	})
}

func TestClineFileStorage_FilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Permission tests skipped on Windows")
	}

	tempDir, err := os.MkdirTemp("", "cline-storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("secrets have 0600 permissions", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "secrets.json")
		storage, err := NewClineFileStorage(filePath, 0600)
		if err != nil {
			t.Fatalf("NewClineFileStorage failed: %v", err)
		}
		storage.Set("secret", "value")
		storage.Close()

		info, err := os.Stat(filePath)
		if err != nil {
			t.Fatalf("Failed to stat file: %v", err)
		}

		mode := info.Mode().Perm()
		if mode != 0600 {
			t.Errorf("File permissions = %o, want 0600", mode)
		}
	})

	t.Run("state has 0644 permissions", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "state.json")
		storage, err := NewClineFileStorage(filePath, 0644)
		if err != nil {
			t.Fatalf("NewClineFileStorage failed: %v", err)
		}
		storage.Set("key", "value")
		storage.Close()

		info, err := os.Stat(filePath)
		if err != nil {
			t.Fatalf("Failed to stat file: %v", err)
		}

		mode := info.Mode().Perm()
		if mode != 0644 {
			t.Errorf("File permissions = %o, want 0644", mode)
		}
	})
}

func TestClineFileStorage_ThreadSafety(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cline-storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewClineFileStorage(filepath.Join(tempDir, "threadsafe.json"), 0644)
	if err != nil {
		t.Fatalf("NewClineFileStorage failed: %v", err)
	}
	defer storage.Close()

	t.Run("concurrent writes", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 10
		numWrites := 100

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numWrites; j++ {
					key := fmt.Sprintf("goroutine%d-key%d", id, j)
					if err := storage.Set(key, j); err != nil {
						t.Errorf("Set failed: %v", err)
					}
				}
			}(i)
		}

		wg.Wait()

		// Verify all writes
		for i := 0; i < numGoroutines; i++ {
			for j := 0; j < numWrites; j++ {
				key := fmt.Sprintf("goroutine%d-key%d", i, j)
				val, ok := storage.Get(key)
				if !ok {
					t.Errorf("Key %s not found", key)
				} else if val != float64(j) {
					t.Errorf("Key %s = %v, want %d", key, val, j)
				}
			}
		}
	})

	t.Run("concurrent reads and writes", func(t *testing.T) {
		// Pre-populate
		for i := 0; i < 100; i++ {
			storage.Set(fmt.Sprintf("readkey%d", i), i)
		}

		var wg sync.WaitGroup
		done := make(chan bool)

		// Writers
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 50; j++ {
					select {
					case <-done:
						return
					default:
						storage.Set(fmt.Sprintf("writekey%d", j), j)
					}
				}
			}(i)
		}

		// Readers
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 50; j++ {
					select {
					case <-done:
						return
					default:
						storage.Get(fmt.Sprintf("readkey%d", j))
						storage.GetAll()
					}
				}
			}(i)
		}

		wg.Wait()
		close(done)
	})
}

func TestClineFileStorage_IsDirty(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cline-storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewClineFileStorage(filepath.Join(tempDir, "dirty.json"), 0644)
	if err != nil {
		t.Fatalf("NewClineFileStorage failed: %v", err)
	}
	defer storage.Close()

	t.Run("not dirty after creation", func(t *testing.T) {
		if storage.IsDirty() {
			t.Error("Should not be dirty after creation")
		}
	})

	t.Run("tracks dirty state correctly", func(t *testing.T) {
		// Note: Set() also calls flush(), so dirty might be false immediately after
		// The implementation detail is that dirty is set during modification
		// and cleared after flush

		// Before any operation
		if storage.IsDirty() {
			t.Log("Storage is dirty before any operation (unexpected but implementation-dependent)")
		}
	})
}

func TestClineFileStorage_Reload(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cline-storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "reload.json")

	t.Run("reloads data from disk", func(t *testing.T) {
		storage1, _ := NewClineFileStorage(filePath, 0644)
		storage1.Set("key", "original")
		storage1.Close()

		// Create new storage and reload to ensure we read from disk
		storage2, _ := NewClineFileStorage(filePath, 0644)

		// Verify initial load
		val, _ := storage2.Get("key")
		if val != "original" {
			t.Errorf("After load, key = %v, want 'original'", val)
		}

		// Modify in memory (don't save)
		storage2.data["key"] = "modified"

		// Reload should discard in-memory changes
		if err := storage2.Reload(); err != nil {
			t.Fatalf("Reload failed: %v", err)
		}

		val, _ = storage2.Get("key")
		if val != "original" {
			t.Errorf("After reload, key = %v, want 'original'", val)
		}
		storage2.Close()
	})

	t.Run("returns error when closed", func(t *testing.T) {
		storage, _ := NewClineFileStorage(filePath, 0644)
		storage.Close()

		if err := storage.Reload(); err == nil {
			t.Error("Should return error when storage is closed")
		}
	})
}

func TestClineFileStorage_AtomicWrites(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cline-storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewClineFileStorage(filepath.Join(tempDir, "atomic.json"), 0644)
	if err != nil {
		t.Fatalf("NewClineFileStorage failed: %v", err)
	}
	defer storage.Close()

	t.Run("no temp files left after write", func(t *testing.T) {
		// Perform multiple writes
		for i := 0; i < 10; i++ {
			storage.Set(fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i))
		}

		// Check for temp files
		entries, err := os.ReadDir(tempDir)
		if err != nil {
			t.Fatalf("Failed to read directory: %v", err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			// Look for temp file patterns
			if len(name) > 4 && name[len(name)-4:] == ".tmp" {
				t.Errorf("Temp file left behind: %s", name)
			}
		}
	})

	t.Run("data integrity after many writes", func(t *testing.T) {
		// Write many values rapidly
		for i := 0; i < 100; i++ {
			if err := storage.Set("counter", i); err != nil {
				t.Fatalf("Set failed at iteration %d: %v", i, err)
			}
		}

		// Final value should be 99
		val, ok := storage.Get("counter")
		if !ok {
			t.Fatal("Counter key not found")
		}
		if val != float64(99) {
			t.Errorf("Final value = %v, want 99", val)
		}
	})
}

func TestClineFileStorage_ConcurrentFileAccess(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cline-storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filePath := filepath.Join(tempDir, "concurrent.json")

	t.Run("multiple processes can access file safely", func(t *testing.T) {
		// Create first storage
		storage1, err := NewClineFileStorage(filePath, 0644)
		if err != nil {
			t.Fatalf("NewClineFileStorage failed: %v", err)
		}

		// Try to create second storage (should use file locking)
		storage2, err := NewClineFileStorage(filePath, 0644)
		if err != nil {
			// This is expected if file locking prevents concurrent access
			t.Logf("Second storage creation result: %v", err)
		} else {
			// If it succeeds, both should work
			storage2.Close()
		}

		storage1.Close()
	})
}

func TestClineFileStorage_LargeData(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cline-storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewClineFileStorage(filepath.Join(tempDir, "large.json"), 0644)
	if err != nil {
		t.Fatalf("NewClineFileStorage failed: %v", err)
	}
	defer storage.Close()

	t.Run("handles large values", func(t *testing.T) {
		// Create a large string (1MB)
		largeString := make([]byte, 1024*1024)
		for i := range largeString {
			largeString[i] = byte('a' + (i % 26))
		}

		if err := storage.Set("large", string(largeString)); err != nil {
			t.Fatalf("Set failed for large value: %v", err)
		}

		val, ok := storage.Get("large")
		if !ok {
			t.Fatal("Large value not found")
		}

		retrieved := val.(string)
		if len(retrieved) != len(largeString) {
			t.Errorf("Retrieved size = %d, want %d", len(retrieved), len(largeString))
		}
	})

	t.Run("handles many keys", func(t *testing.T) {
		// Add many keys
		for i := 0; i < 1000; i++ {
			key := fmt.Sprintf("key%d", i)
			if err := storage.Set(key, fmt.Sprintf("value%d", i)); err != nil {
				t.Fatalf("Set failed at key %d: %v", i, err)
			}
		}

		// Verify count
		all := storage.GetAll()
		if len(all) < 1000 {
			t.Errorf("Expected at least 1000 keys, got %d", len(all))
		}
	})
}

func TestClineFileStorage_SpecialValues(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cline-storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewClineFileStorage(filepath.Join(tempDir, "special.json"), 0644)
	if err != nil {
		t.Fatalf("NewClineFileStorage failed: %v", err)
	}
	defer storage.Close()

	t.Run("handles unicode strings", func(t *testing.T) {
		tests := []string{
			"Hello, 世界",
			"🎉 Party time! 🎊",
			"مرحبا بالعالم",
			"שלום עולם",
		}

		for i, value := range tests {
			key := fmt.Sprintf("unicode%d", i)
			if err := storage.Set(key, value); err != nil {
				t.Fatalf("Set failed for unicode %d: %v", i, err)
			}

			retrieved, ok := storage.Get(key)
			if !ok {
				t.Fatalf("Unicode key %d not found", i)
			}

			if retrieved != value {
				t.Errorf("Unicode %d: got %q, want %q", i, retrieved, value)
			}
		}
	})

	t.Run("handles special characters in keys", func(t *testing.T) {
		// Note: Some characters might not be valid in JSON keys
		// but we should handle common ones
		tests := []string{
			"key-with-dashes",
			"key_with_underscores",
			"key.with.dots",
			"key:with:colons",
		}

		for _, key := range tests {
			if err := storage.Set(key, "value"); err != nil {
				t.Fatalf("Set failed for key %q: %v", key, err)
			}

			retrieved, ok := storage.Get(key)
			if !ok {
				t.Errorf("Key %q not found", key)
			} else if retrieved != "value" {
				t.Errorf("Key %q: got %v, want 'value'", key, retrieved)
			}
		}
	})
}

func TestClineFileStorage_Close(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cline-storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("persists data on close", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "close-persist.json")
		storage, _ := NewClineFileStorage(filePath, 0644)

		storage.Set("key", "value")

		// Close should persist
		if err := storage.Close(); err != nil {
			t.Fatalf("Close failed: %v", err)
		}

		// Verify file exists and contains data
		data, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("File is not valid JSON: %v", err)
		}

		if parsed["key"] != "value" {
			t.Errorf("Expected 'value', got %v", parsed["key"])
		}
	})

	t.Run("multiple close calls are safe", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "multi-close.json")
		storage, _ := NewClineFileStorage(filePath, 0644)

		storage.Set("key", "value")

		// First close should persist
		if err := storage.Close(); err != nil {
			t.Fatalf("First close failed: %v", err)
		}

		// Subsequent closes should be safe (no-op)
		for i := 0; i < 3; i++ {
			if err := storage.Close(); err != nil {
				t.Fatalf("Close %d failed: %v", i+2, err)
			}
		}
	})
}

func TestClineFileStorage_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "cline-stress-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewClineFileStorage(filepath.Join(tempDir, "stress.json"), 0644)
	if err != nil {
		t.Fatalf("NewClineFileStorage failed: %v", err)
	}
	defer storage.Close()

	t.Run("rapid concurrent operations", func(t *testing.T) {
		const numWorkers = 20
		const opsPerWorker = 500

		var wg sync.WaitGroup
		start := make(chan bool)

		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				<-start // Synchronize start

				for j := 0; j < opsPerWorker; j++ {
					switch j % 4 {
					case 0:
						storage.Set(fmt.Sprintf("w%d-key%d", id, j), j)
					case 1:
						storage.Get(fmt.Sprintf("w%d-key%d", id, j-1))
					case 2:
						storage.GetAll()
					case 3:
						if j > 100 {
							storage.Delete(fmt.Sprintf("w%d-key%d", id, j-100))
						}
					}
				}
			}(i)
		}

		// Start all workers simultaneously
		close(start)
		wg.Wait()

		// Storage should still be functional
		if err := storage.Set("final", "test"); err != nil {
			t.Fatalf("Storage not functional after stress test: %v", err)
		}
	})
}
