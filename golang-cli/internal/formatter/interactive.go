// Package formatter provides output formatting capabilities for the Cline CLI.
package formatter

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cline/cline/golang-cli/internal/task"
	"github.com/cline/cline/golang-cli/internal/tui"
)

// InteractiveHandler provides interactive formatting with approval support
type InteractiveHandler struct {
	output      io.Writer
	verbose     bool
	autoApprove bool
	reader      *bufio.Reader

	// Approval handling
	pendingApproval      *ApprovalPrompt
	approvalMu           sync.Mutex
	approvalResponseChan chan string

	// Streaming state
	isStreaming  bool
	streamPaused bool
	streamMu     sync.RWMutex

	// Message tracking
	messages []task.Message

	// TUI integration
	program *tui.Program

	// Exit handler
	exitHandler func()
}

// ApprovalPrompt represents a pending approval request
type ApprovalPrompt struct {
	AskType  string
	Text     string
	Response chan<- string
	Received time.Time
}

// NewInteractiveHandler creates a new interactive handler
func NewInteractiveHandler(output io.Writer, verbose bool, autoApprove bool) *InteractiveHandler {
	return &InteractiveHandler{
		output:               output,
		verbose:              verbose,
		autoApprove:          autoApprove,
		reader:               bufio.NewReader(os.Stdin),
		messages:             make([]task.Message, 0),
		approvalResponseChan: make(chan string, 1),
	}
}

// SetProgram sets the TUI program for sending messages
func (h *InteractiveHandler) SetProgram(program *tui.Program) {
	h.program = program
}

// SetExitHandler sets the exit handler
func (h *InteractiveHandler) SetExitHandler(handler func()) {
	h.exitHandler = handler
}

// HandleMessage processes a single message
func (h *InteractiveHandler) HandleMessage(msg task.Message) error {
	switch msg.Type {
	case task.MessageTypeText:
		return h.OnText(msg.Content, msg.IsPartial)
	case task.MessageTypeTool:
		return h.handleToolMessage(msg)
	case task.MessageTypeAsk:
		return h.handleAskMessage(msg)
	case task.MessageTypeSay:
		return h.handleSayMessage(msg)
	case task.MessageTypeError:
		return h.OnError(fmt.Errorf("%s", msg.Content))
	default:
		// Log other message types if verbose
		if h.verbose {
			fmt.Fprintf(h.output, "[INFO] %s: %s\n", msg.Type, msg.Content)
		}
		return nil
	}
}

// OnText handles text messages
func (h *InteractiveHandler) OnText(content string, isPartial bool) error {
	h.streamMu.Lock()
	h.isStreaming = isPartial
	h.streamMu.Unlock()

	if !isPartial {
		// Complete message - print it
		fmt.Fprintln(h.output, content)
	} else {
		// Partial message - could update a spinner or progress indicator
		if h.program != nil {
			// Send to TUI for rendering
			h.program.Send(tui.StreamMessageMsg{
				Message: &tui.Message{
					Type:    tui.MessageTypeSay,
					Content: content,
					Partial: true,
				},
			})
		} else {
			// Fallback: print with carriage return for updates
			fmt.Fprintf(h.output, "\r%s", content)
		}
	}
	return nil
}

// OnToolUse handles tool use requests with approval
func (h *InteractiveHandler) OnToolUse(toolName string, params map[string]interface{}) (bool, error) {
	if h.autoApprove {
		return true, nil
	}

	// Check if we have a stored approval for this tool
	if h.isToolApproved(toolName) {
		return true, nil
	}

	// Pause streaming during approval
	h.pauseStreaming()

	// Create approval prompt
	responseChan := make(chan string, 1)
	approval := &ApprovalPrompt{
		AskType:  "tool",
		Text:     h.formatToolPrompt(toolName, params),
		Response: responseChan,
		Received: time.Now(),
	}

	h.approvalMu.Lock()
	h.pendingApproval = approval
	h.approvalMu.Unlock()

	// Send to TUI if available
	if h.program != nil {
		h.program.Send(tui.ApprovalRequestMsg{
			AskType:  "tool",
			Text:     approval.Text,
			Response: responseChan,
		})
	} else {
		// Fallback: console-based approval
		h.consoleApproval(approval)
	}

	// Wait for response with timeout
	select {
	case response := <-responseChan:
		h.approvalMu.Lock()
		h.pendingApproval = nil
		h.approvalMu.Unlock()
		h.resumeStreaming()

		switch response {
		case "yesButtonClicked", "y":
			return true, nil
		case "alwaysButtonClicked", "a":
			h.approveTool(toolName)
			return true, nil
		default:
			return false, nil
		}
	case <-time.After(5 * time.Minute):
		h.approvalMu.Lock()
		h.pendingApproval = nil
		h.approvalMu.Unlock()
		h.resumeStreaming()
		return false, fmt.Errorf("approval timeout")
	}
}

