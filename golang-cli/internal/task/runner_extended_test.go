// Package task provides task execution functionality for the Cline CLI.
// This file contains extended tests for Phase 1 coverage improvements.
package task

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunnerMessageHandler tests message handler integration
func TestRunnerMessageHandler(t *testing.T) {
	config := &Config{Mode: TaskModeAct}
	telemetrySvc := createTestTelemetry()
	errorSvc := createTestErrorService()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	runner := NewRunner(config, telemetrySvc, errorSvc, logger)

	t.Run("sets and gets message handler", func(t *testing.T) {
		handler := NewDefaultMessageHandler()
		
		// Set handler
		runner.SetMessageHandler(handler)
		
		// Get handler
		retrieved := runner.GetMessageHandler()
		assert.Equal(t, handler, retrieved)
	})

	t.Run("handles nil handler gracefully", func(t *testing.T) {
		// Setting nil should not panic
		runner.SetMessageHandler(nil)
		
		retrieved := runner.GetMessageHandler()
		assert.Nil(t, retrieved)
	})
}

// TestRunnerApprovalChannels tests approval channel functionality
func TestRunnerApprovalChannels(t *testing.T) {
	config := &Config{Mode: TaskModeAct}
	telemetrySvc := createTestTelemetry()
	errorSvc := createTestErrorService()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	runner := NewRunner(config, telemetrySvc, errorSvc, logger)

	t.Run("returns approval channel", func(t *testing.T) {
		ch := runner.GetApprovalChannel()
		assert.NotNil(t, ch)
	})

	t.Run("sends approval response", func(t *testing.T) {
		response := &ApprovalResponse{
			Approved: true,
		}

		err := runner.SendApprovalResponse(response)
		assert.NoError(t, err)
	})

	t.Run("returns error when channel full", func(t *testing.T) {
		// Fill the channel
		response := &ApprovalResponse{Approved: true}
		_ = runner.SendApprovalResponse(response)
		
		// Second send should fail (channel buffer is 1)
		err := runner.SendApprovalResponse(response)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "response channel full")
	})
}

// TestRunnerRunMethod tests the Run method
func TestRunnerRunMethod(t *testing.T) {
	t.Run("run with message handler", func(t *testing.T) {
		config := &Config{Mode: TaskModeAct}
		telemetrySvc := createTestTelemetry()
		errorSvc := createTestErrorService()
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		runner := NewRunner(config, telemetrySvc, errorSvc, logger)

		handler := NewDefaultMessageHandler()
		taskConfig := TaskConfig{
			Prompt: "Test prompt",
		}

		// Run should execute successfully
		ctx := context.Background()
		err := runner.Run(ctx, taskConfig, handler)
		
		// Should complete without error (even though it's a mock)
		assert.NoError(t, err)
	})

	t.Run("run generates task ID if empty", func(t *testing.T) {
		config := &Config{Mode: TaskModeAct}
		telemetrySvc := createTestTelemetry()
		errorSvc := createTestErrorService()
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		runner := NewRunner(config, telemetrySvc, errorSvc, logger)

		// Clear task ID
		runner.taskID = ""

		handler := NewDefaultMessageHandler()
		taskConfig := TaskConfig{
			Prompt: "Test prompt",
		}

		ctx := context.Background()
		_ = runner.Run(ctx, taskConfig, handler)

		// Task ID should be generated
		assert.NotEmpty(t, runner.taskID)
	})
}

