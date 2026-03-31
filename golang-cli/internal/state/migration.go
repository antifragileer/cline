// Package state provides VSCode-to-Go CLI state migration functionality.
// This enables seamless transition for users switching from the TypeScript CLI.
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// VSCodeState represents the VSCode extension state structure
type VSCodeState struct {
	Version int                    `json:"version"`
	Secrets map[string]string      `json:"secrets"`
	State   map[string]interface{} `json:"state"`
}

// MigrationConfig contains migration settings
type MigrationConfig struct {
	// VSCodeDataDir is the VSCode data directory
	VSCodeDataDir string
	// TargetDir is the target directory for migrated data
	TargetDir string
	// BackupDir is the directory for backups
	BackupDir string
	// DryRun if true, only simulates migration
	DryRun bool
}

// DefaultMigrationConfig returns a default migration configuration
func DefaultMigrationConfig() *MigrationConfig {
	homeDir, _ := os.UserHomeDir()

	return &MigrationConfig{
		VSCodeDataDir: filepath.Join(homeDir, ".vscode", "extensions", "cline.cline"),
		TargetDir:     filepath.Join(homeDir, ".cline"),
		BackupDir:     filepath.Join(homeDir, ".cline", "backups"),
		DryRun:        false,
	}
}

// Migrator handles state migration from VSCode to Go CLI
type Migrator struct {
	config *MigrationConfig
	writer *AtomicFileWriter
}

// NewMigrator creates a new state migrator
func NewMigrator(config *MigrationConfig) *Migrator {
	if config == nil {
		config = DefaultMigrationConfig()
	}

	return &Migrator{
		config: config,
		writer: NewAtomicFileWriter(),
	}
}

// Migrate performs the state migration
func (m *Migrator) Migrate() error {
	// Check if VSCode state exists
	vscodeStatePath := filepath.Join(m.config.VSCodeDataDir, "state.json")
	if _, err := os.Stat(vscodeStatePath); os.IsNotExist(err) {
		return fmt.Errorf("VSCode state not found at %s", vscodeStatePath)
	}

	// Read VSCode state
	vscodeState, err := m.readVSCodeState(vscodeStatePath)
	if err != nil {
		return fmt.Errorf("failed to read VSCode state: %w", err)
	}

	// Create backup
	if !m.config.DryRun {
		if err := m.createBackup(vscodeState); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	// Migrate global state
	if err := m.migrateGlobalState(vscodeState.State); err != nil {
		return fmt.Errorf("failed to migrate global state: %w", err)
	}

	// Migrate secrets
	if err := m.migrateSecrets(vscodeState.Secrets); err != nil {
		return fmt.Errorf("failed to migrate secrets: %w", err)
	}

	return nil
}

// readVSCodeState reads the VSCode state from file
func (m *Migrator) readVSCodeState(path string) (*VSCodeState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var state VSCodeState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// createBackup creates a backup of the current state
func (m *Migrator) createBackup(state *VSCodeState) error {
	if err := os.MkdirAll(m.config.BackupDir, 0755); err != nil {
		return err
	}

	timestamp := time.Now().Format("20060102-150405")
	backupPath := filepath.Join(m.config.BackupDir, fmt.Sprintf("vscode-backup-%s.json", timestamp))

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return m.writer.WriteFile(backupPath, data, 0644)
}

// migrateGlobalState migrates global state settings
func (m *Migrator) migrateGlobalState(state map[string]interface{}) error {
	if m.config.DryRun {
		fmt.Printf("[DRY RUN] Would migrate %d state keys\n", len(state))
		return nil
	}

	targetPath := filepath.Join(m.config.TargetDir, "data", "globalState.json")

	// Transform VSCode-specific keys to Go CLI format
	transformed := m.transformStateKeys(state)

	data, err := json.MarshalIndent(transformed, "", "  ")
	if err != nil {
		return err
	}

	return m.writer.WriteFile(targetPath, data, 0644)
}

// migrateSecrets migrates encrypted secrets
func (m *Migrator) migrateSecrets(secrets map[string]string) error {
	if m.config.DryRun {
		fmt.Printf("[DRY RUN] Would migrate %d secrets\n", len(secrets))
		return nil
	}

	targetPath := filepath.Join(m.config.TargetDir, "data", "secrets.json")

	// Transform VSCode secret keys to Go CLI format
	transformed := make(map[string]string)
	for key, value := range secrets {
		newKey := m.transformSecretKey(key)
		transformed[newKey] = value
	}

	data, err := json.MarshalIndent(transformed, "", "  ")
	if err != nil {
		return err
	}

	return m.writer.WriteFile(targetPath, data, 0600)
}

// transformStateKeys transforms VSCode state keys to Go CLI format
func (m *Migrator) transformStateKeys(state map[string]interface{}) map[string]interface{} {
	transformed := make(map[string]interface{})

	for key, value := range state {
		newKey := m.transformStateKey(key)
		transformed[newKey] = value
	}

	return transformed
}

// transformStateKey transforms a single state key
func (m *Migrator) transformStateKey(key string) string {
	// Remove VSCode-specific prefixes
	key = strings.TrimPrefix(key, "cline.")
	key = strings.TrimPrefix(key, "vscode-cline.")

	// Map known VSCode keys to Go CLI keys
	keyMap := map[string]string{
		"apiProvider":        "api_provider",
		"apiKey":             "api_key",
		"openRouterApiKey":   "openrouter_api_key",
		"awsAccessKey":       "aws_access_key",
		"awsSecretKey":       "aws_secret_key",
		"awsSessionToken":    "aws_session_token",
		"awsRegion":          "aws_region",
		"bedrockModel":       "bedrock_model",
		"bedrockUseCrossRegion": "bedrock_use_cross_region",
	}

	if newKey, exists := keyMap[key]; exists {
		return newKey
	}

	// Convert camelCase to snake_case
	return camelToSnake(key)
}

// transformSecretKey transforms a secret key
func (m *Migrator) transformSecretKey(key string) string {
	// Remove VSCode-specific prefixes
	key = strings.TrimPrefix(key, "cline.")
	key = strings.TrimPrefix(key, "secret.")

	// Map known secret keys
	secretMap := map[string]string{
		"apiKey":           "api_key",
		"openRouterApiKey": "openrouter_api_key",
		"awsAccessKey":     "aws_access_key",
		"awsSecretKey":     "aws_secret_key",
		"awsSessionToken":  "aws_session_token",
	}

	if newKey, exists := secretMap[key]; exists {
		return newKey
	}

	return camelToSnake(key)
}

// camelToSnake converts camelCase to snake_case
func camelToSnake(s string) string {
	var result strings.Builder

	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('_')
		}
		result.WriteRune(r)
	}

	return strings.ToLower(result.String())
}

