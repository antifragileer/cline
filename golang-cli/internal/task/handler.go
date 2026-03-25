// Package task provides task execution functionality for the Cline CLI.
package task

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

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

// PlainTextHandler handles messages in plain text format.
type PlainTextHandler struct {
	Verbose     bool
	JSONOutput  bool
	Output      io.Writer
	AutoApprove bool
}

// OnSay handles a SAY message from the assistant
func (h *PlainTextHandler) OnSay(sayType string, text string, partial bool) {
	if h.Output == nil {
		h.Output = os.Stdout
	}

	// Don't print partial messages in plain text mode
	if partial {
		return
	}

	switch sayType {
	case "text":
		fmt.Fprintln(h.Output, text)
	case "error":
		fmt.Fprintf(h.Output, "Error: %s\n", text)
	case "command":
		fmt.Fprintf(h.Output, "Command: %s\n", text)
	case "command_output":
		fmt.Fprintf(h.Output, "%s\n", text)
	case "tool":
		fmt.Fprintf(h.Output, "Tool: %s\n", text)
	case "thinking":
		if h.Verbose {
			fmt.Fprintf(h.Output, "Thinking: %s\n", text)
		}
	case "completion_result":
		fmt.Fprintf(h.Output, "\nResult: %s\n", text)
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

	// Auto-approve if enabled
	if h.AutoApprove {
		return "yesButtonClicked", nil
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
	if h.Verbose && h.Output != nil {
		fmt.Fprintf(h.Output, "[INFO] %s\n", text)
	}
}

// OnError handles error messages
func (h *PlainTextHandler) OnError(err error) {
	if h.Output == nil {
		h.Output = os.Stderr
	}
	fmt.Fprintf(h.Output, "Error: %v\n", err)
}

// OnStatus handles status updates
func (h *PlainTextHandler) OnStatus(status string) {
	if h.Verbose && h.Output != nil {
		fmt.Fprintf(h.Output, "[STATUS] %s\n", status)
	}
}

// OnProgress handles progress updates
func (h *PlainTextHandler) OnProgress(current, total int) {
	if h.Verbose && h.Output != nil {
		fmt.Fprintf(h.Output, "[PROGRESS] %d/%d\n", current, total)
	}
}

// JSONHandler handles messages in JSON format.
type JSONHandler struct {
	Output io.Writer
}

// OnSay handles a SAY message from the assistant
func (h *JSONHandler) OnSay(sayType string, text string, partial bool) {
	h.outputJSON(map[string]interface{}{
		"type":    "say",
		"sayType": sayType,
		"text":    text,
		"partial": partial,
	})
}

// OnAsk handles an ASK message that requires user response
func (h *JSONHandler) OnAsk(askType string, text string) (string, error) {
	h.outputJSON(map[string]interface{}{
		"type":    "ask",
		"askType": askType,
		"text":    text,
	})

	// For JSON mode, return auto-approve response
	return "yesButtonClicked", nil
}

// OnInfo handles informational messages
func (h *JSONHandler) OnInfo(text string) {
	h.outputJSON(map[string]interface{}{
		"type": "info",
		"text": text,
	})
}

// OnError handles error messages
func (h *JSONHandler) OnError(err error) {
	h.outputJSON(map[string]interface{}{
		"type":  "error",
		"error": err.Error(),
	})
}

// OnStatus handles status updates
func (h *JSONHandler) OnStatus(status string) {
	h.outputJSON(map[string]interface{}{
		"type":   "status",
		"status": status,
	})
}

// OnProgress handles progress updates
func (h *JSONHandler) OnProgress(current, total int) {
	h.outputJSON(map[string]interface{}{
		"type":    "progress",
		"current": current,
		"total":   total,
	})
}

// outputJSON outputs a JSON message
func (h *JSONHandler) outputJSON(data map[string]interface{}) {
	if h.Output == nil {
		h.Output = os.Stdout
	}
	encoder := json.NewEncoder(h.Output)
	encoder.Encode(data)
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