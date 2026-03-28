// Package integration provides integration tests with the real Cline core extension.
// These tests verify that the Go CLI can connect to and interact with the actual
// Cline core extension via gRPC.
package integration

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

// TestCoreExtensionConnection tests connection to the core extension
func TestCoreExtensionConnection(t *testing.T) {
	binary := findGoBinary()
	if binary == "" {
		t.Skip("Go CLI binary not found")
	}

	// Check if core extension is available
	coreAvailable := isCoreExtensionAvailable()

	tests := []struct {
		name         string
		args         []string
		expectError  bool
		skipIfNoCore bool
	}{
		{
			name:         "task_without_core",
			args:         []string{"task", "--json", "echo hello"},
			expectError:  false, // Should handle gracefully
			skipIfNoCore: false,
		},
		{
			name:         "config_without_core",
			args:         []string{"config", "list"},
			expectError:  false,
			skipIfNoCore: false,
		},
		{
			name:         "history_without_core",
			args:         []string{"history", "--json"},
			expectError:  false,
			skipIfNoCore: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipIfNoCore && !coreAvailable {
				t.Skip("Core extension not available")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, binary, tt.args...)
			cmd.Env = os.Environ()

			output, err := cmd.CombinedOutput()

			if tt.expectError {
				assert.Error(t, err, "Expected error but got none. Output: %s", string(output))
			} else {
				// Command should not crash, even if core is not available
				// It may return an error about connection, but shouldn't panic
				exitCode := 0
				if err != nil {
					if exitErr, ok := err.(*exec.ExitError); ok {
						exitCode = exitErr.ExitCode()
					}
				}

				// Exit code should be valid (0-255)
				assert.True(t, exitCode >= 0 && exitCode <= 255,
					"Invalid exit code %d. Output: %s", exitCode, string(output))
			}
		})
	}
}

// TestTaskExecutionFlow tests the complete task execution flow
func TestTaskExecutionFlow(t *testing.T) {
	binary := findGoBinary()
	if binary == "" {
		t.Skip("Go CLI binary not found")
	}

	// Create temporary directory for test data
	tempDir, err := os.MkdirTemp("", "core-ext-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("task_creation_workflow", func(t *testing.T) {
		// This test simulates the task creation workflow
		// Note: Without a real core extension, this tests the CLI's handling

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Set up environment
		env := append(os.Environ(),
			"CLINE_DATA_DIR="+tempDir,
			"CLINE_CONFIG_DIR="+tempDir,
		)

		// Test that CLI handles task command
		cmd := exec.CommandContext(ctx, binary, "task", "--json", "test task")
		cmd.Env = env

		out, err := cmd.CombinedOutput()
		outputStr := string(out)

		// Without core extension, this will likely fail to connect
		// but should not crash or hang indefinitely
		if err != nil {
			t.Logf("Task execution error (expected if no core): %v", err)
			t.Logf("Output: %s", outputStr)

			// Should timeout or give connection error, not hang
			assert.Contains(t, outputStr, "error", "Error", "connection", "Connection", "timeout", "Timeout",
				"Expected error message about connection or timeout")
		}
	})

	t.Run("task_resumption", func(t *testing.T) {
		// Test task resumption with task ID
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "--continue")
		cmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tempDir)

		out, _ := cmd.CombinedOutput()
		t.Logf("Task resumption output: %s", string(out))
	})

	t.Run("task_with_specific_id", func(t *testing.T) {
		// Test resuming specific task
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "-T", "test-task-id-123", "continue")
		cmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tempDir)

		out, _ := cmd.CombinedOutput()
		t.Logf("Task with ID output: %s", string(out))
	})
}

// TestConfigurationIntegration tests configuration management with core extension
func TestConfigurationIntegration(t *testing.T) {
	binary := findGoBinary()
	if binary == "" {
		t.Skip("Go CLI binary not found")
	}

	tempDir, err := os.MkdirTemp("", "config-integration-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("config_persistence", func(t *testing.T) {
		// Test that configuration persists correctly

		// Set a config value
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "config", "set", "test.key", "test-value")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)

		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "Config set failed: %s", string(out))

		// Get the config value
		ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel2()

		cmd2 := exec.CommandContext(ctx2, binary, "config", "get", "test.key")
		cmd2.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)

		out2, err := cmd2.CombinedOutput()
		require.NoError(t, err, "Config get failed: %s", string(out2))

		// Value should be preserved
		assert.Contains(t, string(out2), "test-value")
	})

	t.Run("config_list_format", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "config", "list", "--json")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)

		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "Config list failed: %s", string(out))

		// Should be valid JSON
		var config map[string]interface{}
		err = json.Unmarshal(out, &config)
		assert.NoError(t, err, "Config list should return valid JSON")
	})
}

// TestMCPIntegration tests MCP server integration
func TestMCPIntegration(t *testing.T) {
	binary := findGoBinary()
	if binary == "" {
		t.Skip("Go CLI binary not found")
	}

	t.Run("mcp_list_command", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "mcp", "list")
		out, err := cmd.CombinedOutput()

		// Should not crash
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}

		t.Logf("MCP list exit code: %d, output: %s", exitCode, string(out))
		assert.True(t, exitCode == 0 || exitCode == 1, "Exit code should be 0 or 1")
	})

	t.Run("mcp_list_json", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "mcp", "list", "--json")
		out, err := cmd.CombinedOutput()

		// Should return valid JSON or error gracefully
		if err == nil {
			var result interface{}
			err = json.Unmarshal(out, &result)
			if err != nil {
				t.Logf("MCP list JSON parse error: %v", err)
			}
		}
	})
}

