package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStorageContext(t *testing.T) {
	t.Run("creates storage context with paths", func(t *testing.T) {
		tempDir := t.TempDir()

		ctx, err := NewStorageContext(tempDir, "")
		require.NoError(t, err)
		assert.NotNil(t, ctx)

		assert.NotNil(t, ctx.GlobalState)
		assert.NotNil(t, ctx.Secrets)
		// WorkspaceState is nil when workspaceHash is empty
		assert.Nil(t, ctx.WorkspaceState)
	})

	t.Run("creates directories if they don't exist", func(t *testing.T) {
		tempDir := t.TempDir()
		configDir := filepath.Join(tempDir, "config")
		dataDir := filepath.Join(configDir, "data")

		_, err := NewStorageContext(dataDir, "test-workspace")
		require.NoError(t, err)

		assert.DirExists(t, dataDir)
	})

	t.Run("creates workspace storage when hash provided", func(t *testing.T) {
		tempDir := t.TempDir()

		ctx, err := NewStorageContext(tempDir, "test-workspace")
		require.NoError(t, err)

		assert.NotNil(t, ctx.WorkspaceState)
	})
}

func TestStorageContext_GlobalState(t *testing.T) {
	tempDir := t.TempDir()
	ctx, err := NewStorageContext(tempDir, "")
	require.NoError(t, err)
	defer ctx.Close()

	t.Run("set and get global state", func(t *testing.T) {
		err := ctx.GlobalState.Set("test-key", "test-value")
		require.NoError(t, err)

		value, ok := ctx.GlobalState.Get("test-key")
		require.True(t, ok)
		assert.Equal(t, "test-value", value)
	})

	t.Run("get non-existent key returns false", func(t *testing.T) {
		_, ok := ctx.GlobalState.Get("non-existent-key")
		assert.False(t, ok)
	})

	t.Run("update existing key", func(t *testing.T) {
		err := ctx.GlobalState.Set("update-key", "initial")
		require.NoError(t, err)

		err = ctx.GlobalState.Set("update-key", "updated")
		require.NoError(t, err)

		value, ok := ctx.GlobalState.Get("update-key")
		require.True(t, ok)
		assert.Equal(t, "updated", value)
	})
}

func TestStorageContext_WorkspaceState(t *testing.T) {
	tempDir := t.TempDir()
	ctx, err := NewStorageContext(tempDir, "test-workspace")
	require.NoError(t, err)
	defer ctx.Close()

	t.Run("set and get workspace state", func(t *testing.T) {
		err := ctx.WorkspaceState.Set("ws-key", "ws-value")
		require.NoError(t, err)

		value, ok := ctx.WorkspaceState.Get("ws-key")
		require.True(t, ok)
		assert.Equal(t, "ws-value", value)
	})
}

func TestStorageContext_Secrets(t *testing.T) {
	tempDir := t.TempDir()
	ctx, err := NewStorageContext(tempDir, "")
	require.NoError(t, err)
	defer ctx.Close()

	t.Run("set and get secrets", func(t *testing.T) {
		err := ctx.Secrets.Set("secret-key", "secret-value")
		require.NoError(t, err)

		value, ok := ctx.Secrets.Get("secret-key")
		require.True(t, ok)
		assert.Equal(t, "secret-value", value)
	})

	t.Run("secrets stored in separate file", func(t *testing.T) {
		err := ctx.Secrets.Set("another-secret", "another-value")
		require.NoError(t, err)

		// Verify secrets file exists
		secretsPath := filepath.Join(tempDir, "secrets.json")
		_, err = os.Stat(secretsPath)
		assert.NoError(t, err)
	})
}

