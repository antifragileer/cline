// Package formatter provides plain text output formatting for the Cline CLI.
// This file implements complete plain text output with proper exit code handling.
package formatter

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/cline/cline/golang-cli/internal/exit"
	"github.com/cline/cline/golang-cli/internal/task"
)

// PlainFormatter handles plain text output formatting with full message type support
type PlainFormatter struct {
	output      io.Writer
	errOutput   io.Writer
	useColor    bool
	verbose     bool
	exitHandler *exit.Handler
	
	// Track state for proper formatting
	inProgress  bool
	lastWasSame bool
}

// NewPlainFormatter creates a new plain text formatter
func NewPlainFormatter(output, errOutput io.Writer, useColor, verbose bool) *PlainFormatter {
	if output == nil {
		output = os.Stdout
	}
	if errOutput == nil {
		errOutput = os.Stderr
	}

	return &PlainFormatter{
		output:    output,
		errOutput: errOutput,
		useColor:  useColor,
		verbose:   verbose,
	}
}

// SetExitHandler sets the exit handler for proper exit code management
func (f *PlainFormatter) SetExitHandler(handler *exit.Handler) {
	f.exitHandler = handler
}

// FormatSayMessage formats a SAY message in plain text
func (f *PlainFormatter) FormatSayMessage(sayType string, text string, partial bool) error {
	// Skip partial messages in plain text mode (they're for streaming UI)
	if partial {
		return nil
	}

	switch sayType {
	case "text":
		f.printText(text)
	case "error":
		f.printError(text)
		if f.exitHandler != nil {
			f.exitHandler.SetExitCode(exit.TaskFailed)
		}
	case "task":
		f.printTask(text)
	case "api_req_started":
		if f.verbose {
			f.printInfo("API request started")
		}
	case "api_req_finished":
		if f.verbose {
			f.printInfo("API request finished")
		}
	case "command":
		f.printCommand(text)
	case "command_output":
		f.printCommandOutput(text)
	case "tool":
		f.printTool(text)
	case "completion_result":
		f.printCompletionResult(text)
	case "thinking", "reasoning":
		if f.verbose {
			f.printThinking(text)
		}
	case "browser_action":
		if f.verbose {
			f.printInfo(fmt.Sprintf("Browser action: %s", text))
		}
	case "browser_action_result":
		if f.verbose {
			f.printInfo(fmt.Sprintf("Browser result: %s", text))
		}
	case "checkpoint_created":
		if f.verbose {
			f.printInfo(fmt.Sprintf("Checkpoint created: %s", text))
		}
	case "mcp_server_request_started":
		if f.verbose {
			f.printInfo(fmt.Sprintf("MCP request: %s", text))
		}
	case "mcp_server_response":
		if f.verbose {
			f.printInfo(fmt.Sprintf("MCP response: %s", text))
		}
	case "user_feedback":
		if f.verbose {
			f.printInfo(fmt.Sprintf("Feedback: %s", text))
		}
	case "diff_error":
		f.printError(fmt.Sprintf("Diff error: %s", text))
	case "shell_integration_warning":
		f.printWarning(text)
	case "clineignore_error":
		f.printError(fmt.Sprintf(".clineignore error: %s", text))
	case "command_permission_denied":
		f.printError(fmt.Sprintf("Permission denied: %s", text))
		if f.exitHandler != nil {
			f.exitHandler.SetExitCode(exit.PermissionDenied)
		}
	case "info":
		if f.verbose {
			f.printInfo(text)
		}
	case "task_progress":
		if f.verbose {
			f.printProgress(text)
		}
	default:
		if f.verbose {
			f.printInfo(fmt.Sprintf("[%s] %s", sayType, text))
		}
	}

	return nil
}

// FormatAskMessage formats an ASK message in plain text
func (f *PlainFormatter) FormatAskMessage(askType string, text string) (string, error) {
	// Print the question/prompt
	fmt.Fprintln(f.output)
	
	switch askType {
	case "command":
		f.printPrompt(fmt.Sprintf("Approve command execution?\n%s", text))
	case "tool":
		f.printPrompt(fmt.Sprintf("Approve tool use?\n%s", text))
	case "browser_action_launch":
		f.printPrompt(fmt.Sprintf("Approve browser action?\n%s", text))
	case "completion_result":
		f.printPrompt(fmt.Sprintf("Task completed:\n%s", text))
	case "followup":
		f.printPrompt(fmt.Sprintf("Follow-up question:\n%s", text))
	case "plan_mode_respond":
		f.printPrompt(fmt.Sprintf("Plan mode:\n%s", text))
	case "mistake_limit_reached":
		f.printWarning(fmt.Sprintf("Mistake limit reached:\n%s", text))
		// Don't ask for input, just return
		return "noButtonClicked", nil
	case "api_req_failed":
		f.printError(fmt.Sprintf("API request failed:\n%s", text))
		if f.exitHandler != nil {
			f.exitHandler.SetExitCode(exit.ConnectionError)
		}
		return "noButtonClicked", nil
	default:
		f.printPrompt(text)
	}

	// Read response from user
	fmt.Fprint(f.output, "Response (y/n/a): ")
	
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		// Default to yes on error
		return "yesButtonClicked", nil
	}

	response = strings.ToLower(strings.TrimSpace(response))
	
	switch response {
	case "y", "yes", "":
		return "yesButtonClicked", nil
	case "n", "no":
		return "noButtonClicked", nil
	case "a", "always":
		return "yesButtonClicked", nil
	default:
		// Treat any other input as a message response
		return "messageResponse", nil
	}
}

