// Package storage provides secure storage implementations for the Cline CLI.
// This file contains platform-agnostic keyring definitions.

package storage

// keyringProvider abstracts the OS-specific keyring implementations
type keyringProvider interface {
	Get(service, key string) (string, error)
	Set(service, key, value string) error
	Delete(service, key string) error
	List(service string) ([]string, error)
	IsAvailable() bool
}