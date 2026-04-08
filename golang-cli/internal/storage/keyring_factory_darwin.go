//go:build darwin && cgo

package storage

// getPlatformKeyring returns the appropriate keyring provider for macOS
func getPlatformKeyring() keyringProvider {
	return &macOSKeyring{}
}
