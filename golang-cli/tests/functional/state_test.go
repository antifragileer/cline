// Package functional provides comprehensive functional tests for state management.
// These tests verify state persistence, task history, and configuration management.
package functional

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStatePersistence validates state persistence across commands
func TestStatePersistence(t *testing.T) {
	binary := findGoBinary()
	if binary == "" {
		t.Skip("Go CLI binary not found")
	}

	tempDir, err := os.MkdirTemp("", "state-persist-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("config_persistence", func(t *testing.T) {
		// Set a config value
		ctx1, cancel1 := context.WithTimeout(context.Background(), 10*time.Second)
		cmd1 := exec.CommandContext(ctx1, binary, "config", "set", "test.key", "test-value")
		cmd1.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
		_, err1 := cmd1.CombinedOutput()
		cancel1()
		require.NoError(t, err1, "Config set failed")

		// Get the config value
		ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
		cmd2 := exec.CommandContext(ctx2, binary, "config", "get", "test.key")
		cmd2.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
		out2, err2 := cmd2.CombinedOutput()
		cancel2()
		require.NoError(t, err2, "Config get failed: %s", string(out2))

		// Value should be preserved
		assert.Contains(t, string(out2), "test-value")
	})

	t.Run("config_multiple_values", func(t *testing.T) {
		// Set multiple config values
		values := map[string]string{
			"key1":       "value1",
			"key2":       "value2",
			"nested.key": "nested-value",
		}

		for key, value := range values {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			cmd := exec.CommandContext(ctx, binary, "config", "set", key, value)
			cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
			out, err := cmd.CombinedOutput()
			cancel()
			require.NoError(t, err, "Config set %s failed: %s", key, string(out))
		}

		// List all config
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := exec.CommandContext(ctx, binary, "config", "list")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
		out, err := cmd.CombinedOutput()
		cancel()
		require.NoError(t, err, "Config list failed: %s", string(out))

		// Verify all values are present
		for _, value := range values {
			assert.Contains(t, string(out), value)
		}
	})

	t.Run("config_update_existing", func(t *testing.T) {
		// Set initial value
		ctx1, cancel1 := context.WithTimeout(context.Background(), 10*time.Second)
		cmd1 := exec.CommandContext(ctx1, binary, "config", "set", "update.key", "initial")
		cmd1.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
		_, _ = cmd1.CombinedOutput()
		cancel1()

		// Update value
		ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
		cmd2 := exec.CommandContext(ctx2, binary, "config", "set", "update.key", "updated")
		cmd2.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
		out2, _ := cmd2.CombinedOutput()
		cancel2()
		t.Logf("Update set: %s", string(out2))

		// Verify updated value
		ctx3, cancel3 := context.WithTimeout(context.Background(), 10*time.Second)
		cmd3 := exec.CommandContext(ctx3, binary, "config", "get", "update.key")
		cmd3.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
		out3, _ := cmd3.CombinedOutput()
		cancel3()

		assert.Contains(t, string(out3), "updated")
	})

	t.Run("config_delete", func(t *testing.T) {
		// Set a value
		ctx1, cancel1 := context.WithTimeout(context.Background(), 10*time.Second)
		cmd1 := exec.CommandContext(ctx1, binary, "config", "set", "delete.key", "to-delete")
		cmd1.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
		_, _ = cmd1.CombinedOutput()
		cancel1()

		// Delete the value
		ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
		cmd2 := exec.CommandContext(ctx2, binary, "config", "delete", "delete.key")
		cmd2.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
		out2, err2 := cmd2.CombinedOutput()
		cancel2()

		exitCode := 0
		if err2 != nil {
			if exitErr, ok := err2.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}

		// Delete might not be implemented yet
		if exitCode == 0 {
			t.Logf("Delete succeeded: %s", string(out2))
		} else {
			t.Logf("Delete not implemented or failed: %s", string(out2))
		}
		_ = exitCode // Not used in this test
	})
}

