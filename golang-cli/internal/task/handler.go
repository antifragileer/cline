// Package task provides task execution functionality for the Cline CLI.
package task

import (
	"encoding/json"
)

// MessageHandler handles messages from the Cline core extension
type MessageHandler interface {
	// OnSay is called when the assistant sends a message
	OnSay(sayType, text string, partial bool)
	// OnAsk is called when the assistant asks for user input
	// Returns the user's response
	OnAsk(askType, text string) (string, error)
	// OnInfo is called for informational messages
	OnInfo(msg string)
	// OnError is called when an error occurs
	OnError(err error)
	// OnComplete is called when the task is complete
	OnComplete(success bool, message string)
}

// PlainTextHandler is a MessageHandler that outputs plain text
type PlainTextHandler struct {
	Verbose    bool
	JSONOutput bool
	Output     interface {
		Write(p []byte) (n int, err error)
	}
	AutoApprove bool
}

// OnSay handles a SAY message
func (h *PlainTextHandler) OnSay(sayType, text string, partial bool) {
	if h.JSONOutput {
		// JSON output is handled separately
		return
	}

	// Skip partial messages in plain text mode (they're for TUI)
	if partial {
		return
	}

	// Format output based on type
	switch sayType {
	case "text":
		h.write(text)
	case "error":
		h.writeError("Error: " + text)
	case "api_req_started":
		if h.Verbose {
			h.write("→ API request started")
		}
	case "api_req_finished":
		if h.Verbose {
			h.write("→ API request finished")
		}
	case "command":
		h.write("→ Executing: " + text)
	case "command_output":
		h.write(text)
	case "tool":
		h.write("→ Tool: " + text)
	case "success":
		h.write("✓ " + text)
	case "completion_result":
		h.write("✓ Task completed")
		h.write(text)
	case "checkpoint_created":
		if h.Verbose {
			h.write("→ Checkpoint created")
		}
	case "checkpoint_restored":
		if h.Verbose {
			h.write("→ Checkpoint restored")
		}
	default:
		if h.Verbose {
			h.write("[" + sayType + "] " + text)
		}
	}
}

// OnAsk handles an ASK message
func (h *PlainTextHandler) OnAsk(askType, text string) (string, error) {
	if h.AutoApprove {
		return "yesButtonClicked", nil
	}

	// In plain text mode with piped input, we can't interactively ask
	// So we auto-approve or return a default response
	switch askType {
	case "tool_approval":
		h.write("→ Tool approval required (auto-approved in plain mode)")
		return "yesButtonClicked", nil
	case "command_approval":
		h.write("→ Command approval required (auto-approved in plain mode)")
		return "yesButtonClicked", nil
	case "followup":
		// For followup questions, we need input but can't get it interactively
		// Return empty to indicate we can't answer
		h.write("? " + text)
		return "", nil
	default:
		// Auto-approve other asks in plain mode
		return "yesButtonClicked", nil
	}
}

// OnInfo handles informational messages
func (h *PlainTextHandler) OnInfo(msg string) {
	if h.Verbose && !h.JSONOutput {
		h.write("ℹ " + msg)
	}
}

// OnError handles errors
func (h *PlainTextHandler) OnError(err error) {
	if h.JSONOutput {
		// JSON errors handled separately
		return
	}
	h.writeError("Error: " + err.Error())
}

// OnComplete handles task completion
func (h *PlainTextHandler) OnComplete(success bool, message string) {
	if h.JSONOutput {
		return
	}
	if success {
		h.write("✓ " + message)
	} else {
		h.writeError("✗ " + message)
	}
}

func (h *PlainTextHandler) write(s string) {
	if h.Output != nil {
		h.Output.Write([]byte(s + "\n"))
	}
}

func (h *PlainTextHandler) writeError(s string) {
	if h.Output != nil {
		// Write to stderr via prefix
		h.Output.Write([]byte(s + "\n"))
	}
}

// JSONHandler is a MessageHandler that outputs structured JSON
type JSONHandler struct {
	Output interface {
		Write(p []byte) (n int, err error)
	}
	Messages []map[string]interface{}
}

// OnSay handles a SAY message
func (h *JSONHandler) OnSay(sayType, text string, partial bool) {
	msg := map[string]interface{}{
		"type":    "say",
		"sayType": sayType,
		"text":    text,
		"partial": partial,
	}
	h.Messages = append(h.Messages, msg)
}

// OnAsk handles an ASK message
func (h *JSONHandler) OnAsk(askType, text string) (string, error) {
	msg := map[string]interface{}{
		"type":    "ask",
		"askType": askType,
		"text":    text,
	}
	h.Messages = append(h.Messages, msg)
	// Auto-approve in JSON mode
	return "yesButtonClicked", nil
}

// OnInfo handles informational messages
func (h *JSONHandler) OnInfo(msg string) {
	h.Messages = append(h.Messages, map[string]interface{}{
		"type":    "info",
		"message": msg,
	})
}

// OnError handles errors
func (h *JSONHandler) OnError(err error) {
	h.Messages = append(h.Messages, map[string]interface{}{
		"type":  "error",
		"error": err.Error(),
	})
}

// OnComplete handles task completion
func (h *JSONHandler) OnComplete(success bool, message string) {
	h.Messages = append(h.Messages, map[string]interface{}{
		"type":    "complete",
		"success": success,
		"message": message,
	})
	h.flush()
}

// flush outputs all messages as JSON
func (h *JSONHandler) flush() {
	if h.Output == nil {
		return
	}
	
	// Output as JSON array
	encoder := json.NewEncoder(h.Output)
	encoder.SetIndent("", "  ")
	encoder.Encode(h.Messages)
}
