// Package formatter provides test coverage for plain text formatting.
package formatter

import (
	"bytes"
	"errors"
	"testing"

	"github.com/cline/cline/golang-cli/internal/exit"
	"github.com/cline/cline/golang-cli/internal/task"
)

func TestPlainHandlerOnInfo(t *testing.T) {
	var buf bytes.Buffer
	handler := NewPlainHandler(&buf, true, false) // verbose = true

	err := handler.OnInfo("info message")
	if err != nil {
		t.Errorf("OnInfo failed: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Expected output for OnInfo in verbose mode")
	}
}

func TestPlainHandlerOnError(t *testing.T) {
	var buf bytes.Buffer
	var errBuf bytes.Buffer
	formatter := NewPlainFormatter(&buf, &errBuf, false, false)
	handler := &PlainHandler{
		formatter:   formatter,
		autoApprove: false,
		verbose:     false,
	}

	testErr := errors.New("test error")
	handlerErr := handler.OnError(testErr)
	if handlerErr != nil {
		t.Errorf("OnError failed: %v", handlerErr)
	}

	output := errBuf.String()
	if output == "" {
		t.Error("Expected error output for OnError")
	}
}

func TestPlainHandlerOnStatus(t *testing.T) {
	var buf bytes.Buffer
	handler := NewPlainHandler(&buf, true, false) // verbose = true

	err := handler.OnStatus("status message")
	if err != nil {
		t.Errorf("OnStatus failed: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Expected output for OnStatus in verbose mode")
	}
}

func TestPlainHandlerOnProgress(t *testing.T) {
	var buf bytes.Buffer
	handler := NewPlainHandler(&buf, true, false) // verbose = true

	err := handler.OnProgress(50, 100)
	if err != nil {
		t.Errorf("OnProgress failed: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Expected output for OnProgress in verbose mode")
	}
}

