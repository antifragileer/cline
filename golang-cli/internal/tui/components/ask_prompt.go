// Package components provides TUI components for the Cline CLI.
package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AskType represents the type of ask prompt
type AskType string

const (
	// AskTypeTool is for tool use approval
	AskTypeTool AskType = "tool"
	// AskTypeCommand is for command execution approval
	AskTypeCommand AskType = "command"
	// AskTypeCommandOutput is for command output approval
	AskTypeCommandOutput AskType = "command_output"
	// AskTypeBrowser is for browser action approval
	AskTypeBrowser AskType = "browser_action_launch"
	// AskTypeCompletion is for task completion
	AskTypeCompletion AskType = "completion_result"
	// AskTypeResume is for task resumption
	AskTypeResume AskType = "resume_task"
	// AskTypeResumeCompleted is for completed task resumption
	AskTypeResumeCompleted AskType = "resume_completed_task"
	// AskTypeNewTask is for starting a new task
	AskTypeNewTask AskType = "new_task"
	// AskTypeFollowup is for follow-up questions
	AskTypeFollowup AskType = "followup"
	// AskTypePlanMode is for plan mode responses
	AskTypePlanMode AskType = "plan_mode_respond"
	// AskTypeAPIFailed is for API request failures
	AskTypeAPIFailed AskType = "api_req_failed"
	// AskTypeMistakeLimit is for mistake limit reached
	AskTypeMistakeLimit AskType = "mistake_limit_reached"
	// AskTypeMCP is for MCP server approval
	AskTypeMCP AskType = "use_mcp_server"
)

// AskResponse represents the user's response to an ask prompt
type AskResponse string

const (
	// AskResponseYes approves the request
	AskResponseYes AskResponse = "yesButtonClicked"
	// AskResponseNo denies the request
	AskResponseNo AskResponse = "noButtonClicked"
	// AskResponseAlways approves and remembers
	AskResponseAlways AskResponse = "alwaysButtonClicked"
	// AskResponseProceed proceeds while running
	AskResponseProceed AskResponse = "proceedButtonClicked"
)

// AskPrompt displays an interactive prompt for user approval/decisions
type AskPrompt struct {
	// Configuration
	askType AskType
	title   string
	message string
	details string

	// State
	width       int
	height      int
	selected    int // 0=Yes, 1=No, 2=Always
	showDetails bool
	done        bool
	response    AskResponse

	// Styling
	containerStyle   lipgloss.Style
	titleStyle       lipgloss.Style
	messageStyle     lipgloss.Style
	detailsStyle     lipgloss.Style
	buttonYesStyle   lipgloss.Style
	buttonNoStyle    lipgloss.Style
	buttonAlwaysStyle lipgloss.Style
	selectedStyle    lipgloss.Style
	helpStyle        lipgloss.Style
	warningStyle     lipgloss.Style
}

// NewAskPrompt creates a new ask prompt
func NewAskPrompt(askType AskType, title, message, details string) *AskPrompt {
	return &AskPrompt{
		askType:     askType,
		title:       title,
		message:     message,
		details:     details,
		selected:    0,
		showDetails: true,
		response:    "",

		containerStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1, 2).
			Width(70),

		titleStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")),

		messageStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0E0E0")),

		detailsStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A0A0A0")).
			Background(lipgloss.Color("#2a2a2a")).
			Padding(1, 1).
			Width(66),

		buttonYesStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#00FF00")).
			Padding(0, 2).
			Bold(true),

		buttonNoStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#FF4444")).
			Padding(0, 2).
			Bold(true),

		buttonAlwaysStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#FFB000")).
			Padding(0, 2).
			Bold(true),

		selectedStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#00D9FF")).
			Padding(0, 1),

		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#606060")).
			Italic(true),

		warningStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B")).
			Bold(true),
	}
}

// SetDimensions sets the prompt dimensions
func (ap *AskPrompt) SetDimensions(width, height int) {
	ap.width = width
	ap.height = height
}

