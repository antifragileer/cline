// Package task provides task execution functionality for the Cline CLI.
package task

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// MessageType represents the type of message from the core extension
type MessageType string

const (
	// MessageTypeText represents a text message
	MessageTypeText MessageType = "text"
	// MessageTypeTool represents a tool execution
	MessageTypeTool MessageType = "tool"
	// MessageTypeAsk represents an ask prompt
	MessageTypeAsk MessageType = "ask"
	// MessageTypeSay represents a say message
	MessageTypeSay MessageType = "say"
	// MessageTypeCommand represents a command execution
	MessageTypeCommand MessageType = "command"
	// MessageTypeError represents an error
	MessageTypeError MessageType = "error"
	// MessageTypeSystem represents a system message
	MessageTypeSystem MessageType = "system"
	// MessageTypeCheckpoint represents a checkpoint
	MessageTypeCheckpoint MessageType = "checkpoint"
	// MessageTypeBrowser represents a browser action
	MessageTypeBrowser MessageType = "browser"
	// MessageTypeMCP represents an MCP tool
	MessageTypeMCP MessageType = "mcp"
	// MessageTypeCompletion represents task completion
	MessageTypeCompletion MessageType = "completion"
)

// MessageHandler handles messages from the core extension
type MessageHandler interface {
	// HandleMessage processes a single message
	HandleMessage(msg Message) error
	// OnText handles text messages
	OnText(content string, isPartial bool) error
	// OnToolUse handles tool use requests
	OnToolUse(toolName string, params map[string]interface{}) (bool, error)
	// OnToolResult handles tool execution results
	OnToolResult(toolName string, result string, success bool) error
	// OnAsk handles ask prompts
	OnAsk(promptType string, question string) (string, error)
	// OnSay handles say messages
	OnSay(sayType string, content string, partial bool) error
	// OnCommand handles command execution requests
	OnCommand(command string, requiresApproval bool) (string, error)
	// OnCommandOutput handles command output
	OnCommandOutput(output string, isComplete bool) error
	// OnError handles errors
	OnError(err error) error
	// OnInfo handles info messages
	OnInfo(message string) error
	// OnStatus handles status messages
	OnStatus(status string) error
	// OnProgress handles progress updates
	OnProgress(current, total int) error
	// OnCheckpoint handles checkpoint events
	OnCheckpoint(checkpointID string, action string) error
	// OnBrowserAction handles browser actions
	OnBrowserAction(action string, url string) (string, error)
	// OnMCPRequest handles MCP tool requests
	OnMCPRequest(server string, tool string, params map[string]interface{}) (string, error)
	// OnCompletion handles task completion
	OnCompletion(success bool, summary string) error
}

// ClineMessage represents a message from the Cline core
type ClineMessage struct {
	Type    string `json:"type"`
	Say     string `json:"say,omitempty"`
	Ask     string `json:"ask,omitempty"`
	Text    string `json:"text,omitempty"`
	Ts      int64  `json:"ts"`
	Partial bool   `json:"partial,omitempty"`
}

// ClineSay represents SAY message types
type ClineSay string

const (
	// ClineSayText is a text message
	ClineSayText ClineSay = "text"
	// ClineSayReasoning is a reasoning message
	ClineSayReasoning ClineSay = "reasoning"
	// ClineSayCommand is a command message
	ClineSayCommand ClineSay = "command"
	// ClineSayCommandOutput is command output
	ClineSayCommandOutput ClineSay = "command_output"
	// ClineSayTool is a tool message
	ClineSayTool ClineSay = "tool"
	// ClineSayCompletionResult is a completion result
	ClineSayCompletionResult ClineSay = "completion_result"
	// ClineSayAPIReqStarted is an API request started message
	ClineSayAPIReqStarted ClineSay = "api_req_started"
	// ClineSayAPIReqFinished is an API request finished message
	ClineSayAPIReqFinished ClineSay = "api_req_finished"
	// ClineSayError is an error message
	ClineSayError ClineSay = "error"
	// ClineSayCheckpointCreated is a checkpoint created message
	ClineSayCheckpointCreated ClineSay = "checkpoint_created"
)

