// Package tui provides a History View component for the Cline CLI.
// This implements an interactive task history browser with keyboard navigation,
// pagination, search/filter functionality, and task management operations.
package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cline/cline/golang-cli/internal/storage"
)

// HistoryViewState represents the current state of the history view.
type HistoryViewState int

const (
	// HistoryViewStateLoading is the initial loading state.
	HistoryViewStateLoading HistoryViewState = iota
	// HistoryViewStateDisplaying shows the history list.
	HistoryViewStateDisplaying
	// HistoryViewStateSearching shows the search input.
	HistoryViewStateSearching
	// HistoryViewStateConfirmDelete shows the delete confirmation.
	HistoryViewStateConfirmDelete
	// HistoryViewStateExited indicates the user exited.
	HistoryViewStateExited
	// HistoryViewStateTaskSelected indicates a task was selected.
	HistoryViewStateTaskSelected
)

// TaskHistoryEntry represents a single task history record.
type TaskHistoryEntry struct {
	ID        string   `json:"id"`
	Task      string   `json:"task"`
	Timestamp int64    `json:"ts"`
	Metadata  Metadata `json:"metadata,omitempty"`
}

// Metadata contains additional task information.
type Metadata struct {
	Model     string  `json:"model,omitempty"`
	Mode      string  `json:"mode,omitempty"`
	Completed bool    `json:"completed,omitempty"`
	TotalCost float64 `json:"totalCost,omitempty"`
}

// HistoryViewConfig configures the history view behavior.
type HistoryViewConfig struct {
	// Limit is the maximum number of entries to display per page.
	Limit int
	// StorageContext provides access to persistent storage.
	StorageContext *storage.StorageContext
	// OnSelectTask is called when a task is selected.
	OnSelectTask func(taskID string)
	// OnDeleteTask is called when a task is deleted.
	OnDeleteTask func(taskID string) error
}

// DefaultHistoryViewConfig returns a default history view configuration.
func DefaultHistoryViewConfig() HistoryViewConfig {
	return HistoryViewConfig{
		Limit: 10,
	}
}

// HistoryViewKeyMap defines key bindings for the history view.
type HistoryViewKeyMap struct {
	Up           key.Binding
	Down         key.Binding
	PageUp       key.Binding
	PageDown     key.Binding
	Select       key.Binding
	Search       key.Binding
	Delete       key.Binding
	NextPage     key.Binding
	PrevPage     key.Binding
	Quit         key.Binding
	Cancel       key.Binding
	ConfirmYes   key.Binding
	ConfirmNo    key.Binding
}

// DefaultHistoryViewKeyMap returns the default key bindings.
func DefaultHistoryViewKeyMap() HistoryViewKeyMap {
	return HistoryViewKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "K"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "J"),
			key.WithHelp("pgdown", "page down"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d", "delete"),
			key.WithHelp("d", "delete"),
		),
		NextPage: key.NewBinding(
			key.WithKeys("right", "n", "l"),
			key.WithHelp("→/n", "next page"),
		),
		PrevPage: key.NewBinding(
			key.WithKeys("left", "p", "h"),
			key.WithHelp("←/p", "prev page"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		ConfirmYes: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "yes"),
		),
		ConfirmNo: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "no"),
		),
	}
}

// HistoryViewResult contains the outcome of showing the history view.
type HistoryViewResult struct {
	State        HistoryViewState
	SelectedTask string
	WasDeleted   bool
}

// HistoryViewModel is the Bubble Tea model for the history view.
type HistoryViewModel struct {
	config     HistoryViewConfig
	keyMap     HistoryViewKeyMap
	state      HistoryViewState
	width      int
	height     int

	// Data
	allEntries    []TaskHistoryEntry
	filteredEntries []TaskHistoryEntry

	// Pagination
	currentPage   int
	totalPages    int
	selectedIndex int

	// Search
	searchInput   textinput.Model
	searchQuery   string
	isSearching   bool

	// Delete confirmation
	deleteCandidate *TaskHistoryEntry

	// Error
	err error
}

