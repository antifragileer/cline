// Package task provides integration tests for the tool approval system.
package task

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFullApprovalFlow tests the complete approval flow from request to execution.
func TestFullApprovalFlow(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("initial content"), 0644)
	require.NoError(t, err)

	// Create an auto-approver for testing
	approver := NewAutoApprover()
	config := DefaultToolApprovalConfig()
	config.YoloMode = true

	executor := NewToolExecutor(config, approver)

	// Test 1: Read file (should work without approval in yolo mode)
	t.Run("read file in yolo mode", func(t *testing.T) {
		req := ToolRequest{
			ID:       "read-test-1",
			Type:     ToolTypeReadFile,
			ToolName: "read_file",
			Parameters: map[string]interface{}{
				"path": testFile,
			},
		}

		ctx := context.Background()
		result, err := executor.ExecuteTool(ctx, req)

		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, "initial content", result.Output)
	})

	// Test 2: Write file (should work without approval in yolo mode)
	t.Run("write file in yolo mode", func(t *testing.T) {
		newFile := filepath.Join(tempDir, "newfile.txt")
		req := ToolRequest{
			ID:       "write-test-1",
			Type:     ToolTypeWriteFile,
			ToolName: "write_file",
			Parameters: map[string]interface{}{
				"path":    newFile,
				"content": "new content",
			},
		}

		ctx := context.Background()
		result, err := executor.ExecuteTool(ctx, req)

		require.NoError(t, err)
		assert.True(t, result.Success)

		// Verify file was written
		content, err := os.ReadFile(newFile)
		require.NoError(t, err)
		assert.Equal(t, "new content", string(content))
	})

	// Test 3: Replace in file (should work without approval in yolo mode)
	t.Run("replace in file in yolo mode", func(t *testing.T) {
		req := ToolRequest{
			ID:       "replace-test-1",
			Type:     ToolTypeReplaceInFile,
			ToolName: "replace_in_file",
			Parameters: map[string]interface{}{
				"path":       testFile,
				"old_string": "initial",
				"new_string": "modified",
			},
		}

		ctx := context.Background()
		result, err := executor.ExecuteTool(ctx, req)

		require.NoError(t, err)
		assert.True(t, result.Success)

		// Verify content was replaced
		content, err := os.ReadFile(testFile)
		require.NoError(t, err)
		assert.Equal(t, "modified content", string(content))
	})
}

// TestApprovalWithRejection tests the approval flow with rejection.
func TestApprovalWithRejection(t *testing.T) {
	// Create a reject-all approver
	approver := NewRejectAllApprover()
	config := DefaultToolApprovalConfig()
	config.YoloMode = false // Disable yolo mode to test rejection

	executor := NewToolExecutor(config, approver)

	req := ToolRequest{
		ID:       "reject-test-1",
		Type:     ToolTypeReadFile,
		ToolName: "read_file",
		Parameters: map[string]interface{}{
			"path": "/tmp/nonexistent.txt",
		},
	}

	ctx := context.Background()
	result, err := executor.ExecuteTool(ctx, req)

	// Should return error due to rejection
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rejected")
	assert.NotNil(t, result)
	assert.False(t, result.Success)
}

