// Package task provides task execution functionality for the Cline CLI.
package task

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
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
	// OnStatus handles status updates
	OnStatus(status string)
	// OnProgress handles progress updates
	OnProgress(current, total int)
}

// ClineMessage represents a message in the format matching Node.js CLI
type ClineMessage struct {
	Ts                            int64    `json:"ts"`
	Type                          string   `json:"type"` // "ask" or "say"
	Ask                           string   `json:"ask,omitempty"`
	Say                           string   `json:"say,omitempty"`
	Text                          string   `json:"text,omitempty"`
	Reasoning                     string   `json:"reasoning,omitempty"`
	Images                        []string `json:"images,omitempty"`
	Files                         []string `json:"files,omitempty"`
	Partial                       bool     `json:"partial,omitempty"`
	CommandCompleted              bool     `json:"commandCompleted,omitempty"`
	LastCheckpointHash            string   `json:"lastCheckpointHash,omitempty"`
	IsCheckpointCheckedOut        bool     `json:"isCheckpointCheckedOut,omitempty"`
	IsOperationOutsideWorkspace   bool     `json:"isOperationOutsideWorkspace,omitempty"`
	ConversationHistoryIndex      int      `json:"conversationHistoryIndex,omitempty"`
	ConversationHistoryDeletedRange []int  `json:"conversationHistoryDeletedRange,omitempty"`
}

// ClineAsk represents the ask types matching Node.js CLI
type ClineAsk string

const (
	ClineAskFollowup              ClineAsk = "followup"
	ClineAskPlanModeRespond       ClineAsk = "plan_mode_respond"
	ClineAskActModeRespond        ClineAsk = "act_mode_respond"
	ClineAskCommand               ClineAsk = "command"
	ClineAskCommandOutput         ClineAsk = "command_output"
	ClineAskCompletionResult      ClineAsk = "completion_result"
	ClineAskTool                  ClineAsk = "tool"
	ClineAskAPIReqFailed          ClineAsk = "api_req_failed"
	ClineAskResumeTask            ClineAsk = "resume_task"
	ClineAskResumeCompletedTask   ClineAsk = "resume_completed_task"
	ClineAskMistakeLimitReached   ClineAsk = "mistake_limit_reached"
	ClineAskBrowserActionLaunch   ClineAsk = "browser_action_launch"
	ClineAskUseMcpServer          ClineAsk = "use_mcp_server"
	ClineAskNewTask               ClineAsk = "new_task"
	ClineAskCondense              ClineAsk = "condense"
	ClineAskSummarizeTask         ClineAsk = "summarize_task"
	ClineAskReportBug             ClineAsk = "report_bug"
	ClineAskUseSubagents          ClineAsk = "use_subagents"
)

// ClineSay represents the say types matching Node.js CLI
type ClineSay string

const (
	ClineSayTask                           ClineSay = "task"
	ClineSayError                          ClineSay = "error"
	ClineSayErrorRetry                     ClineSay = "error_retry"
	ClineSayAPIReqStarted                  ClineSay = "api_req_started"
	ClineSayAPIReqFinished                 ClineSay = "api_req_finished"
	ClineSayText                           ClineSay = "text"
	ClineSayReasoning                      ClineSay = "reasoning"
	ClineSayCompletionResult               ClineSay = "completion_result"
	ClineSayUserFeedback                   ClineSay = "user_feedback"
	ClineSayUserFeedbackDiff               ClineSay = "user_feedback_diff"
	ClineSayAPIReqRetried                  ClineSay = "api_req_retried"
	ClineSayCommand                        ClineSay = "command"
	ClineSayCommandOutput                  ClineSay = "command_output"
	ClineSayTool                           ClineSay = "tool"
	ClineSayShellIntegrationWarning        ClineSay = "shell_integration_warning"
	ClineSayShellIntegrationWarningWithSuggestion ClineSay = "shell_integration_warning_with_suggestion"
	ClineSayBrowserActionLaunch            ClineSay = "browser_action_launch"
	ClineSayBrowserAction                  ClineSay = "browser_action"
	ClineSayBrowserActionResult            ClineSay = "browser_action_result"
	ClineSayMcpServerRequestStarted        ClineSay = "mcp_server_request_started"
	ClineSayMcpServerResponse              ClineSay = "mcp_server_response"
	ClineSayMcpNotification                ClineSay = "mcp_notification"
	ClineSayUseMcpServer                   ClineSay = "use_mcp_server"
	ClineSayDiffError                      ClineSay = "diff_error"
	ClineSayDeletedAPIReqs                 ClineSay = "deleted_api_reqs"
	ClineSayClineignoreError               ClineSay = "clineignore_error"
	ClineSayCommandPermissionDenied        ClineSay = "command_permission_denied"
	ClineSayCheckpointCreated              ClineSay = "checkpoint_created"
	ClineSayLoadMcpDocumentation           ClineSay = "load_mcp_documentation"
	ClineSayGenerateExplanation            ClineSay = "generate_explanation"
	ClineSayInfo                           ClineSay = "info"
	ClineSayTaskProgress                   ClineSay = "task_progress"
	ClineSayHookStatus                     ClineSay = "hook_status"
	ClineSayHookOutputStream               ClineSay = "hook_output_stream"
	ClineSaySubagent                       ClineSay = "subagent"
	ClineSayUseSubagents                   ClineSay = "use_subagents"
	ClineSaySubagentUsage                  ClineSay = "subagent_usage"
	ClineSayConditionalRulesApplied        ClineSay = "conditional_rules_applied"
)

