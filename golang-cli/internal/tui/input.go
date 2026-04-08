// Package tui provides terminal UI components for user input handling.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// InputResult represents the result of an input operation.
type InputResult struct {
	Value   string
	Ok      bool
	History []string
}

// InputMode represents the type of input being collected.
type InputMode int

const (
	// SingleLineInput is for single-line text input.
	SingleLineInput InputMode = iota
	// MultiLineInput is for multi-line textarea input.
	MultiLineInput
)

// InputKeyMap defines the key bindings for input components.
type InputKeyMap struct {
	Submit  key.Binding
	Quit    key.Binding
	Approve key.Binding
	Reject  key.Binding
	Up      key.Binding
	Down    key.Binding
	History key.Binding
}

// DefaultInputKeyMap returns the default key bindings.
func DefaultInputKeyMap() InputKeyMap {
	return InputKeyMap{
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
		History: key.NewBinding(
			key.WithKeys("ctrl+p", "ctrl+n"),
			key.WithHelp("ctrl+p/n", "history"),
		),
	}
}

// InputModel is the Bubble Tea model for input handling.
type InputModel struct {
	mode        InputMode
	textInput   textinput.Model
	textArea    textarea.Model
	keyMap      InputKeyMap
	history     *History
	placeholder string
	title       string
	width       int
	height      int
	result      string
	quitting    bool
	submitted   bool
	err         error
}

// NewInputModel creates a new input model with the specified mode and title.
func NewInputModel(mode InputMode, title string) *InputModel {
	m := &InputModel{
		mode:    mode,
		keyMap:  DefaultInputKeyMap(),
		title:   title,
		history: NewHistory(100),
	}

	switch mode {
	case SingleLineInput:
		m.textInput = textinput.New()
		m.textInput.Focus()
	case MultiLineInput:
		m.textArea = textarea.New()
		m.textArea.Focus()
	}

	return m
}

// NewSingleLineInput creates a new single-line input model.
func NewSingleLineInput(title, placeholder string) *InputModel {
	m := NewInputModel(SingleLineInput, title)
	m.placeholder = placeholder
	m.textInput.Placeholder = placeholder
	return m
}

// NewMultiLineInput creates a new multi-line textarea input model.
func NewMultiLineInput(title, placeholder string) *InputModel {
	m := NewInputModel(MultiLineInput, title)
	m.placeholder = placeholder
	m.textArea.Placeholder = placeholder
	return m
}

// SetHistory sets the input history for navigation.
func (m *InputModel) SetHistory(history *History) {
	m.history = history
}

// GetHistory returns the current input history.
func (m *InputModel) GetHistory() *History {
	return m.history
}

// AddToHistory adds a value to the history.
func (m *InputModel) AddToHistory(value string) {
	if m.history != nil && value != "" {
		m.history.Add(value)
	}
}

// SetWidth sets the width of the input component.
func (m *InputModel) SetWidth(width int) {
	m.width = width
	if m.mode == MultiLineInput {
		m.textArea.SetWidth(width)
	} else {
		m.textInput.Width = width
	}
}

// SetHeight sets the height of the input component.
func (m *InputModel) SetHeight(height int) {
	m.height = height
	if m.mode == MultiLineInput {
		m.textArea.SetHeight(height)
	}
}

// Init initializes the input model.
func (m *InputModel) Init() tea.Cmd {
	if m.mode == SingleLineInput {
		return textinput.Blink
	}
	return textarea.Blink
}

