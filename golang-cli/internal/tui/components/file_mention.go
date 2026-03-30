// Package components provides TUI components for the Cline CLI.
package components

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FileResult represents a file search result
type FileResult struct {
	Path   string
	Score  float64
	IsFile bool
}

// FileMentionMenu displays a menu for file mentions with @
type FileMentionMenu struct {
	// State
	query              string
	results            []FileResult
	selectedIndex      int
	isLoading          bool
	showRipgrepWarning bool

	// Dimensions
	width  int
	height int

	// Styling
	containerStyle lipgloss.Style
	selectedStyle  lipgloss.Style
	itemStyle      lipgloss.Style
	queryStyle     lipgloss.Style
	loadingStyle   lipgloss.Style
	emptyStyle     lipgloss.Style
	warningStyle   lipgloss.Style
	fileIconStyle  lipgloss.Style
	dirIconStyle   lipgloss.Style
	pathStyle      lipgloss.Style
}

// NewFileMentionMenu creates a new file mention menu
func NewFileMentionMenu() *FileMentionMenu {
	return &FileMentionMenu{
		results:       make([]FileResult, 0),
		selectedIndex: 0,
		containerStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			Width(60),
		selectedStyle: lipgloss.NewStyle().
			Background(lipgloss.Color("#1a1a1a")).
			Foreground(lipgloss.Color("#00D9FF")).
			Bold(true),
		itemStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0E0E0")),
		queryStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true),
		loadingStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")).
			Italic(true),
		emptyStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")).
			Italic(true),
		warningStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFB000")),
		fileIconStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")),
		dirIconStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFB000")),
		pathStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")),
	}
}

// SetDimensions sets the terminal dimensions
func (m *FileMentionMenu) SetDimensions(width, height int) {
	m.width = width
	m.height = height
	// Adjust container width based on available space
	if width < 70 {
		m.containerStyle = m.containerStyle.Width(width - 10)
	}
}

// SetQuery sets the search query
func (m *FileMentionMenu) SetQuery(query string) {
	m.query = query
	m.selectedIndex = 0
}

// SetResults sets the search results
func (m *FileMentionMenu) SetResults(results []FileResult) {
	m.results = results
	// Reset selection if out of bounds
	if m.selectedIndex >= len(results) {
		m.selectedIndex = 0
	}
}

// SetLoading sets the loading state
func (m *FileMentionMenu) SetLoading(loading bool) {
	m.isLoading = loading
}

// SetRipgrepWarning sets whether to show the ripgrep warning
func (m *FileMentionMenu) SetRipgrepWarning(show bool) {
	m.showRipgrepWarning = show
}

// GetSelectedResult returns the currently selected file result
func (m *FileMentionMenu) GetSelectedResult() *FileResult {
	if m.selectedIndex < 0 || m.selectedIndex >= len(m.results) {
		return nil
	}
	return &m.results[m.selectedIndex]
}

// MoveSelectionUp moves the selection up
func (m *FileMentionMenu) MoveSelectionUp() {
	if m.selectedIndex > 0 {
		m.selectedIndex--
	}
}

// MoveSelectionDown moves the selection down
func (m *FileMentionMenu) MoveSelectionDown() {
	if m.selectedIndex < len(m.results)-1 {
		m.selectedIndex++
	}
}

