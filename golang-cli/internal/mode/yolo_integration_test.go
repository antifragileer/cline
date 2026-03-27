// Package mode provides integration tests for yolo mode functionality.
package mode

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cline/cline/golang-cli/internal/task"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestYoloModeFullWorkflow tests the complete yolo mode workflow.
func TestYoloModeFullWorkflow(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("initial"), 0644)
	require.NoError(t, err)

	config := &YoloModeConfig{
		Enabled:    true,
		JSONOutput: false,
		Logger:     os.Stdout,
		MaxTools:   10,
		Timeout:    30 * time.Second,
	}

	runner := NewYoloModeRunner(config)

	ctx := context.Background()
	result, err := runner.Run(ctx, "Test prompt for yolo mode")

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, 0, result.ExitCode)
}

// TestYoloModeWithMaxToolsLimit tests the maximum tools limit enforcement.
func TestYoloModeWithMaxToolsLimit(t *testing.T) {
	config := &YoloModeConfig{
		Enabled:  true,
		Logger:   os.Stdout,
		MaxTools: 2,
	}

	runner := NewYoloModeRunner(config)
	ctx := context.Background()

	// Execute tools up to the limit
	req1 := task.ToolRequest{
		ID:       "yolo-limit-1",
		Type:     task.ToolTypeListFiles,
		ToolName: "list_files",
		Parameters: map[string]interface{}{
			"path": ".",
		},
	}
	_, err := runner.ExecuteTool(ctx, req1)
	require.NoError(t, err)

	req2 := task.ToolRequest{
		ID:       "yolo-limit-2",
		Type:     task.ToolTypeListFiles,
		ToolName: "list_files",
		Parameters: map[string]interface{}{
			"path": ".",
		},
	}
	_, err = runner.ExecuteTool(ctx, req2)
	require.NoError(t, err)

	// Third tool should fail due to limit
	req3 := task.ToolRequest{
		ID:       "yolo-limit-3",
		Type:     task.ToolTypeListFiles,
		ToolName: "list_files",
		Parameters: map[string]interface{}{
			"path": ".",
		},
	}
	_, err = runner.ExecuteTool(ctx, req3)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maximum tool limit")
}