// Message represents a message from the core extension
type Message struct {
	Type      MessageType            `json:"type"`
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	Content   string                 `json:"content,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	IsPartial bool                   `json:"is_partial,omitempty"`
}

// DefaultMessageHandler provides a default implementation of MessageHandler
type DefaultMessageHandler struct {
	// Callbacks for UI updates
	textCallback       func(string, bool)
	toolCallback       func(string, map[string]interface{}) (bool, error)
	commandCallback    func(string, bool) (string, error)
	askCallback        func(string, string) (string, error)
	errorCallback      func(error)
	completionCallback func(bool, string)

	// State
	currentTaskID string
	approvalState map[string]bool // tool -> approved
}

// NewDefaultMessageHandler creates a new default message handler
func NewDefaultMessageHandler() *DefaultMessageHandler {
	return &DefaultMessageHandler{
		approvalState: make(map[string]bool),
	}
}

// SetTextCallback sets the text message callback
func (h *DefaultMessageHandler) SetTextCallback(cb func(string, bool)) {
	h.textCallback = cb
}

// SetToolCallback sets the tool callback
func (h *DefaultMessageHandler) SetToolCallback(cb func(string, map[string]interface{}) (bool, error)) {
	h.toolCallback = cb
}

// SetCommandCallback sets the command callback
func (h *DefaultMessageHandler) SetCommandCallback(cb func(string, bool) (string, error)) {
	h.commandCallback = cb
}

// SetAskCallback sets the ask callback
func (h *DefaultMessageHandler) SetAskCallback(cb func(string, string) (string, error)) {
	h.askCallback = cb
}

// SetErrorCallback sets the error callback
func (h *DefaultMessageHandler) SetErrorCallback(cb func(error)) {
	h.errorCallback = cb
}

// SetCompletionCallback sets the completion callback
func (h *DefaultMessageHandler) SetCompletionCallback(cb func(bool, string)) {
	h.completionCallback = cb
}

// HandleMessage processes a message based on its type
func (h *DefaultMessageHandler) HandleMessage(msg Message) error {
	switch msg.Type {
	case MessageTypeText:
		return h.OnText(msg.Content, msg.IsPartial)
	case MessageTypeTool:
		return h.handleToolMessage(msg)
	case MessageTypeAsk:
		return h.handleAskMessage(msg)
	case MessageTypeSay:
		return h.handleSayMessage(msg)
	case MessageTypeCommand:
		return h.handleCommandMessage(msg)
	case MessageTypeError:
		return h.OnError(fmt.Errorf(msg.Content))
	case MessageTypeCheckpoint:
		return h.handleCheckpointMessage(msg)
	case MessageTypeBrowser:
		return h.handleBrowserMessage(msg)
	case MessageTypeMCP:
		return h.handleMCPMessage(msg)
	case MessageTypeCompletion:
		return h.handleCompletionMessage(msg)
	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

// OnText handles text messages
func (h *DefaultMessageHandler) OnText(content string, isPartial bool) error {
	if h.textCallback != nil {
		h.textCallback(content, isPartial)
	}
	return nil
}

// OnToolUse handles tool use requests
func (h *DefaultMessageHandler) OnToolUse(toolName string, params map[string]interface{}) (bool, error) {
	if h.toolCallback != nil {
		return h.toolCallback(toolName, params)
	}
	// Default: approve if previously approved
	return h.approvalState[toolName], nil
}

// OnToolResult handles tool execution results
func (h *DefaultMessageHandler) OnToolResult(toolName string, result string, success bool) error {
	// Default implementation just logs
	return nil
}

// OnAsk handles ask prompts
func (h *DefaultMessageHandler) OnAsk(promptType string, question string) (string, error) {
	if h.askCallback != nil {
		return h.askCallback(promptType, question)
	}
	// Default: empty response
	return "", nil
}

// OnSay handles say messages
func (h *DefaultMessageHandler) OnSay(sayType string, content string, partial bool) error {
	// Handle different say types
	switch sayType {
	case "text":
		return h.OnText(content, partial)
	case "error":
		return h.OnError(fmt.Errorf(content))
	default:
		// Log other say types
		return nil
	}
}

// OnInfo handles info messages
func (h *DefaultMessageHandler) OnInfo(message string) error {
	// Default: just log to stdout
	fmt.Println("[INFO]", message)
	return nil
}

// OnCommand handles command execution requests
func (h *DefaultMessageHandler) OnCommand(command string, requiresApproval bool) (string, error) {
	if h.commandCallback != nil {
		return h.commandCallback(command, requiresApproval)
	}
	// Default: refuse
	return "", fmt.Errorf("command execution not allowed")
}

// OnCommandOutput handles command output
func (h *DefaultMessageHandler) OnCommandOutput(output string, isComplete bool) error {
	// Treat as text output
	return h.OnText(output, !isComplete)
}

// OnError handles errors
func (h *DefaultMessageHandler) OnError(err error) error {
	if h.errorCallback != nil {
		h.errorCallback(err)
	}
	return err
}

// OnCheckpoint handles checkpoint events
func (h *DefaultMessageHandler) OnCheckpoint(checkpointID string, action string) error {
	// Default: just acknowledge
	return nil
}

// OnBrowserAction handles browser actions
func (h *DefaultMessageHandler) OnBrowserAction(action string, url string) (string, error) {
	// Default: not implemented
	return "", fmt.Errorf("browser actions not supported")
}

// OnMCPRequest handles MCP tool requests
func (h *DefaultMessageHandler) OnMCPRequest(server string, tool string, params map[string]interface{}) (string, error) {
	// Default: not implemented
	return "", fmt.Errorf("MCP not supported")
}

// OnStatus handles status messages
func (h *DefaultMessageHandler) OnStatus(status string) error {
	// Default: just log to stdout
	fmt.Println("[STATUS]", status)
	return nil
}

// OnProgress handles progress updates
func (h *DefaultMessageHandler) OnProgress(current, total int) error {
	// Default: just log progress
	if total > 0 {
		percent := float64(current) * 100.0 / float64(total)
		fmt.Printf("[PROGRESS] %d/%d (%.1f%%)\n", current, total, percent)
	}
	return nil
}

// OnCompletion handles task completion
func (h *DefaultMessageHandler) OnCompletion(success bool, summary string) error {
	if h.completionCallback != nil {
		h.completionCallback(success, summary)
	}
	return nil
}

// handleToolMessage handles tool messages
func (h *DefaultMessageHandler) handleToolMessage(msg Message) error {
	toolName, _ := msg.Metadata["tool"].(string)
	params, _ := msg.Metadata["params"].(map[string]interface{})
	
	approved, err := h.OnToolUse(toolName, params)
	if err != nil {
		return err
	}

	if approved {
		h.approvalState[toolName] = true
	}

	return nil
}

// handleAskMessage handles ask messages
func (h *DefaultMessageHandler) handleAskMessage(msg Message) error {
	promptType, _ := msg.Metadata["ask_type"].(string)
	response, err := h.OnAsk(promptType, msg.Content)
	if err != nil {
		return err
	}

	// Store response for next message
	msg.Metadata["response"] = response
	return nil
}

// handleSayMessage handles say messages
func (h *DefaultMessageHandler) handleSayMessage(msg Message) error {
	sayType, _ := msg.Metadata["say_type"].(string)
	partial, _ := msg.Metadata["partial"].(bool)
	return h.OnSay(sayType, msg.Content, partial)
}

// handleCommandMessage handles command messages
func (h *DefaultMessageHandler) handleCommandMessage(msg Message) error {
	command, _ := msg.Metadata["command"].(string)
	requiresApproval, _ := msg.Metadata["requires_approval"].(bool)
	
	output, err := h.OnCommand(command, requiresApproval)
	if err != nil {
		return err
	}

	// Store output
	msg.Metadata["output"] = output
	return nil
}

// handleCheckpointMessage handles checkpoint messages
func (h *DefaultMessageHandler) handleCheckpointMessage(msg Message) error {
	checkpointID, _ := msg.Metadata["checkpoint_id"].(string)
	action, _ := msg.Metadata["action"].(string)
	return h.OnCheckpoint(checkpointID, action)
}

// handleBrowserMessage handles browser messages
func (h *DefaultMessageHandler) handleBrowserMessage(msg Message) error {
	action, _ := msg.Metadata["action"].(string)
	url, _ := msg.Metadata["url"].(string)
	
	result, err := h.OnBrowserAction(action, url)
	if err != nil {
		return err
	}

	msg.Metadata["result"] = result
	return nil
}

// handleMCPMessage handles MCP messages
func (h *DefaultMessageHandler) handleMCPMessage(msg Message) error {
	server, _ := msg.Metadata["server"].(string)
	tool, _ := msg.Metadata["tool"].(string)
	params, _ := msg.Metadata["params"].(map[string]interface{})
	
	result, err := h.OnMCPRequest(server, tool, params)
	if err != nil {
		return err
	}

	msg.Metadata["result"] = result
	return nil
}

// handleCompletionMessage handles completion messages
func (h *DefaultMessageHandler) handleCompletionMessage(msg Message) error {
	success, _ := msg.Metadata["success"].(bool)
	summary, _ := msg.Metadata["summary"].(string)
	return h.OnCompletion(success, summary)
}

// ParseMessage parses a raw message into a Message struct
func ParseMessage(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse message: %w", err)
	}
	return &msg, nil
}

// SerializeMessage serializes a Message to bytes
func SerializeMessage(msg *Message) ([]byte, error) {
	return json.Marshal(msg)
}

// MessageFilter filters messages based on criteria
type MessageFilter struct {
	Types     []MessageType
	Since     time.Time
	TaskID    string
	Contains  string
}

// Filter applies the filter to a slice of messages
func (f *MessageFilter) Filter(messages []Message) []Message {
	var filtered []Message
	
	for _, msg := range messages {
		// Filter by type
		if len(f.Types) > 0 {
			found := false
			for _, t := range f.Types {
				if msg.Type == t {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Filter by time
		if !f.Since.IsZero() && msg.Timestamp.Before(f.Since) {
			continue
		}

		// Filter by content
		if f.Contains != "" && !strings.Contains(msg.Content, f.Contains) {
			continue
		}

		filtered = append(filtered, msg)
	}

	return filtered
}

// MessageBuffer buffers messages for batch processing
type MessageBuffer struct {
	messages []Message
	maxSize  int
	timeout  time.Duration
	lastFlush time.Time
}

// NewMessageBuffer creates a new message buffer
func NewMessageBuffer(maxSize int, timeout time.Duration) *MessageBuffer {
	return &MessageBuffer{
		messages:  make([]Message, 0, maxSize),
		maxSize:   maxSize,
		timeout:   timeout,
		lastFlush: time.Now(),
	}
}

// Add adds a message to the buffer
func (b *MessageBuffer) Add(msg Message) []Message {
	b.messages = append(b.messages, msg)
	
	// Check if we need to flush
	if len(b.messages) >= b.maxSize || time.Since(b.lastFlush) >= b.timeout {
		return b.Flush()
	}
	
	return nil
}

// Flush returns all buffered messages and clears the buffer
func (b *MessageBuffer) Flush() []Message {
	messages := b.messages
	b.messages = make([]Message, 0, b.maxSize)
	b.lastFlush = time.Now()
	return messages
}

// IsEmpty returns true if the buffer is empty
func (b *MessageBuffer) IsEmpty() bool {
	return len(b.messages) == 0
}

// Size returns the number of buffered messages
func (b *MessageBuffer) Size() int {
	return len(b.messages)
}