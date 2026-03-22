package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// mockKeyring is a mock implementation of keyringProvider for testing
type mockKeyring struct {
	data      map[string]string
	available bool
}

func newMockKeyring(available bool) *mockKeyring {
	return &mockKeyring{
		data:      make(map[string]string),
		available: available,
	}
}

func (m *mockKeyring) Get(service, key string) (string, error) {
	if !m.available {
		return "", ErrKeyringUnavailable
	}
	value, ok := m.data[key]
	if !ok {
		return "", ErrSecretNotFound
	}
	return value, nil
}

func (m *mockKeyring) Set(service, key, value string) error {
	if !m.available {
		return ErrKeyringUnavailable
	}
	m.data[key] = value
	return nil
}

func (m *mockKeyring) Delete(service, key string) error {
	if !m.available {
		return ErrKeyringUnavailable
	}
	if _, ok := m.data[key]; !ok {
		return ErrSecretNotFound
	}
	delete(m.data, key)
	return nil
}

func (m *mockKeyring) List(service string) ([]string, error) {
	if !m.available {
		return nil, ErrKeyringUnavailable
	}
	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	return keys, nil
}

func (m *mockKeyring) IsAvailable() bool {
	return m.available
}

func TestNewSecretsManager(t *testing.T) {
	t.Run("creates manager with custom config dir", func(t *testing.T) {
		tmpDir := t.TempDir()
		sm, err := NewSecretsManager(SecretsManagerOptions{
			ConfigDir: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create SecretsManager: %v", err)
		}
		defer sm.Close()

		if sm.fallbackPath != filepath.Join(tmpDir, fallbackFileName) {
			t.Errorf("Expected fallback path %s, got %s", filepath.Join(tmpDir, fallbackFileName), sm.fallbackPath)
		}
	})

	t.Run("creates default config dir", func(t *testing.T) {
		homeDir, _ := os.UserHomeDir()
		sm, err := NewSecretsManager(SecretsManagerOptions{})
		if err != nil {
			t.Fatalf("Failed to create SecretsManager: %v", err)
		}
		defer sm.Close()

		expectedPath := filepath.Join(homeDir, ".cline", fallbackFileName)
		if sm.fallbackPath != expectedPath {
			t.Errorf("Expected fallback path %s, got %s", expectedPath, sm.fallbackPath)
		}
	})

	t.Run("calls warning handler when keyring unavailable", func(t *testing.T) {
		tmpDir := t.TempDir()
		warningCalled := false
		var warningMsg string

		// Create manager with mock keyring that simulates unavailability
		sm, err := NewSecretsManager(SecretsManagerOptions{
			ConfigDir: tmpDir,
			WarningHandler: func(msg string) {
				warningCalled = true
				warningMsg = msg
			},
		})
		if err != nil {
			t.Fatalf("Failed to create SecretsManager: %v", err)
		}
		defer sm.Close()

		// On platforms where keyring might not be available, warning should be called
		if !sm.useKeyring && !warningCalled {
			t.Error("Warning handler should have been called when keyring is unavailable")
		}

		if warningCalled && warningMsg == "" {
			t.Error("Warning message should not be empty")
		}
	})
}

func TestSecretsManager_SetAndGet(t *testing.T) {
	t.Run("set and get secret with keyring", func(t *testing.T) {
		tmpDir := t.TempDir()
		sm, err := NewSecretsManager(SecretsManagerOptions{
			ConfigDir: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create SecretsManager: %v", err)
		}
		defer sm.Close()

		// Use mock keyring for testing
		mock := newMockKeyring(true)
		sm.provider = mock
		sm.useKeyring = true

		// Set a secret
		if err := sm.Set("test-key", "test-value"); err != nil {
			t.Fatalf("Failed to set secret: %v", err)
		}

		// Get the secret
		value, err := sm.Get("test-key")
		if err != nil {
			t.Fatalf("Failed to get secret: %v", err)
		}

		if value != "test-value" {
			t.Errorf("Expected 'test-value', got '%s'", value)
		}
	})

	t.Run("set and get secret with file fallback", func(t *testing.T) {
		tmpDir := t.TempDir()
		sm, err := NewSecretsManager(SecretsManagerOptions{
			ConfigDir: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create SecretsManager: %v", err)
		}
		defer sm.Close()

		// Force file fallback
		sm.useKeyring = false

		// Set a secret
		if err := sm.Set("test-key", "test-value"); err != nil {
			t.Fatalf("Failed to set secret: %v", err)
		}

		// Get the secret
		value, err := sm.Get("test-key")
		if err != nil {
			t.Fatalf("Failed to get secret: %v", err)
		}

		if value != "test-value" {
			t.Errorf("Expected 'test-value', got '%s'", value)
		}
	})

	t.Run("get non-existent secret returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		sm, err := NewSecretsManager(SecretsManagerOptions{
			ConfigDir: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create SecretsManager: %v", err)
		}
		defer sm.Close()

		sm.useKeyring = false

		_, err = sm.Get("non-existent-key")
		if !errors.Is(err, ErrSecretNotFound) {
			t.Errorf("Expected ErrSecretNotFound, got %v", err)
		}
	})

	t.Run("empty key returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		sm, err := NewSecretsManager(SecretsManagerOptions{
			ConfigDir: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create SecretsManager: %v", err)
		}
		defer sm.Close()

		err = sm.Set("", "value")
		if err == nil {
			t.Error("Expected error for empty key, got nil")
		}
	})
}

