// Package formatter provides JSON output formatting for the Cline CLI.
// This file implements complete JSON output matching the TypeScript CLI format.
package formatter

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/cline/cline/golang-cli/internal/task"
)

// JSONFormatter handles JSON output formatting with full message type support
type JSONFormatter struct {
	output    io.Writer
	errOutput io.Writer
	encoder   *json.Encoder
	streaming bool
	// Track processed messages to avoid duplicates
	processedMessages map[int64]bool
	// Message ordering for consistent output
	messageOrder int64
}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter(output, errOutput io.Writer, streaming bool) *JSONFormatter {
	if output == nil {
		output = os.Stdout
	}
	if errOutput == nil {
		errOutput = os.Stderr
	}

	f := &JSONFormatter{
		output:            output,
		errOutput:         errOutput,
		streaming:         streaming,
		processedMessages: make(map[int64]bool),
		messageOrder:      time.Now().UnixMilli(),
	}

	f.encoder = json.NewEncoder(output)
	if !streaming {
		f.encoder.SetIndent("", "  ")
	}

	return f
}

// JSONMessage represents a complete JSON message matching TypeScript CLI format
// Fields are ordered to match Node.js output format exactly
type JSONMessage struct {
	// Standard fields (always first)
	Ts      int64  `json:"ts"`
	Type    string `json:"type"`
	Text    string `json:"text,omitempty"`
	Partial bool   `json:"partial,omitempty"`

	// SAY/ASK type fields
	Say string `json:"say,omitempty"`
	Ask string `json:"ask,omitempty"`

	// Reasoning fields
	Reasoning string `json:"reasoning,omitempty"`

	// Media fields
	Images []string `json:"images,omitempty"`
	Files  []string `json:"files,omitempty"`

	// Command fields
	CommandCompleted bool `json:"commandCompleted,omitempty"`

	// Checkpoint fields
	LastCheckpointHash     string `json:"lastCheckpointHash,omitempty"`
	IsCheckpointCheckedOut bool   `json:"isCheckpointCheckedOut,omitempty"`

	// Workspace fields
	IsOperationOutsideWorkspace bool `json:"isOperationOutsideWorkspace,omitempty"`

	// Conversation fields
	ConversationHistoryIndex        int   `json:"conversationHistoryIndex,omitempty"`
	ConversationHistoryDeletedRange []int `json:"conversationHistoryDeletedRange,omitempty"`

	// Tool execution fields
	ToolName   string                 `json:"toolName,omitempty"`
	ToolInput  map[string]interface{} `json:"toolInput,omitempty"`
	ToolResult string                 `json:"toolResult,omitempty"`

	// API request fields
	APIRequestStarted  *APIRequestInfo `json:"apiRequestStarted,omitempty"`
	APIRequestFinished *APIRequestInfo `json:"apiRequestFinished,omitempty"`

	// Error fields (must be in consistent order)
	Error   string                 `json:"error,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`

	// Metadata always last
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// OrderedJSONOutput produces JSON with fields in specific order for byte-for-byte matching
type OrderedJSONOutput struct {
	buf *strings.Builder
}

// NewOrderedJSONOutput creates a new ordered JSON output
func NewOrderedJSONOutput() *OrderedJSONOutput {
	return &OrderedJSONOutput{
		buf: &strings.Builder{},
	}
}

// WriteMessage writes a message with fields in specific order
func (o *OrderedJSONOutput) WriteMessage(msg JSONMessage) error {
	o.buf.WriteString("{")
	
	// Write fields in order
	first := true
	
	// ts (required)
	first = o.writeField(first, "ts", msg.Ts)
	
	// type (required)
	first = o.writeField(first, "type", msg.Type)
	
	// text
	if msg.Text != "" {
		first = o.writeField(first, "text", msg.Text)
	}
	
	// partial
	if msg.Partial {
		first = o.writeField(first, "partial", msg.Partial)
	}
	
	// say
	if msg.Say != "" {
		first = o.writeField(first, "say", msg.Say)
	}
	
	// ask
	if msg.Ask != "" {
		first = o.writeField(first, "ask", msg.Ask)
	}
	
	// reasoning
	if msg.Reasoning != "" {
		first = o.writeField(first, "reasoning", msg.Reasoning)
	}
	
	// images
	if len(msg.Images) > 0 {
		first = o.writeField(first, "images", msg.Images)
	}
	
	// files
	if len(msg.Files) > 0 {
		first = o.writeField(first, "files", msg.Files)
	}
	
	// commandCompleted
	if msg.CommandCompleted {
		first = o.writeField(first, "commandCompleted", msg.CommandCompleted)
	}
	
	// lastCheckpointHash
	if msg.LastCheckpointHash != "" {
		first = o.writeField(first, "lastCheckpointHash", msg.LastCheckpointHash)
	}
	
	// isCheckpointCheckedOut
	if msg.IsCheckpointCheckedOut {
		first = o.writeField(first, "isCheckpointCheckedOut", msg.IsCheckpointCheckedOut)
	}
	
	// isOperationOutsideWorkspace
	if msg.IsOperationOutsideWorkspace {
		first = o.writeField(first, "isOperationOutsideWorkspace", msg.IsOperationOutsideWorkspace)
	}
	
	// conversationHistoryIndex
	if msg.ConversationHistoryIndex != 0 {
		first = o.writeField(first, "conversationHistoryIndex", msg.ConversationHistoryIndex)
	}
	
	// conversationHistoryDeletedRange
	if len(msg.ConversationHistoryDeletedRange) > 0 {
		first = o.writeField(first, "conversationHistoryDeletedRange", msg.ConversationHistoryDeletedRange)
	}
	
	// toolName
	if msg.ToolName != "" {
		first = o.writeField(first, "toolName", msg.ToolName)
	}
	
	// toolInput
	if len(msg.ToolInput) > 0 {
		first = o.writeField(first, "toolInput", msg.ToolInput)
	}
	
	// toolResult
	if msg.ToolResult != "" {
		first = o.writeField(first, "toolResult", msg.ToolResult)
	}
	
	// apiRequestStarted
	if msg.APIRequestStarted != nil {
		first = o.writeField(first, "apiRequestStarted", msg.APIRequestStarted)
	}
	
	// apiRequestFinished
	if msg.APIRequestFinished != nil {
		first = o.writeField(first, "apiRequestFinished", msg.APIRequestFinished)
	}
	
	// error
	if msg.Error != "" {
		first = o.writeField(first, "error", msg.Error)
	}
	
	// details
	if len(msg.Details) > 0 {
		first = o.writeField(first, "details", msg.Details)
	}
	
	// metadata always last
	if len(msg.Metadata) > 0 {
		first = o.writeField(first, "metadata", msg.Metadata)
	}
	
	o.buf.WriteString("}")
	return nil
}

func (o *OrderedJSONOutput) writeField(first bool, name string, value interface{}) bool {
	if !first {
		o.buf.WriteString(",")
	}
	
	data, err := json.Marshal(value)
	if err != nil {
		data = []byte("null")
	}
	
	o.buf.WriteString(fmt.Sprintf(`"%s":%s`, name, string(data)))
	return false
}

// String returns the accumulated JSON
func (o *OrderedJSONOutput) String() string {
	return o.buf.String()
}

// APIRequestInfo contains API request details
type APIRequestInfo struct {
	RequestID string `json:"requestId,omitempty"`
	Model     string `json:"model,omitempty"`
	TokensIn  int    `json:"tokensIn,omitempty"`
	TokensOut int    `json:"tokensOut,omitempty"`
}

// FormatMessage formats a generic message as JSON
func (f *JSONFormatter) FormatMessage(msgType string, text string, partial bool) error {
	message := JSONMessage{
		Ts:      f.getNextTimestamp(),
		Type:    msgType,
		Text:    text,
		Partial: partial,
	}
	return f.outputJSON(message)
}

// FormatSayMessage formats a SAY message as JSON
func (f *JSONFormatter) FormatSayMessage(sayType string, text string, partial bool, metadata map[string]interface{}) error {
	message := JSONMessage{
		Ts:      f.getNextTimestamp(),
		Type:    "say",
		Say:     sayType,
		Text:    text,
		Partial: partial,
	}

	// Add metadata if provided
	if metadata != nil {
		message.Metadata = metadata
	}

	// Add type-specific fields based on sayType
	switch sayType {
	case "api_req_started":
		if apiInfo, ok := metadata["apiRequest"].(map[string]interface{}); ok {
			message.APIRequestStarted = &APIRequestInfo{
				RequestID: getString(apiInfo, "requestId"),
				Model:     getString(apiInfo, "model"),
				TokensIn:  getInt(apiInfo, "tokensIn"),
				TokensOut: getInt(apiInfo, "tokensOut"),
			}
		}

	case "api_req_finished":
		if apiInfo, ok := metadata["apiRequest"].(map[string]interface{}); ok {
			message.APIRequestFinished = &APIRequestInfo{
				RequestID: getString(apiInfo, "requestId"),
				Model:     getString(apiInfo, "model"),
				TokensIn:  getInt(apiInfo, "tokensIn"),
				TokensOut: getInt(apiInfo, "tokensOut"),
			}
		}

	case "command":
		if completed, ok := metadata["commandCompleted"].(bool); ok {
			message.CommandCompleted = completed
		}

	case "checkpoint_created":
		if hash, ok := metadata["checkpointHash"].(string); ok {
			message.LastCheckpointHash = hash
		}
		if checkedOut, ok := metadata["isCheckpointCheckedOut"].(bool); ok {
			message.IsCheckpointCheckedOut = checkedOut
		}

	case "reasoning", "thinking":
		if reasoning, ok := metadata["reasoning"].(string); ok {
			message.Reasoning = reasoning
		}

	case "tool_use":
		if toolName, ok := metadata["toolName"].(string); ok {
			message.ToolName = toolName
		}
		if toolInput, ok := metadata["toolInput"].(map[string]interface{}); ok {
			message.ToolInput = toolInput
		}

	case "tool_result":
		if toolName, ok := metadata["toolName"].(string); ok {
			message.ToolName = toolName
		}
		if result, ok := metadata["toolResult"].(string); ok {
			message.ToolResult = result
		}
	}

	return f.outputJSON(message)
}

// FormatAskMessage formats an ASK message as JSON
func (f *JSONFormatter) FormatAskMessage(askType string, text string, metadata map[string]interface{}) error {
	message := JSONMessage{
		Ts:   f.getNextTimestamp(),
		Type: "ask",
		Ask:  askType,
		Text: text,
	}

	// Add metadata if provided
	if metadata != nil {
		message.Metadata = metadata
	}

	return f.outputJSON(message)
}

// FormatToolUse formats a tool_use message
func (f *JSONFormatter) FormatToolUse(toolName string, toolInput map[string]interface{}, partial bool) error {
	message := JSONMessage{
		Ts:        f.getNextTimestamp(),
		Type:      "say",
		Say:       "tool_use",
		ToolName:  toolName,
		ToolInput: toolInput,
		Partial:   partial,
	}
	return f.outputJSON(message)
}

// FormatToolResult formats a tool_result message
func (f *JSONFormatter) FormatToolResult(toolName string, result string, isError bool) error {
	message := JSONMessage{
		Ts:         f.getNextTimestamp(),
		Type:       "say",
		Say:        "tool_result",
		ToolName:   toolName,
		ToolResult: result,
	}

	if isError {
		message.Error = result
	}

	return f.outputJSON(message)
}

// FormatError formats an error message as JSON
func (f *JSONFormatter) FormatError(err error) error {
	message := JSONMessage{
		Ts:    f.getNextTimestamp(),
		Type:  "error",
		Error: err.Error(),
	}
	return f.outputJSON(message)
}

// FormatAPIRequestStarted formats an API request started message
func (f *JSONFormatter) FormatAPIRequestStarted(requestID, model string, tokensIn int) error {
	message := JSONMessage{
		Ts:   f.getNextTimestamp(),
		Type: "say",
		Say:  "api_req_started",
		APIRequestStarted: &APIRequestInfo{
			RequestID: requestID,
			Model:     model,
			TokensIn:  tokensIn,
		},
	}
	return f.outputJSON(message)
}

// FormatAPIRequestFinished formats an API request finished message
func (f *JSONFormatter) FormatAPIRequestFinished(requestID, model string, tokensIn, tokensOut int) error {
	message := JSONMessage{
		Ts:   f.getNextTimestamp(),
		Type: "say",
		Say:  "api_req_finished",
		APIRequestFinished: &APIRequestInfo{
			RequestID: requestID,
			Model:     model,
			TokensIn:  tokensIn,
			TokensOut: tokensOut,
		},
	}
	return f.outputJSON(message)
}

// FormatCheckpoint formats a checkpoint message
func (f *JSONFormatter) FormatCheckpoint(hash string, isCheckedOut bool) error {
	message := JSONMessage{
		Ts:                     f.getNextTimestamp(),
		Type:                   "say",
		Say:                    "checkpoint_created",
		LastCheckpointHash:     hash,
		IsCheckpointCheckedOut: isCheckedOut,
	}
	return f.outputJSON(message)
}

// FormatCommand formats a command execution message
func (f *JSONFormatter) FormatCommand(command string, completed bool, output string) error {
	message := JSONMessage{
		Ts:               f.getNextTimestamp(),
		Type:             "say",
		Say:              "command",
		Text:             command,
		CommandCompleted: completed,
	}

	if output != "" {
		message.Metadata = map[string]interface{}{
			"output": output,
		}
	}

	return f.outputJSON(message)
}

// FormatBrowserAction formats a browser action message
func (f *JSONFormatter) FormatBrowserAction(action string, url string, result string) error {
	message := JSONMessage{
		Ts:   f.getNextTimestamp(),
		Type: "say",
		Say:  "browser_action",
		Text: action,
		Metadata: map[string]interface{}{
			"url":    url,
			"result": result,
		},
	}
	return f.outputJSON(message)
}

// FormatMCPRequest formats an MCP server request message
func (f *JSONFormatter) FormatMCPRequest(serverName string, request string) error {
	message := JSONMessage{
		Ts:   f.getNextTimestamp(),
		Type: "say",
		Say:  "mcp_server_request_started",
		Text: request,
		Metadata: map[string]interface{}{
			"server": serverName,
		},
	}
	return f.outputJSON(message)
}

// FormatMCPResponse formats an MCP server response message
func (f *JSONFormatter) FormatMCPResponse(serverName string, response string) error {
	message := JSONMessage{
		Ts:   f.getNextTimestamp(),
		Type: "say",
		Say:  "mcp_server_response",
		Text: response,
		Metadata: map[string]interface{}{
			"server": serverName,
		},
	}
	return f.outputJSON(message)
}

// FormatCompletionResult formats a completion result message
func (f *JSONFormatter) FormatCompletionResult(result string, metadata map[string]interface{}) error {
	message := JSONMessage{
		Ts:   f.getNextTimestamp(),
		Type: "say",
		Say:  "completion_result",
		Text: result,
	}

	if metadata != nil {
		message.Metadata = metadata
	}

	return f.outputJSON(message)
}

// FormatProgress formats a progress update
func (f *JSONFormatter) FormatProgress(current, total int, message string) error {
	msg := JSONMessage{
		Ts:   f.getNextTimestamp(),
		Type: "say",
		Say:  "task_progress",
		Text: message,
		Metadata: map[string]interface{}{
			"current": current,
			"total":   total,
			"percent": calculatePercent(current, total),
		},
	}
	return f.outputJSON(msg)
}

// Flush flushes any buffered output
func (f *JSONFormatter) Flush() error {
	if flusher, ok := f.output.(interface{ Flush() error }); ok {
		return flusher.Flush()
	}
	return nil
}

// outputJSON outputs a JSON message
func (f *JSONFormatter) outputJSON(message JSONMessage) error {
	// Track processed messages
	if f.processedMessages[message.Ts] {
		// Ensure unique timestamp
		message.Ts = f.getNextTimestamp()
	}
	f.processedMessages[message.Ts] = true

	// In streaming mode, output each message on a new line (JSON Lines format)
	if f.streaming {
		// Use ordered output for consistent field ordering
		ordered := NewOrderedJSONOutput()
		if err := ordered.WriteMessage(message); err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		_, err := fmt.Fprintln(f.output, ordered.String())
		return err
	}

	// Non-streaming mode: output as part of a JSON array or object
	// Use ordered output for consistency
	ordered := NewOrderedJSONOutput()
	if err := ordered.WriteMessage(message); err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	
	// In non-streaming mode, we need to output valid JSON array
	_, err := fmt.Fprintln(f.output, ordered.String())
	return err
}

// FormatErrorOutput formats an error as JSON output
func (f *JSONFormatter) FormatErrorOutput(err error) error {
	message := JSONMessage{
		Ts:    f.getNextTimestamp(),
		Type:  "error",
		Error: err.Error(),
		Details: map[string]interface{}{
			"message": err.Error(),
			"type":    "error",
		},
	}
	return f.outputJSON(message)
}

// FormatWarningOutput formats a warning as JSON output
func (f *JSONFormatter) FormatWarningOutput(message string) error {
	msg := JSONMessage{
		Ts:      f.getNextTimestamp(),
		Type:    "warning",
		Text:    message,
		Partial: false,
	}
	return f.outputJSON(msg)
}

// FormatInfoOutput formats an info message as JSON output
func (f *JSONFormatter) FormatInfoOutput(message string) error {
	msg := JSONMessage{
		Ts:      f.getNextTimestamp(),
		Type:    "info",
		Text:    message,
		Partial: false,
	}
	return f.outputJSON(msg)
}

// getNextTimestamp generates a unique timestamp for message ordering
func (f *JSONFormatter) getNextTimestamp() int64 {
	f.messageOrder++
	return f.messageOrder
}

// calculatePercent calculates a percentage
func calculatePercent(current, total int) float64 {
	if total <= 0 {
		return 0
	}
	return float64(current) * 100.0 / float64(total)
}

// Helper functions for type-safe metadata extraction
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	switch v := m[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if i, err := parseInt(v); err == nil {
			return i
		}
	}
	return 0
}

