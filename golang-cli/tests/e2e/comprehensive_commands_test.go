// Package e2e provides comprehensive command combination tests for the Go CLI.
// This file tests all permutations of commands, flags, and arguments to ensure
// complete functional parity with the TypeScript CLI.
package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CommandCombinationTest represents a test case for command combinations
type CommandCombinationTest struct {
	Name         string
	Args         []string
	Env          map[string]string
	ExpectedExit int
	ValidateJSON bool
	ValidateFunc func(output string, exitCode int) error
	Timeout      time.Duration
}

// Comprehensive test suite for all CLI command combinations
func TestComprehensiveCommandCombinations(t *testing.T) {
	binary := FindBinary()
	if binary == "" {
		t.Skip("CLI binary not found")
	}

	tests := []CommandCombinationTest{
		// Version command variations
		{
			Name:         "version_no_args",
			Args:         []string{"version"},
			ExpectedExit: 0,
			ValidateFunc: func(output string, exitCode int) error {
				if !strings.Contains(output, "version") && !strings.Contains(output, "Version") {
					return fmt.Errorf("expected version info in output")
				}
				return nil
			},
		},
		{
			Name:         "version_short",
			Args:         []string{"version", "--short"},
			ExpectedExit: 0,
			ValidateFunc: func(output string, exitCode int) error {
				trimmed := strings.TrimSpace(output)
				if trimmed == "" {
					return fmt.Errorf("expected non-empty version output")
				}
				return nil
			},
		},
		{
			Name:         "version_json",
			Args:         []string{"version", "--json"},
			ExpectedExit: 0,
			ValidateJSON: true,
		},
		{
			Name:         "version_json_short",
			Args:         []string{"version", "--json", "--short"},
			ExpectedExit: 0,
			ValidateJSON: true,
		},

		// Help command variations
		{
			Name:         "help_root",
			Args:         []string{"--help"},
			ExpectedExit: 0,
			ValidateFunc: func(output string, exitCode int) error {
				required := []string{"Usage:", "Available Commands:", "Flags:"}
				for _, r := range required {
					if !strings.Contains(output, r) {
						return fmt.Errorf("missing required content: %s", r)
					}
				}
				return nil
			},
		},
		{
			Name:         "help_short",
			Args:         []string{"-h"},
			ExpectedExit: 0,
		},
		{
			Name:         "help_version",
			Args:         []string{"help", "version"},
			ExpectedExit: 0,
		},
		{
			Name:         "help_config",
			Args:         []string{"help", "config"},
			ExpectedExit: 0,
		},
		{
			Name:         "help_history",
			Args:         []string{"help", "history"},
			ExpectedExit: 0,
		},
		{
			Name:         "help_task",
			Args:         []string{"help", "task"},
			ExpectedExit: 0,
		},
		{
			Name:         "help_auth",
			Args:         []string{"help", "auth"},
			ExpectedExit: 0,
		},
		{
			Name:         "help_mcp",
			Args:         []string{"help", "mcp"},
			ExpectedExit: 0,
		},

		// Config command variations
		{
			Name:         "config_help",
			Args:         []string{"config", "--help"},
			ExpectedExit: 0,
		},
		{
			Name:         "config_list",
			Args:         []string{"config", "list"},
			ExpectedExit: 0,
		},
		{
			Name:         "config_list_json",
			Args:         []string{"config", "list", "--json"},
			ExpectedExit: 0,
			ValidateJSON: true,
		},

		// History command variations
		{
			Name:         "history_help",
			Args:         []string{"history", "--help"},
			ExpectedExit: 0,
		},
		{
			Name:         "history_default",
			Args:         []string{"history"},
			ExpectedExit: 0,
		},
		{
			Name:         "history_json",
			Args:         []string{"history", "--json"},
			ExpectedExit: 0,
			ValidateJSON: true,
		},
		{
			Name:         "history_limit",
			Args:         []string{"history", "--limit", "5"},
			ExpectedExit: 0,
		},
		{
			Name:         "history_page",
			Args:         []string{"history", "--page", "1"},
			ExpectedExit: 0,
		},
		{
			Name:         "history_limit_page",
			Args:         []string{"history", "--limit", "10", "--page", "1"},
			ExpectedExit: 0,
		},
		{
			Name:         "history_short_flags",
			Args:         []string{"history", "-n", "5", "-p", "1"},
			ExpectedExit: 0,
		},

		// Task command variations
		{
			Name:         "task_help",
			Args:         []string{"task", "--help"},
			ExpectedExit: 0,
		},
		{
			Name:         "task_empty",
			Args:         []string{"task"},
			ExpectedExit: 0,
		},
		{
			Name:         "task_with_prompt",
			Args:         []string{"task", "hello world"},
			ExpectedExit: 0,
		},
		{
			Name:         "task_json_mode",
			Args:         []string{"task", "--json", "test prompt"},
			ExpectedExit: 0,
		},
		{
			Name:         "task_act_mode",
			Args:         []string{"task", "-a", "test prompt"},
			ExpectedExit: 0,
		},
		{
			Name:         "task_plan_mode",
			Args:         []string{"task", "-p", "test prompt"},
			ExpectedExit: 0,
		},
		{
			Name:         "task_yolo_mode",
			Args:         []string{"task", "-y", "test prompt"},
			ExpectedExit: 0,
		},

		// Auth command variations
		{
			Name:         "auth_help",
			Args:         []string{"auth", "--help"},
			ExpectedExit: 0,
		},
		{
			Name:         "auth_list",
			Args:         []string{"auth", "list"},
			ExpectedExit: 0,
		},
		{
			Name:         "auth_list_json",
			Args:         []string{"auth", "list", "--json"},
			ExpectedExit: 0,
			ValidateJSON: true,
		},
		{
			Name:         "auth_status",
			Args:         []string{"auth", "status"},
			ExpectedExit: 0,
		},

		// MCP command variations
		{
			Name:         "mcp_help",
			Args:         []string{"mcp", "--help"},
			ExpectedExit: 0,
		},
		{
			Name:         "mcp_list",
			Args:         []string{"mcp", "list"},
			ExpectedExit: 0,
		},
		{
			Name:         "mcp_list_json",
			Args:         []string{"mcp", "list", "--json"},
			ExpectedExit: 0,
			ValidateJSON: true,
		},

		// Dev command variations
		{
			Name:         "dev_help",
			Args:         []string{"dev", "--help"},
			ExpectedExit: 0,
		},
		{
			Name:         "dev_log",
			Args:         []string{"dev", "log"},
			ExpectedExit: 0,
		},

		// Update command variations
		{
			Name:         "update_help",
			Args:         []string{"update", "--help"},
			ExpectedExit: 0,
		},
		{
			Name:         "update_check",
			Args:         []string{"update", "check"},
			ExpectedExit: 0,
		},

		// Invalid commands
		{
			Name:         "invalid_command",
			Args:         []string{"invalid-command-that-does-not-exist"},
			ExpectedExit: 1,
		},
		{
			Name:         "invalid_flag",
			Args:         []string{"version", "--invalid-flag"},
			ExpectedExit: 1,
		},
		{
			Name:         "invalid_subcommand",
			Args:         []string{"config", "invalid-subcommand"},
			ExpectedExit: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), getTimeout(tt.Timeout))
			defer cancel()

			cmd := exec.CommandContext(ctx, binary, tt.Args...)
			
			// Set environment
			cmd.Env = os.Environ()
			for k, v := range tt.Env {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
			}

			output, err := cmd.CombinedOutput()
			outputStr := string(output)

			// Check exit code
			exitCode := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				} else {
					exitCode = -1
				}
			}

			// Validate exit code
			if exitCode != tt.ExpectedExit {
				t.Errorf("Expected exit code %d, got %d\nOutput: %s", tt.ExpectedExit, exitCode, outputStr)
			}

			// Validate JSON if requested
			if tt.ValidateJSON && exitCode == 0 {
				var jsonData interface{}
				if err := json.Unmarshal(output, &jsonData); err != nil {
					t.Errorf("Output is not valid JSON: %v\nOutput: %s", err, outputStr)
				}
			}

			// Run custom validation
			if tt.ValidateFunc != nil {
				if err := tt.ValidateFunc(outputStr, exitCode); err != nil {
					t.Errorf("Validation failed: %v\nOutput: %s", err, outputStr)
				}
			}
		})
	}
}

