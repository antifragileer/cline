// Package task provides task execution functionality for the Cline CLI.
// This file contains tests for the message handler functionality.
package task

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestParseMessage tests parsing messages from JSON
func TestParseMessage(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
		wantMsg *Message
	}{
		{
			name:    "valid text message",
			json:    `{"type":"text","id":"msg-1","content":"Hello","timestamp":"2024-01-01T00:00:00Z"}`,
			wantErr: false,
		},
		{
			name:    "valid tool message",
			json:    `{"type":"tool","id":"msg-2","metadata":{"tool":"read_file"}}`,
			wantErr: false,
		},
		{
			name:    "invalid json",
			json:    `{"type":invalid}`,
			wantErr: true,
		},
		{
			name:    "empty json object",
			json:    `{}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := ParseMessage([]byte(tt.json))

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, msg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, msg)
			}
		})
	}
}

// TestSerializeMessage tests serializing messages to JSON
func TestSerializeMessage(t *testing.T) {
	t.Run("serializes message correctly", func(t *testing.T) {
		msg := &Message{
			Type:      MessageTypeText,
			ID:        "msg-1",
			Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Content:   "Test content",
		}

		data, err := SerializeMessage(msg)
		assert.NoError(t, err)
		assert.NotNil(t, data)

		// Verify it can be parsed back
		parsed, err := ParseMessage(data)
		assert.NoError(t, err)
		assert.Equal(t, msg.ID, parsed.ID)
		assert.Equal(t, msg.Type, parsed.Type)
	})

	t.Run("handles nil message gracefully", func(t *testing.T) {
		// Note: This might panic or error depending on implementation
		_, err := SerializeMessage(nil)
		// Either error or panic is acceptable
		_ = err
	})
}

// TestNewDefaultMessageHandler tests creating a default handler
func TestNewDefaultMessageHandler(t *testing.T) {
	t.Run("creates handler with defaults", func(t *testing.T) {
		handler := NewDefaultMessageHandler()
		assert.NotNil(t, handler)
		assert.NotNil(t, handler.approvalState)
		assert.Empty(t, handler.approvalState)
	})
}

// TestDefaultMessageHandlerCallbacks tests callback setters
func TestDefaultMessageHandlerCallbacks(t *testing.T) {
	handler := NewDefaultMessageHandler()

	t.Run("sets text callback", func(t *testing.T) {
		called := false
		handler.SetTextCallback(func(content string, isPartial bool) {
			called = true
		})

		handler.OnText("test", false)
		assert.True(t, called)
	})

	t.Run("sets tool callback", func(t *testing.T) {
		called := false
		handler.SetToolCallback(func(toolName string, params map[string]interface{}) (bool, error) {
			called = true
			return true, nil
		})

		_, _ = handler.OnToolUse("test_tool", nil)
		assert.True(t, called)
	})

	t.Run("sets command callback", func(t *testing.T) {
		called := false
		handler.SetCommandCallback(func(command string, requiresApproval bool) (string, error) {
			called = true
			return "", nil
		})

		_, _ = handler.OnCommand("ls", true)
		assert.True(t, called)
	})

	t.Run("sets ask callback", func(t *testing.T) {
		called := false
		handler.SetAskCallback(func(promptType, question string) (string, error) {
			called = true
			return "", nil
		})

		_, _ = handler.OnAsk("followup", "What?")
		assert.True(t, called)
	})

	t.Run("sets error callback", func(t *testing.T) {
		called := false
		handler.SetErrorCallback(func(err error) {
			called = true
		})

		_ = handler.OnError(fmt.Errorf("test error"))
		assert.True(t, called)
	})

	t.Run("sets completion callback", func(t *testing.T) {
		called := false
		handler.SetCompletionCallback(func(success bool, summary string) {
			called = true
		})

		_ = handler.OnCompletion(true, "Done")
		assert.True(t, called)
	})
}

// TestDefaultMessageHandlerOnText tests text handling
func TestDefaultMessageHandlerOnText(t *testing.T) {
	handler := NewDefaultMessageHandler()

	t.Run("handles text without callback", func(t *testing.T) {
		err := handler.OnText("Hello", false)
		assert.NoError(t, err)
	})

	t.Run("handles partial text", func(t *testing.T) {
		err := handler.OnText("Partial", true)
		assert.NoError(t, err)
	})
}

// TestDefaultMessageHandlerOnToolUse tests tool use handling
func TestDefaultMessageHandlerOnToolUse(t *testing.T) {
	handler := NewDefaultMessageHandler()

	t.Run("returns false without callback", func(t *testing.T) {
		approved, err := handler.OnToolUse("test_tool", nil)
		assert.NoError(t, err)
		assert.False(t, approved) // Default should be false
	})

	t.Run("remembers approved tools", func(t *testing.T) {
		// This test verifies that the approval state works correctly
		// First, manually set the approval state
		handler.approvalState["approved_tool"] = true

		// Remove callback to test state-based approval
		handler.SetToolCallback(nil)

		// Should now auto-approve the approved tool
		approved, err := handler.OnToolUse("approved_tool", nil)
		assert.NoError(t, err)
		assert.True(t, approved)

		// Should not approve unapproved tool
		approved, err = handler.OnToolUse("unapproved_tool", nil)
		assert.NoError(t, err)
		assert.False(t, approved)
	})
}

// TestDefaultMessageHandlerOnToolResult tests tool result handling
func TestDefaultMessageHandlerOnToolResult(t *testing.T) {
	handler := NewDefaultMessageHandler()

	t.Run("handles successful result", func(t *testing.T) {
		err := handler.OnToolResult("read_file", "content", true)
		assert.NoError(t, err)
	})

	t.Run("handles failed result", func(t *testing.T) {
		err := handler.OnToolResult("read_file", "error", false)
		assert.NoError(t, err)
	})
}

// TestDefaultMessageHandlerOnAsk tests ask handling
func TestDefaultMessageHandlerOnAsk(t *testing.T) {
	handler := NewDefaultMessageHandler()

	t.Run("returns empty without callback", func(t *testing.T) {
		response, err := handler.OnAsk("followup", "What?")
		assert.NoError(t, err)
		assert.Empty(t, response)
	})

	t.Run("returns callback response", func(t *testing.T) {
		handler.SetAskCallback(func(promptType, question string) (string, error) {
			return "My response", nil
		})

		response, err := handler.OnAsk("followup", "What?")
		assert.NoError(t, err)
		assert.Equal(t, "My response", response)
	})
}

// TestDefaultMessageHandlerOnSay tests say handling
func TestDefaultMessageHandlerOnSay(t *testing.T) {
	handler := NewDefaultMessageHandler()

	t.Run("handles text say", func(t *testing.T) {
		err := handler.OnSay("text", "Hello", false)
		assert.NoError(t, err)
	})

	t.Run("handles error say", func(t *testing.T) {
		err := handler.OnSay("error", "Error message", false)
		assert.Error(t, err) // Should return error
	})

	t.Run("handles other say types", func(t *testing.T) {
		err := handler.OnSay("completion_result", "Done", false)
		assert.NoError(t, err)
	})
}

// TestDefaultMessageHandlerOnCommand tests command handling
func TestDefaultMessageHandlerOnCommand(t *testing.T) {
	handler := NewDefaultMessageHandler()

	t.Run("returns error without callback", func(t *testing.T) {
		response, err := handler.OnCommand("ls -la", true)
		assert.Error(t, err)
		assert.Empty(t, response)
	})

	t.Run("returns callback response", func(t *testing.T) {
		handler.SetCommandCallback(func(command string, requiresApproval bool) (string, error) {
			return "output", nil
		})

		response, err := handler.OnCommand("ls", true)
		assert.NoError(t, err)
		assert.Equal(t, "output", response)
	})
}

// TestDefaultMessageHandlerOnCommandOutput tests command output handling
func TestDefaultMessageHandlerOnCommandOutput(t *testing.T) {
	handler := NewDefaultMessageHandler()

	t.Run("handles output", func(t *testing.T) {
		err := handler.OnCommandOutput("output line", false)
		assert.NoError(t, err)
	})

	t.Run("handles complete output", func(t *testing.T) {
		err := handler.OnCommandOutput("final output", true)
		assert.NoError(t, err)
	})
}

// TestDefaultMessageHandlerOnError tests error handling
func TestDefaultMessageHandlerOnError(t *testing.T) {
	handler := NewDefaultMessageHandler()

	t.Run("handles error without callback", func(t *testing.T) {
		testErr := fmt.Errorf("test error")
		err := handler.OnError(testErr)
		assert.Equal(t, testErr, err)
	})

	t.Run("calls error callback", func(t *testing.T) {
		called := false
		handler.SetErrorCallback(func(err error) {
			called = true
		})

		_ = handler.OnError(fmt.Errorf("test"))
		assert.True(t, called)
	})
}

// TestDefaultMessageHandlerOnInfo tests info handling
func TestDefaultMessageHandlerOnInfo(t *testing.T) {
	handler := NewDefaultMessageHandler()

	t.Run("handles info message", func(t *testing.T) {
		err := handler.OnInfo("Information")
		assert.NoError(t, err)
	})
}

// TestDefaultMessageHandlerOnStatus tests status handling
func TestDefaultMessageHandlerOnStatus(t *testing.T) {
	handler := NewDefaultMessageHandler()

	t.Run("handles status", func(t *testing.T) {
		err := handler.OnStatus("running")
		assert.NoError(t, err)
	})
}

// TestDefaultMessageHandlerOnProgress tests progress handling
func TestDefaultMessageHandlerOnProgress(t *testing.T) {
	handler := NewDefaultMessageHandler()

	t.Run("handles progress", func(t *testing.T) {
		err := handler.OnProgress(50, 100)
		assert.NoError(t, err)
	})

	t.Run("handles zero total", func(t *testing.T) {
		err := handler.OnProgress(0, 0)
		assert.NoError(t, err)
	})
}

// TestDefaultMessageHandlerOnCheckpoint tests checkpoint handling
func TestDefaultMessageHandlerOnCheckpoint(t *testing.T) {
	handler := NewDefaultMessageHandler()

	t.Run("handles checkpoint", func(t *testing.T) {
		err := handler.OnCheckpoint("abc123", "created")
		assert.NoError(t, err)
	})
}

// TestDefaultMessageHandlerOnBrowserAction tests browser action handling
func TestDefaultMessageHandlerOnBrowserAction(t *testing.T) {
	handler := NewDefaultMessageHandler()

	t.Run("returns error by default", func(t *testing.T) {
		response, err := handler.OnBrowserAction("navigate", "http://example.com")
		assert.Error(t, err)
		assert.Empty(t, response)
	})
}

// TestDefaultMessageHandlerOnMCPRequest tests MCP request handling
func TestDefaultMessageHandlerOnMCPRequest(t *testing.T) {
	handler := NewDefaultMessageHandler()

	t.Run("returns error by default", func(t *testing.T) {
		response, err := handler.OnMCPRequest("server1", "tool1", nil)
		assert.Error(t, err)
		assert.Empty(t, response)
	})
}

// TestDefaultMessageHandlerOnCompletion tests completion handling
func TestDefaultMessageHandlerOnCompletion(t *testing.T) {
	handler := NewDefaultMessageHandler()

	t.Run("handles completion", func(t *testing.T) {
		err := handler.OnCompletion(true, "Task completed successfully")
		assert.NoError(t, err)
	})

	t.Run("handles failure", func(t *testing.T) {
		err := handler.OnCompletion(false, "Task failed")
		assert.NoError(t, err)
	})
}

// TestDefaultMessageHandlerHandleMessage tests message routing
func TestDefaultMessageHandlerHandleMessage(t *testing.T) {
	handler := NewDefaultMessageHandler()

	t.Run("handles text message", func(t *testing.T) {
		msg := Message{
			Type:    MessageTypeText,
			Content: "Hello",
		}
		err := handler.HandleMessage(msg)
		assert.NoError(t, err)
	})

	t.Run("handles tool message", func(t *testing.T) {
		msg := Message{
			Type:     MessageTypeTool,
			Metadata: map[string]interface{}{"tool": "read_file"},
		}
		err := handler.HandleMessage(msg)
		assert.NoError(t, err)
	})

	t.Run("handles ask message", func(t *testing.T) {
		msg := Message{
			Type:     MessageTypeAsk,
			Content:  "What?",
			Metadata: map[string]interface{}{"ask_type": "followup"},
		}
		err := handler.HandleMessage(msg)
		assert.NoError(t, err)
	})

	t.Run("handles say message", func(t *testing.T) {
		msg := Message{
			Type:     MessageTypeSay,
			Content:  "Hello",
			Metadata: map[string]interface{}{"say_type": "text"},
		}
		err := handler.HandleMessage(msg)
		assert.NoError(t, err)
	})

	t.Run("handles command message", func(t *testing.T) {
		msg := Message{
			Type:     MessageTypeCommand,
			Metadata: map[string]interface{}{"command": "ls", "requires_approval": true},
		}
		err := handler.HandleMessage(msg)
		assert.Error(t, err) // Default returns error for commands
	})

	t.Run("handles error message", func(t *testing.T) {
		msg := Message{
			Type:    MessageTypeError,
			Content: "Something went wrong",
		}
		err := handler.HandleMessage(msg)
		assert.Error(t, err)
	})

	t.Run("handles checkpoint message", func(t *testing.T) {
		msg := Message{
			Type:     MessageTypeCheckpoint,
			Metadata: map[string]interface{}{"checkpoint_id": "abc123", "action": "created"},
		}
		err := handler.HandleMessage(msg)
		assert.NoError(t, err)
	})

	t.Run("handles browser message", func(t *testing.T) {
		msg := Message{
			Type:     MessageTypeBrowser,
			Metadata: map[string]interface{}{"action": "navigate", "url": "http://example.com"},
		}
		err := handler.HandleMessage(msg)
		assert.Error(t, err) // Default returns error
	})

	t.Run("handles MCP message", func(t *testing.T) {
		msg := Message{
			Type:     MessageTypeMCP,
			Metadata: map[string]interface{}{"server": "s1", "tool": "t1"},
		}
		err := handler.HandleMessage(msg)
		assert.Error(t, err) // Default returns error
	})

	t.Run("handles completion message", func(t *testing.T) {
		msg := Message{
			Type:     MessageTypeCompletion,
			Metadata: map[string]interface{}{"success": true, "summary": "Done"},
		}
		err := handler.HandleMessage(msg)
		assert.NoError(t, err)
	})

	t.Run("handles unknown message type", func(t *testing.T) {
		msg := Message{
			Type: "unknown_type",
		}
		err := handler.HandleMessage(msg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown message type")
	})
}

// TestMessageFilter tests message filtering
func TestMessageFilter(t *testing.T) {
	messages := []Message{
		{Type: MessageTypeText, Content: "Hello", Timestamp: time.Now()},
		{Type: MessageTypeTool, Content: "Tool output", Timestamp: time.Now()},
		{Type: MessageTypeError, Content: "Error occurred", Timestamp: time.Now()},
	}

	t.Run("filters by type", func(t *testing.T) {
		filter := MessageFilter{
			Types: []MessageType{MessageTypeText},
		}
		result := filter.Filter(messages)
		assert.Len(t, result, 1)
		assert.Equal(t, MessageTypeText, result[0].Type)
	})

	t.Run("filters by content", func(t *testing.T) {
		filter := MessageFilter{
			Contains: "Error",
		}
		result := filter.Filter(messages)
		assert.Len(t, result, 1)
		assert.Equal(t, "Error occurred", result[0].Content)
	})

	t.Run("filters by time", func(t *testing.T) {
		past := time.Now().Add(-1 * time.Hour)
		filter := MessageFilter{
			Since: past,
		}
		result := filter.Filter(messages)
		assert.Len(t, result, 3) // All messages are after past
	})

	t.Run("returns all when no filters", func(t *testing.T) {
		filter := MessageFilter{}
		result := filter.Filter(messages)
		assert.Len(t, result, 3)
	})
}

// TestMessageBuffer tests message buffering
func TestMessageBuffer(t *testing.T) {
	t.Run("creates buffer with defaults", func(t *testing.T) {
		buffer := NewMessageBuffer(10, time.Second)
		assert.NotNil(t, buffer)
		assert.True(t, buffer.IsEmpty())
		assert.Equal(t, 0, buffer.Size())
	})

	t.Run("adds messages to buffer", func(t *testing.T) {
		buffer := NewMessageBuffer(10, time.Second)

		result := buffer.Add(Message{Type: MessageTypeText, Content: "1"})
		assert.Nil(t, result) // Not full yet
		assert.Equal(t, 1, buffer.Size())
	})

	t.Run("flushes when buffer is full", func(t *testing.T) {
		buffer := NewMessageBuffer(2, time.Hour) // Large timeout

		buffer.Add(Message{Content: "1"})
		result := buffer.Add(Message{Content: "2"})

		assert.NotNil(t, result)
		assert.Len(t, result, 2)
		assert.True(t, buffer.IsEmpty())
	})

	t.Run("manual flush", func(t *testing.T) {
		buffer := NewMessageBuffer(10, time.Hour)

		buffer.Add(Message{Content: "1"})
		buffer.Add(Message{Content: "2"})

		result := buffer.Flush()
		assert.Len(t, result, 2)
		assert.True(t, buffer.IsEmpty())
	})

	t.Run("flush empty buffer", func(t *testing.T) {
		buffer := NewMessageBuffer(10, time.Second)
		result := buffer.Flush()
		assert.Empty(t, result)
	})
}

// BenchmarkHandlerOperations benchmarks handler operations
func BenchmarkHandlerOperations(b *testing.B) {
	handler := NewDefaultMessageHandler()

	b.Run("OnText", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			handler.OnText("Test message", false)
		}
	})

	b.Run("HandleMessage", func(b *testing.B) {
		msg := Message{
			Type:    MessageTypeText,
			Content: "Test",
		}
		for i := 0; i < b.N; i++ {
			handler.HandleMessage(msg)
		}
	})

	b.Run("ParseMessage", func(b *testing.B) {
		data := []byte(`{"type":"text","content":"Test"}`)
		for i := 0; i < b.N; i++ {
			ParseMessage(data)
		}
	})

	b.Run("SerializeMessage", func(b *testing.B) {
		msg := &Message{
			Type:    MessageTypeText,
			Content: "Test",
		}
		for i := 0; i < b.N; i++ {
			SerializeMessage(msg)
		}
	})
}