// OnToolResult handles tool execution results
func (h *InteractiveHandler) OnToolResult(toolName string, result string, success bool) error {
	status := "✓"
	if !success {
		status = "✗"
	}
	fmt.Fprintf(h.output, "%s Tool %s completed\n", status, toolName)
	if h.verbose && result != "" {
		fmt.Fprintf(h.output, "Result: %s\n", result)
	}
	return nil
}

// OnAsk handles ask prompts
func (h *InteractiveHandler) OnAsk(promptType string, question string) (string, error) {
	// Pause streaming during approval
	h.pauseStreaming()

	responseChan := make(chan string, 1)
	approval := &ApprovalPrompt{
		AskType:  promptType,
		Text:     question,
		Response: responseChan,
		Received: time.Now(),
	}

	h.approvalMu.Lock()
	h.pendingApproval = approval
	h.approvalMu.Unlock()

	// Send to TUI if available
	if h.program != nil {
		h.program.Send(tui.ApprovalRequestMsg{
			AskType:  promptType,
			Text:     question,
			Response: responseChan,
		})
	} else {
		// Fallback: console-based input
		h.consoleAsk(approval)
	}

	// Wait for response with timeout
	select {
	case response := <-responseChan:
		h.approvalMu.Lock()
		h.pendingApproval = nil
		h.approvalMu.Unlock()
		h.resumeStreaming()
		return response, nil
	case <-time.After(5 * time.Minute):
		h.approvalMu.Lock()
		h.pendingApproval = nil
		h.approvalMu.Unlock()
		h.resumeStreaming()
		return "", fmt.Errorf("response timeout")
	}
}

// OnSay handles say messages
func (h *InteractiveHandler) OnSay(sayType string, content string, partial bool) error {
	switch sayType {
	case "text":
		return h.OnText(content, partial)
	case "error":
		return h.OnError(fmt.Errorf("%s", content))
	case "command":
		fmt.Fprintf(h.output, "\n🖥  %s\n", content)
	case "tool":
		fmt.Fprintf(h.output, "\n🔧 %s\n", content)
	default:
		if !partial {
			fmt.Fprintln(h.output, content)
		}
	}
	return nil
}

// OnCommand handles command execution requests
func (h *InteractiveHandler) OnCommand(command string, requiresApproval bool) (string, error) {
	if !requiresApproval || h.autoApprove {
		fmt.Fprintf(h.output, "\n🖥  Executing: %s\n", command)
		return "approved", nil
	}

	return h.OnAsk("command", fmt.Sprintf("Execute command: %s?", command))
}

// OnCommandOutput handles command output
func (h *InteractiveHandler) OnCommandOutput(output string, isComplete bool) error {
	if h.verbose || !isComplete {
		fmt.Fprint(h.output, output)
	}
	return nil
}

// OnError handles errors
func (h *InteractiveHandler) OnError(err error) error {
	fmt.Fprintf(h.output, "\n❌ Error: %s\n", err.Error())
	return err
}

// OnInfo handles info messages
func (h *InteractiveHandler) OnInfo(message string) error {
	if h.verbose {
		fmt.Fprintf(h.output, "[INFO] %s\n", message)
	}
	return nil
}

// OnStatus handles status messages
func (h *InteractiveHandler) OnStatus(status string) error {
	if h.verbose {
		fmt.Fprintf(h.output, "[STATUS] %s\n", status)
	}
	return nil
}

// OnProgress handles progress updates
func (h *InteractiveHandler) OnProgress(current, total int) error {
	if total > 0 {
		percent := float64(current) * 100.0 / float64(total)
		fmt.Fprintf(h.output, "\r[PROGRESS] %d/%d (%.1f%%)", current, total, percent)
	}
	return nil
}

// OnCheckpoint handles checkpoint events
func (h *InteractiveHandler) OnCheckpoint(checkpointID string, action string) error {
	if h.verbose {
		fmt.Fprintf(h.output, "[CHECKPOINT] %s: %s\n", action, checkpointID)
	}
	return nil
}

