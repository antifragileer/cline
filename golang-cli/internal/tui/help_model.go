// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HelpSection represents a section in the help view.
type HelpSection struct {
	Title   string
	Content string
}

// HelpModel represents the help screen state.
type HelpModel struct {
	width    int
	height   int
	sections []HelpSection
	cursor   int
	goBack   bool
	styles   HelpStyles
}

// HelpStyles holds styling for the help screen.
type HelpStyles struct {
	containerStyle   lipgloss.Style
	titleStyle       lipgloss.Style
	subtitleStyle    lipgloss.Style
	sectionStyle     lipgloss.Style
	sectionTitleStyle lipgloss.Style
	contentStyle     lipgloss.Style
	keyStyle         lipgloss.Style
	descriptionStyle lipgloss.Style
	helpStyle        lipgloss.Style
}

// DefaultHelpStyles returns default help screen styles.
func DefaultHelpStyles() HelpStyles {
	return HelpStyles{
		containerStyle: lipgloss.NewStyle().
			Padding(2, 4),

		titleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true).
			MarginBottom(1),

		subtitleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")).
			MarginBottom(2),

		sectionStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0E0E0")).
			MarginTop(1).
			MarginBottom(1),

		sectionTitleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00D9FF")).
			Bold(true),

		contentStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#C0C0C0")),

		keyStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true),

		descriptionStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")),

		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#606060")).
			MarginTop(1),
	}
}

// NewHelpModel creates a new help model.
func NewHelpModel() *HelpModel {
	m := &HelpModel{
		sections: getHelpSections(),
		styles:   DefaultHelpStyles(),
		cursor:   0,
	}
	return m
}

// getHelpSections returns the help content sections.
func getHelpSections() []HelpSection {
	return []HelpSection{
		{
			Title: "Quick Start",
			Content: `cline "your task here"     Execute a single task
cline                       Start interactive mode
cline -p "task"            Run in plan mode
cline -y "task"            Run with auto-approve (yolo mode)`,
		},
		{
			Title: "Global Shortcuts",
			Content: `Ctrl+C                      Quit application
q                           Go back / quit current view
?                           Show this help`,
		},
		{
			Title: "Welcome Screen",
			Content: `n                           New task
c                           Continue recent task
h                           View task history
s                           Open settings
?                           Show help`,
		},
		{
			Title: "Chat Interface",
			Content: `↑/↓ or k/j                  Scroll through messages
Enter                       Send message
Ctrl+L                      Clear conversation
Esc                         Go back to welcome`,
		},
		{
			Title: "History View",
			Content: `↑/↓ or k/j                  Navigate tasks
Enter                       Select/resume task
/                           Search tasks
d                           Delete task
←/→ or h/l                  Change page`,
		},
		{
			Title: "Settings View",
			Content: `↑/↓ or k/j                  Navigate settings
Enter/E                     Edit setting
←/→ or h/l                  Switch tabs (in config)`,
		},
		{
			Title: "Command Flags",
			Content: `-a, --act                   Run in act mode (default)
-p, --plan                  Run in plan mode
-y, --yolo                  Auto-approve all actions
-m, --model MODEL           Use specific model
-t, --timeout SECONDS       Set timeout
--json                      Output as JSON
-T, --taskId ID             Resume task by ID
--continue                  Resume most recent task`,
		},
		{
			Title: "Environment Variables",
			Content: `CLINE_API_KEY              API key for authentication
CLINE_PROVIDER             Default provider (cline, openai, etc.)
CLINE_MODEL                Default model
HOME                       Used to locate config files`,
		},
		{
			Title: "Configuration Files",
			Content: `~/.cline/data/globalState.json    Global settings
~/.cline/data/secrets.json        API keys (encrypted)
~/.cline/data/taskHistory.json    Task history`,
		},
	}
}

