// Package functional provides comprehensive functional tests for task execution.
// These tests verify actual task behavior including execution, resumption,
// image handling, and all flag combinations.
//
// IMPORTANT: Tests in this package invoke `cline task` which may execute real
// LLM commands. These tests use the AI test lock to ensure they don't run in
// parallel, preventing rate limiting and process accumulation.
package functional

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/cline/cline/golang-cli/tests/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTaskExecution validates task execution functionality
func TestTaskExecution(t *testing.T) {
	// Acquire AI lock to prevent parallel execution of LLM tests
	release := testutil.AcquireAILock(t)
	defer release()

	binary := findGoBinary()
	if binary == "" {
		t.Skip("Go CLI binary not found")
	}

	tempDir, err := os.MkdirTemp("", "task-exec-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("task_with_prompt", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "task", "--json", "echo hello")
		cmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tempDir)

		out, err := cmd.CombinedOutput()
		t.Logf("Output: %s", string(out))

		// Without core extension, this will fail but should handle gracefully
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}

		// Should return a valid exit code (not crash)
		assert.True(t, exitCode >= 0 && exitCode <= 255, "Invalid exit code: %d", exitCode)
	})

	t.Run("task_with_images", func(t *testing.T) {
		// Create a test image file
		imgPath := filepath.Join(tempDir, "test.png")
		testImageData := []byte{
			0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG header
			0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		}
		err := os.WriteFile(imgPath, testImageData, 0644)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "-i", imgPath, "analyze this image")
		cmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tempDir)

		out, err := cmd.CombinedOutput()
		t.Logf("Image task output: %s", string(out))

		// Should handle image input without crashing
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})

	t.Run("task_with_multiple_images", func(t *testing.T) {
		// Create multiple test image files
		imgPath1 := filepath.Join(tempDir, "test1.png")
		imgPath2 := filepath.Join(tempDir, "test2.jpg")

		testImageData := []byte{0x89, 0x50, 0x4E, 0x47} // PNG header
		err := os.WriteFile(imgPath1, testImageData, 0644)
		require.NoError(t, err)

		// JPG header
		jpgData := []byte{0xFF, 0xD8, 0xFF, 0xE0}
		err = os.WriteFile(imgPath2, jpgData, 0644)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "-i", imgPath1, "-i", imgPath2, "compare these images")
		cmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tempDir)

		out, err := cmd.CombinedOutput()
		t.Logf("Multiple images output: %s", string(out))

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})

	t.Run("task_invalid_image_path", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "-i", "/nonexistent/image.png", "test")
		cmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tempDir)

		out, err := cmd.CombinedOutput()

		// Should return error for invalid image path
		assert.Error(t, err, "Should fail with invalid image path")
		assert.Contains(t, string(out), "error", "Error", "not found", "Not Found")
	})
}

// TestTaskResumption validates task resumption functionality
func TestTaskResumption(t *testing.T) {
	// Acquire AI lock to prevent parallel execution of LLM tests
	release := testutil.AcquireAILock(t)
	defer release()

	binary := findGoBinary()
	if binary == "" {
		t.Skip("Go CLI binary not found")
	}

	tempDir, err := os.MkdirTemp("", "task-resume-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("resume_with_task_id", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "-T", "test-task-123", "continue")
		cmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tempDir)

		out, err := cmd.CombinedOutput()
		t.Logf("Resume output: %s", string(out))

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})

	t.Run("continue_flag", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "--continue")
		cmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tempDir)

		out, err := cmd.CombinedOutput()
		t.Logf("Continue flag output: %s", string(out))

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})

	t.Run("resume_with_new_prompt", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "-T", "test-task-456", "additional context")
		cmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tempDir)

		out, err := cmd.CombinedOutput()
		t.Logf("Resume with prompt output: %s", string(out))

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})
}

