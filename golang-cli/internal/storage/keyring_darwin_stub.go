//go:build darwin && !cgo

package storage

// macOSKeyring stub for macOS without CGO
type macOSKeyring struct{ unsupportedKeyring }

// linuxKeyring stub for macOS without CGO
type linuxKeyring struct{ unsupportedKeyring }

// windowsKeyring stub for macOS without CGO
type windowsKeyring struct{ unsupportedKeyring }
