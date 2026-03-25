// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cline/cline/golang-cli/internal/storage"
)

// LogoASCII is the ASCII art logo for Cline
const LogoASCII = `
   ██████╗██╗     ██╗███╗   ██╗███████╗
  ██╔════╝██║     ██║████╗  ██║██╔════╝
  ██║     ██║     ██║██╔██╗ ██║█████╗  
  ██║     ██║     ██║██║╚██╗██║██╔══╝  
  ╚██████╗███████╗██║██║ ╚████║███████╗
   ╚═════╝╚══════╝╚═╝╚═╝  ╚═══╝╚══════╝
`

// WelcomeModel is the Bubble Tea model for the welcome screen
type WelcomeModel struct {
	width        int
	height       int
	selected     int
	recentTasks  []TaskHistoryItem
	hasConfig    bool
	showHints    bool
	showLogo     bool
	version      string
	storageCtx   *storage.StorageContext
	quitting     bool
	action       string
	selectedData interface{}
	err          error
}

// WelcomeMsg is sent when the welcome screen should be shown
type WelcomeMsg struct{}

// NewTaskMsg is sent when a new task should be started
type NewTaskMsg struct{}

// ResumeTaskMsg is sent when a task should be resumed
type ResumeTaskMsg struct {
	TaskID string
}

// SelectTaskMsg is sent when a task is selected from history
type SelectTaskMsg struct {
	Index int
}

// ShowSettingsMsg is sent when settings should be shown
type ShowSettingsMsg struct{}

// ShowHelpMsg is sent when help should be shown
type ShowHelpMsg struct{}

// QuitMsg is sent when the user wants to quit
type QuitMsg struct{}

// NewWelcomeModel creates a new welcome model
func NewWelcomeModel(recentTasks []TaskHistoryItem, hasConfig bool, version string) WelcomeModel {
	return WelcomeModel{
		selected:    0,
		recentTasks: recentTasks,
		hasConfig:   hasConfig,
		showHints:   true,
		showLogo:    true,
		version:     version,
	}
}

// NewWelcomeModelWithStorage creates a new welcome model with storage context
func NewWelcomeModelWithStorage(recentTasks []TaskHistoryItem, hasConfig bool, version string, storageCtx *storage.StorageContext) WelcomeModel {
	m := NewWelcomeModel(recentTasks, hasConfig, version)
	m.storageCtx = storageCtx
	return m
}

// Init implements tea.Model
func (m WelcomeModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m WelcomeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			m.action = "quit"
			return m, tea.Quit

		case "n":
			m.action = "new_task"
			return m, tea.Quit

		case "enter":
			if m.selected == 0 {
				m.action = "new_task"
				return m, tea.Quit
			} else if m.selected == 1 {
				m.action = "settings"
				return m, tea.Quit
			} else if m.selected >= 2 && m.selected < 2+len(m.recentTasks) {
				taskIdx := m.selected - 2
				if taskIdx < len(m.recentTasks) {
					m.action = "resume_task"
					m.selectedData = m.recentTasks[taskIdx].ID
					return m, tea.Quit
				}
			} else if m.selected == 2+len(m.recentTasks) {
				m.action = "help"
				return m, tea.Quit
			}
			return m, nil

		case "r":
			if len(m.recentTasks) > 0 && m.selected >= 2 && m.selected < 2+len(m.recentTasks) {
				taskIdx := m.selected - 2
				m.action = "resume_task"
				m.selectedData = m.recentTasks[taskIdx].ID
				return m, tea.Quit
			}
			return m, nil

		case "s":
			m.action = "settings"
			return m, tea.Quit

		case "h":
			m.action = "help"
			return m, tea.Quit

		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
			return m, nil

		case "down", "j":
			maxIdx := 2 + len(m.recentTasks) + 1 // New Task, Settings, tasks, Help
			if m.selected < maxIdx-1 {
				m.selected++
			}
			return m, nil
		}

	case error:
		m.err = msg
		return m, nil
	}

	return m, nil
}