// NewHistoryViewModel creates a new history view model with the given configuration.
func NewHistoryViewModel(config HistoryViewConfig) *HistoryViewModel {
	ti := textinput.New()
	ti.Placeholder = "Search tasks..."
	ti.Prompt = "🔍 "

	return &HistoryViewModel{
		config:          config,
		keyMap:          DefaultHistoryViewKeyMap(),
		state:           HistoryViewStateLoading,
		width:           80,
		height:          24,
		allEntries:      make([]TaskHistoryEntry, 0),
		filteredEntries: make([]TaskHistoryEntry, 0),
		currentPage:     1,
		selectedIndex:   0,
		searchInput:     ti,
	}
}

// Init implements tea.Model.
func (m *HistoryViewModel) Init() tea.Cmd {
	return m.loadDataCmd()
}

// loadDataCmd returns a command that loads data asynchronously.
func (m *HistoryViewModel) loadDataCmd() tea.Cmd {
	return func() tea.Msg {
		return m.loadData()
	}
}

// historyDataLoadedMsg is sent when data loading completes.
type historyDataLoadedMsg struct {
	entries []TaskHistoryEntry
	err     error
}

// loadData loads task history from storage.
func (m *HistoryViewModel) loadData() tea.Msg {
	entries, err := m.loadTaskHistory()
	if err != nil {
		return historyDataLoadedMsg{err: err}
	}

	// Sort entries by timestamp (newest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp > entries[j].Timestamp
	})

	return historyDataLoadedMsg{entries: entries}
}

// loadTaskHistory loads task entries from the history file.
func (m *HistoryViewModel) loadTaskHistory() ([]TaskHistoryEntry, error) {
	historyPath := getTaskHistoryPath()

	// Check if file exists
	if _, err := os.Stat(historyPath); os.IsNotExist(err) {
		return []TaskHistoryEntry{}, nil
	}

	// Create storage instance
	fileStorage, err := storage.NewClineFileStorage(historyPath, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}
	defer fileStorage.Close()

	// Try to get entries
	val, ok := fileStorage.Get("entries")
	if !ok {
		return []TaskHistoryEntry{}, nil
	}

	// Handle different possible structures
	switch v := val.(type) {
	case []interface{}:
		return m.convertToEntries(v)
	case []TaskHistoryEntry:
		return v, nil
	default:
		// Try to marshal/unmarshal to handle map[string]interface{} from JSON
		data, err := jsonMarshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal history data: %w", err)
		}

		var entries []TaskHistoryEntry
		if err := jsonUnmarshal(data, &entries); err != nil {
			return nil, fmt.Errorf("failed to unmarshal history entries: %w", err)
		}
		return entries, nil
	}
}

// Helper functions for JSON operations
func jsonMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func jsonUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Note: truncateString and getTaskHistoryPath are defined in welcome.go
// and are reused here for consistency across the TUI package.

