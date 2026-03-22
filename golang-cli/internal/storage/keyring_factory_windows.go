//go:build windows

package storage

// getPlatformKeyring returns the appropriate keyring provider for Windows
func getPlatformKeyring() keyringProvider {
	return &windowsKeyring{}
}