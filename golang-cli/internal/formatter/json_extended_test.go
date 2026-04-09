// Package formatter provides test coverage for JSON formatting functionality.
// This file focuses on Phase 1 critical fixes: message streaming and partial flag handling.
package formatter

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/cline/cline/golang-cli/internal/task"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJSONOutputFormatMatchesNodeJS tests that JSON output format matches Node.js reference
// This is the acceptance criteria test from Phase 1 implementation plan
func TestJSONOutputFormatMatchesNodeJS(t *testing.T) {
	t.Run("produces correct JSON output with partial flag", func(t *testing.T) {
		var buf bytes.Buffer
		f := NewJSONFormatter(&buf, os.Stderr, false)

		// Format a message with partial flag
		err := f.FormatSayMessage("text", "Hello", true, map[string]interface{}{
			"ts": int64(1234567890),
		})
		require.NoError(t, err)

		// Verify output is valid JSON with partial flag
		output := buf.String()
		var msg JSONMessage
		err = json.Unmarshal([]byte(output), &msg)
		require.NoError(t, err, "output should be valid JSON: %s", output)

		assert.Equal(t, "say", msg.Type)
		assert.Equal(t, "Hello", msg.Text)
		assert.True(t, msg.Partial, "partial flag should be true")
	})

	t.Run("handles streaming JSON lines format", func(t *testing.T) {
		var buf bytes.Buffer
		f := NewJSONFormatter(&buf, os.Stderr, true) // streaming mode

		// Write multiple messages
		err := f.FormatSayMessage("text", "Message 1", false, nil)
		require.NoError(t, err)

		err = f.FormatSayMessage("text", "Message 2", true, nil)
		require.NoError(t, err)

		err = f.FormatSayMessage("text", "Message 3", false, nil)
		require.NoError(t, err)

		// Verify each line is valid JSON
		lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		require.Len(t, lines, 3, "should have 3 JSON lines")

		for i, line := range lines {
			var msg JSONMessage
			err = json.Unmarshal([]byte(line), &msg)
			require.NoError(t, err, "line %d should be valid JSON: %s", i, line)

			assert.Equal(t, "say", msg.Type)
			assert.Equal(t, "text", msg.Say)

			// Check partial flag on second message
			if i == 1 {
				assert.True(t, msg.Partial, "second message should have partial=true")
			} else {
				assert.False(t, msg.Partial, "message %d should have partial=false", i)
			}
		}
	})
}

// TestJSONFormatterFlush tests immediate flushing after each message (Phase 1 requirement)
func TestJSONFormatterFlush(t *testing.T) {
	t.Run("flushes immediately in streaming mode", func(t *testing.T) {
		var buf flushableBuffer
		f := NewJSONFormatter(&buf, os.Stderr, true)

		err := f.FormatSayMessage("text", "Test", false, nil)
		require.NoError(t, err)

		// Verify flush was called
		assert.True(t, buf.flushCalled, "Flush should be called immediately in streaming mode")
	})

	t.Run("flush handles non-flushable writers gracefully", func(t *testing.T) {
		var buf bytes.Buffer // Not flushable
		f := NewJSONFormatter(&buf, os.Stderr, true)

		err := f.FormatSayMessage("text", "Test", false, nil)
		require.NoError(t, err)

		// Should not panic even without flush support
		err = f.Flush()
		assert.NoError(t, err)
	})

	t.Run("flush error propagation", func(t *testing.T) {
		ef := &errorFlushWriter{}
		f := NewJSONFormatter(ef, os.Stderr, true)

		// Format should fail when flush fails
		err := f.FormatSayMessage("text", "Test", false, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "flush failed")
	})
}