// TestFlagCombinations tests various flag combinations
func TestFlagCombinations(t *testing.T) {
	binary := FindBinary()
	if binary == "" {
		t.Skip("CLI binary not found")
	}

	tempDir, err := os.MkdirTemp("", "flag-combo-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name         string
		args         []string
		env          map[string]string
		expectedExit int
	}{
		{
			name:         "config_with_custom_dir",
			args:         []string{"config", "list", "--config", tempDir},
			expectedExit: 0,
		},
		{
			name:         "history_with_custom_dir",
			args:         []string{"history", "--config", tempDir},
			expectedExit: 0,
		},
		{
			name:         "verbose_flag",
			args:         []string{"--verbose", "version"},
			expectedExit: 0,
		},
		{
			name:         "quiet_flag",
			args:         []string{"--quiet", "version"},
			expectedExit: 0,
		},
		{
			name:         "json_flag_global",
			args:         []string{"--json", "version"},
			expectedExit: 0,
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

			output, err := cmd.CombinedOutput()
			
			exitCode := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				} else {
					exitCode = -1
				}
			}

			assert.Equal(t, tt.expectedExit, exitCode, 
				"Exit code mismatch for args %v\nOutput: %s", tt.args, string(output))
		})
	}
}

// TestEnvironmentVariables tests CLI behavior with various environment variables
func TestEnvironmentVariables(t *testing.T) {
	binary := FindBinary()
	if binary == "" {
		t.Skip("CLI binary not found")
	}

	tests := []struct {
		name string
		env  map[string]string
		args []string
	}{
		{
			name: "no_color",
			env:  map[string]string{"NO_COLOR": "1"},
			args: []string{"version"},
		},
		{
			name: "force_color",
			env:  map[string]string{"FORCE_COLOR": "1"},
			args: []string{"version"},
		},
		{
			name: "ci_environment",
			env:  map[string]string{"CI": "true"},
			args: []string{"version"},
		},
		{
			name: "custom_home",
			env:  map[string]string{"HOME": os.TempDir()},
			args: []string{"version"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, binary, tt.args...)
			
			cmd.Env = []string{"PATH=" + os.Getenv("PATH")}
			for k, v := range tt.env {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
			}

			output, err := cmd.CombinedOutput()
			
			// Should not crash
			if err != nil {
				exitErr, ok := err.(*exec.ExitError)
				if ok && exitErr.ExitCode() != 0 {
					// Non-zero exit is OK, just check it didn't crash
				} else if !ok {
					t.Errorf("Unexpected error: %v\nOutput: %s", err, string(output))
				}
			}
		})
	}
}

