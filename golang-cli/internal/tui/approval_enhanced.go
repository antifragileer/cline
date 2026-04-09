// Package tui provides terminal UI components for the Cline CLI.
// This file implements enhanced approval workflows with full keyboard shortcuts.
package tui

import (
	"fmt"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ApprovalOption represents the user's approval choice
type ApprovalOption string

const (
	// ApprovalOptionYes approves once
	ApprovalOptionYes ApprovalOption = "yes"
	// ApprovalOptionNo rejects once
	ApprovalOptionNo ApprovalOption = "no"
	// ApprovalOptionAlways approves and remembers for this tool type
	ApprovalOptionAlways ApprovalOption = "always"
	// ApprovalOptionNever rejects and remembers for this tool type
	ApprovalOptionNever ApprovalOption = "never"
)

// ApprovalState tracks approval preferences per session
type ApprovalState struct {
	mu sync.RWMutex

	// approvedTools tracks tools the user chose to always approve
	approvedTools map[string]bool

	// rejectedTools tracks tools the user chose to always reject
	rejectedTools map[string]bool

	// toolHistory tracks last choices for suggestions
	toolHistory map[string]ApprovalOption
}

// NewApprovalState creates a new approval state
func NewApprovalState() *ApprovalState {
	return &ApprovalState{
		approvedTools: make(map[string]bool),
		rejectedTools: make(map[string]bool),
		toolHistory:   make(map[string]ApprovalOption),
	}
}

// IsToolApproved checks if a tool is permanently approved
func (s *ApprovalState) IsToolApproved(toolName string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.approvedTools[toolName]
}

// IsToolRejected checks if a tool is permanently rejected
func (s *ApprovalState) IsToolRejected(toolName string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.rejectedTools[toolName]
}

// ApproveTool permanently approves a tool
func (s *ApprovalState) ApproveTool(toolName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.approvedTools[toolName] = true
	delete(s.rejectedTools, toolName)
	s.toolHistory[toolName] = ApprovalOptionAlways
}

// RejectTool permanently rejects a tool
func (s *ApprovalState) RejectTool(toolName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rejectedTools[toolName] = true
	delete(s.approvedTools, toolName)
	s.toolHistory[toolName] = ApprovalOptionNever
}

// RecordChoice records a one-time choice
func (s *ApprovalState) RecordChoice(toolName string, option ApprovalOption) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.toolHistory[toolName] = option
}

// GetLastChoice gets the last choice for a tool
func (s *ApprovalState) GetLastChoice(toolName string) (ApprovalOption, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	choice, ok := s.toolHistory[toolName]
	return choice, ok
}

// Clear clears all approval state
func (s *ApprovalState) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.approvedTools = make(map[string]bool)
	s.rejectedTools = make(map[string]bool)
	s.toolHistory = make(map[string]ApprovalOption)
}

// EnhancedApprovalModel provides a rich approval prompt with all options
type EnhancedApprovalModel struct {
	// Request details
	toolName    string
	toolType    string
	description string
	details     string

	// UI state
	width       int
	height      int
	selected    int // 0=Yes, 1=No, 2=Always, 3=Never
	options     []approvalOption
	showDetails bool

	// Styling
	styles approvalStyles

	// Result
	done   bool
	result ApprovalOption
	err    error

	// Mode
	plainTextMode bool
}

type approvalOption struct {
	key         string
	label       string
	description string
	shortcut    string
	value       ApprovalOption
}

type approvalStyles struct {
	modal         lipgloss.Style
	title         lipgloss.Style
	message       lipgloss.Style
	details       lipgloss.Style
	detailsHidden lipgloss.Style
	button        lipgloss.Style
	buttonSelected lipgloss.Style
	buttonAlways   lipgloss.Style
	buttonNever    lipgloss.Style
	help          lipgloss.Style
	warning       lipgloss.Style
}

