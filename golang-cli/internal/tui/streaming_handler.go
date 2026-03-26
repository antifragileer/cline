// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"fmt"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// MessageHandler handles messages from the task runner.
type MessageHandler interface {
	// OnSay handles a SAY message from the assistant
	OnSay(sayType string, text string, partial bool)
	// OnAsk handles an ASK message that requires user response
	OnAsk(askType string, text string) (string, error)
	// OnInfo handles informational messages
	OnInfo(text string)
	// OnError handles error messages
	OnError(err error)
}

// StreamingHandler handles real-time message streaming from gRPC to the TUI.
// It acts as a bridge between the gRPC stream and the Bubble Tea program.
type StreamingHandler struct {
	program      *tea.Program
	mu           sync.RWMutex
	messages     []Message
	streamingMsg *Message
	isStreaming  bool
	taskID       string
}

// NewStreamingHandler creates a new streaming handler
func NewStreamingHandler() *StreamingHandler {
	return &StreamingHandler{
		messages: make([]Message, 0),
	}
}

// SetProgram sets the Bubble Tea program for sending messages
func (h *StreamingHandler) SetProgram(program *tea.Program) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.program = program
}

// SetTaskID sets the current task ID
func (h *StreamingHandler) SetTaskID(taskID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.taskID = taskID
}

// OnSay handles a SAY message from the assistant
func (h *StreamingHandler) OnSay(sayType string, text string, partial bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.program == nil {
		return
	}

	msgType := h.mapSayTypeToMessageType(sayType)

	if partial {
		// Update or create streaming message
		if h.streamingMsg == nil {
			h.streamingMsg = &Message{
				Type:      msgType,
				Content:   text,
				Timestamp: time.Now(),
				Partial:   true,
			}
		} else {
			h.streamingMsg.Content = text
			h.streamingMsg.Type = msgType
		}
		h.isStreaming = true

		// Send stream chunk to TUI
		h.program.Send(StreamChunkMsg{
			Chunk: text,
			Done:  false,
		})
	} else {
		// Finalize streaming message or add new message
		if h.streamingMsg != nil && h.isStreaming {
			// Finalize the streaming message
			h.streamingMsg.Partial = false
			h.streamingMsg.Content = text
			h.messages = append(h.messages, *h.streamingMsg)
			h.streamingMsg = nil
			h.isStreaming = false
		} else {
			// Add as new message
			h.messages = append(h.messages, Message{
				Type:      msgType,
				Content:   text,
				Timestamp: time.Now(),
				Partial:   false,
			})
		}

		// Send update to TUI
		h.program.Send(ChatUpdateMsg{
			Messages: h.GetMessages(),
		})
	}
}

// OnAsk handles an ASK message that requires user response
func (h *StreamingHandler) OnAsk(askType string, text string) (string, error) {
	h.mu.Lock()
	if h.program == nil {
		h.mu.Unlock()
		return "yesButtonClicked", nil // Auto-approve if no TUI
	}
	h.mu.Unlock()

	// Finalize any streaming message first
	if h.isStreaming {
		h.OnSay("text", h.streamingMsg.Content, false)
	}

	// Map ask type to message type
	var msgType MessageType
	switch askType {
	case "command":
		msgType = MessageTypeAsk
	case "tool":
		msgType = MessageTypeToolUse
	case "browser_action_launch":
		msgType = MessageTypeAsk
	case "completion_result":
		msgType = MessageTypeSay
	case "followup":
		msgType = MessageTypeAsk
	default:
		msgType = MessageTypeAsk
	}

	// Add the ask message
	h.mu.Lock()
	h.messages = append(h.messages, Message{
		Type:      msgType,
		Content:   text,
		Timestamp: time.Now(),
		Partial:   false,
		Metadata: map[string]interface{}{
			"askType": askType,
		},
	})
	h.mu.Unlock()

	// Send update to TUI
	h.program.Send(ChatUpdateMsg{
		Messages: h.GetMessages(),
	})

	// Send approval request to TUI
	responseChan := make(chan string, 1)
	h.program.Send(ApprovalRequestMsg{
		AskType:  askType,
		Text:     text,
		Response: responseChan,
	})

	// Wait for response with timeout
	select {
	case response := <-responseChan:
		return response, nil
	case <-time.After(30 * time.Minute): // Long timeout for user interaction
		return "noButtonClicked", fmt.Errorf("approval timeout")
	}
}

