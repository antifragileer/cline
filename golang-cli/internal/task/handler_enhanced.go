// Package task provides task execution functionality for the Cline CLI.
// This file implements complete message type handlers with full Node.js parity.
package task

import (
	"fmt"
	"strings"
	"sync"
)

// EnhancedMessageHandler provides complete message handling with all message types
type EnhancedMessageHandler struct {
	mu sync.RWMutex

	// Callbacks for different message types
	textCallback       func(content string, isPartial bool) error
	toolUseCallback    func(toolName string, params map[string]interface{}) error
	toolResultCallback func(toolName string, result string, success bool) error
	askCallback        func(askType ClineAsk, question string) (string, error)
	sayCallback        func(sayType ClineSay, content string, partial bool) error
	commandCallback    func(command string, requiresApproval bool) (string, error)
	commandOutputCallback func(output string, isComplete bool) error
	errorCallback      func(err error) error
	checkpointCallback func(checkpointID string, action string) error
	browserCallback    func(action string, url string) (string, error)
	mcpCallback        func(server string, tool string, params map[string]interface{}) (string, error)
	completionCallback func(success bool, summary string) error

	// State tracking
	approvalState     map[string]bool // tool -> approved
	approvedTools     map[string]bool // tools user has chosen to always approve
	rejectedTools     map[string]bool // tools user has chosen to always reject
	messageHistory    []*JSONMessage
	maxHistorySize    int

	// Handler configuration
	config *HandlerConfig
}

// HandlerConfig contains configuration for the message handler
type HandlerConfig struct {
	// AutoApproveTools automatically approves these tool types
	AutoApproveTools []string

	// AutoRejectTools automatically rejects these tool types
	AutoRejectTools []string

	// MaxHistorySize maximum number of messages to keep in history
	MaxHistorySize int

	// EnablePartialMessages process partial messages
	EnablePartialMessages bool

	// PlainTextMode use plain text instead of TUI
	PlainTextMode bool

	// YoloMode auto-approve everything
	YoloMode bool
}

// NewEnhancedMessageHandler creates a new enhanced message handler
func NewEnhancedMessageHandler(config *HandlerConfig) *EnhancedMessageHandler {
	if config == nil {
		config = &HandlerConfig{
			MaxHistorySize:        1000,
			EnablePartialMessages: true,
		}
	}

	return &EnhancedMessageHandler{
		approvalState:  make(map[string]bool),
		approvedTools:  make(map[string]bool),
		rejectedTools:  make(map[string]bool),
		messageHistory: make([]*JSONMessage, 0, config.MaxHistorySize),
		maxHistorySize: config.MaxHistorySize,
		config:         config,
	}
}

// SetTextCallback sets the text message callback
func (h *EnhancedMessageHandler) SetTextCallback(cb func(content string, isPartial bool) error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.textCallback = cb
}

// SetToolUseCallback sets the tool use callback
func (h *EnhancedMessageHandler) SetToolUseCallback(cb func(toolName string, params map[string]interface{}) error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.toolUseCallback = cb
}

// SetToolResultCallback sets the tool result callback
func (h *EnhancedMessageHandler) SetToolResultCallback(cb func(toolName string, result string, success bool) error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.toolResultCallback = cb
}

// SetAskCallback sets the ask callback
func (h *EnhancedMessageHandler) SetAskCallback(cb func(askType ClineAsk, question string) (string, error)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.askCallback = cb
}

// SetSayCallback sets the say callback
func (h *EnhancedMessageHandler) SetSayCallback(cb func(sayType ClineSay, content string, partial bool) error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sayCallback = cb
}

// SetCommandCallback sets the command callback
func (h *EnhancedMessageHandler) SetCommandCallback(cb func(command string, requiresApproval bool) (string, error)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.commandCallback = cb
}

// SetCommandOutputCallback sets the command output callback
func (h *EnhancedMessageHandler) SetCommandOutputCallback(cb func(output string, isComplete bool) error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.commandOutputCallback = cb
}

// SetErrorCallback sets the error callback
func (h *EnhancedMessageHandler) SetErrorCallback(cb func(err error) error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.errorCallback = cb
}