// TestRunnerStreamingExecution tests streaming task execution
func TestRunnerStreamingExecution(t *testing.T) {
	t.Run("run with streaming in yolo mode", func(t *testing.T) {
		config := &Config{Mode: TaskModeAct, Yolo: true}
		telemetrySvc := createTestTelemetry()
		errorSvc := createTestErrorService()
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		runner := NewRunner(config, telemetrySvc, errorSvc, logger)

		ctx := context.Background()
		exitCode, output, err := runner.RunWithStreaming(ctx, "Test prompt", true)

		// Should complete (mock implementation returns success)
		assert.NoError(t, err)
		assert.Equal(t, 0, exitCode)
		assert.NotNil(t, output)
	})

	t.Run("run streaming with auto-approve tools", func(t *testing.T) {
		config := &Config{Mode: TaskModeAct}
		telemetrySvc := createTestTelemetry()
		errorSvc := createTestErrorService()
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		runner := NewRunner(config, telemetrySvc, errorSvc, logger)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		exitCode, _, err := runner.RunWithStreaming(ctx, "Test prompt", true)

		// Should complete
		assert.NoError(t, err)
		assert.Equal(t, 0, exitCode)
	})

	t.Run("fails when task already running", func(t *testing.T) {
		config := &Config{Mode: TaskModeAct}
		telemetrySvc := createTestTelemetry()
		errorSvc := createTestErrorService()
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		runner := NewRunner(config, telemetrySvc, errorSvc, logger)

		// Manually set running state
		runner.mu.Lock()
		runner.isRunning = true
		runner.mu.Unlock()

		ctx := context.Background()
		exitCode, _, err := runner.RunWithStreaming(ctx, "Test prompt", false)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "task already running")
		assert.Equal(t, 1, exitCode)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		config := &Config{Mode: TaskModeAct}
		telemetrySvc := createTestTelemetry()
		errorSvc := createTestErrorService()
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		runner := NewRunner(config, telemetrySvc, errorSvc, logger)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, _, err := runner.RunWithStreaming(ctx, "Test prompt", false)
		
		// Context cancelled error is expected
		assert.Error(t, err)
	})
}

// TestRunnerYoloModeExecution tests yolo mode execution
func TestRunnerYoloModeExecution(t *testing.T) {
	t.Run("execute yolo mode with exit on completion", func(t *testing.T) {
		config := &Config{Mode: TaskModeAct}
		telemetrySvc := createTestTelemetry()
		errorSvc := createTestErrorService()
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		runner := NewRunner(config, telemetrySvc, errorSvc, logger)

		yoloConfig := YoloModeConfig{
			AutoApprove:      true,
			ExitOnCompletion: true,
			PlainText:        true,
			CaptureOutput:    true,
		}

		ctx := context.Background()
		exitCode, output, err := runner.ExecuteYoloMode(ctx, "Test prompt", yoloConfig)

		// Should show yolo warning and execute
		assert.NoError(t, err)
		assert.Equal(t, 0, exitCode)
		assert.NotNil(t, output)
	})

	t.Run("yolo mode sets config flags", func(t *testing.T) {
		config := &Config{Mode: TaskModeAct}
		telemetrySvc := createTestTelemetry()
		errorSvc := createTestErrorService()
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		runner := NewRunner(config, telemetrySvc, errorSvc, logger)

		yoloConfig := YoloModeConfig{
			AutoApprove:   true,
			PlainText:     true,
		}

		_, _, _ = runner.ExecuteYoloMode(context.Background(), "Test prompt", yoloConfig)

		// Config should be updated
		assert.True(t, runner.config.Yolo)
		assert.True(t, runner.config.PlainTextMode)
	})
}

// TestRunnerPauseState tests pause state management
func TestRunnerPauseState(t *testing.T) {
	config := &Config{Mode: TaskModeAct}
	telemetrySvc := createTestTelemetry()
	errorSvc := createTestErrorService()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	runner := NewRunner(config, telemetrySvc, errorSvc, logger)

	t.Run("not paused by default", func(t *testing.T) {
		assert.False(t, runner.IsPaused())
	})

	t.Run("pauses during approval request", func(t *testing.T) {
		// This would normally block, but with yolo mode it completes immediately
		yoloConfig := &Config{Mode: TaskModeAct, Yolo: true}
		runner2 := NewRunner(yoloConfig, telemetrySvc, errorSvc, logger)

		_, _ = runner2.RequestApprovalWithCallback(
			"test_tool",
			"Test",
			nil,
			"",
			nil, // No callback - uses approval handler
		)

		// After completion, should not be paused
		assert.False(t, runner2.IsPaused())
	})
}

// TestRunnerExitCodeAndCompletion tests exit code and completion tracking
func TestRunnerExitCodeAndCompletion(t *testing.T) {
	config := &Config{Mode: TaskModeAct}
	telemetrySvc := createTestTelemetry()
	errorSvc := createTestErrorService()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	runner := NewRunner(config, telemetrySvc, errorSvc, logger)

	t.Run("default exit code is 0", func(t *testing.T) {
		assert.Equal(t, 0, runner.GetExitCode())
	})

	t.Run("default completion is false", func(t *testing.T) {
		assert.False(t, runner.IsCompleted())
	})

	t.Run("default result is empty", func(t *testing.T) {
		assert.Empty(t, runner.GetTaskResult())
	})
}