// OnInfo handles informational messages
func (h *StreamingHandler) OnInfo(text string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.program == nil {
		return
	}

	h.messages = append(h.messages, Message{
		Type:      MessageTypeText,
		Content:   text,
		Timestamp: time.Now(),
		Partial:   false,
	})

	h.program.Send(ChatUpdateMsg{
		Messages: h.GetMessages(),
	})
}

// OnError handles error messages
func (h *StreamingHandler) OnError(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.program == nil {
		return
	}

	h.messages = append(h.messages, Message{
		Type:      MessageTypeError,
		Content:   err.Error(),
		Timestamp: time.Now(),
		Partial:   false,
	})

	h.program.Send(ChatErrorMsg{
		Error: err,
	})
}

// OnStatus handles status updates
func (h *StreamingHandler) OnStatus(status string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.program == nil {
		return
	}

	// Status updates don't add messages, just update status
	h.program.Send(StatusUpdateMsg{
		Status: status,
	})
}

// OnProgress handles progress updates
func (h *StreamingHandler) OnProgress(current, total int) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.program == nil {
		return
	}

	h.program.Send(ProgressUpdateMsg{
		Current: current,
		Total:   total,
	})
}

// GetMessages returns all messages (thread-safe)
func (h *StreamingHandler) GetMessages() []Message {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Return a copy
	result := make([]Message, len(h.messages))
	copy(result, h.messages)

	// Include streaming message if active
	if h.streamingMsg != nil && h.isStreaming {
		result = append(result, *h.streamingMsg)
	}

	return result
}

// GetTaskID returns the current task ID
func (h *StreamingHandler) GetTaskID() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.taskID
}

// Clear clears all messages
func (h *StreamingHandler) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.messages = make([]Message, 0)
	h.streamingMsg = nil
	h.isStreaming = false
}

// mapSayTypeToMessageType maps say types to message types
func (h *StreamingHandler) mapSayTypeToMessageType(sayType string) MessageType {
	switch sayType {
	case "text":
		return MessageTypeSay
	case "error":
		return MessageTypeError
	case "command":
		return MessageTypeToolUse
	case "command_output":
		return MessageTypeToolResult
	case "tool":
		return MessageTypeToolUse
	case "completion_result":
		return MessageTypeSay
	case "user_feedback":
		return MessageTypeUser
	case "thinking", "reasoning":
		return MessageTypeSay
	default:
		return MessageTypeSay
	}
}

// ChatUpdateMsg is sent when chat messages are updated.
type ChatUpdateMsg struct {
	Messages []Message
}

// StreamChunkMsg is sent during streaming.
type StreamChunkMsg struct {
	Chunk string
	Done  bool
}

// ChatErrorMsg is sent when an error occurs.
type ChatErrorMsg struct {
	Error error
}

// ApprovalRequestMsg is sent when an approval is requested
type ApprovalRequestMsg struct {
	AskType  string
	Text     string
	Response chan<- string
}

// StatusUpdateMsg is sent for status updates
type StatusUpdateMsg struct {
	Status string
}

// ProgressUpdateMsg is sent for progress updates
type ProgressUpdateMsg struct {
	Current int
	Total   int
}

// StreamStateMsg is sent when the stream state changes
type StreamStateMsg struct {
	State interface{} // StreamState from host package
}

// StreamMessageMsg is sent when a message is received from the stream
type StreamMessageMsg struct {
	Message Message
}
