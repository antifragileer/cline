// Package task provides task execution functionality for the Cline CLI.
// This file contains tests for the task executor functionality.
package task

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewExecutor tests the creation of a new executor
func TestNewExecutor(t *testing.T) {
	t.Run("creates executor with valid connection", func(t *testing.T) {
		executor := NewExecutor(nil)
		assert.NotNil(t, executor)
		assert.NotNil(t, executor.completionCh)
		assert.NotNil(t, executor.errorCh)
		assert.NotNil(t, executor.sigChan)
	})

	t.Run("executor initializes with correct defaults", func(t *testing.T) {
		executor := NewExecutor(nil)
		assert.False(t, executor.IsRunning())
		assert.False(t, executor.IsCancelled())
		assert.Empty(t, executor.GetTaskID())
	})
}

// TestExecutorValidateConfig tests configuration validation
func TestExecutorValidateConfig(t *testing.T) {
	executor := NewExecutor(nil)

	tests := []struct {
		name       string
		config     TaskConfig
		wantErr    bool
		errContain string
	}{
		{
			name: "valid config with prompt",
			config: TaskConfig{
				Prompt: "Test prompt",
				Mode:   TaskModeAct,
			},
			wantErr: false,
		},
		{
			name: "valid config with task ID (resume)",
			config: TaskConfig{
				TaskID: "existing-task-123",
				Mode:   TaskModeAct,
			},
			wantErr: false,
		},
		{
			name: "invalid - no prompt and no task ID",
			config: TaskConfig{
				Mode: TaskModeAct,
			},
			wantErr:    true,
			errContain: "prompt required",
		},
		{
			name: "invalid mode",
			config: TaskConfig{
				Prompt: "Test",
				Mode:   "invalid_mode",
			},
			wantErr:    true,
			errContain: "invalid mode",
		},
		{
			name: "non-existent image file",
			config: TaskConfig{
				Prompt: "Test",
				Mode:   TaskModeAct,
				Images: []string{"/nonexistent/path.png"},
			},
			wantErr:    true,
			errContain: "image file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.validateConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestExecutorValidateConfigWithExistingImage tests image validation with real file
func TestExecutorValidateConfigWithExistingImage(t *testing.T) {
	tempDir := t.TempDir()
	testImage := filepath.Join(tempDir, "test.png")
	err := os.WriteFile(testImage, []byte("fake png content"), 0644)
	require.NoError(t, err)

	executor := NewExecutor(nil)

	config := TaskConfig{
		Prompt: "Test with image",
		Mode:   TaskModeAct,
		Images: []string{testImage},
	}

	err = executor.validateConfig(config)
	assert.NoError(t, err)
}

// TestExecutorPrepareImages tests image preparation
func TestExecutorPrepareImages(t *testing.T) {
	tempDir := t.TempDir()
	testImage := filepath.Join(tempDir, "test.png")

	executor := NewExecutor(nil)

	t.Run("prepare empty images", func(t *testing.T) {
		result, err := executor.prepareImages([]string{})
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("prepare with non-existent file", func(t *testing.T) {
		_, err := executor.prepareImages([]string{"/nonexistent.png"})
		assert.Error(t, err)
	})

	t.Run("prepare with valid file", func(t *testing.T) {
		err := os.WriteFile(testImage, []byte("test content"), 0644)
		require.NoError(t, err)

		result, err := executor.prepareImages([]string{testImage})
		assert.NoError(t, err)
		assert.Len(t, result, 1)
	})
}

// TestExecutorIsRunning tests the IsRunning method
func TestExecutorIsRunning(t *testing.T) {
	executor := NewExecutor(nil)

	t.Run("not running initially", func(t *testing.T) {
		assert.False(t, executor.IsRunning())
	})

	t.Run("running when set", func(t *testing.T) {
		executor.isRunning = true
		assert.True(t, executor.IsRunning())
		executor.isRunning = false
	})
}

// TestExecutorIsCancelled tests the IsCancelled method
func TestExecutorIsCancelled(t *testing.T) {
	executor := NewExecutor(nil)

	t.Run("not cancelled initially", func(t *testing.T) {
		assert.False(t, executor.IsCancelled())
	})

	t.Run("cancelled when set", func(t *testing.T) {
		executor.isCancelled = true
		assert.True(t, executor.IsCancelled())
		executor.isCancelled = false
	})
}

// TestExecutorGetTaskID tests the GetTaskID method
func TestExecutorGetTaskID(t *testing.T) {
	executor := NewExecutor(nil)

	t.Run("empty initially", func(t *testing.T) {
		assert.Empty(t, executor.GetTaskID())
	})

	t.Run("returns set ID", func(t *testing.T) {
		executor.taskID = "test-id-123"
		assert.Equal(t, "test-id-123", executor.GetTaskID())
		executor.taskID = ""
	})
}

// TestExecutorCancel tests cancellation
func TestExecutorCancel(t *testing.T) {
	executor := NewExecutor(nil)

	t.Run("cancel when not running", func(t *testing.T) {
		// Should not panic
		executor.Cancel()
		assert.False(t, executor.IsCancelled())
	})

	t.Run("cancel when running", func(t *testing.T) {
		executor.isRunning = true
		executor.isCancelled = false

		cancelCalled := false
		executor.cancelFunc = func() {
			cancelCalled = true
		}

		executor.Cancel()

		assert.True(t, executor.IsCancelled())
		assert.True(t, cancelCalled)

		// Reset
		executor.isRunning = false
		executor.isCancelled = false
		executor.cancelFunc = nil
	})

	t.Run("double cancel is safe", func(t *testing.T) {
		executor.isRunning = true
		executor.isCancelled = false

		executor.Cancel()
		// Second cancel should be safe
		executor.Cancel()

		// Reset
		executor.isRunning = false
		executor.isCancelled = false
	})
}

// TestExecutorClose tests closing the executor
func TestExecutorClose(t *testing.T) {
	executor := NewExecutor(nil)

	t.Run("close returns nil", func(t *testing.T) {
		err := executor.Close()
		assert.NoError(t, err)
	})
}

// TestExecutorBuildSettingsFromConfig tests building settings
func TestExecutorBuildSettingsFromConfig(t *testing.T) {
	t.Run("act mode settings", func(t *testing.T) {
		config := TaskConfig{
			Mode: TaskModeAct,
		}
		settings := buildSettingsFromConfig(config)
		assert.NotNil(t, settings)
		assert.NotNil(t, settings.Mode)
	})

	t.Run("plan mode settings", func(t *testing.T) {
		config := TaskConfig{
			Mode: TaskModePlan,
		}
		settings := buildSettingsFromConfig(config)
		assert.NotNil(t, settings)
		assert.NotNil(t, settings.Mode)
	})
}

// TestExecutorShouldAutoApprove tests auto-approval logic
func TestExecutorShouldAutoApprove(t *testing.T) {
	tests := []struct {
		name     string
		askType  string
		yolo     bool
		expected bool
	}{
		{
			name:     "yolo mode approves all",
			askType:  "command",
			yolo:     true,
			expected: true,
		},
		{
			name:     "auto-approve api_req_started",
			askType:  "api_req_started",
			yolo:     false,
			expected: true,
		},
		{
			name:     "auto-approve checkpoint_created",
			askType:  "checkpoint_created",
			yolo:     false,
			expected: true,
		},
		{
			name:     "no auto-approve for command",
			askType:  "command",
			yolo:     false,
			expected: false,
		},
		{
			name:     "no auto-approve for tool",
			askType:  "tool",
			yolo:     false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := TaskConfig{
				Yolo: tt.yolo,
			}
			result := shouldAutoApprove(tt.askType, config)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestExecutorCheckCompletionOrCancellation tests completion detection
func TestExecutorCheckCompletionOrCancellation(t *testing.T) {
	executor := NewExecutor(nil)

	t.Run("detects cancellation", func(t *testing.T) {
		executor.isCancelled = true
		msg := &ClineMessage{Type: "say", Say: "text"}
		result := executor.checkCompletionOrCancellation(msg)
		assert.True(t, result)
		executor.isCancelled = false
	})

	t.Run("detects completion say", func(t *testing.T) {
		msg := &ClineMessage{Type: "say", Say: "completion_result"}
		result := executor.checkCompletionOrCancellation(msg)
		assert.True(t, result)
	})

	t.Run("detects completion ask", func(t *testing.T) {
		msg := &ClineMessage{Type: "ask", Ask: "completion_result"}
		result := executor.checkCompletionOrCancellation(msg)
		assert.True(t, result)
	})

	t.Run("continues for normal message", func(t *testing.T) {
		msg := &ClineMessage{Type: "say", Say: "text"}
		result := executor.checkCompletionOrCancellation(msg)
		assert.False(t, result)
	})
}

// TestExecutorProcessSayMessage tests say message processing
func TestExecutorProcessSayMessage(t *testing.T) {
	executor := NewExecutor(nil)
	handler := NewMockMessageHandler()

	t.Run("processes say message", func(t *testing.T) {
		config := TaskConfig{Verbose: true}
		msg := &ClineMessage{
			Type: "say",
			Say:  "text",
			Text: "Hello world",
		}

		err := executor.processSayMessage(msg, config, handler)
		assert.NoError(t, err)
		assert.Len(t, handler.sayCalls, 1)
		assert.Equal(t, "text", handler.sayCalls[0].sayType)
	})

	t.Run("skips partial messages when not verbose", func(t *testing.T) {
		config := TaskConfig{Verbose: false}
		msg := &ClineMessage{
			Type:    "say",
			Say:     "text",
			Text:    "Partial",
			Partial: true,
		}

		err := executor.processSayMessage(msg, config, handler)
		assert.NoError(t, err)
	})

	t.Run("processes partial messages when verbose", func(t *testing.T) {
		config := TaskConfig{Verbose: true}
		msg := &ClineMessage{
			Type:    "say",
			Say:     "text",
			Text:    "Partial",
			Partial: true,
		}

		err := executor.processSayMessage(msg, config, handler)
		assert.NoError(t, err)
	})
}

// TestExecutorSendAskResponse tests sending ask responses
func TestExecutorSendAskResponse(t *testing.T) {
	t.Run("returns error with nil client", func(t *testing.T) {
		// Skipping this test as it requires a real gRPC client
		// The method will panic with nil client, which is expected behavior
		t.Skip("Requires real gRPC client connection")
	})
}

// TestExecutorSendAutoApproval tests sending auto-approval
func TestExecutorSendAutoApproval(t *testing.T) {
	t.Run("returns error with nil client", func(t *testing.T) {
		// Skipping this test as it requires a real gRPC client
		// The method will panic with nil client, which is expected behavior
		t.Skip("Requires real gRPC client connection")
	})
}

// TestExecutorProcessMessage tests message processing
func TestExecutorProcessMessage(t *testing.T) {
	executor := NewExecutor(nil)
	handler := NewMockMessageHandler()

	t.Run("processes say message", func(t *testing.T) {
		config := TaskConfig{Verbose: true}
		msg := &ClineMessage{
			Type: "say",
			Say:  "text",
			Text: "Hello",
		}

		err := executor.processMessage(msg, config, handler)
		assert.NoError(t, err)
	})

	t.Run("processes unknown type gracefully", func(t *testing.T) {
		config := TaskConfig{}
		msg := &ClineMessage{
			Type: "unknown",
		}

		err := executor.processMessage(msg, config, handler)
		assert.NoError(t, err)
	})
}

// TestExecutorExecute tests the Execute method
func TestExecutorExecute(t *testing.T) {
	t.Run("requires real gRPC client", func(t *testing.T) {
		// Skipping this test as it requires a real gRPC client
		// The method will panic with nil client, which is expected behavior
		t.Skip("Requires real gRPC client connection")
	})
}

// TestExecutorConcurrencySafety tests thread safety
func TestExecutorConcurrencySafety(t *testing.T) {
	executor := NewExecutor(nil)

	t.Run("concurrent status checks", func(t *testing.T) {
		done := make(chan bool, 20)
		for i := 0; i < 10; i++ {
			go func() {
				_ = executor.IsRunning()
				done <- true
			}()
			go func() {
				_ = executor.IsCancelled()
				done <- true
			}()
		}
		for i := 0; i < 20; i++ {
			<-done
		}
	})

	t.Run("concurrent GetTaskID", func(t *testing.T) {
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				_ = executor.GetTaskID()
				done <- true
			}()
		}
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

// TestGetThinkingBudget tests the thinking budget helper
func TestGetThinkingBudget(t *testing.T) {
	t.Run("thinking enabled", func(t *testing.T) {
		config := TaskConfig{Thinking: true}
		budget := getThinkingBudget(config)
		assert.Equal(t, int32(0), budget)
	})

	t.Run("thinking disabled", func(t *testing.T) {
		config := TaskConfig{Thinking: false}
		budget := getThinkingBudget(config)
		assert.Equal(t, int32(-1), budget)
	})
}

// BenchmarkExecutorOperations benchmarks executor operations
func BenchmarkExecutorOperations(b *testing.B) {
	executor := NewExecutor(nil)

	b.Run("IsRunning", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = executor.IsRunning()
		}
	})

	b.Run("IsCancelled", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = executor.IsCancelled()
		}
	})

	b.Run("GetTaskID", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = executor.GetTaskID()
		}
	})

	b.Run("Cancel", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			executor.Cancel()
		}
	})
}