func TestPlainHandlerOnText(t *testing.T) {
	var buf bytes.Buffer
	handler := NewPlainHandler(&buf, true, false) // verbose = true

	err := handler.OnText("text content", false)
	if err != nil {
		t.Errorf("OnText failed: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("text content")) {
		t.Errorf("Expected output to contain 'text content', got: %s", output)
	}
}

func TestPlainHandlerOnToolUse(t *testing.T) {
	var buf bytes.Buffer
	handler := NewPlainHandler(&buf, true, false)

	params := map[string]interface{}{
		"file": "test.txt",
	}
	approved, err := handler.OnToolUse("read_file", params)
	if err != nil {
		t.Errorf("OnToolUse failed: %v", err)
	}

	// Should return true for approval
	if !approved {
		t.Error("Expected OnToolUse to return approved=true")
	}
}

func TestPlainHandlerOnToolUseAutoApprove(t *testing.T) {
	var buf bytes.Buffer
	handler := NewPlainHandler(&buf, true, true) // autoApprove = true

	params := map[string]interface{}{
		"file": "test.txt",
	}
	approved, err := handler.OnToolUse("read_file", params)
	if err != nil {
		t.Errorf("OnToolUse failed: %v", err)
	}

	// Should auto-approve
	if !approved {
		t.Error("Expected OnToolUse to auto-approve when autoApprove=true")
	}
}

func TestPlainHandlerOnToolResult(t *testing.T) {
	var buf bytes.Buffer
	handler := NewPlainHandler(&buf, true, false) // verbose = true

	err := handler.OnToolResult("read_file", "success", true)
	if err != nil {
		t.Errorf("OnToolResult failed: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Expected output for OnToolResult in verbose mode")
	}
}

func TestPlainHandlerOnToolResultFailure(t *testing.T) {
	var buf bytes.Buffer
	var errBuf bytes.Buffer
	formatter := NewPlainFormatter(&buf, &errBuf, false, false)
	handler := &PlainHandler{
		formatter:   formatter,
		autoApprove: false,
		verbose:     false,
	}

	err := handler.OnToolResult("read_file", "file not found", false)
	if err != nil {
		t.Errorf("OnToolResult failed: %v", err)
	}

	output := errBuf.String()
	// Should show output even in non-verbose mode for failures
	if output == "" {
		t.Error("Expected error output for OnToolResult failure even in non-verbose mode")
	}
}

func TestPlainHandlerOnCommand(t *testing.T) {
	var buf bytes.Buffer
	handler := NewPlainHandler(&buf, false, false)

	response, err := handler.OnCommand("ls -la", false)
	if err != nil {
		t.Errorf("OnCommand failed: %v", err)
	}

	if response != "execute" {
		t.Errorf("Expected 'execute', got: %s", response)
	}
}

func TestPlainHandlerOnCommandRequiresApproval(t *testing.T) {
	var buf bytes.Buffer
	handler := NewPlainHandler(&buf, false, false)

	_, err := handler.OnCommand("rm -rf /", true)
	if err == nil {
		t.Error("Expected error for command requiring approval")
	}
}

func TestPlainHandlerOnCommandAutoApprove(t *testing.T) {
	var buf bytes.Buffer
	handler := NewPlainHandler(&buf, false, true) // autoApprove = true

	response, err := handler.OnCommand("ls -la", true)
	if err != nil {
		t.Errorf("OnCommand failed: %v", err)
	}

	if response != "execute" {
		t.Errorf("Expected 'execute', got: %s", response)
	}
}

func TestPlainHandlerOnCommandOutput(t *testing.T) {
	var buf bytes.Buffer
	handler := NewPlainHandler(&buf, true, false) // verbose = true

	err := handler.OnCommandOutput("command output", true)
	if err != nil {
		t.Errorf("OnCommandOutput failed: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Expected output for OnCommandOutput in verbose mode")
	}
}

func TestPlainHandlerOnCheckpoint(t *testing.T) {
	var buf bytes.Buffer
	handler := NewPlainHandler(&buf, true, false) // verbose = true

	err := handler.OnCheckpoint("checkpoint-123", "created")
	if err != nil {
		t.Errorf("OnCheckpoint failed: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Expected output for OnCheckpoint in verbose mode")
	}
}

func TestPlainHandlerOnBrowserAction(t *testing.T) {
	var buf bytes.Buffer
	handler := NewPlainHandler(&buf, true, false) // verbose = true

	_, err := handler.OnBrowserAction("navigate", "https://example.com")
	if err != nil {
		t.Errorf("OnBrowserAction failed: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Expected output for OnBrowserAction in verbose mode")
	}
}

func TestPlainHandlerOnMCPRequest(t *testing.T) {
	var buf bytes.Buffer
	handler := NewPlainHandler(&buf, true, false) // verbose = true

	params := map[string]interface{}{
		"query": "test",
	}
	_, err := handler.OnMCPRequest("server1", "tool1", params)
	if err != nil {
		t.Errorf("OnMCPRequest failed: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Expected output for OnMCPRequest in verbose mode")
	}
}

func TestPlainHandlerOnCompletion(t *testing.T) {
	var buf bytes.Buffer
	handler := NewPlainHandler(&buf, false, false)

	err := handler.OnCompletion(true, "Task completed successfully")
	if err != nil {
		t.Errorf("OnCompletion failed: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Expected output for OnCompletion")
	}
}

func TestPlainHandlerFlush(t *testing.T) {
	var buf bytes.Buffer
	handler := NewPlainHandler(&buf, false, false)

	err := handler.Flush()
	if err != nil {
		t.Errorf("Flush failed: %v", err)
	}
}

func TestPlainHandlerSetExitHandler(t *testing.T) {
	var buf bytes.Buffer
	handler := NewPlainHandler(&buf, false, false)
	exitHandler := exit.NewHandler()

	handler.SetExitHandler(exitHandler)
	// No assertion needed - just verify it doesn't panic
}

func TestPlainHandlerHandleMessage(t *testing.T) {
	var buf bytes.Buffer
	handler := NewPlainHandler(&buf, true, false) // verbose = true

	tests := []struct {
		name    string
		msg     task.Message
		wantErr bool
	}{
		{
			name: "say message",
			msg: task.Message{
				Type:    task.MessageTypeSay,
				Content: "hello",
				Metadata: map[string]interface{}{
					"say_type": "text",
					"partial":  false,
				},
			},
			wantErr: false,
		},
		{
			name: "ask message",
			msg: task.Message{
				Type:    task.MessageTypeAsk,
				Content: "Approve?",
				Metadata: map[string]interface{}{
					"ask_type": "command",
				},
			},
			wantErr: false,
		},
		{
			name: "error message",
			msg: task.Message{
				Type:    task.MessageTypeError,
				Content: "error occurred",
			},
			wantErr: false,
		},
		{
			name: "unknown message type",
			msg: task.Message{
				Type:    "unknown",
				Content: "unknown content",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.HandleMessage(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPlainFormatterYOLOMode(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewPlainFormatter(&buf, &buf, false, false)

	// Test YOLO mode
	formatter.SetYOLOMode(true)

	if !formatter.yoloMode {
		t.Error("Expected YOLO mode to be enabled")
	}

	if !formatter.autoApproveRead {
		t.Error("Expected autoApproveRead to be true in YOLO mode")
	}

	if !formatter.autoApproveEdit {
		t.Error("Expected autoApproveEdit to be true in YOLO mode")
	}

	// Test ShouldAutoApprove with YOLO mode
	if !formatter.ShouldAutoApprove("", false, false, false, false) {
		t.Error("Expected ShouldAutoApprove to return true in YOLO mode")
	}

	// Test GetYOLOIndicator
	indicator := formatter.GetYOLOIndicator()
	if indicator == "" {
		t.Error("Expected non-empty YOLO indicator")
	}
}

func TestPlainFormatterAutoApproveSettings(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewPlainFormatter(&buf, &buf, false, false)

	// Test individual auto-approve settings
	formatter.SetAutoApproveSettings(true, false, true, false)

	if !formatter.autoApproveRead {
		t.Error("Expected autoApproveRead to be true")
	}

	if formatter.autoApproveEdit {
		t.Error("Expected autoApproveEdit to be false")
	}

	// Test ShouldAutoApprove with specific settings
	if !formatter.ShouldAutoApprove("", true, false, false, false) {
		t.Error("Expected ShouldAutoApprove to return true for read operations")
	}

	if formatter.ShouldAutoApprove("", false, true, false, false) {
		t.Error("Expected ShouldAutoApprove to return false for write operations")
	}
}

func TestPlainFormatterFormatAskMessage(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewPlainFormatter(&buf, &buf, false, false)

	// This will prompt for input, which we can't test automatically
	// Just verify it doesn't panic
	response, err := formatter.FormatAskMessage("tool", "Approve read_file?")
	// We expect an error since we can't provide input in tests
	if err == nil {
		t.Logf("Got response: %s", response)
	}
}

func TestPlainFormatterFormatProgress(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewPlainFormatter(&buf, &buf, false, true) // verbose = true

	err := formatter.FormatProgress(50, 100, "50/100")
	if err != nil {
		t.Errorf("FormatProgress failed: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Expected output for FormatProgress in verbose mode")
	}
}

func TestPlainFormatterFormatTimeout(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewPlainFormatter(&buf, &buf, false, false)
	exitHandler := exit.NewHandler()
	formatter.SetExitHandler(exitHandler)

	err := formatter.FormatTimeout(30)
	if err != nil {
		t.Errorf("FormatTimeout failed: %v", err)
	}

	if exitHandler.GetExitCode() != exit.Timeout {
		t.Errorf("Expected exit code %d, got %d", exit.Timeout, exitHandler.GetExitCode())
	}
}

func TestPlainFormatterFormatInterrupted(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewPlainFormatter(&buf, &buf, false, false)
	exitHandler := exit.NewHandler()
	formatter.SetExitHandler(exitHandler)

	err := formatter.FormatInterrupted()
	if err != nil {
		t.Errorf("FormatInterrupted failed: %v", err)
	}

	if exitHandler.GetExitCode() != exit.Interrupted {
		t.Errorf("Expected exit code %d, got %d", exit.Interrupted, exitHandler.GetExitCode())
	}
}

func TestPlainFormatterFormatCheckpoint(t *testing.T) {
	var buf bytes.Buffer
	handler := NewPlainHandler(&buf, true, false) // verbose = true

	err := handler.OnCheckpoint("abc123", "created")
	if err != nil {
		t.Errorf("OnCheckpoint failed: %v", err)
	}
}

func TestPlainFormatterFormatCommand(t *testing.T) {
	var buf bytes.Buffer
	handler := NewPlainHandler(&buf, true, false) // verbose = true

	response, err := handler.OnCommand("ls -la", false)
	if err != nil {
		t.Errorf("OnCommand failed: %v", err)
	}

	if response != "execute" {
		t.Errorf("Expected 'execute', got: %s", response)
	}
}

func TestPlainFormatterFormatToolUse(t *testing.T) {
	var buf bytes.Buffer
	handler := NewPlainHandler(&buf, true, false)

	params := map[string]interface{}{
		"file":    "test.txt",
		"content": "hello",
	}
	approved, err := handler.OnToolUse("write_file", params)
	if err != nil {
		t.Errorf("OnToolUse failed: %v", err)
	}

	if !approved {
		t.Error("Expected OnToolUse to return approved=true")
	}
}

func TestPlainFormatterFormatToolResult(t *testing.T) {
	var buf bytes.Buffer
	handler := NewPlainHandler(&buf, true, false) // verbose = true

	err := handler.OnToolResult("write_file", "success", true)
	if err != nil {
		t.Errorf("OnToolResult failed: %v", err)
	}
}

func TestPlainFormatterFormatError(t *testing.T) {
	var buf bytes.Buffer
	var errBuf bytes.Buffer
	formatter := NewPlainFormatter(&buf, &errBuf, false, false)

	testErr := errors.New("test error")
	err := formatter.FormatError(testErr)
	if err != nil {
		t.Errorf("FormatError failed: %v", err)
	}

	output := errBuf.String()
	if output == "" {
		t.Error("Expected error output")
	}
}

func TestPlainFormatterFormatStatus(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewPlainFormatter(&buf, &buf, false, true) // verbose = true

	err := formatter.FormatStatus("status message")
	if err != nil {
		t.Errorf("FormatStatus failed: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Expected output for FormatStatus in verbose mode")
	}
}

func TestPlainFormatterSayMessageTypes(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewPlainFormatter(&buf, &buf, false, true) // verbose = true

	tests := []struct {
		name    string
		sayType string
		text    string
	}{
		{"text", "text", "Hello"},
		{"task", "task", "Task started"},
		{"api_req_started", "api_req_started", ""},
		{"api_req_finished", "api_req_finished", ""},
		{"command", "command", "ls -la"},
		{"command_output", "command_output", "output"},
		{"tool", "tool", "tool message"},
		{"tool_use", "tool_use", "Using tool"},
		{"tool_result", "tool_result", "Tool completed"},
		{"completion_result", "completion_result", "Done"},
		{"thinking", "thinking", "Thinking..."},
		{"reasoning", "reasoning", "Reasoning..."},
		{"browser_action", "browser_action", "navigate"},
		{"browser_action_result", "browser_action_result", "result"},
		{"checkpoint_created", "checkpoint_created", "checkpoint-123"},
		{"mcp_server_request_started", "mcp_server_request_started", "request"},
		{"mcp_server_response", "mcp_server_response", "response"},
		{"user_feedback", "user_feedback", "feedback"},
		{"diff_error", "diff_error", "diff failed"},
		{"shell_integration_warning", "shell_integration_warning", "warning"},
		{"clineignore_error", "clineignore_error", "error"},
		{"command_permission_denied", "command_permission_denied", "denied"},
		{"info", "info", "info message"},
		{"task_progress", "task_progress", "progress"},
		{"unknown", "unknown_type", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			err := formatter.FormatSayMessage(tt.sayType, tt.text, false)
			if err != nil {
				t.Errorf("FormatSayMessage(%s) failed: %v", tt.sayType, err)
			}
			// Some types only output in verbose mode, which we've enabled
			_ = buf.String() // Just verify no panic
		})
	}
}

func TestScriptingHandlerOnSay(t *testing.T) {
	var buf bytes.Buffer
	handler := NewScriptingHandler(&buf, false)

	handler.OnSay("text", "Hello", false)

	output := buf.String()
	if output == "" {
		t.Error("Expected output from ScriptingHandler.OnSay")
	}
}

func TestScriptingHandlerOnSayFilters(t *testing.T) {
	var buf bytes.Buffer
	handler := NewScriptingHandler(&buf, false) // verbose = false

	// These should be filtered out in non-verbose mode
	handler.OnSay("command", "ls -la", false)
	handler.OnSay("command_output", "output", false)

	output := buf.String()
	if output != "" {
		t.Errorf("Expected no output for filtered types in non-verbose mode, got: %s", output)
	}
}

func TestScriptingHandlerOnAsk(t *testing.T) {
	var buf bytes.Buffer
	handler := NewScriptingHandler(&buf, false)

	response, err := handler.OnAsk("tool", "Approve?")
	if err != nil {
		t.Errorf("OnAsk failed: %v", err)
	}

	if response != "yesButtonClicked" {
		t.Errorf("Expected 'yesButtonClicked', got: %s", response)
	}
}