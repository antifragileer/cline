// Package tui provides terminal UI components for user input handling.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PromptResult represents the result of a prompt operation.
type PromptResult struct {
	Approved bool
	Ok       bool
}

// PromptType represents the type of prompt being displayed.
type PromptType int

const (
	// YesNoPrompt is a simple yes/no confirmation.
	YesNoPrompt PromptType = iota
	// ApprovalPrompt shows an approval/rejection dialog with details.
	ApprovalPrompt
	// ChoicePrompt allows selecting from multiple options.
	ChoicePrompt
)

// PromptKeyMap defines the key bindings for prompt components.
type PromptKeyMap struct {
	Submit  key.Binding
	Quit    key.Binding
	Approve key.Binding
	Reject  key.Binding
	Up      key.Binding
	Down    key.Binding
}

// DefaultPromptKeyMap returns the default key bindings.
func DefaultPromptKeyMap() PromptKeyMap {
	return PromptKeyMap{
		Submit: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "submit"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c", "q"),
			key.WithHelp("ctrl+c/q", "quit"),
		),
		Approve: key.NewBinding(
			key.WithKeys("ctrl+a", "y"),
			key.WithHelp("ctrl+a/y", "approve"),
		),
		Reject: key.NewBinding(
			key.WithKeys("ctrl+r", "n"),
			key.WithHelp("ctrl+r/n", "reject"),
		),
		Up: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "previous"),
		),
		Down: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "next"),
		),
	}
}

// PromptModel is the Bubble Tea model for prompt handling.
type PromptModel struct {
	promptType  PromptType
	title       string
	message     string
	details     string
	choices     []string
	selected    int
	keyMap      PromptKeyMap
	width       int
	height      int
	result      bool
	quitting    bool
	submitted   bool
	showDetails bool
}

// NewPromptModel creates a new prompt model with the specified type and message.
func NewPromptModel(promptType PromptType, title, message string) *PromptModel {
	return &PromptModel{
		promptType: promptType,
		title:      title,
		message:    message,
		keyMap:     DefaultPromptKeyMap(),
		selected:   0,
	}
}

// NewYesNoPrompt creates a simple yes/no confirmation prompt.
func NewYesNoPrompt(title, message string) *PromptModel {
	return NewPromptModel(YesNoPrompt, title, message)
}

// NewApprovalPrompt creates an approval prompt with details.
func NewApprovalPrompt(title, message, details string) *PromptModel {
	m := NewPromptModel(ApprovalPrompt, title, message)
	m.details = details
	return m
}

// NewChoicePrompt creates a choice prompt with multiple options.
func NewChoicePrompt(title, message string, choices []string) *PromptModel {
	m := NewPromptModel(ChoicePrompt, title, message)
	m.choices = choices
	return m
}

// SetWidth sets the width of the prompt component.
func (m *PromptModel) SetWidth(width int) {
	m.width = width
}

// SetHeight sets the height of the prompt component.
func (m *PromptModel) SetHeight(height int) {
	m.height = height
}

// Init initializes the prompt model.
func (m *PromptModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the prompt model.
func (m *PromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetWidth(msg.Width)
		m.SetHeight(msg.Height)

	case tea.KeyMsg:
		// Handle quit keys
		if key.Matches(msg, m.keyMap.Quit) {
			m.quitting = true
			m.result = false
			return m, tea.Quit
		}

		// Handle approval (yes)
		if key.Matches(msg, m.keyMap.Approve) {
			m.result = true
			m.submitted = true
			return m, tea.Quit
		}

		// Handle rejection (no)
		if key.Matches(msg, m.keyMap.Reject) {
			m.result = false
			m.submitted = true
			return m, tea.Quit
		}

		// Handle choice navigation
		if m.promptType == ChoicePrompt {
			switch {
			case key.Matches(msg, m.keyMap.Up):
				if m.selected > 0 {
					m.selected--
				}
			case key.Matches(msg, m.keyMap.Down):
				if m.selected < len(m.choices)-1 {
					m.selected++
				}
			case key.Matches(msg, m.keyMap.Submit):
				m.result = m.selected == 0 // First choice is typically "yes/approve"
				m.submitted = true
				return m, tea.Quit
			}
		}

		// Toggle details view
		if msg.String() == "d" || msg.String() == "?" {
			m.showDetails = !m.showDetails
		}
	}

	return m, nil
}

// View renders the prompt model.
func (m *PromptModel) View() string {
	if m.quitting && !m.submitted {
		return ""
	}

	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4"))
	b.WriteString(titleStyle.Render(m.title))
	b.WriteString("\n\n")

	// Message
	messageStyle := lipgloss.NewStyle()
	b.WriteString(messageStyle.Render(m.message))
	b.WriteString("\n\n")

	// Details (if any and if shown)
	if m.details != "" && m.showDetails {
		detailsStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Border(lipgloss.RoundedBorder()).
			Padding(1).
			Width(m.width - 4)
		b.WriteString(detailsStyle.Render(m.details))
		b.WriteString("\n\n")
	}

	// Choices or buttons
	switch m.promptType {
	case YesNoPrompt, ApprovalPrompt:
		b.WriteString(m.renderButtons())
	case ChoicePrompt:
		b.WriteString(m.renderChoices())
	}

	b.WriteString("\n\n")

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))
	b.WriteString(helpStyle.Render(m.renderHelp()))

	return b.String()
}

