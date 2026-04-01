// Package task provides task execution functionality for the Cline CLI.
// This file contains tests for the task runner functionality.
package task

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockMessageHandler is a mock implementation of MessageHandler for testing
type MockMessageHandler struct {
	sayCalls         []SayCall
	askResponses     map[string]string
	askCalls         []AskCall
	infoMessages     []string
	errorMessages    []error
	statusMessages   []string
	progressUpdates  []ProgressUpdate
	checkpointCalls  []CheckpointCall
	toolResults      []ToolResult
	completionCalled bool
	completionResult struct {
		success bool
		summary string
	}
}

type SayCall struct {
	sayType string
	content string
	partial bool
}

type AskCall struct {
	askType  string
	question string
	response string
	err      error
}

type ProgressUpdate struct {
	current int
	total   int
}

type CheckpointCall struct {
	checkpointID string
	action       string
}

func NewMockMessageHandler() *MockMessageHandler {
	return &MockMessageHandler{
		sayCalls:        make([]SayCall, 0),
		askResponses:    make(map[string]string),
		askCalls:        make([]AskCall, 0),
		infoMessages:    make([]string, 0),
		errorMessages:   make([]error, 0),
		statusMessages:  make([]string, 0),
		progressUpdates: make([]ProgressUpdate, 0),
		checkpointCalls: make([]CheckpointCall, 0),
		toolResults:     make([]ToolResult, 0),
	}
}

func (m *MockMessageHandler) SetAskResponse(promptType, response string) {
	m.askResponses[promptType] = response
}

func (m *MockMessageHandler) HandleMessage(msg Message) error {
	return nil
}

func (m *MockMessageHandler) OnText(content string, isPartial bool) error {
	return nil
}

func (m *MockMessageHandler) OnToolUse(toolName string, params map[string]interface{}) (bool, error) {
	return true, nil
}

func (m *MockMessageHandler) OnToolResult(toolName string, result string, success bool) error {
	m.toolResults = append(m.toolResults, ToolResult{
		RequestID: toolName,
		Output:    result,
		Success:   success,
	})
	return nil
}

func (m *MockMessageHandler) OnAsk(promptType string, question string) (string, error) {
	response := m.askResponses[promptType]
	m.askCalls = append(m.askCalls, AskCall{
		askType:  promptType,
		question: question,
		response: response,
	})
	return response, nil
}

func (m *MockMessageHandler) OnSay(sayType string, content string, partial bool) error {
	m.sayCalls = append(m.sayCalls, SayCall{
		sayType: sayType,
		content: content,
		partial: partial,
	})
	return nil
}

func (m *MockMessageHandler) OnCommand(command string, requiresApproval bool) (string, error) {
	return "", nil
}

func (m *MockMessageHandler) OnCommandOutput(output string, isComplete bool) error {
	return nil
}

func (m *MockMessageHandler) OnError(err error) error {
	m.errorMessages = append(m.errorMessages, err)
	return err
}

func (m *MockMessageHandler) OnInfo(message string) error {
	m.infoMessages = append(m.infoMessages, message)
	return nil
}

func (m *MockMessageHandler) OnStatus(status string) error {
	m.statusMessages = append(m.statusMessages, status)
	return nil
}

func (m *MockMessageHandler) OnProgress(current, total int) error {
	m.progressUpdates = append(m.progressUpdates, ProgressUpdate{current, total})
	return nil
}

func (m *MockMessageHandler) OnCheckpoint(checkpointID string, action string) error {
	m.checkpointCalls = append(m.checkpointCalls, CheckpointCall{checkpointID, action})
	return nil
}

func (m *MockMessageHandler) OnBrowserAction(action string, url string) (string, error) {
	return "", nil
}

func (m *MockMessageHandler) OnMCPRequest(server string, tool string, params map[string]interface{}) (string, error) {
	return "", nil
}

func (m *MockMessageHandler) OnCompletion(success bool, summary string) error {
	m.completionCalled = true
	m.completionResult.success = success
	m.completionResult.summary = summary
	return nil
}

// TestNewRunner tests the creation of a new task runner
func TestNewRunner(t *testing.T) {
	t.Run("creates runner with valid connection", func(t *testing.T) {
		runner := NewRunner(nil)
		assert.NotNil(t, runner)
		assert.NotNil(t, runner.executor)
		assert.NotNil(t, runner.resumeManager)
		assert.NotNil(t, runner.attachmentMgr)
	})

	t.Run("runner components are initialized", func(t *testing.T) {
		runner := NewRunner(nil)
		assert.NotNil(t, runner.executor)
		assert.NotNil(t, runner.resumeManager)
		assert.NotNil(t, runner.attachmentMgr)
	})
}

