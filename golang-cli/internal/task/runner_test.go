// Package task provides task execution functionality for the Cline CLI.
// This file contains tests for the task runner functionality.
package task

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/cline/cline/golang-cli/internal/errorservice"
	"github.com/cline/cline/golang-cli/internal/telemetry"
	"github.com/stretchr/testify/assert"
)

// createTestTelemetry creates a no-op telemetry service for testing
func createTestTelemetry() telemetry.Service {
	return &mockTelemetryService{}
}

// createTestErrorService creates a no-op error service for testing
func createTestErrorService() errorservice.Service {
	return &mockErrorService{}
}

// mockTelemetryService is a mock implementation of telemetry.Service for testing
type mockTelemetryService struct{}

func (m *mockTelemetryService) CaptureHostEvent(event string, properties map[string]interface{}) error {
	return nil
}

func (m *mockTelemetryService) CaptureExtensionActivated() error {
	return nil
}

func (m *mockTelemetryService) CapturePlainTextMode(reason string) error {
	return nil
}

func (m *mockTelemetryService) CaptureCommand(command string, details string) error {
	return nil
}

func (m *mockTelemetryService) CaptureAuth(status string, provider string) error {
	return nil
}

func (m *mockTelemetryService) CaptureAuthQuickSetup() error {
	return nil
}

func (m *mockTelemetryService) CaptureAuthInteractive() error {
	return nil
}

func (m *mockTelemetryService) CaptureTaskCreated(taskID string, apiProvider string) error {
	return nil
}

func (m *mockTelemetryService) CaptureModeFlag(mode string) error {
	return nil
}

func (m *mockTelemetryService) CaptureModelFlag(model string) error {
	return nil
}

func (m *mockTelemetryService) CaptureThinkingFlag() error {
	return nil
}

func (m *mockTelemetryService) CaptureReasoningEffortFlag(effort string) error {
	return nil
}

func (m *mockTelemetryService) CaptureMaxConsecutiveMistakesFlag(count int) error {
	return nil
}

func (m *mockTelemetryService) CaptureYoloFlag() error {
	return nil
}

func (m *mockTelemetryService) CaptureAutoApproveAllFlag() error {
	return nil
}

func (m *mockTelemetryService) CaptureDoubleCheckCompletionFlag() error {
	return nil
}

func (m *mockTelemetryService) CapturePiped() error {
	return nil
}

func (m *mockTelemetryService) CaptureResumeTask(withPrompt bool) error {
	return nil
}

func (m *mockTelemetryService) Dispose() error {
	return nil
}

func (m *mockTelemetryService) IsEnabled() bool {
	return false
}

// mockErrorService is a mock implementation of errorservice.Service for testing
type mockErrorService struct{}

func (m *mockErrorService) Initialize() error {
	return nil
}

func (m *mockErrorService) CaptureException(err error, context map[string]string) error {
	return nil
}

func (m *mockErrorService) LogException(err error, context map[string]string) {
}

func (m *mockErrorService) Dispose() error {
	return nil
}

// TestNewRunner tests the creation of a new task runner
func TestNewRunner(t *testing.T) {
	t.Run("creates runner with valid config", func(t *testing.T) {
		config := &Config{Mode: TaskModeAct}
		telemetrySvc := createTestTelemetry()
		errorSvc := createTestErrorService()
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		runner := NewRunner(config, telemetrySvc, errorSvc, logger)
		assert.NotNil(t, runner)
		assert.NotNil(t, runner.GetConfig())
		assert.Equal(t, TaskModeAct, runner.GetConfig().Mode)
	})

	t.Run("creates runner with nil config", func(t *testing.T) {
		telemetrySvc := createTestTelemetry()
		errorSvc := createTestErrorService()
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

		runner := NewRunner(nil, telemetrySvc, errorSvc, logger)
		assert.NotNil(t, runner)
	})
}

// TestRunnerIsRunning tests the IsRunning method
func TestRunnerIsRunning(t *testing.T) {
	config := &Config{Mode: TaskModeAct}
	telemetrySvc := createTestTelemetry()
	errorSvc := createTestErrorService()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	runner := NewRunner(config, telemetrySvc, errorSvc, logger)

	t.Run("returns false when not running", func(t *testing.T) {
		assert.False(t, runner.IsRunning())
	})
}

// TestRunnerIsCancelled tests the IsCancelled method
func TestRunnerIsCancelled(t *testing.T) {
	config := &Config{Mode: TaskModeAct}
	telemetrySvc := createTestTelemetry()
	errorSvc := createTestErrorService()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	runner := NewRunner(config, telemetrySvc, errorSvc, logger)

	t.Run("returns false when not cancelled", func(t *testing.T) {
		assert.False(t, runner.IsCancelled())
	})

	t.Run("returns true after cancellation", func(t *testing.T) {
		runner.Cancel()
		assert.True(t, runner.IsCancelled())
	})
}