// convertToEntries converts []interface{} to []TaskHistoryEntry.
func (m *HistoryViewModel) convertToEntries(data []interface{}) ([]TaskHistoryEntry, error) {
	entries := make([]TaskHistoryEntry, 0, len(data))

	for i, item := range data {
		entry, err := m.convertToEntry(item)
		if err != nil {
			return nil, fmt.Errorf("invalid entry at index %d: %w", i, err)
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// convertToEntry converts a single interface{} to TaskHistoryEntry.
func (m *HistoryViewModel) convertToEntry(data interface{}) (TaskHistoryEntry, error) {
	// Use a map-based approach for flexibility
	entryMap, ok := data.(map[string]interface{})
	if !ok {
		return TaskHistoryEntry{}, fmt.Errorf("expected map, got %T", data)
	}

	entry := TaskHistoryEntry{}

	// Extract ID
	if idVal, ok := entryMap["id"]; ok {
		if id, ok := idVal.(string); ok {
			entry.ID = id
		}
	}

	// Extract Task
	if taskVal, ok := entryMap["task"]; ok {
		if task, ok := taskVal.(string); ok {
			entry.Task = task
		}
	}

	// Extract Timestamp
	if tsVal, ok := entryMap["ts"]; ok {
		switch ts := tsVal.(type) {
		case float64:
			entry.Timestamp = int64(ts)
		case int64:
			entry.Timestamp = ts
		case int:
			entry.Timestamp = int64(ts)
		}
	}

	// Extract Metadata
	entry.Metadata = m.extractMetadata(entryMap)

	return entry, nil
}

// extractMetadata extracts metadata from the entry map.
func (m *HistoryViewModel) extractMetadata(entryMap map[string]interface{}) Metadata {
	meta := Metadata{}

	if metaVal, ok := entryMap["metadata"]; ok {
		if metaMap, ok := metaVal.(map[string]interface{}); ok {
			// Extract Model
			if modelVal, ok := metaMap["model"]; ok {
				if model, ok := modelVal.(string); ok {
					meta.Model = model
				}
			}

			// Extract Mode
			if modeVal, ok := metaMap["mode"]; ok {
				if mode, ok := modeVal.(string); ok {
					meta.Mode = mode
				}
			}

			// Extract Completed
			if completedVal, ok := metaMap["completed"]; ok {
				if completed, ok := completedVal.(bool); ok {
					meta.Completed = completed
				}
			}

			// Extract TotalCost
			if costVal, ok := metaMap["totalCost"]; ok {
				switch cost := costVal.(type) {
				case float64:
					meta.TotalCost = cost
				case float32:
					meta.TotalCost = float64(cost)
				case int:
					meta.TotalCost = float64(cost)
				case int64:
					meta.TotalCost = float64(cost)
				}
			}
		}
	}

	// Also check top-level fields for backward compatibility
	if modelVal, ok := entryMap["modelId"]; ok {
		if model, ok := modelVal.(string); ok {
			meta.Model = model
		}
	}

	if costVal, ok := entryMap["totalCost"]; ok {
		switch cost := costVal.(type) {
		case float64:
			meta.TotalCost = cost
		case float32:
			meta.TotalCost = float64(cost)
		case int:
			meta.TotalCost = float64(cost)
		case int64:
			meta.TotalCost = float64(cost)
		}
	}

	return meta
}

// filterEntries filters entries based on the search query.
func (m *HistoryViewModel) filterEntries() {
	if m.searchQuery == "" {
		m.filteredEntries = make([]TaskHistoryEntry, len(m.allEntries))
		copy(m.filteredEntries, m.allEntries)
		return
	}

	query := strings.ToLower(m.searchQuery)
	m.filteredEntries = make([]TaskHistoryEntry, 0)

	for _, entry := range m.allEntries {
		// Search in task description
		if strings.Contains(strings.ToLower(entry.Task), query) {
			m.filteredEntries = append(m.filteredEntries, entry)
			continue
		}

		// Search in task ID
		if strings.Contains(strings.ToLower(entry.ID), query) {
			m.filteredEntries = append(m.filteredEntries, entry)
			continue
		}

		// Search in model
		if strings.Contains(strings.ToLower(entry.Metadata.Model), query) {
			m.filteredEntries = append(m.filteredEntries, entry)
			continue
		}
	}
}

// calculatePagination recalculates pagination based on filtered entries.
func (m *HistoryViewModel) calculatePagination() {
	totalEntries := len(m.filteredEntries)
	if totalEntries == 0 {
		m.totalPages = 1
		m.currentPage = 1
		return
	}

	m.totalPages = (totalEntries + m.config.Limit - 1) / m.config.Limit
	if m.totalPages == 0 {
		m.totalPages = 1
	}

	// Ensure current page is valid
	if m.currentPage > m.totalPages {
		m.currentPage = m.totalPages
	}
	if m.currentPage < 1 {
		m.currentPage = 1
	}

	// Reset selected index if out of bounds
	pageEntries := m.getCurrentPageEntries()
	if m.selectedIndex >= len(pageEntries) {
		m.selectedIndex = len(pageEntries) - 1
	}
	if m.selectedIndex < 0 {
		m.selectedIndex = 0
	}
}

// getCurrentPageEntries returns the entries for the current page.
func (m *HistoryViewModel) getCurrentPageEntries() []TaskHistoryEntry {
	start := (m.currentPage - 1) * m.config.Limit
	end := start + m.config.Limit

	if start >= len(m.filteredEntries) {
		return []TaskHistoryEntry{}
	}
	if end > len(m.filteredEntries) {
		end = len(m.filteredEntries)
	}

	return m.filteredEntries[start:end]
}

// Update implements tea.Model.
func (m *HistoryViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case historyDataLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = HistoryViewStateExited
			return m, tea.Quit
		}

		m.allEntries = msg.entries
		m.filterEntries()
		m.calculatePagination()
		m.state = HistoryViewStateDisplaying

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	// Update search input if searching
	if m.isSearching {
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// handleKeyMsg handles keyboard input.
func (m *HistoryViewModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle quit first (works in all states except confirmation)
	if m.state != HistoryViewStateConfirmDelete && key.Matches(msg, m.keyMap.Quit) {
		m.state = HistoryViewStateExited
		return m, tea.Quit
	}

	switch m.state {
	case HistoryViewStateDisplaying:
		return m.handleDisplayingKeys(msg)

	case HistoryViewStateSearching:
		return m.handleSearchingKeys(msg)

	case HistoryViewStateConfirmDelete:
		return m.handleConfirmDeleteKeys(msg)
	}

	return m, nil
}

// handleDisplayingKeys handles keys when displaying the history list.
func (m *HistoryViewModel) handleDisplayingKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	pageEntries := m.getCurrentPageEntries()

	switch {
	case key.Matches(msg, m.keyMap.Up):
		if m.selectedIndex > 0 {
			m.selectedIndex--
		} else if m.currentPage > 1 {
			// Go to previous page and select last item
			m.currentPage--
			pageEntries = m.getCurrentPageEntries()
			m.selectedIndex = len(pageEntries) - 1
		}

	case key.Matches(msg, m.keyMap.Down):
		if m.selectedIndex < len(pageEntries)-1 {
			m.selectedIndex++
		} else if m.currentPage < m.totalPages {
			// Go to next page and select first item
			m.currentPage++
			m.selectedIndex = 0
		}

	case key.Matches(msg, m.keyMap.PageUp):
		if m.currentPage > 1 {
			m.currentPage--
			m.selectedIndex = 0
		}

	case key.Matches(msg, m.keyMap.PageDown):
		if m.currentPage < m.totalPages {
			m.currentPage++
			m.selectedIndex = 0
		}

	case key.Matches(msg, m.keyMap.PrevPage):
		if m.currentPage > 1 {
			m.currentPage--
			m.selectedIndex = 0
		}

	case key.Matches(msg, m.keyMap.NextPage):
		if m.currentPage < m.totalPages {
			m.currentPage++
			m.selectedIndex = 0
		}

	case key.Matches(msg, m.keyMap.Select):
		if len(pageEntries) > 0 && m.selectedIndex < len(pageEntries) {
			selected := pageEntries[m.selectedIndex]
			m.state = HistoryViewStateTaskSelected
			if m.config.OnSelectTask != nil {
				m.config.OnSelectTask(selected.ID)
			}
			return m, tea.Quit
		}

	case key.Matches(msg, m.keyMap.Search):
		m.state = HistoryViewStateSearching
		m.isSearching = true
		m.searchInput.Focus()
		return m, textinput.Blink

	case key.Matches(msg, m.keyMap.Delete):
		if len(pageEntries) > 0 && m.selectedIndex < len(pageEntries) {
			m.deleteCandidate = &pageEntries[m.selectedIndex]
			m.state = HistoryViewStateConfirmDelete
		}

	case key.Matches(msg, m.keyMap.Cancel):
		m.state = HistoryViewStateExited
		return m, tea.Quit
	}

	return m, nil
}

// handleSearchingKeys handles keys when in search mode.
func (m *HistoryViewModel) handleSearchingKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keyMap.Cancel):
		m.state = HistoryViewStateDisplaying
		m.isSearching = false
		m.searchInput.SetValue("")
		m.searchQuery = ""
		m.filterEntries()
		m.calculatePagination()

	case key.Matches(msg, m.keyMap.Select):
		m.searchQuery = m.searchInput.Value()
		m.state = HistoryViewStateDisplaying
		m.isSearching = false
		m.filterEntries()
		m.calculatePagination()
		m.selectedIndex = 0
		m.currentPage = 1

	default:
		// Let the search input handle other keys
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleConfirmDeleteKeys handles keys when confirming deletion.
func (m *HistoryViewModel) handleConfirmDeleteKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keyMap.ConfirmYes):
		if m.deleteCandidate != nil {
			if m.config.OnDeleteTask != nil {
				if err := m.config.OnDeleteTask(m.deleteCandidate.ID); err != nil {
					m.err = err
				}
			}
			// Remove from local list
			m.removeEntry(m.deleteCandidate.ID)
		}
		m.deleteCandidate = nil
		m.state = HistoryViewStateDisplaying
		m.calculatePagination()

	case key.Matches(msg, m.keyMap.ConfirmNo), key.Matches(msg, m.keyMap.Cancel):
		m.deleteCandidate = nil
		m.state = HistoryViewStateDisplaying
	}

	return m, nil
}