// TestRunnerIsRunning tests the IsRunning method
func TestRunnerIsRunning(t *testing.T) {
	runner := NewRunner(nil)

	t.Run("returns false when not running", func(t *testing.T) {
		assert.False(t, runner.IsRunning())
	})

	t.Run("returns true when executor is running", func(t *testing.T) {
		runner.executor.isRunning = true
		assert.True(t, runner.IsRunning())

		runner.executor.isRunning = false
	})
}

// TestRunnerGetTaskID tests getting the current task ID
func TestRunnerGetTaskID(t *testing.T) {
	runner := NewRunner(nil)

	t.Run("returns empty when no task", func(t *testing.T) {
		assert.Empty(t, runner.GetTaskID())
	})

	t.Run("returns task ID when set", func(t *testing.T) {
		runner.executor.taskID = "test-task-123"
		assert.Equal(t, "test-task-123", runner.GetTaskID())

		runner.executor.taskID = ""
	})
}

// TestRunnerCancel tests cancellation
func TestRunnerCancel(t *testing.T) {
	runner := NewRunner(nil)

	t.Run("cancel doesn't panic when not running", func(t *testing.T) {
		assert.NotPanics(t, func() {
			runner.Cancel()
		})
	})

	t.Run("cancel marks executor as cancelled", func(t *testing.T) {
		runner.executor.isRunning = true
		runner.executor.cancelFunc = func() {}

		runner.Cancel()

		assert.True(t, runner.executor.IsCancelled())
	})
}

// TestRunnerClose tests closing the runner
func TestRunnerClose(t *testing.T) {
	runner := NewRunner(nil)

	t.Run("close doesn't panic", func(t *testing.T) {
		err := runner.Close()
		assert.NoError(t, err)
	})
}

// TestRunnerRunWithInvalidImages tests validation of images
func TestRunnerRunWithInvalidImages(t *testing.T) {
	runner := NewRunner(nil)

	t.Run("fails with invalid image path", func(t *testing.T) {
		ctx := context.Background()
		config := TaskConfig{
			Mode:   TaskModeAct,
			Prompt: "Test prompt",
			Images: []string{"/nonexistent/path/to/image.png"},
		}
		handler := NewMockMessageHandler()

		err := runner.Run(ctx, config, handler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "image validation failed")
	})
}

// TestRunnerRunWithEmptyPrompt tests running without prompt
func TestRunnerRunWithEmptyPrompt(t *testing.T) {
	runner := NewRunner(nil)

	t.Run("fails with empty prompt and no task ID", func(t *testing.T) {
		ctx := context.Background()
		config := TaskConfig{
			Mode:   TaskModeAct,
			Prompt: "",
		}
		handler := NewMockMessageHandler()

		err := runner.Run(ctx, config, handler)
		assert.Error(t, err)
	})
}

// TestRunnerRunWithInvalidMode tests validation of mode
func TestRunnerRunWithInvalidMode(t *testing.T) {
	runner := NewRunner(nil)

	t.Run("fails with invalid mode", func(t *testing.T) {
		ctx := context.Background()
		config := TaskConfig{
			Mode:   "invalid_mode",
			Prompt: "Test prompt",
		}
		handler := NewMockMessageHandler()

		err := runner.Run(ctx, config, handler)
		assert.Error(t, err)
	})
}

// TestRunnerGetResumableTasks tests getting resumable tasks
func TestRunnerGetResumableTasks(t *testing.T) {
	runner := NewRunner(nil)

	t.Run("returns error when no connection", func(t *testing.T) {
		t.Skip("Requires real gRPC client connection")
		ctx := context.Background()
		tasks, err := runner.GetResumableTasks(ctx)
		assert.Error(t, err)
		assert.Nil(t, tasks)
	})
}

// TestRunnerCanResume tests checking if a task can be resumed
func TestRunnerCanResume(t *testing.T) {
	t.Run("returns false for non-existent task", func(t *testing.T) {
		t.Skip("Requires real gRPC client connection")
		runner := NewRunner(nil)
		ctx := context.Background()
		canResume, reason := runner.CanResume(ctx, "non-existent-task")
		assert.False(t, canResume)
		assert.NotEmpty(t, reason)
	})
}