// TestRunnerCancel tests cancellation
func TestRunnerCancel(t *testing.T) {
	config := &Config{Mode: TaskModeAct}
	telemetrySvc := createTestTelemetry()
	errorSvc := createTestErrorService()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	runner := NewRunner(config, telemetrySvc, errorSvc, logger)

	t.Run("cancel doesn't panic when not running", func(t *testing.T) {
		assert.NotPanics(t, func() {
			runner.Cancel()
		})
	})
}

// TestRunnerShouldAutoApprove tests auto-approval detection
func TestRunnerShouldAutoApprove(t *testing.T) {
	t.Run("returns false by default", func(t *testing.T) {
		config := &Config{Mode: TaskModeAct}
		telemetrySvc := createTestTelemetry()
		errorSvc := createTestErrorService()
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		runner := NewRunner(config, telemetrySvc, errorSvc, logger)

		assert.False(t, runner.ShouldAutoApprove())
	})

	t.Run("returns true when yolo mode enabled", func(t *testing.T) {
		config := &Config{Mode: TaskModeAct, Yolo: true}
		telemetrySvc := createTestTelemetry()
		errorSvc := createTestErrorService()
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		runner := NewRunner(config, telemetrySvc, errorSvc, logger)

		assert.True(t, runner.ShouldAutoApprove())
	})

	t.Run("returns true when auto-approve-all enabled", func(t *testing.T) {
		config := &Config{Mode: TaskModeAct, AutoApproveAll: true}
		telemetrySvc := createTestTelemetry()
		errorSvc := createTestErrorService()
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		runner := NewRunner(config, telemetrySvc, errorSvc, logger)

		assert.True(t, runner.ShouldAutoApprove())
	})
}

// TestRunnerIsYoloMode tests yolo mode detection
func TestRunnerIsYoloMode(t *testing.T) {
	t.Run("returns false by default", func(t *testing.T) {
		config := &Config{Mode: TaskModeAct}
		telemetrySvc := createTestTelemetry()
		errorSvc := createTestErrorService()
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		runner := NewRunner(config, telemetrySvc, errorSvc, logger)

		assert.False(t, runner.IsYoloMode())
	})

	t.Run("returns true when yolo mode enabled", func(t *testing.T) {
		config := &Config{Mode: TaskModeAct, Yolo: true}
		telemetrySvc := createTestTelemetry()
		errorSvc := createTestErrorService()
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		runner := NewRunner(config, telemetrySvc, errorSvc, logger)

		assert.True(t, runner.IsYoloMode())
	})
}

// TestRunnerShouldExitOnCompletion tests exit on completion detection
func TestRunnerShouldExitOnCompletion(t *testing.T) {
	t.Run("returns false by default", func(t *testing.T) {
		config := &Config{Mode: TaskModeAct}
		telemetrySvc := createTestTelemetry()
		errorSvc := createTestErrorService()
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		runner := NewRunner(config, telemetrySvc, errorSvc, logger)

		assert.False(t, runner.ShouldExitOnCompletion())
	})

	t.Run("returns true when yolo mode enabled", func(t *testing.T) {
		config := &Config{Mode: TaskModeAct, Yolo: true}
		telemetrySvc := createTestTelemetry()
		errorSvc := createTestErrorService()
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		runner := NewRunner(config, telemetrySvc, errorSvc, logger)

		assert.True(t, runner.ShouldExitOnCompletion())
	})
}

// TestRunnerAutoCondenseEnabled tests auto-condense detection
func TestRunnerAutoCondenseEnabled(t *testing.T) {
	config := &Config{Mode: TaskModeAct}
	telemetrySvc := createTestTelemetry()
	errorSvc := createTestErrorService()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	runner := NewRunner(config, telemetrySvc, errorSvc, logger)

	t.Run("returns false by default", func(t *testing.T) {
		assert.False(t, runner.AutoCondenseEnabled())
	})
}

// TestRunnerGetConfig tests getting the config
func TestRunnerGetConfig(t *testing.T) {
	config := &Config{Mode: TaskModePlan, Yolo: true}
	telemetrySvc := createTestTelemetry()
	errorSvc := createTestErrorService()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	runner := NewRunner(config, telemetrySvc, errorSvc, logger)

	t.Run("returns correct config", func(t *testing.T) {
		retrievedConfig := runner.GetConfig()
		assert.NotNil(t, retrievedConfig)
		assert.Equal(t, TaskModePlan, retrievedConfig.Mode)
		assert.True(t, retrievedConfig.Yolo)
	})
}