// TestFormatSayMessageTypes tests all say message type variations
func TestFormatSayMessageTypes(t *testing.T) {
	tests := []struct {
		name     string
		sayType  string
		text     string
		metadata map[string]interface{}
		wantErr  bool
		validate func(t *testing.T, msg JSONMessage)
	}{
		{
			name:    "api_req_started with metadata",
			sayType: "api_req_started",
			text:    "API request started",
			metadata: map[string]interface{}{
				"apiRequest": map[string]interface{}{
					"requestId": "req-123",
					"model":     "claude-3",
					"tokensIn":  100,
				},
			},
			validate: func(t *testing.T, msg JSONMessage) {
				assert.NotNil(t, msg.APIRequestStarted)
				assert.Equal(t, "req-123", msg.APIRequestStarted.RequestID)
				assert.Equal(t, "claude-3", msg.APIRequestStarted.Model)
				assert.Equal(t, 100, msg.APIRequestStarted.TokensIn)
			},
		},
		{
			name:    "api_req_finished with metadata",
			sayType: "api_req_finished",
			text:    "API request finished",
			metadata: map[string]interface{}{
				"apiRequest": map[string]interface{}{
					"requestId": "req-123",
					"model":     "claude-3",
					"tokensIn":  100,
					"tokensOut": 50,
				},
			},
			validate: func(t *testing.T, msg JSONMessage) {
				assert.NotNil(t, msg.APIRequestFinished)
				assert.Equal(t, "req-123", msg.APIRequestFinished.RequestID)
				assert.Equal(t, 50, msg.APIRequestFinished.TokensOut)
			},
		},
		{
			name:    "command with completion",
			sayType: "command",
			text:    "ls -la",
			metadata: map[string]interface{}{
				"commandCompleted": true,
				"output":           "file1.txt file2.txt",
			},
			validate: func(t *testing.T, msg JSONMessage) {
				assert.True(t, msg.CommandCompleted)
			},
		},
		{
			name:    "checkpoint_created with hash",
			sayType: "checkpoint_created",
			text:    "Checkpoint created",
			metadata: map[string]interface{}{
				"checkpointHash":         "abc123def456",
				"isCheckpointCheckedOut": true,
			},
			validate: func(t *testing.T, msg JSONMessage) {
				assert.Equal(t, "abc123def456", msg.LastCheckpointHash)
				assert.True(t, msg.IsCheckpointCheckedOut)
			},
		},
		{
			name:    "reasoning with content",
			sayType: "reasoning",
			text:    "Let me think...",
			metadata: map[string]interface{}{
				"reasoning": "This is my reasoning process",
			},
			validate: func(t *testing.T, msg JSONMessage) {
				assert.Equal(t, "This is my reasoning process", msg.Reasoning)
			},
		},
		{
			name:    "tool_use with input",
			sayType: "tool_use",
			text:    "Using tool",
			metadata: map[string]interface{}{
				"toolName": "write_file",
				"toolInput": map[string]interface{}{
					"file":    "test.txt",
					"content": "hello world",
				},
			},
			validate: func(t *testing.T, msg JSONMessage) {
				assert.Equal(t, "write_file", msg.ToolName)
				assert.NotNil(t, msg.ToolInput)
			},
		},
		{
			name:    "tool_result with result",
			sayType: "tool_result",
			text:    "Tool executed",
			metadata: map[string]interface{}{
				"toolName":   "write_file",
				"toolResult": "File written successfully",
			},
			validate: func(t *testing.T, msg JSONMessage) {
				assert.Equal(t, "write_file", msg.ToolName)
				assert.Equal(t, "File written successfully", msg.ToolResult)
			},
		},
		{
			name:    "simple text message",
			sayType: "text",
			text:    "Hello World",
			validate: func(t *testing.T, msg JSONMessage) {
				assert.Equal(t, "Hello World", msg.Text)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			f := NewJSONFormatter(&buf, os.Stderr, false)

			err := f.FormatSayMessage(tt.sayType, tt.text, false, tt.metadata)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			var msg JSONMessage
			err = json.Unmarshal(buf.Bytes(), &msg)
			require.NoError(t, err)

			assert.Equal(t, "say", msg.Type)
			assert.Equal(t, tt.sayType, msg.Say)
			assert.Equal(t, tt.text, msg.Text)

			if tt.validate != nil {
				tt.validate(t, msg)
			}
		})
	}
}

// TestFormatAPIRequestMethods tests API request formatting methods
func TestFormatAPIRequestMethods(t *testing.T) {
	t.Run("FormatAPIRequestStarted", func(t *testing.T) {
		var buf bytes.Buffer
		f := NewJSONFormatter(&buf, os.Stderr, false)

		err := f.FormatAPIRequestStarted("req-456", "gpt-4", 200)
		require.NoError(t, err)

		var msg JSONMessage
		err = json.Unmarshal(buf.Bytes(), &msg)
		require.NoError(t, err)

		assert.Equal(t, "say", msg.Type)
		assert.Equal(t, "api_req_started", msg.Say)
		assert.NotNil(t, msg.APIRequestStarted)
		assert.Equal(t, "req-456", msg.APIRequestStarted.RequestID)
		assert.Equal(t, "gpt-4", msg.APIRequestStarted.Model)
		assert.Equal(t, 200, msg.APIRequestStarted.TokensIn)
	})

	t.Run("FormatAPIRequestFinished", func(t *testing.T) {
		var buf bytes.Buffer
		f := NewJSONFormatter(&buf, os.Stderr, false)

		err := f.FormatAPIRequestFinished("req-789", "claude-3-opus", 500, 1000)
		require.NoError(t, err)

		var msg JSONMessage
		err = json.Unmarshal(buf.Bytes(), &msg)
		require.NoError(t, err)

		assert.Equal(t, "say", msg.Type)
		assert.Equal(t, "api_req_finished", msg.Say)
		assert.NotNil(t, msg.APIRequestFinished)
		assert.Equal(t, "req-789", msg.APIRequestFinished.RequestID)
		assert.Equal(t, "claude-3-opus", msg.APIRequestFinished.Model)
		assert.Equal(t, 500, msg.APIRequestFinished.TokensIn)
		assert.Equal(t, 1000, msg.APIRequestFinished.TokensOut)
	})
}