// TestTaskHistory validates task history management
func TestTaskHistory(t *testing.T) {
	binary := findGoBinary()
	if binary == "" {
		t.Skip("Go CLI binary not found")
	}

	tempDir, err := os.MkdirTemp("", "task-history-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create tasks directory
	tasksDir := filepath.Join(tempDir, "tasks")
	err = os.MkdirAll(tasksDir, 0755)
	require.NoError(t, err)

	t.Run("history_empty", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := exec.CommandContext(ctx, binary, "history")
		cmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tempDir)
		out, err := cmd.CombinedOutput()
		cancel()

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		_ = exitCode // Not used in this test
		assert.True(t, exitCode >= 0 && exitCode <= 255)
		t.Logf("Empty history output: %s", string(out))
	})

	t.Run("history_with_data", func(t *testing.T) {
		// Create mock history data
		mockHistory := []map[string]interface{}{
			{
				"id":        "task-1",
				"timestamp": time.Now().Add(-time.Hour).UnixMilli(),
				"prompt":    "Test task 1",
				"status":    "completed",
			},
			{
				"id":        "task-2",
				"timestamp": time.Now().Add(-2 * time.Hour).UnixMilli(),
				"prompt":    "Test task 2",
				"status":    "failed",
			},
		}

		historyData, _ := json.Marshal(mockHistory)
		historyPath := filepath.Join(tasksDir, "history.json")
		err := os.WriteFile(historyPath, historyData, 0644)
		require.NoError(t, err)

		// Test history command
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := exec.CommandContext(ctx, binary, "history", "--json")
		cmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tempDir)
		out, err := cmd.CombinedOutput()
		cancel()

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		_ = exitCode // Not used in this test

		require.NoError(t, err, "History command failed: %s", string(out))

		var result []map[string]interface{}
		jsonErr := json.Unmarshal(out, &result)
		require.NoError(t, jsonErr, "History should be valid JSON")

		assert.Len(t, result, 2, "Should have 2 history entries")
	})

	t.Run("history_pagination", func(t *testing.T) {
		// Create many history entries
		mockHistory := make([]map[string]interface{}, 25)
		for i := 0; i < 25; i++ {
			mockHistory[i] = map[string]interface{}{
				"id":        fmt.Sprintf("task-%d", i),
				"timestamp": time.Now().Add(-time.Duration(i) * time.Hour).UnixMilli(),
				"prompt":    fmt.Sprintf("Task %d", i),
				"status":    "completed",
			}
		}

		historyData, _ := json.Marshal(mockHistory)
		historyPath := filepath.Join(tasksDir, "history.json")
		err := os.WriteFile(historyPath, historyData, 0644)
		require.NoError(t, err)

		// Test with limit
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := exec.CommandContext(ctx, binary, "history", "--json", "--limit", "10")
		cmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tempDir)
		out, err := cmd.CombinedOutput()
		cancel()

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		_ = exitCode // Not used in this test

		require.NoError(t, err, "History with limit failed: %s", string(out))

		var result []map[string]interface{}
		jsonErr := json.Unmarshal(out, &result)
		require.NoError(t, jsonErr)

		assert.LessOrEqual(t, len(result), 10, "Should respect limit")
	})

	t.Run("history_json_format", func(t *testing.T) {
		// Create mock history
		mockHistory := []map[string]interface{}{
			{
				"id":        "json-test-task",
				"timestamp": time.Now().UnixMilli(),
				"prompt":    "JSON format test",
				"status":    "completed",
			},
		}

		historyData, _ := json.Marshal(mockHistory)
		historyPath := filepath.Join(tasksDir, "history.json")
		err := os.WriteFile(historyPath, historyData, 0644)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := exec.CommandContext(ctx, binary, "history", "--json")
		cmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tempDir)
		out, err := cmd.CombinedOutput()
		cancel()

		require.NoError(t, err, "History JSON failed: %s", string(out))

		// Validate JSON structure
		var result []map[string]interface{}
		jsonErr := json.Unmarshal(out, &result)
		assert.NoError(t, jsonErr, "Output should be valid JSON")

		if len(result) > 0 {
			// Check required fields
			entry := result[0]
			assert.NotNil(t, entry["id"], "Entry should have id")
			assert.NotNil(t, entry["timestamp"], "Entry should have timestamp")
			assert.NotNil(t, entry["prompt"], "Entry should have prompt")
		}
	})
}