// Init initializes the prompt
func (ap AskPrompt) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (ap AskPrompt) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		ap.SetDimensions(msg.Width, msg.Height)
		return ap, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			ap.done = true
			ap.response = AskResponseNo
			return ap, tea.Quit

		case "y":
			ap.done = true
			ap.response = AskResponseYes
			return ap, tea.Quit

		case "n":
			ap.done = true
			ap.response = AskResponseNo
			return ap, tea.Quit

		case "a":
			if ap.canShowAlways() {
				ap.done = true
				ap.response = AskResponseAlways
				return ap, tea.Quit
			}

		case "left", "h":
			if ap.selected > 0 {
				ap.selected--
			}
			return ap, nil

		case "right", "l":
			if ap.selected < ap.getMaxSelection() {
				ap.selected++
			}
			return ap, nil

		case "tab":
			maxSel := ap.getMaxSelection()
			if maxSel >= 0 {
				ap.selected = (ap.selected + 1) % (maxSel + 1)
			}
			return ap, nil

		case "shift+tab":
			maxSel := ap.getMaxSelection()
			if maxSel >= 0 {
				ap.selected--
				if ap.selected < 0 {
					ap.selected = maxSel
				}
			}
			return ap, nil

		case "enter", " ":
			ap.done = true
			switch ap.selected {
			case 0:
				ap.response = ap.getPrimaryResponse()
			case 1:
				ap.response = ap.getSecondaryResponse()
			case 2:
				ap.response = AskResponseAlways
			}
			return ap, tea.Quit

		case "d":
			// Toggle details
			ap.showDetails = !ap.showDetails
			return ap, nil
		}
	}

	return ap, nil
}

// View renders the ask prompt
func (ap AskPrompt) View() string {
	if ap.width == 0 || ap.height == 0 {
		return "Loading..."
	}

	var content strings.Builder

	// Title with icon
	icon := ap.getIcon()
	title := fmt.Sprintf("%s %s", icon, ap.title)
	content.WriteString(ap.titleStyle.Render(title))
	content.WriteString("\n\n")

	// Message
	content.WriteString(ap.messageStyle.Render(ap.wrapText(ap.message, 66)))
	content.WriteString("\n")

	// Warning for dangerous operations
	if ap.isDangerous() {
		content.WriteString("\n")
		warning := ap.warningStyle.Render("⚠️  Warning: This operation may modify files or system state.")
		content.WriteString(warning)
		content.WriteString("\n")
	}

	// Details (if shown)
	if ap.showDetails && ap.details != "" {
		content.WriteString("\n")
		detailsLabel := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Render("Details (press 'd' to toggle):")
		content.WriteString(detailsLabel)
		content.WriteString("\n")

		// Truncate details if too long
		details := ap.details
		maxDetailsLen := 500
		if len(details) > maxDetailsLen {
			details = details[:maxDetailsLen] + "\n... (truncated)"
		}
		content.WriteString(ap.detailsStyle.Render(ap.wrapText(details, 64)))
		content.WriteString("\n")
	}

	// Buttons
	content.WriteString("\n")
	content.WriteString(ap.renderButtons())
	content.WriteString("\n")

	// Help text
	help := ap.getHelpText()
	content.WriteString(ap.helpStyle.Render(help))

	// Apply container styling and center
	modalContent := content.String()

	centered := lipgloss.Place(
		ap.width,
		ap.height,
		lipgloss.Center,
		lipgloss.Center,
		ap.containerStyle.Render(modalContent),
	)

	return centered
}

// renderButtons renders the action buttons
func (ap AskPrompt) renderButtons() string {
	var buttons []string

	// Primary button (Yes/Proceed/Restore/etc)
	primaryText, secondaryText := ap.getButtonLabels()

	// Primary button
	if ap.selected == 0 {
		buttons = append(buttons, ap.selectedStyle.Render(ap.buttonYesStyle.Render(" "+primaryText+" ")))
	} else {
		buttons = append(buttons, ap.buttonYesStyle.Render(" "+primaryText+" "))
	}

	// Secondary button (No/Exit/Cancel)
	if ap.selected == 1 {
		buttons = append(buttons, ap.selectedStyle.Render(ap.buttonNoStyle.Render(" "+secondaryText+" ")))
	} else {
		buttons = append(buttons, ap.buttonNoStyle.Render(" "+secondaryText+" "))
	}

	// Always button (if applicable)
	if ap.canShowAlways() {
		if ap.selected == 2 {
			buttons = append(buttons, ap.selectedStyle.Render(ap.buttonAlwaysStyle.Render(" Always ")))
		} else {
			buttons = append(buttons, ap.buttonAlwaysStyle.Render(" Always "))
		}
	}

	return "  " + strings.Join(buttons, "    ")
}