// TestFormatMCPMethods tests MCP message formatting
func TestFormatMCPMethods(t *testing.T) {
	t.Run("FormatMCPRequest", func(t *testing.T) {
		var buf bytes.Buffer
		f := NewJSONFormatter(&buf, os.Stderr, false)

		err := f.FormatMCPRequest("filesystem-server", "read_file")
		require.NoError(t, err)

		var msg JSONMessage
		err = json.Unmarshal(buf.Bytes(), &msg)
		require.NoError(t, err)

		assert.Equal(t, "say", msg.Type)
		assert.Equal(t, "mcp_server_request_started", msg.Say)
		assert.Equal(t, "read_file", msg.Text)
		assert.NotNil(t, msg.Metadata)
		assert.Equal(t, "filesystem-server", msg.Metadata["server"])
	})

	t.Run("FormatMCPResponse", func(t *testing.T) {
		var buf bytes.Buffer
		f := NewJSONFormatter(&buf, os.Stderr, false)

		err := f.FormatMCPResponse("filesystem-server", "File content here")
		require.NoError(t, err)

		var msg JSONMessage
		err = json.Unmarshal(buf.Bytes(), &msg)
		require.NoError(t, err)

		assert.Equal(t, "say", msg.Type)
		assert.Equal(t, "mcp_server_response", msg.Say)
		assert.Equal(t, "File content here", msg.Text)
		assert.NotNil(t, msg.Metadata)
		assert.Equal(t, "filesystem-server", msg.Metadata["server"])
	})
}

// TestFormatBrowserAction tests browser action formatting
func TestFormatBrowserAction(t *testing.T) {
	var buf bytes.Buffer
	f := NewJSONFormatter(&buf, os.Stderr, false)

	err := f.FormatBrowserAction("navigate", "https://example.com", "Page loaded")
	require.NoError(t, err)

	var msg JSONMessage
	err = json.Unmarshal(buf.Bytes(), &msg)
	require.NoError(t, err)

	assert.Equal(t, "say", msg.Type)
	assert.Equal(t, "browser_action", msg.Say)
	assert.Equal(t, "navigate", msg.Text)
	assert.NotNil(t, msg.Metadata)
	assert.Equal(t, "https://example.com", msg.Metadata["url"])
	assert.Equal(t, "Page loaded", msg.Metadata["result"])
}

// TestFormatCompletionResult tests completion result formatting
func TestFormatCompletionResult(t *testing.T) {
	var buf bytes.Buffer
	f := NewJSONFormatter(&buf, os.Stderr, false)

	err := f.FormatCompletionResult("Task completed successfully", map[string]interface{}{
		"success":    true,
		"duration":   "5.2s",
		"tokenUsage": 1500,
	})
	require.NoError(t, err)

	var msg JSONMessage
	err = json.Unmarshal(buf.Bytes(), &msg)
	require.NoError(t, err)

	assert.Equal(t, "say", msg.Type)
	assert.Equal(t, "completion_result", msg.Say)
	assert.Equal(t, "Task completed successfully", msg.Text)
	assert.NotNil(t, msg.Metadata)
	assert.Equal(t, true, msg.Metadata["success"])
}