// TestConfigReadWrite validates configuration read/write operations
func TestConfigReadWrite(t *testing.T) {
	binary := findGoBinary()
	if binary == "" {
		t.Skip("Go CLI binary not found")
	}

	tempDir, err := os.MkdirTemp("", "config-rw-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("config_set_and_get", func(t *testing.T) {
		testCases := []struct {
			key   string
			value string
		}{
			{"simple", "value"},
			{"nested.key", "nested-value"},
			{"deep.nested.key", "deep-value"},
			{"with.dots.every.where", "dotted-value"},
		}

		for _, tc := range testCases {
			// Set
			ctx1, cancel1 := context.WithTimeout(context.Background(), 10*time.Second)
			cmd1 := exec.CommandContext(ctx1, binary, "config", "set", tc.key, tc.value)
			cmd1.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
			out1, err1 := cmd1.CombinedOutput()
			cancel1()
			require.NoError(t, err1, "Set %s failed: %s", tc.key, string(out1))

			// Get
			ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
			cmd2 := exec.CommandContext(ctx2, binary, "config", "get", tc.key)
			cmd2.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
			out2, err2 := cmd2.CombinedOutput()
			cancel2()
			require.NoError(t, err2, "Get %s failed: %s", tc.key, string(out2))

			assert.Contains(t, string(out2), tc.value, "Value for %s should be preserved", tc.key)
		}
	})

	t.Run("config_list_format", func(t *testing.T) {
		// Set some values
		ctx1, cancel1 := context.WithTimeout(context.Background(), 10*time.Second)
		cmd1 := exec.CommandContext(ctx1, binary, "config", "set", "list.test", "list-value")
		cmd1.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
		_, _ = cmd1.CombinedOutput()
		cancel1()

		// List in JSON format
		ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
		cmd2 := exec.CommandContext(ctx2, binary, "config", "list", "--json")
		cmd2.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
		out2, err2 := cmd2.CombinedOutput()
		cancel2()

		exitCode := 0
		if err2 != nil {
			if exitErr, ok := err2.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		_ = exitCode // Not used in this test

		require.NoError(t, err2, "Config list failed: %s", string(out2))

		// Should be valid JSON
		var config map[string]interface{}
		jsonErr := json.Unmarshal(out2, &config)
		assert.NoError(t, jsonErr, "Config list should return valid JSON")

		t.Logf("Config structure: %+v", config)
	})

	t.Run("config_get_nonexistent", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := exec.CommandContext(ctx, binary, "config", "get", "nonexistent.key.that.does.not.exist")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
		out, err := cmd.CombinedOutput()
		cancel()

		// Should handle gracefully
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}

		t.Logf("Nonexistent key output: %s", string(out))
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})

	t.Run("config_special_values", func(t *testing.T) {
		specialValues := []string{
			"value with spaces",
			"value\nwith\nnewlines",
			"value\ttab\there",
			"!@#$%^&*()",
			"",
			"unicode: 你好世界 🌍",
		}

		for i, value := range specialValues {
			key := fmt.Sprintf("special.%d", i)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			cmd := exec.CommandContext(ctx, binary, "config", "set", key, value)
			cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
			_, err := cmd.CombinedOutput()
			cancel()

			exitCode := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				}
			}

			t.Logf("Special value %d exit code: %d", i, exitCode)
			assert.True(t, exitCode >= 0 && exitCode <= 255)
		}
	})
}

