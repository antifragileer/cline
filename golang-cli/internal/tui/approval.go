// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ApprovalType represents the type of approval request
type ApprovalType string

const (
	// ApprovalTypeCommand is for command execution approval
	ApprovalTypeCommand ApprovalType = "command"
	// ApprovalTypeTool is for tool use approval
	ApprovalTypeTool ApprovalType = "tool"
	// ApprovalTypeEdit is for file edit approval
	ApprovalTypeEdit ApprovalType = "edit"
	// ApprovalTypeBrowser is for browser action approval
	ApprovalTypeBrowser ApprovalType = "browser"
)

// ApprovalResponse represents the user's response to an approval request
type ApprovalResponse string

const (
	// ApprovalYes approves the request
	ApprovalYes ApprovalResponse = "yes"
	// ApprovalNo denies the request
	ApprovalNo ApprovalResponse = "no"
	// ApprovalAlways approves and remembers for future
	ApprovalAlways ApprovalResponse = "always"
)

// ToolRequest represents a request for tool approval
type ToolRequest struct {
	// ToolType is the type of tool (command, file_edit, browser, etc.)
	ToolType string
	// ToolName is the name of the tool
	ToolName string
	// Description is a human-readable description of what the tool will do
	Description string
	// Command is the actual command to execute (for command tools)
	Command string
	// FilePath is the target file path (for file operations)
	FilePath string
	// Params contains additional tool parameters
	Params map[string]interface{}
	// IsDangerous indicates if this operation might be harmful
	IsDangerous bool
	// RequiresApproval indicates if this tool requires user approval
	RequiresApproval bool
}

// SimpleApprovalHistory tracks approval decisions (simple version for basic approval tracking)
type SimpleApprovalHistory struct {
	mu        sync.RWMutex
	decisions []SimpleApprovalDecision
}

// SimpleApprovalDecision represents a single approval decision
type SimpleApprovalDecision struct {
	Timestamp    time.Time
	ToolType     string
	ToolName     string
	Approved     bool
	AutoApproved bool
}

// NewSimpleApprovalHistory creates a new simple approval history tracker
func NewSimpleApprovalHistory() *SimpleApprovalHistory {
	return &SimpleApprovalHistory{
		decisions: make([]SimpleApprovalDecision, 0),
	}
}

// RecordDecision records an approval decision
func (ah *SimpleApprovalHistory) RecordDecision(toolType, toolName string, approved, autoApproved bool) {
	ah.mu.Lock()
	defer ah.mu.Unlock()

	ah.decisions = append(ah.decisions, SimpleApprovalDecision{
		Timestamp:    time.Now(),
		ToolType:     toolType,
		ToolName:     toolName,
		Approved:     approved,
		AutoApproved: autoApproved,
	})
}

// GetRecentDecisions returns the most recent approval decisions
func (ah *SimpleApprovalHistory) GetRecentDecisions(limit int) []SimpleApprovalDecision {
	ah.mu.RLock()
	defer ah.mu.RUnlock()

	if limit <= 0 || limit > len(ah.decisions) {
		return ah.decisions
	}
	return ah.decisions[len(ah.decisions)-limit:]
}

// GetDecisionCount returns the total number of decisions
func (ah *SimpleApprovalHistory) GetDecisionCount() int {
	ah.mu.RLock()
	defer ah.mu.RUnlock()
	return len(ah.decisions)
}

// Approval handles approval workflows with support for keyboard shortcuts and auto-approve
type Approval struct {
	mu sync.RWMutex
	
	// Current request
	currentRequest *ToolRequest
	responseChan   chan<- string
	
	// State
	isPending      bool
	isAutoApprove  bool
	decisionMade   bool
	approved       bool
	
	// History
	history *SimpleApprovalHistory
	
	// UI state
	showDetails    bool
	keyboardShortcuts map[string]string
}

// NewApproval creates a new approval handler
func NewApproval() *Approval {
	return &Approval{
		history:           NewSimpleApprovalHistory(),
		showDetails:       true,
		keyboardShortcuts: map[string]string{
			"y": "approve",
			"Y": "approve",
			"n": "reject",
			"N": "reject",
			"\r": "approve", // Enter key
			"\n": "approve",
		},
	}
}

