// Package formatter provides test coverage for output formatting.
package formatter

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/cline/cline/golang-cli/internal/exit"
)

func TestNewFormatter(t *testing.T) {
	tests := []struct {
		name     string
		opts     []FormatterOption
		expected Format
	}{
		{
			name:     "default formatter",
			opts:     nil,
			expected: FormatPlain,
		},
		{
			name:     "json formatter",
			opts:     []FormatterOption{WithFormat(FormatJSON)},
			expected: FormatJSON,
		},
		{
			name:     "json lines formatter",
			opts:     []FormatterOption{WithFormat(FormatJSONLines)},
			expected: FormatJSONLines,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFormatter(tt.opts...)
			if f.format != tt.expected {
				t.Errorf("expected format %v, got %v", tt.expected, f.format)
			}
		})
	}
}

func TestJSONFormatter(t *testing.T) {
	t.Run("FormatSayMessage", func(t *testing.T) {
		var buf bytes.Buffer
		f := NewJSONFormatter(&buf, os.Stderr, false)

		err := f.FormatSayMessage("text", "Hello World", false, nil)
		if err != nil {
			t.Fatalf("FormatSayMessage failed: %v", err)
		}

		var msg JSONMessage
		if err := json.Unmarshal(buf.Bytes(), &msg); err != nil {
			t.Fatalf("Failed to unmarshal JSON: %v", err)
		}

		if msg.Type != "say" {
			t.Errorf("expected type 'say', got %s", msg.Type)
		}
		if msg.Say != "text" {
			t.Errorf("expected say 'text', got %s", msg.Say)
		}
		if msg.Text != "Hello World" {
			t.Errorf("expected text 'Hello World', got %s", msg.Text)
		}
	})

	t.Run("FormatAskMessage", func(t *testing.T) {
		var buf bytes.Buffer
		f := NewJSONFormatter(&buf, os.Stderr, false)

		err := f.FormatAskMessage("command", "Approve?", nil)
		if err != nil {
			t.Fatalf("FormatAskMessage failed: %v", err)
		}

		var msg JSONMessage
		if err := json.Unmarshal(buf.Bytes(), &msg); err != nil {
			t.Fatalf("Failed to unmarshal JSON: %v", err)
		}

		if msg.Type != "ask" {
			t.Errorf("expected type 'ask', got %s", msg.Type)
		}
		if msg.Ask != "command" {
			t.Errorf("expected ask 'command', got %s", msg.Ask)
		}
	})

	t.Run("FormatError", func(t *testing.T) {
		var buf bytes.Buffer
		f := NewJSONFormatter(&buf, os.Stderr, false)

		testErr := errors.New("test error")
		err := f.FormatError(testErr)
		if err != nil {
			t.Fatalf("FormatError failed: %v", err)
		}

		var msg JSONMessage
		if err := json.Unmarshal(buf.Bytes(), &msg); err != nil {
			t.Fatalf("Failed to unmarshal JSON: %v", err)
		}

		if msg.Type != "error" {
			t.Errorf("expected type 'error', got %s", msg.Type)
		}
		if msg.Error != "test error" {
			t.Errorf("expected error 'test error', got %s", msg.Error)
		}
	})

	t.Run("FormatProgress", func(t *testing.T) {
		var buf bytes.Buffer
		f := NewJSONFormatter(&buf, os.Stderr, false)

		err := f.FormatProgress(50, 100, "50/100")
		if err != nil {
			t.Fatalf("FormatProgress failed: %v", err)
		}

		var msg JSONMessage
		if err := json.Unmarshal(buf.Bytes(), &msg); err != nil {
			t.Fatalf("Failed to unmarshal JSON: %v", err)
		}

		if msg.Type != "say" {
			t.Errorf("expected type 'say', got %s", msg.Type)
		}
		if msg.Say != "task_progress" {
			t.Errorf("expected say 'task_progress', got %s", msg.Say)
		}
	})

	t.Run("Streaming mode", func(t *testing.T) {
		var buf bytes.Buffer
		f := NewJSONFormatter(&buf, os.Stderr, true) // streaming mode

		err := f.FormatSayMessage("text", "Message 1", false, nil)
		if err != nil {
			t.Fatalf("FormatSayMessage failed: %v", err)
		}

		err = f.FormatSayMessage("text", "Message 2", false, nil)
		if err != nil {
			t.Fatalf("FormatSayMessage failed: %v", err)
		}

		lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		if len(lines) != 2 {
			t.Errorf("expected 2 lines in streaming mode, got %d", len(lines))
		}

		// Verify each line is valid JSON
		for i, line := range lines {
			var msg JSONMessage
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				t.Errorf("line %d is not valid JSON: %v", i, err)
			}
		}
	})

	t.Run("FormatCheckpoint", func(t *testing.T) {
		var buf bytes.Buffer
		f := NewJSONFormatter(&buf, os.Stderr, false)

		err := f.FormatCheckpoint("abc123", true)
		if err != nil {
			t.Fatalf("FormatCheckpoint failed: %v", err)
		}

		var msg JSONMessage
		if err := json.Unmarshal(buf.Bytes(), &msg); err != nil {
			t.Fatalf("Failed to unmarshal JSON: %v", err)
		}

		if msg.LastCheckpointHash != "abc123" {
			t.Errorf("expected hash 'abc123', got %s", msg.LastCheckpointHash)
		}
		if !msg.IsCheckpointCheckedOut {
			t.Error("expected IsCheckpointCheckedOut to be true")
		}
	})

	t.Run("FormatCommand", func(t *testing.T) {
		var buf bytes.Buffer
		f := NewJSONFormatter(&buf, os.Stderr, false)

		err := f.FormatCommand("ls -la", true, "output")
		if err != nil {
			t.Fatalf("FormatCommand failed: %v", err)
		}

		var msg JSONMessage
		if err := json.Unmarshal(buf.Bytes(), &msg); err != nil {
			t.Fatalf("Failed to unmarshal JSON: %v", err)
		}

		if msg.Text != "ls -la" {
			t.Errorf("expected command 'ls -la', got %s", msg.Text)
		}
		if !msg.CommandCompleted {
			t.Error("expected CommandCompleted to be true")
		}
	})

	t.Run("FormatToolUse", func(t *testing.T) {
		var buf bytes.Buffer
		f := NewJSONFormatter(&buf, os.Stderr, false)

		toolInput := map[string]interface{}{
			"file":    "test.txt",
			"content": "hello",
		}
		err := f.FormatToolUse("write_file", toolInput, false)
		if err != nil {
			t.Fatalf("FormatToolUse failed: %v", err)
		}

		var msg JSONMessage
		if err := json.Unmarshal(buf.Bytes(), &msg); err != nil {
			t.Fatalf("Failed to unmarshal JSON: %v", err)
		}

		if msg.ToolName != "write_file" {
			t.Errorf("expected tool name 'write_file', got %s", msg.ToolName)
		}
		if msg.ToolInput == nil {
			t.Error("expected ToolInput to be set")
		}
	})

	t.Run("FormatToolResult", func(t *testing.T) {
		var buf bytes.Buffer
		f := NewJSONFormatter(&buf, os.Stderr, false)

		err := f.FormatToolResult("write_file", "success", false)
		if err != nil {
			t.Fatalf("FormatToolResult failed: %v", err)
		}

		var msg JSONMessage
		if err := json.Unmarshal(buf.Bytes(), &msg); err != nil {
			t.Fatalf("Failed to unmarshal JSON: %v", err)
		}

		if msg.ToolResult != "success" {
			t.Errorf("expected result 'success', got %s", msg.ToolResult)
		}
	})
}