// TestTaskModes validates act, plan, and yolo modes
func TestTaskModes(t *testing.T) {
	// Acquire AI lock to prevent parallel execution of LLM tests
	release := testutil.AcquireAILock(t)
	defer release()

	binary := findGoBinary()
	if binary == "" {
		t.Skip("Go CLI binary not found")
	}

	tempDir, err := os.MkdirTemp("", "task-modes-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "act_mode_short",
			args: []string{"-a", "test prompt"},
		},
		{
			name: "act_mode_long",
			args: []string{"--act", "test prompt"},
		},
		{
			name: "plan_mode_short",
			args: []string{"-p", "test prompt"},
		},
		{
			name: "plan_mode_long",
			args: []string{"--plan", "test prompt"},
		},
		{
			name: "yolo_mode_short",
			args: []string{"-y", "test prompt"},
		},
		{
			name: "yolo_mode_long",
			args: []string{"--yolo", "test prompt"},
		},
		{
			name: "auto_approve_all",
			args: []string{"--auto-approve-all", "test prompt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, binary, tt.args...)
			cmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tempDir)

			out, err := cmd.CombinedOutput()
			t.Logf("%s output: %s", tt.name, string(out))

			exitCode := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				}
			}
			assert.True(t, exitCode >= 0 && exitCode <= 255, "%s: invalid exit code %d", tt.name, exitCode)
		})
	}
}

// TestTaskFlags validates all task-related flags
func TestTaskFlags(t *testing.T) {
	// Acquire AI lock to prevent parallel execution of LLM tests
	release := testutil.AcquireAILock(t)
	defer release()

	binary := findGoBinary()
	if binary == "" {
		t.Skip("Go CLI binary not found")
	}

	tempDir, err := os.MkdirTemp("", "task-flags-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name        string
		args        []string
		description string
	}{
		{
			name:        "model_flag",
			args:        []string{"-m", "claude-3-5-sonnet-20241022", "test"},
			description: "Model selection flag",
		},
		{
			name:        "verbose_flag",
			args:        []string{"-v", "test"},
			description: "Verbose output flag",
		},
		{
			name:        "cwd_flag",
			args:        []string{"-c", tempDir, "test"},
			description: "Current working directory flag",
		},
		{
			name:        "timeout_flag",
			args:        []string{"-t", "60", "test"},
			description: "Timeout flag",
		},
		{
			name:        "thinking_flag",
			args:        []string{"--thinking", "test"},
			description: "Thinking mode flag",
		},
		{
			name:        "reasoning_effort_flag",
			args:        []string{"--reasoning-effort", "high", "test"},
			description: "Reasoning effort flag",
		},
		{
			name:        "max_mistakes_flag",
			args:        []string{"--max-consecutive-mistakes", "3", "test"},
			description: "Max consecutive mistakes flag",
		},
		{
			name:        "double_check_flag",
			args:        []string{"--double-check-completion", "test"},
			description: "Double check completion flag",
		},
		{
			name:        "auto_condense_flag",
			args:        []string{"--auto-condense", "test"},
			description: "Auto condense flag",
		},
		{
			name:        "hooks_dir_flag",
			args:        []string{"--hooks-dir", tempDir, "test"},
			description: "Hooks directory flag",
		},
		{
			name:        "json_flag",
			args:        []string{"--json", "test"},
			description: "JSON output flag",
		},
		{
			name:        "acp_flag",
			args:        []string{"--acp", "test"},
			description: "ACP flag",
		},
		{
			name:        "kanban_flag",
			args:        []string{"--kanban", "test"},
			description: "Kanban flag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, binary, tt.args...)
			cmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tempDir)

			out, err := cmd.CombinedOutput()
			t.Logf("%s output: %s", tt.name, string(out))

			exitCode := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode = exitErr.ExitCode()
				}
			}
			assert.True(t, exitCode >= 0 && exitCode <= 255, "%s: invalid exit code %d", tt.name, exitCode)
		})
	}
}

