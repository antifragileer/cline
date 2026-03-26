// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// WelcomeModel represents the welcome screen state.
type WelcomeModel struct {
	width    int
	height   int
	actions  []WelcomeActionItem
	cursor   int
	selected WelcomeAction
	styles   WelcomeStyles
	// Quick task input
	showInput    bool
	inputValue   string
	// Metadata
	hasHistory   bool
	recentTask   string
}

// WelcomeActionItem represents an action item on the welcome screen.
type WelcomeActionItem struct {
	Action      WelcomeAction
	Title       string
	Description string
	Shortcut    string
	Enabled     bool
}

// WelcomeStyles holds styling for the welcome screen.
type WelcomeStyles struct {
	containerStyle   lipgloss.Style
	titleStyle       lipgloss.Style
	subtitleStyle    lipgloss.Style
	actionStyle      lipgloss.Style
	selectedStyle    lipgloss.Style
	disabledStyle    lipgloss.Style
	shortcutStyle    lipgloss.Style
	descriptionStyle lipgloss.Style
	inputStyle       lipgloss.Style
}

// DefaultWelcomeStyles returns default welcome screen styles.
func DefaultWelcomeStyles() WelcomeStyles {
	return WelcomeStyles{
		containerStyle: lipgloss.NewStyle().
			Padding(2, 4),

		titleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true).
			MarginBottom(1),

		subtitleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")).
			MarginBottom(2),

		actionStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			PaddingLeft(2),

		selectedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00D9FF")).
			Bold(true).
			PaddingLeft(2).
			Background(lipgloss.Color("#1a1a1a")),

		disabledStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#505050")).
			PaddingLeft(2),

		shortcutStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true),

		descriptionStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#606060")).
			PaddingLeft(4),

		inputStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(0, 1),
	}
}

// NewWelcomeModel creates a new welcome model.
func NewWelcomeModel() *WelcomeModel {
	m := &WelcomeModel{
		actions:    make([]WelcomeActionItem, 0),
		styles:     DefaultWelcomeStyles(),
		hasHistory: false,
		cursor:     0,
	}

	m.refreshActions()
	return m
}

// refreshActions updates the action list based on current state.
func (m *WelcomeModel) refreshActions() {
	m.actions = []WelcomeActionItem{
		{
			Action:      ActionNewTask,
			Title:       "New Task",
			Description: "Start a new task with Cline",
			Shortcut:    "n",
			Enabled:     true,
		},
		{
			Action:      ActionContinueTask,
			Title:       "Continue Task",
			Description: "Resume the most recent task",
			Shortcut:    "c",
			Enabled:     m.hasHistory,
		},
		{
			Action:      ActionHistory,
			Title:       "Task History",
			Description: "Browse and resume previous tasks",
			Shortcut:    "h",
			Enabled:     true,
		},
		{
			Action:      ActionSettings,
			Title:       "Settings",
			Description: "Configure API keys and preferences",
			Shortcut:    "s",
			Enabled:     true,
		},
		{
			Action:      ActionHelp,
			Title:       "Help",
			Description: "View documentation and shortcuts",
			Shortcut:    "?",
			Enabled:     true,
		},
		{
			Action:      ActionQuit,
			Title:       "Quit",
			Description: "Exit Cline CLI",
			Shortcut:    "q",
			Enabled:     true,
		},
	}
}

// SetHasHistory sets whether there is task history available.
func (m *WelcomeModel) SetHasHistory(hasHistory bool) {
	m.hasHistory = hasHistory
	m.refreshActions()
}

// SetRecentTask sets the most recent task description.
func (m *WelcomeModel) SetRecentTask(task string) {
	m.recentTask = task
	if task != "" {
		// Update continue action description
		for i, action := range m.actions {
			if action.Action == ActionContinueTask {
				m.actions[i].Description = fmt.Sprintf("Resume: %s", truncateString(task, 40))
				break
			}
		}
	}
}

// Init initializes the model.
func (m WelcomeModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model.
func (m *WelcomeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		if m.showInput {
			return m.handleInputMode(msg)
		}
		return m.handleNavigation(msg)
	}

	return m, nil
}

// handleInputMode handles input when in quick task mode.
func (m *WelcomeModel) handleInputMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		if m.inputValue != "" {
			m.selected = ActionNewTask
			return m, tea.Quit
		}
		m.showInput = false

	case tea.KeyEsc:
		m.showInput = false
		m.inputValue = ""

	case tea.KeyBackspace:
		if len(m.inputValue) > 0 {
			m.inputValue = m.inputValue[:len(m.inputValue)-1]
		}

	case tea.KeyRunes:
		m.inputValue += msg.String()
	}

	return m, nil
}

// handleNavigation handles navigation in the menu.
func (m *WelcomeModel) handleNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp, tea.KeyCtrlP:
		m.moveCursor(-1)

	case tea.KeyDown, tea.KeyCtrlN:
		m.moveCursor(1)

	case tea.KeyEnter:
		return m.selectCurrent()

	case tea.KeyRunes:
		// Check for shortcut keys
		switch msg.String() {
		case "n", "N":
			m.cursor = m.findActionIndex(ActionNewTask)
			m.showInput = true
			return m, nil

		case "c", "C":
			if m.hasHistory {
				m.cursor = m.findActionIndex(ActionContinueTask)
				return m.selectCurrent()
			}

		case "h", "H":
			m.cursor = m.findActionIndex(ActionHistory)
			return m.selectCurrent()

		case "s", "S":
			m.cursor = m.findActionIndex(ActionSettings)
			return m.selectCurrent()

		case "?":
			m.cursor = m.findActionIndex(ActionHelp)
			return m.selectCurrent()

		case "q", "Q":
			m.cursor = m.findActionIndex(ActionQuit)
			return m.selectCurrent()
		}
	}

	return m, nil
}

