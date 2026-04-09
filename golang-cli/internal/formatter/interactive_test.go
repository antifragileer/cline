package formatter

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cline/cline/golang-cli/internal/task"
)

func TestNewInteractiveHandler(t *testing.T) {
	var output bytes.Buffer
	handler := NewInteractiveHandler(&output, false, false)

	if handler == nil {
		t.Fatal("Expected handler to be created")
	}

	if handler.verbose != false {
		t.Error("Expected verbose to be false")
	}

	if handler.autoApprove != false {
		t.Error("Expected autoApprove to be false")
	}

	if handler.output != &output {
		t.Error("Expected output to be set correctly")
	}

	// Test with different options
	handler2 := NewInteractiveHandler(&output, true, true)
	if handler2.verbose != true {
		t.Error("Expected verbose to be true")
	}
	if handler2.autoApprove != true {
		t.Error("Expected autoApprove to be true")
	}
}

func TestSetExitHandler(t *testing.T) {
	var output bytes.Buffer
	handler := NewInteractiveHandler(&output, false, false)

	called := false
	handler.SetExitHandler(func() {
		called = true
	})

	if handler.exitHandler == nil {
		t.Error("Expected exit handler to be set")
	}

	// Test calling the handler
	handler.exitHandler()
	if !called {
		t.Error("Expected exit handler to be called")
	}
}

func TestOnText(t *testing.T) {
	t.Run("complete message", func(t *testing.T) {
		var output bytes.Buffer
		handler := NewInteractiveHandler(&output, false, false)

		err := handler.OnText("Hello, World!", false)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		expected := "Hello, World!\n"
		if output.String() != expected {
			t.Errorf("Expected %q, got %q", expected, output.String())
		}
	})

	t.Run("streaming state tracking", func(t *testing.T) {
		var output bytes.Buffer
		handler := NewInteractiveHandler(&output, false, false)

		// Start streaming
		err := handler.OnText("Partial", true)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if !handler.isStreaming {
			t.Error("Expected isStreaming to be true")
		}

		// Complete streaming
		err = handler.OnText("Complete", false)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if handler.isStreaming {
			t.Error("Expected isStreaming to be false after complete")
		}
	})
}

func TestOnToolResult(t *testing.T) {
	t.Run("successful tool", func(t *testing.T) {
		var output bytes.Buffer
		handler := NewInteractiveHandler(&output, false, false)

		err := handler.OnToolResult("test-tool", "result", true)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if !strings.Contains(output.String(), "✓") {
			t.Error("Expected checkmark for successful tool")
		}
		if !strings.Contains(output.String(), "test-tool") {
			t.Error("Expected tool name in output")
		}
	})

	t.Run("failed tool", func(t *testing.T) {
		var output bytes.Buffer
		handler := NewInteractiveHandler(&output, false, false)

		err := handler.OnToolResult("test-tool", "error", false)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if !strings.Contains(output.String(), "✗") {
			t.Error("Expected X mark for failed tool")
		}
	})

	t.Run("verbose output", func(t *testing.T) {
		var output bytes.Buffer
		handler := NewInteractiveHandler(&output, true, false)

		err := handler.OnToolResult("test-tool", "detailed result", true)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if !strings.Contains(output.String(), "detailed result") {
			t.Error("Expected result in verbose output")
		}
	})
}

func TestOnSay(t *testing.T) {
	tests := []struct {
		sayType  string
		content  string
		partial  bool
		expected string
	}{
		{"text", "Hello", false, "Hello"},
		{"error", "Something went wrong", false, "Error"},
		{"command", "ls -la", false, "🖥"},
		{"tool", "Using tool", false, "🔧"},
		{"unknown", "Unknown message", false, "Unknown message"},
	}

	for _, test := range tests {
		t.Run(test.sayType, func(t *testing.T) {
			var output bytes.Buffer
			handler := NewInteractiveHandler(&output, false, false)

			err := handler.OnSay(test.sayType, test.content, test.partial)
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			if !strings.Contains(output.String(), test.expected) {
				t.Errorf("Expected output to contain %q, got %q", test.expected, output.String())
			}
		})
	}
}

func TestOnCommand(t *testing.T) {
	t.Run("auto approve", func(t *testing.T) {
		var output bytes.Buffer
		handler := NewInteractiveHandler(&output, false, true)

		result, err := handler.OnCommand("ls -la", true)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if result != "approved" {
			t.Errorf("Expected 'approved', got %q", result)
		}

		if !strings.Contains(output.String(), "Executing") {
			t.Error("Expected command execution message")
		}
	})

	t.Run("no approval required", func(t *testing.T) {
		var output bytes.Buffer
		handler := NewInteractiveHandler(&output, false, false)

		result, err := handler.OnCommand("ls -la", false)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if result != "approved" {
			t.Errorf("Expected 'approved', got %q", result)
		}
	})
}