func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}

// JSONHandler is a task.MessageHandler implementation for JSON output
type JSONHandler struct {
	formatter *JSONFormatter
	verbose   bool
}

// NewJSONHandler creates a new JSON handler for task execution
func NewJSONHandler(output io.Writer, verbose bool) *JSONHandler {
	return &JSONHandler{
		formatter: NewJSONFormatter(output, os.Stderr, true),
		verbose:   verbose,
	}
}

// HandleMessage implements task.MessageHandler
func (h *JSONHandler) HandleMessage(msg task.Message) error {
	// Handle based on message type
	switch msg.Type {
	case task.MessageTypeSay:
		sayType, _ := msg.Metadata["say_type"].(string)
		partial, _ := msg.Metadata["partial"].(bool)
		return h.OnSay(sayType, msg.Content, partial)
	case task.MessageTypeAsk:
		askType, _ := msg.Metadata["ask_type"].(string)
		_, err := h.OnAsk(askType, msg.Content)
		return err
	case task.MessageTypeError:
		return h.OnError(fmt.Errorf("%s", msg.Content))
	default:
		// For other types, just format as info
		if h.verbose {
			h.OnInfo(msg.Content)
		}
		return nil
	}
}

// OnSay handles SAY messages
func (h *JSONHandler) OnSay(sayType string, text string, partial bool) error {
	// Skip partial messages unless verbose (for cleaner JSON output)
	if partial && !h.verbose {
		return nil
	}

	// Map internal types to ClineSay types
	clineSay := task.ClineSay(sayType)

	// Handle special cases
	switch clineSay {
	case task.ClineSayText, task.ClineSayReasoning, task.ClineSayCommand,
		task.ClineSayCommandOutput, task.ClineSayTool, task.ClineSayCompletionResult,
		task.ClineSayAPIReqStarted, task.ClineSayAPIReqFinished, task.ClineSayError:
		return h.formatter.FormatSayMessage(sayType, text, partial, nil)
	case task.ClineSayCheckpointCreated:
		return h.formatter.FormatCheckpoint(text, false)
	default:
		if h.verbose || !partial {
			return h.formatter.FormatSayMessage(sayType, text, partial, nil)
		}
	}
	return nil
}

