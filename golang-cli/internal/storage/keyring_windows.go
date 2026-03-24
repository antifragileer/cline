//go:build windows

package storage

import (
	"errors"
	"fmt"

	"github.com/danieljoos/wincred"
)

// windowsKeyring implements the keyringProvider interface for Windows Credential Manager
type windowsKeyring struct{}

// macOSKeyring stub for Windows
type macOSKeyring struct{ unsupportedKeyring }

// linuxKeyring stub for Windows
type linuxKeyring struct{ unsupportedKeyring }

// makeTargetName creates a unique target name for Windows Credential Manager
func (w *windowsKeyring) makeTargetName(service, key string) string {
	return fmt.Sprintf("%s:%s", service, sanitizeKey(key))
}

// parseTargetName extracts the key from a Windows Credential Manager target name
func (w *windowsKeyring) parseTargetName(service, targetName string) (string, bool) {
	prefix := service + ":"
	if len(targetName) > len(prefix) && targetName[:len(prefix)] == prefix {
		return targetName[len(prefix):], true
	}
	return "", false
}

// Get retrieves a secret from the Windows Credential Manager
func (w *windowsKeyring) Get(service, key string) (string, error) {
	targetName := w.makeTargetName(service, key)
	cred, err := wincred.GetGenericCredential(targetName)
	if err != nil {
		if errors.Is(err, wincred.ErrElementNotFound) {
			return "", ErrSecretNotFound
		}
		return "", fmt.Errorf("credential manager get failed: %w", err)
	}

	if cred.CredentialBlob == nil {
		return "", ErrSecretNotFound
	}

	return string(cred.CredentialBlob), nil
}

// Set stores a secret in the Windows Credential Manager
func (w *windowsKeyring) Set(service, key, value string) error {
	targetName := w.makeTargetName(service, key)

	// Check if credential already exists
	existing, err := wincred.GetGenericCredential(targetName)
	if err == nil && existing != nil {
		// Update existing credential
		existing.CredentialBlob = []byte(value)
		if err := existing.Write(); err != nil {
			return fmt.Errorf("credential manager update failed: %w", err)
		}
		return nil
	}

	// Create new credential
	cred := wincred.NewGenericCredential(targetName)
	cred.CredentialBlob = []byte(value)

	if err := cred.Write(); err != nil {
		return fmt.Errorf("credential manager write failed: %w", err)
	}

	return nil
}

// Delete removes a secret from the Windows Credential Manager
func (w *windowsKeyring) Delete(service, key string) error {
	targetName := w.makeTargetName(service, key)
	cred, err := wincred.GetGenericCredential(targetName)
	if err != nil {
		if errors.Is(err, wincred.ErrElementNotFound) {
			return ErrSecretNotFound
		}
		return fmt.Errorf("credential manager get failed: %w", err)
	}

	if err := cred.Delete(); err != nil {
		return fmt.Errorf("credential manager delete failed: %w", err)
	}

	return nil
}

// List returns all secret keys from the Windows Credential Manager
func (w *windowsKeyring) List(service string) ([]string, error) {
	creds, err := wincred.List()
	if err != nil {
		return nil, fmt.Errorf("credential manager list failed: %w", err)
	}

	keys := make([]string, 0)
	for _, cred := range creds {
		// Filter by service prefix
		if key, ok := w.parseTargetName(service, cred.TargetName); ok {
			keys = append(keys, key)
		}
	}

	return keys, nil
}

// IsAvailable checks if the Windows Credential Manager is accessible
func (w *windowsKeyring) IsAvailable() bool {
	// Try to list credentials to verify access
	_, err := wincred.List()
	return err == nil
}