// Update handles messages and updates the input model.
func (m *InputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetWidth(msg.Width)
		m.SetHeight(msg.Height)

	case tea.KeyMsg:
		// Handle quit keys
		if key.Matches(msg, m.keyMap.Quit) {
			m.quitting = true
			return m, tea.Quit
		}

		// Handle history navigation
		if m.history != nil {
			if key.Matches(msg, m.keyMap.History) || key.Matches(msg, m.keyMap.Up) || key.Matches(msg, m.keyMap.Down) {
				switch {
				case key.Matches(msg, m.keyMap.Up) || key.Matches(msg, m.keyMap.History) && msg.String() == "ctrl+p":
					if prev := m.history.Previous(); prev != "" {
						if m.mode == SingleLineInput {
							m.textInput.SetValue(prev)
						} else {
							m.textArea.SetValue(prev)
						}
					}
				case key.Matches(msg, m.keyMap.Down) || key.Matches(msg, m.keyMap.History) && msg.String() == "ctrl+n":
					if next := m.history.Next(); next != "" {
						if m.mode == SingleLineInput {
							m.textInput.SetValue(next)
						} else {
							m.textArea.SetValue(next)
						}
					} else {
						// Clear input if at end of history
						if m.mode == SingleLineInput {
							m.textInput.SetValue("")
						} else {
							m.textArea.SetValue("")
						}
					}
				}
				return m, nil
			}
		}

		// Handle submit
		if key.Matches(msg, m.keyMap.Submit) {
			if m.mode == SingleLineInput {
				m.result = m.textInput.Value()
			} else {
				m.result = m.textArea.Value()
			}
			m.submitted = true
			m.AddToHistory(m.result)
			return m, tea.Quit
		}
	}

	// Update the appropriate input component
	if m.mode == SingleLineInput {
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		m.textArea, cmd = m.textArea.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the input model.
func (m *InputModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4"))

	b.WriteString(titleStyle.Render(m.title))
	b.WriteString("\n\n")

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))
	b.WriteString(helpStyle.Render(m.renderHelp()))
	b.WriteString("\n\n")

	// Input
	if m.mode == SingleLineInput {
		b.WriteString(m.textInput.View())
	} else {
		b.WriteString(m.textArea.View())
	}

	// Error display
	if m.err != nil {
		b.WriteString("\n\n")
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("red"))
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
	}

	return b.String()
}

// renderHelp renders the help text for key bindings.
func (m *InputModel) renderHelp() string {
	var parts []string
	parts = append(parts, m.keyMap.Submit.Help().Key+": "+m.keyMap.Submit.Help().Desc)
	parts = append(parts, m.keyMap.Quit.Help().Key+": "+m.keyMap.Quit.Help().Desc)
	if m.history != nil && m.history.Len() > 0 {
		parts = append(parts, m.keyMap.History.Help().Key+": "+m.keyMap.History.Help().Desc)
	}
	return strings.Join(parts, "  ")
}

// Result returns the input result.
func (m *InputModel) Result() InputResult {
	return InputResult{
		Value:   m.result,
		Ok:      m.submitted && !m.quitting,
		History: m.history.GetAll(),
	}
}

// Error returns any error that occurred during input.
func (m *InputModel) Error() error {
	return m.err
}

// Run executes the input model and returns the result.
func (m *InputModel) Run() (InputResult, error) {
	p := tea.NewProgram(m)
	model, err := p.Run()
	if err != nil {
		return InputResult{}, err
	}

	im, ok := model.(*InputModel)
	if !ok {
		return InputResult{}, fmt.Errorf("unexpected model type")
	}

	return im.Result(), nil
}

// GetValue returns the current input value without submitting.
func (m *InputModel) GetValue() string {
	if m.mode == SingleLineInput {
		return m.textInput.Value()
	}
	return m.textArea.Value()
}

// SetValue sets the input value programmatically.
func (m *InputModel) SetValue(value string) {
	if m.mode == SingleLineInput {
		m.textInput.SetValue(value)
	} else {
		m.textArea.SetValue(value)
	}
}

// Focus sets focus on the input.
func (m *InputModel) Focus() tea.Cmd {
	if m.mode == SingleLineInput {
		return m.textInput.Focus()
	}
	return m.textArea.Focus()
}

// Blur removes focus from the input.
func (m *InputModel) Blur() {
	if m.mode == SingleLineInput {
		m.textInput.Blur()
	} else {
		m.textArea.Blur()
	}
}

// Reset clears the input and resets the model state.
func (m *InputModel) Reset() {
	if m.mode == SingleLineInput {
		m.textInput.Reset()
	} else {
		m.textArea.Reset()
	}
	m.result = ""
	m.submitted = false
	m.quitting = false
	m.err = nil
}

// IsSubmitted returns true if the input was submitted.
func (m *InputModel) IsSubmitted() bool {
	return m.submitted
}

// IsQuitting returns true if the user is quitting.
func (m *InputModel) IsQuitting() bool {
	return m.quitting
}
