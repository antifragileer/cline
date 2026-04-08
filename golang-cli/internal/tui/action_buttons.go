// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// ButtonActionType represents the type of button action
type ButtonActionType string

const (
	// ButtonActionApprove sends yesButtonClicked
	ButtonActionApprove ButtonActionType = "approve"
	// ButtonActionReject sends noButtonClicked
	ButtonActionReject ButtonActionType = "reject"
	// ButtonActionProceed sends messageResponse or yesButtonClicked
	ButtonActionProceed ButtonActionType = "proceed"
	// ButtonActionNewTask starts a new task
	ButtonActionNewTask ButtonActionType = "new_task"
	// ButtonActionCancel cancels streaming
	ButtonActionCancel ButtonActionType = "cancel"
	// ButtonActionRetry retries the last action
	ButtonActionRetry ButtonActionType = "retry"
)

// ButtonConfig defines the configuration for action buttons
type ButtonConfig struct {
	SendingDisabled bool
	EnableButtons   bool
	PrimaryText     string
	SecondaryText   string
	PrimaryAction   ButtonActionType
	SecondaryAction ButtonActionType
}

// ActionButtons renders action buttons based on configuration
type ActionButtons struct {
	config        ButtonConfig
	mode          string // "act" or "plan"
	terminalWidth int
}

// NewActionButtons creates a new action buttons component
func NewActionButtons(config ButtonConfig, mode string, terminalWidth int) *ActionButtons {
	return &ActionButtons{
		config:        config,
		mode:          mode,
		terminalWidth: terminalWidth,
	}
}

// SetConfig updates the button configuration
func (ab *ActionButtons) SetConfig(config ButtonConfig) {
	ab.config = config
}

// SetMode updates the mode
func (ab *ActionButtons) SetMode(mode string) {
	ab.mode = mode
}

// SetTerminalWidth updates the terminal width
func (ab *ActionButtons) SetTerminalWidth(width int) {
	ab.terminalWidth = width
}

// ShouldShow returns true if buttons should be shown
func (ab *ActionButtons) ShouldShow() bool {
	if !ab.config.EnableButtons {
		return false
	}

	// Don't show cancel-only buttons (handled by thinking indicator)
	hiddenActions := map[ButtonActionType]bool{
		ButtonActionCancel: true,
	}

	hasPrimary := ab.config.PrimaryText != "" && !hiddenActions[ab.config.PrimaryAction]
	hasSecondary := ab.config.SecondaryText != "" && !hiddenActions[ab.config.SecondaryAction]

	return hasPrimary || hasSecondary
}

// Render renders the action buttons
func (ab *ActionButtons) Render() string {
	if !ab.ShouldShow() {
		return ""
	}

	hiddenActions := map[ButtonActionType]bool{
		ButtonActionCancel: true,
	}

	hasPrimary := ab.config.PrimaryText != "" && !hiddenActions[ab.config.PrimaryAction]
	hasSecondary := ab.config.SecondaryText != "" && !hiddenActions[ab.config.SecondaryAction]

	if !hasPrimary && !hasSecondary {
		return ""
	}

	// Calculate button widths
	buttonCount := 0
	if hasPrimary {
		buttonCount++
	}
	if hasSecondary {
		buttonCount++
	}

	gapWidth := 1
	if buttonCount <= 1 {
		gapWidth = 0
	}

	availableWidth := ab.terminalWidth - 2 - gapWidth // 1 space padding on each side
	buttonWidth := availableWidth / buttonCount

	// Get mode color
	modeColor := lipgloss.Color("#00D9FF") // Act mode blue
	if ab.mode == "plan" {
		modeColor = lipgloss.Color("#FFB000") // Plan mode yellow
	}

	var buttons []string

	if hasPrimary {
		buttons = append(buttons, ab.renderButton(ab.config.PrimaryText, "1", buttonWidth, modeColor))
	}

	if hasSecondary {
		shortcut := "1"
		if hasPrimary {
			shortcut = "2"
		}
		buttons = append(buttons, ab.renderButton(ab.config.SecondaryText, shortcut, buttonWidth, modeColor))
	}

	// Join buttons with gap
	gap := " "
	if gapWidth == 0 {
		gap = ""
	}

	return lipgloss.NewStyle().MarginLeft(1).Render(buttons[0] + gap + buttons[1])
}

// renderButton renders a single button
func (ab *ActionButtons) renderButton(text, shortcut string, width int, color lipgloss.Color) string {
	label := " " + text + " (" + shortcut + ") "
	padding := width - len(label)
	if padding < 0 {
		padding = 0
	}

	leftPad := padding / 2
	rightPad := padding - leftPad

	paddedLabel := ""
	for i := 0; i < leftPad; i++ {
		paddedLabel += " "
	}
	paddedLabel += label
	for i := 0; i < rightPad; i++ {
		paddedLabel += " "
	}

	buttonStyle := lipgloss.NewStyle().
		Background(color).
		Foreground(lipgloss.Color("#000000"))

	return buttonStyle.Render(paddedLabel)
}

