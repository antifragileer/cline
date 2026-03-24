//go:build darwin && !cgo

package storage

// getPlatformKeyring returns a stub keyring provider for macOS without CGO
func getPlatformKeyring() keyringProvider {
	return &unsupportedKeyring{}
}