func TestClineFileStorage(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.json")

	t.Run("creates new file storage", func(t *testing.T) {
		storage, err := NewClineFileStorage(filePath, 0644)
		require.NoError(t, err)
		assert.NotNil(t, storage)
		storage.Close()
	})

	t.Run("set and get string value", func(t *testing.T) {
		storage, err := NewClineFileStorage(filePath, 0644)
		require.NoError(t, err)
		defer storage.Close()

		err = storage.Set("key", "value")
		require.NoError(t, err)

		value, ok := storage.Get("key")
		require.True(t, ok)
		assert.Equal(t, "value", value)
	})

	t.Run("set and get complex value", func(t *testing.T) {
		storage, err := NewClineFileStorage(filePath, 0644)
		require.NoError(t, err)
		defer storage.Close()

		complexValue := map[string]interface{}{
			"nested": map[string]interface{}{
				"key": "value",
			},
			"array": []interface{}{1, 2, 3},
		}

		err = storage.Set("complex", complexValue)
		require.NoError(t, err)

		value, ok := storage.Get("complex")
		require.True(t, ok)

		// Type assertion for complex value
		valueMap, ok := value.(map[string]interface{})
		require.True(t, ok)
		assert.Contains(t, valueMap, "nested")
		assert.Contains(t, valueMap, "array")
	})

	t.Run("delete key", func(t *testing.T) {
		storage, err := NewClineFileStorage(filePath, 0644)
		require.NoError(t, err)
		defer storage.Close()

		err = storage.Set("delete-me", "value")
		require.NoError(t, err)

		err = storage.Delete("delete-me")
		require.NoError(t, err)

		_, ok := storage.Get("delete-me")
		assert.False(t, ok)
	})

	t.Run("get all values", func(t *testing.T) {
		storage, err := NewClineFileStorage(filePath, 0644)
		require.NoError(t, err)
		defer storage.Close()

		err = storage.Set("key1", "value1")
		require.NoError(t, err)
		err = storage.Set("key2", "value2")
		require.NoError(t, err)

		all := storage.GetAll()
		assert.Contains(t, all, "key1")
		assert.Contains(t, all, "key2")
	})

	t.Run("persists across instances", func(t *testing.T) {
		filePath2 := filepath.Join(tempDir, "persist.json")

		// First instance
		storage1, err := NewClineFileStorage(filePath2, 0644)
		require.NoError(t, err)

		err = storage1.Set("persist-key", "persist-value")
		require.NoError(t, err)
		storage1.Close()

		// Second instance reading same file
		storage2, err := NewClineFileStorage(filePath2, 0644)
		require.NoError(t, err)
		defer storage2.Close()

		value, ok := storage2.Get("persist-key")
		require.True(t, ok)
		assert.Equal(t, "persist-value", value)
	})
}

func TestStorageContext_Close(t *testing.T) {
	t.Run("closes without error", func(t *testing.T) {
		tempDir := t.TempDir()
		ctx, err := NewStorageContext(tempDir, "test-workspace")
		require.NoError(t, err)

		err = ctx.Close()
		assert.NoError(t, err)
	})
}

func TestGetWorkspaceHash(t *testing.T) {
	t.Run("generates hash from path", func(t *testing.T) {
		hash, err := GetWorkspaceHash("/home/user/project")
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
	})

	t.Run("handles relative paths", func(t *testing.T) {
		hash, err := GetWorkspaceHash("./project")
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
	})
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal", "normal"},
		{"path/to/file", "path_to_file"},
		{"file:with:colons", "file_with_colons"},
		{"file*with?special<chars>", "file_with_special_chars_"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCloneValue(t *testing.T) {
	t.Run("clones simple value", func(t *testing.T) {
		original := "test-value"
		cloned, err := CloneValue(original)
		require.NoError(t, err)
		assert.Equal(t, original, cloned)
	})

	t.Run("clones complex value", func(t *testing.T) {
		original := map[string]interface{}{
			"key": "value",
			"nested": map[string]interface{}{
				"inner": "data",
			},
		}
		cloned, err := CloneValue(original)
		require.NoError(t, err)

		// Verify it's a deep copy by modifying original
		original["key"] = "modified"
		clonedMap, ok := cloned.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "value", clonedMap["key"])
	})
}

func TestGetTyped(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "typed.json")

	t.Run("retrieves typed value", func(t *testing.T) {
		storage, err := NewClineFileStorage(filePath, 0644)
		require.NoError(t, err)
		defer storage.Close()

		type TestStruct struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}

		storage.Set("test", map[string]interface{}{
			"name":  "test-name",
			"value": 42,
		})

		var result TestStruct
		ok, err := GetTyped(storage, "test", &result)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "test-name", result.Name)
		assert.Equal(t, 42, result.Value)
	})

	t.Run("returns false for missing key", func(t *testing.T) {
		storage, err := NewClineFileStorage(filePath, 0644)
		require.NoError(t, err)
		defer storage.Close()

		var result string
		ok, err := GetTyped(storage, "non-existent", &result)
		require.NoError(t, err)
		assert.False(t, ok)
	})
}

func TestSetBatch(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "batch.json")

	t.Run("sets multiple values atomically", func(t *testing.T) {
		storage, err := NewClineFileStorage(filePath, 0644)
		require.NoError(t, err)
		defer storage.Close()

		batch := map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		}

		err = storage.SetBatch(batch)
		require.NoError(t, err)

		for k, v := range batch {
			value, ok := storage.Get(k)
			require.True(t, ok)
			assert.Equal(t, v, value)
		}
	})
}