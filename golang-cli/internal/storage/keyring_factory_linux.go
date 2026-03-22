//go:build linux

package storage

// getPlatformKeyring returns the appropriate keyring provider for Linux
func getPlatformKeyring() keyringProvider {
	return &linuxKeyring{}
}