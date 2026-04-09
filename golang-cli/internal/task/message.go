// Package task provides task execution functionality for the Cline CLI.
// This file implements complete message type definitions matching Node.js implementation.
package task

import (
	"encoding/json"
	"fmt"
	"time"
)

// ============================================================================
// Extended Message Type Enums - Additional types beyond handler.go
// ============================================================================

// ClineMessageType represents the type of Cline message
type ClineMessageType string

const (
	// ClineMessageTypeSay represents AI responses
	ClineMessageTypeSay ClineMessageType = "say"
	// ClineMessageTypeAsk represents approval requests
	ClineMessageTypeAsk ClineMessageType = "ask"
	// ClineMessageTypeToolUse represents tool invocation
	ClineMessageTypeToolUse ClineMessageType = "tool_use"
	// ClineMessageTypeToolResult represents tool execution results
	ClineMessageTypeToolResult ClineMessageType = "tool_result"
	// ClineMessageTypeCommand represents command execution requests
	ClineMessageTypeCommand ClineMessageType = "command"
	// ClineMessageTypeCommandOutput represents command output streaming
	ClineMessageTypeCommandOutput ClineMessageType = "command_output"
	// ClineMessageTypeCheckpoint represents Git checkpoint creation
	ClineMessageTypeCheckpoint ClineMessageType = "checkpoint"
	// ClineMessageTypeBrowserAction represents browser automation
	ClineMessageTypeBrowserAction ClineMessageType = "browser_action"
	// ClineMessageTypeMCPRequest represents MCP tool requests
	ClineMessageTypeMCPRequest ClineMessageType = "mcp_request"
	// ClineMessageTypeError represents error messages
	ClineMessageTypeError ClineMessageType = "error"
	// ClineMessageTypeSystem represents system messages
	ClineMessageTypeSystem ClineMessageType = "system"
)

// Extended ClineSay types (additional to those in handler.go)
const (
	// ClineSayTask - Initial task message
	ClineSayTask ClineSay = "task"
	// ClineSayUserFeedback - User feedback
	ClineSayUserFeedback ClineSay = "user_feedback"
	// ClineSayAPIReqRetried - API request retried
	ClineSayAPIReqRetried ClineSay = "api_req_retried"
	// ClineSayToolUse - Tool use
	ClineSayToolUse ClineSay = "tool_use"
	// ClineSayToolResult - Tool result
	ClineSayToolResult ClineSay = "tool_result"
	// ClineSayBrowserAction - Browser action
	ClineSayBrowserAction ClineSay = "browser_action"
	// ClineSayBrowserActionResult - Browser action result
	ClineSayBrowserActionResult ClineSay = "browser_action_result"
	// ClineSayMCPServerRequestStarted - MCP server request started
	ClineSayMCPServerRequestStarted ClineSay = "mcp_server_request_started"
	// ClineSayMCPServerResponse - MCP server response
	ClineSayMCPServerResponse ClineSay = "mcp_server_response"
	// ClineSayInfo - Info message
	ClineSayInfo ClineSay = "info"
)

// ClineAsk represents the type of ASK message
type ClineAsk string

const (
	// ClineAskFollowup - Follow-up question
	ClineAskFollowup ClineAsk = "followup"
	// ClineAskPlanModeRespond - Plan mode respond
	ClineAskPlanModeRespond ClineAsk = "plan_mode_respond"
	// ClineAskActModeRespond - Act mode respond
	ClineAskActModeRespond ClineAsk = "act_mode_respond"
	// ClineAskCommand - Command approval
	ClineAskCommand ClineAsk = "command"
	// ClineAskCommandOutput - Command output
	ClineAskCommandOutput ClineAsk = "command_output"
	// ClineAskCompletionResult - Completion result approval
	ClineAskCompletionResult ClineAsk = "completion_result"
	// ClineAskTool - Tool approval
	ClineAskTool ClineAsk = "tool"
	// ClineAskAPIReqFailed - API request failed
	ClineAskAPIReqFailed ClineAsk = "api_req_failed"
	// ClineAskResumeTask - Resume task
	ClineAskResumeTask ClineAsk = "resume_task"
	// ClineAskResumeCompletedTask - Resume completed task
	ClineAskResumeCompletedTask ClineAsk = "resume_completed_task"
	// ClineAskMistakeLimitReached - Mistake limit reached
	ClineAskMistakeLimitReached ClineAsk = "mistake_limit_reached"
	// ClineAskBrowserActionLaunch - Browser action launch
	ClineAskBrowserActionLaunch ClineAsk = "browser_action_launch"
	// ClineAskUseMCPServer - Use MCP server
	ClineAskUseMCPServer ClineAsk = "use_mcp_server"
	// ClineAskNewTask - New task
	ClineAskNewTask ClineAsk = "new_task"
)

