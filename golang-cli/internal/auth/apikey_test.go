package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cline/cline/golang-cli/internal/storage"
)

// mockSecretsManager is a mock implementation of SecretsStore for testing
type mockSecretsManager struct {
	secrets map[string]string
	listErr error
	getErr  error
	setErr  error
	delErr  error
}

func newMockSecretsManager() *mockSecretsManager {
	return &mockSecretsManager{
		secrets: make(map[string]string),
	}
}

func (m *mockSecretsManager) Get(key string) (string, error) {
	if m.getErr != nil {
		return "", m.getErr
	}
	value, ok := m.secrets[key]
	if !ok {
		return "", storage.ErrSecretNotFound
	}
	return value, nil
}

func (m *mockSecretsManager) Set(key string, value string) error {
	if m.setErr != nil {
		return m.setErr
	}
	m.secrets[key] = value
	return nil
}

func (m *mockSecretsManager) Delete(key string) error {
	if m.delErr != nil {
		return m.delErr
	}
	if _, ok := m.secrets[key]; !ok {
		return storage.ErrSecretNotFound
	}
	delete(m.secrets, key)
	return nil
}

func (m *mockSecretsManager) List() ([]string, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	keys := make([]string, 0, len(m.secrets))
	for k := range m.secrets {
		keys = append(keys, k)
	}
	return keys, nil
}

func (m *mockSecretsManager) Close() error {
	return nil
}

// TestNewAPIKeyManager tests the creation of APIKeyManager
func TestNewAPIKeyManager(t *testing.T) {
	t.Run("creates manager with valid secrets manager", func(t *testing.T) {
		mock := newMockSecretsManager()
		manager, err := NewAPIKeyManager(mock)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if manager == nil {
			t.Fatal("expected manager to not be nil")
		}
	})

	t.Run("returns error for nil secrets manager", func(t *testing.T) {
		manager, err := NewAPIKeyManager(nil)
		if err == nil {
			t.Fatal("expected error for nil secrets manager")
		}
		if manager != nil {
			t.Fatal("expected manager to be nil")
		}
	})

	t.Run("registers default providers", func(t *testing.T) {
		mock := newMockSecretsManager()
		manager, err := NewAPIKeyManager(mock)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		providers := manager.ListProviders()
		expectedProviders := []string{
			string(ProviderAnthropic),
			string(ProviderOpenAI),
			string(ProviderOpenRouter),
			string(ProviderGemini),
			string(ProviderBedrock),
			string(ProviderOllama),
			string(ProviderLMStudio),
		}

		if len(providers) != len(expectedProviders) {
			t.Fatalf("expected %d providers, got %d", len(expectedProviders), len(providers))
		}
	})
}

// TestRegisterProvider tests provider registration
func TestRegisterProvider(t *testing.T) {
	mock := newMockSecretsManager()
	manager, _ := NewAPIKeyManager(mock)

	t.Run("registers valid provider", func(t *testing.T) {
		config := ProviderKeyConfig{
			Name:          "test-provider",
			SecretKeyName: "test_key",
			ValidateFormat: func(key string) error {
				return nil
			},
		}
		err := manager.RegisterProvider(config)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		retrieved, ok := manager.GetProvider("test-provider")
		if !ok {
			t.Fatal("expected provider to be found")
		}
		if retrieved.Name != "test-provider" {
			t.Errorf("expected name 'test-provider', got %s", retrieved.Name)
		}
	})

	t.Run("returns error for empty name", func(t *testing.T) {
		config := ProviderKeyConfig{
			Name:          "",
			SecretKeyName: "test_key",
		}
		err := manager.RegisterProvider(config)
		if err == nil {
			t.Fatal("expected error for empty name")
		}
	})
}