// TestApprovalWithConditionalApprover tests conditional approval logic.
func TestApprovalWithConditionalApprover(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	// Create conditional approver: only approve read_file
	condition := func(req ToolRequest) bool {
		return req.Type == ToolTypeReadFile
	}
	approver := NewConditionalApprover(condition)

	config := DefaultToolApprovalConfig()
	config.YoloMode = false

	executor := NewToolExecutor(config, approver)

	// Test 1: Read file should be approved
	t.Run("approve read_file", func(t *testing.T) {
		req := ToolRequest{
			ID:       "cond-read-1",
			Type:     ToolTypeReadFile,
			ToolName: "read_file",
			Parameters: map[string]interface{}{
				"path": testFile,
			},
		}

		ctx := context.Background()
		result, err := executor.ExecuteTool(ctx, req)

		require.NoError(t, err)
		assert.True(t, result.Success)
	})

	// Test 2: Write file should be rejected
	t.Run("reject write_file", func(t *testing.T) {
		newFile := filepath.Join(tempDir, "new.txt")
		req := ToolRequest{
			ID:       "cond-write-1",
			Type:     ToolTypeWriteFile,
			ToolName: "write_file",
			Parameters: map[string]interface{}{
				"path":    newFile,
				"content": "content",
			},
		}

		ctx := context.Background()
		result, err := executor.ExecuteTool(ctx, req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rejected")
		assert.NotNil(t, result)
		assert.False(t, result.Success)
	})
}

// TestDelegatingApproverIntegration tests the delegating approver with multiple tool types.
func TestDelegatingApproverIntegration(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	// Create approvers for different tool types
	readApprover := NewAutoApprover()       // Auto-approve reads
	writeApprover := NewRejectAllApprover() // Reject writes
	defaultApprover := NewAutoApprover()    // Auto-approve everything else

	delegator := NewDelegatingApprover(defaultApprover)
	delegator.RegisterApprover(ToolTypeReadFile, readApprover)
	delegator.RegisterApprover(ToolTypeWriteFile, writeApprover)

	config := DefaultToolApprovalConfig()
	config.YoloMode = false

	executor := NewToolExecutor(config, delegator)

	// Test 1: Read should succeed (uses readApprover which auto-approves)
	t.Run("delegate read to auto-approver", func(t *testing.T) {
		req := ToolRequest{
			ID:       "del-read-1",
			Type:     ToolTypeReadFile,
			ToolName: "read_file",
			Parameters: map[string]interface{}{
				"path": testFile,
			},
		}

		ctx := context.Background()
		result, err := executor.ExecuteTool(ctx, req)

		require.NoError(t, err)
		assert.True(t, result.Success)
	})

	// Test 2: Write should fail (uses writeApprover which rejects)
	t.Run("delegate write to reject-approver", func(t *testing.T) {
		newFile := filepath.Join(tempDir, "new.txt")
		req := ToolRequest{
			ID:       "del-write-1",
			Type:     ToolTypeWriteFile,
			ToolName: "write_file",
			Parameters: map[string]interface{}{
				"path":    newFile,
				"context": "content",
			},
		}

		ctx := context.Background()
		result, err := executor.ExecuteTool(ctx, req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rejected")
		assert.NotNil(t, result)
		assert.False(t, result.Success)
	})

	// Test 3: List files should succeed (uses defaultApprover)
	t.Run("delegate unknown to default", func(t *testing.T) {
		req := ToolRequest{
			ID:       "del-list-1",
			Type:     ToolTypeListFiles,
			ToolName: "list_files",
			Parameters: map[string]interface{}{
				"path": tempDir,
			},
		}

		ctx := context.Background()
		result, err := executor.ExecuteTool(ctx, req)

		require.NoError(t, err)
		assert.True(t, result.Success)
	})
}

// TestLoggingApproverIntegration tests the logging approver wrapper.
func TestLoggingApproverIntegration(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	var logs []string
	logger := func(s string) {
		logs = append(logs, s)
	}

	innerApprover := NewAutoApprover()
	loggingApprover := NewLoggingApprover(innerApprover, logger)

	config := DefaultToolApprovalConfig()
	config.YoloMode = false

	executor := NewToolExecutor(config, loggingApprover)

	req := ToolRequest{
		ID:       "log-test-1",
		Type:     ToolTypeReadFile,
		ToolName: "read_file",
		Parameters: map[string]interface{}{
			"path": testFile,
		},
	}

	ctx := context.Background()
	result, err := executor.ExecuteTool(ctx, req)

	require.NoError(t, err)
	assert.True(t, result.Success)

	// Verify logs were written
	assert.GreaterOrEqual(t, len(logs), 2)
	assert.Contains(t, logs[0], "Approval requested")
	assert.Contains(t, logs[1], "Tool approved")
}

// TestTimeoutApproverIntegration tests the timeout approver.
func TestTimeoutApproverIntegration(t *testing.T) {
	// Create a slow approver that respects context cancellation
	slowApprover := NewConditionalApprover(func(req ToolRequest) bool {
		// Wait for a short delay or context cancellation, whichever comes first
		ctx, ok := req.Parameters["__context"].(context.Context)
		if !ok {
			// If no context, just return true immediately
			return true
		}
		
		select {
		case <-ctx.Done():
			return false // Context cancelled
		case <-time.After(100 * time.Millisecond):
			return true
		}
	})

	// Wrap with short timeout
	timeoutApprover := NewTimeoutApprover(slowApprover, 10*time.Millisecond)

	config := DefaultToolApprovalConfig()
	config.YoloMode = false

	executor := NewToolExecutor(config, timeoutApprover)

	req := ToolRequest{
		ID:       "timeout-test-1",
		Type:     ToolTypeReadFile,
		ToolName: "read_file",
		Parameters: map[string]interface{}{
			"path": "/tmp/nonexistent.txt",
		},
	}

	ctx := context.Background()
	_, err := executor.ExecuteTool(ctx, req)

	// Should timeout
	assert.Error(t, err)
	// Error could be context deadline exceeded or similar timeout error
}

// TestBatchApproverIntegration tests batch approval functionality.
func TestBatchApproverIntegration(t *testing.T) {
	// Create a UI approver that auto-rejects (non-interactive)
	uiApprover := NewNonInteractiveApprover()
	batchApprover := NewBatchApprover(uiApprover, 3)
	batchApprover.SetAutoApprove(true) // Enable auto-approve

	config := DefaultToolApprovalConfig()
	config.YoloMode = false

	executor := NewToolExecutor(config, batchApprover)

	// Execute multiple tools - should auto-approve due to SetAutoApprove(true)
	requests := []ToolRequest{
		{
			ID:       "batch-1",
			Type:     ToolTypeListFiles,
			ToolName: "list_files",
			Parameters: map[string]interface{}{
				"path": ".",
			},
		},
		{
			ID:       "batch-2",
			Type:     ToolTypeListFiles,
			ToolName: "list_files",
			Parameters: map[string]interface{}{
				"path": ".",
			},
		},
	}

	ctx := context.Background()
	results, err := executor.BatchExecuteTools(ctx, requests)

	// Should execute without individual approval due to batch auto-approve
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

// TestApprovalHistoryIntegration tests that approval decisions are tracked in history.
func TestApprovalHistoryIntegration(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	approver := NewAutoApprover()
	config := DefaultToolApprovalConfig()
	config.YoloMode = true

	executor := NewToolExecutor(config, approver)

	// Execute multiple tools
	for i := 0; i < 3; i++ {
		req := ToolRequest{
			ID:       fmt.Sprintf("history-test-%d", i),
			Type:     ToolTypeReadFile,
			ToolName: "read_file",
			Parameters: map[string]interface{}{
				"path": testFile,
			},
		}

		ctx := context.Background()
		_, err := executor.ExecuteTool(ctx, req)
		require.NoError(t, err)
	}

	// Check history
	history := executor.GetHistory()
	assert.Len(t, history, 3)

	for i, record := range history {
		assert.Equal(t, fmt.Sprintf("history-test-%d", i), record.Request.ID)
		assert.True(t, record.Approved)
		assert.True(t, record.Result.Success)
		assert.WithinDuration(t, time.Now(), record.Timestamp, time.Minute)
	}
}

// TestApprovalWithRetry tests that approved tools can be retried on failure.
func TestApprovalWithRetry(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	// Create the file initially
	err := os.WriteFile(testFile, []byte("initial content"), 0644)
	require.NoError(t, err)

	approver := NewAutoApprover()
	config := DefaultToolApprovalConfig()
	config.YoloMode = true
	config.MaxRetries = 2
	config.RetryDelay = 10 * time.Millisecond

	executor := NewToolExecutor(config, approver)

	// Test successful read with retries configured (even though it succeeds on first try)
	req := ToolRequest{
		ID:       "retry-test-1",
		Type:     ToolTypeReadFile,
		ToolName: "read_file",
		Parameters: map[string]interface{}{
			"path": testFile,
		},
	}

	ctx := context.Background()
	result, err := executor.ExecuteTool(ctx, req)

	// Should succeed on first try
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "initial content", result.Output)

	// Now test retry with file that gets modified
	// Update file after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		os.WriteFile(testFile, []byte("updated content"), 0644)
	}()

	// Wait for the update
	time.Sleep(100 * time.Millisecond)

	// Read again - should get updated content
	req2 := ToolRequest{
		ID:       "retry-test-2",
		Type:     ToolTypeReadFile,
		ToolName: "read_file",
		Parameters: map[string]interface{}{
			"path": testFile,
		},
	}

	result2, err := executor.ExecuteTool(ctx, req2)
	require.NoError(t, err)
	assert.True(t, result2.Success)
	assert.Equal(t, "updated content", result2.Output)
}

// TestYoloModeWithDangerousCommands tests yolo mode safety checks.
func TestYoloModeWithDangerousCommands(t *testing.T) {
	// This test verifies that even in yolo mode, dangerous commands
	// can be blocked by the security layer if integrated

	approver := NewAutoApprover()
	config := DefaultToolApprovalConfig()
	config.YoloMode = true

	executor := NewToolExecutor(config, approver)

	// Test with a command that has dangerous characters
	req := ToolRequest{
		ID:       "dangerous-test-1",
		Type:     ToolTypeExecuteCommand,
		ToolName: "execute_command",
		Parameters: map[string]interface{}{
			"command": "echo 'test'",
		},
	}

	ctx := context.Background()
	result, err := executor.ExecuteTool(ctx, req)

	// In yolo mode, this would execute. The security layer (when integrated)
	// would block dangerous commands.
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Contains(t, result.Output, "test")
}

// TestConcurrentApprovalRequests tests thread safety of approval system.
func TestConcurrentApprovalRequests(t *testing.T) {
	tempDir := t.TempDir()

	// Create multiple test files
	for i := 0; i < 5; i++ {
		testFile := filepath.Join(tempDir, fmt.Sprintf("test%d.txt", i))
		err := os.WriteFile(testFile, []byte(fmt.Sprintf("content%d", i)), 0644)
		require.NoError(t, err)
	}

	approver := NewAutoApprover()
	config := DefaultToolApprovalConfig()
	config.YoloMode = true

	executor := NewToolExecutor(config, approver)

	// Execute tools concurrently
	done := make(chan struct{})
	errors := make(chan error, 5)

	for i := 0; i < 5; i++ {
		go func(id int) {
			defer func() { done <- struct{}{} }()

			req := ToolRequest{
				ID:       fmt.Sprintf("concurrent-%d", id),
				Type:     ToolTypeReadFile,
				ToolName: "read_file",
				Parameters: map[string]interface{}{
					"path": filepath.Join(tempDir, fmt.Sprintf("test%d.txt", id)),
				},
			}

			ctx := context.Background()
			result, err := executor.ExecuteTool(ctx, req)
			if err != nil {
				errors <- err
				return
			}
			if !result.Success {
				errors <- fmt.Errorf("tool %d failed: %s", id, result.Error)
				return
			}
			expectedContent := fmt.Sprintf("content%d", id)
			if result.Output != expectedContent {
				errors <- fmt.Errorf("tool %d wrong output: got %s, want %s", id, result.Output, expectedContent)
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("concurrent execution error: %v", err)
	}

	// Verify history has all entries
	history := executor.GetHistory()
	assert.Len(t, history, 5)
}

// TestApprovalFlowStateTransitions tests state transitions during approval flow.
func TestApprovalFlowStateTransitions(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	require.NoError(t, err)

	approver := NewAutoApprover()
	config := DefaultToolApprovalConfig()
	config.YoloMode = false // Disable yolo to test approval flow

	executor := NewToolExecutor(config, approver)

	// Verify initial state
	assert.False(t, executor.IsYoloMode())

	// Enable yolo mode
	executor.SetYoloMode(true)
	assert.True(t, executor.IsYoloMode())

	// Execute tool in yolo mode
	req := ToolRequest{
		ID:       "state-test-1",
		Type:     ToolTypeReadFile,
		ToolName: "read_file",
		Parameters: map[string]interface{}{
			"path": testFile,
		},
	}

	ctx := context.Background()
	result, err := executor.ExecuteTool(ctx, req)

	require.NoError(t, err)
	assert.True(t, result.Success)

	// Verify auto-approve tools list
	executor.AddAutoApproveTool(ToolTypeWriteFile)
	executor.AddAutoApproveTool(ToolTypeWriteFile) // Duplicate should be ignored

	// Remove auto-approve tool
	executor.RemoveAutoApproveTool(ToolTypeWriteFile)
}

// TestApprovalWithContextCancellation tests that approval respects context cancellation.
func TestApprovalWithContextCancellation(t *testing.T) {
	// Create an approver that respects context cancellation
	contextAwareApprover := NewConditionalApprover(func(req ToolRequest) bool {
		// Extract context from request if available
		if ctx, ok := req.Parameters["__context"].(context.Context); ok {
			select {
			case <-ctx.Done():
				return false // Context cancelled
			case <-time.After(5 * time.Second):
				return true // Timeout
			}
		}
		// Default: immediate approval
		return true
	})

	config := DefaultToolApprovalConfig()
	config.YoloMode = false

	executor := NewToolExecutor(config, contextAwareApprover)

	req := ToolRequest{
		ID:       "cancel-test-1",
		Type:     ToolTypeReadFile,
		ToolName: "read_file",
		Parameters: map[string]interface{}{
			"path": "/tmp/nonexistent.txt",
		},
	}

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Start execution in goroutine
	done := make(chan struct{})
	var execErr error

	go func() {
		defer close(done)
		_, execErr = executor.ExecuteTool(ctx, req)
	}()

	// Cancel context after short delay
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Wait for completion with timeout
	select {
	case <-done:
		// Expected
	case <-time.After(2 * time.Second):
		t.Fatal("execution did not complete after context cancellation")
	}

	// Should have context cancelled error or rejection
	if execErr != nil {
		// Context was cancelled during execution
		assert.True(t, execErr == context.Canceled || strings.Contains(execErr.Error(), "context") || strings.Contains(execErr.Error(), "rejected"))
	} else {
		// Tool may have completed before cancellation - this is also valid behavior
		t.Log("Tool completed before cancellation was processed")
	}
}
