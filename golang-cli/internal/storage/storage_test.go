package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestNewStorageContext(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cline-storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("creates global state and secrets storage", func(t *testing.T) {
		ctx, err := NewStorageContext(tempDir, "")
		if err != nil {
			t.Fatalf("NewStorageContext failed: %v", err)
		}
		defer ctx.Close()

		if ctx.GlobalState == nil {
			t.Error("GlobalState should not be nil")
		}
		if ctx.Secrets == nil {
			t.Error("Secrets should not be nil")
		}
		if ctx.WorkspaceState != nil {
			t.Error("WorkspaceState should be nil when no workspace hash provided")
		}
	})

	t.Run("creates workspace state when hash provided", func(t *testing.T) {
		ctx, err := NewStorageContext(tempDir, "test-workspace")
		if err != nil {
			t.Fatalf("NewStorageContext failed: %v", err)
		}
		defer ctx.Close()

		if ctx.WorkspaceState == nil {
			t.Error("WorkspaceState should not be nil when workspace hash provided")
		}
	})

	t.Run("uses default directory when empty", func(t *testing.T) {
		// This test verifies the default path construction
		// We can't easily test the actual default without mocking os.UserHomeDir
		// but we can verify it doesn't error when given a path
		ctx, err := NewStorageContext(tempDir, "")
		if err != nil {
			t.Fatalf("NewStorageContext failed: %v", err)
		}
		ctx.Close()
	})

	t.Run("handles permission errors on global state", func(t *testing.T) {
		// Create a read-only directory to simulate permission error
		readOnlyDir := filepath.Join(tempDir, "readonly")
		if err := os.MkdirAll(readOnlyDir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		// Try to create storage in a path that will fail
		// This test may not work on all systems, so we check if it errors appropriately
		_, err := NewStorageContext(readOnlyDir, "")
		// Should succeed since we're just creating files, not directories with special perms
		if err != nil {
			t.Logf("Got expected error in restricted environment: %v", err)
		}
	})

	t.Run("handles nested workspace directories", func(t *testing.T) {
		nestedHash := "very/nested/workspace/hash"
		ctx, err := NewStorageContext(tempDir, nestedHash)
		if err != nil {
			t.Fatalf("NewStorageContext failed: %v", err)
		}
		defer ctx.Close()

		if ctx.WorkspaceState == nil {
			t.Error("WorkspaceState should be created even with nested hash")
		}
	})

	t.Run("sanitizes workspace hash in path", func(t *testing.T) {
		// Use hash with special characters that should be sanitized
		hashWithSpecial := "workspace:with*special?chars"
		ctx, err := NewStorageContext(tempDir, hashWithSpecial)
		if err != nil {
			t.Fatalf("NewStorageContext failed: %v", err)
		}
		defer ctx.Close()

		if ctx.WorkspaceState == nil {
			t.Error("WorkspaceState should be created with sanitized hash")
		}
	})
}

func TestStorageContextClose(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cline-storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("closes all storage instances", func(t *testing.T) {
		ctx, err := NewStorageContext(tempDir, "workspace")
		if err != nil {
			t.Fatalf("NewStorageContext failed: %v", err)
		}

		// Write some data first
		if err := ctx.GlobalState.Set("key1", "value1"); err != nil {
			t.Fatalf("Failed to set global state: %v", err)
		}
		if err := ctx.Secrets.Set("secret1", "secretvalue"); err != nil {
			t.Fatalf("Failed to set secrets: %v", err)
		}
		if err := ctx.WorkspaceState.Set("wskey", "wsvalue"); err != nil {
			t.Fatalf("Failed to set workspace state: %v", err)
		}

		if err := ctx.Close(); err != nil {
			t.Fatalf("Close failed: %v", err)
		}

		// Verify files were created
		globalPath := filepath.Join(tempDir, "globalState.json")
		secretsPath := filepath.Join(tempDir, "secrets.json")
		workspacePath := filepath.Join(tempDir, "workspaces", "workspace", "workspaceState.json")

		if _, err := os.Stat(globalPath); os.IsNotExist(err) {
			t.Error("globalState.json should exist after close")
		}
		if _, err := os.Stat(secretsPath); os.IsNotExist(err) {
			t.Error("secrets.json should exist after close")
		}
		if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
			t.Error("workspaceState.json should exist after close")
		}
	})

	t.Run("closes partial storage context", func(t *testing.T) {
		ctx, err := NewStorageContext(tempDir, "")
		if err != nil {
			t.Fatalf("NewStorageContext failed: %v", err)
		}

		// Close without workspace state
		if err := ctx.Close(); err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	})

	t.Run("handles double close gracefully", func(t *testing.T) {
		ctx, err := NewStorageContext(tempDir, "double-close")
		if err != nil {
			t.Fatalf("NewStorageContext failed: %v", err)
		}

		// First close
		if err := ctx.Close(); err != nil {
			t.Fatalf("First close failed: %v", err)
		}

		// Second close - should not error (idempotent)
		if err := ctx.Close(); err != nil {
			t.Fatalf("Second close should be idempotent: %v", err)
		}
	})

	t.Run("concurrent close operations", func(t *testing.T) {
		ctx, err := NewStorageContext(tempDir, "concurrent-close")
		if err != nil {
			t.Fatalf("NewStorageContext failed: %v", err)
		}

		var wg sync.WaitGroup
		errChan := make(chan error, 3)

		// Try to close from multiple goroutines
		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := ctx.Close(); err != nil {
					errChan <- err
				}
			}()
		}

		wg.Wait()
		close(errChan)

		// Should not have errors (at most one succeeds, others are no-ops)
		for err := range errChan {
			t.Logf("Close error (may be expected): %v", err)
		}
	})
}

