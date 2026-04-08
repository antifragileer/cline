// Package components provides TUI components for the Cline CLI.
package components

import (
	"fmt"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ApprovalType represents the type of approval being requested
type ApprovalType string

const (
	// ApprovalTypeTool represents a tool execution approval
	ApprovalTypeTool ApprovalType = "tool"
	// ApprovalTypeCommand represents a command execution approval
	ApprovalTypeCommand ApprovalType = "command"
	// ApprovalTypeFile represents a file operation approval
	ApprovalTypeFile ApprovalType = "file"
	// ApprovalTypeBrowser represents a browser action approval
	ApprovalTypeBrowser ApprovalType = "browser"
	// ApprovalTypeMCP represents an MCP tool approval
	ApprovalTypeMCP ApprovalType = "mcp"
)

// ApprovalPrompt represents a tool approval prompt
type ApprovalPrompt struct {
	ID          string
	Type        ApprovalType
	Title       string
	Description string
	Details     map[string]string
	Timestamp   string
}

// ApprovalResult represents the result of an approval decision
type ApprovalResult struct {
	PromptID string
	Approved bool
	Remember bool // Whether to remember this decision
}

// ApprovalModel is a Bubble Tea model for approval prompts
type ApprovalModel struct {
	prompt      *ApprovalPrompt
	selected    int
	choices     []string
	showDetails bool
	remember    bool
	width       int
	height      int
}

// NewApprovalModel creates a new approval model
func NewApprovalModel() *ApprovalModel {
	return &ApprovalModel{
		selected: 0,
		choices:  []string{"Yes", "No", "Yes, always for this task", "View details"},
		remember: false,
	}
}

// SetPrompt sets the current approval prompt
func (am *ApprovalModel) SetPrompt(prompt *ApprovalPrompt) {
	am.prompt = prompt
	am.selected = 0
	am.showDetails = false
	am.remember = false
}

// Clear clears the current prompt
func (am *ApprovalModel) Clear() {
	am.prompt = nil
	am.selected = 0
	am.showDetails = false
}

// HasPrompt returns true if there's an active prompt
func (am *ApprovalModel) HasPrompt() bool {
	return am.prompt != nil
}

// SetDimensions sets the model dimensions
func (am *ApprovalModel) SetDimensions(width, height int) {
	am.width = width
	am.height = height
}

// Init implements tea.Model
func (am ApprovalModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (am ApprovalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if am.prompt == nil {
		return am, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if am.selected > 0 {
				am.selected--
			}
		case "down", "j":
			if am.selected < len(am.choices)-1 {
				am.selected++
			}
		case "enter":
			return am.handleSelection()
		case "y":
			if !am.showDetails {
				return am, func() tea.Msg {
					return ApprovalResult{PromptID: am.prompt.ID, Approved: true, Remember: false}
				}
			}
		case "n":
			if !am.showDetails {
				return am, func() tea.Msg {
					return ApprovalResult{PromptID: am.prompt.ID, Approved: false, Remember: false}
				}
			}
		case "d":
			am.showDetails = !am.showDetails
		case "r":
			if !am.showDetails {
				am.remember = !am.remember
			}
		case "esc":
			if am.showDetails {
				am.showDetails = false
				return am, nil
			}
		}
	}

	return am, nil
}

// handleSelection handles the enter key selection
func (am ApprovalModel) handleSelection() (tea.Model, tea.Cmd) {
	choice := am.choices[am.selected]

	switch choice {
	case "Yes":
		return am, func() tea.Msg {
			return ApprovalResult{PromptID: am.prompt.ID, Approved: true, Remember: false}
		}
	case "No":
		return am, func() tea.Msg {
			return ApprovalResult{PromptID: am.prompt.ID, Approved: false, Remember: false}
		}
	case "Yes, always for this task":
		return am, func() tea.Msg {
			return ApprovalResult{PromptID: am.prompt.ID, Approved: true, Remember: true}
		}
	case "View details":
		am.showDetails = !am.showDetails
		return am, nil
	}

	return am, nil
}

// View implements tea.Model
func (am ApprovalModel) View() string {
	if am.prompt == nil {
		return ""
	}

	var s strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#F39C12")).
		MarginBottom(1)

	s.WriteString(titleStyle.Render("⚠️  Approval Required"))
	s.WriteString("\n\n")

	// Prompt details
	promptStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#F39C12")).
		Padding(1, 2).
		Width(am.width - 4)

	content := fmt.Sprintf("%s\n\n%s", am.prompt.Title, am.prompt.Description)
	s.WriteString(promptStyle.Render(content))
	s.WriteString("\n\n")

	// Show details if requested
	if am.showDetails {
		detailsStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CCCCCC")).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#666666")).
			Padding(1, 2).
			Width(am.width - 4)

		var details strings.Builder
		for key, value := range am.prompt.Details {
			details.WriteString(fmt.Sprintf("%s: %s\n", key, value))
		}

		s.WriteString(detailsStyle.Render(details.String()))
		s.WriteString("\n\n")
	}

	// Choices
	for i, choice := range am.choices {
		if i == len(am.choices)-1 && am.showDetails {
			choice = "Hide details"
		}

		cursor := "  "
		if am.selected == i {
			cursor = "▸ "
		}

		choiceStyle := lipgloss.NewStyle()
		if am.selected == i {
			choiceStyle = choiceStyle.Bold(true).Foreground(lipgloss.Color("#4A90D9"))
		} else {
			choiceStyle = choiceStyle.Foreground(lipgloss.Color("#888888"))
		}

		s.WriteString(choiceStyle.Render(cursor + choice))
		s.WriteString("\n")
	}

	// Remember checkbox
	if !am.showDetails {
		s.WriteString("\n")
		checkbox := "[ ]"
		if am.remember {
			checkbox = "[x]"
		}
		rememberStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
		s.WriteString(rememberStyle.Render(fmt.Sprintf("%s Remember this choice (press 'r')", checkbox)))
	}

	// Help text
	s.WriteString("\n\n")
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Italic(true)
	s.WriteString(helpStyle.Render("y: yes | n: no | d: toggle details | ↑↓: navigate | enter: select"))

	return s.String()
}