// PlainTextHandler handles messages in plain text format.
type PlainTextHandler struct {
	Verbose     bool
	JSONOutput  bool
	Output      io.Writer
	AutoApprove bool
	
	// For tracking messages in non-JSON mode
	processedMessages map[int64]string
	completionCutoffTs int64
}

// NewPlainTextHandler creates a new plain text handler
func NewPlainTextHandler(verbose, jsonOutput, autoApprove bool, output io.Writer) *PlainTextHandler {
	if output == nil {
		output = os.Stdout
	}
	return &PlainTextHandler{
		Verbose:           verbose,
		JSONOutput:        jsonOutput,
		Output:            output,
		AutoApprove:       autoApprove,
		processedMessages: make(map[int64]string),
		completionCutoffTs: time.Now().UnixMilli(),
	}
}

// OnSay handles a SAY message from the assistant
func (h *PlainTextHandler) OnSay(sayType string, text string, partial bool) {
	if h.Output == nil {
		h.Output = os.Stdout
	}

	ts := time.Now().UnixMilli()

	// JSON mode: stream all messages as JSON
	if h.JSONOutput {
		msg := ClineMessage{
			Ts:      ts,
			Type:    "say",
			Say:     sayType,
			Text:    text,
			Partial: partial,
		}
		h.outputJSON(msg)
		return
	}

	// Non-JSON mode: don't print partial messages
	if partial {
		return
	}

	// Track processed messages
	h.processedMessages[ts] = text

	// Non-JSON mode output
	switch sayType {
	case "text":
		if h.Verbose {
			fmt.Fprintln(h.Output, text)
		}
	case "error":
		fmt.Fprintf(os.Stderr, "Error: %s\n", text)
	case "command":
		if h.Verbose {
			fmt.Fprintf(h.Output, "Command: %s\n", text)
		}
	case "command_output":
		if h.Verbose {
			fmt.Fprintf(h.Output, "%s\n", text)
		}
	case "tool":
		if h.Verbose {
			fmt.Fprintf(h.Output, "Tool: %s\n", text)
		}
	case "thinking":
		if h.Verbose {
			fmt.Fprintf(h.Output, "Thinking: %s\n", text)
		}
	case "completion_result":
		if h.Verbose {
			fmt.Fprintf(h.Output, "\nResult: %s\n", text)
		}
	default:
		if h.Verbose {
			fmt.Fprintf(h.Output, "[%s] %s\n", sayType, text)
		}
	}
}

// OnAsk handles an ASK message that requires user response
func (h *PlainTextHandler) OnAsk(askType string, text string) (string, error) {
	if h.Output == nil {
		h.Output = os.Stdout
	}

	ts := time.Now().UnixMilli()

	// JSON mode: stream ask as JSON
	if h.JSONOutput {
		msg := ClineMessage{
			Ts:   ts,
			Type: "ask",
			Ask:  askType,
			Text: text,
		}
		h.outputJSON(msg)
		
		// For JSON mode in plain text handler, auto-approve
		return "yesButtonClicked", nil
	}

	// Track processed messages
	h.processedMessages[ts] = text

	// Auto-approve if enabled
	if h.AutoApprove {
		return "yesButtonClicked", nil
	}

	// Non-JSON mode: print approval prompt to stderr
	switch askType {
	case "tool", "command", "browser_action_launch":
		fmt.Fprintf(os.Stderr, "Waiting for approval (use --yolo for auto-approve): %s\n", askType)
	default:
		if h.Verbose {
			fmt.Fprintf(os.Stderr, "Question: %s\n", text)
		}
	}

	// Print the question
	fmt.Fprintf(h.Output, "\n%s\n", text)
	fmt.Fprint(h.Output, "Response (y/n/a): ")

	// Read response from stdin
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		// Default to yes if we can't read
		return "yesButtonClicked", nil
	}

	// Normalize response
	response = strings.ToLower(strings.TrimSpace(response))
	switch response {
	case "y", "yes", "":
		return "yesButtonClicked", nil
	case "n", "no":
		return "noButtonClicked", nil
	case "a", "always":
		return "yesButtonClicked", nil
	default:
		return "messageResponse", nil
	}
}

