// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cline/cline/golang-cli/internal/storage"
)

// HistoryItem represents a task history item.
type HistoryItem struct {
	ID        string
	Timestamp time.Time
	Task      string
	Model     string
	Cost      float64
	Tokens    int
}

// HistoryModel represents the history screen state.
type HistoryModel struct {
	width    int
	height   int
	items    []HistoryItem
	filtered []HistoryItem // Filtered items when search is active
	cursor   int
	selected string
	goBack   bool
	styles   HistoryStyles

	// Search functionality
	searchMode    bool
	searchQuery   string
	searchInput   string
	isSearching   bool

	// Pagination
	page       int
	pageSize   int
	totalPages int

	// Task preview
	previewMode   bool
	previewTask   *HistoryItem

	// Selection callback
	onSelectTask func(string)
}

// HistoryStyles holds styling for the history screen.
type HistoryStyles struct {
	containerStyle lipgloss.Style
	titleStyle     lipgloss.Style
	itemStyle      lipgloss.Style
	selectedStyle  lipgloss.Style
	dimStyle       lipgloss.Style
	helpStyle      lipgloss.Style
}

// DefaultHistoryStyles returns default history styles.
func DefaultHistoryStyles() HistoryStyles {
	return HistoryStyles{
		containerStyle: lipgloss.NewStyle().
			Padding(1, 2),

		titleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(PrimaryBlue)).
			Bold(true).
			MarginBottom(1),

		itemStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(White)).
			PaddingLeft(2),

		selectedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(SelectionBlue)).
			Bold(true).
			Background(lipgloss.Color(DarkBackground)).
			PaddingLeft(2),

		dimStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(Gray)),

		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(Gray)).
			MarginTop(1),
	}
}

// NewHistoryModel creates a new history model.
func NewHistoryModel() *HistoryModel {
	return &HistoryModel{
		items:      make([]HistoryItem, 0),
		filtered:   make([]HistoryItem, 0),
		cursor:     0,
		page:       1,
		pageSize:   10,
		styles:     DefaultHistoryStyles(),
		searchMode: false,
	}
}

// SetDimensions sets the terminal dimensions.
func (m *HistoryModel) SetDimensions(width, height int) {
	m.width = width
	m.height = height
}

// Init initializes the model.
func (m HistoryModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model.
func (m *HistoryModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle search mode input first
	if m.searchMode {
		return m.handleSearchInput(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			if m.previewMode {
				m.previewMode = false
				m.previewTask = nil
				return m, nil
			}
			m.goBack = true
			return m, nil

		case tea.KeyUp:
			m.moveUp()
			return m, nil

		case tea.KeyDown:
			m.moveDown()
			return m, nil

		case tea.KeyLeft:
			if m.page > 1 {
				m.page--
				m.cursor = 0
			}
			return m, nil

		case tea.KeyRight:
			if m.page < m.totalPages {
				m.page++
				m.cursor = 0
			}
			return m, nil

		case tea.KeyEnter:
			m.handleEnter()
			return m, nil

		case tea.KeyRunes:
			return m.handleRuneInput(msg.String())
		}
	}

	return m, nil
}

// handleSearchInput handles input when in search mode
func (m *HistoryModel) handleSearchInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			m.searchMode = false
			m.searchQuery = ""
			m.searchInput = ""
			m.applyFilter()
			return m, nil

		case tea.KeyEnter:
			m.searchMode = false
			m.searchQuery = m.searchInput
			m.applyFilter()
			return m, nil

		case tea.KeyBackspace:
			if len(m.searchInput) > 0 {
				m.searchInput = m.searchInput[:len(m.searchInput)-1]
			}
			return m, nil

		case tea.KeyRunes:
			m.searchInput += msg.String()
			return m, nil
		}
	}
	return m, nil
}

// handleRuneInput handles character input for shortcuts
func (m *HistoryModel) handleRuneInput(input string) (tea.Model, tea.Cmd) {
	switch input {
	case "q", "Q":
		m.goBack = true
		return m, nil
	case "r", "R":
		m.Refresh()
		return m, nil
	case "j":
		m.moveDown()
		return m, nil
	case "k":
		m.moveUp()
		return m, nil
	case "/", "?":
		m.searchMode = true
		m.searchInput = m.searchQuery
		return m, nil
	case "n":
		if m.page < m.totalPages {
			m.page++
			m.cursor = 0
		}
		return m, nil
	case "p":
		if m.page > 1 {
			m.page--
			m.cursor = 0
		}
		return m, nil
	case " ", "v":
		// Toggle preview mode
		m.togglePreview()
		return m, nil
	}
	return m, nil
}