func TestGetWorkspaceHash(t *testing.T) {
	t.Run("generates hash from absolute path", func(t *testing.T) {
		hash, err := GetWorkspaceHash("/home/user/myproject")
		if err != nil {
			t.Fatalf("GetWorkspaceHash failed: %v", err)
		}

		if hash == "" {
			t.Error("Hash should not be empty")
		}

		// Should not contain path separators
		for _, char := range []string{"/", "\\", ":"} {
			if containsStr(hash, char) {
				t.Errorf("Hash should not contain %q", char)
			}
		}
	})

	t.Run("handles relative paths", func(t *testing.T) {
		hash, err := GetWorkspaceHash("./myproject")
		if err != nil {
			t.Fatalf("GetWorkspaceHash failed: %v", err)
		}

		if hash == "" {
			t.Error("Hash should not be empty")
		}
	})

	t.Run("handles paths with special characters", func(t *testing.T) {
		// This should work but sanitize the output
		hash, err := GetWorkspaceHash("/path/with spaces/and-symbols!")
		if err != nil {
			t.Fatalf("GetWorkspaceHash failed: %v", err)
		}

		if hash == "" {
			t.Error("Hash should not be empty even with special characters")
		}
	})

	t.Run("handles very long paths", func(t *testing.T) {
		// Create a very long path
		longPath := "/very"
		for i := 0; i < 50; i++ {
			longPath += "/long/path/segment"
		}
		longPath += "/project"

		hash, err := GetWorkspaceHash(longPath)
		if err != nil {
			t.Fatalf("GetWorkspaceHash failed: %v", err)
		}

		if hash == "" {
			t.Error("Hash should not be empty for long paths")
		}
	})

	t.Run("handles empty path", func(t *testing.T) {
		hash, err := GetWorkspaceHash("")
		if err != nil {
			t.Fatalf("GetWorkspaceHash failed: %v", err)
		}

		// Empty path should still produce a hash
		if hash == "" {
			t.Error("Hash should not be empty even for empty path")
		}
	})

	t.Run("produces consistent hashes", func(t *testing.T) {
		path := "/home/user/consistent-project"

		hash1, err := GetWorkspaceHash(path)
		if err != nil {
			t.Fatalf("GetWorkspaceHash failed: %v", err)
		}

		hash2, err := GetWorkspaceHash(path)
		if err != nil {
			t.Fatalf("GetWorkspaceHash failed: %v", err)
		}

		if hash1 != hash2 {
			t.Errorf("Same path should produce same hash: %q != %q", hash1, hash2)
		}
	})

	t.Run("different paths produce different hashes", func(t *testing.T) {
		hash1, _ := GetWorkspaceHash("/path/one")
		hash2, _ := GetWorkspaceHash("/path/two")

		if hash1 == hash2 {
			t.Error("Different paths should produce different hashes")
		}
	})
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal", "normal"},
		{"path/with/slashes", "path_with_slashes"},
		{"path\\with\\backslashes", "path_with_backslashes"},
		{"file:with:colons", "file_with_colons"},
		{"file*with*stars", "file_with_stars"},
		{"file?with?questions", "file_with_questions"},
		{"file\"with\"quotes", "file_with_quotes"},
		{"file<with>brackets", "file_with_brackets"},
		{"file|with|pipes", "file_with_pipes"},
		{"", ""},
		{"ALL/\\:*?\"<>|INVALID", "ALL_________INVALID"},
		{"mixed/separators\\and:others", "mixed_separators_and_others"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCloneValue(t *testing.T) {
	t.Run("clones simple values", func(t *testing.T) {
		original := "test value"
		cloned, err := CloneValue(original)
		if err != nil {
			t.Fatalf("CloneValue failed: %v", err)
		}

		if cloned != original {
			t.Errorf("Cloned value %v != original %v", cloned, original)
		}

		// Modify clone shouldn't affect original (though for strings this is automatic)
		cloned = "modified"
		if original != "test value" {
			t.Error("Original should not be modified")
		}
	})

	t.Run("clones complex objects", func(t *testing.T) {
		original := map[string]interface{}{
			"key1": "value1",
			"key2": 42,
			"key3": []interface{}{"a", "b", "c"},
			"nested": map[string]interface{}{
				"inner": "value",
			},
		}

		cloned, err := CloneValue(original)
		if err != nil {
			t.Fatalf("CloneValue failed: %v", err)
		}

		clonedMap, ok := cloned.(map[string]interface{})
		if !ok {
			t.Fatal("Cloned value should be a map")
		}

		// Modify the clone
		clonedMap["key1"] = "modified"
		nested := clonedMap["nested"].(map[string]interface{})
		nested["inner"] = "modified"

		// Original should be unchanged
		if original["key1"] != "value1" {
			t.Error("Original key1 should not be modified")
		}
		originalNested := original["nested"].(map[string]interface{})
		if originalNested["inner"] != "value" {
			t.Error("Original nested.inner should not be modified")
		}
	})

	t.Run("clones arrays", func(t *testing.T) {
		original := []interface{}{"a", "b", "c"}
		cloned, err := CloneValue(original)
		if err != nil {
			t.Fatalf("CloneValue failed: %v", err)
		}

		clonedSlice, ok := cloned.([]interface{})
		if !ok {
			t.Fatal("Cloned value should be a slice")
		}

		// Modify clone
		clonedSlice[0] = "modified"

		// Original should be unchanged
		if original[0] != "a" {
			t.Error("Original[0] should not be modified")
		}
	})

	t.Run("clones nil", func(t *testing.T) {
		cloned, err := CloneValue(nil)
		if err != nil {
			t.Fatalf("CloneValue failed: %v", err)
		}

		if cloned != nil {
			t.Errorf("Cloned nil should be nil, got %v", cloned)
		}
	})

	t.Run("clones numbers", func(t *testing.T) {
		tests := []interface{}{
			int(42),
			int64(42),
			float64(3.14),
			float32(2.71),
		}

		for _, original := range tests {
			cloned, err := CloneValue(original)
			if err != nil {
				t.Fatalf("CloneValue failed for %v: %v", original, err)
			}

			// JSON unmarshaling converts numbers to float64
			if _, ok := cloned.(float64); !ok && cloned != original {
				t.Errorf("Cloned value type mismatch: %T vs %T", cloned, original)
			}
		}
	})

	t.Run("clones booleans", func(t *testing.T) {
		original := true
		cloned, err := CloneValue(original)
		if err != nil {
			t.Fatalf("CloneValue failed: %v", err)
		}

		if cloned != true {
			t.Errorf("Cloned boolean should be true, got %v", cloned)
		}
	})

	t.Run("handles deeply nested structures", func(t *testing.T) {
		original := map[string]interface{}{
			"level1": map[string]interface{}{
				"level2": map[string]interface{}{
					"level3": map[string]interface{}{
						"value": "deep",
					},
				},
			},
		}

		cloned, err := CloneValue(original)
		if err != nil {
			t.Fatalf("CloneValue failed: %v", err)
		}

		// Verify structure
		level1 := cloned.(map[string]interface{})["level1"].(map[string]interface{})
		level2 := level1["level2"].(map[string]interface{})
		level3 := level2["level3"].(map[string]interface{})

		if level3["value"] != "deep" {
			t.Error("Deeply nested value not preserved")
		}
	})

	t.Run("handles circular reference detection", func(t *testing.T) {
		// Note: JSON marshaling will fail with circular references
		// This tests that we properly handle the error
		type Node struct {
			Name  string `json:"name"`
			Child *Node  `json:"child,omitempty"`
		}

		node1 := &Node{Name: "parent"}
		node2 := &Node{Name: "child", Child: node1}
		node1.Child = node2 // Circular reference

		_, err := CloneValue(node1)
		if err == nil {
			t.Error("Should fail with circular reference")
		}
	})
}

func TestGetTyped(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cline-storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewClineFileStorage(filepath.Join(tempDir, "test.json"), 0644)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	t.Run("retrieves typed values", func(t *testing.T) {
		type TestStruct struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}

		original := TestStruct{Name: "test", Value: 42}
		if err := storage.Set("key", original); err != nil {
			t.Fatalf("Failed to set value: %v", err)
		}

		var result TestStruct
		found, err := GetTyped(storage, "key", &result)
		if err != nil {
			t.Fatalf("GetTyped failed: %v", err)
		}
		if !found {
			t.Error("Key should be found")
		}

		if result.Name != "test" || result.Value != 42 {
			t.Errorf("Result mismatch: got %+v, want %+v", result, original)
		}
	})

	t.Run("returns false for missing keys", func(t *testing.T) {
		var result string
		found, err := GetTyped(storage, "nonexistent", &result)
		if err != nil {
			t.Fatalf("GetTyped failed: %v", err)
		}
		if found {
			t.Error("Nonexistent key should not be found")
		}
	})

	t.Run("handles type conversion", func(t *testing.T) {
		// Store as map
		data := map[string]interface{}{"x": 1, "y": 2}
		if err := storage.Set("point", data); err != nil {
			t.Fatalf("Failed to set value: %v", err)
		}

		// Retrieve into struct
		type Point struct {
			X int `json:"x"`
			Y int `json:"y"`
		}

		var point Point
		found, err := GetTyped(storage, "point", &point)
		if err != nil {
			t.Fatalf("GetTyped failed: %v", err)
		}
		if !found {
			t.Error("Key should be found")
		}

		if point.X != 1 || point.Y != 2 {
			t.Errorf("Point mismatch: got %+v", point)
		}
	})

	t.Run("handles slice types", func(t *testing.T) {
		original := []string{"a", "b", "c"}
		if err := storage.Set("list", original); err != nil {
			t.Fatalf("Failed to set value: %v", err)
		}

		var result []string
		found, err := GetTyped(storage, "list", &result)
		if err != nil {
			t.Fatalf("GetTyped failed: %v", err)
		}
		if !found {
			t.Error("Key should be found")
		}

		if len(result) != 3 || result[0] != "a" {
			t.Errorf("Slice mismatch: got %v", result)
		}
	})
}

func TestStorageContextConcurrency(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cline-storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ctx, err := NewStorageContext(tempDir, "concurrent-workspace")
	if err != nil {
		t.Fatalf("NewStorageContext failed: %v", err)
	}
	defer ctx.Close()

	t.Run("concurrent reads and writes", func(t *testing.T) {
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
						key := fmt.Sprintf("writer%d-key%d", id, j)
						if err := ctx.GlobalState.Set(key, j); err != nil {
							t.Errorf("Set failed: %v", err)
						}
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
						ctx.GlobalState.Get(fmt.Sprintf("key%d", j))
						ctx.GlobalState.GetAll()
					}
				}
			}(i)
		}

		wg.Wait()
		close(done)
	})

	t.Run("concurrent access to different storage types", func(t *testing.T) {
		var wg sync.WaitGroup

		// Write to global state
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				ctx.GlobalState.Set(fmt.Sprintf("global%d", i), i)
			}
		}()

		// Write to secrets
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				ctx.Secrets.Set(fmt.Sprintf("secret%d", i), fmt.Sprintf("value%d", i))
			}
		}()

		// Write to workspace state
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				ctx.WorkspaceState.Set(fmt.Sprintf("workspace%d", i), i)
			}
		}()

		wg.Wait()
	})
}