// ApprovalKeyMap defines keybindings for the approval prompt
type ApprovalKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Select   key.Binding
	Yes      key.Binding
	No       key.Binding
	Details  key.Binding
	Remember key.Binding
	Quit     key.Binding
}

// DefaultApprovalKeyMap returns default keybindings
func DefaultApprovalKeyMap() ApprovalKeyMap {
	return ApprovalKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Yes: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "yes"),
		),
		No: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "no"),
		),
		Details: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "details"),
		),
		Remember: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "remember"),
		),
		Quit: key.NewBinding(
			key.WithKeys("esc", "q"),
			key.WithHelp("esc/q", "quit"),
		),
	}
}

// ApprovalQueue manages multiple pending approvals
type ApprovalQueue struct {
	prompts []*ApprovalPrompt
	mu      sync.Mutex
	current int
}

// AddPrompt adds a prompt to the queue
func (aq *ApprovalQueue) AddPrompt(prompt *ApprovalPrompt) {
	aq.mu.Lock()
	defer aq.mu.Unlock()
	aq.prompts = append(aq.prompts, prompt)
}

// GetCurrent returns the current prompt
func (aq *ApprovalQueue) GetCurrent() *ApprovalPrompt {
	aq.mu.Lock()
	defer aq.mu.Unlock()
	if aq.current >= len(aq.prompts) {
		return nil
	}
	return aq.prompts[aq.current]
}

// Next moves to the next prompt
func (aq *ApprovalQueue) Next() {
	aq.mu.Lock()
	defer aq.mu.Unlock()
	if aq.current < len(aq.prompts) {
		aq.current++
	}
}

// IsEmpty returns true if the queue is empty
func (aq *ApprovalQueue) IsEmpty() bool {
	aq.mu.Lock()
	defer aq.mu.Unlock()
	return aq.current >= len(aq.prompts)
}

// Count returns the number of pending approvals
func (aq *ApprovalQueue) Count() int {
	aq.mu.Lock()
	defer aq.mu.Unlock()
	return len(aq.prompts) - aq.current
}

// Clear clears all prompts
func (aq *ApprovalQueue) Clear() {
	aq.mu.Lock()
	defer aq.mu.Unlock()
	aq.prompts = make([]*ApprovalPrompt, 0)
	aq.current = 0
}
