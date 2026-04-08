//go:build darwin && cgo

package storage

import (
	"errors"
	"fmt"

	"github.com/keybase/go-keychain"
)

// macOSKeyring implements the keyringProvider interface for macOS Keychain
type macOSKeyring struct{}

// linuxKeyring stub for macOS
type linuxKeyring struct{ unsupportedKeyring }

// windowsKeyring stub for macOS
type windowsKeyring struct{ unsupportedKeyring }

// Get retrieves a secret from the macOS Keychain
func (m *macOSKeyring) Get(service, key string) (string, error) {
	query := keychain.NewItem()
	query.SetSecClass(keychain.SecClassGenericPassword)
	query.SetService(service)
	query.SetAccount(sanitizeKey(key))
	query.SetMatchLimit(keychain.MatchLimitOne)
	query.SetReturnData(true)

	results, err := keychain.QueryItem(query)
	if err != nil {
		if errors.Is(err, keychain.ErrorItemNotFound) {
			return "", ErrSecretNotFound
		}
		return "", fmt.Errorf("keychain query failed: %w", err)
	}

	if len(results) == 0 {
		return "", ErrSecretNotFound
	}

	return string(results[0].Data), nil
}

// Set stores a secret in the macOS Keychain
func (m *macOSKeyring) Set(service, key, value string) error {
	// Use NewGenericPassword which creates an item with proper accessibility
	item := keychain.NewGenericPassword(service, sanitizeKey(key), "", []byte(value), "")

	// Try to add the item
	err := keychain.AddItem(item)
	if err == nil {
		return nil
	}

	// If item already exists, update it
	if errors.Is(err, keychain.ErrorDuplicateItem) {
		return m.updateItem(service, key, value)
	}

	return fmt.Errorf("failed to add item to keychain: %w", err)
}

func (m *macOSKeyring) updateItem(service, key, value string) error {
	query := keychain.NewItem()
	query.SetSecClass(keychain.SecClassGenericPassword)
	query.SetService(service)
	query.SetAccount(sanitizeKey(key))

	update := keychain.NewItem()
	update.SetData([]byte(value))

	return keychain.UpdateItem(query, update)
}

// Delete removes a secret from the macOS Keychain
func (m *macOSKeyring) Delete(service, key string) error {
	query := keychain.NewItem()
	query.SetSecClass(keychain.SecClassGenericPassword)
	query.SetService(service)
	query.SetAccount(sanitizeKey(key))

	err := keychain.DeleteItem(query)
	if err != nil {
		if errors.Is(err, keychain.ErrorItemNotFound) {
			return ErrSecretNotFound
		}
		return fmt.Errorf("failed to delete item from keychain: %w", err)
	}
	return nil
}

// List returns all secret keys from the macOS Keychain
func (m *macOSKeyring) List(service string) ([]string, error) {
	query := keychain.NewItem()
	query.SetSecClass(keychain.SecClassGenericPassword)
	query.SetService(service)
	query.SetMatchLimit(keychain.MatchLimitAll)
	query.SetReturnAttributes(true)

	results, err := keychain.QueryItem(query)
	if err != nil {
		return nil, fmt.Errorf("keychain query failed: %w", err)
	}

	keys := make([]string, 0, len(results))
	for _, result := range results {
		if result.Account != "" {
			keys = append(keys, result.Account)
		}
	}

	return keys, nil
}

// IsAvailable checks if the macOS Keychain is accessible
func (m *macOSKeyring) IsAvailable() bool {
	// Try a simple query to verify keychain access
	query := keychain.NewItem()
	query.SetSecClass(keychain.SecClassGenericPassword)
	query.SetMatchLimit(keychain.MatchLimitOne)
	query.SetReturnAttributes(true)

	_, err := keychain.QueryItem(query)
	// Even if no items are found, the query itself should succeed
	// if the keychain is accessible
	return err == nil || errors.Is(err, keychain.ErrorItemNotFound)
}