// ============================================================================
// JSON Message Format - Exact matching Node.js implementation
// ============================================================================

// JSONMessage represents a complete JSON message matching Node.js format exactly
type JSONMessage struct {
	// Standard fields (always present)
	Ts   int64  `json:"ts"`
	Type string `json:"type"`
	Text string `json:"text,omitempty"`

	// Message type specific fields
	Say string `json:"say,omitempty"`
	Ask string `json:"ask,omitempty"`

	// Reasoning fields
	Reasoning string `json:"reasoning,omitempty"`

	// Partial content indicator
	Partial bool `json:"partial,omitempty"`

	// Media fields
	Images []string `json:"images,omitempty"`
	Files  []string `json:"files,omitempty"`

	// Command fields
	Command          string `json:"command,omitempty"`
	CommandCompleted bool   `json:"commandCompleted,omitempty"`

	// Tool fields
	ToolName   string                 `json:"toolName,omitempty"`
	ToolInput  map[string]interface{} `json:"toolInput,omitempty"`
	ToolResult string                 `json:"toolResult,omitempty"`

	// Checkpoint fields
	LastCheckpointHash     string `json:"lastCheckpointHash,omitempty"`
	IsCheckpointCheckedOut bool   `json:"isCheckpointCheckedOut,omitempty"`

	// Browser action fields
	BrowserAction string `json:"browserAction,omitempty"`
	BrowserURL    string `json:"browserUrl,omitempty"`

	// MCP fields
	MCPServer string `json:"mcpServer,omitempty"`
	MCPTool   string `json:"mcpTool,omitempty"`

	// Error fields
	Error   string                 `json:"error,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`

	// API request fields
	APIRequestStarted  *APIRequestInfo `json:"apiRequestStarted,omitempty"`
	APIRequestFinished *APIRequestInfo `json:"apiRequestFinished,omitempty"`

	// Metadata always last
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// APIRequestInfo contains API request details
type APIRequestInfo struct {
	RequestID string  `json:"requestId,omitempty"`
	Model     string  `json:"model,omitempty"`
	TokensIn  int     `json:"tokensIn,omitempty"`
	TokensOut int     `json:"tokensOut,omitempty"`
	Cost      float64 `json:"cost,omitempty"`
}

// ============================================================================
// Message Serialization/Deserialization
// ============================================================================

// MessageSerializer handles message serialization/deserialization
type MessageSerializer struct {
	// Track message ordering
	messageCounter int64
}

// NewMessageSerializer creates a new message serializer
func NewMessageSerializer() *MessageSerializer {
	return &MessageSerializer{
		messageCounter: time.Now().UnixMilli(),
	}
}

// Serialize serializes a JSONMessage to bytes
func (s *MessageSerializer) Serialize(msg *JSONMessage) ([]byte, error) {
	if msg == nil {
		return nil, fmt.Errorf("cannot serialize nil message")
	}

	// Ensure timestamp is set
	if msg.Ts == 0 {
		msg.Ts = s.getNextTimestamp()
	}

	return json.Marshal(msg)
}

// SerializeStreaming serializes a message for streaming output (one line per message)
func (s *MessageSerializer) SerializeStreaming(msg *JSONMessage) ([]byte, error) {
	data, err := s.Serialize(msg)
	if err != nil {
		return nil, err
	}

	// Add newline for JSON Lines format
	result := make([]byte, len(data)+1)
	copy(result, data)
	result[len(data)] = '\n'

	return result, nil
}

// Deserialize deserializes bytes to a JSONMessage
func (s *MessageSerializer) Deserialize(data []byte) (*JSONMessage, error) {
	var msg JSONMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to deserialize message: %w", err)
	}
	return &msg, nil
}

