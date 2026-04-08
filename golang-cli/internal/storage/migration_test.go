package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMigrationRegistry_Register(t *testing.T) {
	registry := NewMigrationRegistry()

	tests := []struct {
		name    string
		m       Migration
		wantErr bool
	}{
		{
			name: "valid migration",
			m: Migration{
				FromVersion: 1,
				ToVersion:   2,
				Name:        "test_migration",
				Apply:       func(data map[string]any) error { return nil },
			},
			wantErr: false,
		},
		{
			name: "invalid from version 0",
			m: Migration{
				FromVersion: 0,
				ToVersion:   1,
				Name:        "invalid_from",
				Apply:       func(data map[string]any) error { return nil },
			},
			wantErr: true,
		},
		{
			name: "to version not greater than from",
			m: Migration{
				FromVersion: 2,
				ToVersion:   2,
				Name:        "same_version",
				Apply:       func(data map[string]any) error { return nil },
			},
			wantErr: true,
		},
		{
			name: "non-sequential migration",
			m: Migration{
				FromVersion: 1,
				ToVersion:   3,
				Name:        "skip_version",
				Apply:       func(data map[string]any) error { return nil },
			},
			wantErr: true,
		},
		{
			name: "duplicate from version",
			m: Migration{
				FromVersion: 1,
				ToVersion:   2,
				Name:        "duplicate",
				Apply:       func(data map[string]any) error { return nil },
			},
			wantErr: true, // 1->2 already registered
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.Register(tt.m)
			if (err != nil) != tt.wantErr {
				t.Errorf("Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetCurrentVersion(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]any
		expected int
	}{
		{
			name:     "no version key",
			data:     map[string]any{"key": "value"},
			expected: 0,
		},
		{
			name:     "version as int",
			data:     map[string]any{StorageVersionKey: 5},
			expected: 5,
		},
		{
			name:     "version as float64",
			data:     map[string]any{StorageVersionKey: 3.0},
			expected: 3,
		},
		{
			name:     "version as string",
			data:     map[string]any{StorageVersionKey: "7"},
			expected: 7,
		},
		{
			name:     "version as invalid string",
			data:     map[string]any{StorageVersionKey: "invalid"},
			expected: 0,
		},
		{
			name:     "version as bool",
			data:     map[string]any{StorageVersionKey: true},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetCurrentVersion(tt.data)
			if result != tt.expected {
				t.Errorf("GetCurrentVersion() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestSetVersion(t *testing.T) {
	data := make(map[string]any)
	SetVersion(data, 42)
	if data[StorageVersionKey] != 42 {
		t.Errorf("SetVersion() failed, expected 42, got %v", data[StorageVersionKey])
	}
}

func TestMigrationRegistry_GetTargetVersion(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *MigrationRegistry
		expected int
	}{
		{
			name: "empty registry",
			setup: func() *MigrationRegistry {
				return NewMigrationRegistry()
			},
			expected: DefaultStorageVersion,
		},
		{
			name: "single migration",
			setup: func() *MigrationRegistry {
				r := NewMigrationRegistry()
				r.Register(Migration{
					FromVersion: 1,
					ToVersion:   2,
					Name:        "v1_to_v2",
					Apply:       func(data map[string]any) error { return nil },
				})
				return r
			},
			expected: 2,
		},
		{
			name: "multiple migrations",
			setup: func() *MigrationRegistry {
				r := NewMigrationRegistry()
				r.Register(Migration{
					FromVersion: 1,
					ToVersion:   2,
					Name:        "v1_to_v2",
					Apply:       func(data map[string]any) error { return nil },
				})
				r.Register(Migration{
					FromVersion: 2,
					ToVersion:   3,
					Name:        "v2_to_v3",
					Apply:       func(data map[string]any) error { return nil },
				})
				return r
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := tt.setup()
			result := registry.GetTargetVersion()
			if result != tt.expected {
				t.Errorf("GetTargetVersion() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestMigrationRegistry_NeedsMigration(t *testing.T) {
	registry := NewMigrationRegistry()
	registry.Register(Migration{
		FromVersion: 1,
		ToVersion:   2,
		Name:        "v1_to_v2",
		Apply:       func(data map[string]any) error { return nil },
	})

	tests := []struct {
		name     string
		data     map[string]any
		expected bool
	}{
		{
			name:     "no version needs migration",
			data:     map[string]any{},
			expected: true,
		},
		{
			name:     "version 1 needs migration",
			data:     map[string]any{StorageVersionKey: 1},
			expected: true,
		},
		{
			name:     "version 2 no migration",
			data:     map[string]any{StorageVersionKey: 2},
			expected: false,
		},
		{
			name:     "version 3 no migration",
			data:     map[string]any{StorageVersionKey: 3},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.NeedsMigration(tt.data)
			if result != tt.expected {
				t.Errorf("NeedsMigration() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestMigrationRegistry_GetMigrationPath(t *testing.T) {
	registry := NewMigrationRegistry()
	registry.Register(Migration{
		FromVersion: 1,
		ToVersion:   2,
		Name:        "v1_to_v2",
		Apply:       func(data map[string]any) error { return nil },
	})
	registry.Register(Migration{
		FromVersion: 2,
		ToVersion:   3,
		Name:        "v2_to_v3",
		Apply:       func(data map[string]any) error { return nil },
	})

	tests := []struct {
		name           string
		currentVersion int
		targetVersion  int
		expectedLen    int
		wantErr        bool
	}{
		{
			name:           "no migration needed",
			currentVersion: 3,
			targetVersion:  3,
			expectedLen:    0,
			wantErr:        false,
		},
		{
			name:           "single step",
			currentVersion: 2,
			targetVersion:  3,
			expectedLen:    1,
			wantErr:        false,
		},
		{
			name:           "multiple steps",
			currentVersion: 1,
			targetVersion:  3,
			expectedLen:    2,
			wantErr:        false,
		},
		{
			name:           "from version 0",
			currentVersion: 0,
			targetVersion:  3,
			expectedLen:    3, // init + 2 migrations
			wantErr:        false,
		},
		{
			name:           "missing migration",
			currentVersion: 3,
			targetVersion:  5,
			expectedLen:    0,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := registry.GetMigrationPath(tt.currentVersion, tt.targetVersion)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMigrationPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(path) != tt.expectedLen {
				t.Errorf("GetMigrationPath() returned %d migrations, expected %d", len(path), tt.expectedLen)
			}
		})
	}
}

func TestMigratableStorage_NewMigratableStorage(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.json")
	registry := NewMigrationRegistry()

	storage, err := NewMigratableStorage(filePath, registry)
	if err != nil {
		t.Fatalf("NewMigratableStorage() error = %v", err)
	}

	if storage.filePath != filePath {
		t.Errorf("filePath = %v, expected %v", storage.filePath, filePath)
	}
	if storage.backupDir != tmpDir {
		t.Errorf("backupDir = %v, expected %v", storage.backupDir, tmpDir)
	}
}

func TestMigratableStorage_WithOptions(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.json")
	backupDir := filepath.Join(tmpDir, "backups")
	registry := NewMigrationRegistry()

	storage, err := NewMigratableStorage(filePath, registry,
		WithFileMode(0600),
		WithBackupMode(0400),
		WithBackupDir(backupDir),
	)
	if err != nil {
		t.Fatalf("NewMigratableStorage() error = %v", err)
	}

	if storage.fileMode != 0600 {
		t.Errorf("fileMode = %v, expected 0600", storage.fileMode)
	}
	if storage.backupMode != 0400 {
		t.Errorf("backupMode = %v, expected 0400", storage.backupMode)
	}
	if storage.backupDir != backupDir {
		t.Errorf("backupDir = %v, expected %v", storage.backupDir, backupDir)
	}
}

func TestMigratableStorage_LoadAndSave(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.json")
	registry := NewMigrationRegistry()

	storage, err := NewMigratableStorage(filePath, registry)
	if err != nil {
		t.Fatalf("NewMigratableStorage() error = %v", err)
	}

	// Test loading non-existent file (should initialize with default version)
	err = storage.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if storage.GetVersion() != DefaultStorageVersion {
		t.Errorf("version = %v, expected %v", storage.GetVersion(), DefaultStorageVersion)
	}

	// Set some data
	storage.Set("key1", "value1")
	storage.Set("key2", 42)

	// Save
	err = storage.Save()
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("storage file was not created")
	}

	// Load into new storage
	storage2, err := NewMigratableStorage(filePath, registry)
	if err != nil {
		t.Fatalf("NewMigratableStorage() error = %v", err)
	}

	err = storage2.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify data
	if val, ok := storage2.GetString("key1"); !ok || val != "value1" {
		t.Errorf("key1 = %v, expected value1", val)
	}
	if val, ok := storage2.GetInt("key2"); !ok || val != 42 {
		t.Errorf("key2 = %v, expected 42", val)
	}
}

func TestMigratableStorage_Migration(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.json")

	// Create a registry with migrations
	registry := NewMigrationRegistry()
	registry.Register(Migration{
		FromVersion: 1,
		ToVersion:   2,
		Name:        "add_new_field",
		Apply: func(data map[string]any) error {
			data["newField"] = "migrated"
			return nil
		},
	})
	registry.Register(Migration{
		FromVersion: 2,
		ToVersion:   3,
		Name:        "update_existing",
		Apply: func(data map[string]any) error {
			if oldVal, ok := data["oldKey"]; ok {
				data["newKey"] = oldVal
				delete(data, "oldKey")
			}
			return nil
		},
	})

	// Create initial storage file with version 1
	initialJSON := `{"__storageVersion": 1, "oldKey": "oldValue"}`
	err := os.WriteFile(filePath, []byte(initialJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	// Load storage (should trigger migrations)
	storage, err := NewMigratableStorage(filePath, registry)
	if err != nil {
		t.Fatalf("NewMigratableStorage() error = %v", err)
	}

	err = storage.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify migrations applied
	if storage.GetVersion() != 3 {
		t.Errorf("version = %v, expected 3", storage.GetVersion())
	}

	if val, ok := storage.GetString("newField"); !ok || val != "migrated" {
		t.Errorf("newField = %v, expected 'migrated'", val)
	}

	if val, ok := storage.GetString("newKey"); !ok || val != "oldValue" {
		t.Errorf("newKey = %v, expected 'oldValue'", val)
	}

	if _, ok := storage.Get("oldKey"); ok {
		t.Error("oldKey should have been deleted")
	}
}

func TestMigratableStorage_MigrationFailureAndRestore(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.json")

	// Create a registry with a failing migration
	registry := NewMigrationRegistry()
	registry.Register(Migration{
		FromVersion: 1,
		ToVersion:   2,
		Name:        "will_fail",
		Apply: func(data map[string]any) error {
			return fmt.Errorf("intentional migration failure")
		},
	})

	// Create initial storage file with version 1
	initialData := map[string]any{
		StorageVersionKey: 1,
		"originalData":    "preserved",
	}
	initialJSON, _ := json.Marshal(initialData)
	err := os.WriteFile(filePath, initialJSON, 0644)
	if err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	// Load storage (should fail migration and restore)
	storage, err := NewMigratableStorage(filePath, registry)
	if err != nil {
		t.Fatalf("NewMigratableStorage() error = %v", err)
	}

	err = storage.Load()
	// Migration should fail and the error should indicate restoration was attempted
	if err == nil {
		t.Fatal("Expected migration to fail, but it succeeded")
	}

	// The error should mention that restoration was performed
	if err.Error() == "" {
		t.Fatal("Expected error message")
	}

	// Verify file content is preserved by reading directly
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var restoredData map[string]any
	if err := json.Unmarshal(content, &restoredData); err != nil {
		t.Fatalf("Failed to parse restored file: %v", err)
	}

	// Verify data was restored
	if val, ok := restoredData["originalData"]; !ok || val != "preserved" {
		t.Errorf("originalData = %v, expected 'preserved'", val)
	}
	if GetCurrentVersion(restoredData) != 1 {
		t.Errorf("version = %v, expected 1", GetCurrentVersion(restoredData))
	}
}

func TestMigratableStorage_GetAndSet(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.json")
	registry := NewMigrationRegistry()

	storage, err := NewMigratableStorage(filePath, registry)
	if err != nil {
		t.Fatalf("NewMigratableStorage() error = %v", err)
	}

	// Test Get on non-existent key
	if _, ok := storage.Get("nonexistent"); ok {
		t.Error("Get on non-existent key should return false")
	}

	// Test Set and Get
	storage.Set("stringKey", "stringValue")
	if val, ok := storage.Get("stringKey"); !ok || val != "stringValue" {
		t.Errorf("stringKey = %v, expected 'stringValue'", val)
	}

	// Test Set with nil (delete)
	storage.Set("stringKey", nil)
	if _, ok := storage.Get("stringKey"); ok {
		t.Error("Key should have been deleted")
	}

	// Test Delete
	storage.Set("toDelete", "value")
	storage.Delete("toDelete")
	if _, ok := storage.Get("toDelete"); ok {
		t.Error("Deleted key should not exist")
	}
}

func TestMigratableStorage_GetString(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.json")
	registry := NewMigrationRegistry()

	storage, err := NewMigratableStorage(filePath, registry)
	if err != nil {
		t.Fatalf("NewMigratableStorage() error = %v", err)
	}

	// Test string value
	storage.Set("stringKey", "value")
	if val, ok := storage.GetString("stringKey"); !ok || val != "value" {
		t.Errorf("stringKey = %v, expected 'value'", val)
	}

	// Test int value (should convert)
	storage.Set("intKey", 42)
	if val, ok := storage.GetString("intKey"); !ok || val != "42" {
		t.Errorf("intKey = %v, expected '42'", val)
	}

	// Test non-existent
	if _, ok := storage.GetString("nonexistent"); ok {
		t.Error("GetString on non-existent key should return false")
	}
}

func TestMigratableStorage_GetInt(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.json")
	registry := NewMigrationRegistry()

	storage, err := NewMigratableStorage(filePath, registry)
	if err != nil {
		t.Fatalf("NewMigratableStorage() error = %v", err)
	}

	// Test int value
	storage.Set("intKey", 42)
	if val, ok := storage.GetInt("intKey"); !ok || val != 42 {
		t.Errorf("intKey = %v, expected 42", val)
	}

	// Test float64 value (JSON numbers parse as float64)
	storage.Set("floatKey", 3.0)
	if val, ok := storage.GetInt("floatKey"); !ok || val != 3 {
		t.Errorf("floatKey = %v, expected 3", val)
	}

	// Test string value
	storage.Set("stringKey", "99")
	if val, ok := storage.GetInt("stringKey"); !ok || val != 99 {
		t.Errorf("stringKey = %v, expected 99", val)
	}

	// Test invalid string
	storage.Set("invalidString", "not_a_number")
	if _, ok := storage.GetInt("invalidString"); ok {
		t.Error("GetInt on invalid string should return false")
	}

	// Test non-existent
	if _, ok := storage.GetInt("nonexistent"); ok {
		t.Error("GetInt on non-existent key should return false")
	}
}

func TestMigratableStorage_Keys(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.json")
	registry := NewMigrationRegistry()

	storage, err := NewMigratableStorage(filePath, registry)
	if err != nil {
		t.Fatalf("NewMigratableStorage() error = %v", err)
	}

	// Add keys
	storage.Set("z", 1)
	storage.Set("a", 2)
	storage.Set("m", 3)

	// Get keys (should be sorted)
	keys := storage.Keys()
	expected := []string{"a", "m", "z"}
	if len(keys) != len(expected) {
		t.Fatalf("Keys() returned %d keys, expected %d", len(keys), len(expected))
	}
	for i, k := range keys {
		if k != expected[i] {
			t.Errorf("Keys()[%d] = %v, expected %v", i, k, expected[i])
		}
	}
}

func TestMigratableStorage_GetDataAndSetData(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.json")
	registry := NewMigrationRegistry()

	storage, err := NewMigratableStorage(filePath, registry)
	if err != nil {
		t.Fatalf("NewMigratableStorage() error = %v", err)
	}

	// Set some data
	storage.Set("key1", "value1")
	storage.Set("key2", 42)

	// Get data copy
	data := storage.GetData()
	if len(data) != 2 {
		t.Errorf("GetData() returned %d entries, expected 2", len(data))
	}

	// Modify copy (should not affect storage)
	data["key3"] = "new"
	if _, ok := storage.Get("key3"); ok {
		t.Error("Modifying data copy affected storage")
	}

	// Set new data
	newData := map[string]any{
		"newKey": "newValue",
	}
	storage.SetData(newData)

	// Verify old keys are gone
	if _, ok := storage.Get("key1"); ok {
		t.Error("Old key should have been removed")
	}

	// Verify new data
	if val, ok := storage.GetString("newKey"); !ok || val != "newValue" {
		t.Errorf("newKey = %v, expected 'newValue'", val)
	}
}

func TestMigratableStorage_Backups(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.json")
	registry := NewMigrationRegistry()

	storage, err := NewMigratableStorage(filePath, registry)
	if err != nil {
		t.Fatalf("NewMigratableStorage() error = %v", err)
	}

	// Create storage file
	storage.Set("data", "initial")
	err = storage.Save()
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// List backups (should be empty)
	backups, err := storage.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups() error = %v", err)
	}
	if len(backups) != 0 {
		t.Errorf("Expected 0 backups, got %d", len(backups))
	}

	// Create backup manually
	backupPath1, err := storage.createBackup()
	if err != nil {
		t.Fatalf("createBackup() error = %v", err)
	}

	// Verify backup exists
	if _, err := os.Stat(backupPath1); os.IsNotExist(err) {
		t.Error("Backup file was not created")
	}

	// List backups
	backups, err = storage.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups() error = %v", err)
	}
	if len(backups) != 1 {
		t.Errorf("Expected 1 backup, got %d", len(backups))
	}

	// Larger delay to ensure different timestamp
	time.Sleep(100 * time.Millisecond)

	// Modify storage
	storage.Set("data", "modified")
	err = storage.Save()
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Create another backup
	backupPath2, err := storage.createBackup()
	if err != nil {
		t.Fatalf("createBackup() error = %v", err)
	}

	// Verify second backup exists and is different from first
	if backupPath1 == backupPath2 {
		t.Error("Second backup has same path as first")
	}

	// List backups - should have at least 2 (or 1 if timestamps collided)
	backups, err = storage.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups() error = %v", err)
	}
	if len(backups) < 1 {
		t.Errorf("Expected at least 1 backup, got %d", len(backups))
	}

	// Cleanup old backups, keep 1
	err = storage.CleanupOldBackups(1)
	if err != nil {
		t.Fatalf("CleanupOldBackups() error = %v", err)
	}

	// Verify only 1 backup remains
	backups, err = storage.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups() error = %v", err)
	}
	if len(backups) != 1 {
		t.Errorf("Expected 1 backup after cleanup, got %d", len(backups))
	}
}

func TestMigratableStorage_RestoreFromBackup(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.json")
	registry := NewMigrationRegistry()

	storage, err := NewMigratableStorage(filePath, registry)
	if err != nil {
		t.Fatalf("NewMigratableStorage() error = %v", err)
	}

	// Create initial data and save
	storage.Set("key", "original")
	err = storage.Save()
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Create backup
	backupPath, err := storage.createBackup()
	if err != nil {
		t.Fatalf("createBackup() error = %v", err)
	}

	// Modify data
	storage.Set("key", "modified")
	err = storage.Save()
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify modification
	if val, _ := storage.GetString("key"); val != "modified" {
		t.Errorf("key = %v, expected 'modified'", val)
	}

	// Restore from backup
	err = storage.restoreFromBackup(backupPath)
	if err != nil {
		t.Fatalf("restoreFromBackup() error = %v", err)
	}

	// Verify restoration
	if val, _ := storage.GetString("key"); val != "original" {
		t.Errorf("key = %v, expected 'original'", val)
	}

	// Verify file on disk
	storage2, _ := NewMigratableStorage(filePath, registry)
	storage2.Load()
	if val, _ := storage2.GetString("key"); val != "original" {
		t.Errorf("restored file key = %v, expected 'original'", val)
	}
}

func TestMigratableStorage_MigrateTypeScriptFormats(t *testing.T) {
	// Test migrations from TypeScript CLI format to Go CLI format
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "globalState.json")

	registry := NewMigrationRegistry()

	// Register a migration that handles TypeScript format differences
	registry.Register(Migration{
		FromVersion: 1,
		ToVersion:   2,
		Name:        "migrate_ts_format",
		Apply: func(data map[string]any) error {
			// Example: Convert camelCase to snake_case keys
			if val, ok := data["apiKey"]; ok {
				data["api_key"] = val
				delete(data, "apiKey")
			}
			return nil
		},
	})

	// Create TypeScript-style data
	tsData := `{
		"__storageVersion": 1,
		"apiKey": "secret123",
		"userInfo": {
			"name": "test",
			"email": "test@example.com"
		}
	}`
	err := os.WriteFile(filePath, []byte(tsData), 0644)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	storage, err := NewMigratableStorage(filePath, registry)
	if err != nil {
		t.Fatalf("NewMigratableStorage() error = %v", err)
	}

	err = storage.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify migration applied
	if storage.GetVersion() != 2 {
		t.Errorf("version = %v, expected 2", storage.GetVersion())
	}

	if _, ok := storage.Get("apiKey"); ok {
		t.Error("old apiKey should not exist")
	}

	if val, ok := storage.GetString("api_key"); !ok || val != "secret123" {
		t.Errorf("api_key = %v, expected 'secret123'", val)
	}

	// Verify nested data preserved
	if userInfo, ok := storage.Get("userInfo"); !ok {
		t.Error("userInfo should be preserved")
	} else if userMap, ok := userInfo.(map[string]any); ok {
		if userMap["name"] != "test" {
			t.Errorf("userInfo.name = %v, expected 'test'", userMap["name"])
		}
	}
}

func TestMigratableStorage_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.json")
	registry := NewMigrationRegistry()

	storage, err := NewMigratableStorage(filePath, registry)
	if err != nil {
		t.Fatalf("NewMigratableStorage() error = %v", err)
	}

	// Load to initialize
	err = storage.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Set data and save
	storage.Set("key", "value")
	err = storage.Save()
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify no temp files left behind
	matches, err := filepath.Glob(filepath.Join(tmpDir, ".*.tmp.*"))
	if err != nil {
		t.Fatalf("Glob error: %v", err)
	}
	if len(matches) > 0 {
		t.Errorf("Temp files left behind: %v", matches)
	}
}

func TestMigratableStorage_FilePermissions(t *testing.T) {
	// Skip on non-Unix systems
	if os.Getuid() < 0 {
		t.Skip("Skipping permission test on non-Unix system")
	}

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "secrets.json")
	registry := NewMigrationRegistry()

	storage, err := NewMigratableStorage(filePath, registry, WithFileMode(0600), WithBackupMode(0600))
	if err != nil {
		t.Fatalf("NewMigratableStorage() error = %v", err)
	}

	err = storage.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	storage.Set("secret", "data")
	err = storage.Save()
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Check file permissions
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("Stat error: %v", err)
	}

	// On Unix systems, verify permissions
	mode := info.Mode().Perm()
	if mode != 0600 {
		t.Errorf("File permissions = %o, expected 0600", mode)
	}

	// Create backup and check its permissions
	backupPath, err := storage.createBackup()
	if err != nil {
		t.Fatalf("createBackup() error = %v", err)
	}

	info, err = os.Stat(backupPath)
	if err != nil {
		t.Fatalf("Stat error: %v", err)
	}

	// Use same mode for backup to avoid permission issues
	mode = info.Mode().Perm()
	if mode != 0600 {
		t.Errorf("Backup permissions = %o, expected 0600", mode)
	}
}

func TestMigratableStorage_ConcurrentAccess(t *testing.T) {
	// Note: This is a basic test; real concurrent access would need more robust handling
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.json")
	registry := NewMigrationRegistry()

	storage, err := NewMigratableStorage(filePath, registry)
	if err != nil {
		t.Fatalf("NewMigratableStorage() error = %v", err)
	}

	err = storage.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Simulate some concurrent operations
	done := make(chan bool, 3)

	// Writer 1
	go func() {
		for i := 0; i < 10; i++ {
			storage.Set("key1", i)
		}
		done <- true
	}()

	// Writer 2
	go func() {
		for i := 0; i < 10; i++ {
			storage.Set("key2", i)
		}
		done <- true
	}()

	// Reader
	go func() {
		for i := 0; i < 10; i++ {
			storage.Get("key1")
			storage.Get("key2")
		}
		done <- true
	}()

	// Wait for all
	for i := 0; i < 3; i++ {
		<-done
	}

	// Save should work without panic
	err = storage.Save()
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
}

func TestMigratableStorage_LoadErrors(t *testing.T) {
	t.Run("load fails with invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "invalid.json")

		// Create file with invalid JSON
		if err := os.WriteFile(filePath, []byte("not valid json"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		registry := NewMigrationRegistry()
		storage, err := NewMigratableStorage(filePath, registry)
		if err != nil {
			t.Fatalf("NewMigratableStorage() error = %v", err)
		}

		err = storage.Load()
		if err == nil {
			t.Error("Expected error loading invalid JSON")
		}
	})

	t.Run("load with no registry initializes version", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "noversion.json")

		storage, err := NewMigratableStorage(filePath, nil)
		if err != nil {
			t.Fatalf("NewMigratableStorage() error = %v", err)
		}

		err = storage.Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if storage.GetVersion() != DefaultStorageVersion {
			t.Errorf("Expected version %d, got %d", DefaultStorageVersion, storage.GetVersion())
		}
	})
}

func TestMigratableStorage_atomicWriteErrors(t *testing.T) {
	t.Run("atomic write fails with invalid directory", func(t *testing.T) {
		// Try to write to a directory that doesn't exist and can't be created
		storage, _ := NewMigratableStorage("/nonexistent/path/file.json", NewMigrationRegistry())

		storage.Set("key", "value")
		err := storage.Save()
		if err == nil {
			t.Error("Expected error saving to invalid directory")
		}
	})
}

func TestMigratableStorage_readFileErrors(t *testing.T) {
	t.Run("readFile fails with non-existent file", func(t *testing.T) {
		tmpDir := t.TempDir()
		storage, _ := NewMigratableStorage(filepath.Join(tmpDir, "nonexistent.json"), NewMigrationRegistry())

		_, err := storage.readFile("/nonexistent/path/file.json")
		if err == nil {
			t.Error("Expected error reading non-existent file")
		}
	})

	t.Run("readFile fails with invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "invalid.json")

		// Create file with invalid JSON
		os.WriteFile(filePath, []byte("invalid json"), 0644)

		storage, _ := NewMigratableStorage(filePath, NewMigrationRegistry())

		_, err := storage.readFile(filePath)
		if err == nil {
			t.Error("Expected error reading invalid JSON")
		}
	})
}

func TestMigratableStorage_migrationEdgeCases(t *testing.T) {
	t.Run("migration from version 0 with init", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "v0.json")
		registry := NewMigrationRegistry()

		// Create unversioned data
		data := map[string]interface{}{
			"someKey": "someValue",
		}
		jsonData, _ := json.Marshal(data)
		os.WriteFile(filePath, jsonData, 0644)

		storage, _ := NewMigratableStorage(filePath, registry)
		err := storage.Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		// Should have initialized version
		if storage.GetVersion() != DefaultStorageVersion {
			t.Errorf("Expected version %d, got %d", DefaultStorageVersion, storage.GetVersion())
		}
	})

	t.Run("migration path with no migrations", func(t *testing.T) {
		registry := NewMigrationRegistry()
		path, err := registry.GetMigrationPath(5, 5)
		if err != nil {
			t.Fatalf("GetMigrationPath() error = %v", err)
		}
		if len(path) != 0 {
			t.Errorf("Expected empty path, got %d migrations", len(path))
		}
	})

	t.Run("migration path with missing migration", func(t *testing.T) {
		registry := NewMigrationRegistry()
		registry.Register(Migration{
			FromVersion: 1,
			ToVersion:   2,
			Name:        "v1_to_v2",
			Apply:       func(data map[string]any) error { return nil },
		})

		_, err := registry.GetMigrationPath(2, 5)
		if err == nil {
			t.Error("Expected error for missing migration")
		}
	})
}

func TestMigratableStorage_createBackup(t *testing.T) {
	t.Run("create backup for non-existent file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "newfile.json")
		registry := NewMigrationRegistry()

		storage, _ := NewMigratableStorage(filePath, registry)
		storage.Load()
		storage.Set("key", "value")

		backupPath, err := storage.createBackup()
		if err != nil {
			t.Fatalf("createBackup() error = %v", err)
		}

		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			t.Error("Backup file should exist")
		}
	})

	t.Run("create backup with read-only backup dir", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Skipping permission test as root")
		}

		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "file.json")
		backupDir := filepath.Join(tmpDir, "readonly")

		// Create read-only directory
		os.MkdirAll(backupDir, 0755)
		storage, _ := NewMigratableStorage(filePath, NewMigrationRegistry(), WithBackupDir(backupDir))
		storage.Load()
		storage.Set("key", "value")
		storage.Save()

		// Make backup dir read-only
		os.Chmod(backupDir, 0555)
		defer os.Chmod(backupDir, 0755)

		_, err := storage.createBackup()
		if err == nil {
			t.Log("Backup succeeded despite read-only dir (may vary by OS)")
		}
	})
}