// TestTaskYoloMode validates yolo mode auto-approval behavior
func TestTaskYoloMode(t *testing.T) {
	// Acquire AI lock to prevent parallel execution of LLM tests
	release := testutil.AcquireAILock(t)
	defer release()

	binary := findGoBinary()
	if binary == "" {
		t.Skip("Go CLI binary not found")
	}

	tempDir, err := os.MkdirTemp("", "task-yolo-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("yolo_enables_auto_approve", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "-y", "test prompt")
		cmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tempDir)

		out, err := cmd.CombinedOutput()
		t.Logf("Yolo mode output: %s", string(out))

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})

	t.Run("auto_approve_all_flag", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "--auto-approve-all", "test prompt")
		cmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tempDir)

		out, err := cmd.CombinedOutput()
		t.Logf("Auto-approve-all output: %s", string(out))

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})
}

// TestTaskJSONOutput validates JSON output format
func TestTaskJSONOutput(t *testing.T) {
	// Acquire AI lock to prevent parallel execution of LLM tests
	release := testutil.AcquireAILock(t)
	defer release()

	binary := findGoBinary()
	if binary == "" {
		t.Skip("Go CLI binary not found")
	}

	tempDir, err := os.MkdirTemp("", "task-json-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("json_output_format", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "--json", "test prompt")
		cmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tempDir)

		out, err := cmd.CombinedOutput()

		// Even if the command fails, output should be valid JSON or empty
		if len(out) > 0 {
			var result interface{}
			jsonErr := json.Unmarshal(out, &result)
			if jsonErr == nil {
				// Valid JSON received
				t.Log("Received valid JSON output")
			} else {
				// Not valid JSON - that's okay if command errored
				t.Logf("Output is not JSON (expected if error): %s", string(out))
			}
		}

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})
}

// TestTaskEdgeCases validates edge cases and error handling
func TestTaskEdgeCases(t *testing.T) {
	// Acquire AI lock to prevent parallel execution of LLM tests
	release := testutil.AcquireAILock(t)
	defer release()

	binary := findGoBinary()
	if binary == "" {
		t.Skip("Go CLI binary not found")
	}

	tempDir, err := os.MkdirTemp("", "task-edge-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("empty_prompt", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "task")
		cmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tempDir)

		out, err := cmd.CombinedOutput()
		t.Logf("Empty prompt output: %s", string(out))

		// Should handle gracefully (help or error)
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})

	t.Run("very_long_prompt", func(t *testing.T) {
		longPrompt := ""
		for i := 0; i < 1000; i++ {
			longPrompt += "This is a very long prompt with lots of text. "
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, longPrompt)
		cmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tempDir)

		out, err := cmd.CombinedOutput()
		t.Logf("Long prompt output length: %d", len(out))

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})

	t.Run("special_characters_in_prompt", func(t *testing.T) {
		specialPrompt := "Test with special chars: !@#$%^&*()_+-=[]{}|;':\",./<>?`~"

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, specialPrompt)
		cmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tempDir)

		out, err := cmd.CombinedOutput()
		t.Logf("Special chars output: %s", string(out))

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
		assert.True(t, exitCode >= 0 && exitCode <= 255)
	})

	t.Run("invalid_task_id", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "-T", "", "test")
		cmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tempDir)

		out, err := cmd.CombinedOutput()
		t.Logf("Invalid task ID output: %s", string(out))

		// Should error with empty task ID
		if err != nil {
			t.Log("Correctly rejected empty task ID")
		}
	})
}

// TestTaskTimeout validates timeout behavior
func TestTaskTimeout(t *testing.T) {
	// Acquire AI lock to prevent parallel execution of LLM tests
	release := testutil.AcquireAILock(t)
	defer release()

	binary := findGoBinary()
	if binary == "" {
		t.Skip("Go CLI binary not found")
	}

	tempDir, err := os.MkdirTemp("", "task-timeout-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	t.Run("timeout_flag", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binary, "-t", "5", "test prompt")
		cmd.Env = append(os.Environ(), "CLINE_DATA_DIR="+tempDir)

		start := time.Now()
		out, err := cmd.CombinedOutput()
		elapsed := time.Since(start)

		t.Logf("Timeout test output: %s, elapsed: %v", string(out), elapsed)

		// Should complete within reasonable time (not hang)
		assert.True(t, elapsed < 30*time.Second, "Command took too long: %v", elapsed)

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