// CheckMigrationNeeded checks if migration from VSCode is needed
func (m *Migrator) CheckMigrationNeeded() bool {
	// Check if VSCode state exists
	vscodeStatePath := filepath.Join(m.config.VSCodeDataDir, "state.json")
	if _, err := os.Stat(vscodeStatePath); os.IsNotExist(err) {
		return false
	}

	// Check if Go CLI state already exists
	goStatePath := filepath.Join(m.config.TargetDir, "data", "globalState.json")
	if _, err := os.Stat(goStatePath); err == nil {
		// Go CLI state exists, check if it's newer than VSCode state
		vscodeInfo, _ := os.Stat(vscodeStatePath)
		goInfo, _ := os.Stat(goStatePath)

		if goInfo.ModTime().After(vscodeInfo.ModTime()) {
			return false
		}
	}

	return true
}

// GetMigrationStatus returns detailed migration status
func (m *Migrator) GetMigrationStatus() (*MigrationStatus, error) {
	status := &MigrationStatus{}

	// Check VSCode state
	vscodeStatePath := filepath.Join(m.config.VSCodeDataDir, "state.json")
	if info, err := os.Stat(vscodeStatePath); err == nil {
		status.VSCodeStateExists = true
		status.VSCodeStateSize = info.Size()
		status.VSCodeStateModified = info.ModTime()
	}

	// Check Go CLI state
	goStatePath := filepath.Join(m.config.TargetDir, "data", "globalState.json")
	if info, err := os.Stat(goStatePath); err == nil {
		status.GoStateExists = true
		status.GoStateSize = info.Size()
		status.GoStateModified = info.ModTime()
	}

	// Check if migration is needed
	status.MigrationNeeded = m.CheckMigrationNeeded()

	return status, nil
}

// MigrationStatus contains migration status information
type MigrationStatus struct {
	VSCodeStateExists   bool
	VSCodeStateSize     int64
	VSCodeStateModified time.Time
	GoStateExists       bool
	GoStateSize         int64
	GoStateModified     time.Time
	MigrationNeeded     bool
}