// GetButtonConfig returns the appropriate button config for a message
func GetButtonConfig(msgType, msgSubType string, isStreaming, isPartial bool) ButtonConfig {
	// Error recovery states
	switch msgSubType {
	case "api_req_failed":
		return ButtonConfig{
			SendingDisabled: true,
			EnableButtons:   true,
			PrimaryText:     "Retry",
			SecondaryText:   "Start New Task",
			PrimaryAction:   ButtonActionRetry,
			SecondaryAction: ButtonActionNewTask,
		}
	case "mistake_limit_reached":
		return ButtonConfig{
			SendingDisabled: false,
			EnableButtons:   true,
			PrimaryText:     "Proceed Anyways",
			SecondaryText:   "Start New Task",
			PrimaryAction:   ButtonActionProceed,
			SecondaryAction: ButtonActionNewTask,
		}
	}

	// Handle streaming/partial states
	if isStreaming && !isPartial {
		return ButtonConfig{
			SendingDisabled: true,
			EnableButtons:   true,
			SecondaryText:   "Cancel",
			SecondaryAction: ButtonActionCancel,
		}
	}

	// Handle ask messages
	if msgType == "ask" {
		switch msgSubType {
		case "tool":
			return ButtonConfig{
				SendingDisabled: false,
				EnableButtons:   true,
				PrimaryText:     "Approve",
				SecondaryText:   "Reject",
				PrimaryAction:   ButtonActionApprove,
				SecondaryAction: ButtonActionReject,
			}
		case "command":
			return ButtonConfig{
				SendingDisabled: false,
				EnableButtons:   true,
				PrimaryText:     "Run Command",
				SecondaryText:   "Reject",
				PrimaryAction:   ButtonActionApprove,
				SecondaryAction: ButtonActionReject,
			}
		case "command_output":
			return ButtonConfig{
				SendingDisabled: false,
				EnableButtons:   true,
				PrimaryText:     "Proceed While Running",
				PrimaryAction:   ButtonActionProceed,
			}
		case "browser_action_launch", "use_mcp_server":
			return ButtonConfig{
				SendingDisabled: false,
				EnableButtons:   true,
				PrimaryText:     "Approve",
				SecondaryText:   "Reject",
				PrimaryAction:   ButtonActionApprove,
				SecondaryAction: ButtonActionReject,
			}
		case "completion_result":
			return ButtonConfig{
				SendingDisabled: false,
				EnableButtons:   true,
				PrimaryText:     "Start New Task",
				SecondaryText:   "Exit",
				PrimaryAction:   ButtonActionNewTask,
				SecondaryAction: ButtonActionReject,
			}
		case "resume_task":
			return ButtonConfig{
				SendingDisabled: false,
				EnableButtons:   true,
				PrimaryText:     "Resume Task",
				SecondaryText:   "Exit",
				PrimaryAction:   ButtonActionProceed,
				SecondaryAction: ButtonActionReject,
			}
		case "resume_completed_task":
			return ButtonConfig{
				SendingDisabled: false,
				EnableButtons:   true,
				PrimaryText:     "Start New Task",
				SecondaryText:   "Exit",
				PrimaryAction:   ButtonActionNewTask,
				SecondaryAction: ButtonActionReject,
			}
		case "new_task":
			return ButtonConfig{
				SendingDisabled: false,
				EnableButtons:   true,
				PrimaryText:     "Start New Task with Context",
				SecondaryText:   "Exit",
				PrimaryAction:   ButtonActionNewTask,
				SecondaryAction: ButtonActionReject,
			}
		case "followup", "plan_mode_respond":
			return ButtonConfig{
				SendingDisabled: false,
				EnableButtons:   false,
			}
		}
	}

	// Handle say messages
	if msgType == "say" && msgSubType == "api_req_started" {
		return ButtonConfig{
			SendingDisabled: true,
			EnableButtons:   true,
			SecondaryText:   "Cancel",
			SecondaryAction: ButtonActionCancel,
		}
	}

	// Default: no buttons
	return ButtonConfig{
		SendingDisabled: false,
		EnableButtons:   false,
	}
}

// HandleButtonAction returns the response type for a button action
func HandleButtonAction(action ButtonActionType, pendingAskType string) string {
	switch action {
	case ButtonActionApprove, ButtonActionProceed:
		return "yesButtonClicked"
	case ButtonActionReject:
		// Check for states that should trigger exit
		if pendingAskType == "resume_task" ||
			pendingAskType == "resume_completed_task" ||
			pendingAskType == "completion_result" ||
			pendingAskType == "new_task" {
			return "noButtonClicked" // Will trigger exit
		}
		return "noButtonClicked"
	case ButtonActionNewTask:
		return "yesButtonClicked"
	case ButtonActionRetry:
		return "yesButtonClicked"
	default:
		return "noButtonClicked"
	}
}