// FormatError formats an error message
func (f *PlainFormatter) FormatError(err error) error {
	f.printError(err.Error())
	if f.exitHandler != nil {
		f.exitHandler.SetExitCode(exit.GeneralError)
	}
	return nil
}

// FormatStatus formats a status message
func (f *PlainFormatter) FormatStatus(status string) error {
	if f.verbose {
		f.printInfo(status)
	}
	return nil
}

// FormatProgress formats a progress update
func (f *PlainFormatter) FormatProgress(current, total int, message string) error {
	if f.verbose {
		bar := f.renderProgressBar(current, total, 30)
		percentage := 0.0
		if total > 0 {
			percentage = float64(current) * 100.0 / float64(total)
		}
		fmt.Fprintf(f.output, "\r%s %3.0f%% %s", f.styleInfo("[PROGRESS]"), percentage, bar)
		if current >= total {
			fmt.Fprintln(f.output) // New line when complete
		}
	}
	return nil
}

// FormatTimeout formats a timeout message and sets exit code
func (f *PlainFormatter) FormatTimeout(duration time.Duration) error {
	f.printError(fmt.Sprintf("Operation timed out after %v", duration))
	if f.exitHandler != nil {
		f.exitHandler.SetExitCode(exit.Timeout)
	}
	return nil
}

// FormatInterrupted formats an interrupted message and sets exit code
func (f *PlainFormatter) FormatInterrupted() error {
	f.printError("Operation interrupted")
	if f.exitHandler != nil {
		f.exitHandler.SetExitCode(exit.Interrupted)
	}
	return nil
}

// Flush flushes any buffered output
func (f *PlainFormatter) Flush() error {
	if flusher, ok := f.output.(interface{ Flush() error }); ok {
		return flusher.Flush()
	}
	return nil
}

// printText prints plain text output
func (f *PlainFormatter) printText(text string) {
	fmt.Fprintln(f.output, text)
}

// printTask prints a task description
func (f *PlainFormatter) printTask(text string) {
	fmt.Fprintf(f.output, "%s %s\n", f.styleBold("Task:"), text)
}

// printCommand prints a command execution message
func (f *PlainFormatter) printCommand(cmd string) {
	fmt.Fprintf(f.output, "%s %s\n", f.styleInfo("Command:"), f.styleCommand(cmd))
}

// printCommandOutput prints command output
func (f *PlainFormatter) printCommandOutput(output string) {
	// Indent output for readability
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		fmt.Fprintf(f.output, "  %s\n", line)
	}
}

// printTool prints a tool use message
func (f *PlainFormatter) printTool(tool string) {
	fmt.Fprintf(f.output, "%s %s\n", f.styleInfo("Tool:"), f.styleTool(tool))
}

// printCompletionResult prints the final completion result
func (f *PlainFormatter) printCompletionResult(result string) {
	fmt.Fprintln(f.output)
	fmt.Fprintf(f.output, "%s\n", f.styleSuccess("=== Completion Result ==="))
	fmt.Fprintln(f.output, result)
	fmt.Fprintf(f.output, "%s\n", f.styleSuccess("========================"))
	fmt.Fprintln(f.output)
}

// printThinking prints thinking/reasoning output
func (f *PlainFormatter) printThinking(text string) {
	fmt.Fprintf(f.output, "%s %s\n", f.styleDim("Thinking:"), f.styleDim(text))
}

// printPrompt prints an interactive prompt
func (f *PlainFormatter) printPrompt(text string) {
	fmt.Fprintf(f.output, "\n%s\n", f.styleQuestion(text))
}

// printError prints an error message
func (f *PlainFormatter) printError(text string) {
	fmt.Fprintf(f.errOutput, "%s: %s\n", f.styleError("Error"), text)
}

// printWarning prints a warning message
func (f *PlainFormatter) printWarning(text string) {
	fmt.Fprintf(f.errOutput, "%s: %s\n", f.styleWarning("Warning"), text)
}