// moveUp moves the cursor up
func (m *HistoryModel) moveUp() {
	if m.cursor > 0 {
		m.cursor--
	} else if m.page > 1 {
		m.page--
		items := m.getCurrentPageItems()
		m.cursor = len(items) - 1
	}
}

// moveDown moves the cursor down
func (m *HistoryModel) moveDown() {
	items := m.getCurrentPageItems()
	if m.cursor < len(items)-1 {
		m.cursor++
	} else if m.page < m.totalPages {
		m.page++
		m.cursor = 0
	}
}

// handleEnter handles enter key press
func (m *HistoryModel) handleEnter() {
	items := m.getCurrentPageItems()
	if m.cursor < len(items) {
		task := items[m.cursor]
		m.selected = task.ID
		if m.onSelectTask != nil {
			m.onSelectTask(task.ID)
		}
	}
}

// togglePreview toggles preview mode for the selected task
func (m *HistoryModel) togglePreview() {
	if m.previewMode {
		m.previewMode = false
		m.previewTask = nil
	} else {
		items := m.getCurrentPageItems()
		if m.cursor < len(items) {
			m.previewMode = true
			task := items[m.cursor]
			m.previewTask = &task
		}
	}
}

// applyFilter applies the search filter to items
func (m *HistoryModel) applyFilter() {
	if m.searchQuery == "" {
		m.filtered = m.items
	} else {
		query := strings.ToLower(m.searchQuery)
		m.filtered = make([]HistoryItem, 0)
		for _, item := range m.items {
			if strings.Contains(strings.ToLower(item.Task), query) ||
				strings.Contains(strings.ToLower(item.ID), query) ||
				strings.Contains(strings.ToLower(item.Model), query) {
				m.filtered = append(m.filtered, item)
			}
		}
	}
	m.page = 1
	m.cursor = 0
	m.calculateTotalPages()
}

// View renders the history screen.
func (m HistoryModel) View() string {
	var content strings.Builder

	// Title with count
	totalCount := len(m.items)
	filteredCount := len(m.filtered)
	title := "Task History"
	if filteredCount != totalCount {
		title = fmt.Sprintf("Task History (%d of %d)", filteredCount, totalCount)
	} else {
		title = fmt.Sprintf("Task History (%d total)", totalCount)
	}
	content.WriteString(m.styles.titleStyle.Render("📜 " + title))
	content.WriteString("\n")

	// Search mode indicator
	if m.searchMode {
		content.WriteString(m.styles.dimStyle.Render("Search: " + m.searchInput + "▌"))
		content.WriteString("\n")
	} else if m.searchQuery != "" {
		content.WriteString(m.styles.dimStyle.Render("Filter: " + m.searchQuery + " [/ to search]"))
		content.WriteString("\n")
	} else {
		content.WriteString(m.styles.dimStyle.Render("Use ↑↓/j/k to navigate, Enter to select, / to search"))
		content.WriteString("\n")
	}

	// Pagination info
	if m.totalPages > 1 {
		content.WriteString(m.styles.dimStyle.Render(fmt.Sprintf("Page %d of %d", m.page, m.totalPages)))
		if m.page > 1 {
			content.WriteString(m.styles.dimStyle.Render(" [←/p prev]"))
		}
		if m.page < m.totalPages {
			content.WriteString(m.styles.dimStyle.Render(" [next/n →]"))
		}
		content.WriteString("\n")
	}

	// Separator
	separatorWidth := m.width - 4
	if separatorWidth < 0 {
		separatorWidth = 0
	}
	content.WriteString(m.styles.dimStyle.Render(strings.Repeat("─", separatorWidth)))
	content.WriteString("\n")

	// Get current page items
	items := m.getCurrentPageItems()

	if len(items) == 0 {
		if m.searchQuery != "" {
			content.WriteString(m.styles.dimStyle.Render("No tasks match your search."))
		} else {
			content.WriteString(m.styles.dimStyle.Render("No task history found."))
		}
		content.WriteString("\n")
	} else {
		// Items
		for i, item := range items {
			isSelected := i == m.cursor
			line := m.formatItem(item, isSelected)

			if isSelected {
				content.WriteString(m.styles.selectedStyle.Render(line))
			} else {
				content.WriteString(m.styles.itemStyle.Render(line))
			}
			content.WriteString("\n")
		}
	}

	// Preview mode
	if m.previewMode && m.previewTask != nil {
		content.WriteString("\n")
		previewSepWidth := m.width - 4
		if previewSepWidth < 0 {
			previewSepWidth = 0
		}
		content.WriteString(m.styles.dimStyle.Render(strings.Repeat("─", previewSepWidth)))
		content.WriteString("\n")
		content.WriteString(m.renderPreview())
	}

	// Separator and help
	helpSepWidth := m.width - 4
	if helpSepWidth < 0 {
		helpSepWidth = 0
	}
	content.WriteString(m.styles.dimStyle.Render(strings.Repeat("─", helpSepWidth)))
	content.WriteString("\n")
	
	if m.previewMode {
		content.WriteString(m.styles.helpStyle.Render("Space/v: close preview • Enter: select task • Esc/q: back"))
	} else if m.searchMode {
		content.WriteString(m.styles.helpStyle.Render("Enter: search • Esc: cancel • Type to filter"))
	} else {
		content.WriteString(m.styles.helpStyle.Render("↑↓/j/k: navigate • /: search • Space/v: preview • Enter: select • Esc/q: back"))
	}

	return m.styles.containerStyle.Render(content.String())
}