// TestRunnerResumeWithNoRecentTasks tests resuming when no recent tasks exist
func TestRunnerResumeMostRecentWithNoRecentTasks(t *testing.T) {
	runner := NewRunner(nil)

	t.Run("fails when no recent tasks", func(t *testing.T) {
		t.Skip("Requires real gRPC client connection")
		ctx := context.Background()
		handler := NewMockMessageHandler()

		err := runner.ResumeMostRecent(ctx, "test prompt", nil, handler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no recent tasks found")
	})
}

// TestRunnerContextCancellation tests that Run respects context cancellation
func TestRunnerContextCancellation(t *testing.T) {
	runner := NewRunner(nil)

	t.Run("context cancellation is handled", func(t *testing.T) {
		t.Skip("Requires real gRPC client connection")
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		config := TaskConfig{
			Mode:   TaskModeAct,
			Prompt: "Test prompt",
		}
		handler := NewMockMessageHandler()

		err := runner.Run(ctx, config, handler)
		assert.Error(t, err)
	})
}

// TestRunnerStateTransitions tests state management during task execution
func TestRunnerStateTransitions(t *testing.T) {
	runner := NewRunner(nil)

	t.Run("initial state is correct", func(t *testing.T) {
		assert.False(t, runner.IsRunning())
		assert.Empty(t, runner.GetTaskID())
	})
}

// TestModeHandlerIntegration tests integration with ModeHandler
func TestModeHandlerIntegration(t *testing.T) {
	t.Run("mode handler is set up correctly for act mode", func(t *testing.T) {
		t.Skip("Requires real gRPC client connection")
		runner := NewRunner(nil)

		ctx := context.Background()
		config := TaskConfig{
			Mode:   TaskModeAct,
			Prompt: "Test prompt",
		}
		handler := NewMockMessageHandler()

		_ = runner.Run(ctx, config, handler)

		assert.NotNil(t, runner.modeHandler)
		assert.Equal(t, TaskModeAct, runner.modeHandler.GetMode())
	})

	t.Run("mode handler is set up correctly for plan mode", func(t *testing.T) {
		t.Skip("Requires real gRPC client connection")
		runner := NewRunner(nil)

		ctx := context.Background()
		config := TaskConfig{
			Mode:   TaskModePlan,
			Prompt: "Test prompt",
		}
		handler := NewMockMessageHandler()

		_ = runner.Run(ctx, config, handler)

		assert.NotNil(t, runner.modeHandler)
		assert.Equal(t, TaskModePlan, runner.modeHandler.GetMode())
	})
}

// TestRunnerConcurrencySafety tests thread safety
func TestRunnerConcurrencySafety(t *testing.T) {
	runner := NewRunner(nil)

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

	t.Run("concurrent GetTaskID calls", func(t *testing.T) {
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				_ = runner.GetTaskID()
				done <- true
			}()
		}
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

// TestRunnerImageValidation tests image validation during Run
func TestRunnerImageValidation(t *testing.T) {
	runner := NewRunner(nil)

	t.Run("validates multiple images", func(t *testing.T) {
		ctx := context.Background()
		config := TaskConfig{
			Mode:   TaskModeAct,
			Prompt: "Test with images",
			Images: []string{
				"/nonexistent1.png",
				"/nonexistent2.png",
			},
		}
		handler := NewMockMessageHandler()

		err := runner.Run(ctx, config, handler)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "image validation failed")
	})

	t.Run("empty images slice is valid", func(t *testing.T) {
		t.Skip("Requires real gRPC client connection")
		ctx := context.Background()
		config := TaskConfig{
			Mode:   TaskModeAct,
			Prompt: "Test without images",
			Images: []string{},
		}
		handler := NewMockMessageHandler()

		err := runner.Run(ctx, config, handler)
		if err != nil {
			assert.NotContains(t, err.Error(), "image validation failed")
		}
	})
}

// BenchmarkRunnerOperations benchmarks runner operations
func BenchmarkRunnerOperations(b *testing.B) {
	runner := NewRunner(nil)

	b.Run("IsRunning", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = runner.IsRunning()
		}
	})

	b.Run("GetTaskID", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = runner.GetTaskID()
		}
	})

	b.Run("Cancel", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			runner.Cancel()
		}
	})
}