// TestValidateKey tests key validation
func TestValidateKey(t *testing.T) {
	mock := newMockSecretsManager()
	manager, _ := NewAPIKeyManager(mock)

	t.Run("validates Anthropic key format", func(t *testing.T) {
		err := manager.ValidateKey(string(ProviderAnthropic), "sk-ant-api03-validkey12345678901234567890")
		if err != nil {
			t.Errorf("expected valid key, got error: %v", err)
		}

		// Key too short triggers length check before format validation
		err = manager.ValidateKey(string(ProviderAnthropic), "sk-ant-short")
		if !errors.Is(err, ErrKeyTooShort) {
			t.Errorf("expected ErrKeyTooShort, got %v", err)
		}

		// Wrong prefix but long enough key triggers format validation
		err = manager.ValidateKey(string(ProviderAnthropic), "invalid-prefix-key1234567890123456")
		if !errors.Is(err, ErrInvalidKeyFormat) {
			t.Errorf("expected ErrInvalidKeyFormat, got %v", err)
		}
	})

	t.Run("validates OpenAI key format", func(t *testing.T) {
		err := manager.ValidateKey(string(ProviderOpenAI), "sk-validkey1234567890123456789012345678")
		if err != nil {
			t.Errorf("expected valid key, got error: %v", err)
		}

		// Key too short triggers length check first
		err = manager.ValidateKey(string(ProviderOpenAI), "invalid-prefix")
		if !errors.Is(err, ErrKeyTooShort) {
			t.Errorf("expected ErrKeyTooShort, got %v", err)
		}

		// Wrong prefix but long enough key triggers format validation
		err = manager.ValidateKey(string(ProviderOpenAI), "invalid-prefix-key1234567890123456")
		if !errors.Is(err, ErrInvalidKeyFormat) {
			t.Errorf("expected ErrInvalidKeyFormat, got %v", err)
		}
	})

	t.Run("returns error for unknown provider", func(t *testing.T) {
		err := manager.ValidateKey("unknown-provider", "some-key")
		if !errors.Is(err, ErrProviderNotFound) {
			t.Errorf("expected ErrProviderNotFound, got %v", err)
		}
	})

	t.Run("Ollama accepts any key", func(t *testing.T) {
		err := manager.ValidateKey(string(ProviderOllama), "")
		if err != nil {
			t.Errorf("expected no error for Ollama, got %v", err)
		}
	})
}