// TestRunnerGRPCConnection tests gRPC connection methods
func TestRunnerGRPCConnection(t *testing.T) {
	t.Run("connect and disconnect gRPC", func(t *testing.T) {
		config := &Config{Mode: TaskModeAct}
		telemetrySvc := createTestTelemetry()
		errorSvc := createTestErrorService()
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		runner := NewRunner(config, telemetrySvc, errorSvc, logger)

		// Connect to invalid target should fail
		err := runner.ConnectGRPC("invalid:target")
		assert.Error(t, err)

		// Disconnect with no client should not panic
		err = runner.DisconnectGRPC()
		assert.NoError(t, err)
	})
}

// TestRunnerRequestApprovalWithCallback tests callback-based approval
func TestRunnerRequestApprovalWithCallback(t *testing.T) {
	t.Run("uses callback when provided", func(t *testing.T) {
		config := &Config{Mode: TaskModeAct}
		telemetrySvc := createTestTelemetry()
		errorSvc := createTestErrorService()
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		runner := NewRunner(config, telemetrySvc, errorSvc, logger)

		callbackCalled := false
		callback := func(request *ApprovalRequest) (*ApprovalResponse, error) {
			callbackCalled = true
			return &ApprovalResponse{Approved: true}, nil
		}

		response, err := runner.RequestApprovalWithCallback(
			"test_tool",
			"Test description",
			map[string]string{"param": "value"},
			"diff content",
			callback,
		)

		require.NoError(t, err)
		assert.True(t, callbackCalled)
		assert.True(t, response.Approved)
	})

	t.Run("callback receives request details", func(t *testing.T) {
		config := &Config{Mode: TaskModeAct}
		telemetrySvc := createTestTelemetry()
		errorSvc := createTestErrorService()
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		runner := NewRunner(config, telemetrySvc, errorSvc, logger)

		var receivedRequest *ApprovalRequest
		callback := func(request *ApprovalRequest) (*ApprovalResponse, error) {
			receivedRequest = request
			return &ApprovalResponse{Approved: false}, nil
		}

		_, _ = runner.RequestApprovalWithCallback(
			"write_file",
			"Write to test.txt",
			map[string]string{"file": "test.txt"},
			"diff here",
			callback,
		)

		require.NotNil(t, receivedRequest)
		assert.Equal(t, "write_file", receivedRequest.ToolName)
		assert.Equal(t, "Write to test.txt", receivedRequest.Description)
		assert.Equal(t, "diff here", receivedRequest.Diff)
	})

	t.Run("handles callback error", func(t *testing.T) {
		config := &Config{Mode: TaskModeAct}
		telemetrySvc := createTestTelemetry()
		errorSvc := createTestErrorService()
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		runner := NewRunner(config, telemetrySvc, errorSvc, logger)

		callback := func(request *ApprovalRequest) (*ApprovalResponse, error) {
			return nil, assert.AnError
		}

		_, err := runner.RequestApprovalWithCallback(
			"test_tool",
			"Test",
			nil,
			"",
			callback,
		)

		assert.Error(t, err)
	})
}

// TestRunnerSessionStats tests session statistics
func TestRunnerSessionStats(t *testing.T) {
	config := &Config{Mode: TaskModeAct}
	telemetrySvc := createTestTelemetry()
	errorSvc := createTestErrorService()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	runner := NewRunner(config, telemetrySvc, errorSvc, logger)

	t.Run("returns session stats", func(t *testing.T) {
		stats := runner.GetSessionStats()
		// Stats should be returned (values depend on session state)
		assert.NotNil(t, stats)
	})
}

// TestRunnerExecuteTask tests the internal executeTask method
func TestRunnerExecuteTask(t *testing.T) {
	t.Run("execute task handles context", func(t *testing.T) {
		config := &Config{Mode: TaskModeAct}
		telemetrySvc := createTestTelemetry()
		errorSvc := createTestErrorService()
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		runner := NewRunner(config, telemetrySvc, errorSvc, logger)

		ctx := context.Background()
		err := runner.executeTask(ctx, "Test prompt")
		
		// Should complete without error in mock implementation
		assert.NoError(t, err)
	})

	t.Run("execute task respects context cancellation", func(t *testing.T) {
		config := &Config{Mode: TaskModeAct}
		telemetrySvc := createTestTelemetry()
		errorSvc := createTestErrorService()
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		runner := NewRunner(config, telemetrySvc, errorSvc, logger)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := runner.executeTask(ctx, "Test prompt")
		
		// Should return context error
		assert.Error(t, err)
	})
}