// OnAsk handles ASK messages (in JSON mode, auto-approve)
func (h *JSONHandler) OnAsk(askType string, text string) (string, error) {
	h.formatter.FormatAskMessage(askType, text, nil)
	// In JSON mode, auto-approve to allow scripting
	return "yesButtonClicked", nil
}

// OnInfo handles info messages
func (h *JSONHandler) OnInfo(text string) error {
	if h.verbose {
		return h.formatter.FormatSayMessage("info", text, false, nil)
	}
	return nil
}

// OnError handles error messages
func (h *JSONHandler) OnError(err error) error {
	return h.formatter.FormatError(err)
}

// OnStatus handles status messages
func (h *JSONHandler) OnStatus(status string) error {
	if h.verbose {
		return h.formatter.FormatSayMessage("info", status, false, nil)
	}
	return nil
}

// OnProgress handles progress messages
func (h *JSONHandler) OnProgress(current, total int) error {
	if h.verbose {
		return h.formatter.FormatProgress(current, total, fmt.Sprintf("%d/%d", current, total))
	}
	return nil
}

// OnText handles text messages
func (h *JSONHandler) OnText(content string, isPartial bool) error {
	return h.OnSay("text", content, isPartial)
}

// OnToolUse handles tool use requests
func (h *JSONHandler) OnToolUse(toolName string, params map[string]interface{}) (bool, error) {
	// In JSON mode, auto-approve
	return true, nil
}

