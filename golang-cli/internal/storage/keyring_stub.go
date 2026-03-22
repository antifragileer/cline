//go:build !darwin && !linux && !windows

package storage

// macOSKeyring stub for unsupported platforms
type macOSKeyring struct{ unsupportedKeyring }

// linuxKeyring stub for unsupported platforms
type linuxKeyring struct{ unsupportedKeyring }

// windowsKeyring stub for unsupported platforms
type windowsKeyring struct{ unsupportedKeyring }