func TestSecretsManager_Delete(t *testing.T) {
	t.Run("delete existing secret", func(t *testing.T) {
		tmpDir := t.TempDir()
		sm, err := NewSecretsManager(SecretsManagerOptions{
			ConfigDir: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create SecretsManager: %v", err)
		}
		defer sm.Close()

		sm.useKeyring = false

		// Set and then delete
		if err := sm.Set("delete-key", "value"); err != nil {
			t.Fatalf("Failed to set secret: %v", err)
		}

		if err := sm.Delete("delete-key"); err != nil {
			t.Fatalf("Failed to delete secret: %v", err)
		}

		// Verify deletion
		_, err = sm.Get("delete-key")
		if !errors.Is(err, ErrSecretNotFound) {
			t.Errorf("Expected ErrSecretNotFound after delete, got %v", err)
		}
	})

	t.Run("delete non-existent secret returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		sm, err := NewSecretsManager(SecretsManagerOptions{
			ConfigDir: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create SecretsManager: %v", err)
		}
		defer sm.Close()

		sm.useKeyring = false

		err = sm.Delete("non-existent-key")
		if !errors.Is(err, ErrSecretNotFound) {
			t.Errorf("Expected ErrSecretNotFound, got %v", err)
		}
	})

	t.Run("empty key returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		sm, err := NewSecretsManager(SecretsManagerOptions{
			ConfigDir: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create SecretsManager: %v", err)
		}
		defer sm.Close()

		err = sm.Delete("")
		if err == nil {
			t.Error("Expected error for empty key, got nil")
		}
	})
}