// TestFormatOutputMethods tests output formatting methods
func TestFormatOutputMethods(t *testing.T) {
	t.Run("FormatErrorOutput", func(t *testing.T) {
		var buf bytes.Buffer
		f := NewJSONFormatter(&buf, os.Stderr, false)

		err := f.FormatErrorOutput(errors.New("something went wrong"))
		require.NoError(t, err)

		var msg JSONMessage
		err = json.Unmarshal(buf.Bytes(), &msg)
		require.NoError(t, err)

		assert.Equal(t, "error", msg.Type)
		assert.Equal(t, "something went wrong", msg.Error)
		assert.NotNil(t, msg.Details)
	})

	t.Run("FormatWarningOutput", func(t *testing.T) {
		var buf bytes.Buffer
		f := NewJSONFormatter(&buf, os.Stderr, false)

		err := f.FormatWarningOutput("This is a warning")
		require.NoError(t, err)

		var msg JSONMessage
		err = json.Unmarshal(buf.Bytes(), &msg)
		require.NoError(t, err)

		assert.Equal(t, "warning", msg.Type)
		assert.Equal(t, "This is a warning", msg.Text)
		assert.False(t, msg.Partial)
	})

	t.Run("FormatInfoOutput", func(t *testing.T) {
		var buf bytes.Buffer
		f := NewJSONFormatter(&buf, os.Stderr, false)

		err := f.FormatInfoOutput("Information message")
		require.NoError(t, err)

		var msg JSONMessage
		err = json.Unmarshal(buf.Bytes(), &msg)
		require.NoError(t, err)

		assert.Equal(t, "info", msg.Type)
		assert.Equal(t, "Information message", msg.Text)
		assert.False(t, msg.Partial)
	})
}

// TestOrderedJSONOutput tests ordered JSON output functionality
func TestOrderedJSONOutput(t *testing.T) {
	t.Run("produces valid JSON", func(t *testing.T) {
		o := NewOrderedJSONOutput()

		msg := JSONMessage{
			Ts:   1234567890,
			Type: "say",
			Text: "Hello",
			Say:  "text",
		}

		err := o.WriteMessage(msg)
		require.NoError(t, err)

		output := o.String()
		assert.Contains(t, output, `"ts"`)
		assert.Contains(t, output, `"type"`)
		assert.Contains(t, output, `"text"`)
		assert.Contains(t, output, `"say"`)
		assert.NotContains(t, output, `"partial"`) // Should not be included when false
	})

	t.Run("produces JSON with partial", func(t *testing.T) {
		o := NewOrderedJSONOutput()

		msg := JSONMessage{
			Ts:      1234567890,
			Type:    "say",
			Text:    "Hello",
			Partial: true,
			Say:     "text",
		}

		err := o.WriteMessage(msg)
		require.NoError(t, err)

		output := o.String()
		assert.Contains(t, output, `"ts"`)
		assert.Contains(t, output, `"type"`)
		assert.Contains(t, output, `"text"`)
		assert.Contains(t, output, `"partial"`) // Should be included when true
		assert.Contains(t, output, `"say"`)
	})

	t.Run("handles empty optional fields", func(t *testing.T) {
		o := NewOrderedJSONOutput()

		msg := JSONMessage{
			Ts:   1234567890,
			Type: "say",
		}

		err := o.WriteMessage(msg)
		require.NoError(t, err)

		output := o.String()
		assert.Contains(t, output, `"ts"`)
		assert.Contains(t, output, `"type"`)
		assert.NotContains(t, output, `"partial"`) // Should not be included when false
	})
}

