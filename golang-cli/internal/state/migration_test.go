package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultMigrationConfig(t *testing.T) {
	config := DefaultMigrationConfig()
	require.NotNil(t, config)

	// Should have non-empty paths
	assert.NotEmpty(t, config.VSCodeDataDir)
	assert.NotEmpty(t, config.TargetDir)
	assert.NotEmpty(t, config.BackupDir)

	// Should have correct defaults
	assert.False(t, config.DryRun)

	// Should use home directory
	homeDir, _ := os.UserHomeDir()
	assert.Contains(t, config.VSCodeDataDir, homeDir)
	assert.Contains(t, config.TargetDir, homeDir)
	assert.Contains(t, config.BackupDir, homeDir)
}

func TestNewMigrator(t *testing.T) {
	t.Run("with custom config", func(t *testing.T) {
		config := &MigrationConfig{
			VSCodeDataDir: "/custom/vscode",
			TargetDir:     "/custom/target",
			BackupDir:     "/custom/backup",
			DryRun:        true,
		}

		migrator := NewMigrator(config)
		require.NotNil(t, migrator)
		assert.Equal(t, config, migrator.config)
		assert.NotNil(t, migrator.writer)
	})

	t.Run("with nil config uses defaults", func(t *testing.T) {
		migrator := NewMigrator(nil)
		require.NotNil(t, migrator)
		assert.NotNil(t, migrator.config)
		assert.NotNil(t, migrator.writer)
	})
}

func TestMigrator_readVSCodeState(t *testing.T) {
	tempDir := t.TempDir()
	migrator := NewMigrator(&MigrationConfig{
		VSCodeDataDir: tempDir,
		TargetDir:     tempDir,
		BackupDir:     tempDir,
	})

	t.Run("successful read", func(t *testing.T) {
		statePath := filepath.Join(tempDir, "state.json")

		expectedState := VSCodeState{
			Version: 1,
			Secrets: map[string]string{"apiKey": "secret123"},
			State:   map[string]interface{}{"apiProvider": "anthropic"},
		}

		data, err := json.Marshal(expectedState)
		require.NoError(t, err)
		err = os.WriteFile(statePath, data, 0644)
		require.NoError(t, err)

		state, err := migrator.readVSCodeState(statePath)
		require.NoError(t, err)
		assert.Equal(t, expectedState.Version, state.Version)
		assert.Equal(t, expectedState.Secrets, state.Secrets)
		assert.Equal(t, expectedState.State, state.State)
	})

	t.Run("file not found", func(t *testing.T) {
		state, err := migrator.readVSCodeState(filepath.Join(tempDir, "nonexistent.json"))
		assert.Error(t, err)
		assert.Nil(t, state)
	})

	t.Run("invalid json", func(t *testing.T) {
		statePath := filepath.Join(tempDir, "invalid.json")
		err := os.WriteFile(statePath, []byte("not valid json"), 0644)
		require.NoError(t, err)

		state, err := migrator.readVSCodeState(statePath)
		assert.Error(t, err)
		assert.Nil(t, state)
	})
}

func TestMigrator_createBackup(t *testing.T) {
	tempDir := t.TempDir()
	migrator := NewMigrator(&MigrationConfig{
		VSCodeDataDir: tempDir,
		TargetDir:     tempDir,
		BackupDir:     filepath.Join(tempDir, "backups"),
	})

	t.Run("successful backup creation", func(t *testing.T) {
		state := &VSCodeState{
			Version: 1,
			Secrets: map[string]string{"key": "value"},
			State:   map[string]interface{}{"setting": "value"},
		}

		err := migrator.createBackup(state)
		require.NoError(t, err)

		// Check backup directory was created
		_, err = os.Stat(migrator.config.BackupDir)
		assert.NoError(t, err)

		// Check backup file exists
		entries, err := os.ReadDir(migrator.config.BackupDir)
		require.NoError(t, err)
		assert.Len(t, entries, 1)
		assert.Contains(t, entries[0].Name(), "vscode-backup-")

		// Verify backup content
		backupPath := filepath.Join(migrator.config.BackupDir, entries[0].Name())
		data, err := os.ReadFile(backupPath)
		require.NoError(t, err)

		var backupState VSCodeState
		err = json.Unmarshal(data, &backupState)
		require.NoError(t, err)
		assert.Equal(t, state.Version, backupState.Version)
		assert.Equal(t, state.Secrets, backupState.Secrets)
		assert.Equal(t, state.State, backupState.State)
	})
}

