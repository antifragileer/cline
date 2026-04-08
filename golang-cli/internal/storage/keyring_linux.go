//go:build linux

package storage

import (
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

// linuxKeyring implements the keyringProvider interface for Linux Secret Service
type linuxKeyring struct{}

// macOSKeyring stub for Linux
type macOSKeyring struct{ unsupportedKeyring }

// windowsKeyring stub for Linux
type windowsKeyring struct{ unsupportedKeyring }

// Get retrieves a secret from the Linux Secret Service
func (l *linuxKeyring) Get(service, key string) (string, error) {
	value, err := keyring.Get(service, sanitizeKey(key))
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", ErrSecretNotFound
		}
		return "", fmt.Errorf("secret service get failed: %w", err)
	}
	return value, nil
}

// Set stores a secret in the Linux Secret Service
func (l *linuxKeyring) Set(service, key, value string) error {
	if err := keyring.Set(service, sanitizeKey(key), value); err != nil {
		return fmt.Errorf("secret service set failed: %w", err)
	}
	return nil
}

// Delete removes a secret from the Linux Secret Service
func (l *linuxKeyring) Delete(service, key string) error {
	if err := keyring.Delete(service, sanitizeKey(key)); err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return ErrSecretNotFound
		}
		return fmt.Errorf("secret service delete failed: %w", err)
	}
	return nil
}

// List returns all secret keys from the Linux Secret Service
func (l *linuxKeyring) List(service string) ([]string, error) {
	// go-keyring doesn't support listing directly via the Secret Service API
	// We return an empty list as a limitation of the library
	// In production, this could be implemented using the Secret Service D-Bus API directly
	return []string{}, nil
}

// IsAvailable checks if the Linux Secret Service is accessible
func (l *linuxKeyring) IsAvailable() bool {
	// Try to get a non-existent secret to check if the service is available
	_, err := keyring.Get(serviceName, "__test_availability__")
	// If we get a "not found" error, the service is available
	// Other errors indicate the service is not available
	return err == nil || errors.Is(err, keyring.ErrNotFound)
}
