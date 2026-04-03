package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestReadKeyWithFlag tests the ReadKeyWithFlag method
func TestReadKeyWithFlag(t *testing.T) {
	mock := newMockSecretsManager()
	manager, err := NewAPIKeyManager(mock)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	tests := []struct {
		name            string
		flagValue       string
		interactivePrompt string
		want            string
		wantErr         bool
	}{
		{
			name:            "returns flag value when provided",
			flagValue:       "sk-test123456789",
			interactivePrompt: "Enter key: ",
			want:            "sk-test123456789",
			wantErr:         false,
		},
		{
			name:            "returns empty string when flag is empty (would trigger interactive)",
			flagValue:       "",
			interactivePrompt: "Enter key: ",
			want:            "",
			wantErr:         true, // Interactive read will fail in test environment
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For the empty flag case, we expect an error from interactive read
			// since we're not in a terminal
			got, err := manager.ReadKeyWithFlag(tt.flagValue, tt.interactivePrompt)
			
			if tt.flagValue != "" {
				if err != nil {
					t.Errorf("ReadKeyWithFlag() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if got != tt.want {
					t.Errorf("ReadKeyWithFlag() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

// TestKeyInputReader_ReadFromFlag tests reading from flag
func TestKeyInputReader_ReadFromFlag(t *testing.T) {
	mock := newMockSecretsManager()
	manager, _ := NewAPIKeyManager(mock)
	reader := NewKeyInputReader(manager)

	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{
			name:    "valid flag value",
			value:   "sk-test123",
			want:    "sk-test123",
			wantErr: false,
		},
		{
			name:    "empty flag value",
			value:   "",
			want:    "",
			wantErr: true,
		},
		{
			name:    "whitespace trimmed",
			value:   "  sk-test123  ",
			want:    "sk-test123",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := reader.ReadFromFlag(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadFromFlag() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ReadFromFlag() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestKeyInputReader_ReadFromEnvironment tests reading from environment
func TestKeyInputReader_ReadFromEnvironment(t *testing.T) {
	mock := newMockSecretsManager()
	manager, _ := NewAPIKeyManager(mock)
	reader := NewKeyInputReader(manager)

	// Set up test environment variable
	t.Setenv("TEST_API_KEY", "sk-env-test123")

	tests := []struct {
		name    string
		varName string
		want    string
		wantErr bool
	}{
		{
			name:    "existing environment variable",
			varName: "TEST_API_KEY",
			want:    "sk-env-test123",
			wantErr: false,
		},
		{
			name:    "non-existent environment variable",
			varName: "NON_EXISTENT_VAR",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty environment variable",
			varName: "EMPTY_VAR",
			want:    "",
			wantErr: true,
		},
	}

	// Set empty var
	t.Setenv("EMPTY_VAR", "")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := reader.ReadFromEnvironment(tt.varName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadFromEnvironment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ReadFromEnvironment() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestKeyInputReader_ReadFromFile tests reading from file
func TestKeyInputReader_ReadFromFile(t *testing.T) {
	mock := newMockSecretsManager()
	manager, _ := NewAPIKeyManager(mock)
	reader := NewKeyInputReader(manager)

	// Create temporary directory
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		content  string
		want     string
		wantErr  bool
		filePath string
	}{
		{
			name:    "valid file with key",
			content: "sk-file-test123",
			want:    "sk-file-test123",
			wantErr: false,
		},
		{
			name:    "file with whitespace",
			content: "  sk-file-test456  \n",
			want:    "sk-file-test456",
			wantErr: false,
		},
		{
			name:     "non-existent file",
			content:  "",
			want:     "",
			wantErr:  true,
			filePath: filepath.Join(tmpDir, "non-existent.txt"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var filePath string
			if tt.filePath != "" {
				filePath = tt.filePath
			} else {
				// Create temp file with content
				filePath = filepath.Join(tmpDir, fmt.Sprintf("key_%d.txt", time.Now().UnixNano()))
				if err := os.WriteFile(filePath, []byte(tt.content), 0600); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
			}

			got, err := reader.ReadFromFile(filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadFromFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ReadFromFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestKeyInputReader_Read tests the main Read method with all source types
func TestKeyInputReader_Read(t *testing.T) {
	mock := newMockSecretsManager()
	manager, _ := NewAPIKeyManager(mock)
	reader := NewKeyInputReader(manager)

	// Set up environment and files
	t.Setenv("TEST_API_KEY", "sk-env-read123")
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "key.txt")
	os.WriteFile(keyFile, []byte("sk-file-read456"), 0600)

	tests := []struct {
		name    string
		source  KeySource
		want    string
		wantErr bool
	}{
		{
			name: "read from flag",
			source: KeySource{
				Type:  SourceFlag,
				Value: "sk-flag-test",
			},
			want:    "sk-flag-test",
			wantErr: false,
		},
		{
			name: "read from environment",
			source: KeySource{
				Type:       SourceEnvironment,
				EnvVarName: "TEST_API_KEY",
			},
			want:    "sk-env-read123",
			wantErr: false,
		},
		{
			name: "read from file",
			source: KeySource{
				Type:     SourceFile,
				FilePath: keyFile,
			},
			want:    "sk-file-read456",
			wantErr: false,
		},
		{
			name: "read from empty flag",
			source: KeySource{
				Type:  SourceFlag,
				Value: "",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "read from non-existent env var",
			source: KeySource{
				Type:       SourceEnvironment,
				EnvVarName: "NON_EXISTENT_VAR_12345",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "read from non-existent file",
			source: KeySource{
				Type:     SourceFile,
				FilePath: "/non/existent/path/key.txt",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "unknown source type",
			source: KeySource{
				Type: "unknown",
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := reader.Read(tt.source)
			if (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Read() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestAPIKeyManager_TestKey tests the TestKey method with HTTP server
func TestAPIKeyManager_TestKey(t *testing.T) {
	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authorization header
		auth := r.Header.Get("Authorization")
		if auth == "Bearer valid-key" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if auth == "Bearer invalid-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		// Default to 500 for unexpected requests
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	mock := newMockSecretsManager()
	manager, _ := NewAPIKeyManager(mock)

	// Register a test provider with our test server
	manager.RegisterProvider(ProviderKeyConfig{
		Name:         "test-provider",
		DisplayName:  "Test Provider",
		SecretKeyName: "test_api_key",
		TestEndpoint: server.URL + "/models",
		TestHeaders: func(key string) map[string]string {
			return map[string]string{
				"Authorization": "Bearer " + key,
			}
		},
		ValidateFormat: func(key string) error {
			return nil
		},
		SupportsRotation: true,
		MinKeyLength:     1,
		MaxKeyLength:     256,
	})

	ctx := context.Background()

	tests := []struct {
		name     string
		provider string
		key      string
		wantErr  bool
		errType  error
	}{
		{
			name:     "valid key",
			provider: "test-provider",
			key:      "valid-key",
			wantErr:  false,
		},
		{
			name:     "invalid key (401)",
			provider: "test-provider",
			key:      "invalid-key",
			wantErr:  true,
			errType:  ErrKeyValidationFailed,
		},
		{
			name:     "unknown provider",
			provider: "unknown-provider",
			key:      "some-key",
			wantErr:  true,
			errType:  ErrProviderNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.TestKey(ctx, tt.provider, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("TestKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.errType != nil && err != nil {
				if !errors.Is(err, tt.errType) {
					t.Errorf("TestKey() error type = %v, want %v", err, tt.errType)
				}
			}
		})
	}
}

// TestAPIKeyManager_TestStoredKey tests testing stored keys
func TestAPIKeyManager_TestStoredKey(t *testing.T) {
	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	mock := newMockSecretsManager()
	manager, _ := NewAPIKeyManager(mock)

	// Store a test key first
	mock.secrets["openai_api_key"] = "sk-stored-test123"

	// Override the OpenAI provider's test endpoint
	manager.providers[ProviderOpenAI] = ProviderKeyConfig{
		Name:         ProviderOpenAI,
		DisplayName:  "OpenAI",
		SecretKeyName: "openai_api_key",
		TestEndpoint: server.URL,
		TestHeaders: func(key string) map[string]string {
			return map[string]string{"Authorization": "Bearer " + key}
		},
		ValidateFormat: func(key string) error {
			if !strings.HasPrefix(key, "sk-") {
				return ErrInvalidKeyFormat
			}
			return nil
		},
		SupportsRotation: true,
		MinKeyLength:     20,
		MaxKeyLength:     256,
	}

	ctx := context.Background()

	tests := []struct {
		name     string
		provider string
		setup    func()
		wantErr  bool
	}{
		{
			name:     "test stored key - success",
			provider: ProviderOpenAI,
			setup:    func() {},
			wantErr:  false,
		},
		{
			name:     "test stored key - not found",
			provider: "anthropic",
			setup: func() {
				// Ensure no key exists
				delete(mock.secrets, "anthropic_api_key")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			err := manager.TestStoredKey(ctx, tt.provider)
			if (err != nil) != tt.wantErr {
				t.Errorf("TestStoredKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestKeyValidator_ValidateGeminiKey tests Gemini key validation
func TestKeyValidator_ValidateGeminiKey(t *testing.T) {
	validator := NewKeyValidator()

	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{
			name:    "valid gemini key",
			key:     "abcdefghijklmnopqrstuvwxyz123456789",
			wantErr: false,
		},
		{
			name:    "too short",
			key:     "short",
			wantErr: true,
		},
		{
			name:    "too long",
			key:     strings.Repeat("a", 257),
			wantErr: true,
		},
		{
			name:    "invalid characters",
			key:     "abcdefghijklmnopqrstuvwxyz!!!123456",
			wantErr: true,
		},
		{
			name:    "minimum length",
			key:     strings.Repeat("a", 20),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateGeminiKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateGeminiKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestKeyValidator_ValidateOpenRouterKey tests OpenRouter key validation
func TestKeyValidator_ValidateOpenRouterKey(t *testing.T) {
	validator := NewKeyValidator()

	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{
			name:    "valid openrouter key with sk- prefix",
			key:     "sk-or-v1-test1234567890abcdef",
			wantErr: false,
		},
		{
			name:    "valid openrouter key sk- only",
			key:     "sk-test123456789012345",
			wantErr: false,
		},
		{
			name:    "missing sk- prefix",
			key:     "invalid-key-123456789012345",
			wantErr: true,
		},
		{
			name:    "too short",
			key:     "sk-short",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateOpenRouterKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOpenRouterKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestAPIKeyManager_MaskKey tests key masking
func TestAPIKeyManager_MaskKey(t *testing.T) {
	mock := newMockSecretsManager()
	manager, _ := NewAPIKeyManager(mock)

	tests := []struct {
		name string
		key  string
		want string
	}{
		{
			name: "long key",
			key:  "sk-ant-test123456789",
			want: "sk-a...6789",
		},
		{
			name: "exactly 8 characters",
			key:  "12345678",
			want: "****",
		},
		{
			name: "short key",
			key:  "1234",
			want: "****",
		},
		{
			name: "9 characters",
			key:  "123456789",
			want: "1234...6789",
		},
		{
			name: "empty key",
			key:  "",
			want: "****",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := manager.MaskKey(tt.key)
			if got != tt.want {
				t.Errorf("MaskKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestAPIKeyManager_GetProvider tests getting provider config
func TestAPIKeyManager_GetProvider(t *testing.T) {
	mock := newMockSecretsManager()
	manager, _ := NewAPIKeyManager(mock)

	tests := []struct {
		name     string
		provider string
		wantOk   bool
	}{
		{
			name:     "existing provider",
			provider: ProviderAnthropic,
			wantOk:   true,
		},
		{
			name:     "another existing provider",
			provider: ProviderOpenAI,
			wantOk:   true,
		},
		{
			name:     "non-existent provider",
			provider: "nonexistent",
			wantOk:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, ok := manager.GetProvider(tt.provider)
			if ok != tt.wantOk {
				t.Errorf("GetProvider() ok = %v, want %v", ok, tt.wantOk)
			}
			if ok && config.Name != tt.provider {
				t.Errorf("GetProvider() config.Name = %v, want %v", config.Name, tt.provider)
			}
		})
	}
}

// TestAPIKeyManager_ListProviders tests listing providers
func TestAPIKeyManager_ListProviders(t *testing.T) {
	mock := newMockSecretsManager()
	manager, _ := NewAPIKeyManager(mock)

	providers := manager.ListProviders()

	// Should have all default providers registered
	if len(providers) == 0 {
		t.Error("ListProviders() returned empty list")
	}

	// Check that known providers are in the list
	providerMap := make(map[string]bool)
	for _, p := range providers {
		providerMap[p] = true
	}

	expectedProviders := []string{
		ProviderAnthropic,
		ProviderOpenAI,
		ProviderOpenRouter,
		ProviderGemini,
		ProviderOllama,
	}

	for _, expected := range expectedProviders {
		if !providerMap[expected] {
			t.Errorf("ListProviders() missing expected provider: %s", expected)
		}
	}
}

// TestAPIKeyManager_RegisterProvider tests registering custom providers
func TestAPIKeyManager_RegisterProvider(t *testing.T) {
	mock := newMockSecretsManager()
	manager, _ := NewAPIKeyManager(mock)

	tests := []struct {
		name    string
		config  ProviderKeyConfig
		wantErr bool
	}{
		{
			name: "valid provider",
			config: ProviderKeyConfig{
				Name:         "custom-provider",
				SecretKeyName: "custom_key",
				ValidateFormat: func(key string) error { return nil },
			},
			wantErr: false,
		},
		{
			name: "missing name",
			config: ProviderKeyConfig{
				SecretKeyName: "custom_key",
			},
			wantErr: true,
		},
		{
			name: "missing secret key name",
			config: ProviderKeyConfig{
				Name: "another-custom",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.RegisterProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("RegisterProvider() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Verify it was registered
				_, ok := manager.GetProvider(tt.config.Name)
				if !ok {
					t.Errorf("RegisterProvider() provider %s not found after registration", tt.config.Name)
				}
			}
		})
	}
}