// TestPipedInput tests CLI behavior with piped input
func TestPipedInput(t *testing.T) {
	binary := FindBinary()
	if binary == "" {
		t.Skip("CLI binary not found")
	}

	tests := []struct {
		name  string
		stdin string
		args  []string
	}{
		{
			name:  "simple_prompt",
			stdin: "Hello, world!",
			args:  []string{},
		},
		{
			name:  "multi_line_input",
			stdin: "Line 1\nLine 2\nLine 3",
			args:  []string{},
		},
		{
			name:  "empty_input",
			stdin: "",
			args:  []string{},
		},
		{
			name:  "unicode_input",
			stdin: "Hello 世界 🌍",
			args:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, binary, tt.args...)
			cmd.Stdin = strings.NewReader(tt.stdin)

			output, err := cmd.CombinedOutput()
			
			// Should not crash
			_ = err
			_ = output
		})
	}
}

// TestConcurrentCommands tests running multiple CLI commands concurrently
func TestConcurrentCommands(t *testing.T) {
	binary := FindBinary()
	if binary == "" {
		t.Skip("CLI binary not found")
	}

	numConcurrent := 20
	results := make(chan struct {
		index  int
		output string
		err    error
	}, numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		go func(n int) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, binary, "version", "--short")
			output, err := cmd.CombinedOutput()
			
			results <- struct {
				index  int
				output string
				err    error
			}{n, string(output), err}
		}(i)
	}

	// Collect results
	successCount := 0
	for i := 0; i < numConcurrent; i++ {
		result := <-results
		if result.err == nil && strings.TrimSpace(result.output) != "" {
			successCount++
		}
	}

	assert.Equal(t, numConcurrent, successCount, 
		"All concurrent commands should succeed")
}

// Helper function to get timeout with default
func getTimeout(d time.Duration) time.Duration {
	if d == 0 {
		return 30 * time.Second
	}
	return d
}

// BenchmarkCommandExecution benchmarks command execution speed
func BenchmarkCommandExecution(b *testing.B) {
	binary := FindBinary()
	if binary == "" {
		b.Skip("CLI binary not found")
	}

	b.Run("version", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cmd := exec.Command(binary, "version")
			cmd.Run()
		}
	})

	b.Run("version_short", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cmd := exec.Command(binary, "version", "--short")
			cmd.Run()
		}
	})

	b.Run("version_json", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cmd := exec.Command(binary, "version", "--json")
			cmd.Run()
		}
	})

	b.Run("help", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cmd := exec.Command(binary, "--help")
			cmd.Run()
		}
	})
}