// TestStoreKey tests storing API keys
func TestStoreKey(t *testing.T) {
	mock := newMockSecretsManager()
	manager, _ := NewAPIKeyManager(mock)

	t.Run("stores valid key", func(t *testing.T) {
		key := "sk-ant-api03-validkey12345678901234567890"
		err := manager.StoreKey(string(ProviderAnthropic), key, nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		storedKey, err := mock.Get("anthropic_api_key")
		if err != nil {
			t.Fatalf("expected key to be stored, got error: %v", err)
		}
		if storedKey != key {
			t.Errorf("expected stored key to match, got %s", storedKey)
		}
	})

	t.Run("returns error for invalid key", func(t *testing.T) {
		err := manager.StoreKey(string(ProviderAnthropic), "invalid-key", nil)
		if err == nil {
			t.Fatal("expected error for invalid key")
		}
	})
}

// TestGetKey tests retrieving API keys
func TestGetKey(t *testing.T) {
	mock := newMockSecretsManager()
	manager, _ := NewAPIKeyManager(mock)

	t.Run("retrieves stored key", func(t *testing.T) {
		expectedKey := "sk-ant-api03-validkey12345678901234567890"
		mock.secrets["anthropic_api_key"] = expectedKey

		key, err := manager.GetKey(string(ProviderAnthropic))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if key != expectedKey {
			t.Errorf("expected key %s, got %s", expectedKey, key)
		}
	})

	t.Run("returns error for missing key", func(t *testing.T) {
		delete(mock.secrets, "openai_api_key")

		_, err := manager.GetKey(string(ProviderOpenAI))
		if !errors.Is(err, ErrKeyNotFound) {
			t.Errorf("expected ErrKeyNotFound, got %v", err)
		}
	})
}

// TestDeleteKey tests deleting API keys
func TestDeleteKey(t *testing.T) {
	mock := newMockSecretsManager()
	manager, _ := NewAPIKeyManager(mock)

	t.Run("deletes stored key", func(t *testing.T) {
		mock.secrets["anthropic_api_key"] = "sk-ant-api03-validkey12345678901234567890"

		err := manager.DeleteKey(string(ProviderAnthropic))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		_, err = mock.Get("anthropic_api_key")
		if !errors.Is(err, storage.ErrSecretNotFound) {
			t.Error("expected key to be deleted")
		}
	})
}

// TestKeyExists tests checking if a key exists
func TestKeyExists(t *testing.T) {
	mock := newMockSecretsManager()
	manager, _ := NewAPIKeyManager(mock)

	t.Run("returns true for existing key", func(t *testing.T) {
		mock.secrets["anthropic_api_key"] = "some-key"

		exists, err := manager.KeyExists(string(ProviderAnthropic))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !exists {
			t.Error("expected key to exist")
		}
	})

	t.Run("returns false for missing key", func(t *testing.T) {
		delete(mock.secrets, "openai_api_key")

		exists, err := manager.KeyExists(string(ProviderOpenAI))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if exists {
			t.Error("expected key to not exist")
		}
	})
}

// TestTestKey tests the key testing functionality
func TestTestKey(t *testing.T) {
	mock := newMockSecretsManager()
	manager, _ := NewAPIKeyManager(mock)

	t.Run("returns error for unknown provider", func(t *testing.T) {
		ctx := context.Background()
		err := manager.TestKey(ctx, "unknown-provider", "some-key")
		if !errors.Is(err, ErrProviderNotFound) {
			t.Errorf("expected ErrProviderNotFound, got %v", err)
		}
	})

	t.Run("skips test for local providers", func(t *testing.T) {
		ctx := context.Background()
		
		err := manager.TestKey(ctx, string(ProviderOllama), "any-key")
		if err != nil {
			t.Errorf("expected no error for Ollama, got %v", err)
		}
	})
}

// TestRotateKey tests key rotation
func TestRotateKey(t *testing.T) {
	mock := newMockSecretsManager()
	manager, _ := NewAPIKeyManager(mock)

	t.Run("rotates key for supported provider", func(t *testing.T) {
		oldKey := "sk-ant-api03-oldkey1234567890123456789012"
		newKey := "sk-ant-api03-newkey1234567890123456789012"
		
		mock.secrets["anthropic_api_key"] = oldKey

		rotationInfo, err := manager.RotateKey(string(ProviderAnthropic), newKey, 24*time.Hour)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if rotationInfo.OldKey != oldKey {
			t.Errorf("expected old key to be set")
		}
		if rotationInfo.NewKey != newKey {
			t.Errorf("expected new key to be set")
		}

		storedKey, err := mock.Get("anthropic_api_key")
		if err != nil {
			t.Fatalf("expected key to be stored, got error: %v", err)
		}
		if storedKey != newKey {
			t.Errorf("expected new key to be stored, got %s", storedKey)
		}
	})

	t.Run("returns error for unsupported provider", func(t *testing.T) {
		_, err := manager.RotateKey(string(ProviderBedrock), "new-key", 24*time.Hour)
		if !errors.Is(err, ErrRotationNotSupported) {
			t.Errorf("expected ErrRotationNotSupported, got %v", err)
		}
	})
}

// TestMaskKey tests key masking
func TestMaskKey(t *testing.T) {
	mock := newMockSecretsManager()
	manager, _ := NewAPIKeyManager(mock)

	tests := []struct {
		input    string
		expected string
	}{
		{"sk-ant-api03-validkey12345", "sk-a...2345"},
		{"short", "****"},
		{"", "****"},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("masks %d char key", len(test.input)), func(t *testing.T) {
			masked := manager.MaskKey(test.input)
			if masked != test.expected {
				t.Errorf("expected %s, got %s", test.expected, masked)
			}
		})
	}
}