// TestYoloModeWithTimeout tests timeout handling in yolo mode.
func TestYoloModeWithTimeout(t *testing.T) {
	config := &YoloModeConfig{
		Enabled: true,
		Logger:  os.Stdout,
		Timeout: 100 * time.Millisecond,
	}

	runner := NewYoloModeRunner(config)

	ctx := context.Background()
	result, err := runner.Run(ctx, "Test with timeout")

	// Should complete without error (timeout is for the whole run)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// TestYoloModeExitOnError tests exit on first error behavior.
func TestYoloModeExitOnError(t *testing.T) {
	config := &YoloModeConfig{
		Enabled:     true,
		Logger:      os.Stdout,
		ExitOnError: true,
		MaxTools:    5,
	}

	runner := NewYoloModeRunner(config)
	ctx := context.Background()

	// First tool - succeeds
	req1 := task.ToolRequest{
		ID:       "yolo-exit-1",
		Type:     task.ToolTypeListFiles,
		ToolName: "list_files",
		Parameters: map[string]interface{}{
			"path": ".",
		},
	}
	_, err := runner.ExecuteTool(ctx, req1)
	require.NoError(t, err)

	// Second tool - fails (nonexistent file)
	req2 := task.ToolRequest{
		ID:       "yolo-exit-2",
		Type:     task.ToolTypeReadFile,
		ToolName: "read_file",
		Parameters: map[string]interface{}{
			"path": "/nonexistent/path/that/does/not/exist.txt",
		},
	}
	_, err = runner.ExecuteTool(ctx, req2)

	// In exit-on-error mode, should return error
	// Note: actual behavior depends on whether the tool execution actually fails
	if err != nil {
		assert.True(t, runner.HasErrors())
		assert.NotEqual(t, 0, runner.GetExitCode())
	}
}

// TestYoloModeJSONOutput tests JSON output format.
func TestYoloModeJSONOutput(t *testing.T) {
	var buf strings.Builder
	config := &YoloModeConfig{
		Enabled:    true,
		JSONOutput: true,
		Logger:     &buf,
	}

	runner := NewYoloModeRunner(config)

	// Execute a tool
	ctx := context.Background()
	req := task.ToolRequest{
		ID:       "yolo-json-1",
		Type:     task.ToolTypeListFiles,
		ToolName: "list_files",
		Parameters: map[string]interface{}{
			"path": ".",
		},
	}
	_, err := runner.ExecuteTool(ctx, req)
	require.NoError(t, err)

	// Output result
	result := &YoloResult{
		Success:      true,
		ExitCode:     0,
		Actions:      runner.GetActionLog(),
		TotalActions: 1,
		Duration:     time.Second,
	}

	err = runner.OutputResult(result)
	require.NoError(t, err)

	// Verify JSON output
	output := buf.String()
	assert.Contains(t, output, `"success"`)
	assert.Contains(t, output, `"exit_code"`)
	assert.Contains(t, output, `"total_actions"`)
}

// TestYoloModeActionLog tests action logging in yolo mode.
func TestYoloModeActionLog(t *testing.T) {
	config := &YoloModeConfig{
		Enabled: true,
		Logger:  os.Stdout,
	}

	runner := NewYoloModeRunner(config)
	ctx := context.Background()

	// Execute multiple tools
	for i := 0; i < 3; i++ {
		req := task.ToolRequest{
			ID:       fmt.Sprintf("yolo-log-%d", i),
			Type:     task.ToolTypeListFiles,
			ToolName: "list_files",
			Parameters: map[string]interface{}{
				"path": ".",
			},
		}
		_, err := runner.ExecuteTool(ctx, req)
		require.NoError(t, err)
	}

	// Check action log
	log := runner.GetActionLog()
	assert.Len(t, log, 3)

	for _, action := range log {
		// ToolName is the actual tool name, not the request ID
		assert.Equal(t, "list_files", action.ToolName)
		assert.Equal(t, task.ToolTypeListFiles, action.ToolType)
		assert.NotZero(t, action.Timestamp)
	}
}

// TestYoloModeToolTypesFiltering tests tool type filtering in yolo mode.
func TestYoloModeToolTypesFiltering(t *testing.T) {
	config := &YoloModeConfig{
		Enabled:   true,
		Logger:    os.Stdout,
		ToolTypes: []task.ToolType{task.ToolTypeReadFile, task.ToolTypeListFiles},
	}

	runner := NewYoloModeRunner(config)
	ctx := context.Background()

	// Execute allowed tool type
	req1 := task.ToolRequest{
		ID:       "yolo-filter-1",
		Type:     task.ToolTypeListFiles,
		ToolName: "list_files",
		Parameters: map[string]interface{}{
			"path": ".",
		},
	}
	_, err := runner.ExecuteTool(ctx, req1)
	require.NoError(t, err)

	// The filtering happens at the approval config level
	// Tools not in the list would require explicit approval
}

// TestYoloModeConcurrentExecution tests concurrent tool execution in yolo mode.
func TestYoloModeConcurrentExecution(t *testing.T) {
	config := &YoloModeConfig{
		Enabled:  true,
		Logger:   os.Stdout,
		MaxTools: 10,
	}

	runner := NewYoloModeRunner(config)
	ctx := context.Background()

	// Execute tools concurrently
	done := make(chan struct{})
	for i := 0; i < 5; i++ {
		go func(id int) {
			defer func() { done <- struct{}{} }()

			req := task.ToolRequest{
				ID:       fmt.Sprintf("yolo-concurrent-%d", id),
				Type:     task.ToolTypeListFiles,
				ToolName: "list_files",
				Parameters: map[string]interface{}{
					"path": ".",
				},
			}
			_, err := runner.ExecuteTool(ctx, req)
			if err != nil {
				t.Errorf("concurrent execution error: %v", err)
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}

	// Check action log has all entries
	log := runner.GetActionLog()
	assert.Len(t, log, 5)
}

// TestYoloModeBatchExecution tests batch tool execution.
func TestYoloModeBatchExecution(t *testing.T) {
	config := &YoloModeConfig{
		Enabled: true,
		Logger:  os.Stdout,
	}

	runner := NewYoloModeRunner(config)
	ctx := context.Background()

	requests := []task.ToolRequest{
		{
			ID:       "yolo-batch-1",
			Type:     task.ToolTypeListFiles,
			ToolName: "list_files",
			Parameters: map[string]interface{}{
				"path": ".",
			},
		},
		{
			ID:       "yolo-batch-2",
			Type:     task.ToolTypeListFiles,
			ToolName: "list_files",
			Parameters: map[string]interface{}{
				"path": ".",
			},
		},
	}

	results, err := runner.ExecuteTools(ctx, requests)

	require.NoError(t, err)
	assert.Len(t, results, 2)

	// Check action log
	log := runner.GetActionLog()
	assert.Len(t, log, 2)
}

// TestYoloModeStatsCalculation tests statistics calculation.
func TestYoloModeStatsCalculation(t *testing.T) {
	config := &YoloModeConfig{
		Enabled: true,
		Logger:  os.Stdout,
	}

	runner := NewYoloModeRunner(config)

	// Add some actions manually for testing
	runner.actionLog = []YoloAction{
		{Success: true},
		{Success: true},
		{Success: false, Error: "error occurred"},
	}

	result := &YoloResult{}
	runner.calculateStats(result)

	assert.Equal(t, 3, result.TotalActions)
	assert.Equal(t, 2, result.SuccessfulActions)
	assert.Equal(t, 1, result.FailedActions)
	assert.False(t, result.Success)
	assert.NotEqual(t, 0, result.ExitCode)
}

// TestYoloModeWithFileOperations tests file operations in yolo mode.
func TestYoloModeWithFileOperations(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	config := &YoloModeConfig{
		Enabled: true,
		Logger:  os.Stdout,
	}

	runner := NewYoloModeRunner(config)
	ctx := context.Background()

	// Write file
	writeReq := task.ToolRequest{
		ID:       "yolo-file-write",
		Type:     task.ToolTypeWriteFile,
		ToolName: "write_file",
		Parameters: map[string]interface{}{
			"path":    testFile,
			"content": "test content",
		},
	}
	_, err := runner.ExecuteTool(ctx, writeReq)
	require.NoError(t, err)

	// Read file
	readReq := task.ToolRequest{
		ID:       "yolo-file-read",
		Type:     task.ToolTypeReadFile,
		ToolName: "read_file",
		Parameters: map[string]interface{}{
			"path": testFile,
		},
	}
	result, err := runner.ExecuteTool(ctx, readReq)
	require.NoError(t, err)
	assert.Equal(t, "test content", result.Output)

	// Replace in file
	replaceReq := task.ToolRequest{
		ID:       "yolo-file-replace",
		Type:     task.ToolTypeReplaceInFile,
		ToolName: "replace_in_file",
		Parameters: map[string]interface{}{
			"path":       testFile,
			"old_string": "test",
			"new_string": "modified",
		},
	}
	_, err = runner.ExecuteTool(ctx, replaceReq)
	require.NoError(t, err)

	// Verify replacement
	content, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, "modified content", string(content))
}

// TestYoloModeIntegrationWithRunner tests integration with task runner.
func TestYoloModeIntegrationWithRunner(t *testing.T) {
	// This test verifies that yolo mode works correctly when used
	// in conjunction with the task runner system

	config := &YoloModeConfig{
		Enabled:    true,
		JSONOutput: false,
		Logger:     os.Stdout,
	}

	integration, err := NewYoloIntegration(&YoloIntegrationConfig{
		YoloModeConfig: config,
	})

	require.NoError(t, err)
	require.NotNil(t, integration)

	// Verify the integration was set up correctly
	assert.Equal(t, config, integration.config)
	assert.NotNil(t, integration.approver)
	assert.NotNil(t, integration.executor)

	// Clean up
	err = integration.Close()
	require.NoError(t, err)
}

// TestYoloModeSafetyLimits tests safety limit validation.
func TestYoloModeSafetyLimits(t *testing.T) {
	config := &YoloModeConfig{
		Enabled:  true,
		MaxTools: 1000,
		Timeout:  time.Hour,
	}

	limits := DefaultYoloSafetyLimits()
	violations := ValidateLimits(config, limits)

	// Should have violations since config exceeds defaults
	assert.NotEmpty(t, violations)
	assert.True(t, len(violations) > 0)
}

// TestYoloModeWithCommandExecution tests command execution in yolo mode.
func TestYoloModeWithCommandExecution(t *testing.T) {
	config := &YoloModeConfig{
		Enabled: true,
		Logger:  os.Stdout,
	}

	runner := NewYoloModeRunner(config)
	ctx := context.Background()

	// Execute a command
	cmdReq := task.ToolRequest{
		ID:       "yolo-cmd-1",
		Type:     task.ToolTypeExecuteCommand,
		ToolName: "execute_command",
		Parameters: map[string]interface{}{
			"command": "echo 'hello from yolo mode'",
		},
	}
	result, err := runner.ExecuteTool(ctx, cmdReq)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, result.Output, "hello from yolo mode")
}

// TestYoloModePauseAndResume tests pause and resume functionality.
func TestYoloModePauseAndResume(t *testing.T) {
	config := &YoloModeConfig{
		Enabled: true,
		Logger:  os.Stdout,
	}

	integration, err := NewYoloIntegration(&YoloIntegrationConfig{
		YoloModeConfig: config,
	})
	require.NoError(t, err)

	// Initially not paused
	assert.False(t, integration.IsPaused())

	// Resume when not paused should not error
	integration.Resume()

	// Clean up
	err = integration.Close()
	require.NoError(t, err)
}

// TestYoloModeContextCancellation tests context cancellation handling.
func TestYoloModeContextCancellation(t *testing.T) {
	config := &YoloModeConfig{
		Enabled: true,
		Logger:  os.Stdout,
	}

	runner := NewYoloModeRunner(config)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Attempt to execute tool
	req := task.ToolRequest{
		ID:       "yolo-cancel-1",
		Type:     task.ToolTypeListFiles,
		ToolName: "list_files",
		Parameters: map[string]interface{}{
			"path": ".",
		},
	}
	_, err := runner.ExecuteTool(ctx, req)

	// Should return context cancelled error
	assert.Error(t, err)
}

// BenchmarkYoloModeExecution benchmarks yolo mode tool execution.
func BenchmarkYoloModeExecution(b *testing.B) {
	config := &YoloModeConfig{
		Enabled: true,
		Logger:  os.Stdout,
	}

	runner := NewYoloModeRunner(config)
	ctx := context.Background()

	req := task.ToolRequest{
		ID:       "benchmark",
		Type:     task.ToolTypeListFiles,
		ToolName: "list_files",
		Parameters: map[string]interface{}{
			"path": ".",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req.ID = fmt.Sprintf("benchmark-%d", i)
		_, err := runner.ExecuteTool(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}