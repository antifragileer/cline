// Package secrets provides encryption and decryption functionality for sensitive data
// in the Cline CLI. It uses AES-256-GCM for authenticated encryption.
package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"time"
)

const (
	// KeyLength is the AES-256 key length in bytes
	KeyLength = 32
	// NonceLength is the GCM nonce length in bytes
	NonceLength = 12
)

var (
	// ErrDecryptionFailed is returned when decryption fails
	ErrDecryptionFailed = errors.New("decryption failed")
	// ErrInvalidCiphertext is returned when ciphertext is invalid
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
	// ErrEncryptionFailed is returned when encryption fails
	ErrEncryptionFailed = errors.New("encryption failed")
)

// Encryptor provides encryption/decryption functionality
type Encryptor struct {
	key []byte
}

// NewEncryptor creates a new encryptor with the given key
func NewEncryptor(key []byte) (*Encryptor, error) {
	if len(key) != KeyLength {
		return nil, fmt.Errorf("key must be %d bytes, got %d", KeyLength, len(key))
	}

	// Create a copy to prevent external modification
	keyCopy := make([]byte, KeyLength)
	copy(keyCopy, key)

	return &Encryptor{
		key: keyCopy,
	}, nil
}

// NewEncryptorFromString creates a new encryptor from a base64-encoded key string
func NewEncryptorFromString(keyString string) (*Encryptor, error) {
	key, err := base64.StdEncoding.DecodeString(keyString)
	if err != nil {
		return nil, fmt.Errorf("failed to decode key: %w", err)
	}

	return NewEncryptor(key)
}

// Encrypt encrypts plaintext using AES-256-GCM
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("%w: failed to create cipher: %v", ErrEncryptionFailed, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("%w: failed to create GCM: %v", ErrEncryptionFailed, err)
	}

	nonce := make([]byte, NonceLength)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("%w: failed to generate nonce: %v", ErrEncryptionFailed, err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts ciphertext using AES-256-GCM
func (e *Encryptor) Decrypt(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("%w: failed to decode ciphertext: %v", ErrInvalidCiphertext, err)
	}

	if len(data) < NonceLength {
		return "", fmt.Errorf("%w: ciphertext too short", ErrInvalidCiphertext)
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("%w: failed to create cipher: %v", ErrDecryptionFailed, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("%w: failed to create GCM: %v", ErrDecryptionFailed, err)
	}

	nonce, ciphertextBytes := data[:NonceLength], data[NonceLength:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	return string(plaintext), nil
}

// EncryptBytes encrypts raw bytes using AES-256-GCM
func (e *Encryptor) EncryptBytes(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create cipher: %v", ErrEncryptionFailed, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create GCM: %v", ErrEncryptionFailed, err)
	}

	nonce := make([]byte, NonceLength)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("%w: failed to generate nonce: %v", ErrEncryptionFailed, err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptBytes decrypts raw bytes using AES-256-GCM
func (e *Encryptor) DecryptBytes(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < NonceLength {
		return nil, fmt.Errorf("%w: ciphertext too short", ErrInvalidCiphertext)
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create cipher: %v", ErrDecryptionFailed, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create GCM: %v", ErrDecryptionFailed, err)
	}

	nonce, ciphertextBytes := ciphertext[:NonceLength], ciphertext[NonceLength:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	return plaintext, nil
}

// GenerateKey generates a new random encryption key
func GenerateKey() ([]byte, error) {
	key := make([]byte, KeyLength)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	return key, nil
}

// GenerateKeyString generates a new random encryption key as a base64 string
func GenerateKeyString() (string, error) {
	key, err := GenerateKey()
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// DeriveKey derives a key from a password using PBKDF2
func DeriveKey(password string, salt []byte) []byte {
	// Simple derivation using SHA-256
	// In production, consider using golang.org/x/crypto/pbkdf2
	hash := sha256.Sum256(append([]byte(password), salt...))
	return hash[:]
}

// RotatingEncryptor manages key rotation for encryption
type RotatingEncryptor struct {
	current   *Encryptor
	previous  *Encryptor
	keyID     string
	rotationTime time.Time
}

// NewRotatingEncryptor creates a new rotating encryptor
func NewRotatingEncryptor(currentKey []byte, previousKey []byte, keyID string) (*RotatingEncryptor, error) {
	current, err := NewEncryptor(currentKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create current encryptor: %w", err)
	}

	re := &RotatingEncryptor{
		current:      current,
		keyID:        keyID,
		rotationTime: time.Now(),
	}

	if previousKey != nil {
		previous, err := NewEncryptor(previousKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create previous encryptor: %w", err)
		}
		re.previous = previous
	}

	return re, nil
}

// Encrypt encrypts with the current key
func (re *RotatingEncryptor) Encrypt(plaintext string) (string, error) {
	ciphertext, err := re.current.Encrypt(plaintext)
	if err != nil {
		return "", err
	}
	// Prepend key ID for later decryption
	return re.keyID + ":" + ciphertext, nil
}

// Decrypt decrypts, trying previous key if current fails
func (re *RotatingEncryptor) Decrypt(ciphertext string) (string, error) {
	// Strip key ID if present
	if idx := len(re.keyID) + 1; len(ciphertext) > idx && ciphertext[len(re.keyID)] == ':' {
		if ciphertext[:len(re.keyID)] == re.keyID {
			ciphertext = ciphertext[idx:]
		}
	}

	// Try current key first
	plaintext, err := re.current.Decrypt(ciphertext)
	if err == nil {
		return plaintext, nil
	}

	// Try previous key if available
	if re.previous != nil {
		return re.previous.Decrypt(ciphertext)
	}

	return "", err
}

// ShouldRotate returns true if key rotation is recommended
func (re *RotatingEncryptor) ShouldRotate(maxAge time.Duration) bool {
	return time.Since(re.rotationTime) > maxAge
}