// TestJSONHandlerExtended tests extended JSON handler functionality
func TestJSONHandlerExtended(t *testing.T) {
	t.Run("HandleMessage routes correctly", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewJSONHandler(&buf, false)

		// Test SAY message
		sayMsg := task.Message{
			Type:    task.MessageTypeSay,
			Content: "Hello",
			Metadata: map[string]interface{}{
				"say_type": "text",
				"partial":  false,
			},
		}
		err := h.HandleMessage(sayMsg)
		require.NoError(t, err)

		// Test ASK message
		buf.Reset()
		askMsg := task.Message{
			Type:    task.MessageTypeAsk,
			Content: "Approve?",
			Metadata: map[string]interface{}{
				"ask_type": "command",
			},
		}
		err = h.HandleMessage(askMsg)
		require.NoError(t, err)

		var msg JSONMessage
		err = json.Unmarshal(buf.Bytes(), &msg)
		require.NoError(t, err)
		assert.Equal(t, "ask", msg.Type)

		// Test ERROR message
		buf.Reset()
		errMsg := task.Message{
			Type:    task.MessageTypeError,
			Content: "Something went wrong",
		}
		err = h.HandleMessage(errMsg)
		require.NoError(t, err)

		err = json.Unmarshal(buf.Bytes(), &msg)
		require.NoError(t, err)
		assert.Equal(t, "error", msg.Type)
	})

	t.Run("OnText passes through to OnSay", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewJSONHandler(&buf, false)

		err := h.OnText("Hello text", false)
		require.NoError(t, err)

		var msg JSONMessage
		err = json.Unmarshal(buf.Bytes(), &msg)
		require.NoError(t, err)
		assert.Equal(t, "text", msg.Say)
	})

	t.Run("OnToolUse auto-approves", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewJSONHandler(&buf, false)

		approved, err := h.OnToolUse("read_file", map[string]interface{}{"file": "test.txt"})
		require.NoError(t, err)
		assert.True(t, approved, "JSON handler should auto-approve in scripting mode")
	})

	t.Run("OnToolResult formats correctly", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewJSONHandler(&buf, false)

		err := h.OnToolResult("write_file", "File written", true)
		require.NoError(t, err)

		var msg JSONMessage
		err = json.Unmarshal(buf.Bytes(), &msg)
		require.NoError(t, err)
		assert.Equal(t, "say", msg.Type)
		assert.Equal(t, "tool_result", msg.Say)
	})

	t.Run("OnCommand requests execution", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewJSONHandler(&buf, false)

		response, err := h.OnCommand("ls -la", true)
		require.NoError(t, err)
		assert.Equal(t, "execute", response)
	})

	t.Run("OnCommandOutput formats output", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewJSONHandler(&buf, false)

		err := h.OnCommandOutput("line 1", false)
		require.NoError(t, err)
	})

	t.Run("OnCheckpoint formats checkpoint", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewJSONHandler(&buf, false)

		err := h.OnCheckpoint("abc123", "created")
		require.NoError(t, err)
	})

	t.Run("OnBrowserAction returns empty", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewJSONHandler(&buf, false)

		result, err := h.OnBrowserAction("navigate", "http://example.com")
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("OnMCPRequest returns empty", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewJSONHandler(&buf, false)

		result, err := h.OnMCPRequest("server1", "tool1", nil)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("OnCompletion formats completion", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewJSONHandler(&buf, false)

		err := h.OnCompletion(true, "Task completed")
		require.NoError(t, err)
	})

	t.Run("OnProgress in verbose mode", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewJSONHandler(&buf, true) // verbose = true

		err := h.OnProgress(50, 100)
		require.NoError(t, err)
	})

	t.Run("OnInfo in verbose mode", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewJSONHandler(&buf, true) // verbose = true

		err := h.OnInfo("Info message")
		require.NoError(t, err)
	})

	t.Run("OnStatus in verbose mode", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewJSONHandler(&buf, true) // verbose = true

		err := h.OnStatus("running")
		require.NoError(t, err)
	})
}

// TestJSONFormatterTimestamp tests timestamp generation
func TestJSONFormatterTimestamp(t *testing.T) {
	var buf bytes.Buffer
	f := NewJSONFormatter(&buf, os.Stderr, false)

	// Format two messages quickly
	err := f.FormatMessage("test", "Message 1", false)
	require.NoError(t, err)

	firstLine := strings.TrimSpace(buf.String())
	buf.Reset()

	err = f.FormatMessage("test", "Message 2", false)
	require.NoError(t, err)

	secondLine := strings.TrimSpace(buf.String())

	var msg1, msg2 JSONMessage
	err = json.Unmarshal([]byte(firstLine), &msg1)
	require.NoError(t, err)
	err = json.Unmarshal([]byte(secondLine), &msg2)
	require.NoError(t, err)

	// Timestamps should be unique and increasing
	assert.NotEqual(t, msg1.Ts, msg2.Ts, "timestamps should be unique")
	assert.Less(t, msg1.Ts, msg2.Ts, "timestamps should increase")
}

// TestJSONFormatterDuplicatePrevention tests duplicate message prevention
func TestJSONFormatterDuplicatePrevention(t *testing.T) {
	var buf bytes.Buffer
	f := NewJSONFormatter(&buf, os.Stderr, false)

	// Manually set a processed timestamp
	f.processedMessages[1000] = true

	// Create message with same timestamp
	msg := JSONMessage{
		Ts:   1000, // This should trigger duplicate detection
		Type: "test",
		Text: "Test message",
	}

	err := f.outputJSON(msg)
	require.NoError(t, err)

	// The message should have been re-timestamped
	output := buf.String()
	var parsed JSONMessage
	err = json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)
	assert.NotEqual(t, int64(1000), parsed.Ts, "timestamp should be regenerated for duplicate")
}

// Helper types for testing

type flushableBuffer struct {
	bytes.Buffer
	flushCalled bool
}

func (f *flushableBuffer) Flush() error {
	f.flushCalled = true
	return nil
}

type errorFlushWriter struct {
	flushCalled bool
}

func (e *errorFlushWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (e *errorFlushWriter) Flush() error {
	e.flushCalled = true
	return errors.New("flush failed")
}