// removeEntry removes an entry from the local lists.
func (m *HistoryViewModel) removeEntry(id string) {
	// Remove from allEntries
	for i, entry := range m.allEntries {
		if entry.ID == id {
			m.allEntries = append(m.allEntries[:i], m.allEntries[i+1:]...)
			break
		}
	}

	// Remove from filteredEntries
	for i, entry := range m.filteredEntries {
		if entry.ID == id {
			m.filteredEntries = append(m.filteredEntries[:i], m.filteredEntries[i+1:]...)
			break
		}
	}
}

// View implements tea.Model.
func (m *HistoryViewModel) View() string {
	switch m.state {
	case HistoryViewStateLoading:
		return m.renderLoading()

	case HistoryViewStateDisplaying:
		return m.renderDisplaying()

	case HistoryViewStateSearching:
		return m.renderSearching()

	case HistoryViewStateConfirmDelete:
		return m.renderConfirmDelete()

	default:
		return ""
	}
}

// renderLoading renders the loading state.
func (m *HistoryViewModel) renderLoading() string {
	return "Loading task history..."
}

// renderDisplaying renders the history list.
func (m *HistoryViewModel) renderDisplaying() string {
	var sections []string

	// Header
	sections = append(sections, m.renderHeader())

	// Help text
	sections = append(sections, m.renderHelp())

	// Pagination info
	if m.totalPages > 1 {
		sections = append(sections, m.renderPagination())
	}

	// Separator
	sections = append(sections, m.renderSeparator())

	// Task list
	sections = append(sections, m.renderTaskList())

	// Bottom separator
	sections = append(sections, m.renderSeparator())

	return strings.Join(sections, "\n")
}

