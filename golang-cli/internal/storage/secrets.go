// Package storage provides secure storage implementations for the Cline CLI.
package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	// serviceName is the identifier used for keyring entries
	serviceName = "cline-cli"
	// fallbackFileName is the name of the fallback secrets file
	fallbackFileName = "secrets.enc"
	// keyLength is the AES-256 key length in bytes
	keyLength = 32
	// nonceLength is the GCM nonce length
	nonceLength = 12
)

// ErrKeyringUnavailable is returned when the OS keyring is not available
var ErrKeyringUnavailable = errors.New("OS keyring is unavailable")

// ErrSecretNotFound is returned when a secret is not found
var ErrSecretNotFound = errors.New("secret not found")

// SecretsStore defines the interface for secret storage operations
type SecretsStore interface {
	// Get retrieves a secret by key
	Get(key string) (string, error)
	// Set stores a secret with the given key
	Set(key string, value string) error
	// Delete removes a secret by key
	Delete(key string) error
	// List returns all secret keys
	List() ([]string, error)
	// Close cleans up any resources
	Close() error
}

// SecretsManager provides secure secret storage using OS keyring with
// AES-256-GCM encrypted file fallback
type SecretsManager struct {
	provider       keyringProvider
	useKeyring     bool
	fallbackPath   string
	warningHandler func(string)
}

// SecretsManagerOptions configures the SecretsManager
type SecretsManagerOptions struct {
	// ConfigDir is the directory for fallback file storage
	ConfigDir string
	// WarningHandler is called when falling back to file-based encryption
	WarningHandler func(message string)
}

// NewSecretsManager creates a new SecretsManager with OS keyring support
// and AES-256-GCM encrypted file fallback
func NewSecretsManager(opts SecretsManagerOptions) (*SecretsManager, error) {
	if opts.ConfigDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		opts.ConfigDir = filepath.Join(homeDir, ".cline")
	}

	// Ensure config directory exists
	if err := os.MkdirAll(opts.ConfigDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	sm := &SecretsManager{
		fallbackPath:   filepath.Join(opts.ConfigDir, fallbackFileName),
		warningHandler: opts.WarningHandler,
	}

	// Try to initialize the OS keyring provider
	sm.provider = getPlatformKeyring()
	sm.useKeyring = sm.provider != nil && sm.provider.IsAvailable()

	if !sm.useKeyring && sm.warningHandler != nil {
		sm.warningHandler(fmt.Sprintf(
			"Warning: OS keyring unavailable on %s. Falling back to file-based encryption. "+
				"Secrets will be stored in %s with AES-256-GCM encryption.",
			runtime.GOOS, sm.fallbackPath))
	}

	return sm, nil
}

// Get retrieves a secret by key
func (sm *SecretsManager) Get(key string) (string, error) {
	if key == "" {
		return "", errors.New("key cannot be empty")
	}

	if sm.useKeyring {
		value, err := sm.provider.Get(serviceName, key)
		if err != nil {
			if errors.Is(err, ErrSecretNotFound) {
				return "", ErrSecretNotFound
			}
			return "", fmt.Errorf("keyring get failed: %w", err)
		}
		return value, nil
	}

	// Fallback to file-based storage
	return sm.getFromFile(key)
}

// Set stores a secret with the given key
func (sm *SecretsManager) Set(key string, value string) error {
	if key == "" {
		return errors.New("key cannot be empty")
	}

	if sm.useKeyring {
		if err := sm.provider.Set(serviceName, key, value); err != nil {
			return fmt.Errorf("keyring set failed: %w", err)
		}
		return nil
	}

	// Fallback to file-based storage
	return sm.setInFile(key, value)
}

// Delete removes a secret by key
func (sm *SecretsManager) Delete(key string) error {
	if key == "" {
		return errors.New("key cannot be empty")
	}

	if sm.useKeyring {
		if err := sm.provider.Delete(serviceName, key); err != nil {
			if errors.Is(err, ErrSecretNotFound) {
				return ErrSecretNotFound
			}
			return fmt.Errorf("keyring delete failed: %w", err)
		}
		return nil
	}

	// Fallback to file-based storage
	return sm.deleteFromFile(key)
}