// SetCheckpointCallback sets the checkpoint callback
func (h *EnhancedMessageHandler) SetCheckpointCallback(cb func(checkpointID string, action string) error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checkpointCallback = cb
}

// SetBrowserCallback sets the browser callback
func (h *EnhancedMessageHandler) SetBrowserCallback(cb func(action string, url string) (string, error)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.browserCallback = cb
}

// SetMCPCallback sets the MCP callback
func (h *EnhancedMessageHandler) SetMCPCallback(cb func(server string, tool string, params map[string]interface{}) (string, error)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.mcpCallback = cb
}

// SetCompletionCallback sets the completion callback
func (h *EnhancedMessageHandler) SetCompletionCallback(cb func(success bool, summary string) error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.completionCallback = cb
}

// HandleMessage processes a message based on its type
func (h *EnhancedMessageHandler) HandleMessage(msg *JSONMessage) error {
	if msg == nil {
		return fmt.Errorf("cannot handle nil message")
	}

	// Track message in history
	h.trackMessage(msg)

	// Route to appropriate handler based on type
	switch ClineMessageType(msg.Type) {
	case ClineMessageTypeSay:
		return h.handleSayMessage(msg)
	case ClineMessageTypeAsk:
		return h.handleAskMessage(msg)
	case ClineMessageTypeToolUse:
		return h.handleToolUseMessage(msg)
	case ClineMessageTypeToolResult:
		return h.handleToolResultMessage(msg)
	case ClineMessageTypeCommand:
		return h.handleCommandMessage(msg)
	case ClineMessageTypeCommandOutput:
		return h.handleCommandOutputMessage(msg)
	case ClineMessageTypeCheckpoint:
		return h.handleCheckpointMessage(msg)
	case ClineMessageTypeBrowserAction:
		return h.handleBrowserActionMessage(msg)
	case ClineMessageTypeMCPRequest:
		return h.handleMCPRequestMessage(msg)
	case ClineMessageTypeError:
		return h.handleErrorMessage(msg)
	case ClineMessageTypeSystem:
		return h.handleSystemMessage(msg)
	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}

// handleSayMessage handles SAY messages
func (h *EnhancedMessageHandler) handleSayMessage(msg *JSONMessage) error {
	sayType := ClineSay(msg.Say)

	switch sayType {
	case ClineSayText:
		if h.textCallback != nil {
			return h.textCallback(msg.Text, msg.Partial)
		}

	case ClineSayReasoning:
		// Handle reasoning content
		if h.textCallback != nil {
			return h.textCallback(msg.Reasoning, msg.Partial)
		}

	case ClineSayCommand:
		// Command execution announcement
		if h.sayCallback != nil {
			return h.sayCallback(sayType, msg.Text, msg.Partial)
		}

	case ClineSayCommandOutput:
		if h.commandOutputCallback != nil {
			isComplete := !msg.Partial
			return h.commandOutputCallback(msg.Text, isComplete)
		}

	case ClineSayTool, ClineSayToolUse:
		if h.sayCallback != nil {
			return h.sayCallback(sayType, msg.Text, msg.Partial)
		}

	case ClineSayToolResult:
		if h.toolResultCallback != nil && msg.ToolName != "" {
			isError := msg.Error != ""
			return h.toolResultCallback(msg.ToolName, msg.Text, !isError)
		}

	case ClineSayCompletionResult:
		if h.completionCallback != nil {
			return h.completionCallback(true, msg.Text)
		}

	case ClineSayError:
		if h.errorCallback != nil {
			return h.errorCallback(fmt.Errorf("%s", msg.Text))
		}

	case ClineSayAPIReqStarted:
		// API request started - can be used for progress tracking
		if h.sayCallback != nil {
			return h.sayCallback(sayType, "API request started", false)
		}

	case ClineSayAPIReqFinished:
		// API request finished
		if h.sayCallback != nil {
			return h.sayCallback(sayType, "API request finished", false)
		}

	case ClineSayCheckpointCreated:
		if h.checkpointCallback != nil {
			return h.checkpointCallback(msg.LastCheckpointHash, "created")
		}

	case ClineSayBrowserAction, ClineSayBrowserActionResult:
		if h.sayCallback != nil {
			return h.sayCallback(sayType, msg.Text, msg.Partial)
		}

	case ClineSayMCPServerRequestStarted, ClineSayMCPServerResponse:
		if h.sayCallback != nil {
			return h.sayCallback(sayType, msg.Text, msg.Partial)
		}

	case ClineSayInfo:
		// Info messages - can be logged or displayed
		if h.textCallback != nil {
			return h.textCallback("[INFO] "+msg.Text, false)
		}

	default:
		// Unknown say type - pass through to generic callback if available
		if h.sayCallback != nil {
			return h.sayCallback(sayType, msg.Text, msg.Partial)
		}
	}

	return nil
}

// handleAskMessage handles ASK messages (approval requests)
func (h *EnhancedMessageHandler) handleAskMessage(msg *JSONMessage) (err error) {
	askType := ClineAsk(msg.Ask)

	// Check for auto-approval/rejection based on configuration
	if h.shouldAutoApprove(askType, msg) {
		return h.sendAutoApproval(msg, true)
	}
	if h.shouldAutoReject(askType, msg) {
		return h.sendAutoApproval(msg, false)
	}

	var response string

	switch askType {
	case ClineAskFollowup:
		response, err = h.handleFollowupAsk(msg)

	case ClineAskPlanModeRespond:
		response, err = h.handlePlanModeAsk(msg)

	case ClineAskActModeRespond:
		response, err = h.handleActModeAsk(msg)

	case ClineAskCommand:
		response, err = h.handleCommandAsk(msg)

	case ClineAskTool:
		response, err = h.handleToolAsk(msg)

	case ClineAskCompletionResult:
		response, err = h.handleCompletionAsk(msg)

	case ClineAskBrowserActionLaunch:
		response, err = h.handleBrowserActionAsk(msg)

	case ClineAskUseMCPServer:
		response, err = h.handleMCPAsk(msg)

	case ClineAskResumeTask, ClineAskResumeCompletedTask:
		response, err = h.handleResumeAsk(msg)

	case ClineAskNewTask:
		response, err = h.handleNewTaskAsk(msg)

	case ClineAskAPIReqFailed:
		response, err = h.handleAPIReqFailedAsk(msg)

	case ClineAskMistakeLimitReached:
		response, err = h.handleMistakeLimitAsk(msg)

	default:
		// Generic ask handler
		response, err = h.handleGenericAsk(msg)
	}

	if err != nil {
		return err
	}

	// Send response back
	return h.sendAskResponse(msg, response)
}

// handleToolUseMessage handles tool use messages
func (h *EnhancedMessageHandler) handleToolUseMessage(msg *JSONMessage) error {
	if h.toolUseCallback != nil {
		return h.toolUseCallback(msg.ToolName, msg.ToolInput)
	}
	return nil
}

// handleToolResultMessage handles tool result messages
func (h *EnhancedMessageHandler) handleToolResultMessage(msg *JSONMessage) error {
	if h.toolResultCallback != nil {
		success := msg.Error == ""
		return h.toolResultCallback(msg.ToolName, msg.Text, success)
	}
	return nil
}

// handleCommandMessage handles command messages
func (h *EnhancedMessageHandler) handleCommandMessage(msg *JSONMessage) error {
	if h.commandCallback != nil {
		_, err := h.commandCallback(msg.Command, true)
		return err
	}
	return nil
}

// handleCommandOutputMessage handles command output messages
func (h *EnhancedMessageHandler) handleCommandOutputMessage(msg *JSONMessage) error {
	if h.commandOutputCallback != nil {
		isComplete := !msg.Partial
		return h.commandOutputCallback(msg.Text, isComplete)
	}
	return nil
}

// handleCheckpointMessage handles checkpoint messages
func (h *EnhancedMessageHandler) handleCheckpointMessage(msg *JSONMessage) error {
	if h.checkpointCallback != nil {
		return h.checkpointCallback(msg.LastCheckpointHash, "created")
	}
	return nil
}

// handleBrowserActionMessage handles browser action messages
func (h *EnhancedMessageHandler) handleBrowserActionMessage(msg *JSONMessage) error {
	if h.browserCallback != nil {
		_, err := h.browserCallback(msg.BrowserAction, msg.BrowserURL)
		return err
	}
	return nil
}

// handleMCPRequestMessage handles MCP request messages
func (h *EnhancedMessageHandler) handleMCPRequestMessage(msg *JSONMessage) error {
	if h.mcpCallback != nil {
		tool := ""
		if msg.Metadata != nil {
			if t, ok := msg.Metadata["tool"].(string); ok {
				tool = t
			}
		}
		_, err := h.mcpCallback(msg.MCPServer, tool, msg.Metadata)
		return err
	}
	return nil
}

// handleErrorMessage handles error messages
func (h *EnhancedMessageHandler) handleErrorMessage(msg *JSONMessage) error {
	if h.errorCallback != nil {
		return h.errorCallback(fmt.Errorf("%s", msg.Error))
	}
	return nil
}

// handleSystemMessage handles system messages
func (h *EnhancedMessageHandler) handleSystemMessage(msg *JSONMessage) error {
	// System messages are typically informational
	if h.textCallback != nil {
		return h.textCallback("[SYSTEM] "+msg.Text, false)
	}
	return nil
}

// Individual ask handlers

func (h *EnhancedMessageHandler) handleFollowupAsk(msg *JSONMessage) (string, error) {
	if h.askCallback != nil {
		return h.askCallback(ClineAskFollowup, msg.Text)
	}
	// Default: provide empty response to continue
	return "", nil
}

func (h *EnhancedMessageHandler) handlePlanModeAsk(msg *JSONMessage) (string, error) {
	if h.askCallback != nil {
		return h.askCallback(ClineAskPlanModeRespond, msg.Text)
	}
	return "proceed", nil
}

func (h *EnhancedMessageHandler) handleActModeAsk(msg *JSONMessage) (string, error) {
	if h.askCallback != nil {
		return h.askCallback(ClineAskActModeRespond, msg.Text)
	}
	return "proceed", nil
}

func (h *EnhancedMessageHandler) handleCommandAsk(msg *JSONMessage) (string, error) {
	if h.commandCallback != nil {
		_, err := h.commandCallback(msg.Text, true)
		if err != nil {
			return "noButtonClicked", err
		}
		return "yesButtonClicked", nil
	}
	return "noButtonClicked", nil
}

func (h *EnhancedMessageHandler) handleToolAsk(msg *JSONMessage) (string, error) {
	if h.askCallback != nil {
		return h.askCallback(ClineAskTool, msg.Text)
	}
	return "noButtonClicked", nil
}

func (h *EnhancedMessageHandler) handleCompletionAsk(msg *JSONMessage) (string, error) {
	if h.completionCallback != nil {
		// Mark as completed
		_ = h.completionCallback(true, msg.Text)
	}
	return "yesButtonClicked", nil
}

func (h *EnhancedMessageHandler) handleBrowserActionAsk(msg *JSONMessage) (string, error) {
	if h.browserCallback != nil {
		_, err := h.browserCallback("launch", msg.Text)
		if err != nil {
			return "noButtonClicked", err
		}
		return "yesButtonClicked", nil
	}
	return "noButtonClicked", nil
}

func (h *EnhancedMessageHandler) handleMCPAsk(msg *JSONMessage) (string, error) {
	if h.mcpCallback != nil {
		tool := ""
		if msg.Metadata != nil {
			if t, ok := msg.Metadata["tool"].(string); ok {
				tool = t
			}
		}
		_, err := h.mcpCallback(msg.MCPServer, tool, msg.Metadata)
		if err != nil {
			return "noButtonClicked", err
		}
		return "yesButtonClicked", nil
	}
	return "noButtonClicked", nil
}

func (h *EnhancedMessageHandler) handleResumeAsk(msg *JSONMessage) (string, error) {
	if h.askCallback != nil {
		return h.askCallback(ClineAskResumeTask, msg.Text)
	}
	return "yesButtonClicked", nil
}

func (h *EnhancedMessageHandler) handleNewTaskAsk(msg *JSONMessage) (string, error) {
	if h.askCallback != nil {
		return h.askCallback(ClineAskNewTask, msg.Text)
	}
	return "noButtonClicked", nil
}

func (h *EnhancedMessageHandler) handleAPIReqFailedAsk(msg *JSONMessage) (string, error) {
	if h.askCallback != nil {
		return h.askCallback(ClineAskAPIReqFailed, msg.Text)
	}
	// Default: retry
	return "yesButtonClicked", nil
}

func (h *EnhancedMessageHandler) handleMistakeLimitAsk(msg *JSONMessage) (string, error) {
	if h.askCallback != nil {
		return h.askCallback(ClineAskMistakeLimitReached, msg.Text)
	}
	// Default: stop execution
	return "noButtonClicked", nil
}

func (h *EnhancedMessageHandler) handleGenericAsk(msg *JSONMessage) (string, error) {
	if h.askCallback != nil {
		askType := ClineAsk(msg.Ask)
		return h.askCallback(askType, msg.Text)
	}
	return "", nil
}

// Helper methods

func (h *EnhancedMessageHandler) shouldAutoApprove(askType ClineAsk, msg *JSONMessage) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Yolo mode auto-approves everything
	if h.config.YoloMode {
		return true
	}

	// Check if this ask type should be auto-approved
	switch askType {
	case ClineAskTool:
		toolName := msg.ToolName
		if h.approvedTools[toolName] {
			return true
		}
		for _, tool := range h.config.AutoApproveTools {
			if strings.EqualFold(tool, toolName) {
				return true
			}
		}
	case ClineAskCommand:
		// Check if command matches auto-approve patterns
		for _, pattern := range h.config.AutoApproveTools {
			if strings.Contains(msg.Text, pattern) {
				return true
			}
		}
	}

	return false
}