// View implements tea.Model
func (m WelcomeModel) View() string {
	if m.width == 0 {
		m.width = 80
	}
	if m.height == 0 {
		m.height = 24
	}

	var content strings.Builder

	// Render logo
	if m.showLogo {
		content.WriteString(m.renderLogo())
		content.WriteString("\n\n")
	}

	// Render main menu
	content.WriteString(m.renderMenu())
	content.WriteString("\n\n")

	// Render recent tasks
	if len(m.recentTasks) > 0 {
		content.WriteString(m.renderRecentTasks())
		content.WriteString("\n\n")
	}

	// Render hints
	if m.showHints {
		content.WriteString(m.renderHints())
		content.WriteString("\n")
	}

	// Render footer
	content.WriteString(m.renderFooter())

	// Center everything vertically if there's room
	mainContent := content.String()
	contentHeight := lipgloss.Height(mainContent)
	if contentHeight < m.height {
		padding := (m.height - contentHeight) / 2
		if padding > 0 {
			mainContent = strings.Repeat("\n", padding) + mainContent
		}
	}

	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Render(mainContent)
}

// renderLogo renders the Cline logo
func (m WelcomeModel) renderLogo() string {
	logoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4")).
		Bold(true)

	versionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#606060")).
		Italic(true)

	logo := logoStyle.Render(LogoASCII)
	if m.version != "" {
		version := versionStyle.Render(fmt.Sprintf("  v%s", m.version))
		logo = lipgloss.JoinHorizontal(lipgloss.Top, logo, version)
	}

	return lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		Render(logo)
}

// renderMenu renders the main menu
func (m WelcomeModel) renderMenu() string {
	menuItems := []struct {
		label string
		key   string
		desc  string
	}{
		{
			label: "New Task",
			key:   "n",
			desc:  "Start a new conversation",
		},
		{
			label: "Settings",
			key:   "s",
			desc:  "Configure API keys and preferences",
		},
	}

	var items []string
	for i, item := range menuItems {
		isSelected := i == m.selected
		items = append(items, m.renderMenuItem(item.label, item.key, item.desc, isSelected))
	}

	return lipgloss.JoinVertical(lipgloss.Left, items...)
}

// renderMenuItem renders a single menu item
func (m WelcomeModel) renderMenuItem(label, key, desc string, selected bool) string {
	var result strings.Builder

	// Key hint
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080")).
		Width(3)

	// Label
	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Width(15)

	// Description
	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808060")).
		Italic(true)

	// Selected styles
	if selected {
		keyStyle = keyStyle.Foreground(lipgloss.Color("#7D56F4"))
		labelStyle = labelStyle.Foreground(lipgloss.Color("#7D56F4"))
		descStyle = descStyle.Foreground(lipgloss.Color("#E0E0E0"))

		// Add selection indicator
		result.WriteString("▶ ")
	} else {
		result.WriteString("  ")
	}

	result.WriteString(keyStyle.Render(fmt.Sprintf("[%s]", key)))
	result.WriteString(" ")
	result.WriteString(labelStyle.Render(label))
	result.WriteString("  ")
	result.WriteString(descStyle.Render(desc))

	return result.String()
}

// renderRecentTasks renders the recent tasks section
func (m WelcomeModel) renderRecentTasks() string {
	var result strings.Builder

	// Section header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#E0E0E0"))

	result.WriteString(headerStyle.Render("Recent Tasks"))
	result.WriteString("\n\n")

	// Task items
	for i, task := range m.recentTasks {
		if i >= 5 { // Show max 5 recent tasks
			break
		}

		idx := 2 + i // Offset by menu items
		isSelected := m.selected == idx

		result.WriteString(m.renderTaskItem(task, i, isSelected))
		result.WriteString("\n")
	}

	return result.String()
}

// renderTaskItem renders a single task item
func (m WelcomeModel) renderTaskItem(task TaskHistoryItem, index int, selected bool) string {
	var result strings.Builder

	// Selection indicator
	if selected {
		result.WriteString("▶ ")
	} else {
		result.WriteString("  ")
	}

	// Index and key hint
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080"))

	result.WriteString(keyStyle.Render(fmt.Sprintf("[%d] ", index+1)))

	// Description (truncated if too long)
	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E0E0E0")).
		Width(50)

	if selected {
		descStyle = descStyle.Foreground(lipgloss.Color("#7D56F4"))
	}

	desc := task.Description
	if len(desc) > 50 {
		desc = desc[:47] + "..."
	}
	result.WriteString(descStyle.Render(desc))

	// Timestamp
	timeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#606060")).
		Italic(true)

	var timeStr string
	if task.Timestamp != "" {
		// Try parsing as RFC3339 first
		if t, err := time.Parse(time.RFC3339, task.Timestamp); err == nil {
			timeStr = formatTimeAgo(t)
		} else {
			// Try as Unix timestamp
			if ts, err := parseUnixTimestamp(task.Timestamp); err == nil {
				timeStr = formatTimeAgo(time.Unix(ts, 0))
			}
		}
	}

	if timeStr != "" {
		result.WriteString("  ")
		result.WriteString(timeStyle.Render(timeStr))
	}

	return result.String()
}