// List returns all secret keys
func (sm *SecretsManager) List() ([]string, error) {
	if sm.useKeyring {
		return sm.provider.List(serviceName)
	}

	// Fallback to file-based storage
	return sm.listFromFile()
}

// Close cleans up any resources
func (sm *SecretsManager) Close() error {
	// Nothing to clean up currently
	return nil
}

// IsUsingKeyring returns true if the OS keyring is being used
func (sm *SecretsManager) IsUsingKeyring() bool {
	return sm.useKeyring
}

// FallbackEncryptionKey derives an encryption key from machine-specific data
// This provides basic protection - not as secure as the OS keyring
func (sm *SecretsManager) FallbackEncryptionKey() ([]byte, error) {
	// Use a combination of machine-specific identifiers
	var seed string

	// Try to use hostname
	hostname, err := os.Hostname()
	if err == nil && hostname != "" {
		seed += hostname
	}

	// Add OS-specific identifiers
	switch runtime.GOOS {
	case "darwin":
		// On macOS, we could use hardware UUID, but for fallback
		// we use a combination of home directory and hostname
		home, _ := os.UserHomeDir()
		seed += home
	case "linux":
		// On Linux, use machine-id if available
		if data, err := os.ReadFile("/etc/machine-id"); err == nil {
			seed += string(data)
		} else if data, err := os.ReadFile("/var/lib/dbus/machine-id"); err == nil {
			seed += string(data)
		}
	case "windows":
		// On Windows, use computer name and user profile
		seed += os.Getenv("COMPUTERNAME")
		seed += os.Getenv("USERPROFILE")
	}

	// If we couldn't get any identifiers, use a default (less secure but functional)
	if seed == "" {
		seed = "cline-fallback-seed-v1"
	}

	// Derive a 32-byte key using SHA-256
	hash := sha256.Sum256([]byte(seed))
	return hash[:], nil
}

// encrypt encrypts plaintext using AES-256-GCM
func (sm *SecretsManager) encrypt(plaintext string) (string, error) {
	key, err := sm.FallbackEncryptionKey()
	if err != nil {
		return "", fmt.Errorf("failed to derive encryption key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, nonceLength)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts ciphertext using AES-256-GCM
func (sm *SecretsManager) decrypt(ciphertext string) (string, error) {
	key, err := sm.FallbackEncryptionKey()
	if err != nil {
		return "", fmt.Errorf("failed to derive encryption key: %w", err)
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	if len(data) < nonceLength {
		return "", errors.New("ciphertext too short")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce, ciphertextBytes := data[:nonceLength], data[nonceLength:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// fallbackData represents the structure of the fallback secrets file
type fallbackData struct {
	Secrets map[string]string `json:"secrets"`
	Version int               `json:"version"`
}

// getFromFile retrieves a secret from the encrypted fallback file
func (sm *SecretsManager) getFromFile(key string) (string, error) {
	data, err := sm.readFallbackFile()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", ErrSecretNotFound
		}
		return "", err
	}

	value, ok := data.Secrets[key]
	if !ok {
		return "", ErrSecretNotFound
	}

	decrypted, err := sm.decrypt(value)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt secret: %w", err)
	}

	return decrypted, nil
}

// setInFile stores a secret in the encrypted fallback file
func (sm *SecretsManager) setInFile(key string, value string) error {
	data, err := sm.readFallbackFile()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if data.Secrets == nil {
		data.Secrets = make(map[string]string)
		data.Version = 1
	}

	encrypted, err := sm.encrypt(value)
	if err != nil {
		return fmt.Errorf("failed to encrypt secret: %w", err)
	}

	data.Secrets[key] = encrypted
	return sm.writeFallbackFile(data)
}

// deleteFromFile removes a secret from the encrypted fallback file
func (sm *SecretsManager) deleteFromFile(key string) error {
	data, err := sm.readFallbackFile()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrSecretNotFound
		}
		return err
	}

	if _, ok := data.Secrets[key]; !ok {
		return ErrSecretNotFound
	}

	delete(data.Secrets, key)
	return sm.writeFallbackFile(data)
}

// listFromFile returns all secret keys from the fallback file
func (sm *SecretsManager) listFromFile() ([]string, error) {
	data, err := sm.readFallbackFile()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []string{}, nil
		}
		return nil, err
	}

	keys := make([]string, 0, len(data.Secrets))
	for k := range data.Secrets {
		keys = append(keys, k)
	}
	return keys, nil
}