// renderHeader renders the header section.
func (m *HistoryViewModel) renderHeader() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF"))

	totalCount := len(m.allEntries)
	filteredCount := len(m.filteredEntries)

	if m.searchQuery != "" {
		return titleStyle.Render(fmt.Sprintf("📜 Task History (%d of %d matches for '%s')", filteredCount, totalCount, m.searchQuery))
	}

	return titleStyle.Render(fmt.Sprintf("📜 Task History (%d total)", totalCount))
}

// renderHelp renders the help text.
func (m *HistoryViewModel) renderHelp() string {
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080"))

	return helpStyle.Render("Use ↑↓/j/k to navigate, Enter to select, / to search, d to delete, q to quit")
}

// renderPagination renders pagination info.
func (m *HistoryViewModel) renderPagination() string {
	var parts []string

	pageStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080"))

	navStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4"))

	disabledStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#404040"))

	parts = append(parts, pageStyle.Render(fmt.Sprintf("Page %d of %d", m.currentPage, m.totalPages)))

	if m.currentPage > 1 {
		parts = append(parts, navStyle.Render("[← prev]"))
	} else {
		parts = append(parts, disabledStyle.Render("[← prev]"))
	}

	if m.currentPage < m.totalPages {
		parts = append(parts, navStyle.Render("[next →]"))
	} else {
		parts = append(parts, disabledStyle.Render("[next →]"))
	}

	return strings.Join(parts, " ")
}