// ShowPrompt displays an approval prompt for a tool request
func (a *Approval) ShowPrompt(tool *ToolRequest) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	a.currentRequest = tool
	a.isPending = true
	a.decisionMade = false
	a.approved = false
	
	return nil
}

// HandleInput processes keyboard input for approval
// Returns (approved, done, error)
func (a *Approval) HandleInput(key string) (bool, bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	if !a.isPending {
		return false, true, nil
	}
	
	action, exists := a.keyboardShortcuts[key]
	if !exists {
		// Unknown key, still pending
		return false, false, nil
	}
	
	a.decisionMade = true
	a.isPending = false
	
	switch action {
	case "approve":
		a.approved = true
		if a.responseChan != nil {
			a.responseChan <- "yesButtonClicked"
		}
		a.history.RecordDecision(a.currentRequest.ToolType, a.currentRequest.ToolName, true, false)
		return true, true, nil
		
	case "reject":
		a.approved = false
		if a.responseChan != nil {
			a.responseChan <- "noButtonClicked"
		}
		a.history.RecordDecision(a.currentRequest.ToolType, a.currentRequest.ToolName, false, false)
		return false, true, nil
		
	default:
		return false, false, fmt.Errorf("unknown action: %s", action)
	}
}

// IsAutoApproveEnabled returns true if auto-approve (YOLO mode) is enabled
func (a *Approval) IsAutoApproveEnabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.isAutoApprove
}

// SetAutoApprove enables or disables auto-approve mode
func (a *Approval) SetAutoApprove(enabled bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.isAutoApprove = enabled
}

// AutoApprove approves the current request automatically (for YOLO mode)
func (a *Approval) AutoApprove() (bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	if !a.isPending || a.currentRequest == nil {
		return false, fmt.Errorf("no pending approval request")
	}
	
	// Don't auto-approve dangerous operations unless explicitly configured
	if a.currentRequest.IsDangerous && !a.isAutoApprove {
		return false, fmt.Errorf("dangerous operation requires manual approval")
	}
	
	a.decisionMade = true
	a.approved = true
	a.isPending = false
	
	if a.responseChan != nil {
		a.responseChan <- "yesButtonClicked"
	}
	
	a.history.RecordDecision(a.currentRequest.ToolType, a.currentRequest.ToolName, true, true)
	
	return true, nil
}

// IsPending returns true if there's a pending approval request
func (a *Approval) IsPending() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.isPending
}

// GetCurrentRequest returns the current pending request
func (a *Approval) GetCurrentRequest() *ToolRequest {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.currentRequest
}

// GetHistory returns the approval history
func (a *Approval) GetHistory() *SimpleApprovalHistory {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.history
}

// Reset clears the current approval state
func (a *Approval) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	a.currentRequest = nil
	a.isPending = false
	a.decisionMade = false
	a.approved = false
}

// GetToolDetails returns formatted details about the current tool request
func (a *Approval) GetToolDetails() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	if a.currentRequest == nil {
		return ""
	}
	
	tool := a.currentRequest
	var details strings.Builder
	
	details.WriteString(fmt.Sprintf("Tool: %s\n", tool.ToolName))
	details.WriteString(fmt.Sprintf("Type: %s\n", tool.ToolType))
	
	if tool.Description != "" {
		details.WriteString(fmt.Sprintf("Description: %s\n", tool.Description))
	}
	
	if tool.Command != "" {
		details.WriteString(fmt.Sprintf("Command: %s\n", tool.Command))
	}
	
	if tool.FilePath != "" {
		details.WriteString(fmt.Sprintf("File: %s\n", tool.FilePath))
	}
	
	if tool.IsDangerous {
		details.WriteString("\n⚠️  Warning: This operation may modify files or system state\n")
	}
	
	return details.String()
}

// SetResponseChannel sets the channel for sending approval responses
func (a *Approval) SetResponseChannel(ch chan<- string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.responseChan = ch
}

