package storage

// unsupportedKeyring is a stub implementation for unavailable platforms
// This is defined in a common file so it can be embedded by platform stubs
type unsupportedKeyring struct{}

// Get always returns ErrKeyringUnavailable
func (u *unsupportedKeyring) Get(service, key string) (string, error) {
	return "", ErrKeyringUnavailable
}

// Set always returns ErrKeyringUnavailable
func (u *unsupportedKeyring) Set(service, key, value string) error {
	return ErrKeyringUnavailable
}

// Delete always returns ErrKeyringUnavailable
func (u *unsupportedKeyring) Delete(service, key string) error {
	return ErrKeyringUnavailable
}

// List always returns empty list
func (u *unsupportedKeyring) List(service string) ([]string, error) {
	return []string{}, nil
}

// IsAvailable always returns false
func (u *unsupportedKeyring) IsAvailable() bool {
	return false
}