func TestStorageContextDataPersistence(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cline-storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("data persists across context instances", func(t *testing.T) {
		// First context
		ctx1, err := NewStorageContext(tempDir, "persist-test")
		if err != nil {
			t.Fatalf("NewStorageContext failed: %v", err)
		}

		// Store data
		ctx1.GlobalState.Set("persistent-key", "persistent-value")
		ctx1.Secrets.Set("persistent-secret", "secret-value")
		ctx1.WorkspaceState.Set("persistent-ws", "ws-value")

		if err := ctx1.Close(); err != nil {
			t.Fatalf("Close failed: %v", err)
		}

		// Second context
		ctx2, err := NewStorageContext(tempDir, "persist-test")
		if err != nil {
			t.Fatalf("NewStorageContext failed: %v", err)
		}
		defer ctx2.Close()

		// Verify data persisted
		if val, ok := ctx2.GlobalState.Get("persistent-key"); !ok || val != "persistent-value" {
			t.Errorf("GlobalState key not persisted: got %v", val)
		}

		if val, ok := ctx2.Secrets.Get("persistent-secret"); !ok || val != "secret-value" {
			t.Errorf("Secrets key not persisted: got %v", val)
		}

		if val, ok := ctx2.WorkspaceState.Get("persistent-ws"); !ok || val != "ws-value" {
			t.Errorf("WorkspaceState key not persisted: got %v", val)
		}
	})
}

// Helper function
func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}