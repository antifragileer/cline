// Package components provides TUI components for the Cline CLI.
package components

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SlashCommand represents a slash command
type SlashCommand struct {
	Name        string
	Description string
	Category    string // "general", "workflow", "action"
	Shortcut    string // Optional keyboard shortcut
}

// SlashCommandMenu displays a menu for slash commands
type SlashCommandMenu struct {
	// State
	commands      []SlashCommand
	filteredCmds  []SlashCommand
	query         string
	selectedIndex int

	// Dimensions
	width  int
	height int

	// Styling
	containerStyle lipgloss.Style
	selectedStyle  lipgloss.Style
	itemStyle      lipgloss.Style
	queryStyle     lipgloss.Style
	emptyStyle     lipgloss.Style
	categoryStyle  lipgloss.Style
	nameStyle      lipgloss.Style
	descStyle      lipgloss.Style
	workflowStyle  lipgloss.Style
	shortcutStyle  lipgloss.Style
}

// NewSlashCommandMenu creates a new slash command menu
func NewSlashCommandMenu() *SlashCommandMenu {
	m := &SlashCommandMenu{
		commands:      getDefaultSlashCommands(),
		filteredCmds:  make([]SlashCommand, 0),
		selectedIndex: 0,
		containerStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#00D9FF")).
			Padding(0, 1).
			Width(70),
		selectedStyle: lipgloss.NewStyle().
			Background(lipgloss.Color("#1a1a1a")).
			Foreground(lipgloss.Color("#00D9FF")).
			Bold(true),
		itemStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0E0E0")),
		queryStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00D9FF")).
			Bold(true),
		emptyStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")).
			Italic(true),
		categoryStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true),
		nameStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true),
		descStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")),
		workflowStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFB000")),
		shortcutStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true),
	}

	// Sort commands: workflows first, then alphabetically
	m.sortCommands()
	m.filteredCmds = m.commands

	return m
}

// SetDimensions sets the terminal dimensions
func (m *SlashCommandMenu) SetDimensions(width, height int) {
	m.width = width
	m.height = height
	if width < 80 {
		m.containerStyle = m.containerStyle.Width(width - 10)
	}
}

// SetQuery sets the filter query and updates filtered commands
func (m *SlashCommandMenu) SetQuery(query string) {
	m.query = query
	m.filterCommands()
	m.selectedIndex = 0
}

// filterCommands filters the commands based on the query
func (m *SlashCommandMenu) filterCommands() {
	if m.query == "" {
		m.filteredCmds = m.commands
		return
	}

	query := strings.ToLower(m.query)
	var filtered []SlashCommand

	for _, cmd := range m.commands {
		if strings.Contains(strings.ToLower(cmd.Name), query) ||
			strings.Contains(strings.ToLower(cmd.Description), query) {
			filtered = append(filtered, cmd)
		}
	}

	m.filteredCmds = filtered
}

// sortCommands sorts commands: workflows first, then by name
func (m *SlashCommandMenu) sortCommands() {
	sort.Slice(m.commands, func(i, j int) bool {
		// Workflows come first
		if m.commands[i].Category == "workflow" && m.commands[j].Category != "workflow" {
			return true
		}
		if m.commands[i].Category != "workflow" && m.commands[j].Category == "workflow" {
			return false
		}
		// Then sort by name
		return m.commands[i].Name < m.commands[j].Name
	})
}

// MoveSelectionUp moves the selection up
func (m *SlashCommandMenu) MoveSelectionUp() {
	if m.selectedIndex > 0 {
		m.selectedIndex--
	}
}

// MoveSelectionDown moves the selection down
func (m *SlashCommandMenu) MoveSelectionDown() {
	if m.selectedIndex < len(m.filteredCmds)-1 {
		m.selectedIndex++
	}
}

// GetSelectedCommand returns the currently selected command
func (m *SlashCommandMenu) GetSelectedCommand() *SlashCommand {
	if m.selectedIndex < 0 || m.selectedIndex >= len(m.filteredCmds) {
		return nil
	}
	return &m.filteredCmds[m.selectedIndex]
}