// OnBrowserAction handles browser actions
func (h *InteractiveHandler) OnBrowserAction(action string, url string) (string, error) {
	fmt.Fprintf(h.output, "\n🌐 Browser action: %s %s\n", action, url)
	return "approved", nil // Auto-approve browser actions for now
}

// OnMCPRequest handles MCP tool requests
func (h *InteractiveHandler) OnMCPRequest(server string, tool string, params map[string]interface{}) (string, error) {
	if h.autoApprove {
		return "approved", nil
	}
	return h.OnAsk("mcp", fmt.Sprintf("Execute MCP tool %s from server %s?", tool, server))
}

// OnCompletion handles task completion
func (h *InteractiveHandler) OnCompletion(success bool, summary string) error {
	if success {
		fmt.Fprintln(h.output, "\n✓ Task completed successfully")
	} else {
		fmt.Fprintln(h.output, "\n✗ Task failed")
	}
	if summary != "" {
		fmt.Fprintln(h.output, summary)
	}
	return nil
}

// Helper methods

func (h *InteractiveHandler) handleToolMessage(msg task.Message) error {
	toolName, _ := msg.Metadata["tool"].(string)
	params, _ := msg.Metadata["params"].(map[string]interface{})

	approved, err := h.OnToolUse(toolName, params)
	if err != nil {
		return err
	}

	if approved {
		fmt.Fprintf(h.output, "\n🔧 Using %s...\n", toolName)
	}

	return nil
}

func (h *InteractiveHandler) handleAskMessage(msg task.Message) error {
	promptType, _ := msg.Metadata["ask_type"].(string)
	response, err := h.OnAsk(promptType, msg.Content)
	if err != nil {
		return err
	}

	// Store response in metadata for next message
	msg.Metadata["response"] = response
	return nil
}

func (h *InteractiveHandler) handleSayMessage(msg task.Message) error {
	sayType, _ := msg.Metadata["say_type"].(string)
	partial, _ := msg.Metadata["partial"].(bool)
	return h.OnSay(sayType, msg.Content, partial)
}

func (h *InteractiveHandler) formatToolPrompt(toolName string, params map[string]interface{}) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Approve tool: %s?\n", toolName))

	// Format parameters
	if len(params) > 0 {
		sb.WriteString("\nParameters:\n")
		for key, value := range params {
			sb.WriteString(fmt.Sprintf("  %s: %v\n", key, value))
		}
	}

	sb.WriteString("\n[y]es, [n]o, [a]lways: ")
	return sb.String()
}

func (h *InteractiveHandler) consoleApproval(approval *ApprovalPrompt) {
	fmt.Fprintln(h.output, approval.Text)

	// Read user input
	input, err := h.reader.ReadString('\n')
	if err != nil {
		approval.Response <- "no"
		return
	}

	input = strings.TrimSpace(strings.ToLower(input))

	switch input {
	case "y", "yes":
		approval.Response <- "yesButtonClicked"
	case "a", "always":
		approval.Response <- "alwaysButtonClicked"
	default:
		approval.Response <- "noButtonClicked"
	}
}

func (h *InteractiveHandler) consoleAsk(approval *ApprovalPrompt) {
	fmt.Fprintln(h.output, approval.Text)
	fmt.Fprint(h.output, "Response: ")

	input, err := h.reader.ReadString('\n')
	if err != nil {
		approval.Response <- ""
		return
	}

	approval.Response <- strings.TrimSpace(input)
}

func (h *InteractiveHandler) pauseStreaming() {
	h.streamMu.Lock()
	defer h.streamMu.Unlock()
	h.streamPaused = true
}

func (h *InteractiveHandler) resumeStreaming() {
	h.streamMu.Lock()
	defer h.streamMu.Unlock()
	h.streamPaused = false
}

func (h *InteractiveHandler) isStreamingPaused() bool {
	h.streamMu.RLock()
	defer h.streamMu.RUnlock()
	return h.streamPaused
}

// Tool approval storage
var (
	approvedTools   = make(map[string]bool)
	approvedToolsMu sync.RWMutex
)

func (h *InteractiveHandler) isToolApproved(toolName string) bool {
	approvedToolsMu.RLock()
	defer approvedToolsMu.RUnlock()
	return approvedTools[toolName]
}

func (h *InteractiveHandler) approveTool(toolName string) {
	approvedToolsMu.Lock()
	defer approvedToolsMu.Unlock()
	approvedTools[toolName] = true
}

// Ensure InteractiveHandler implements MessageHandler
var _ task.MessageHandler = (*InteractiveHandler)(nil)