// readFallbackFile reads and parses the fallback secrets file
func (sm *SecretsManager) readFallbackFile() (*fallbackData, error) {
	data := &fallbackData{
		Secrets: nil, // Will be initialized on first write
		Version: 0,
	}

	file, err := os.Open(sm.fallbackPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return data, nil
		}
		return nil, fmt.Errorf("failed to open secrets file: %w", err)
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(data); err != nil {
		return nil, fmt.Errorf("failed to decode secrets file: %w", err)
	}

	return data, nil
}

// writeFallbackFile writes the fallback secrets file
func (sm *SecretsManager) writeFallbackFile(data *fallbackData) error {
	// Create temp file in the same directory for atomic write
	dir := filepath.Dir(sm.fallbackPath)
	tempFile, err := os.CreateTemp(dir, ".secrets-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tempFile.Name()

	// Ensure cleanup on error
	cleanup := func() {
		tempFile.Close()
		os.Remove(tempPath)
	}

	// Set restrictive permissions (owner only)
	if err := tempFile.Chmod(0600); err != nil {
		cleanup()
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	// Write JSON data
	encoder := json.NewEncoder(tempFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		cleanup()
		return fmt.Errorf("failed to encode secrets: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		cleanup()
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, sm.fallbackPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to write secrets file: %w", err)
	}

	return nil
}

// sanitizeKey sanitizes a key for use with keyring services
// Some keyring implementations have restrictions on key characters
func sanitizeKey(key string) string {
	// Replace problematic characters with underscores
	sanitized := strings.ReplaceAll(key, "/", "_")
	sanitized = strings.ReplaceAll(sanitized, "\\", "_")
	sanitized = strings.ReplaceAll(sanitized, ":", "_")
	return sanitized
}

// RecoverFromFallback migrates secrets from file-based storage to keyring
// This can be called when keyring becomes available (e.g., after user unlocks keychain)
func (sm *SecretsManager) RecoverFromFallback() error {
	if sm.useKeyring {
		// Already using keyring, nothing to do
		return nil
	}

	// Check if fallback file exists
	if _, err := os.Stat(sm.fallbackPath); os.IsNotExist(err) {
		return nil // No fallback file, nothing to migrate
	}

	// Re-initialize with fresh keyring check
	sm.provider = getPlatformKeyring()
	if sm.provider == nil || !sm.provider.IsAvailable() {
		return ErrKeyringUnavailable
	}

	// Read fallback data
	data, err := sm.readFallbackFile()
	if err != nil {
		return fmt.Errorf("failed to read fallback file: %w", err)
	}

	// Migrate each secret
	migrated := 0
	for key, encryptedValue := range data.Secrets {
		decrypted, err := sm.decrypt(encryptedValue)
		if err != nil {
			continue // Skip unrecoverable secrets
		}

		if err := sm.provider.Set(serviceName, key, decrypted); err != nil {
			continue // Skip secrets that can't be migrated
		}
		migrated++
	}

	// Update state
	sm.useKeyring = true

	// Optionally remove fallback file after successful migration
	// Keep it for safety, but could be removed with user confirmation
	if migrated == len(data.Secrets) {
		// All secrets migrated successfully
		_ = os.Remove(sm.fallbackPath)
	}

	return nil
}