// Render renders the slash command menu
func (m *SlashCommandMenu) Render() string {
	var content strings.Builder

	// Header with query
	content.WriteString(m.queryStyle.Render(fmt.Sprintf("/%s", m.query)))
	content.WriteString("\n")

	// Empty state
	if len(m.filteredCmds) == 0 {
		content.WriteString(m.emptyStyle.Render("No commands found"))
		return m.containerStyle.Render(content.String())
	}

	// Group commands by category
	workflows := make([]SlashCommand, 0)
	general := make([]SlashCommand, 0)
	actions := make([]SlashCommand, 0)

	for _, cmd := range m.filteredCmds {
		switch cmd.Category {
		case "workflow":
			workflows = append(workflows, cmd)
		case "action":
			actions = append(actions, cmd)
		default:
			general = append(general, cmd)
		}
	}

	// Render workflows section
	if len(workflows) > 0 {
		content.WriteString(m.categoryStyle.Render("Workflows"))
		content.WriteString("\n")
		for _, cmd := range workflows {
			isSelected := m.isSelected(cmd)
			line := m.renderCommand(cmd, isSelected)
			content.WriteString(line)
			content.WriteString("\n")
		}
		content.WriteString("\n")
	}

	// Render general section
	if len(general) > 0 {
		content.WriteString(m.categoryStyle.Render("Commands"))
		content.WriteString("\n")
		for _, cmd := range general {
			isSelected := m.isSelected(cmd)
			line := m.renderCommand(cmd, isSelected)
			content.WriteString(line)
			content.WriteString("\n")
		}
		content.WriteString("\n")
	}

	// Render actions section
	if len(actions) > 0 {
		content.WriteString(m.categoryStyle.Render("Actions"))
		content.WriteString("\n")
		for _, cmd := range actions {
			isSelected := m.isSelected(cmd)
			line := m.renderCommand(cmd, isSelected)
			content.WriteString(line)
			content.WriteString("\n")
		}
	}

	return m.containerStyle.Render(content.String())
}

// isSelected checks if a command is currently selected
func (m *SlashCommandMenu) isSelected(cmd SlashCommand) bool {
	if m.selectedIndex < 0 || m.selectedIndex >= len(m.filteredCmds) {
		return false
	}
	return m.filteredCmds[m.selectedIndex].Name == cmd.Name
}

// renderCommand renders a single command
func (m *SlashCommandMenu) renderCommand(cmd SlashCommand, isSelected bool) string {
	var nameStyle lipgloss.Style
	if cmd.Category == "workflow" {
		nameStyle = m.workflowStyle
	} else {
		nameStyle = m.nameStyle
	}

	name := nameStyle.Render("/" + cmd.Name)
	desc := m.descStyle.Render(cmd.Description)

	var line string
	if cmd.Shortcut != "" {
		shortcut := m.shortcutStyle.Render(fmt.Sprintf("[%s]", cmd.Shortcut))
		line = fmt.Sprintf("  %s %s %s", name, desc, shortcut)
	} else {
		line = fmt.Sprintf("  %s %s", name, desc)
	}

	if isSelected {
		return m.selectedStyle.Render("▶" + line[1:])
	}
	return m.itemStyle.Render(line)
}

// getDefaultSlashCommands returns the default set of slash commands
func getDefaultSlashCommands() []SlashCommand {
	return []SlashCommand{
		// Workflows
		{Name: "commit", Description: "Generate a commit message", Category: "workflow"},
		{Name: "review", Description: "Review code changes", Category: "workflow"},
		{Name: "explain", Description: "Explain the selected code", Category: "workflow"},
		{Name: "fix", Description: "Fix issues in the code", Category: "workflow"},
		{Name: "test", Description: "Generate tests for the code", Category: "workflow"},
		{Name: "docs", Description: "Generate documentation", Category: "workflow"},

		// General commands
		{Name: "help", Description: "Show help information", Category: "general", Shortcut: "?"},
		{Name: "settings", Description: "Open settings panel", Category: "general", Shortcut: "s"},
		{Name: "history", Description: "Show task history", Category: "general", Shortcut: "h"},
		{Name: "models", Description: "Open model picker", Category: "general", Shortcut: "m"},
		{Name: "skills", Description: "Browse available skills", Category: "general"},

		// Actions
		{Name: "clear", Description: "Clear the conversation", Category: "action"},
		{Name: "exit", Description: "Exit Cline", Category: "action", Shortcut: "q"},
	}
}