// renderSeparator renders a horizontal separator line.
func (m *HistoryViewModel) renderSeparator() string {
	sepStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#404040"))

	return sepStyle.Render(strings.Repeat("─", m.width-2))
}

// renderTaskList renders the list of tasks.
func (m *HistoryViewModel) renderTaskList() string {
	pageEntries := m.getCurrentPageEntries()

	if len(pageEntries) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")).
			Italic(true)

		if m.searchQuery != "" {
			return emptyStyle.Render("No tasks match your search.")
		}

		return emptyStyle.Render("No task history available.")
	}

	var lines []string

	// Calculate visible window around selected item
	visibleCount := m.calculateVisibleCount()
	halfVisible := visibleCount / 2

	startIndex := m.selectedIndex - halfVisible
	if startIndex < 0 {
		startIndex = 0
	}

	endIndex := startIndex + visibleCount
	if endIndex > len(pageEntries) {
		endIndex = len(pageEntries)
		startIndex = endIndex - visibleCount
		if startIndex < 0 {
			startIndex = 0
		}
	}

	// Show up indicator if there are more items above
	if startIndex > 0 {
		indicatorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080"))
		lines = append(lines, indicatorStyle.Render(fmt.Sprintf("  ↑ %d more above", startIndex)))
	}

	// Render visible tasks
	for i := startIndex; i < endIndex && i < len(pageEntries); i++ {
		entry := pageEntries[i]
		isSelected := i == m.selectedIndex
		lines = append(lines, m.renderTaskEntry(entry, isSelected))
	}

	// Show down indicator if there are more items below
	if endIndex < len(pageEntries) {
		indicatorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080"))
		lines = append(lines, indicatorStyle.Render(fmt.Sprintf("  ↓ %d more below", len(pageEntries)-endIndex)))
	}

	return strings.Join(lines, "\n")
}

// calculateVisibleCount calculates how many tasks can be visible.
func (m *HistoryViewModel) calculateVisibleCount() int {
	// Reserve lines for header, help, pagination, separators
	headerLines := 4 // header, help, separator
	if m.totalPages > 1 {
		headerLines++ // pagination line
	}
	footerLines := 1 // bottom separator

	availableHeight := m.height - headerLines - footerLines
	if availableHeight < 1 {
		return 1
	}

	// Each task takes approximately 5 lines
	itemHeight := 5
	return max(1, availableHeight/itemHeight)
}

// renderTaskEntry renders a single task entry.
func (m *HistoryViewModel) renderTaskEntry(entry TaskHistoryEntry, isSelected bool) string {
	var parts []string

	// Selection indicator and date
	indicator := "  "
	dateColor := lipgloss.Color("#808080")

	if isSelected {
		indicator = "> "
		dateColor = lipgloss.Color("#4CAF50") // Green for selected
	}

	indicatorStyle := lipgloss.NewStyle().
		Foreground(dateColor)

	dateStr := formatTimestamp(entry.Timestamp)
	parts = append(parts, indicatorStyle.Render(indicator+dateStr))

	// Task ID
	idStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00BCD4")). // Cyan
		MarginLeft(4)

	if isSelected {
		idStyle = idStyle.Bold(true)
	}

	parts = append(parts, idStyle.Render(entry.ID))

	// Task description (truncated)
	taskStyle := lipgloss.NewStyle().
		MarginLeft(4)

	if isSelected {
		taskStyle = taskStyle.
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF"))
	} else {
		taskStyle = taskStyle.Foreground(lipgloss.Color("#E0E0E0"))
	}

	taskText := truncateString(entry.Task, m.width-10)
	if len(entry.Task) > m.width-10 {
		taskText += "..."
	}

	parts = append(parts, taskStyle.Render(taskText))

	// Cost (if available)
	if entry.Metadata.TotalCost > 0 {
		costStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")).
			MarginLeft(4)

		parts = append(parts, costStyle.Render(fmt.Sprintf("Cost: $%.4f", entry.Metadata.TotalCost)))
	}

	// Model (if available)
	if entry.Metadata.Model != "" {
		modelStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")).
			MarginLeft(4)

		parts = append(parts, modelStyle.Render(fmt.Sprintf("Model: %s", entry.Metadata.Model)))
	}

	return strings.Join(parts, "\n")
}