func TestPlainFormatter(t *testing.T) {
	t.Run("FormatSayMessage text", func(t *testing.T) {
		var buf bytes.Buffer
		f := NewPlainFormatter(&buf, os.Stderr, false, false)

		err := f.FormatSayMessage("text", "Hello World", false)
		if err != nil {
			t.Fatalf("FormatSayMessage failed: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "Hello World") {
			t.Errorf("expected output to contain 'Hello World', got %s", output)
		}
	})

	t.Run("FormatSayMessage skips partial", func(t *testing.T) {
		var buf bytes.Buffer
		f := NewPlainFormatter(&buf, os.Stderr, false, false)

		err := f.FormatSayMessage("text", "Hello", true) // partial = true
		if err != nil {
			t.Fatalf("FormatSayMessage failed: %v", err)
		}

		if buf.Len() != 0 {
			t.Errorf("expected no output for partial message, got %s", buf.String())
		}
	})

	t.Run("FormatSayMessage error", func(t *testing.T) {
		var buf bytes.Buffer
		var errBuf bytes.Buffer
		f := NewPlainFormatter(&buf, &errBuf, false, false)

		err := f.FormatSayMessage("error", "Something went wrong", false)
		if err != nil {
			t.Fatalf("FormatSayMessage failed: %v", err)
		}

		errOutput := errBuf.String()
		if !strings.Contains(errOutput, "Error") {
			t.Errorf("expected error output to contain 'Error', got %s", errOutput)
		}
		if !strings.Contains(errOutput, "Something went wrong") {
			t.Errorf("expected error output to contain message, got %s", errOutput)
		}
	})

	t.Run("FormatSayMessage with exit handler", func(t *testing.T) {
		var buf bytes.Buffer
		f := NewPlainFormatter(&buf, os.Stderr, false, false)
		handler := exit.NewHandler()
		f.SetExitHandler(handler)

		f.FormatSayMessage("error", "Task failed", false)

		if handler.GetExitCode() != exit.TaskFailed {
			t.Errorf("expected exit code %d, got %d", exit.TaskFailed, handler.GetExitCode())
		}
	})

	t.Run("FormatError", func(t *testing.T) {
		var buf bytes.Buffer
		var errBuf bytes.Buffer
		f := NewPlainFormatter(&buf, &errBuf, false, false)

		testErr := errors.New("test error")
		err := f.FormatError(testErr)
		if err != nil {
			t.Fatalf("FormatError failed: %v", err)
		}

		errOutput := errBuf.String()
		if !strings.Contains(errOutput, "test error") {
			t.Errorf("expected error output to contain 'test error', got %s", errOutput)
		}
	})

	t.Run("FormatProgress", func(t *testing.T) {
		var buf bytes.Buffer
		f := NewPlainFormatter(&buf, os.Stderr, false, true) // verbose = true

		err := f.FormatProgress(50, 100, "")
		if err != nil {
			t.Fatalf("FormatProgress failed: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "PROGRESS") {
			t.Errorf("expected output to contain 'PROGRESS', got %s", output)
		}
	})

	t.Run("FormatTimeout", func(t *testing.T) {
		var buf bytes.Buffer
		f := NewPlainFormatter(&buf, os.Stderr, false, false)
		handler := exit.NewHandler()
		f.SetExitHandler(handler)

		err := f.FormatTimeout(30)
		if err != nil {
			t.Fatalf("FormatTimeout failed: %v", err)
		}

		if handler.GetExitCode() != exit.Timeout {
			t.Errorf("expected exit code %d, got %d", exit.Timeout, handler.GetExitCode())
		}
	})

	t.Run("FormatInterrupted", func(t *testing.T) {
		var buf bytes.Buffer
		f := NewPlainFormatter(&buf, os.Stderr, false, false)
		handler := exit.NewHandler()
		f.SetExitHandler(handler)

		err := f.FormatInterrupted()
		if err != nil {
			t.Fatalf("FormatInterrupted failed: %v", err)
		}

		if handler.GetExitCode() != exit.Interrupted {
			t.Errorf("expected exit code %d, got %d", exit.Interrupted, handler.GetExitCode())
		}
	})
}

func TestJSONHandler(t *testing.T) {
	t.Run("OnSay", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewJSONHandler(&buf, false)

		h.OnSay("text", "Hello", false)

		var msg JSONMessage
		if err := json.Unmarshal(buf.Bytes(), &msg); err != nil {
			t.Fatalf("Failed to unmarshal JSON: %v", err)
		}

		if msg.Text != "Hello" {
			t.Errorf("expected text 'Hello', got %s", msg.Text)
		}
	})

	t.Run("OnSay partial skipped", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewJSONHandler(&buf, false) // verbose = false

		h.OnSay("text", "Hello", true) // partial = true

		if buf.Len() != 0 {
			t.Errorf("expected no output for partial message in non-verbose mode, got %s", buf.String())
		}
	})

	t.Run("OnAsk auto-approves", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewJSONHandler(&buf, false)

		response, err := h.OnAsk("command", "Approve?")
		if err != nil {
			t.Fatalf("OnAsk failed: %v", err)
		}

		if response != "yesButtonClicked" {
			t.Errorf("expected 'yesButtonClicked', got %s", response)
		}
	})

	t.Run("OnError", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewJSONHandler(&buf, false)

		testErr := errors.New("test error")
		h.OnError(testErr)

		var msg JSONMessage
		if err := json.Unmarshal(buf.Bytes(), &msg); err != nil {
			t.Fatalf("Failed to unmarshal JSON: %v", err)
		}

		if msg.Error != "test error" {
			t.Errorf("expected error 'test error', got %s", msg.Error)
		}
	})
}