// printInfo prints an info message
func (f *PlainFormatter) printInfo(text string) {
	fmt.Fprintf(f.output, "%s %s\n", f.styleInfo("[INFO]"), text)
}

// printProgress prints a progress message
func (f *PlainFormatter) printProgress(text string) {
	fmt.Fprintf(f.output, "%s %s\n", f.styleInfo("[PROGRESS]"), text)
}

// renderProgressBar renders a text-based progress bar
func (f *PlainFormatter) renderProgressBar(current, total, width int) string {
	if total <= 0 {
		return "[" + strings.Repeat("-", width) + "]"
	}
	filled := int(float64(current) * float64(width) / float64(total))
	if filled > width {
		filled = width
	}
	empty := width - filled
	return "[" + strings.Repeat("=", filled) + strings.Repeat("-", empty) + "]"
}

// Color/style functions
func (f *PlainFormatter) styleError(text string) string {
	if !f.useColor {
		return text
	}
	return fmt.Sprintf("\033[31m%s\033[0m", text) // Red
}

func (f *PlainFormatter) styleWarning(text string) string {
	if !f.useColor {
		return text
	}
	return fmt.Sprintf("\033[33m%s\033[0m", text) // Yellow
}

func (f *PlainFormatter) styleSuccess(text string) string {
	if !f.useColor {
		return text
	}
	return fmt.Sprintf("\033[32m%s\033[0m", text) // Green
}

func (f *PlainFormatter) styleInfo(text string) string {
	if !f.useColor {
		return text
	}
	return fmt.Sprintf("\033[36m%s\033[0m", text) // Cyan
}

func (f *PlainFormatter) styleCommand(text string) string {
	if !f.useColor {
		return text
	}
	return fmt.Sprintf("\033[33m%s\033[0m", text) // Yellow
}

func (f *PlainFormatter) styleTool(text string) string {
	if !f.useColor {
		return text
	}
	return fmt.Sprintf("\033[35m%s\033[0m", text) // Magenta
}

func (f *PlainFormatter) styleQuestion(text string) string {
	if !f.useColor {
		return text
	}
	return fmt.Sprintf("\033[1m\033[35m%s\033[0m", text) // Bold Magenta
}

func (f *PlainFormatter) styleDim(text string) string {
	if !f.useColor {
		return text
	}
	return fmt.Sprintf("\033[90m%s\033[0m", text) // Gray
}

func (f *PlainFormatter) styleBold(text string) string {
	if !f.useColor {
		return text
	}
	return fmt.Sprintf("\033[1m%s\033[0m", text) // Bold
}

// PlainHandler is a task.MessageHandler implementation for plain text output
type PlainHandler struct {
	formatter   *PlainFormatter
	autoApprove bool
	verbose     bool
	// Track responses for non-interactive mode
	responseCount int
}

// NewPlainHandler creates a new plain text handler for task execution
func NewPlainHandler(output io.Writer, verbose, autoApprove bool) *PlainHandler {
	return &PlainHandler{
		formatter:   NewPlainFormatter(output, os.Stderr, true, verbose),
		autoApprove: autoApprove,
		verbose:     verbose,
	}
}

// SetExitHandler sets the exit handler
func (h *PlainHandler) SetExitHandler(handler *exit.Handler) {
	h.formatter.SetExitHandler(handler)
}

// HandleMessage implements task.MessageHandler
func (h *PlainHandler) HandleMessage(msg task.Message) error {
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
		return h.OnError(fmt.Errorf(msg.Content))
	default:
		// For other types, just show as info
		if h.verbose {
			h.OnInfo(msg.Content)
		}
		return nil
	}
}

// OnSay handles SAY messages
func (h *PlainHandler) OnSay(sayType string, text string, partial bool) error {
	// Skip partial messages unless verbose
	if partial && !h.verbose {
		return nil
	}

	// Map the say type to the formatter
	return h.formatter.FormatSayMessage(sayType, text, partial)
}

// OnAsk handles ASK messages
func (h *PlainHandler) OnAsk(askType string, text string) (string, error) {
	// Auto-approve if enabled
	if h.autoApprove {
		if h.verbose {
			h.formatter.printInfo(fmt.Sprintf("Auto-approved: %s", askType))
		}
		return "yesButtonClicked", nil
	}

	// Otherwise, prompt the user
	return h.formatter.FormatAskMessage(askType, text)
}

// OnInfo handles info messages
func (h *PlainHandler) OnInfo(text string) error {
	return h.formatter.FormatStatus(text)
}

// OnError handles error messages
func (h *PlainHandler) OnError(err error) error {
	return h.formatter.FormatError(err)
}

// OnStatus handles status messages
func (h *PlainHandler) OnStatus(status string) error {
	return h.formatter.FormatStatus(status)
}