// SlashQueryState tracks the slash command state in the input
type SlashQueryState struct {
	InSlashMode bool
	Query       string
	SlashIndex  int
}

// ExtractSlashQuery extracts slash command query from input text
func ExtractSlashQuery(text string, cursorPos int) SlashQueryState {
	state := SlashQueryState{
		InSlashMode: false,
		Query:       "",
		SlashIndex:  -1,
	}

	if cursorPos == 0 {
		return state
	}

	// Look backwards from cursor for /
	slashIndex := -1
	for i := cursorPos - 1; i >= 0; i-- {
		if text[i] == '/' {
			// Make sure this is at the start or after whitespace
			if i == 0 || text[i-1] == ' ' || text[i-1] == '\n' || text[i-1] == '\t' {
				slashIndex = i
				break
			}
		}
		// Stop if we hit whitespace (command is complete)
		if text[i] == ' ' || text[i] == '\n' || text[i] == '\t' {
			break
		}
	}

	if slashIndex == -1 {
		return state
	}

	state.InSlashMode = true
	state.SlashIndex = slashIndex
	state.Query = text[slashIndex+1 : cursorPos]

	return state
}

// InsertSlashCommand inserts a slash command into the text
func InsertSlashCommand(text string, slashIndex int, commandName string) string {
	// Find the end of the command (next whitespace or end of string)
	endIndex := slashIndex + 1
	for endIndex < len(text) {
		if text[endIndex] == ' ' || text[endIndex] == '\n' || text[endIndex] == '\t' {
			break
		}
		endIndex++
	}

	before := text[:slashIndex]
	after := text[endIndex:]

	// Add a space after the command if there's more text
	if len(after) > 0 && after[0] != ' ' {
		after = " " + after
	}

	return before + "/" + commandName + after
}

// GetStandaloneSlashCommandToExecute checks if we should execute a standalone slash command
func GetStandaloneSlashCommandToExecute(prompt string, inSlashMode bool, hasSlashMenu bool, hasPendingAsk bool, isSpinnerActive bool) string {
	// Only execute if:
	// - We're in slash mode
	// - No slash menu is open
	// - No pending ask
	// - Not streaming
	if !inSlashMode || hasSlashMenu || hasPendingAsk || isSpinnerActive {
		return ""
	}

	// Check if prompt is just a slash command with no arguments
	if !strings.HasPrefix(prompt, "/") {
		return ""
	}

	// Get the command name (everything after / until first space)
	trimmed := strings.TrimPrefix(prompt, "/")
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return ""
	}

	commandName := parts[0]

	// Only execute if no arguments
	if len(parts) > 1 {
		return ""
	}

	return commandName
}

// SlashCommandMenuModel is a Bubble Tea model for the slash command menu
type SlashCommandMenuModel struct {
	menu *SlashCommandMenu
}

// NewSlashCommandMenuModel creates a new slash command menu model
func NewSlashCommandMenuModel() SlashCommandMenuModel {
	return SlashCommandMenuModel{
		menu: NewSlashCommandMenu(),
	}
}

// Init initializes the model
func (m SlashCommandMenuModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m SlashCommandMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.menu.SetDimensions(msg.Width, msg.Height)

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			m.menu.MoveSelectionUp()
		case tea.KeyDown:
			m.menu.MoveSelectionDown()
		}
	}

	return m, nil
}

// View renders the menu
func (m SlashCommandMenuModel) View() string {
	return m.menu.Render()
}