// TestKeyValidator tests the standalone key validator
func TestKeyValidator(t *testing.T) {
	validator := NewKeyValidator()

	t.Run("validates Anthropic key", func(t *testing.T) {
		err := validator.ValidateAnthropicKey("sk-ant-api03-validkey12345678901234567890")
		if err != nil {
			t.Errorf("expected valid key, got error: %v", err)
		}

		err = validator.ValidateAnthropicKey("sk-invalid")
		if err == nil {
			t.Error("expected error for invalid prefix")
		}
	})

	t.Run("validates OpenAI key", func(t *testing.T) {
		err := validator.ValidateOpenAIKey("sk-validkey1234567890123456789012345678")
		if err != nil {
			t.Errorf("expected valid key, got error: %v", err)
		}

		err = validator.ValidateOpenAIKey("invalid-prefix")
		if err == nil {
			t.Error("expected error for invalid prefix")
		}
	})
}

// TestKeyInputReader tests the key input reader
func TestKeyInputReader(t *testing.T) {
	mock := newMockSecretsManager()
	manager, _ := NewAPIKeyManager(mock)
	reader := NewKeyInputReader(manager)

	t.Run("reads from flag", func(t *testing.T) {
		key, err := reader.ReadFromFlag("  sk-valid-key12345678901234567890  ")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if key != "sk-valid-key12345678901234567890" {
			t.Errorf("expected trimmed key, got %s", key)
		}
	})

	t.Run("reads from environment", func(t *testing.T) {
		os.Setenv("TEST_API_KEY", "sk-env-key12345678901234567890")
		defer os.Unsetenv("TEST_API_KEY")

		key, err := reader.ReadFromEnvironment("TEST_API_KEY")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if key != "sk-env-key12345678901234567890" {
			t.Errorf("expected key from env, got %s", key)
		}
	})

	t.Run("reads from file", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "apikey.txt")
		
		content := "sk-file-key12345678901234567890\n"
		if err := os.WriteFile(tmpFile, []byte(content), 0600); err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}

		key, err := reader.ReadFromFile(tmpFile)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if key != "sk-file-key12345678901234567890" {
			t.Errorf("expected key from file, got %s", key)
		}
	})

	t.Run("reads from source", func(t *testing.T) {
		source := KeySource{
			Type:  SourceFlag,
			Value: "sk-source-key12345678901234567890",
		}
		key, err := reader.Read(source)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if key != "sk-source-key12345678901234567890" {
			t.Errorf("expected key from source, got %s", key)
		}
	})
}

// TestIntegration tests the integration of multiple operations
func TestIntegration(t *testing.T) {
	mock := newMockSecretsManager()
	manager, _ := NewAPIKeyManager(mock)

	t.Run("full key lifecycle", func(t *testing.T) {
		key := "sk-ant-api03-lifecycle12345678901234567890"
		metadata := &KeyMetadata{
			Description: "Lifecycle test key",
		}

		err := manager.StoreKey(string(ProviderAnthropic), key, metadata)
		if err != nil {
			t.Fatalf("failed to store key: %v", err)
		}

		exists, err := manager.KeyExists(string(ProviderAnthropic))
		if err != nil {
			t.Fatalf("failed to check key exists: %v", err)
		}
		if !exists {
			t.Fatal("expected key to exist")
		}

		retrieved, err := manager.GetKey(string(ProviderAnthropic))
		if err != nil {
			t.Fatalf("failed to get key: %v", err)
		}
		if retrieved != key {
			t.Errorf("expected %s, got %s", key, retrieved)
		}

		err = manager.DeleteKey(string(ProviderAnthropic))
		if err != nil {
			t.Fatalf("failed to delete key: %v", err)
		}

		exists, _ = manager.KeyExists(string(ProviderAnthropic))
		if exists {
			t.Error("expected key to not exist after deletion")
		}
	})
}