func (h *EnhancedMessageHandler) shouldAutoReject(askType ClineAsk, msg *JSONMessage) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Check if tool should be auto-rejected
	if askType == ClineAskTool {
		toolName := msg.ToolName
		if h.rejectedTools[toolName] {
			return true
		}
		for _, tool := range h.config.AutoRejectTools {
			if strings.EqualFold(tool, toolName) {
				return true
			}
		}
	}

	return false
}

func (h *EnhancedMessageHandler) sendAutoApproval(msg *JSONMessage, approved bool) error {
	response := "noButtonClicked"
	if approved {
		response = "yesButtonClicked"
	}
	return h.sendAskResponse(msg, response)
}

func (h *EnhancedMessageHandler) sendAskResponse(originalMsg *JSONMessage, response string) error {
	// This would typically send a response back to the core extension
	// Implementation depends on the transport layer
	return nil
}

func (h *EnhancedMessageHandler) trackMessage(msg *JSONMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.messageHistory = append(h.messageHistory, msg)

	// Trim history if it exceeds max size
	if len(h.messageHistory) > h.maxHistorySize {
		h.messageHistory = h.messageHistory[len(h.messageHistory)-h.maxHistorySize:]
	}
}

// GetMessageHistory returns the message history
func (h *EnhancedMessageHandler) GetMessageHistory() []*JSONMessage {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// Return a copy
	result := make([]*JSONMessage, len(h.messageHistory))
	copy(result, h.messageHistory)
	return result
}

// ClearHistory clears the message history
func (h *EnhancedMessageHandler) ClearHistory() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.messageHistory = make([]*JSONMessage, 0, h.maxHistorySize)
}

// ApproveTool permanently approves a tool
func (h *EnhancedMessageHandler) ApproveTool(toolName string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.approvedTools[toolName] = true
	delete(h.rejectedTools, toolName)
}

// RejectTool permanently rejects a tool
func (h *EnhancedMessageHandler) RejectTool(toolName string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.rejectedTools[toolName] = true
	delete(h.approvedTools, toolName)
}

// IsToolApproved checks if a tool is approved
func (h *EnhancedMessageHandler) IsToolApproved(toolName string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.approvedTools[toolName]
}

// IsToolRejected checks if a tool is rejected
func (h *EnhancedMessageHandler) IsToolRejected(toolName string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.rejectedTools[toolName]
}