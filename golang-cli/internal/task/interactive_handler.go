// Package task provides task execution functionality for the Cline CLI.
package task

import (
	"fmt"
	"strings"

	"github.com/cline/cline/golang-cli/internal/tui"
)

// InteractiveHandler handles messages in interactive TUI mode with approval prompts.
type InteractiveHandler struct {
	// Approval settings
	AutoApprove bool
	Verbose     bool

	// State for tracking current operation
	currentAsk     string
	currentAskType string
}

// NewInteractiveHandler creates a new interactive handler
func NewInteractiveHandler(autoApprove, verbose bool) *InteractiveHandler {
	return &InteractiveHandler{
		AutoApprove: autoApprove,
		Verbose:     verbose,
	}
}

// OnSay handles a SAY message from the assistant
func (h *InteractiveHandler) OnSay(sayType string, text string, partial bool) {
	// In interactive mode, we display messages through the chat TUI
	// The chat model handles the actual display
}

// OnAsk handles an ASK message that requires user response
func (h *InteractiveHandler) OnAsk(askType string, text string) (string, error) {
	h.currentAsk = text
	h.currentAskType = askType

	// Auto-approve if enabled
	if h.AutoApprove {
		return "yesButtonClicked", nil
	}

	// Show appropriate approval prompt based on ask type
	switch askType {
	case "command":
		return h.handleCommandApproval(text)
	case "tool":
		return h.handleToolApproval(text)
	case "browser_action_launch":
		return h.handleBrowserApproval(text)
	case "completion_result":
		// Completion results don't need approval, just acknowledgment
		return "yesButtonClicked", nil
	case "api_req_failed":
		// API failures - ask if user wants to retry
		return h.handleRetryPrompt(text)
	case "followup":
		// Follow-up questions - need text response
		return h.handleFollowup(text)
	case "use_mcp_server":
		// MCP server usage approval
		return h.handleMCPApproval(text)
	default:
		// Default approval prompt for unknown types
		return h.handleGenericApproval(askType, text)
	}
}

// handleMCPApproval shows an MCP server usage approval prompt
func (h *InteractiveHandler) handleMCPApproval(text string) (string, error) {
	response, err := tui.ShowApprovalPrompt(
		tui.ApprovalTypeTool,
		"MCP Server Request",
		text,
		"",
	)
	if err != nil {
		return "noButtonClicked", err
	}

	return h.mapResponse(response), nil
}

// handleCommandApproval shows a command approval prompt
func (h *InteractiveHandler) handleCommandApproval(text string) (string, error) {
	// Check if command is dangerous
	isDangerous := isDangerousCommand(text)

	// Show approval prompt
	response, err := tui.CommandApprovalPrompt(text, isDangerous)
	if err != nil {
		return "noButtonClicked", err
	}

	return h.mapResponse(response), nil
}

// handleToolApproval shows a tool approval prompt
func (h *InteractiveHandler) handleToolApproval(text string) (string, error) {
	// Parse tool name and params from text
	toolName, params := parseToolInfo(text)

	// Show approval prompt
	response, err := tui.ToolApprovalPrompt(toolName, params)
	if err != nil {
		return "noButtonClicked", err
	}

	return h.mapResponse(response), nil
}

// handleBrowserApproval shows a browser action approval prompt
func (h *InteractiveHandler) handleBrowserApproval(text string) (string, error) {
	// Extract URL from text
	url := extractURL(text)

	// Show approval prompt
	response, err := tui.BrowserApprovalPrompt("launch", url)
	if err != nil {
		return "noButtonClicked", err
	}

	return h.mapResponse(response), nil
}

// handleRetryPrompt shows a retry prompt for failed API requests
func (h *InteractiveHandler) handleRetryPrompt(text string) (string, error) {
	// For now, auto-retry in interactive mode
	// Could be enhanced with a proper retry prompt
	return "yesButtonClicked", nil
}

// handleFollowup handles follow-up questions that need text input
func (h *InteractiveHandler) handleFollowup(text string) (string, error) {
	// This would need to integrate with the chat TUI for text input
	// For now, return a generic response
	return "messageResponse", nil
}

// handleGenericApproval shows a generic approval prompt
func (h *InteractiveHandler) handleGenericApproval(askType, text string) (string, error) {
	title := fmt.Sprintf("%s Approval", strings.Title(askType))
	response, err := tui.ShowApprovalPrompt(
		tui.ApprovalTypeTool,
		title,
		text,
		"",
	)
	if err != nil {
		return "noButtonClicked", err
	}

	return h.mapResponse(response), nil
}

// mapResponse maps TUI approval response to gRPC response type
func (h *InteractiveHandler) mapResponse(response tui.ApprovalResponse) string {
	switch response {
	case tui.ApprovalYes, tui.ApprovalAlways:
		return "yesButtonClicked"
	case tui.ApprovalNo:
		return "noButtonClicked"
	default:
		return "noButtonClicked"
	}
}

// OnInfo handles informational messages
func (h *InteractiveHandler) OnInfo(text string) {
	if h.Verbose {
		// Could show in TUI status area
	}
}

// OnError handles error messages
func (h *InteractiveHandler) OnError(err error) {
	// Errors are displayed in the chat TUI
}

// OnStatus handles status updates
func (h *InteractiveHandler) OnStatus(status string) {
	if h.Verbose {
		// Could show in TUI status area
	}
}

// OnProgress handles progress updates
func (h *InteractiveHandler) OnProgress(current, total int) {
	// Progress could be shown in TUI
}

// isDangerousCommand checks if a command is potentially dangerous
func isDangerousCommand(cmd string) bool {
	dangerousPatterns := []string{
		"rm -rf",
		"rm -r /",
		"mkfs",
		"dd if=",
		"> /dev/",
		"sudo",
		"chmod 777",
		"curl.*|.*sh",
		"wget.*|.*sh",
	}

	cmdLower := strings.ToLower(cmd)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(cmdLower, pattern) {
			return true
		}
	}
	return false
}

// parseToolInfo parses tool name and parameters from text
func parseToolInfo(text string) (string, map[string]interface{}) {
	params := make(map[string]interface{})

	// Try to extract tool name from first line
	lines := strings.Split(text, "\n")
	toolName := "unknown"

	if len(lines) > 0 {
		firstLine := strings.TrimSpace(lines[0])
		// Look for patterns like "Tool: read_file" or "read_file"
		if strings.HasPrefix(firstLine, "Tool:") {
			parts := strings.SplitN(firstLine, ":", 2)
			if len(parts) == 2 {
				toolName = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(firstLine, " ") {
			toolName = strings.Split(firstLine, " ")[0]
		} else {
			toolName = firstLine
		}
	}

	// Extract key-value pairs from text
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, ":") && !strings.HasPrefix(line, "Tool:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				params[key] = value
			}
		}
	}

	return toolName, params
}

// extractURL extracts a URL from text
func extractURL(text string) string {
	// Simple URL extraction - look for http:// or https://
	words := strings.Fields(text)
	for _, word := range words {
		if strings.HasPrefix(word, "http://") || strings.HasPrefix(word, "https://") {
			// Remove any trailing punctuation
			word = strings.TrimRight(word, ".,;:!?")
			return word
		}
	}
	return ""
}
