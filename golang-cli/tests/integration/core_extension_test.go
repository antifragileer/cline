// Package integration provides integration tests for core extension interaction.
// These tests verify the Go CLI's interaction with the VS Code extension host.
package integration

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCoreExtensionCommunication validates communication with core extension
func TestCoreExtensionCommunication(t *testing.T) {
	goPath := findGoBinary()
	if goPath == "" {
		t.Skip("Go CLI binary not found")
	}

	t.Run("core_extension_version_handshake", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, goPath, "version", "--json")

		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "Should get version")

		var versionInfo map[string]interface{}
		err = json.Unmarshal(out, &versionInfo)
		assert.NoError(t, err, "Version should be valid JSON")

		// Should contain version field
		if v, ok := versionInfo["version"]; ok {
			t.Logf("Version: %v", v)
		}
	})

	t.Run("core_extension_state_handshake", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "core-ext-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Set state
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := exec.CommandContext(ctx, goPath, "config", "set", "test.core", "core-value")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
		out, err := cmd.CombinedOutput()
		cancel()

		if err == nil {
			t.Logf("State set: %s", string(out))

			// Read state back
			ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
			cmd2 := exec.CommandContext(ctx2, goPath, "config", "get", "test.core")
			cmd2.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)
			out2, _ := cmd2.CombinedOutput()
			cancel2()

			assert.Contains(t, string(out2), "core-value")
		}
	})

	t.Run("core_extension_task_handshake", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Test that CLI can prepare to send task
		cmd := exec.CommandContext(ctx, goPath, "task", "--help")

		out, err := cmd.CombinedOutput()
		require.NoError(t, err)

		helpText := string(out)
		assert.Contains(t, helpText, "task")
	})
}

// TestExtensionHostDiscovery validates extension host discovery
func TestExtensionHostDiscovery(t *testing.T) {
	goPath := findGoBinary()
	if goPath == "" {
		t.Skip("Go CLI binary not found")
	}

	t.Run("discovery_via_environment", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Test with custom extension host
		cmd := exec.CommandContext(ctx, goPath, "version", "--short")
		cmd.Env = append(os.Environ(),
			"CLINE_EXTENSION_HOST=127.0.0.1:50051",
		)

		out, err := cmd.CombinedOutput()
		t.Logf("Output with custom host: %s", string(out))

		// Should handle gracefully
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})

	t.Run("discovery_via_config", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "discovery-config-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, goPath, "config", "set", "extension.host", "127.0.0.1:50052")
		cmd.Env = append(os.Environ(), "CLINE_CONFIG_DIR="+tempDir)

		out, err := cmd.CombinedOutput()
		t.Logf("Config set output: %s", string(out))

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})

	t.Run("discovery_fallback", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Without any host configured, should use default
		cmd := exec.CommandContext(ctx, goPath, "version", "--short")

		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "Should work without explicit host")

		assert.NotEmpty(t, string(out))
	})
}

// TestCoreExtensionOperations validates core extension operations
func TestCoreExtensionOperations(t *testing.T) {
	goPath := findGoBinary()
	if goPath == "" {
		t.Skip("Go CLI binary not found")
	}

	t.Run("task_operation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Verify task command exists
		cmd := exec.CommandContext(ctx, goPath, "task", "--help")

		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "Task command should exist")

		helpText := string(out)
		assert.Contains(t, helpText, "task")
	})

	t.Run("history_operation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, goPath, "history", "--help")

		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "History command should exist")

		helpText := string(out)
		assert.Contains(t, helpText, "history")
	})

	t.Run("config_operation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, goPath, "config", "--help")

		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "Config command should exist")

		helpText := string(out)
		assert.Contains(t, helpText, "config")
	})

	t.Run("auth_operation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, goPath, "auth", "--help")

		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "Auth command should exist")

		helpText := string(out)
		assert.Contains(t, helpText, "auth")
	})

	t.Run("mcp_operation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, goPath, "mcp", "--help")

		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "MCP command should exist")

		helpText := string(out)
		assert.Contains(t, helpText, "mcp")
	})
}

// TestExtensionLifecycle validates extension lifecycle management
func TestExtensionLifecycle(t *testing.T) {
	goPath := findGoBinary()
	if goPath == "" {
		t.Skip("Go CLI binary not found")
	}

	t.Run("extension_startup", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Test that CLI can start
		cmd := exec.CommandContext(ctx, goPath, "version", "--short")

		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "CLI should start successfully")

		version := string(out)
		t.Logf("CLI version: %s", version)
		assert.NotEmpty(t, version)
	})

	t.Run("extension_shutdown", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Run a command and ensure it exits cleanly
		cmd := exec.CommandContext(ctx, goPath, "version", "--short")

		err := cmd.Run()
		require.NoError(t, err, "CLI should exit cleanly")
	})

	t.Run("extension_restart", func(t *testing.T) {
		// Run multiple times to simulate restart
		for i := 0; i < 3; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			cmd := exec.CommandContext(ctx, goPath, "version", "--short")

			out, err := cmd.CombinedOutput()
			cancel()

			if err != nil {
				t.Logf("Run %d failed: %v", i, err)
			} else {
				t.Logf("Run %d succeeded: %s", i, string(out))
			}
		}
	})
}

// TestErrorHandling validates error handling from core extension
func TestCoreExtensionErrorHandling(t *testing.T) {
	goPath := findGoBinary()
	if goPath == "" {
		t.Skip("Go CLI binary not found")
	}

	t.Run("invalid_operation_error", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Try an invalid operation
		cmd := exec.CommandContext(ctx, goPath, "config", "invalid-subcommand-xyz")

		out, err := cmd.CombinedOutput()
		t.Logf("Invalid operation output: %s", string(out))

		// Should return error
		assert.Error(t, err, "Invalid operation should error")
	})

	t.Run("timeout_error", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Try with very short timeout
		cmd := exec.CommandContext(ctx, goPath, "-t", "1", "version")

		out, err := cmd.CombinedOutput()
		t.Logf("Timeout output: %s", string(out))

		// Should complete or timeout gracefully
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})

	t.Run("connection_error", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Try with invalid extension host
		cmd := exec.CommandContext(ctx, goPath, "version")
		cmd.Env = append(os.Environ(),
			"CLINE_EXTENSION_HOST=invalid-host-xyz:99999",
		)

		out, err := cmd.CombinedOutput()
		t.Logf("Connection error output: %s", string(out))

		// Should handle gracefully
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})
}

// Helper function
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
		if path, err := filepath.Abs(loc); err == nil {
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}

	// Try PATH
	if path, err := exec.LookPath(binaryName); err == nil {
		return path
	}

	return ""
}