// NewEnhancedApprovalModel creates a new enhanced approval model
func NewEnhancedApprovalModel(toolName, toolType, description, details string, plainTextMode bool) *EnhancedApprovalModel {
	options := []approvalOption{
		{
			key:         "y",
			label:       "Yes",
			description: "Approve this once",
			shortcut:    "(y)",
			value:       ApprovalOptionYes,
		},
		{
			key:         "n",
			label:       "No",
			description: "Reject this once",
			shortcut:    "(n)",
			value:       ApprovalOptionNo,
		},
		{
			key:         "a",
			label:       "Always",
			description: "Always approve this tool",
			shortcut:    "(a)",
			value:       ApprovalOptionAlways,
		},
		{
			key:         "s",
			label:       "Never",
			description: "Never approve this tool",
			shortcut:    "(s)",
			value:       ApprovalOptionNever,
		},
	}

	// Base colors
	primaryColor := lipgloss.Color("#7D56F4")
	successColor := lipgloss.Color("#00D26A")
	dangerColor := lipgloss.Color("#FF6B6B")
	warningColor := lipgloss.Color("#FFB800")
	textColor := lipgloss.Color("#E0E0E0")
	mutedColor := lipgloss.Color("#808080")
	bgColor := lipgloss.Color("#1a1a1a")

	return &EnhancedApprovalModel{
		toolName:      toolName,
		toolType:      toolType,
		description:   description,
		details:       details,
		selected:      0,
		options:       options,
		showDetails:   true,
		plainTextMode: plainTextMode,

		styles: approvalStyles{
			modal: lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(primaryColor).
				Padding(1, 2).
				Width(80),

			title: lipgloss.NewStyle().
				Bold(true).
				Foreground(primaryColor).
				MarginBottom(1),

			message: lipgloss.NewStyle().
				Foreground(textColor).
				MarginBottom(1),

			details: lipgloss.NewStyle().
				Foreground(textColor).
				Background(bgColor).
				Padding(1, 1).
				MarginTop(1).
				MarginBottom(1).
				Width(76),

			detailsHidden: lipgloss.NewStyle().
				Foreground(mutedColor).
				Italic(true),

			button: lipgloss.NewStyle().
				Foreground(textColor).
				Padding(0, 1),

			buttonSelected: lipgloss.NewStyle().
				Foreground(successColor).
				Bold(true).
				Background(bgColor).
				Padding(0, 1),

			buttonAlways: lipgloss.NewStyle().
				Foreground(warningColor).
				Bold(true).
				Background(bgColor).
				Padding(0, 1),

			buttonNever: lipgloss.NewStyle().
				Foreground(dangerColor).
				Bold(true).
				Background(bgColor).
				Padding(0, 1),

			help: lipgloss.NewStyle().
				Foreground(mutedColor).
				Italic(true).
				MarginTop(1),

			warning: lipgloss.NewStyle().
				Foreground(warningColor).
				Bold(true),
		},
	}
}

// SetDimensions sets the terminal dimensions
func (m *EnhancedApprovalModel) SetDimensions(width, height int) {
	m.width = width
	m.height = height
}

// Init initializes the model
func (m *EnhancedApprovalModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m *EnhancedApprovalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetDimensions(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.done = true
			m.result = ApprovalOptionNo
			return m, tea.Quit

		case "y":
			m.done = true
			m.result = ApprovalOptionYes
			return m, tea.Quit

		case "n":
			m.done = true
			m.result = ApprovalOptionNo
			return m, tea.Quit

		case "a":
			m.done = true
			m.result = ApprovalOptionAlways
			return m, tea.Quit

		case "s":
			m.done = true
			m.result = ApprovalOptionNever
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
			m.result = m.options[m.selected].value
			return m, tea.Quit

		case "d":
			m.showDetails = !m.showDetails
			return m, nil

		case "?":
			// Show help (could toggle a help view)
			return m, nil
		}
	}

	return m, nil
}

// View renders the approval modal
func (m *EnhancedApprovalModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	// Build modal content
	var content strings.Builder

	// Title with tool icon
	icon := m.getToolIcon()
	title := fmt.Sprintf("%s Tool Approval: %s", icon, m.toolName)
	content.WriteString(m.styles.title.Render(title))
	content.WriteString("\n\n")

	// Description
	content.WriteString(m.styles.message.Render(m.description))
	content.WriteString("\n")

	// Details section
	if m.showDetails && m.details != "" {
		content.WriteString("\n")
		detailsLabel := m.styles.warning.Render("⚠ Details (press 'd' to hide):")
		content.WriteString(detailsLabel)
		content.WriteString("\n")

		// Truncate and format details
		details := m.formatDetails(m.details, 74)
		content.WriteString(m.styles.details.Render(details))
		content.WriteString("\n")
	} else if m.details != "" {
		content.WriteString("\n")
		content.WriteString(m.styles.detailsHidden.Render("Details hidden (press 'd' to show)"))
		content.WriteString("\n")
	}

	// Buttons
	content.WriteString("\n")
	content.WriteString(m.renderButtons())
	content.WriteString("\n")

	// Help text
	helpText := "y: yes  n: no  a: always  s: never  ←→: navigate  enter: select  d: toggle details  q: quit"
	content.WriteString(m.styles.help.Render(helpText))

	// Apply modal styling
	modalContent := content.String()

	// Center the modal
	centered := lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		m.styles.modal.Render(modalContent),
	)

	return centered
}

// renderButtons renders the approval buttons
func (m *EnhancedApprovalModel) renderButtons() string {
	var buttons []string

	for i, option := range m.options {
		var button string
		label := fmt.Sprintf("%s %s", option.label, option.shortcut)

		if i == m.selected {
			// Use different highlight colors for always/never
			switch option.value {
			case ApprovalOptionAlways:
				button = m.styles.buttonAlways.Render(" " + label + " ")
			case ApprovalOptionNever:
				button = m.styles.buttonNever.Render(" " + label + " ")
			default:
				button = m.styles.buttonSelected.Render(" " + label + " ")
			}
		} else {
			button = m.styles.button.Render(" " + label + " ")
		}

		buttons = append(buttons, button)
	}

	return "  " + strings.Join(buttons, "    ")
}