// Render renders the file mention menu
func (m *FileMentionMenu) Render() string {
	var content strings.Builder

	// Header with query
	content.WriteString(m.queryStyle.Render(fmt.Sprintf("@%s", m.query)))
	content.WriteString("\n")

	// Ripgrep warning
	if m.showRipgrepWarning {
		content.WriteString(m.warningStyle.Render("⚠ ripgrep not found - file search will be slower"))
		content.WriteString("\n")
	}

	// Loading state
	if m.isLoading {
		content.WriteString(m.loadingStyle.Render("Searching..."))
		return m.containerStyle.Render(content.String())
	}

	// Empty state
	if len(m.results) == 0 {
		if m.query != "" {
			content.WriteString(m.emptyStyle.Render("No files found"))
		} else {
			content.WriteString(m.emptyStyle.Render("Type to search files..."))
		}
		return m.containerStyle.Render(content.String())
	}

	// Results
	maxResults := 10
	if len(m.results) < maxResults {
		maxResults = len(m.results)
	}

	for i := 0; i < maxResults; i++ {
		result := m.results[i]
		line := m.renderResult(result, i == m.selectedIndex)
		content.WriteString(line)
		content.WriteString("\n")
	}

	// More results indicator
	if len(m.results) > maxResults {
		content.WriteString(m.emptyStyle.Render(fmt.Sprintf("... and %d more", len(m.results)-maxResults)))
	}

	return m.containerStyle.Render(content.String())
}

// renderResult renders a single file result
func (m *FileMentionMenu) renderResult(result FileResult, isSelected bool) string {
	var icon, name string

	if result.IsFile {
		icon = m.fileIconStyle.Render("📄")
		name = filepath.Base(result.Path)
	} else {
		icon = m.dirIconStyle.Render("📁")
		name = filepath.Base(result.Path) + "/"
	}

	// Get parent directory for context
	dir := filepath.Dir(result.Path)
	if dir == "." || dir == result.Path {
		dir = ""
	}

	var line string
	if dir != "" {
		line = fmt.Sprintf("%s %s %s", icon, name, m.pathStyle.Render(dir))
	} else {
		line = fmt.Sprintf("%s %s", icon, name)
	}

	if isSelected {
		return m.selectedStyle.Render("▶ " + line)
	}
	return m.itemStyle.Render("  " + line)
}

// FileMentionState tracks the file mention state in the input
type FileMentionState struct {
	InMentionMode bool
	Query         string
	AtIndex       int
}

// ExtractMentionQuery extracts mention query from input text
// Returns the mention state with whether we're in mention mode and the query
func ExtractMentionQuery(text string, cursorPos int) FileMentionState {
	state := FileMentionState{
		InMentionMode: false,
		Query:         "",
		AtIndex:       -1,
	}

	// Find the @ before cursor
	if cursorPos == 0 {
		return state
	}

	// Look backwards from cursor for @
	atIndex := -1
	for i := cursorPos - 1; i >= 0; i-- {
		if text[i] == '@' {
			atIndex = i
			break
		}
		// Stop if we hit whitespace or other special chars
		if text[i] == ' ' || text[i] == '\n' || text[i] == '\t' {
			break
		}
	}

	if atIndex == -1 {
		return state
	}

	// Check if there's whitespace between @ and cursor (invalid mention)
	query := text[atIndex+1 : cursorPos]
	if strings.ContainsAny(query, " \t\n") {
		return state
	}

	state.InMentionMode = true
	state.Query = query
	state.AtIndex = atIndex

	return state
}

// InsertMention inserts a file mention into the text at the specified position
func InsertMention(text string, atIndex int, filePath string) string {
	// Find the end of the mention (next whitespace or end of string)
	endIndex := atIndex + 1
	for endIndex < len(text) {
		if text[endIndex] == ' ' || text[endIndex] == '\n' || text[endIndex] == '\t' {
			break
		}
		endIndex++
	}

	// Replace the @query with the file path
	before := text[:atIndex]
	after := text[endIndex:]

	return before + "@" + filePath + after
}

// FileMentionMenuModel is a Bubble Tea model for the file mention menu
type FileMentionMenuModel struct {
	menu *FileMentionMenu
}

// NewFileMentionMenuModel creates a new file mention menu model
func NewFileMentionMenuModel() FileMentionMenuModel {
	return FileMentionMenuModel{
		menu: NewFileMentionMenu(),
	}
}

// Init initializes the model
func (m FileMentionMenuModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m FileMentionMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
func (m FileMentionMenuModel) View() string {
	return m.menu.Render()
}