// getNextTimestamp generates a unique timestamp
func (s *MessageSerializer) getNextTimestamp() int64 {
	s.messageCounter++
	return s.messageCounter
}

// ============================================================================
// Message Routing
// ============================================================================

// MessageRouter routes messages to appropriate handlers
type MessageRouter struct {
	handlers map[ClineMessageType]MessageTypeHandler
}

// MessageTypeHandler handles a specific message type
type MessageTypeHandler interface {
	Handle(msg *JSONMessage) error
}

// MessageTypeHandlerFunc is a function that implements MessageTypeHandler
type MessageTypeHandlerFunc func(msg *JSONMessage) error

// Handle implements MessageTypeHandler
func (f MessageTypeHandlerFunc) Handle(msg *JSONMessage) error {
	return f(msg)
}

// NewMessageRouter creates a new message router
func NewMessageRouter() *MessageRouter {
	return &MessageRouter{
		handlers: make(map[ClineMessageType]MessageTypeHandler),
	}
}

// RegisterHandler registers a handler for a message type
func (r *MessageRouter) RegisterHandler(msgType ClineMessageType, handler MessageTypeHandler) {
	r.handlers[msgType] = handler
}

// RegisterHandlerFunc registers a function handler for a message type
func (r *MessageRouter) RegisterHandlerFunc(msgType ClineMessageType, handler func(msg *JSONMessage) error) {
	r.handlers[msgType] = MessageTypeHandlerFunc(handler)
}

// Route routes a message to its handler
func (r *MessageRouter) Route(msg *JSONMessage) error {
	if msg == nil {
		return fmt.Errorf("cannot route nil message")
	}

	msgType := ClineMessageType(msg.Type)
	handler, ok := r.handlers[msgType]
	if !ok {
		return fmt.Errorf("no handler registered for message type: %s", msg.Type)
	}

	return handler.Handle(msg)
}

// HasHandler checks if a handler is registered for a message type
func (r *MessageRouter) HasHandler(msgType ClineMessageType) bool {
	_, ok := r.handlers[msgType]
	return ok
}

// ============================================================================
// Message Validation
// ============================================================================

// MessageValidator validates messages
type MessageValidator struct{}

// NewMessageValidator creates a new message validator
func NewMessageValidator() *MessageValidator {
	return &MessageValidator{}
}

// Validate validates a message
func (v *MessageValidator) Validate(msg *JSONMessage) error {
	if msg == nil {
		return fmt.Errorf("message is nil")
	}

	// Check required fields
	if msg.Ts == 0 {
		return fmt.Errorf("message timestamp is required")
	}

	if msg.Type == "" {
		return fmt.Errorf("message type is required")
	}

	// Validate message type
	switch ClineMessageType(msg.Type) {
	case ClineMessageTypeSay, ClineMessageTypeAsk, ClineMessageTypeToolUse,
		ClineMessageTypeToolResult, ClineMessageTypeCommand,
		ClineMessageTypeCommandOutput, ClineMessageTypeCheckpoint,
		ClineMessageTypeBrowserAction, ClineMessageTypeMCPRequest,
		ClineMessageTypeError, ClineMessageTypeSystem:
		// Valid
	default:
		return fmt.Errorf("invalid message type: %s", msg.Type)
	}

	// Type-specific validation
	switch ClineMessageType(msg.Type) {
	case ClineMessageTypeSay:
		if msg.Say == "" {
			return fmt.Errorf("say type is required for say messages")
		}
	case ClineMessageTypeAsk:
		if msg.Ask == "" {
			return fmt.Errorf("ask type is required for ask messages")
		}
	}

	return nil
}

// ValidateBatch validates multiple messages
func (v *MessageValidator) ValidateBatch(messages []*JSONMessage) []error {
	var errors []error
	for i, msg := range messages {
		if err := v.Validate(msg); err != nil {
			errors = append(errors, fmt.Errorf("message %d: %w", i, err))
		}
	}
	return errors
}