// formatDetails formats and truncates details
func (m *EnhancedApprovalModel) formatDetails(details string, maxWidth int) string {
	// Limit total length
	maxLen := 500
	if len(details) > maxLen {
		details = details[:maxLen] + "\n... (truncated)"
	}

	// Wrap lines
	var lines []string
	currentLine := ""
	for _, word := range strings.Fields(details) {
		if len(currentLine)+len(word)+1 > maxWidth {
			lines = append(lines, currentLine)
			currentLine = word
		} else {
			if currentLine != "" {
				currentLine += " "
			}
			currentLine += word
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return strings.Join(lines, "\n")
}

// getToolIcon returns an icon based on tool type
func (m *EnhancedApprovalModel) getToolIcon() string {
	switch strings.ToLower(m.toolType) {
	case "command", "shell", "exec":
		return "⚡"
	case "file", "edit", "write", "read":
		return "📄"
	case "browser", "web":
		return "🌐"
	case "mcp", "server":
		return "🔌"
	case "search":
		return "🔍"
	case "delete":
		return "🗑️"
	default:
		return "🔧"
	}
}

// IsDone returns true if the approval is complete
func (m *EnhancedApprovalModel) IsDone() bool {
	return m.done
}

// GetResult returns the approval result
func (m *EnhancedApprovalModel) GetResult() ApprovalOption {
	return m.result
}

// GetError returns any error
func (m *EnhancedApprovalModel) GetError() error {
	return m.err
}

// ============================================================================
// Plain Text Mode Approval (Phase 1.3)
// ============================================================================

// ShowEnhancedApprovalPrompt shows an enhanced approval prompt and returns the user's response
// This is the enhanced version that supports all 4 options (yes/no/always/never)
func ShowEnhancedApprovalPrompt(toolName, toolType, description, details string, plainTextMode bool) (ApprovalOption, error) {
	if plainTextMode {
		return showPlainTextApprovalPrompt(toolName, description, details)
	}

	model := NewEnhancedApprovalModel(toolName, toolType, description, details, false)

	p := tea.NewProgram(model, tea.WithAltScreen())

	m, err := p.Run()
	if err != nil {
		return ApprovalOptionNo, err
	}

	approvalModel, ok := m.(*EnhancedApprovalModel)
	if !ok {
		return ApprovalOptionNo, fmt.Errorf("unexpected model type")
	}

	return approvalModel.GetResult(), nil
}

// showPlainTextApprovalPrompt shows a plain text approval prompt
func showPlainTextApprovalPrompt(toolName, description, details string) (ApprovalOption, error) {
	fmt.Printf("\n%s Tool Approval: %s\n", getEnhancedToolIcon(toolName), toolName)
	fmt.Printf("%s\n\n", description)

	if details != "" {
		fmt.Println("Details:")
		fmt.Println(truncateStringEnhanced(details, 300))
		fmt.Println()
	}

	fmt.Println("Options:")
	fmt.Println("  y - Yes (approve this once)")
	fmt.Println("  n - No (reject this once)")
	fmt.Println("  a - Always (approve this tool type)")
	fmt.Println("  s - Never (reject this tool type)")
	fmt.Print("\nYour choice: ")

	var input string
	_, err := fmt.Scanln(&input)
	if err != nil {
		return ApprovalOptionNo, err
	}

	switch strings.ToLower(strings.TrimSpace(input)) {
	case "y", "yes":
		return ApprovalOptionYes, nil
	case "n", "no":
		return ApprovalOptionNo, nil
	case "a", "always":
		return ApprovalOptionAlways, nil
	case "s", "never":
		return ApprovalOptionNever, nil
	default:
		fmt.Println("Invalid choice, defaulting to No")
		return ApprovalOptionNo, nil
	}
}

// helper functions for enhanced approval

func getEnhancedToolIcon(toolType string) string {
	switch strings.ToLower(toolType) {
	case "command", "shell", "exec":
		return "⚡"
	case "file", "edit", "write", "read":
		return "📄"
	case "browser", "web":
		return "🌐"
	case "mcp", "server":
		return "🔌"
	case "search":
		return "🔍"
	case "delete":
		return "🗑️"
	default:
		return "🔧"
	}
}

func truncateStringEnhanced(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// MCPApprovalPrompt prompts for MCP server approval using enhanced options
func MCPApprovalPromptEnhanced(serverName, toolName string) (ApprovalOption, error) {
	description := fmt.Sprintf("Cline wants to use MCP server '%s'", serverName)
	details := fmt.Sprintf("Tool: %s", toolName)
	return ShowEnhancedApprovalPrompt("mcp", "mcp", description, details, false)
}