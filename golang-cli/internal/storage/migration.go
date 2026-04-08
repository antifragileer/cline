// Package storage provides file-backed storage with migration support.
package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"
)

const (
	// StorageVersionKey is the key used to track storage schema version.
	StorageVersionKey = "__storageVersion"

	// DefaultStorageVersion is the initial storage version for new stores.
	DefaultStorageVersion = 1

	// BackupFileSuffix is the suffix appended to backup files.
	BackupFileSuffix = ".backup"

	// TempFileSuffix is the suffix appended to temporary files during atomic writes.
	TempFileSuffix = ".tmp"
)

// MigrationFunc is a function that performs a single migration step.
// It receives the storage data and should modify it in place.
type MigrationFunc func(data map[string]any) error

// Migration represents a single migration from one version to the next.
type Migration struct {
	FromVersion int
	ToVersion   int
	Name        string
	Apply       MigrationFunc
}

// MigrationRegistry manages registered migrations and executes them.
type MigrationRegistry struct {
	migrations map[int]Migration
}

// NewMigrationRegistry creates a new empty migration registry.
func NewMigrationRegistry() *MigrationRegistry {
	return &MigrationRegistry{
		migrations: make(map[int]Migration),
	}
}

// Register adds a migration to the registry.
// The migration is keyed by its FromVersion.
func (r *MigrationRegistry) Register(m Migration) error {
	if m.FromVersion < 1 {
		return fmt.Errorf("migration from version must be >= 1, got %d", m.FromVersion)
	}
	if m.ToVersion <= m.FromVersion {
		return fmt.Errorf("migration to version (%d) must be greater than from version (%d)", m.ToVersion, m.FromVersion)
	}
	if m.ToVersion != m.FromVersion+1 {
		return fmt.Errorf("migrations must be sequential: from %d can only go to %d, got %d", m.FromVersion, m.FromVersion+1, m.ToVersion)
	}
	if _, exists := r.migrations[m.FromVersion]; exists {
		return fmt.Errorf("migration from version %d already registered", m.FromVersion)
	}
	r.migrations[m.FromVersion] = m
	return nil
}

// GetCurrentVersion returns the current version of the storage data.
// If no version is set, it returns 0 (indicating pre-versioned data).
func GetCurrentVersion(data map[string]any) int {
	versionRaw, ok := data[StorageVersionKey]
	if !ok {
		return 0
	}

	switch v := versionRaw.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
		return 0
	default:
		return 0
	}
}

// SetVersion sets the storage version in the data map.
func SetVersion(data map[string]any, version int) {
	data[StorageVersionKey] = version
}

// GetTargetVersion returns the highest version supported by the registry.
func (r *MigrationRegistry) GetTargetVersion() int {
	maxVersion := DefaultStorageVersion
	for fromVersion, migration := range r.migrations {
		if fromVersion > maxVersion {
			maxVersion = fromVersion
		}
		if migration.ToVersion > maxVersion {
			maxVersion = migration.ToVersion
		}
	}
	return maxVersion
}

// NeedsMigration returns true if the data needs to be migrated.
func (r *MigrationRegistry) NeedsMigration(data map[string]any) bool {
	current := GetCurrentVersion(data)
	target := r.GetTargetVersion()
	return current < target
}

// GetMigrationPath returns the ordered list of migrations needed to go from
// currentVersion to targetVersion.
func (r *MigrationRegistry) GetMigrationPath(currentVersion, targetVersion int) ([]Migration, error) {
	if currentVersion >= targetVersion {
		return nil, nil
	}

	var path []Migration
	for v := currentVersion; v < targetVersion; v++ {
		// If no migration exists for this version, check if it's the initial state
		if v == 0 {
			// Version 0 means unversioned data, apply version 1 initialization
			if targetVersion >= 1 {
				path = append(path, Migration{
					FromVersion: 0,
					ToVersion:   1,
					Name:        "init_storage_version",
					Apply: func(data map[string]any) error {
						SetVersion(data, 1)
						return nil
					},
				})
			}
			continue
		}

		migration, ok := r.migrations[v]
		if !ok {
			return nil, fmt.Errorf("no migration found from version %d to %d", v, v+1)
		}
		path = append(path, migration)
	}

	return path, nil
}

// MigratableStorage handles file-backed JSON storage with migration support.
type MigratableStorage struct {
	filePath   string
	backupDir  string
	registry   *MigrationRegistry
	data       map[string]any
	dataMu     sync.RWMutex // Protects data map for concurrent access
	fileMode   os.FileMode
	backupMode os.FileMode
}