func TestPlainHandler(t *testing.T) {
	t.Run("OnSay", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewPlainHandler(&buf, false, false)

		h.OnSay("text", "Hello", false)

		output := buf.String()
		if !strings.Contains(output, "Hello") {
			t.Errorf("expected output to contain 'Hello', got %s", output)
		}
	})

	t.Run("OnAsk auto-approve", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewPlainHandler(&buf, false, true) // autoApprove = true

		response, err := h.OnAsk("command", "Approve?")
		if err != nil {
			t.Fatalf("OnAsk failed: %v", err)
		}

		if response != "yesButtonClicked" {
			t.Errorf("expected 'yesButtonClicked', got %s", response)
		}
	})

	t.Run("OnAsk interactive", func(t *testing.T) {
		// This test would require stdin mocking, skipping for now
		t.Skip("Skipping interactive test - requires stdin mocking")
	})
}

func TestScriptingHandler(t *testing.T) {
	t.Run("OnSay filters messages", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewScriptingHandler(&buf, false)

		h.OnSay("text", "Main output", false)
		h.OnSay("command", "ls -la", false) // Should be filtered in non-verbose
		h.OnSay("completion_result", "Done", false)

		output := buf.String()
		if !strings.Contains(output, "Main output") {
			t.Errorf("expected output to contain 'Main output', got %s", output)
		}
		if strings.Contains(output, "ls -la") {
			t.Errorf("expected command to be filtered in non-verbose mode, got %s", output)
		}
		if !strings.Contains(output, "Done") {
			t.Errorf("expected output to contain 'Done', got %s", output)
		}
	})

	t.Run("OnSay verbose includes commands", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewScriptingHandler(&buf, true) // verbose = true

		h.OnSay("command", "ls -la", false)

		output := buf.String()
		if !strings.Contains(output, "ls -la") {
			t.Errorf("expected output to contain command in verbose mode, got %s", output)
		}
	})

	t.Run("OnAsk always auto-approves", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewScriptingHandler(&buf, false)

		response, err := h.OnAsk("command", "Approve?")
		if err != nil {
			t.Fatalf("OnAsk failed: %v", err)
		}

		if response != "yesButtonClicked" {
			t.Errorf("expected 'yesButtonClicked', got %s", response)
		}
	})
}