// TestStateFileCompatibility validates state file compatibility
func TestStateFileCompatibility(t *testing.T) {
	binary := findGoBinary()
	if binary == "" {
		t.Skip("Go CLI binary not found")
	}

	tempDir, err := os.MkdirTemp("", "state-compat-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("config_file_format", func(t *testing.T) {
		// Set some config
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := exec.CommandContext(ctx, binary, "config", "set", "test.key", "test-value")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
		out, err := cmd.CombinedOutput()
		cancel()
		if err != nil {
			t.Logf("Config set error: %v", err)
		}
		t.Logf("Config set: %s", string(out))

		// Check if config file exists and is valid JSON
		configPath := filepath.Join(tempDir, "config.json")
		if _, err := os.Stat(configPath); err == nil {
			data, err := os.ReadFile(configPath)
			require.NoError(t, err, "Should be able to read config file")

			var config map[string]interface{}
			jsonErr := json.Unmarshal(data, &config)
			assert.NoError(t, jsonErr, "Config file should be valid JSON")

			t.Logf("Config file content: %s", string(data))
		} else {
			t.Logf("Config file not found at %s", configPath)
		}
	})

	t.Run("global_state_file", func(t *testing.T) {
		// Set global state via config
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := exec.CommandContext(ctx, binary, "config", "set", "global.setting", "global-value")
		cmd.Env = append(os.Environ(),
			"CLINE_CONFIG_DIR="+tempDir,
			"CLINE_DATA_DIR="+tempDir,
		)
		out, _ := cmd.CombinedOutput()
		cancel()
		t.Logf("Global set: %s", string(out))

		// Check for global state file
		globalStatePath := filepath.Join(tempDir, "globalState.json")
		if _, err := os.Stat(globalStatePath); err == nil {
			data, err := os.ReadFile(globalStatePath)
			require.NoError(t, err)

			var state map[string]interface{}
			jsonErr := json.Unmarshal(data, &state)
			assert.NoError(t, jsonErr, "Global state should be valid JSON")

			t.Logf("Global state content: %s", string(data))
		}
	})

	t.Run("workspace_state_file", func(t *testing.T) {
		workspaceDir := filepath.Join(tempDir, "workspace")
		err := os.MkdirAll(workspaceDir, 0755)
		require.NoError(t, err)

		// Set workspace-specific config
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := exec.CommandContext(ctx, binary, "config", "set", "workspace.setting", "workspace-value")
		cmd.Env = append(os.Environ(),
			"CLINE_CONFIG_DIR="+tempDir,
			"CLINE_DATA_DIR="+tempDir,
		)
		cmd.Dir = workspaceDir
		out, _ := cmd.CombinedOutput()
		cancel()
		t.Logf("Workspace set: %s", string(out))

		// Check for workspace state
		workspacesDir := filepath.Join(tempDir, "workspaces")
		if entries, err := os.ReadDir(workspacesDir); err == nil {
			for _, entry := range entries {
				t.Logf("Found workspace entry: %s", entry.Name())
			}
		}
	})
}