// OnInfo handles informational messages
func (h *PlainTextHandler) OnInfo(text string) {
	if !h.Verbose {
		return
	}

	ts := time.Now().UnixMilli()

	if h.JSONOutput {
		msg := ClineMessage{
			Ts:   ts,
			Type: "say",
			Say:  "info",
			Text: text,
		}
		h.outputJSON(msg)
	} else if h.Output != nil {
		fmt.Fprintf(h.Output, "[INFO] %s\n", text)
	}
}

// OnError handles error messages
func (h *PlainTextHandler) OnError(err error) {
	if h.Output == nil {
		h.Output = os.Stderr
	}

	ts := time.Now().UnixMilli()

	if h.JSONOutput {
		msg := ClineMessage{
			Ts:   ts,
			Type: "say",
			Say:  "error",
			Text: err.Error(),
		}
		h.outputJSON(msg)
	} else {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
}

// OnStatus handles status updates
func (h *PlainTextHandler) OnStatus(status string) {
	if !h.Verbose {
		return
	}

	ts := time.Now().UnixMilli()

	if h.JSONOutput {
		msg := ClineMessage{
			Ts:   ts,
			Type: "say",
			Say:  "info",
			Text: status,
		}
		h.outputJSON(msg)
	} else if h.Output != nil {
		fmt.Fprintf(h.Output, "[STATUS] %s\n", status)
	}
}

// OnProgress handles progress updates
func (h *PlainTextHandler) OnProgress(current, total int) {
	if !h.Verbose {
		return
	}

	ts := time.Now().UnixMilli()

	if h.JSONOutput {
		msg := ClineMessage{
			Ts:   ts,
			Type: "say",
			Say:  "task_progress",
			Text: fmt.Sprintf("%d/%d", current, total),
		}
		h.outputJSON(msg)
	} else if h.Output != nil {
		fmt.Fprintf(h.Output, "[PROGRESS] %d/%d\n", current, total)
	}
}

// outputJSON outputs a message as JSON
func (h *PlainTextHandler) outputJSON(msg ClineMessage) {
	if h.Output == nil {
		h.Output = os.Stdout
	}
	encoder := json.NewEncoder(h.Output)
	encoder.Encode(msg)
}

// WriteFinalOutput writes the final completion result (for non-JSON mode)
func (h *PlainTextHandler) WriteFinalOutput() {
	if h.JSONOutput || h.Verbose {
		return
	}

	// Find the last message (completion_result)
	var lastMsg string
	var maxTs int64
	for ts, msg := range h.processedMessages {
		if ts > maxTs {
			maxTs = ts
			lastMsg = msg
		}
	}

	if lastMsg != "" {
		fmt.Fprintln(h.Output, lastMsg)
	}
}

// JSONHandler handles messages in JSON format matching Node.js CLI exactly.
type JSONHandler struct {
	Output            io.Writer
	processedMessages map[int64]bool
}

// NewJSONHandler creates a new JSON handler
func NewJSONHandler(output io.Writer) *JSONHandler {
	if output == nil {
		output = os.Stdout
	}
	return &JSONHandler{
		Output:            output,
		processedMessages: make(map[int64]bool),
	}
}

// OnSay handles a SAY message from the assistant
func (h *JSONHandler) OnSay(sayType string, text string, partial bool) {
	ts := time.Now().UnixMilli()
	
	// Skip if we've already processed this timestamp
	if h.processedMessages[ts] {
		ts++ // Ensure unique timestamp
	}
	h.processedMessages[ts] = true

	msg := ClineMessage{
		Ts:      ts,
		Type:    "say",
		Say:     sayType,
		Text:    text,
		Partial: partial,
	}
	h.outputJSON(msg)
}

// OnAsk handles an ASK message that requires user response
func (h *JSONHandler) OnAsk(askType string, text string) (string, error) {
	ts := time.Now().UnixMilli()
	
	// Skip if we've already processed this timestamp
	if h.processedMessages[ts] {
		ts++ // Ensure unique timestamp
	}
	h.processedMessages[ts] = true

	msg := ClineMessage{
		Ts:   ts,
		Type: "ask",
		Ask:  askType,
		Text: text,
	}
	h.outputJSON(msg)

	// For JSON mode, auto-approve
	return "yesButtonClicked", nil
}

// OnInfo handles informational messages
func (h *JSONHandler) OnInfo(text string) {
	ts := time.Now().UnixMilli()
	if h.processedMessages[ts] {
		ts++
	}
	h.processedMessages[ts] = true

	msg := ClineMessage{
		Ts:   ts,
		Type: "say",
		Say:  "info",
		Text: text,
	}
	h.outputJSON(msg)
}

// OnError handles error messages
func (h *JSONHandler) OnError(err error) {
	ts := time.Now().UnixMilli()
	if h.processedMessages[ts] {
		ts++
	}
	h.processedMessages[ts] = true

	msg := ClineMessage{
		Ts:   ts,
		Type: "say",
		Say:  "error",
		Text: err.Error(),
	}
	h.outputJSON(msg)
}

// OnStatus handles status updates
func (h *JSONHandler) OnStatus(status string) {
	ts := time.Now().UnixMilli()
	if h.processedMessages[ts] {
		ts++
	}
	h.processedMessages[ts] = true

	msg := ClineMessage{
		Ts:   ts,
		Type: "say",
		Say:  "info",
		Text: status,
	}
	h.outputJSON(msg)
}

// OnProgress handles progress updates
func (h *JSONHandler) OnProgress(current, total int) {
	ts := time.Now().UnixMilli()
	if h.processedMessages[ts] {
		ts++
	}
	h.processedMessages[ts] = true

	msg := ClineMessage{
		Ts:   ts,
		Type: "say",
		Say:  "task_progress",
		Text: fmt.Sprintf("%d/%d", current, total),
	}
	h.outputJSON(msg)
}

// outputJSON outputs a message as JSON
func (h *JSONHandler) outputJSON(msg ClineMessage) {
	encoder := json.NewEncoder(h.Output)
	encoder.Encode(msg)
}

// TUIHandler handles messages in TUI format using Bubble Tea.
type TUIHandler struct {
	program *tea.Program
}

// NewTUIHandler creates a new TUI handler
func NewTUIHandler() *TUIHandler {
	return &TUIHandler{}
}

// OnSay handles a SAY message from the assistant
func (h *TUIHandler) OnSay(sayType string, text string, partial bool) {
	// Send message to TUI
	if h.program != nil {
		h.program.Send(tuiMessage{
			msgType: sayType,
			text:    text,
			partial: partial,
		})
	}
}

// OnAsk handles an ASK message that requires user response
func (h *TUIHandler) OnAsk(askType string, text string) (string, error) {
	// Send ask to TUI and wait for response
	if h.program != nil {
		h.program.Send(tuiAskMessage{
			askType: askType,
			text:    text,
		})
	}
	// For now, return auto-approve
	return "yesButtonClicked", nil
}

// OnInfo handles informational messages
func (h *TUIHandler) OnInfo(text string) {
	if h.program != nil {
		h.program.Send(tuiMessage{
			msgType: "info",
			text:    text,
		})
	}
}

// OnError handles error messages
func (h *TUIHandler) OnError(err error) {
	if h.program != nil {
		h.program.Send(tuiMessage{
			msgType: "error",
			text:    err.Error(),
		})
	}
}

// OnStatus handles status updates
func (h *TUIHandler) OnStatus(status string) {
	if h.program != nil {
		h.program.Send(tuiMessage{
			msgType: "status",
			text:    status,
		})
	}
}

// OnProgress handles progress updates
func (h *TUIHandler) OnProgress(current, total int) {
	if h.program != nil {
		h.program.Send(tuiProgressMessage{
			current: current,
			total:   total,
		})
	}
}

// SetProgram sets the Bubble Tea program
func (h *TUIHandler) SetProgram(program *tea.Program) {
	h.program = program
}

// TUI message types
type tuiMessage struct {
	msgType string
	text    string
	partial bool
}

type tuiAskMessage struct {
	askType string
	text    string
}

type tuiProgressMessage struct {
	current int
	total   int
}