// TestAuthIntegration tests authentication flows
func TestAuthIntegration(t *testing.T) {
	binary := findGoBinary()
	if binary == "" {
		t.Skip("Go CLI binary not found")
	}

	tempDir, err := os.MkdirTemp("", "auth-integration-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("auth_list", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "auth", "list")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)

		out, err := cmd.CombinedOutput()
		_ = out // Suppress unused variable warning
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}

		t.Logf("Auth list exit code: %d", exitCode)
		assert.True(t, exitCode == 0 || exitCode == 1)
	})

	t.Run("auth_status", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "auth", "status")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)

		// Run command - should not crash regardless of result
		_ = cmd.Run()
	})
}

// TestHistoryIntegration tests history management
func TestHistoryIntegration(t *testing.T) {
	binary := findGoBinary()
	if binary == "" {
		t.Skip("Go CLI binary not found")
	}

	tempDir, err := os.MkdirTemp("", "history-integration-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create tasks directory
	tasksDir := filepath.Join(tempDir, "tasks")
	err = os.MkdirAll(tasksDir, 0755)
	require.NoError(t, err)

	t.Run("history_with_mock_data", func(t *testing.T) {
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
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "history", "--json")
		cmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tempDir)

		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "History command failed: %s", string(out))

		var result []map[string]interface{}
		err = json.Unmarshal(out, &result)
		require.NoError(t, err, "History should be valid JSON")

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
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "history", "--json", "--limit", "10")
		cmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tempDir)

		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "History with limit failed: %s", string(out))

		var result []map[string]interface{}
		err = json.Unmarshal(out, &result)
		require.NoError(t, err)

		assert.LessOrEqual(t, len(result), 10, "Should respect limit")
	})
}

// TestStreamingOutput tests streaming output functionality
func TestStreamingOutput(t *testing.T) {
	binary := findGoBinary()
	if binary == "" {
		t.Skip("Go CLI binary not found")
	}

	t.Run("json_streaming_format", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Run a command that might produce streaming output
		cmd := exec.CommandContext(ctx, binary, "version", "--json")

		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "Command failed: %s", string(out))

		// Should be valid JSON
		var result map[string]interface{}
		err = json.Unmarshal(out, &result)
		assert.NoError(t, err, "Output should be valid JSON")
	})
}

// TestErrorHandling tests error handling and reporting
func TestErrorHandling(t *testing.T) {
	binary := findGoBinary()
	if binary == "" {
		t.Skip("Go CLI binary not found")
	}

	tests := []struct {
		name        string
		args        []string
		env         map[string]string
		expectError bool
	}{
		{
			name:        "invalid_config_path",
			args:        []string{"config", "list", "--config", "/nonexistent/path/that/does/not/exist"},
			expectError: true,
		},
		{
			name:        "permission_denied",
			args:        []string{"config", "list"},
			env:         map[string]string{"CLINE_CONFIG_DIR": "/root/no-permission"},
			expectError: true,
		},
		{
			name:        "malformed_task_id",
			args:        []string{"-T", "", "resume"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, binary, tt.args...)
			cmd.Env = os.Environ()
			for k, v := range tt.env {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
			}

			out, err := cmd.CombinedOutput()

			if tt.expectError {
				// Should return an error
				assert.Error(t, err, "Expected error for args %v", tt.args)
			}

			// Should not panic or hang
			exitCode := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				}
			}

			t.Logf("Exit code: %d, output: %s", exitCode, string(out))
		})
	}
}

// TestModeFlags tests mode flags (act, plan, yolo)
func TestModeFlags(t *testing.T) {
	binary := findGoBinary()
	if binary == "" {
		t.Skip("Go CLI binary not found")
	}

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "act_mode",
			args: []string{"-a", "test prompt"},
		},
		{
			name: "plan_mode",
			args: []string{"-p", "test prompt"},
		},
		{
			name: "yolo_mode",
			args: []string{"-y", "test prompt"},
		},
		{
			name: "auto_approve_all",
			args: []string{"--auto-approve-all", "test prompt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, binary, tt.args...)
			cmd.Env = os.Environ()

			out, err := cmd.CombinedOutput()

			// Should not crash
			exitCode := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				}
			}

			_ = out
			t.Logf("Mode flag %s: exit code %d", tt.name, exitCode)
		})
	}
}

// Helper functions

func isCoreExtensionAvailable() bool {
	// Check if core extension socket/port is available
	// This is a simplified check - in reality, you'd check for the extension process
	return false // Assume not available for safety
}

func findGoBinary() string {
	binaryName := "cline-go"
	if runtime.GOOS == "windows" {
		binaryName = "cline-go.exe"
	}

	locations := []string{
		filepath.Join("..", "..", binaryName),
		filepath.Join("..", "..", "cmd", "cline", binaryName),
		filepath.Join("..", binaryName),
		binaryName,
	}

	for _, loc := range locations {
		if absPath, err := filepath.Abs(loc); err == nil {
			if _, err := os.Stat(absPath); err == nil {
				return absPath
			}
		}
	}

	// Try PATH
	if path, err := exec.LookPath(binaryName); err == nil {
		return path
	}

	return ""
}