// ApprovalModel is the Bubble Tea model for approval prompts
type ApprovalModel struct {
	// Request details
	requestType ApprovalType
	title       string
	message     string
	details     string // command, tool params, or diff preview

	// UI state
	width       int
	height      int
	selected    int // 0=Yes, 1=No, 2=Always
	options     []string
	showDetails bool

	// Styling
	modalStyle     lipgloss.Style
	titleStyle     lipgloss.Style
	messageStyle   lipgloss.Style
	detailsStyle   lipgloss.Style
	buttonStyle    lipgloss.Style
	selectedStyle  lipgloss.Style
	highlightStyle lipgloss.Style

	// Response channel
	responseChan chan ApprovalResponse

	// State
	done   bool
	result ApprovalResponse
	err    error
}

// ApprovalResult is sent when approval is complete
type ApprovalResult struct {
	Response ApprovalResponse
	Error    error
}

// NewApprovalModel creates a new approval model
func NewApprovalModel(reqType ApprovalType, title, message, details string) ApprovalModel {
	options := []string{"Yes (y)", "No (n)", "Always (a)"}

	return ApprovalModel{
		requestType:  reqType,
		title:        title,
		message:      message,
		details:      details,
		selected:     0,
		options:      options,
		showDetails:  true,
		responseChan: make(chan ApprovalResponse, 1),

		modalStyle: lipgloss.NewStyle().
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
			Padding(1, 1),

		buttonStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")),

		selectedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00D9FF")).
			Bold(true).
			Background(lipgloss.Color("#1a1a1a")).
			Padding(0, 1),

		highlightStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B")).
			Bold(true),
	}
}

// SetDimensions sets the terminal dimensions
func (m *ApprovalModel) SetDimensions(width, height int) {
	m.width = width
	m.height = height
}

// Init initializes the model
func (m ApprovalModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m ApprovalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetDimensions(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.done = true
			m.result = ApprovalNo
			return m, tea.Quit

		case "y":
			m.done = true
			m.result = ApprovalYes
			return m, tea.Quit

		case "n":
			m.done = true
			m.result = ApprovalNo
			return m, tea.Quit

		case "a":
			m.done = true
			m.result = ApprovalAlways
			return m, tea.Quit

		case "left", "h":
			if m.selected > 0 {
				m.selected--
			}
			return m, nil

		case "right", "l":
			if m.selected < len(m.options)-1 {
				m.selected++
			}
			return m, nil

		case "tab":
			m.selected = (m.selected + 1) % len(m.options)
			return m, nil

		case "shift+tab":
			m.selected--
			if m.selected < 0 {
				m.selected = len(m.options) - 1
			}
			return m, nil

		case "enter", " ":
			m.done = true
			switch m.selected {
			case 0:
				m.result = ApprovalYes
			case 1:
				m.result = ApprovalNo
			case 2:
				m.result = ApprovalAlways
			}
			return m, tea.Quit

		case "d":
			// Toggle details
			m.showDetails = !m.showDetails
			return m, nil
		}
	}

	return m, nil
}

// View renders the approval modal
func (m ApprovalModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	// Build modal content
	var content strings.Builder

	// Title with icon
	icon := m.getIcon()
	title := fmt.Sprintf("%s %s", icon, m.title)
	content.WriteString(m.titleStyle.Render(title))
	content.WriteString("\n\n")

	// Message
	content.WriteString(m.messageStyle.Render(m.wrapText(m.message, 66)))
	content.WriteString("\n")

	// Details (if shown)
	if m.showDetails && m.details != "" {
		content.WriteString("\n")
		detailsLabel := m.highlightStyle.Render("Details (d to toggle):")
		content.WriteString(detailsLabel)
		content.WriteString("\n")

		// Truncate details if too long
		details := m.details
		maxDetailsLen := 300
		if len(details) > maxDetailsLen {
			details = details[:maxDetailsLen] + "..."
		}
		content.WriteString(m.detailsStyle.Render(m.wrapText(details, 64)))
		content.WriteString("\n")
	}

	// Buttons
	content.WriteString("\n")
	content.WriteString(m.renderButtons())
	content.WriteString("\n")

	// Help text
	helpText := "y: yes, n: no, a: always, d: toggle details, ↑↓←→: navigate, enter: select, q: quit"
	content.WriteString(lipgloss.NewStyle().
		Foreground(lipgloss.Color("#606060")).
		Italic(true).
		Render(helpText))

	// Apply modal styling
	modalContent := content.String()

	// Center the modal
	centered := lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		m.modalStyle.Render(modalContent),
	)

	return centered
}