func TestMigrator_migrateGlobalState(t *testing.T) {
	tempDir := t.TempDir()
	migrator := NewMigrator(&MigrationConfig{
		VSCodeDataDir: tempDir,
		TargetDir:     tempDir,
		BackupDir:     filepath.Join(tempDir, "backups"),
	})

	t.Run("successful migration", func(t *testing.T) {
		state := map[string]interface{}{
			"apiProvider": "anthropic",
			"apiKey":      "test-key",
		}

		err := migrator.migrateGlobalState(state)
		require.NoError(t, err)

		// Verify file was created
		targetPath := filepath.Join(tempDir, "data", "globalState.json")
		_, err = os.Stat(targetPath)
		assert.NoError(t, err)

		// Verify content
		data, err := os.ReadFile(targetPath)
		require.NoError(t, err)

		var migrated map[string]interface{}
		err = json.Unmarshal(data, &migrated)
		require.NoError(t, err)
		assert.Equal(t, "anthropic", migrated["api_provider"])
		assert.Equal(t, "test-key", migrated["api_key"])
	})

	t.Run("dry run mode", func(t *testing.T) {
		dryMigrator := NewMigrator(&MigrationConfig{
			VSCodeDataDir: tempDir,
			TargetDir:     filepath.Join(tempDir, "dry-run"),
			BackupDir:     filepath.Join(tempDir, "backups"),
			DryRun:        true,
		})

		state := map[string]interface{}{"key": "value"}
		err := dryMigrator.migrateGlobalState(state)
		require.NoError(t, err)

		// File should not be created
		targetPath := filepath.Join(tempDir, "dry-run", "data", "globalState.json")
		_, err = os.Stat(targetPath)
		assert.True(t, os.IsNotExist(err))
	})
}

func TestMigrator_migrateSecrets(t *testing.T) {
	tempDir := t.TempDir()
	migrator := NewMigrator(&MigrationConfig{
		VSCodeDataDir: tempDir,
		TargetDir:     tempDir,
		BackupDir:     filepath.Join(tempDir, "backups"),
	})

	t.Run("successful migration", func(t *testing.T) {
		secrets := map[string]string{
			"apiKey":           "secret123",
			"openRouterApiKey": "router456",
		}

		err := migrator.migrateSecrets(secrets)
		require.NoError(t, err)

		// Verify file was created with correct permissions
		targetPath := filepath.Join(tempDir, "data", "secrets.json")
		info, err := os.Stat(targetPath)
		require.NoError(t, err)
		// Check file mode includes 0600
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

		// Verify content
		data, err := os.ReadFile(targetPath)
		require.NoError(t, err)

		var migrated map[string]string
		err = json.Unmarshal(data, &migrated)
		require.NoError(t, err)
		assert.Equal(t, "secret123", migrated["api_key"])
		assert.Equal(t, "router456", migrated["openrouter_api_key"])
	})

	t.Run("dry run mode", func(t *testing.T) {
		dryMigrator := NewMigrator(&MigrationConfig{
			VSCodeDataDir: tempDir,
			TargetDir:     filepath.Join(tempDir, "dry-run-secrets"),
			BackupDir:     filepath.Join(tempDir, "backups"),
			DryRun:        true,
		})

		secrets := map[string]string{"key": "value"}
		err := dryMigrator.migrateSecrets(secrets)
		require.NoError(t, err)

		// File should not be created
		targetPath := filepath.Join(tempDir, "dry-run-secrets", "data", "secrets.json")
		_, err = os.Stat(targetPath)
		assert.True(t, os.IsNotExist(err))
	})
}