// renderPreview renders the task preview
func (m *HistoryModel) renderPreview() string {
	if m.previewTask == nil {
		return ""
	}

	var content strings.Builder
	
	// Task preview header
	content.WriteString(m.styles.titleStyle.Render("Task Preview"))
	content.WriteString("\n\n")
	
	// Task details
	task := m.previewTask
	content.WriteString(fmt.Sprintf("ID: %s\n", task.ID))
	content.WriteString(fmt.Sprintf("Date: %s\n", task.Timestamp.Format("2006-01-02 15:04:05")))
	if task.Model != "" {
		content.WriteString(fmt.Sprintf("Model: %s\n", task.Model))
	}
	if task.Cost > 0 {
		content.WriteString(fmt.Sprintf("Cost: $%.4f\n", task.Cost))
	}
	content.WriteString(fmt.Sprintf("Tokens: %d\n", task.Tokens))
	content.WriteString("\n")
	
	// Task description
	content.WriteString("Task:\n")
	taskText := task.Task
	if len(taskText) > 200 {
		taskText = taskText[:200] + "..."
	}
	content.WriteString(taskText)

	return content.String()
}

// getCurrentPageItems returns items for the current page
func (m *HistoryModel) getCurrentPageItems() []HistoryItem {
	start := (m.page - 1) * m.pageSize
	end := start + m.pageSize
	if start > len(m.filtered) {
		return []HistoryItem{}
	}
	if end > len(m.filtered) {
		end = len(m.filtered)
	}
	return m.filtered[start:end]
}

// calculateTotalPages calculates the total number of pages
func (m *HistoryModel) calculateTotalPages() {
	if m.pageSize <= 0 {
		m.totalPages = 1
		return
	}
	m.totalPages = (len(m.filtered) + m.pageSize - 1) / m.pageSize
	if m.totalPages < 1 {
		m.totalPages = 1
	}
}

// formatItem formats a history item for display.
func (m *HistoryModel) formatItem(item HistoryItem, selected bool) string {
	var parts []string

	// ID (shortened)
	shortID := item.ID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	parts = append(parts, fmt.Sprintf("[%s]", shortID))

	// Task (truncated)
	task := item.Task
	if len(task) > 50 {
		task = task[:47] + "..."
	}
	parts = append(parts, task)

	// Model
	if item.Model != "" {
		parts = append(parts, fmt.Sprintf("(%s)", item.Model))
	}

	// Cost
	if item.Cost > 0 {
		parts = append(parts, fmt.Sprintf("$%.3f", item.Cost))
	}

	// Time
	if !item.Timestamp.IsZero() {
		parts = append(parts, m.formatTime(item.Timestamp))
	}

	return strings.Join(parts, " ")
}