// renderButtons renders the yes/no buttons.
func (m *PromptModel) renderButtons() string {
	yesStyle := lipgloss.NewStyle().
		Padding(0, 2).
		MarginRight(2)
	noStyle := lipgloss.NewStyle().
		Padding(0, 2)

	if m.result && m.submitted {
		yesStyle = yesStyle.Background(lipgloss.Color("green")).Foreground(lipgloss.Color("white"))
	} else if !m.result && m.submitted {
		noStyle = noStyle.Background(lipgloss.Color("red")).Foreground(lipgloss.Color("white"))
	} else {
		// Default styling for active selection
		yesStyle = yesStyle.Border(lipgloss.RoundedBorder())
		noStyle = noStyle.Border(lipgloss.RoundedBorder())
	}

	return lipgloss.JoinHorizontal(lipgloss.Center,
		yesStyle.Render("[Y]es"),
		noStyle.Render("[N]o"),
	)
}

// renderChoices renders the choice list.
func (m *PromptModel) renderChoices() string {
	var choices []string
	for i, choice := range m.choices {
		style := lipgloss.NewStyle().Padding(0, 1)
		if i == m.selected {
			style = style.Foreground(lipgloss.Color("green")).
				Bold(true).
				Border(lipgloss.RoundedBorder())
			choice = "> " + choice
		} else {
			choice = "  " + choice
		}
		choices = append(choices, style.Render(choice))
	}
	return strings.Join(choices, "\n")
}

// renderHelp renders the help text for key bindings.
func (m *PromptModel) renderHelp() string {
	var parts []string

	switch m.promptType {
	case YesNoPrompt, ApprovalPrompt:
		parts = append(parts, m.keyMap.Approve.Help().Key+": "+m.keyMap.Approve.Help().Desc)
		parts = append(parts, m.keyMap.Reject.Help().Key+": "+m.keyMap.Reject.Help().Desc)
	case ChoicePrompt:
		parts = append(parts, m.keyMap.Up.Help().Key+": "+m.keyMap.Up.Help().Desc)
		parts = append(parts, m.keyMap.Down.Help().Key+": "+m.keyMap.Down.Help().Desc)
		parts = append(parts, m.keyMap.Submit.Help().Key+": "+m.keyMap.Submit.Help().Desc)
	}

	parts = append(parts, m.keyMap.Quit.Help().Key+": "+m.keyMap.Quit.Help().Desc)

	if m.details != "" {
		if m.showDetails {
			parts = append(parts, "d/? : hide details")
		} else {
			parts = append(parts, "d/? : show details")
		}
	}

	return strings.Join(parts, "  ")
}

// Result returns the prompt result.
func (m *PromptModel) Result() PromptResult {
	return PromptResult{
		Approved: m.result,
		Ok:       m.submitted && !m.quitting,
	}
}

// Run executes the prompt model and returns the result.
func (m *PromptModel) Run() (PromptResult, error) {
	p := tea.NewProgram(m)
	model, err := p.Run()
	if err != nil {
		return PromptResult{}, err
	}

	pm, ok := model.(*PromptModel)
	if !ok {
		return PromptResult{}, fmt.Errorf("unexpected model type")
	}

	return pm.Result(), nil
}

// IsApproved returns true if the user approved.
func (m *PromptModel) IsApproved() bool {
	return m.result && m.submitted
}

// IsRejected returns true if the user rejected.
func (m *PromptModel) IsRejected() bool {
	return !m.result && m.submitted
}

// IsSubmitted returns true if the prompt was submitted.
func (m *PromptModel) IsSubmitted() bool {
	return m.submitted
}

// IsQuitting returns true if the user is quitting.
func (m *PromptModel) IsQuitting() bool {
	return m.quitting
}

// SelectedChoice returns the index of the selected choice for ChoicePrompt.
func (m *PromptModel) SelectedChoice() int {
	return m.selected
}

// SetSelectedChoice sets the selected choice index.
func (m *PromptModel) SetSelectedChoice(index int) {
	if index >= 0 && index < len(m.choices) {
		m.selected = index
	}
}

// Quick helpers for common prompt operations

// AskYesNo displays a yes/no prompt and returns the result.
func AskYesNo(title, message string) (bool, error) {
	p := NewYesNoPrompt(title, message)
	result, err := p.Run()
	if err != nil {
		return false, err
	}
	return result.Approved && result.Ok, nil
}

// AskApproval displays an approval prompt with details and returns the result.
func AskApproval(title, message, details string) (bool, error) {
	p := NewApprovalPrompt(title, message, details)
	result, err := p.Run()
	if err != nil {
		return false, err
	}
	return result.Approved && result.Ok, nil
}

// AskChoice displays a choice prompt and returns the selected index.
func AskChoice(title, message string, choices []string) (int, error) {
	p := NewChoicePrompt(title, message, choices)
	_, err := p.Run()
	if err != nil {
		return -1, err
	}
	return p.SelectedChoice(), nil
}