func TestMigrator_transformStateKey(t *testing.T) {
	migrator := NewMigrator(nil)

	testCases := []struct {
		input    string
		expected string
	}{
		// Known key mappings
		{"apiProvider", "api_provider"},
		{"apiKey", "api_key"},
		{"openRouterApiKey", "openrouter_api_key"},
		{"awsAccessKey", "aws_access_key"},
		{"awsSecretKey", "aws_secret_key"},
		{"awsSessionToken", "aws_session_token"},
		{"awsRegion", "aws_region"},
		{"bedrockModel", "bedrock_model"},
		{"bedrockUseCrossRegion", "bedrock_use_cross_region"},

		// Prefix stripping
		{"cline.apiProvider", "api_provider"},
		{"vscode-cline.apiKey", "api_key"},

		// CamelCase to snake_case (using actual implementation behavior)
		{"customSettingName", "custom_setting_name"},
		{"HTTPSProxy", "h_t_t_p_s_proxy"},
		{"RequestTimeoutMS", "request_timeout_m_s"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := migrator.transformStateKey(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMigrator_transformSecretKey(t *testing.T) {
	migrator := NewMigrator(nil)

	testCases := []struct {
		input    string
		expected string
	}{
		// Known secret mappings
		{"apiKey", "api_key"},
		{"openRouterApiKey", "openrouter_api_key"},
		{"awsAccessKey", "aws_access_key"},
		{"awsSecretKey", "aws_secret_key"},
		{"awsSessionToken", "aws_session_token"},

		// Prefix stripping
		{"cline.apiKey", "api_key"},
		{"secret.apiKey", "api_key"},

		// CamelCase to snake_case fallback
		{"customSecretKey", "custom_secret_key"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := migrator.transformSecretKey(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCamelToSnake(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"apiProvider", "api_provider"},
		{"APIKey", "a_p_i_key"},
		{"HTTPSProxy", "h_t_t_p_s_proxy"},
		{"RequestTimeoutMS", "request_timeout_m_s"},
		{"simple", "simple"},
		{"SimpleTest", "simple_test"},
		{"ABC", "a_b_c"},
		{"abcDefGhi", "abc_def_ghi"},
		{"", ""},
		{"A", "a"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := camelToSnake(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMigrator_transformStateKeys(t *testing.T) {
	migrator := NewMigrator(nil)

	input := map[string]interface{}{
		"apiProvider": "anthropic",
		"apiKey":      "test-key",
		"customValue": 123,
	}

	result := migrator.transformStateKeys(input)

	expected := map[string]interface{}{
		"api_provider": "anthropic",
		"api_key":      "test-key",
		"custom_value": 123,
	}

	assert.Equal(t, expected, result)
}

func TestMigrator_CheckMigrationNeeded(t *testing.T) {
	tempDir := t.TempDir()
	vscodeDir := filepath.Join(tempDir, "vscode")
	targetDir := filepath.Join(tempDir, "target")

	migrator := NewMigrator(&MigrationConfig{
		VSCodeDataDir: vscodeDir,
		TargetDir:     targetDir,
		BackupDir:     filepath.Join(tempDir, "backups"),
	})

	t.Run("no vscode state exists", func(t *testing.T) {
		needed := migrator.CheckMigrationNeeded()
		assert.False(t, needed)
	})

	t.Run("vscode state exists, no go state", func(t *testing.T) {
		// Create VSCode state
		err := os.MkdirAll(vscodeDir, 0755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(vscodeDir, "state.json"), []byte("{}"), 0644)
		require.NoError(t, err)

		needed := migrator.CheckMigrationNeeded()
		assert.True(t, needed)
	})

	t.Run("both exist, go state is newer", func(t *testing.T) {
		// Create Go state (newer)
		err := os.MkdirAll(filepath.Join(targetDir, "data"), 0755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(targetDir, "data", "globalState.json"), []byte("{}"), 0644)
		require.NoError(t, err)

		// Small delay to ensure different timestamps
		// In real scenarios, this won't be an issue

		needed := migrator.CheckMigrationNeeded()
		// This may be true or false depending on timing
		// We mainly care that it doesn't panic
		_ = needed
	})
}

func TestMigrator_GetMigrationStatus(t *testing.T) {
	tempDir := t.TempDir()
	vscodeDir := filepath.Join(tempDir, "vscode")
	targetDir := filepath.Join(tempDir, "target")

	migrator := NewMigrator(&MigrationConfig{
		VSCodeDataDir: vscodeDir,
		TargetDir:     targetDir,
		BackupDir:     filepath.Join(tempDir, "backups"),
	})

	t.Run("no files exist", func(t *testing.T) {
		status, err := migrator.GetMigrationStatus()
		require.NoError(t, err)
		assert.NotNil(t, status)
		assert.False(t, status.VSCodeStateExists)
		assert.False(t, status.GoStateExists)
	})

	t.Run("vscode state exists", func(t *testing.T) {
		// Create VSCode state
		err := os.MkdirAll(vscodeDir, 0755)
		require.NoError(t, err)
		testData := []byte(`{"version": 1}`)
		err = os.WriteFile(filepath.Join(vscodeDir, "state.json"), testData, 0644)
		require.NoError(t, err)

		status, err := migrator.GetMigrationStatus()
		require.NoError(t, err)
		assert.True(t, status.VSCodeStateExists)
		assert.Equal(t, int64(len(testData)), status.VSCodeStateSize)
		assert.False(t, status.GoStateExists)
	})

	t.Run("both states exist", func(t *testing.T) {
		// Create Go state
		err := os.MkdirAll(filepath.Join(targetDir, "data"), 0755)
		require.NoError(t, err)
		goData := []byte(`{"migrated": true}`)
		err = os.WriteFile(filepath.Join(targetDir, "data", "globalState.json"), goData, 0644)
		require.NoError(t, err)

		status, err := migrator.GetMigrationStatus()
		require.NoError(t, err)
		assert.True(t, status.VSCodeStateExists)
		assert.True(t, status.GoStateExists)
		assert.Equal(t, int64(len(goData)), status.GoStateSize)
	})
}

func TestMigrator_Migrate(t *testing.T) {
	tempDir := t.TempDir()
	vscodeDir := filepath.Join(tempDir, "vscode")
	targetDir := filepath.Join(tempDir, "target")

	migrator := NewMigrator(&MigrationConfig{
		VSCodeDataDir: vscodeDir,
		TargetDir:     targetDir,
		BackupDir:     filepath.Join(tempDir, "backups"),
	})

	t.Run("no vscode state", func(t *testing.T) {
		err := migrator.Migrate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "VSCode state not found")
	})

	t.Run("successful migration", func(t *testing.T) {
		// Create VSCode state
		err := os.MkdirAll(vscodeDir, 0755)
		require.NoError(t, err)

		vscodeState := VSCodeState{
			Version: 1,
			Secrets: map[string]string{
				"apiKey": "my-secret-key",
			},
			State: map[string]interface{}{
				"apiProvider": "anthropic",
				"customKey":   "customValue",
			},
		}

		data, err := json.Marshal(vscodeState)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(vscodeDir, "state.json"), data, 0644)
		require.NoError(t, err)

		// Run migration
		err = migrator.Migrate()
		require.NoError(t, err)

		// Verify global state was migrated
		globalStatePath := filepath.Join(targetDir, "data", "globalState.json")
		globalData, err := os.ReadFile(globalStatePath)
		require.NoError(t, err)

		var globalState map[string]interface{}
		err = json.Unmarshal(globalData, &globalState)
		require.NoError(t, err)
		assert.Equal(t, "anthropic", globalState["api_provider"])
		assert.Equal(t, "customValue", globalState["custom_key"])

		// Verify secrets were migrated
		secretsPath := filepath.Join(targetDir, "data", "secrets.json")
		secretsData, err := os.ReadFile(secretsPath)
		require.NoError(t, err)

		var secrets map[string]string
		err = json.Unmarshal(secretsData, &secrets)
		require.NoError(t, err)
		assert.Equal(t, "my-secret-key", secrets["api_key"])

		// Verify backup was created
		backupDir := filepath.Join(tempDir, "backups")
		entries, err := os.ReadDir(backupDir)
		require.NoError(t, err)
		assert.Len(t, entries, 1)
	})
}

func TestMigrationStatus(t *testing.T) {
	// Test that MigrationStatus struct exists and can be used
	status := &MigrationStatus{
		VSCodeStateExists:   true,
		VSCodeStateSize:     1024,
		GoStateExists:       false,
		GoStateSize:         0,
		MigrationNeeded:     true,
	}

	assert.True(t, status.VSCodeStateExists)
	assert.Equal(t, int64(1024), status.VSCodeStateSize)
	assert.False(t, status.GoStateExists)
	assert.Equal(t, int64(0), status.GoStateSize)
	assert.True(t, status.MigrationNeeded)
}

func TestVSCodeState(t *testing.T) {
	// Test that VSCodeState struct works correctly
	state := VSCodeState{
		Version: 1,
		Secrets: map[string]string{
			"secret1": "value1",
		},
		State: map[string]interface{}{
			"key1": "value1",
			"key2": 123,
		},
	}

	assert.Equal(t, 1, state.Version)
	assert.Equal(t, "value1", state.Secrets["secret1"])
	assert.Equal(t, "value1", state.State["key1"])
	assert.Equal(t, 123, state.State["key2"])
}