// OnProgress handles progress messages
func (h *PlainHandler) OnProgress(current, total int) error {
	return h.formatter.FormatProgress(current, total, "")
}

// OnText handles text messages
func (h *PlainHandler) OnText(content string, isPartial bool) error {
	return h.OnSay("text", content, isPartial)
}

// OnToolUse handles tool use requests
func (h *PlainHandler) OnToolUse(toolName string, params map[string]interface{}) (bool, error) {
	if h.autoApprove {
		return true, nil
	}
	// In plain text mode, prompt the user
	h.formatter.printPrompt(fmt.Sprintf("Approve tool %s?", toolName))
	return true, nil
}

// OnToolResult handles tool execution results
func (h *PlainHandler) OnToolResult(toolName string, result string, success bool) error {
	if h.verbose || !success {
		if success {
			h.formatter.printInfo(fmt.Sprintf("Tool %s succeeded", toolName))
		} else {
			h.formatter.printError(fmt.Sprintf("Tool %s failed: %s", toolName, result))
		}
	}
	return nil
}

// OnCommand handles command execution requests
func (h *PlainHandler) OnCommand(command string, requiresApproval bool) (string, error) {
	if !requiresApproval || h.autoApprove {
		return "execute", nil
	}
	return "", fmt.Errorf("command approval required")
}

// OnCommandOutput handles command output
func (h *PlainHandler) OnCommandOutput(output string, isComplete bool) error {
	if h.verbose {
		h.formatter.printCommandOutput(output)
	}
	return nil
}

// OnCheckpoint handles checkpoint events
func (h *PlainHandler) OnCheckpoint(checkpointID string, action string) error {
	if h.verbose {
		h.formatter.printInfo(fmt.Sprintf("Checkpoint: %s", checkpointID))
	}
	return nil
}

// OnBrowserAction handles browser actions
func (h *PlainHandler) OnBrowserAction(action string, url string) (string, error) {
	if h.verbose {
		h.formatter.printInfo(fmt.Sprintf("Browser action: %s %s", action, url))
	}
	return "", nil
}

// OnMCPRequest handles MCP tool requests
func (h *PlainHandler) OnMCPRequest(server string, tool string, params map[string]interface{}) (string, error) {
	if h.verbose {
		h.formatter.printInfo(fmt.Sprintf("MCP request: %s/%s", server, tool))
	}
	return "", nil
}

// OnCompletion handles task completion
func (h *PlainHandler) OnCompletion(success bool, summary string) error {
	h.formatter.printCompletionResult(summary)
	return nil
}

// Flush flushes the formatter output
func (h *PlainHandler) Flush() error {
	return h.formatter.Flush()
}

// ScriptingHandler is a plain text handler optimized for scripting/automation
type ScriptingHandler struct {
	*PlainHandler
	output      io.Writer
	exitHandler *exit.Handler
}

// NewScriptingHandler creates a new scripting handler
func NewScriptingHandler(output io.Writer, verbose bool) *ScriptingHandler {
	return &ScriptingHandler{
		PlainHandler: NewPlainHandler(output, verbose, true), // Auto-approve
		output:       output,
	}
}

// SetExitHandler sets the exit handler
func (h *ScriptingHandler) SetExitHandler(handler *exit.Handler) {
	h.exitHandler = handler
	h.PlainHandler.SetExitHandler(handler)
}

// OnSay handles SAY messages with scripting-specific behavior
func (h *ScriptingHandler) OnSay(sayType string, text string, partial bool) {
	// Skip partial messages in scripting mode
	if partial {
		return
	}

	// Only output specific message types in scripting mode
	switch sayType {
	case "text":
		// Main output
		fmt.Fprintln(h.output, text)
	case "completion_result":
		// Final result
		fmt.Fprintln(h.output, text)
	case "error":
		fmt.Fprintf(os.Stderr, "Error: %s\n", text)
		if h.exitHandler != nil {
			h.exitHandler.SetExitCode(exit.TaskFailed)
		}
	case "command", "tool":
		if h.verbose {
			h.formatter.printInfo(fmt.Sprintf("%s: %s", sayType, text))
		}
	}
}

// OnAsk handles ASK messages in scripting mode (always auto-approve)
func (h *ScriptingHandler) OnAsk(askType string, text string) (string, error) {
	if h.verbose {
		h.formatter.printInfo(fmt.Sprintf("Auto-approved %s: %s", askType, text))
	}
	return "yesButtonClicked", nil
}

// Exit codes for scripting
const (
	// ScriptingExitSuccess indicates success
	ScriptingExitSuccess = 0
	// ScriptingExitError indicates general error
	ScriptingExitError = 1
	// ScriptingExitTimeout indicates timeout
	ScriptingExitTimeout = 124
	// ScriptingExitInterrupted indicates interruption
	ScriptingExitInterrupted = 130
)