// moveCursor moves the cursor by the given offset.
func (m *WelcomeModel) moveCursor(offset int) {
	newCursor := m.cursor + offset

	// Skip disabled items
	for newCursor >= 0 && newCursor < len(m.actions) {
		if m.actions[newCursor].Enabled {
			m.cursor = newCursor
			return
		}
		newCursor += offset
	}

	// Wrap around
	if offset > 0 {
		m.cursor = 0
	} else {
		m.cursor = len(m.actions) - 1
	}

	// Find next enabled item
	for !m.actions[m.cursor].Enabled {
		m.cursor += offset
		if m.cursor < 0 {
			m.cursor = len(m.actions) - 1
		} else if m.cursor >= len(m.actions) {
			m.cursor = 0
		}
	}
}

// selectCurrent selects the current action.
func (m *WelcomeModel) selectCurrent() (tea.Model, tea.Cmd) {
	if m.cursor < 0 || m.cursor >= len(m.actions) {
		return m, nil
	}

	action := m.actions[m.cursor]
	if !action.Enabled {
		return m, nil
	}

	m.selected = action.Action
	return m, tea.Quit
}

// findActionIndex finds the index of an action.
func (m *WelcomeModel) findActionIndex(action WelcomeAction) int {
	for i, a := range m.actions {
		if a.Action == action {
			return i
		}
	}
	return 0
}

// View renders the welcome screen.
func (m WelcomeModel) View() string {
	var content strings.Builder

	// Header
	content.WriteString(m.renderHeader())
	content.WriteString("\n\n")

	// Input mode or menu
	if m.showInput {
		content.WriteString(m.renderInputMode())
	} else {
		content.WriteString(m.renderMenu())
	}

	// Footer
	content.WriteString("\n\n")
	content.WriteString(m.renderFooter())

	return m.styles.containerStyle.Render(content.String())
}

// renderHeader renders the header section.
func (m *WelcomeModel) renderHeader() string {
	var content strings.Builder

	// ASCII art logo
	logo := `
   ____ _     ___ _   _ 
  / ___| |   |_ _| \ | |
 | |   | |    | ||  \| |
 | |___| |___ | || |\  |
  \____|_____|___|_| \_|
`
	content.WriteString(m.styles.titleStyle.Render(logo))
	content.WriteString("\n")
	content.WriteString(m.styles.subtitleStyle.Render("Your AI coding assistant"))

	return content.String()
}

// renderMenu renders the action menu.
func (m WelcomeModel) renderMenu() string {
	var content strings.Builder

	for i, action := range m.actions {
		isSelected := i == m.cursor

		var line strings.Builder

		// Cursor indicator
		if isSelected {
			line.WriteString("▶ ")
		} else {
			line.WriteString("  ")
		}

		// Shortcut
		line.WriteString(m.styles.shortcutStyle.Render(fmt.Sprintf("[%s] ", action.Shortcut)))

		// Title
		if isSelected {
			line.WriteString(m.styles.selectedStyle.Render(action.Title))
		} else if !action.Enabled {
			line.WriteString(m.styles.disabledStyle.Render(action.Title))
		} else {
			line.WriteString(m.styles.actionStyle.Render(action.Title))
		}

		content.WriteString(line.String())
		content.WriteString("\n")

		// Description (only for selected item to save space)
		if isSelected {
			content.WriteString(m.styles.descriptionStyle.Render(action.Description))
			content.WriteString("\n")
		}
	}

	return content.String()
}

// renderInputMode renders the quick task input mode.
func (m WelcomeModel) renderInputMode() string {
	var content strings.Builder

	content.WriteString(m.styles.subtitleStyle.Render("Enter your task prompt:\n"))
	content.WriteString(m.styles.inputStyle.Render(m.inputValue + "▌"))
	content.WriteString("\n")
	content.WriteString(m.styles.descriptionStyle.Render("[Enter] to submit • [Esc] to cancel"))

	return content.String()
}

// renderFooter renders the footer section.
func (m WelcomeModel) renderFooter() string {
	if m.showInput {
		return ""
	}

	hints := []string{
		"↑↓ to navigate",
		"Enter to select",
		"letter for shortcut",
		"q to quit",
	}

	return m.styles.descriptionStyle.Render(strings.Join(hints, " • "))
}

// GetSelected returns the selected action.
func (m *WelcomeModel) GetSelected() WelcomeAction {
	return m.selected
}

// GetInputValue returns the input value (for new task).
func (m *WelcomeModel) GetInputValue() string {
	return m.inputValue
}

// IsInputMode returns true if in input mode.
func (m *WelcomeModel) IsInputMode() bool {
	return m.showInput
}

// Reset resets the model state.
func (m *WelcomeModel) Reset() {
	m.selected = -1
	m.cursor = 0
	m.showInput = false
	m.inputValue = ""
}

// SetMode sets the mode (act/plan).
func (m *WelcomeModel) SetMode(mode string) {
	// Mode is not used in welcome screen, but interface requires it
}

// SetDimensions sets the terminal dimensions.
func (m *WelcomeModel) SetDimensions(width, height int) {
	m.width = width
	m.height = height
}

// truncateString truncates a string to the given length.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// getTaskHistoryPath returns the path to the task history file
func getTaskHistoryPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".cline", "data", "taskHistory.json")
}

// WelcomeResultMsg is sent when the welcome screen completes.
type WelcomeResultMsg struct {
	Action      WelcomeAction
	InputValue  string
}