// formatTime formats a timestamp relative to now.
func (m *HistoryModel) formatTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	case diff < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
	default:
		return t.Format("Jan 2")
	}
}

// Refresh reloads the history from storage.
func (m *HistoryModel) Refresh() {
	// This will be implemented to load from storage
	// For now, use placeholder data if no items
	if len(m.items) == 0 {
		m.items = []HistoryItem{
			{
				ID:        "task-001",
				Timestamp: time.Now().Add(-time.Hour),
				Task:      "Fix bug in authentication module",
				Model:     "claude-sonnet-4",
				Cost:      0.023,
				Tokens:    1250,
			},
			{
				ID:        "task-002",
				Timestamp: time.Now().Add(-2 * time.Hour),
				Task:      "Add unit tests for user service",
				Model:     "gpt-4o",
				Cost:      0.045,
				Tokens:    3200,
			},
		}
	}
	m.applyFilter()
}

// LoadFromStorage loads task history from the storage context.
func (m *HistoryModel) LoadFromStorage(storageCtx *storage.StorageContext) error {
	if storageCtx == nil {
		return fmt.Errorf("storage context is nil")
	}

	// Get task history from global state
	val, ok := storageCtx.GlobalState.Get("taskHistory")
	if !ok {
		// No history yet, that's okay
		m.items = []HistoryItem{}
		return nil
	}

	// Parse the task history
	var historyData []map[string]interface{}

	switch v := val.(type) {
	case []interface{}:
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				historyData = append(historyData, m)
			}
		}
	case []map[string]interface{}:
		historyData = v
	default:
		// Try to marshal/unmarshal
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal history data: %w", err)
		}
		if err := json.Unmarshal(data, &historyData); err != nil {
			return fmt.Errorf("failed to unmarshal history data: %w", err)
		}
	}

	// Convert to HistoryItem
	items := make([]HistoryItem, 0, len(historyData))
	for _, entry := range historyData {
		item := HistoryItem{
			ID:   getStringValue(entry["id"]),
			Task: getStringValue(entry["task"]),
		}

		// Parse timestamp
		if tsVal, ok := entry["ts"]; ok {
			switch ts := tsVal.(type) {
			case float64:
				item.Timestamp = time.Unix(int64(ts)/1000, 0)
			case int64:
				item.Timestamp = time.Unix(ts/1000, 0)
			case int:
				item.Timestamp = time.Unix(int64(ts)/1000, 0)
			}
		}

		// Parse model
		item.Model = getStringValue(entry["modelId"])

		// Parse cost
		if costVal, ok := entry["totalCost"]; ok {
			switch cost := costVal.(type) {
			case float64:
				item.Cost = cost
			case float32:
				item.Cost = float64(cost)
			case int:
				item.Cost = float64(cost)
			}
		}

		items = append(items, item)
	}

	// Sort by timestamp (newest first)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Timestamp.After(items[j].Timestamp)
	})

	m.items = items
	return nil
}

// SetStorageContext sets the storage context and loads history.
func (m *HistoryModel) SetStorageContext(storageCtx *storage.StorageContext) {
	if err := m.LoadFromStorage(storageCtx); err != nil {
		// Log error but don't fail - just show empty history
		m.items = []HistoryItem{}
	}
}

// SetItems sets the history items.
func (m *HistoryModel) SetItems(items []HistoryItem) {
	m.items = items
	m.filtered = items
	m.cursor = 0
	m.page = 1
	m.calculateTotalPages()
}

// SetOnSelectTask sets the callback for task selection
func (m *HistoryModel) SetOnSelectTask(callback func(string)) {
	m.onSelectTask = callback
}

// GetSelectedTaskID returns the selected task ID
func (m *HistoryModel) GetSelectedTaskID() string {
	return m.selected
}

// IsPreviewMode returns whether preview mode is active
func (m *HistoryModel) IsPreviewMode() bool {
	return m.previewMode
}

// ShouldGoBack returns true if the user wants to go back.
func (m *HistoryModel) ShouldGoBack() bool {
	return m.goBack
}

// GetSelectedTask returns the selected task ID.
func (m *HistoryModel) GetSelectedTask() string {
	return m.selected
}

// Reset resets the model state.
func (m *HistoryModel) Reset() {
	m.cursor = 0
	m.selected = ""
	m.goBack = false
}
