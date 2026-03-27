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
	cursor   int
	selected string
	goBack   bool
	styles   HistoryStyles
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
			Padding(2, 4),

		titleStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true).
			MarginBottom(1),

		itemStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0E0E0")).
			PaddingLeft(2),

		selectedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00D9FF")).
			Bold(true).
			Background(lipgloss.Color("#1a1a1a")).
			PaddingLeft(2),

		dimStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#606060")),

		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#606060")).
			MarginTop(1),
	}
}

// NewHistoryModel creates a new history model.
func NewHistoryModel() *HistoryModel {
	return &HistoryModel{
		items:  make([]HistoryItem, 0),
		cursor: 0,
		styles: DefaultHistoryStyles(),
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
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
			return m, nil

		case tea.KeyEnter:
			if m.cursor < len(m.items) {
				m.selected = m.items[m.cursor].ID
			}
			return m, nil

		case tea.KeyRunes:
			switch msg.String() {
			case "q", "Q":
				m.goBack = true
				return m, nil
			case "r", "R":
				m.Refresh()
				return m, nil
			}
		}
	}

	return m, nil
}

// View renders the history screen.
func (m HistoryModel) View() string {
	var content strings.Builder

	// Title
	content.WriteString(m.styles.titleStyle.Render("Task History"))
	content.WriteString("\n\n")

	if len(m.items) == 0 {
		content.WriteString(m.styles.dimStyle.Render("No task history found."))
		content.WriteString("\n")
		content.WriteString(m.styles.helpStyle.Render("Press Esc or q to go back"))
		return m.styles.containerStyle.Render(content.String())
	}

	// Calculate visible range
	maxVisible := m.height - 8
	startIdx := 0
	endIdx := len(m.items)

	if m.cursor >= maxVisible {
		startIdx = m.cursor - maxVisible + 1
		endIdx = startIdx + maxVisible
		if endIdx > len(m.items) {
			endIdx = len(m.items)
			startIdx = endIdx - maxVisible
			if startIdx < 0 {
				startIdx = 0
			}
		}
	}

	// Items
	for i := startIdx; i < endIdx && i < len(m.items); i++ {
		item := m.items[i]
		isSelected := i == m.cursor

		line := m.formatItem(item, isSelected)
		if isSelected {
			content.WriteString(m.styles.selectedStyle.Render(line))
		} else {
			content.WriteString(m.styles.itemStyle.Render(line))
		}
		content.WriteString("\n")
	}

	// Scroll indicator
	if len(m.items) > maxVisible {
		content.WriteString("\n")
		scrollInfo := fmt.Sprintf("Showing %d-%d of %d", startIdx+1, endIdx, len(m.items))
		content.WriteString(m.styles.dimStyle.Render(scrollInfo))
		content.WriteString("\n")
	}

	// Help
	content.WriteString("\n")
	content.WriteString(m.styles.helpStyle.Render("↑↓ to navigate • Enter to select • R to refresh • Esc/q to go back"))

	return m.styles.containerStyle.Render(content.String())
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
	// For now, use placeholder data
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
	m.cursor = 0
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