func TestOnCommandOutput(t *testing.T) {
	t.Run("verbose mode", func(t *testing.T) {
		var output bytes.Buffer
		handler := NewInteractiveHandler(&output, true, false)

		err := handler.OnCommandOutput("output line\n", false)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if output.String() != "output line\n" {
			t.Errorf("Expected output to be printed, got %q", output.String())
		}
	})

	t.Run("non-verbose complete", func(t *testing.T) {
		var output bytes.Buffer
		handler := NewInteractiveHandler(&output, false, false)

		err := handler.OnCommandOutput("output", true)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if output.String() != "" {
			t.Error("Expected no output in non-verbose mode when complete")
		}
	})
}

func TestOnError(t *testing.T) {
	var output bytes.Buffer
	handler := NewInteractiveHandler(&output, false, false)

	testErr := fmt.Errorf("test error")
	err := handler.OnError(testErr)

	if err != testErr {
		t.Error("Expected error to be returned")
	}

	if !strings.Contains(output.String(), "❌") {
		t.Error("Expected error emoji in output")
	}

	if !strings.Contains(output.String(), "test error") {
		t.Error("Expected error message in output")
	}
}

func TestOnInfo(t *testing.T) {
	t.Run("verbose mode", func(t *testing.T) {
		var output bytes.Buffer
		handler := NewInteractiveHandler(&output, true, false)

		err := handler.OnInfo("info message")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if !strings.Contains(output.String(), "[INFO]") {
			t.Error("Expected [INFO] prefix")
		}

		if !strings.Contains(output.String(), "info message") {
			t.Error("Expected message in output")
		}
	})

	t.Run("non-verbose mode", func(t *testing.T) {
		var output bytes.Buffer
		handler := NewInteractiveHandler(&output, false, false)

		err := handler.OnInfo("info message")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if output.String() != "" {
			t.Error("Expected no output in non-verbose mode")
		}
	})
}

func TestOnStatus(t *testing.T) {
	t.Run("verbose mode", func(t *testing.T) {
		var output bytes.Buffer
		handler := NewInteractiveHandler(&output, true, false)

		err := handler.OnStatus("running")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if !strings.Contains(output.String(), "[STATUS]") {
			t.Error("Expected [STATUS] prefix")
		}
	})

	t.Run("non-verbose mode", func(t *testing.T) {
		var output bytes.Buffer
		handler := NewInteractiveHandler(&output, false, false)

		err := handler.OnStatus("running")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if output.String() != "" {
			t.Error("Expected no output in non-verbose mode")
		}
	})
}

func TestOnProgress(t *testing.T) {
	var output bytes.Buffer
	handler := NewInteractiveHandler(&output, false, false)

	err := handler.OnProgress(50, 100)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if !strings.Contains(output.String(), "[PROGRESS]") {
		t.Error("Expected [PROGRESS] prefix")
	}

	if !strings.Contains(output.String(), "50/100") {
		t.Error("Expected progress numbers")
	}

	if !strings.Contains(output.String(), "50.0%") {
		t.Error("Expected percentage")
	}
}

func TestOnCheckpoint(t *testing.T) {
	t.Run("verbose mode", func(t *testing.T) {
		var output bytes.Buffer
		handler := NewInteractiveHandler(&output, true, false)

		err := handler.OnCheckpoint("checkpoint-123", "create")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if !strings.Contains(output.String(), "[CHECKPOINT]") {
			t.Error("Expected [CHECKPOINT] prefix")
		}

		if !strings.Contains(output.String(), "checkpoint-123") {
			t.Error("Expected checkpoint ID in output")
		}
	})

	t.Run("non-verbose mode", func(t *testing.T) {
		var output bytes.Buffer
		handler := NewInteractiveHandler(&output, false, false)

		err := handler.OnCheckpoint("checkpoint-123", "create")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if output.String() != "" {
			t.Error("Expected no output in non-verbose mode")
		}
	})
}

func TestOnBrowserAction(t *testing.T) {
	var output bytes.Buffer
	handler := NewInteractiveHandler(&output, false, false)

	result, err := handler.OnBrowserAction("launch", "https://example.com")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if result != "approved" {
		t.Errorf("Expected 'approved', got %q", result)
	}

	if !strings.Contains(output.String(), "🌐") {
		t.Error("Expected browser emoji")
	}

	if !strings.Contains(output.String(), "launch") {
		t.Error("Expected action in output")
	}

	if !strings.Contains(output.String(), "https://example.com") {
		t.Error("Expected URL in output")
	}
}

func TestOnMCPRequest(t *testing.T) {
	t.Run("auto approve", func(t *testing.T) {
		var output bytes.Buffer
		handler := NewInteractiveHandler(&output, false, true)

		result, err := handler.OnMCPRequest("server1", "tool1", nil)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if result != "approved" {
			t.Errorf("Expected 'approved', got %q", result)
		}
	})
}

