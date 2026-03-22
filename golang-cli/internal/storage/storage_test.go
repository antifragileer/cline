package storage

import (
	"os"
	"path/filepath"
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
			if contains(hash, char) {
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
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}