// SetDimensions sets the terminal dimensions.
func (m *HelpModel) SetDimensions(width, height int) {
	m.width = width
	m.height = height
}

// Init initializes the model.
func (m HelpModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model.
func (m *HelpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			m.goBack = true
			return m, nil

		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case tea.KeyDown:
			if m.cursor < len(m.sections)-1 {
				m.cursor++
			}
			return m, nil

		case tea.KeyRunes:
			switch msg.String() {
			case "q", "Q":
				m.goBack = true
				return m, nil
			}
		}
	}

	return m, nil
}

// View renders the help screen.
func (m HelpModel) View() string {
	var content strings.Builder

	// Title
	content.WriteString(m.styles.titleStyle.Render("❓ Cline CLI Help"))
	content.WriteString("\n")
	content.WriteString(m.styles.subtitleStyle.Render("Keyboard shortcuts and usage guide"))
	content.WriteString("\n\n")

	// Calculate visible range
	maxVisible := m.calculateVisibleSections()
	startIdx := 0
	endIdx := len(m.sections)

	if m.cursor >= maxVisible {
		startIdx = m.cursor - maxVisible + 1
		endIdx = startIdx + maxVisible
		if endIdx > len(m.sections) {
			endIdx = len(m.sections)
			startIdx = endIdx - maxVisible
			if startIdx < 0 {
				startIdx = 0
			}
		}
	}

	// Sections
	for i := startIdx; i < endIdx && i < len(m.sections); i++ {
		section := m.sections[i]
		isSelected := i == m.cursor

		// Section title
		titlePrefix := "  "
		if isSelected {
			titlePrefix = "▶ "
			content.WriteString(m.styles.sectionTitleStyle.Render(titlePrefix + section.Title))
		} else {
			content.WriteString(m.styles.sectionTitleStyle.Render(titlePrefix + section.Title))
		}
		content.WriteString("\n")

		// Section content
		lines := strings.Split(section.Content, "\n")
		for _, line := range lines {
			// Highlight keys in the content
			formatted := m.formatHelpLine(line)
			content.WriteString("    ")
			content.WriteString(formatted)
			content.WriteString("\n")
		}
		content.WriteString("\n")
	}

	// Scroll indicator
	if len(m.sections) > maxVisible {
		scrollInfo := fmt.Sprintf("Showing %d-%d of %d", startIdx+1, endIdx, len(m.sections))
		content.WriteString(m.styles.descriptionStyle.Render(scrollInfo))
		content.WriteString("\n")
	}

	// Help
	content.WriteString("\n")
	content.WriteString(m.styles.helpStyle.Render("↑↓ to navigate • q/Esc to go back"))

	return m.styles.containerStyle.Render(content.String())
}

// calculateVisibleSections calculates how many sections can be visible.
func (m *HelpModel) calculateVisibleSections() int {
	// Reserve lines for header, help
	headerLines := 4 // title, subtitle, blank line
	footerLines := 2 // help text + padding

	availableHeight := m.height - headerLines - footerLines
	if availableHeight < 10 {
		return 1
	}

	// Each section takes approximately 8-10 lines on average
	return availableHeight / 8
}

// formatHelpLine formats a help line with styled keys.
func (m *HelpModel) formatHelpLine(line string) string {
	// Split by tab or multiple spaces to separate key from description
	parts := strings.SplitN(line, "  ", 2)
	if len(parts) == 2 {
		key := strings.TrimSpace(parts[0])
		desc := strings.TrimSpace(parts[1])
		return m.styles.keyStyle.Render(key) + "  " + m.styles.contentStyle.Render(desc)
	}

	// If no clear separation, just return the line
	return m.styles.contentStyle.Render(line)
}

// ShouldGoBack returns true if the user wants to go back.
func (m *HelpModel) ShouldGoBack() bool {
	return m.goBack
}

// Reset resets the model state.
func (m *HelpModel) Reset() {
	m.cursor = 0
	m.goBack = false
}