func TestSecretsManager_List(t *testing.T) {
	t.Run("list secrets with file fallback", func(t *testing.T) {
		tmpDir := t.TempDir()
		sm, err := NewSecretsManager(SecretsManagerOptions{
			ConfigDir: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create SecretsManager: %v", err)
		}
		defer sm.Close()

		sm.useKeyring = false

		// Add some secrets
		secrets := map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		}

		for k, v := range secrets {
			if err := sm.Set(k, v); err != nil {
				t.Fatalf("Failed to set secret: %v", err)
			}
		}

		// List secrets
		keys, err := sm.List()
		if err != nil {
			t.Fatalf("Failed to list secrets: %v", err)
		}

		if len(keys) != len(secrets) {
			t.Errorf("Expected %d keys, got %d", len(secrets), len(keys))
		}

		// Verify all keys are present
		keyMap := make(map[string]bool)
		for _, k := range keys {
			keyMap[k] = true
		}

		for k := range secrets {
			if !keyMap[k] {
				t.Errorf("Expected key '%s' not found in list", k)
			}
		}
	})

	t.Run("list empty secrets", func(t *testing.T) {
		tmpDir := t.TempDir()
		sm, err := NewSecretsManager(SecretsManagerOptions{
			ConfigDir: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create SecretsManager: %v", err)
		}
		defer sm.Close()

		sm.useKeyring = false

		keys, err := sm.List()
		if err != nil {
			t.Fatalf("Failed to list secrets: %v", err)
		}

		if len(keys) != 0 {
			t.Errorf("Expected empty list, got %d keys", len(keys))
		}
	})
}

func TestSecretsManager_Encryption(t *testing.T) {
	t.Run("encrypt and decrypt roundtrip", func(t *testing.T) {
		tmpDir := t.TempDir()
		sm, err := NewSecretsManager(SecretsManagerOptions{
			ConfigDir: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create SecretsManager: %v", err)
		}
		defer sm.Close()

		plaintext := "sensitive-data-12345"
		encrypted, err := sm.encrypt(plaintext)
		if err != nil {
			t.Fatalf("Failed to encrypt: %v", err)
		}

		// Verify encrypted data is different from plaintext
		if encrypted == plaintext {
			t.Error("Encrypted data should differ from plaintext")
		}

		decrypted, err := sm.decrypt(encrypted)
		if err != nil {
			t.Fatalf("Failed to decrypt: %v", err)
		}

		if decrypted != plaintext {
			t.Errorf("Expected '%s', got '%s'", plaintext, decrypted)
		}
	})

	t.Run("decrypt invalid data returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		sm, err := NewSecretsManager(SecretsManagerOptions{
			ConfigDir: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create SecretsManager: %v", err)
		}
		defer sm.Close()

		_, err = sm.decrypt("invalid-base64!!!")
		if err == nil {
			t.Error("Expected error for invalid base64 data")
		}
	})

	t.Run("decrypt too short data returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		sm, err := NewSecretsManager(SecretsManagerOptions{
			ConfigDir: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create SecretsManager: %v", err)
		}
		defer sm.Close()

		// Create base64 encoded data that's too short (less than nonce length)
		shortData := "dGVzdA==" // "test" in base64
		_, err = sm.decrypt(shortData)
		if err == nil {
			t.Error("Expected error for short ciphertext")
		}
	})
}

func TestSecretsManager_FileOperations(t *testing.T) {
	t.Run("file created with correct permissions", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Skipping permission test on Windows")
		}

		tmpDir := t.TempDir()
		sm, err := NewSecretsManager(SecretsManagerOptions{
			ConfigDir: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create SecretsManager: %v", err)
		}
		defer sm.Close()

		sm.useKeyring = false

		// Set a secret to create the file
		if err := sm.Set("key", "value"); err != nil {
			t.Fatalf("Failed to set secret: %v", err)
		}

		// Check file permissions
		info, err := os.Stat(sm.fallbackPath)
		if err != nil {
			t.Fatalf("Failed to stat file: %v", err)
		}

		// Should be 0600 (owner read/write only)
		mode := info.Mode().Perm()
		if mode != 0600 {
			t.Errorf("Expected permissions 0600, got %04o", mode)
		}
	})

	t.Run("corrupted file returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		sm, err := NewSecretsManager(SecretsManagerOptions{
			ConfigDir: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create SecretsManager: %v", err)
		}
		defer sm.Close()

		sm.useKeyring = false

		// Create corrupted file
		corruptedData := []byte("not-valid-json{")
		if err := os.WriteFile(sm.fallbackPath, corruptedData, 0600); err != nil {
			t.Fatalf("Failed to write corrupted file: %v", err)
		}

		_, err = sm.Get("key")
		if err == nil {
			t.Error("Expected error for corrupted file")
		}
	})
}