func TestOnCompletion(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var output bytes.Buffer
		handler := NewInteractiveHandler(&output, false, false)

		err := handler.OnCompletion(true, "Task completed")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if !strings.Contains(output.String(), "✓") {
			t.Error("Expected checkmark for success")
		}

		if !strings.Contains(output.String(), "Task completed") {
			t.Error("Expected summary in output")
		}
	})

	t.Run("failure", func(t *testing.T) {
		var output bytes.Buffer
		handler := NewInteractiveHandler(&output, false, false)

		err := handler.OnCompletion(false, "")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if !strings.Contains(output.String(), "✗") {
			t.Error("Expected X mark for failure")
		}
	})
}

func TestFormatToolPrompt(t *testing.T) {
	var output bytes.Buffer
	handler := NewInteractiveHandler(&output, false, false)

	params := map[string]interface{}{
		"file": "/path/to/file",
		"content": "test content",
	}

	prompt := handler.formatToolPrompt("write_to_file", params)

	if !strings.Contains(prompt, "write_to_file") {
		t.Error("Expected tool name in prompt")
	}

	if !strings.Contains(prompt, "Parameters") {
		t.Error("Expected Parameters section")
	}

	if !strings.Contains(prompt, "file:") {
		t.Error("Expected file parameter")
	}

	if !strings.Contains(prompt, "[y]es, [n]o, [a]lways") {
		t.Error("Expected options")
	}
}

func TestStreamingControl(t *testing.T) {
	handler := &InteractiveHandler{}

	// Test pause
	handler.pauseStreaming()
	if !handler.streamPaused {
		t.Error("Expected stream to be paused")
	}

	if !handler.isStreamingPaused() {
		t.Error("Expected isStreamingPaused to return true")
	}

	// Test resume
	handler.resumeStreaming()
	if handler.streamPaused {
		t.Error("Expected stream to be resumed")
	}

	if handler.isStreamingPaused() {
		t.Error("Expected isStreamingPaused to return false")
	}
}

func TestToolApprovalStorage(t *testing.T) {
	var output bytes.Buffer
	handler := NewInteractiveHandler(&output, false, false)

	// Initially not approved
	if handler.isToolApproved("test-tool") {
		t.Error("Expected tool to not be approved initially")
	}

	// Approve the tool
	handler.approveTool("test-tool")

	// Now should be approved
	if !handler.isToolApproved("test-tool") {
		t.Error("Expected tool to be approved after calling approveTool")
	}

	// Other tools should not be affected
	if handler.isToolApproved("other-tool") {
		t.Error("Expected other tool to not be approved")
	}
}

func TestHandleMessage(t *testing.T) {
	t.Run("text message", func(t *testing.T) {
		var output bytes.Buffer
		handler := NewInteractiveHandler(&output, false, false)

		msg := task.Message{
			Type:    task.MessageTypeText,
			Content: "Hello",
		}

		err := handler.HandleMessage(msg)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if !strings.Contains(output.String(), "Hello") {
			t.Error("Expected message content in output")
		}
	})

	t.Run("error message", func(t *testing.T) {
		var output bytes.Buffer
		handler := NewInteractiveHandler(&output, false, false)

		msg := task.Message{
			Type:    task.MessageTypeError,
			Content: "Something went wrong",
		}

		err := handler.HandleMessage(msg)
		if err == nil {
			t.Error("Expected error to be returned")
		}

		if !strings.Contains(output.String(), "❌") {
			t.Error("Expected error emoji in output")
		}
	})

	t.Run("verbose unknown type", func(t *testing.T) {
		var output bytes.Buffer
		handler := NewInteractiveHandler(&output, true, false)

		msg := task.Message{
			Type:    task.MessageType("unknown"),
			Content: "test",
		}

		err := handler.HandleMessage(msg)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if !strings.Contains(output.String(), "[INFO]") {
			t.Error("Expected [INFO] prefix for verbose unknown type")
		}
	})
}

func TestApprovalPromptTimeout(t *testing.T) {
	// This test verifies the ApprovalPrompt structure
	prompt := &ApprovalPrompt{
		AskType:  "test",
		Text:     "Test prompt",
		Response: make(chan<- string, 1),
		Received: time.Now(),
	}

	if prompt.AskType != "test" {
		t.Error("Expected AskType to be set")
	}

	if prompt.Text != "Test prompt" {
		t.Error("Expected Text to be set")
	}

	if prompt.Response == nil {
		t.Error("Expected Response channel to be set")
	}

	if prompt.Received.IsZero() {
		t.Error("Expected Received time to be set")
	}
}

// Verify interface implementation
func TestInterfaceImplementation(t *testing.T) {
	var output bytes.Buffer
	handler := NewInteractiveHandler(&output, false, false)

	// This will fail at compile time if InteractiveHandler doesn't implement MessageHandler
	var _ task.MessageHandler = handler
}