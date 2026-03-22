// Package auth provides API key management and authentication utilities
// for the Cline CLI. It supports interactive input with masking, flag-based
// input, key format validation, encrypted storage, and key rotation.
package auth

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/cline/cline/golang-cli/internal/storage"
	"golang.org/x/term"
)

// APIKeyManager provides comprehensive API key management functionality
type APIKeyManager struct {
	secretsManager storage.SecretsStore
	providers      map[string]ProviderKeyConfig
}

// ProviderKeyConfig defines key validation and storage configuration for a provider
type ProviderKeyConfig struct {
	// Name is the provider identifier
	Name string

	// DisplayName is the human-readable provider name
	DisplayName string

	// ValidateFormat is a function to validate the key format
	ValidateFormat func(key string) error

	// TestEndpoint is the URL to test the key (optional)
	TestEndpoint string

	// TestHeaders returns headers needed for testing
	TestHeaders func(key string) map[string]string

	// SecretKeyName is the name used in secrets storage
	SecretKeyName string

	// SupportsRotation indicates if the provider supports key rotation
	SupportsRotation bool

	// MaxKeyLength is the maximum allowed key length
	MaxKeyLength int

	// MinKeyLength is the minimum allowed key length
	MinKeyLength int
}

// KeyMetadata stores metadata about a stored key
type KeyMetadata struct {
	Provider    string    `json:"provider"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	LastUsedAt  time.Time `json:"last_used_at"`
	Description string    `json:"description,omitempty"`
	IsActive    bool      `json:"is_active"`
	KeyVersion  int       `json:"key_version"`
}

// KeyRotationInfo stores information about key rotation
type KeyRotationInfo struct {
	OldKey       string    `json:"old_key"`
	NewKey       string    `json:"new_key"`
	RotatedAt    time.Time `json:"rotated_at"`
	OldKeyExpiry time.Time `json:"old_key_expiry,omitempty"`
}

// Common errors
var (
	ErrInvalidKeyFormat     = errors.New("invalid API key format")
	ErrKeyTooShort          = errors.New("API key is too short")
	ErrKeyTooLong           = errors.New("API key is too long")
	ErrKeyNotFound          = errors.New("API key not found")
	ErrProviderNotFound     = errors.New("provider not found")
	ErrKeyExists            = errors.New("API key already exists")
	ErrRotationNotSupported = errors.New("key rotation not supported for this provider")
	ErrKeyValidationFailed  = errors.New("API key validation failed")
	ErrKeyTestFailed        = errors.New("API key test failed")
)


// NewAPIKeyManager creates a new API key manager with the given secrets manager
func NewAPIKeyManager(secretsManager storage.SecretsStore) (*APIKeyManager, error) {
	if secretsManager == nil {
		return nil, errors.New("secrets manager is required")
	}

	manager := &APIKeyManager{
		secretsManager: secretsManager,
		providers:      make(map[string]ProviderKeyConfig),
	}

	// Register default providers
	manager.registerDefaultProviders()

	return manager, nil
}

// registerDefaultProviders registers the built-in provider configurations
func (m *APIKeyManager) registerDefaultProviders() {
	// Anthropic
	m.providers[ProviderAnthropic] = ProviderKeyConfig{
		Name:        ProviderAnthropic,
		DisplayName: "Anthropic",
		ValidateFormat: func(key string) error {
			// Anthropic keys start with "sk-ant-" followed by alphanumeric characters
			if !strings.HasPrefix(key, "sk-ant-") {
				return fmt.Errorf("%w: Anthropic keys must start with 'sk-ant-'", ErrInvalidKeyFormat)
			}
			if len(key) < 20 {
				return fmt.Errorf("%w: minimum length 20 characters", ErrKeyTooShort)
			}
			return nil
		},
		TestEndpoint: "https://api.anthropic.com/v1/models",
		TestHeaders: func(key string) map[string]string {
			return map[string]string{
				"x-api-key":         key,
				"anthropic-version": "2023-06-01",
			}
		},
		SecretKeyName:    "anthropic_api_key",
		SupportsRotation: true,
		MinKeyLength:     20,
		MaxKeyLength:     256,
	}

	// OpenAI
	m.providers[ProviderOpenAI] = ProviderKeyConfig{
		Name:        ProviderOpenAI,
		DisplayName: "OpenAI",
		ValidateFormat: func(key string) error {
			// OpenAI keys start with "sk-" followed by alphanumeric characters
			if !strings.HasPrefix(key, "sk-") {
				return fmt.Errorf("%w: OpenAI keys must start with 'sk-'", ErrInvalidKeyFormat)
			}
			if len(key) < 20 {
				return fmt.Errorf("%w: minimum length 20 characters", ErrKeyTooShort)
			}
			return nil
		},
		TestEndpoint: "https://api.openai.com/v1/models",
		TestHeaders: func(key string) map[string]string {
			return map[string]string{
				"Authorization": "Bearer " + key,
			}
		},
		SecretKeyName:    "openai_api_key",
		SupportsRotation: true,
		MinKeyLength:     20,
		MaxKeyLength:     256,
	}

	// OpenRouter
	m.providers[ProviderOpenRouter] = ProviderKeyConfig{
		Name:        ProviderOpenRouter,
		DisplayName: "OpenRouter",
		ValidateFormat: func(key string) error {
			// OpenRouter keys typically start with "sk-or-" or just "sk-"
			if !strings.HasPrefix(key, "sk-") {
				return fmt.Errorf("%w: OpenRouter keys must start with 'sk-'", ErrInvalidKeyFormat)
			}
			if len(key) < 20 {
				return fmt.Errorf("%w: minimum length 20 characters", ErrKeyTooShort)
			}
			return nil
		},
		TestEndpoint: "https://openrouter.ai/api/v1/models",
		TestHeaders: func(key string) map[string]string {
			return map[string]string{
				"Authorization": "Bearer " + key,
			}
		},
		SecretKeyName:    "openrouter_api_key",
		SupportsRotation: true,
		MinKeyLength:     20,
		MaxKeyLength:     256,
	}

	// Gemini
	m.providers[ProviderGemini] = ProviderKeyConfig{
		Name:        ProviderGemini,
		DisplayName: "Google Gemini",
		ValidateFormat: func(key string) error {
			// Gemini keys are typically 39 character alphanumeric strings
			if len(key) < 20 {
				return fmt.Errorf("%w: minimum length 20 characters", ErrKeyTooShort)
			}
			if len(key) > 256 {
				return fmt.Errorf("%w: maximum length 256 characters", ErrKeyTooLong)
			}
			return nil
		},
		TestEndpoint: "https://generativelanguage.googleapis.com/v1beta/models",
		TestHeaders: func(key string) map[string]string {
			return map[string]string{
				"x-goog-api-key": key,
			}
		},
		SecretKeyName:    "gemini_api_key",
		SupportsRotation: true,
		MinKeyLength:     20,
		MaxKeyLength:     256,
	}

	// Bedrock (uses AWS credentials, not a simple API key)
	m.providers[ProviderBedrock] = ProviderKeyConfig{
		Name:        ProviderBedrock,
		DisplayName: "AWS Bedrock",
		ValidateFormat: func(key string) error {
			// Bedrock uses AWS credentials which are validated differently
			// This is a placeholder for AWS credential validation
			if len(key) < 10 {
				return fmt.Errorf("%w: AWS credentials must be at least 10 characters", ErrKeyTooShort)
			}
			return nil
		},
		SecretKeyName:    "bedrock_credentials",
		SupportsRotation: false, // AWS uses IAM role-based rotation
		MinKeyLength:     10,
		MaxKeyLength:     512,
	}

	// Ollama (local, no API key required)
	m.providers[ProviderOllama] = ProviderKeyConfig{
		Name:        ProviderOllama,
		DisplayName: "Ollama",
		ValidateFormat: func(key string) error {
			// Ollama doesn't require an API key for local instances
			return nil
		},
		SecretKeyName:    "ollama_key",
		SupportsRotation: false,
		MinKeyLength:     0,
		MaxKeyLength:     0,
	}

	// LM Studio (local, no API key required)
	m.providers[ProviderLMStudio] = ProviderKeyConfig{
		Name:        ProviderLMStudio,
		DisplayName: "LM Studio",
		ValidateFormat: func(key string) error {
			// LM Studio doesn't require an API key for local instances
			// but can be configured with one for security
			return nil
		},
		SecretKeyName:    "lmstudio_key",
		SupportsRotation: false,
		MinKeyLength:     0,
		MaxKeyLength:     256,
	}
}

// RegisterProvider registers a custom provider configuration
func (m *APIKeyManager) RegisterProvider(config ProviderKeyConfig) error {
	if config.Name == "" {
		return errors.New("provider name is required")
	}
	if config.SecretKeyName == "" {
		return errors.New("secret key name is required")
	}
	m.providers[config.Name] = config
	return nil
}

// GetProvider returns the provider configuration for the given name
func (m *APIKeyManager) GetProvider(name string) (ProviderKeyConfig, bool) {
	config, ok := m.providers[name]
	return config, ok
}

// ListProviders returns a list of all registered provider names
func (m *APIKeyManager) ListProviders() []string {
	names := make([]string, 0, len(m.providers))
	for name := range m.providers {
		names = append(names, name)
	}
	return names
}

// ReadKeyInteractive reads an API key interactively with masking
func (m *APIKeyManager) ReadKeyInteractive(prompt string) (string, error) {
	fmt.Print(prompt)

	// Check if stdin is a terminal
	if term.IsTerminal(int(syscall.Stdin)) {
		// Use terminal for masked input
		keyBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return "", fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Println() // Add newline after masked input
		return strings.TrimSpace(string(keyBytes)), nil
	}

	// Fallback to regular input (e.g., when piped)
	reader := bufio.NewReader(os.Stdin)
	key, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}
	return strings.TrimSpace(key), nil
}

// ReadKeyWithFlag reads an API key from a flag value or interactively
func (m *APIKeyManager) ReadKeyWithFlag(flagValue string, interactivePrompt string) (string, error) {
	// If flag value is provided, use it
	if flagValue != "" {
		return flagValue, nil
	}

	// Otherwise, read interactively with masking
	return m.ReadKeyInteractive(interactivePrompt)
}

// ValidateKey validates an API key for a specific provider
func (m *APIKeyManager) ValidateKey(provider, key string) error {
	config, ok := m.providers[provider]
	if !ok {
		return fmt.Errorf("%w: %s", ErrProviderNotFound, provider)
	}

	// Check length constraints
	if config.MinKeyLength > 0 && len(key) < config.MinKeyLength {
		return fmt.Errorf("%w: minimum length is %d, got %d", ErrKeyTooShort, config.MinKeyLength, len(key))
	}
	if config.MaxKeyLength > 0 && len(key) > config.MaxKeyLength {
		return fmt.Errorf("%w: maximum length is %d, got %d", ErrKeyTooLong, config.MaxKeyLength, len(key))
	}

	// Run format validation
	if config.ValidateFormat != nil {
		if err := config.ValidateFormat(key); err != nil {
			return err
		}
	}

	return nil
}

// StoreKey stores an API key securely
func (m *APIKeyManager) StoreKey(provider, key string, metadata *KeyMetadata) error {
	if err := m.ValidateKey(provider, key); err != nil {
		return err
	}

	config, ok := m.providers[provider]
	if !ok {
		return fmt.Errorf("%w: %s", ErrProviderNotFound, provider)
	}

	// Store the key
	if err := m.secretsManager.Set(config.SecretKeyName, key); err != nil {
		return fmt.Errorf("failed to store API key: %w", err)
	}

	// Store metadata if provided
	if metadata != nil {
		metadata.Provider = provider
		metadata.UpdatedAt = time.Now()
		if metadata.CreatedAt.IsZero() {
			metadata.CreatedAt = time.Now()
		}
		metadata.IsActive = true
		metadata.KeyVersion++

		metadataKey := config.SecretKeyName + "_metadata"
		if err := m.secretsManager.Set(metadataKey, metadataToString(metadata)); err != nil {
			// Don't fail if metadata storage fails, just log
			fmt.Fprintf(os.Stderr, "Warning: failed to store key metadata: %v\n", err)
		}
	}

	return nil
}

// GetKey retrieves a stored API key
func (m *APIKeyManager) GetKey(provider string) (string, error) {
	config, ok := m.providers[provider]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrProviderNotFound, provider)
	}

	key, err := m.secretsManager.Get(config.SecretKeyName)
	if err != nil {
		if errors.Is(err, storage.ErrSecretNotFound) {
			return "", fmt.Errorf("%w: %s", ErrKeyNotFound, provider)
		}
		return "", fmt.Errorf("failed to retrieve API key: %w", err)
	}

	// Update last used time in metadata
	m.updateLastUsed(provider)

	return key, nil
}

// DeleteKey deletes a stored API key
func (m *APIKeyManager) DeleteKey(provider string) error {
	config, ok := m.providers[provider]
	if !ok {
		return fmt.Errorf("%w: %s", ErrProviderNotFound, provider)
	}

	// Delete the key
	if err := m.secretsManager.Delete(config.SecretKeyName); err != nil {
		if errors.Is(err, storage.ErrSecretNotFound) {
			return fmt.Errorf("%w: %s", ErrKeyNotFound, provider)
		}
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	// Delete metadata
	metadataKey := config.SecretKeyName + "_metadata"
	_ = m.secretsManager.Delete(metadataKey)

	return nil
}

// KeyExists checks if an API key exists for a provider
func (m *APIKeyManager) KeyExists(provider string) (bool, error) {
	config, ok := m.providers[provider]
	if !ok {
		return false, fmt.Errorf("%w: %s", ErrProviderNotFound, provider)
	}

	keys, err := m.secretsManager.List()
	if err != nil {
		return false, fmt.Errorf("failed to list secrets: %w", err)
	}

	for _, key := range keys {
		if key == config.SecretKeyName {
			return true, nil
		}
	}
	return false, nil
}

// TestKey tests an API key against the provider's API
func (m *APIKeyManager) TestKey(ctx context.Context, provider, key string) error {
	config, ok := m.providers[provider]
	if !ok {
		return fmt.Errorf("%w: %s", ErrProviderNotFound, provider)
	}

	// Skip test for local providers
	if provider == ProviderOllama || provider == ProviderLMStudio {
		return nil
	}

	if config.TestEndpoint == "" {
		return fmt.Errorf("%w: no test endpoint configured", ErrKeyTestFailed)
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, config.TestEndpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}

	// Add headers
	if config.TestHeaders != nil {
		for key, value := range config.TestHeaders(key) {
			req.Header.Set(key, value)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrKeyTestFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("%w: invalid API key (HTTP %d)", ErrKeyValidationFailed, resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: test failed with HTTP %d", ErrKeyTestFailed, resp.StatusCode)
	}

	return nil
}

// TestStoredKey tests the stored API key for a provider
func (m *APIKeyManager) TestStoredKey(ctx context.Context, provider string) error {
	key, err := m.GetKey(provider)
	if err != nil {
		return err
	}
	return m.TestKey(ctx, provider, key)
}

// RotateKey rotates an API key for a provider
func (m *APIKeyManager) RotateKey(provider, newKey string, gracePeriod time.Duration) (*KeyRotationInfo, error) {
	config, ok := m.providers[provider]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, provider)
	}

	if !config.SupportsRotation {
		return nil, fmt.Errorf("%w: %s", ErrRotationNotSupported, provider)
	}

	// Validate the new key
	if err := m.ValidateKey(provider, newKey); err != nil {
		return nil, err
	}

	// Get the old key
	oldKey, err := m.GetKey(provider)
	if err != nil && !errors.Is(err, ErrKeyNotFound) {
		return nil, err
	}

	// Store the new key
	if err := m.StoreKey(provider, newKey, nil); err != nil {
		return nil, fmt.Errorf("failed to store new key: %w", err)
	}

	// Create rotation info
	rotationInfo := &KeyRotationInfo{
		NewKey:    newKey,
		RotatedAt: time.Now(),
	}

	if oldKey != "" {
		rotationInfo.OldKey = oldKey
		if gracePeriod > 0 {
			rotationInfo.OldKeyExpiry = time.Now().Add(gracePeriod)
		}

		// Store rotation info for potential rollback
		rotationKey := config.SecretKeyName + "_rotation"
		if err := m.secretsManager.Set(rotationKey, rotationInfoToString(rotationInfo)); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to store rotation info: %v\n", err)
		}
	}

	return rotationInfo, nil
}

// GetKeyMetadata retrieves metadata for a stored key
func (m *APIKeyManager) GetKeyMetadata(provider string) (*KeyMetadata, error) {
	config, ok := m.providers[provider]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, provider)
	}

	metadataKey := config.SecretKeyName + "_metadata"
	metadataStr, err := m.secretsManager.Get(metadataKey)
	if err != nil {
		if errors.Is(err, storage.ErrSecretNotFound) {
			return nil, fmt.Errorf("%w: no metadata for %s", ErrKeyNotFound, provider)
		}
		return nil, fmt.Errorf("failed to retrieve metadata: %w", err)
	}

	return stringToMetadata(metadataStr)
}

// MaskKey returns a masked version of the API key for display
func (m *APIKeyManager) MaskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

// updateLastUsed updates the last used timestamp for a key
func (m *APIKeyManager) updateLastUsed(provider string) {
	config, ok := m.providers[provider]
	if !ok {
		return
	}

	metadataKey := config.SecretKeyName + "_metadata"
	metadataStr, err := m.secretsManager.Get(metadataKey)
	if err != nil {
		return
	}

	metadata, err := stringToMetadata(metadataStr)
	if err != nil {
		return
	}

	metadata.LastUsedAt = time.Now()
	_ = m.secretsManager.Set(metadataKey, metadataToString(metadata))
}

// Helper functions for metadata serialization

func metadataToString(m *KeyMetadata) string {
	// Simple serialization: provider|created_at|updated_at|last_used_at|description|is_active|version
	return fmt.Sprintf("%s|%s|%s|%s|%s|%v|%d",
		m.Provider,
		m.CreatedAt.Format(time.RFC3339),
		m.UpdatedAt.Format(time.RFC3339),
		m.LastUsedAt.Format(time.RFC3339),
		m.Description,
		m.IsActive,
		m.KeyVersion,
	)
}

func stringToMetadata(s string) (*KeyMetadata, error) {
	parts := strings.Split(s, "|")
	if len(parts) < 7 {
		return nil, errors.New("invalid metadata format")
	}

	createdAt, _ := time.Parse(time.RFC3339, parts[1])
	updatedAt, _ := time.Parse(time.RFC3339, parts[2])
	lastUsedAt, _ := time.Parse(time.RFC3339, parts[3])

	isActive := parts[5] == "true"
	var version int
	fmt.Sscanf(parts[6], "%d", &version)

	return &KeyMetadata{
		Provider:    parts[0],
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		LastUsedAt:  lastUsedAt,
		Description: parts[4],
		IsActive:    isActive,
		KeyVersion:  version,
	}, nil
}

func rotationInfoToString(r *KeyRotationInfo) string {
	// Simple serialization: old_key|new_key|rotated_at|old_key_expiry
	expiryStr := ""
	if !r.OldKeyExpiry.IsZero() {
		expiryStr = r.OldKeyExpiry.Format(time.RFC3339)
	}
	return fmt.Sprintf("%s|%s|%s|%s", r.OldKey, r.NewKey, r.RotatedAt.Format(time.RFC3339), expiryStr)
}

// KeyValidator provides standalone key validation functions
type KeyValidator struct{}

// NewKeyValidator creates a new key validator
func NewKeyValidator() *KeyValidator {
	return &KeyValidator{}
}

// ValidateAnthropicKey validates an Anthropic API key
func (v *KeyValidator) ValidateAnthropicKey(key string) error {
	if !strings.HasPrefix(key, "sk-ant-") {
		return fmt.Errorf("%w: must start with 'sk-ant-'", ErrInvalidKeyFormat)
	}
	if len(key) < 20 {
		return fmt.Errorf("%w: minimum 20 characters", ErrKeyTooShort)
	}
	// Check for valid characters (alphanumeric, hyphens, underscores)
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)
	if !validPattern.MatchString(key[7:]) { // Skip "sk-ant-" prefix
		return fmt.Errorf("%w: contains invalid characters", ErrInvalidKeyFormat)
	}
	return nil
}

// ValidateOpenAIKey validates an OpenAI API key
func (v *KeyValidator) ValidateOpenAIKey(key string) error {
	if !strings.HasPrefix(key, "sk-") {
		return fmt.Errorf("%w: must start with 'sk-'", ErrInvalidKeyFormat)
	}
	if len(key) < 20 {
		return fmt.Errorf("%w: minimum 20 characters", ErrKeyTooShort)
	}
	// OpenAI keys are base64-like strings
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)
	if !validPattern.MatchString(key[3:]) { // Skip "sk-" prefix
		return fmt.Errorf("%w: contains invalid characters", ErrInvalidKeyFormat)
	}
	return nil
}

// ValidateOpenRouterKey validates an OpenRouter API key
func (v *KeyValidator) ValidateOpenRouterKey(key string) error {
	if !strings.HasPrefix(key, "sk-") {
		return fmt.Errorf("%w: must start with 'sk-'", ErrInvalidKeyFormat)
	}
	if len(key) < 20 {
		return fmt.Errorf("%w: minimum 20 characters", ErrKeyTooShort)
	}
	return nil
}

// ValidateGeminiKey validates a Google Gemini API key
func (v *KeyValidator) ValidateGeminiKey(key string) error {
	// Gemini keys are typically alphanumeric
	if len(key) < 20 {
		return fmt.Errorf("%w: minimum 20 characters", ErrKeyTooShort)
	}
	if len(key) > 256 {
		return fmt.Errorf("%w: maximum 256 characters", ErrKeyTooLong)
	}
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)
	if !validPattern.MatchString(key) {
		return fmt.Errorf("%w: contains invalid characters", ErrInvalidKeyFormat)
	}
	return nil
}

// KeyInputReader handles reading API keys from various sources
type KeyInputReader struct {
	manager *APIKeyManager
}

// NewKeyInputReader creates a new key input reader
func NewKeyInputReader(manager *APIKeyManager) *KeyInputReader {
	return &KeyInputReader{manager: manager}
}

// ReadFromFlag reads a key from a command-line flag
func (r *KeyInputReader) ReadFromFlag(value string) (string, error) {
	if value == "" {
		return "", errors.New("flag value is empty")
	}
	return strings.TrimSpace(value), nil
}

// ReadFromEnvironment reads a key from an environment variable
func (r *KeyInputReader) ReadFromEnvironment(varName string) (string, error) {
	value := os.Getenv(varName)
	if value == "" {
		return "", fmt.Errorf("environment variable %s is not set", varName)
	}
	return strings.TrimSpace(value), nil
}

// ReadFromFile reads a key from a file
func (r *KeyInputReader) ReadFromFile(filepath string) (string, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

// ReadFromStdin reads a key from standard input
func (r *KeyInputReader) ReadFromStdin() (string, error) {
	return r.manager.ReadKeyInteractive("Enter API key: ")
}

// ReadFromSource reads a key from a specified source
type KeySourceType string

const (
	SourceFlag        KeySourceType = "flag"
	SourceEnvironment KeySourceType = "env"
	SourceFile        KeySourceType = "file"
	SourceStdin       KeySourceType = "stdin"
	SourceInteractive KeySourceType = "interactive"
)

// KeySource represents a key input source
type KeySource struct {
	Type       KeySourceType
	Value      string // For flag, env, file sources
	Prompt     string // For interactive source
	EnvVarName string // For env source
	FilePath   string // For file source
}

// Read reads a key from the configured source
func (r *KeyInputReader) Read(source KeySource) (string, error) {
	switch source.Type {
	case SourceFlag:
		return r.ReadFromFlag(source.Value)
	case SourceEnvironment:
		return r.ReadFromEnvironment(source.EnvVarName)
	case SourceFile:
		return r.ReadFromFile(source.FilePath)
	case SourceStdin, SourceInteractive:
		return r.ReadFromStdin()
	default:
		return "", fmt.Errorf("unknown key source type: %s", source.Type)
	}
}