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

// MessageStatus represents the status of a message
type MessageStatus string

const (
	// MessageStatusPending indicates the message is waiting to be processed
	MessageStatusPending MessageStatus = "pending"
	// MessageStatusStreaming indicates the message is being streamed
	MessageStatusStreaming MessageStatus = "streaming"
	// MessageStatusComplete indicates the message is complete
	MessageStatusComplete MessageStatus = "complete"
	// MessageStatusFailed indicates the message failed
	MessageStatusFailed MessageStatus = "failed"
	// MessageStatusCancelled indicates the message was cancelled
	MessageStatusCancelled MessageStatus = "cancelled"
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
	// Status is the current status of the message
	Status MessageStatus
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
	// SayType is the subtype for say messages (e.g., "text", "tool", "command").
	SayType string
	// AskType is the subtype for ask messages (e.g., "tool", "command", "followup").
	AskType string
	// HasOutput indicates if this message has output content (for commands).
	HasOutput bool
	// CommandCompleted indicates if a command has completed execution.
	CommandCompleted bool
	// SequenceNum is the sequence number for ordering
	SequenceNum int
	// ParentID is the ID of the parent message (for threaded conversations)
	ParentID string
	// IsEdited indicates if the message has been edited
	IsEdited bool
	// EditTimestamp is when the message was last edited
	EditTimestamp time.Time
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

// GetKey returns a unique key for this message for static rendering.
func (m *Message) GetKey() string {
	if m.ID != "" {
		return m.ID
	}
	if m.Timestamp.UnixNano() > 0 {
		return fmt.Sprintf("%d", m.Timestamp.UnixNano())
	}
	return generateID()
}

// SetSayType sets the say type for the message.
func (m *Message) SetSayType(sayType string) *Message {
	m.SayType = sayType
	return m
}

// SetAskType sets the ask type for the message.
func (m *Message) SetAskType(askType string) *Message {
	m.AskType = askType
	return m
}

// SetHasOutput sets whether this message has output.
func (m *Message) SetHasOutput(hasOutput bool) *Message {
	m.HasOutput = hasOutput
	return m
}

// SetCommandCompleted sets whether a command has completed.
func (m *Message) SetCommandCompleted(completed bool) *Message {
	m.CommandCompleted = completed
	return m
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

// FindByID finds a message by its ID.
func (s *MessageStore) FindByID(id string) (*Message, bool) {
	for _, msg := range s.messages {
		if msg.ID == id {
			return msg, true
		}
	}
	return nil, false
}

// Update updates an existing message by ID.
func (s *MessageStore) Update(id string, updater func(*Message)) bool {
	for _, msg := range s.messages {
		if msg.ID == id {
			updater(msg)
			return true
		}
	}
	return false
}

// GetHistory returns the message history formatted for display.
func (s *MessageStore) GetHistory(limit int) []*Message {
	if limit <= 0 || limit > len(s.messages) {
		return s.messages
	}
	return s.messages[len(s.messages)-limit:]
}

// GetMessagesByStatus returns messages with a specific status.
func (s *MessageStore) GetMessagesByStatus(status MessageStatus) []*Message {
	var result []*Message
	for _, msg := range s.messages {
		if msg.Status == status {
			result = append(result, msg)
		}
	}
	return result
}

// GetLastByType returns the most recent message of a specific type.
func (s *MessageStore) GetLastByType(msgType MessageType) (*Message, bool) {
	for i := len(s.messages) - 1; i >= 0; i-- {
		if s.messages[i].Type == msgType {
			return s.messages[i], true
		}
	}
	return nil, false
}

// UpdateMessageStatus updates the status of a message by ID.
func (s *MessageStore) UpdateMessageStatus(id string, status MessageStatus) bool {
	return s.Update(id, func(msg *Message) {
		msg.Status = status
		if status == MessageStatusComplete || status == MessageStatusFailed {
			msg.Partial = false
		}
	})
}

// GetMessageCount returns the total number of messages.
func (s *MessageStore) GetMessageCount() int {
	return len(s.messages)
}

// GetMessageCountByType returns the count of messages by type.
func (s *MessageStore) GetMessageCountByType(msgType MessageType) int {
	count := 0
	for _, msg := range s.messages {
		if msg.Type == msgType {
			count++
		}
	}
	return count
}

// Remove removes a message by ID.
func (s *MessageStore) Remove(id string) bool {
	for i, msg := range s.messages {
		if msg.ID == id {
			s.messages = append(s.messages[:i], s.messages[i+1:]...)
			return true
		}
	}
	return false
}