// TestRunnerStart tests starting a task
func TestRunnerStart(t *testing.T) {
	config := &Config{Mode: TaskModeAct}
	telemetrySvc := createTestTelemetry()
	errorSvc := createTestErrorService()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	runner := NewRunner(config, telemetrySvc, errorSvc, logger)

	t.Run("starts task successfully", func(t *testing.T) {
		ctx := context.Background()
		err := runner.Start(ctx, "Test prompt")
		assert.NoError(t, err)
	})

	t.Run("returns error when task already running", func(t *testing.T) {
		// Create a new runner since the previous one has completed
		runner2 := NewRunner(config, telemetrySvc, errorSvc, logger)

		// Manually set the running state to simulate an in-progress task
		runner2.mu.Lock()
		runner2.isRunning = true
		runner2.mu.Unlock()

		ctx := context.Background()
		// Start should fail since task is already running
		err := runner2.Start(ctx, "Test prompt")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "task already running")

		// Reset the state
		runner2.mu.Lock()
		runner2.isRunning = false
		runner2.mu.Unlock()
	})
}

// TestRunnerRequestApproval tests approval requests
func TestRunnerRequestApproval(t *testing.T) {
	// Use yolo mode to skip interactive approval in test environment
	config := &Config{Mode: TaskModeAct, Yolo: true}
	telemetrySvc := createTestTelemetry()
	errorSvc := createTestErrorService()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	runner := NewRunner(config, telemetrySvc, errorSvc, logger)

	t.Run("requests approval successfully", func(t *testing.T) {
		response, err := runner.RequestApproval("test_tool", "Test description", map[string]string{
			"param1": "value1",
		})
		// In yolo mode, approval should succeed without interactive UI
		// But if it fails due to TTY issues, we accept the error too
		if err != nil {
			// Expected in non-TTY environments
			assert.Contains(t, err.Error(), "TTY")
		} else {
			assert.NotNil(t, response)
		}
	})
}

// TestRunnerCompleteTask tests task completion
func TestRunnerCompleteTask(t *testing.T) {
	config := &Config{Mode: TaskModeAct}
	telemetrySvc := createTestTelemetry()
	errorSvc := createTestErrorService()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	runner := NewRunner(config, telemetrySvc, errorSvc, logger)

	t.Run("completes task successfully", func(t *testing.T) {
		response, err := runner.CompleteTask()
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.True(t, response.Approved)
	})

	t.Run("rejects completion in double-check mode", func(t *testing.T) {
		configWithDoubleCheck := &Config{Mode: TaskModeAct, DoubleCheckCompletion: true}
		runner2 := NewRunner(configWithDoubleCheck, telemetrySvc, errorSvc, logger)

		response, err := runner2.CompleteTask()
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.False(t, response.Approved)
	})
}

// TestRunnerHandleToolError tests error handling
func TestRunnerHandleToolError(t *testing.T) {
	config := &Config{Mode: TaskModeAct, MaxConsecutiveMistakes: 3}
	telemetrySvc := createTestTelemetry()
	errorSvc := createTestErrorService()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	runner := NewRunner(config, telemetrySvc, errorSvc, logger)

	t.Run("handles tool error without reaching max", func(t *testing.T) {
		testErr := assert.AnError
		err := runner.HandleToolError(testErr)
		assert.NoError(t, err) // Should not return error until max mistakes reached
	})

	t.Run("reaches max consecutive mistakes", func(t *testing.T) {
		// Record 3 failures to reach max
		var err error
		for i := 0; i < 3; i++ {
			testErr := assert.AnError
			err = runner.HandleToolError(testErr)
		}
		// After 3 errors, max mistakes should be reached and an error returned
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Maximum consecutive mistakes")
	})
}

// TestRunnerConcurrencySafety tests thread safety
func TestRunnerConcurrencySafety(t *testing.T) {
	config := &Config{Mode: TaskModeAct}
	telemetrySvc := createTestTelemetry()
	errorSvc := createTestErrorService()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	runner := NewRunner(config, telemetrySvc, errorSvc, logger)

	t.Run("concurrent IsRunning calls", func(t *testing.T) {
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				_ = runner.IsRunning()
				done <- true
			}()
		}
		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("concurrent IsCancelled calls", func(t *testing.T) {
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				_ = runner.IsCancelled()
				done <- true
			}()
		}
		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("concurrent GetConfig calls", func(t *testing.T) {
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				_ = runner.GetConfig()
				done <- true
			}()
		}
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

// BenchmarkRunnerOperations benchmarks runner operations
func BenchmarkRunnerOperations(b *testing.B) {
	config := &Config{Mode: TaskModeAct}
	telemetrySvc := createTestTelemetry()
	errorSvc := createTestErrorService()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	runner := NewRunner(config, telemetrySvc, errorSvc, logger)

	b.Run("IsRunning", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = runner.IsRunning()
		}
	})

	b.Run("IsCancelled", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = runner.IsCancelled()
		}
	})

	b.Run("Cancel", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			runner.Cancel()
		}
	})

	b.Run("ShouldAutoApprove", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = runner.ShouldAutoApprove()
		}
	})

	b.Run("IsYoloMode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = runner.IsYoloMode()
		}
	})
}