// renderSearching renders the search input view.
func (m *HistoryViewModel) renderSearching() string {
	var sections []string

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF"))

	sections = append(sections, titleStyle.Render("🔍 Search Tasks"))

	// Help
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080"))

	sections = append(sections, helpStyle.Render("Type to filter tasks, Enter to confirm, Esc to cancel"))

	// Separator
	sepStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#404040"))

	sections = append(sections, sepStyle.Render(strings.Repeat("─", m.width-2)))

	// Search input
	sections = append(sections, m.searchInput.View())

	return strings.Join(sections, "\n")
}

// renderConfirmDelete renders the delete confirmation dialog.
func (m *HistoryViewModel) renderConfirmDelete() string {
	var sections []string

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFC107")) // Warning yellow

	sections = append(sections, titleStyle.Render("⚠️  Confirm Deletion"))

	// Separator
	sepStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#404040"))

	sections = append(sections, sepStyle.Render(strings.Repeat("─", m.width-2)))

	// Confirmation message
	if m.deleteCandidate != nil {
		msgStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0E0E0"))

		taskPreview := truncateString(m.deleteCandidate.Task, 50)
		sections = append(sections, msgStyle.Render(fmt.Sprintf("Delete task: %s", taskPreview)))
		sections = append(sections, msgStyle.Render(fmt.Sprintf("ID: %s", m.deleteCandidate.ID)))
	}

	// Prompt
	promptStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF"))

	sections = append(sections, "")
	sections = append(sections, promptStyle.Render("Are you sure? (y/n)"))

	return strings.Join(sections, "\n")
}

// Result returns the history view result.
func (m *HistoryViewModel) Result() HistoryViewResult {
	selectedTask := ""
	if m.state == HistoryViewStateTaskSelected {
		pageEntries := m.getCurrentPageEntries()
		if m.selectedIndex < len(pageEntries) {
			selectedTask = pageEntries[m.selectedIndex].ID
		}
	}

	return HistoryViewResult{
		State:        m.state,
		SelectedTask: selectedTask,
		WasDeleted:   m.deleteCandidate != nil,
	}
}

// Run executes the history view model and returns the result.
func (m *HistoryViewModel) Run() (HistoryViewResult, error) {
	p := tea.NewProgram(m)
	model, err := p.Run()
	if err != nil {
		return HistoryViewResult{}, err
	}

	hvm, ok := model.(*HistoryViewModel)
	if !ok {
		return HistoryViewResult{}, fmt.Errorf("unexpected model type")
	}

	return hvm.Result(), nil
}

// formatTimestamp formats a Unix timestamp to a human-readable string.
func formatTimestamp(ts int64) string {
	if ts == 0 {
		return "unknown"
	}

	// Handle millisecond timestamps
	if ts > 1e12 {
		ts = ts / 1000
	}

	t := time.Unix(ts, 0)
	return t.Format("2006-01-02 15:04:05")
}

// ShowHistoryView displays the history view and returns the result.
// This is a convenience function for simple use cases.
func ShowHistoryView(storageCtx *storage.StorageContext, onSelectTask func(taskID string)) (HistoryViewResult, error) {
	config := DefaultHistoryViewConfig()
	config.StorageContext = storageCtx
	config.OnSelectTask = onSelectTask

	model := NewHistoryViewModel(config)
	return model.Run()
}