// renderButtons renders the approval buttons
func (m ApprovalModel) renderButtons() string {
	var buttons []string

	for i, option := range m.options {
		if i == m.selected {
			buttons = append(buttons, m.selectedStyle.Render(" "+option+" "))
		} else {
			buttons = append(buttons, m.buttonStyle.Render(" "+option+" "))
		}
	}

	return "  " + strings.Join(buttons, "  ")
}

// getIcon returns the appropriate icon for the approval type
func (m ApprovalModel) getIcon() string {
	switch m.requestType {
	case ApprovalTypeCommand:
		return "⚡"
	case ApprovalTypeTool:
		return "🔧"
	case ApprovalTypeEdit:
		return "✏️"
	case ApprovalTypeBrowser:
		return "🌐"
	default:
		return "❓"
	}
}

// wrapText wraps text to fit within the given width
func (m ApprovalModel) wrapText(text string, width int) string {
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

// IsDone returns true if the approval is complete
func (m ApprovalModel) IsDone() bool {
	return m.done
}

// GetResult returns the approval result
func (m ApprovalModel) GetResult() ApprovalResponse {
	return m.result
}

// GetError returns any error
func (m ApprovalModel) GetError() error {
	return m.err
}

// ShowApprovalPrompt shows an approval prompt and returns the user's response
func ShowApprovalPrompt(reqType ApprovalType, title, message, details string) (ApprovalResponse, error) {
	model := NewApprovalModel(reqType, title, message, details)

	p := tea.NewProgram(model, tea.WithAltScreen())

	m, err := p.Run()
	if err != nil {
		return ApprovalNo, err
	}

	approvalModel, ok := m.(ApprovalModel)
	if !ok {
		return ApprovalNo, fmt.Errorf("unexpected model type")
	}

	return approvalModel.GetResult(), nil
}

// CommandApprovalPrompt prompts for command execution approval
func CommandApprovalPrompt(command string, isDangerous bool) (ApprovalResponse, error) {
	title := "Command Approval"
	message := "Cline wants to execute a command:"

	details := fmt.Sprintf("Command: %s", command)
	if isDangerous {
		message = "⚠️ Cline wants to execute a potentially dangerous command:"
		details += "\n\nWarning: This command may modify files or system state."
	}

	return ShowApprovalPrompt(ApprovalTypeCommand, title, message, details)
}

// ToolApprovalPrompt prompts for tool use approval
func ToolApprovalPrompt(toolName string, params map[string]interface{}) (ApprovalResponse, error) {
	title := "Tool Approval"
	message := fmt.Sprintf("Cline wants to use the %s tool:", toolName)

	// Format params
	var paramStrs []string
	for key, value := range params {
		paramStrs = append(paramStrs, fmt.Sprintf("  %s: %v", key, value))
	}
	details := strings.Join(paramStrs, "\n")

	return ShowApprovalPrompt(ApprovalTypeTool, title, message, details)
}

// EditApprovalPrompt prompts for file edit approval
func EditApprovalPrompt(filename string, diff string) (ApprovalResponse, error) {
	title := "Edit Approval"
	message := fmt.Sprintf("Cline wants to edit %s:", filename)

	return ShowApprovalPrompt(ApprovalTypeEdit, title, message, diff)
}

// BrowserApprovalPrompt prompts for browser action approval
func BrowserApprovalPrompt(action, url string) (ApprovalResponse, error) {
	title := "Browser Action Approval"
	message := fmt.Sprintf("Cline wants to %s in the browser:", action)
	details := fmt.Sprintf("URL: %s", url)

	return ShowApprovalPrompt(ApprovalTypeBrowser, title, message, details)
}
