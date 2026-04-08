//go:build !darwin && !linux && !windows

package storage

// getPlatformKeyring returns a stub keyring provider for unsupported platforms
func getPlatformKeyring() keyringProvider {
	return &unsupportedKeyring{}
}