// MigratableStorageOption configures MigratableStorage options.
type MigratableStorageOption func(*MigratableStorage)

// WithFileMode sets the file permission mode for the storage file.
func WithFileMode(mode os.FileMode) MigratableStorageOption {
	return func(s *MigratableStorage) {
		s.fileMode = mode
	}
}

// WithBackupMode sets the file permission mode for backup files.
func WithBackupMode(mode os.FileMode) MigratableStorageOption {
	return func(s *MigratableStorage) {
		s.backupMode = mode
	}
}

// WithBackupDir sets the directory for backup files.
// If not set, backups are stored in the same directory as the storage file.
func WithBackupDir(dir string) MigratableStorageOption {
	return func(s *MigratableStorage) {
		s.backupDir = dir
	}
}

// NewMigratableStorage creates a new MigratableStorage instance.
func NewMigratableStorage(filePath string, registry *MigrationRegistry, opts ...MigratableStorageOption) (*MigratableStorage, error) {
	s := &MigratableStorage{
		filePath:   filePath,
		registry:   registry,
		data:       make(map[string]any),
		fileMode:   0644,
		backupMode: 0600,
	}

	for _, opt := range opts {
		opt(s)
	}

	// Set default backup dir to same directory as storage file
	if s.backupDir == "" {
		s.backupDir = filepath.Dir(filePath)
	}

	return s, nil
}

// Load reads the storage file and performs any necessary migrations.
func (s *MigratableStorage) Load() error {
	// Read existing data if file exists
	if _, err := os.Stat(s.filePath); err == nil {
		data, err := s.readFile(s.filePath)
		if err != nil {
			return fmt.Errorf("failed to read storage file: %w", err)
		}
		s.data = data
	}

	// Perform migrations if needed
	if s.registry != nil && s.registry.NeedsMigration(s.data) {
		if err := s.migrate(); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	} else if GetCurrentVersion(s.data) == 0 {
		// No registry but no version set - initialize to default
		SetVersion(s.data, DefaultStorageVersion)
		if err := s.Save(); err != nil {
			return fmt.Errorf("failed to initialize storage version: %w", err)
		}
	}

	return nil
}