// OnToolResult handles tool execution results
func (h *JSONHandler) OnToolResult(toolName string, result string, success bool) error {
	return h.formatter.FormatToolResult(toolName, result, !success)
}

// OnCommand handles command execution requests
func (h *JSONHandler) OnCommand(command string, requiresApproval bool) (string, error) {
	err := h.formatter.FormatCommand(command, false, "")
	return "execute", err
}

// OnCommandOutput handles command output
func (h *JSONHandler) OnCommandOutput(output string, isComplete bool) error {
	return h.OnSay("command_output", output, !isComplete)
}

// OnCheckpoint handles checkpoint events
func (h *JSONHandler) OnCheckpoint(checkpointID string, action string) error {
	return h.formatter.FormatCheckpoint(checkpointID, false)
}

// OnBrowserAction handles browser actions
func (h *JSONHandler) OnBrowserAction(action string, url string) (string, error) {
	h.formatter.FormatBrowserAction(action, url, "")
	return "", nil
}

// OnMCPRequest handles MCP tool requests
func (h *JSONHandler) OnMCPRequest(server string, tool string, params map[string]interface{}) (string, error) {
	h.formatter.FormatMCPRequest(server, tool)
	return "", nil
}

// OnCompletion handles task completion
func (h *JSONHandler) OnCompletion(success bool, summary string) error {
	return h.formatter.FormatCompletionResult(summary, map[string]interface{}{
		"success": success,
	})
}

// Flush flushes the formatter output
func (h *JSONHandler) Flush() error {
	return h.formatter.Flush()
}