// renderHints renders the keyboard hints
func (m WelcomeModel) renderHints() string {
	hints := []struct {
		key  string
		desc string
	}{
		{"↑/↓", "navigate"},
		{"Enter", "select"},
		{"n", "new task"},
		{"r", "resume"},
		{"q", "quit"},
	}

	var parts []string
	for _, hint := range hints {
		keyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true)

		descStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080"))

		parts = append(parts, fmt.Sprintf("%s %s", keyStyle.Render(hint.key), descStyle.Render(hint.desc)))
	}

	hintStyle := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center)

	return hintStyle.Render(strings.Join(parts, "  •  "))
}

// renderFooter renders the footer section
func (m WelcomeModel) renderFooter() string {
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#404040")).
		Width(m.width).
		Align(lipgloss.Center)

	status := "Ready"
	if !m.hasConfig {
		status = "⚠ Configuration required - press 's' for settings"
	}

	return footerStyle.Render(status)
}

// formatTimeAgo formats a time as a human-readable "ago" string
func formatTimeAgo(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		return fmt.Sprintf("%d min ago", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%d hours ago", int(diff.Hours()))
	case diff < 7*24*time.Hour:
		return fmt.Sprintf("%d days ago", int(diff.Hours()/24))
	default:
		return t.Format("Jan 2")
	}
}

// parseUnixTimestamp parses a Unix timestamp string
func parseUnixTimestamp(s string) (int64, error) {
	var ts int64
	_, err := fmt.Sscanf(s, "%d", &ts)
	return ts, err
}

// getTaskHistoryPath returns the path to the task history file
func getTaskHistoryPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s/.cline/data/taskHistory.json", homeDir)
}

// truncateString truncates a string to the specified length
func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}

// WelcomeScreenResult contains the result from the welcome screen
type WelcomeScreenResult struct {
	Action   string
	Data     interface{}
	Quitting bool
	Error    error
}

// WelcomeScreen runs the welcome screen and returns the selected action
func WelcomeScreen(recentTasks []TaskHistoryItem, hasConfig bool, version string) (string, interface{}, error) {
	model := NewWelcomeModel(recentTasks, hasConfig, version)
	p := tea.NewProgram(model, tea.WithAltScreen())

	m, err := p.Run()
	if err != nil {
		return "", nil, err
	}

	welcomeModel, ok := m.(WelcomeModel)
	if !ok {
		return "", nil, fmt.Errorf("unexpected model type")
	}

	if welcomeModel.err != nil {
		return "", nil, welcomeModel.err
	}

	return welcomeModel.action, welcomeModel.selectedData, nil
}

// WelcomeScreenWithStorage runs the welcome screen with storage access
func WelcomeScreenWithStorage(recentTasks []TaskHistoryItem, hasConfig bool, version string, storageCtx *storage.StorageContext) WelcomeScreenResult {
	model := NewWelcomeModelWithStorage(recentTasks, hasConfig, version, storageCtx)
	p := tea.NewProgram(model, tea.WithAltScreen())

	m, err := p.Run()
	if err != nil {
		return WelcomeScreenResult{
			Quitting: true,
			Error:    err,
		}
	}

	welcomeModel, ok := m.(WelcomeModel)
	if !ok {
		return WelcomeScreenResult{
			Quitting: true,
			Error:    fmt.Errorf("unexpected model type"),
		}
	}

	return WelcomeScreenResult{
		Action:   welcomeModel.action,
		Data:     welcomeModel.selectedData,
		Quitting: welcomeModel.quitting,
		Error:    welcomeModel.err,
	}
}