// readFile reads and parses a JSON file.
func (s *MigratableStorage) readFile(path string) (map[string]any, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Save writes the current data to disk atomically.
func (s *MigratableStorage) Save() error {
	return s.atomicWrite(s.filePath, s.data, s.fileMode)
}

// atomicWrite writes data to a file atomically using temp file + rename.
func (s *MigratableStorage) atomicWrite(path string, data map[string]any, mode os.FileMode) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create temp file in same directory for atomic rename
	tmpFile, err := os.CreateTemp(dir, fmt.Sprintf(".%s%s.", filepath.Base(path), TempFileSuffix))
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Clean up temp file on error
	defer func() {
		if err != nil {
			os.Remove(tmpPath)
		}
	}()

	// Write data
	encoder := json.NewEncoder(tmpFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to encode data: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Set permissions before rename
	if err := os.Chmod(tmpPath, mode); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// migrate performs all pending migrations atomically with backup/restore.
func (s *MigratableStorage) migrate() error {
	currentVersion := GetCurrentVersion(s.data)
	targetVersion := s.registry.GetTargetVersion()

	if currentVersion >= targetVersion {
		return nil
	}

	// Get migration path
	path, err := s.registry.GetMigrationPath(currentVersion, targetVersion)
	if err != nil {
		return err
	}

	if len(path) == 0 {
		return nil
	}

	// Create backup before migration
	backupPath, err := s.createBackup()
	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Apply migrations
	for _, migration := range path {
		if err := migration.Apply(s.data); err != nil {
			// Restore from backup on error
			if restoreErr := s.restoreFromBackup(backupPath); restoreErr != nil {
				return fmt.Errorf("migration %s failed: %v; restore also failed: %v", migration.Name, err, restoreErr)
			}
			return fmt.Errorf("migration %s failed, restored from backup: %w", migration.Name, err)
		}
		// Update version after successful migration
		SetVersion(s.data, migration.ToVersion)
	}

	// Save migrated data
	if err := s.Save(); err != nil {
		// Restore from backup on save error
		if restoreErr := s.restoreFromBackup(backupPath); restoreErr != nil {
			return fmt.Errorf("failed to save migrated data: %v; restore also failed: %v", err, restoreErr)
		}
		return fmt.Errorf("failed to save migrated data, restored from backup: %w", err)
	}

	return nil
}

// createBackup creates a backup of the current storage file.
func (s *MigratableStorage) createBackup() (string, error) {
	// Generate backup filename with timestamp including milliseconds
	timestamp := time.Now().UTC().Format("20060102_150405.000")
	backupName := fmt.Sprintf("%s.%s%s", filepath.Base(s.filePath), timestamp, BackupFileSuffix)
	backupPath := filepath.Join(s.backupDir, backupName)

	// Ensure backup directory exists
	if err := os.MkdirAll(s.backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// If storage file doesn't exist yet, create a backup of current data
	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		if err := s.atomicWrite(backupPath, s.data, s.backupMode); err != nil {
			return "", err
		}
		return backupPath, nil
	}

	// Copy existing file to backup
	source, err := os.Open(s.filePath)
	if err != nil {
		return "", err
	}
	defer source.Close()

	// Create backup file with restricted permissions
	dest, err := os.OpenFile(backupPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, s.backupMode)
	if err != nil {
		return "", err
	}
	defer dest.Close()

	if _, err := io.Copy(dest, source); err != nil {
		return "", err
	}

	return backupPath, nil
}

// restoreFromBackup restores storage from a backup file.
func (s *MigratableStorage) restoreFromBackup(backupPath string) error {
	// Read backup data
	data, err := s.readFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}

	// Restore data in memory
	s.data = data

	// Restore file on disk
	if err := s.atomicWrite(s.filePath, data, s.fileMode); err != nil {
		return fmt.Errorf("failed to restore file: %w", err)
	}

	return nil
}

// Get retrieves a value from storage.
func (s *MigratableStorage) Get(key string) (any, bool) {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()
	val, ok := s.data[key]
	return val, ok
}

// GetString retrieves a string value from storage.
func (s *MigratableStorage) GetString(key string) (string, bool) {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()
	val, ok := s.data[key]
	if !ok {
		return "", false
	}

	switch v := val.(type) {
	case string:
		return v, true
	case fmt.Stringer:
		return v.String(), true
	default:
		return fmt.Sprintf("%v", v), true
	}
}

// GetInt retrieves an int value from storage.
func (s *MigratableStorage) GetInt(key string) (int, bool) {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()
	val, ok := s.data[key]
	if !ok {
		return 0, false
	}

	switch v := val.(type) {
	case int:
		return v, true
	case float64:
		return int(v), true
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i, true
		}
		return 0, false
	default:
		return 0, false
	}
}

// Set stores a value in storage.
func (s *MigratableStorage) Set(key string, value any) {
	s.dataMu.Lock()
	defer s.dataMu.Unlock()
	if value == nil {
		delete(s.data, key)
	} else {
		s.data[key] = value
	}
}

// Delete removes a key from storage.
func (s *MigratableStorage) Delete(key string) {
	s.dataMu.Lock()
	defer s.dataMu.Unlock()
	delete(s.data, key)
}

// Keys returns all keys in storage.
func (s *MigratableStorage) Keys() []string {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()
	keys := make([]string, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// GetVersion returns the current storage version.
func (s *MigratableStorage) GetVersion() int {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()
	return GetCurrentVersion(s.data)
}

// GetData returns a copy of the underlying data map.
func (s *MigratableStorage) GetData() map[string]any {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()
	result := make(map[string]any, len(s.data))
	for k, v := range s.data {
		result[k] = v
	}
	return result
}

// SetData replaces the entire data map.
func (s *MigratableStorage) SetData(data map[string]any) {
	s.dataMu.Lock()
	defer s.dataMu.Unlock()
	s.data = make(map[string]any, len(data))
	for k, v := range data {
		s.data[k] = v
	}
}

// ListBackups returns a list of backup files for this storage.
func (s *MigratableStorage) ListBackups() ([]string, error) {
	pattern := filepath.Join(s.backupDir, fmt.Sprintf("%s.*%s", filepath.Base(s.filePath), BackupFileSuffix))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	return matches, nil
}

// CleanupOldBackups removes backup files, keeping only the most recent ones.
func (s *MigratableStorage) CleanupOldBackups(keep int) error {
	backups, err := s.ListBackups()
	if err != nil {
		return err
	}

	if len(backups) <= keep {
		return nil
	}

	// Remove oldest backups
	for _, backup := range backups[:len(backups)-keep] {
		if err := os.Remove(backup); err != nil {
			return fmt.Errorf("failed to remove backup %s: %w", backup, err)
		}
	}

	return nil
}