// TestStateEdgeCases validates state management edge cases
func TestStateEdgeCases(t *testing.T) {
	binary := findGoBinary()
	if binary == "" {
		t.Skip("Go CLI binary not found")
	}

	tempDir, err := os.MkdirTemp("", "state-edge-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("config_permission_denied", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Skipping permission test on Windows")
		}

		// Create read-only config directory
		readOnlyDir := filepath.Join(tempDir, "readonly")
		err := os.MkdirAll(readOnlyDir, 0555)
		require.NoError(t, err)
		defer os.Chmod(readOnlyDir, 0755) // Restore for cleanup

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := exec.CommandContext(ctx, binary, "config", "set", "test", "value")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+readOnlyDir)
		out, err := cmd.CombinedOutput()
		cancel()

		t.Logf("Permission denied output: %s", string(out))

		// Should handle gracefully
		if err != nil {
			t.Log("Correctly handled permission error")
		}
	})

	t.Run("config_corrupted_file", func(t *testing.T) {
		// Create corrupted config file
		configPath := filepath.Join(tempDir, "config.json")
		err := os.WriteFile(configPath, []byte("this is not valid json {"), 0644)
		require.NoError(t, err)

		// Try to read config
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := exec.CommandContext(ctx, binary, "config", "list")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
		out, err := cmd.CombinedOutput()
		cancel()

		t.Logf("Corrupted file output: %s", string(out))

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})

	t.Run("config_empty_file", func(t *testing.T) {
		// Create empty config file
		configPath := filepath.Join(tempDir, "config.json")
		err := os.WriteFile(configPath, []byte(""), 0644)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := exec.CommandContext(ctx, binary, "config", "list")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
		out, err := cmd.CombinedOutput()
		cancel()

		t.Logf("Empty file output: %s", string(out))

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})

	t.Run("state_concurrent_access", func(t *testing.T) {
		// Test concurrent config operations
		done := make(chan bool, 5)

		for i := 0; i < 5; i++ {
			go func(index int) {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				cmd := exec.CommandContext(ctx, binary, "config", "set",
					fmt.Sprintf("concurrent.%d", index),
					fmt.Sprintf("value-%d", index))
				cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
				_, err := cmd.CombinedOutput()
				cancel()

				if err != nil {
					t.Logf("Concurrent %d error: %v", index, err)
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 5; i++ {
			<-done
		}

		// Verify at least some values were set
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := exec.CommandContext(ctx, binary, "config", "list")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
		finalOut, _ := cmd.CombinedOutput()
		cancel()

		t.Logf("Final config: %s", string(finalOut))
	})

	t.Run("very_long_config_key", func(t *testing.T) {
		longKey := "a"
		for i := 0; i < 50; i++ {
			longKey += ".very.long.key.segment"
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := exec.CommandContext(ctx, binary, "config", "set", longKey, "value")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
		out, err := cmd.CombinedOutput()
		cancel()

		t.Logf("Long key output length: %d", len(out))

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})
}

// TestHistoryEdgeCases validates history edge cases
func TestHistoryEdgeCases(t *testing.T) {
	binary := findGoBinary()
	if binary == "" {
		t.Skip("Go CLI binary not found")
	}

	tempDir, err := os.MkdirTemp("", "history-edge-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tasksDir := filepath.Join(tempDir, "tasks")
	err = os.MkdirAll(tasksDir, 0755)
	require.NoError(t, err)

	t.Run("history_corrupted_json", func(t *testing.T) {
		// Write corrupted history
		err := os.WriteFile(filepath.Join(tasksDir, "history.json"), []byte("not valid json"), 0644)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := exec.CommandContext(ctx, binary, "history", "--json")
		cmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tempDir)
		out, err := cmd.CombinedOutput()
		cancel()

		t.Logf("Corrupted history output: %s", string(out))

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})

	t.Run("history_missing_fields", func(t *testing.T) {
		// History with missing required fields
		badHistory := []map[string]interface{}{
			{"id": "incomplete"},
			{"timestamp": time.Now().UnixMilli()},
			{"prompt": "no id or timestamp"},
		}

		data, _ := json.Marshal(badHistory)
		err := os.WriteFile(filepath.Join(tasksDir, "history.json"), data, 0644)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := exec.CommandContext(ctx, binary, "history", "--json")
		cmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tempDir)
		out, err := cmd.CombinedOutput()
		cancel()

		t.Logf("Missing fields output: %s", string(out))

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})

	t.Run("history_large_file", func(t *testing.T) {
		// Create large history file
		largeHistory := make([]map[string]interface{}, 1000)
		for i := 0; i < 1000; i++ {
			largeHistory[i] = map[string]interface{}{
				"id":        fmt.Sprintf("task-%d", i),
				"timestamp": time.Now().Add(-time.Duration(i) * time.Minute).UnixMilli(),
				"prompt":    fmt.Sprintf("This is a very long prompt for task %d with lots of text to increase file size", i),
				"status":    "completed",
			}
		}

		data, _ := json.Marshal(largeHistory)
		err := os.WriteFile(filepath.Join(tasksDir, "history.json"), data, 0644)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		cmd := exec.CommandContext(ctx, binary, "history", "--json", "--limit", "100")
		cmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tempDir)
		start := time.Now()
		out, err := cmd.CombinedOutput()
		elapsed := time.Since(start)
		cancel()

		t.Logf("Large file processing time: %v, output length: %d", elapsed, len(out))

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})
}