// getButtonLabels returns the appropriate button labels based on ask type
func (ap AskPrompt) getButtonLabels() (primary, secondary string) {
	switch ap.askType {
	case AskTypeCommand:
		return "Run", "Cancel"
	case AskTypeTool:
		return "Approve", "Reject"
	case AskTypeBrowser:
		return "Allow", "Deny"
	case AskTypeCompletion:
		return "New Task", "Exit"
	case AskTypeResume, AskTypeResumeCompleted:
		return "Resume", "Exit"
	case AskTypeNewTask:
		return "Start", "Exit"
	case AskTypeAPIFailed:
		return "Retry", "New Task"
	case AskTypeMistakeLimit:
		return "Proceed", "New Task"
	case AskTypeMCP:
		return "Allow", "Deny"
	default:
		return "Yes", "No"
	}
}

// getPrimaryResponse returns the primary response based on ask type
func (ap AskPrompt) getPrimaryResponse() AskResponse {
	switch ap.askType {
	case AskTypeMistakeLimit:
		return AskResponseProceed
	default:
		return AskResponseYes
	}
}

// getSecondaryResponse returns the secondary response based on ask type
func (ap AskPrompt) getSecondaryResponse() AskResponse {
	return AskResponseNo
}

// canShowAlways returns true if the "Always" option should be shown
func (ap AskPrompt) canShowAlways() bool {
	// Don't show Always for certain types
	switch ap.askType {
	case AskTypeCompletion, AskTypeResume, AskTypeResumeCompleted,
		AskTypeNewTask, AskTypeAPIFailed, AskTypeMistakeLimit,
		AskTypeFollowup, AskTypePlanMode:
		return false
	default:
		return true
	}
}

// getMaxSelection returns the maximum selection index
func (ap AskPrompt) getMaxSelection() int {
	if ap.canShowAlways() {
		return 2
	}
	return 1
}

// getIcon returns the appropriate icon for the ask type
func (ap AskPrompt) getIcon() string {
	switch ap.askType {
	case AskTypeCommand:
		return "⚡"
	case AskTypeTool:
		return "🔧"
	case AskTypeBrowser:
		return "🌐"
	case AskTypeCompletion:
		return "✅"
	case AskTypeResume, AskTypeResumeCompleted:
		return "🔄"
	case AskTypeNewTask:
		return "🆕"
	case AskTypeAPIFailed:
		return "❌"
	case AskTypeMistakeLimit:
		return "⚠️"
	case AskTypeMCP:
		return "🔌"
	default:
		return "❓"
	}
}

// isDangerous returns true if this is a potentially dangerous operation
func (ap AskPrompt) isDangerous() bool {
	return ap.askType == AskTypeCommand || ap.askType == AskTypeTool
}

// getHelpText returns appropriate help text based on ask type
func (ap AskPrompt) getHelpText() string {
	parts := []string{"y: yes", "n: no"}
	if ap.canShowAlways() {
		parts = append(parts, "a: always")
	}
	parts = append(parts, "←→: navigate", "enter: select", "d: toggle details", "q: quit")
	return strings.Join(parts, " • ")
}

// wrapText wraps text to fit within the given width
func (ap AskPrompt) wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	var lines []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	currentLine := words[0]
	for _, word := range words[1:] {
		if len(currentLine)+1+len(word) <= width {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}
	lines = append(lines, currentLine)

	return strings.Join(lines, "\n")
}

// IsDone returns true if the prompt is complete
func (ap AskPrompt) IsDone() bool {
	return ap.done
}

// GetResponse returns the user's response
func (ap AskPrompt) GetResponse() AskResponse {
	return ap.response
}

// AskPromptResult contains the result of an ask prompt
type AskPromptResult struct {
	Response AskResponse
	AskType  AskType
}

// ShowAskPrompt displays an ask prompt and returns the result
func ShowAskPrompt(askType AskType, title, message, details string) (AskResponse, error) {
	prompt := NewAskPrompt(askType, title, message, details)

	p := tea.NewProgram(prompt, tea.WithAltScreen())

	m, err := p.Run()
	if err != nil {
		return AskResponseNo, err
	}

	askPrompt, ok := m.(AskPrompt)
	if !ok {
		return AskResponseNo, fmt.Errorf("unexpected model type")
	}

	return askPrompt.GetResponse(), nil
}