func TestSecretsManager_FallbackEncryptionKey(t *testing.T) {
	t.Run("encryption key is deterministic", func(t *testing.T) {
		tmpDir := t.TempDir()
		sm, err := NewSecretsManager(SecretsManagerOptions{
			ConfigDir: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create SecretsManager: %v", err)
		}
		defer sm.Close()

		key1, err := sm.FallbackEncryptionKey()
		if err != nil {
			t.Fatalf("Failed to get encryption key: %v", err)
		}

		key2, err := sm.FallbackEncryptionKey()
		if err != nil {
			t.Fatalf("Failed to get encryption key: %v", err)
		}

		// Same machine should produce same key
		if string(key1) != string(key2) {
			t.Error("Encryption key should be deterministic")
		}

		// Key should be 32 bytes (AES-256)
		if len(key1) != 32 {
			t.Errorf("Expected key length 32, got %d", len(key1))
		}
	})
}

func TestSanitizeKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple-key", "simple-key"},
		{"key/with/slashes", "key_with_slashes"},
		{"key\\with\\backslashes", "key_with_backslashes"},
		{"key:with:colons", "key_with_colons"},
		{"mixed/key\\with:all", "mixed_key_with_all"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeKey(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeKey(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSecretsManager_IsUsingKeyring(t *testing.T) {
	t.Run("returns true when using keyring", func(t *testing.T) {
		tmpDir := t.TempDir()
		sm, err := NewSecretsManager(SecretsManagerOptions{
			ConfigDir: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create SecretsManager: %v", err)
		}
		defer sm.Close()

		// Set up mock keyring
		mock := newMockKeyring(true)
		sm.provider = mock
		sm.useKeyring = true

		if !sm.IsUsingKeyring() {
			t.Error("IsUsingKeyring() should return true")
		}
	})

	t.Run("returns false when using fallback", func(t *testing.T) {
		tmpDir := t.TempDir()
		sm, err := NewSecretsManager(SecretsManagerOptions{
			ConfigDir: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create SecretsManager: %v", err)
		}
		defer sm.Close()

		sm.useKeyring = false

		if sm.IsUsingKeyring() {
			t.Error("IsUsingKeyring() should return false")
		}
	})
}

func TestSecretsManager_MultipleSecrets(t *testing.T) {
	t.Run("store and retrieve multiple secrets", func(t *testing.T) {
		tmpDir := t.TempDir()
		sm, err := NewSecretsManager(SecretsManagerOptions{
			ConfigDir: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create SecretsManager: %v", err)
		}
		defer sm.Close()

		sm.useKeyring = false

		secrets := map[string]string{
			"api-key":      "secret-api-key-12345",
			"password":     "my-super-secret-password",
			"token":        "bearer-token-abc",
			"long-value":   "this-is-a-very-long-value-with-many-characters-and-special-symbols-!@#$%^&*()",
			"unicode":      "unicode-文字-🎉-émojis",
		}

		// Store all secrets
		for k, v := range secrets {
			if err := sm.Set(k, v); err != nil {
				t.Fatalf("Failed to set secret %s: %v", k, err)
			}
		}

		// Retrieve and verify all secrets
		for k, expected := range secrets {
			actual, err := sm.Get(k)
			if err != nil {
				t.Fatalf("Failed to get secret %s: %v", k, err)
			}
			if actual != expected {
				t.Errorf("Secret %s: expected %q, got %q", k, expected, actual)
			}
		}
	})
}

func TestSecretsManager_Persistence(t *testing.T) {
	t.Run("secrets persist across manager instances", func(t *testing.T) {
		tmpDir := t.TempDir()

		// First manager instance
		sm1, err := NewSecretsManager(SecretsManagerOptions{
			ConfigDir: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create first SecretsManager: %v", err)
		}

		sm1.useKeyring = false

		// Set some secrets
		if err := sm1.Set("persistent-key", "persistent-value"); err != nil {
			t.Fatalf("Failed to set secret: %v", err)
		}

		sm1.Close()

		// Second manager instance
		sm2, err := NewSecretsManager(SecretsManagerOptions{
			ConfigDir: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create second SecretsManager: %v", err)
		}
		defer sm2.Close()

		sm2.useKeyring = false

		// Retrieve secret from new instance
		value, err := sm2.Get("persistent-key")
		if err != nil {
			t.Fatalf("Failed to get persistent secret: %v", err)
		}

		if value != "persistent-value" {
			t.Errorf("Expected 'persistent-value', got '%s'", value)
		}
	})
}

func TestSecretsManager_FileFormat(t *testing.T) {
	t.Run("file contains valid JSON structure", func(t *testing.T) {
		tmpDir := t.TempDir()
		sm, err := NewSecretsManager(SecretsManagerOptions{
			ConfigDir: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create SecretsManager: %v", err)
		}
		defer sm.Close()

		sm.useKeyring = false

		// Set a secret
		if err := sm.Set("test-key", "test-value"); err != nil {
			t.Fatalf("Failed to set secret: %v", err)
		}

		// Read and parse file
		data, err := os.ReadFile(sm.fallbackPath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		var fileData fallbackData
		if err := json.Unmarshal(data, &fileData); err != nil {
			t.Fatalf("Failed to unmarshal file: %v", err)
		}

		// Verify structure
		if fileData.Version != 1 {
			t.Errorf("Expected version 1, got %d", fileData.Version)
		}

		if fileData.Secrets == nil {
			t.Error("Secrets map should not be nil")
		}

		// Verify encrypted value exists
		encryptedValue, ok := fileData.Secrets["test-key"]
		if !ok {
			t.Error("test-key should exist in secrets")
		}

		// Verify value is encrypted (not plaintext)
		if encryptedValue == "test-value" {
			t.Error("Stored value should be encrypted, not plaintext")
		}
	})
}

func TestSecretsManager_UpdateExisting(t *testing.T) {
	t.Run("update existing secret", func(t *testing.T) {
		tmpDir := t.TempDir()
		sm, err := NewSecretsManager(SecretsManagerOptions{
			ConfigDir: tmpDir,
		})
		if err != nil {
			t.Fatalf("Failed to create SecretsManager: %v", err)
		}
		defer sm.Close()

		sm.useKeyring = false

		// Set initial value
		if err := sm.Set("update-key", "initial-value"); err != nil {
			t.Fatalf("Failed to set initial value: %v", err)
		}

		// Update value
		if err := sm.Set("update-key", "updated-value"); err != nil {
			t.Fatalf("Failed to update value: %v", err)
		}

		// Verify update
		value, err := sm.Get("update-key")
		if err != nil {
			t.Fatalf("Failed to get updated value: %v", err)
		}

		if value != "updated-value" {
			t.Errorf("Expected 'updated-value', got '%s'", value)
		}
	})
}

func BenchmarkEncrypt(b *testing.B) {
	tmpDir := b.TempDir()
	sm, err := NewSecretsManager(SecretsManagerOptions{
		ConfigDir: tmpDir,
	})
	if err != nil {
		b.Fatalf("Failed to create SecretsManager: %v", err)
	}
	defer sm.Close()

	plaintext := "benchmark-secret-value"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := sm.encrypt(plaintext)
		if err != nil {
			b.Fatalf("Failed to encrypt: %v", err)
		}
	}
}

func BenchmarkDecrypt(b *testing.B) {
	tmpDir := b.TempDir()
	sm, err := NewSecretsManager(SecretsManagerOptions{
		ConfigDir: tmpDir,
	})
	if err != nil {
		b.Fatalf("Failed to create SecretsManager: %v", err)
	}
	defer sm.Close()

	plaintext := "benchmark-secret-value"
	encrypted, err := sm.encrypt(plaintext)
	if err != nil {
		b.Fatalf("Failed to encrypt: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := sm.decrypt(encrypted)
		if err != nil {
			b.Fatalf("Failed to decrypt: %v", err)
		}
	}
}

func BenchmarkSet(b *testing.B) {
	tmpDir := b.TempDir()
	sm, err := NewSecretsManager(SecretsManagerOptions{
		ConfigDir: tmpDir,
	})
	if err != nil {
		b.Fatalf("Failed to create SecretsManager: %v", err)
	}
	defer sm.Close()

	sm.useKeyring = false

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		if err := sm.Set(key, "value"); err != nil {
			b.Fatalf("Failed to set: %v", err)
		}
	}
}

func BenchmarkGet(b *testing.B) {
	tmpDir := b.TempDir()
	sm, err := NewSecretsManager(SecretsManagerOptions{
		ConfigDir: tmpDir,
	})
	if err != nil {
		b.Fatalf("Failed to create SecretsManager: %v", err)
	}
	defer sm.Close()

	sm.useKeyring = false

	// Pre-populate
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key-%d", i)
		if err := sm.Set(key, "value"); err != nil {
			b.Fatalf("Failed to set: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%100)
		if _, err := sm.Get(key); err != nil {
			b.Fatalf("Failed to get: %v", err)
		}
	}
}