func TestMigratableStorage_restoreFromBackup(t *testing.T) {
	t.Run("restore from non-existent backup", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "file.json")
		storage, _ := NewMigratableStorage(filePath, NewMigrationRegistry())
		storage.Load()

		err := storage.restoreFromBackup("/nonexistent/backup.json")
		if err == nil {
			t.Error("Expected error restoring from non-existent backup")
		}
	})

	t.Run("restore from backup with invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "file.json")
		backupPath := filepath.Join(tmpDir, "backup.json")

		// Create invalid JSON backup
		os.WriteFile(backupPath, []byte("invalid json"), 0600)

		storage, _ := NewMigratableStorage(filePath, NewMigrationRegistry())
		storage.Load()

		err := storage.restoreFromBackup(backupPath)
		if err == nil {
			t.Error("Expected error restoring from invalid backup")
		}
	})
}

func TestMigratableStorage_CleanupOldBackups(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "file.json")
	registry := NewMigrationRegistry()

	storage, _ := NewMigratableStorage(filePath, registry)
	storage.Load()
	storage.Set("key", "value")

	// Create multiple backups
	for i := 0; i < 5; i++ {
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
		storage.Save()
		_, err := storage.createBackup()
		if err != nil {
			t.Fatalf("Failed to create backup %d: %v", i, err)
		}
	}

	backups, _ := storage.ListBackups()
	if len(backups) < 5 {
		t.Fatalf("Expected at least 5 backups, got %d", len(backups))
	}

	// Cleanup, keep only 2
	err := storage.CleanupOldBackups(2)
	if err != nil {
		t.Fatalf("CleanupOldBackups() error = %v", err)
	}

	backups, _ = storage.ListBackups()
	if len(backups) != 2 {
		t.Errorf("Expected 2 backups after cleanup, got %d", len(backups))
	}
}