func TestCreateHandler(t *testing.T) {
	t.Run("JSON handler", func(t *testing.T) {
		var buf bytes.Buffer
		handler := CreateHandler(FormatJSON, &buf, false, false)

		if handler == nil {
			t.Error("expected handler to not be nil")
		}
	})

	t.Run("Plain handler", func(t *testing.T) {
		var buf bytes.Buffer
		handler := CreateHandler(FormatPlain, &buf, false, false)

		if handler == nil {
			t.Error("expected handler to not be nil")
		}
	})

	t.Run("Default to plain", func(t *testing.T) {
		var buf bytes.Buffer
		handler := CreateHandler("unknown", &buf, false, false)

		if handler == nil {
			t.Error("expected handler to not be nil")
		}
	})
}

func TestGetFormatFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected Format
	}{
		{"json", FormatJSON},
		{"jsonl", FormatJSONLines},
		{"jsonlines", FormatJSONLines},
		{"plain", FormatPlain},
		{"text", FormatPlain},
		{"unknown", FormatPlain},
		{"", FormatPlain},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := GetFormatFromString(tt.input)
			if result != tt.expected {
				t.Errorf("GetFormatFromString(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("getString", func(t *testing.T) {
		m := map[string]interface{}{
			"key": "value",
		}
		if getString(m, "key") != "value" {
			t.Error("expected 'value'")
		}
		if getString(m, "missing") != "" {
			t.Error("expected empty string for missing key")
		}
	})

	t.Run("getInt", func(t *testing.T) {
		m := map[string]interface{}{
			"int":    42,
			"int64":  int64(42),
			"float":  42.0,
			"string": "42",
		}

		tests := []struct {
			key      string
			expected int
		}{
			{"int", 42},
			{"int64", 42},
			{"float", 42},
			{"string", 42},
			{"missing", 0},
		}

		for _, tt := range tests {
			if got := getInt(m, tt.key); got != tt.expected {
				t.Errorf("getInt(m, %q) = %d, expected %d", tt.key, got, tt.expected)
			}
		}
	})

	t.Run("calculatePercent", func(t *testing.T) {
		tests := []struct {
			current, total int
			expected       float64
		}{
			{50, 100, 50.0},
			{25, 100, 25.0},
			{0, 100, 0.0},
			{100, 100, 100.0},
			{50, 0, 0.0}, // Avoid division by zero
		}

		for _, tt := range tests {
			got := calculatePercent(tt.current, tt.total)
			if got != tt.expected {
				t.Errorf("calculatePercent(%d, %d) = %f, expected %f", tt.current, tt.total, got, tt.expected)
			}
		}
	})
}
