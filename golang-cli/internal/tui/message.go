// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"fmt"
	"time"
)

// MessageType represents the type of chat message.
type MessageType string

const (
	// MessageTypeSay represents a message from the AI assistant.
	MessageTypeSay MessageType = "say"
	// MessageTypeAsk represents a question from the AI assistant to the user.
	MessageTypeAsk MessageType = "ask"
	// MessageTypeText represents plain text content.
	MessageTypeText MessageType = "text"
	// MessageTypeToolUse represents a tool invocation.
	MessageTypeToolUse MessageType = "tool_use"
	// MessageTypeToolResult represents the result of a tool invocation.
	MessageTypeToolResult MessageType = "tool_result"
	// MessageTypeError represents an error message.
	MessageTypeError MessageType = "error"
	// MessageTypeUser represents a message from the user.
	MessageTypeUser MessageType = "user"
)

// Message represents a chat message in the conversation.
type Message struct {
	// ID is the unique identifier for the message.
	ID string
	// Type is the type of message.
	Type MessageType
	// Content is the message content.
	Content string
	// Partial indicates if this is a partial/streaming message.
	Partial bool
	// Timestamp is when the message was created.
	Timestamp time.Time
	// Metadata contains additional type-specific data.
	Metadata map[string]interface{}
	// ToolName is set for tool_use messages.
	ToolName string
	// ToolInput is set for tool_use messages.
	ToolInput map[string]interface{}
	// ToolResult is set for tool_result messages.
	ToolResult string
	// Language is set for code blocks.
	Language string
}

// NewMessage creates a new message with the given type and content.
func NewMessage(msgType MessageType, content string) *Message {
	return &Message{
		ID:        generateID(),
		Type:      msgType,
		Content:   content,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
}

// NewUserMessage creates a new user message.
func NewUserMessage(content string) *Message {
	return NewMessage(MessageTypeUser, content)
}

// NewAIMessage creates a new AI assistant message.
func NewAIMessage(content string) *Message {
	return NewMessage(MessageTypeSay, content)
}

// NewErrorMessage creates a new error message.
func NewErrorMessage(content string) *Message {
	return NewMessage(MessageTypeError, content)
}

// NewToolUseMessage creates a new tool use message.
func NewToolUseMessage(toolName string, toolInput map[string]interface{}) *Message {
	msg := NewMessage(MessageTypeToolUse, "")
	msg.ToolName = toolName
	msg.ToolInput = toolInput
	return msg
}

// NewToolResultMessage creates a new tool result message.
func NewToolResultMessage(toolName, result string) *Message {
	msg := NewMessage(MessageTypeToolResult, "")
	msg.ToolName = toolName
	msg.ToolResult = result
	return msg
}

// SetPartial marks the message as partial/streaming.
func (m *Message) SetPartial(partial bool) *Message {
	m.Partial = partial
	return m
}

// SetLanguage sets the language for code blocks.
func (m *Message) SetLanguage(lang string) *Message {
	m.Language = lang
	return m
}

// SetMetadata sets a metadata value.
func (m *Message) SetMetadata(key string, value interface{}) *Message {
	m.Metadata[key] = value
	return m
}

// GetMetadata gets a metadata value.
func (m *Message) GetMetadata(key string) (interface{}, bool) {
	val, ok := m.Metadata[key]
	return val, ok
}

// IsFromUser returns true if the message is from the user.
func (m *Message) IsFromUser() bool {
	return m.Type == MessageTypeUser
}

// IsFromAI returns true if the message is from the AI assistant.
func (m *Message) IsFromAI() bool {
	return m.Type == MessageTypeSay || m.Type == MessageTypeAsk
}

// IsToolRelated returns true if the message is tool-related.
func (m *Message) IsToolRelated() bool {
	return m.Type == MessageTypeToolUse || m.Type == MessageTypeToolResult
}

// String returns a string representation of the message.
func (m *Message) String() string {
	return fmt.Sprintf("[%s] %s: %s", m.Timestamp.Format("15:04:05"), m.Type, m.Content)
}

// generateID generates a simple unique ID for messages.
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// MessageStore provides storage and retrieval of messages.
type MessageStore struct {
	messages []*Message
}

// NewMessageStore creates a new message store.
func NewMessageStore() *MessageStore {
	return &MessageStore{
		messages: make([]*Message, 0),
	}
}

// Add adds a message to the store.
func (s *MessageStore) Add(msg *Message) {
	s.messages = append(s.messages, msg)
}

// Get returns the message at the given index.
func (s *MessageStore) Get(index int) (*Message, bool) {
	if index < 0 || index >= len(s.messages) {
		return nil, false
	}
	return s.messages[index], true
}

// GetLast returns the last message in the store.
func (s *MessageStore) GetLast() (*Message, bool) {
	if len(s.messages) == 0 {
		return nil, false
	}
	return s.messages[len(s.messages)-1], true
}

// UpdateLast updates the last message in the store.
func (s *MessageStore) UpdateLast(msg *Message) bool {
	if len(s.messages) == 0 {
		return false
	}
	s.messages[len(s.messages)-1] = msg
	return true
}

// GetAll returns all messages.
func (s *MessageStore) GetAll() []*Message {
	return s.messages
}

// Len returns the number of messages.
func (s *MessageStore) Len() int {
	return len(s.messages)
}

// Clear removes all messages.
func (s *MessageStore) Clear() {
	s.messages = make([]*Message, 0)
}

// Filter returns messages matching the given type.
func (s *MessageStore) Filter(msgType MessageType) []*Message {
	var result []*Message
	for _, msg := range s.messages {
		if msg.Type == msgType {
			result = append(result, msg)
		}
	}
	return result
}