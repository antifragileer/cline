// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HistorySearch handles search functionality for task history.
type HistorySearch struct {
	query       string
	input       string
	isActive    bool
	isFiltering bool
	filterType  SearchFilterType
	dateFrom    time.Time
	dateTo      time.Time
	styles      HistorySearchStyles
	width       int
	height      int
}

// SearchFilterType represents different types of search filters.
type SearchFilterType string

const (
	// SearchFilterAll searches in all fields.
	SearchFilterAll SearchFilterType = "all"
	// SearchFilterTask searches in task content only.
	SearchFilterTask SearchFilterType = "task"
	// SearchFilterID searches in task ID only.
	SearchFilterID SearchFilterType = "id"
	// SearchFilterModel searches in model name only.
	SearchFilterModel SearchFilterType = "model"
	// SearchFilterDate searches by date range.
	SearchFilterDate SearchFilterType = "date"
)

// HistorySearchStyles holds styles for the search component.
type HistorySearchStyles struct {
	containerStyle    lipgloss.Style
	promptStyle       lipgloss.Style
	inputStyle        lipgloss.Style
	helpStyle         lipgloss.Style
	filterStyle       lipgloss.Style
	activeFilterStyle lipgloss.Style
	resultStyle       lipgloss.Style
}

// DefaultHistorySearchStyles returns default styles.
func DefaultHistorySearchStyles() HistorySearchStyles {
	return HistorySearchStyles{
		containerStyle: lipgloss.NewStyle().
			Padding(0, 1),

		promptStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(PrimaryBlue)).
			Bold(true),

		inputStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(White)).
			Background(lipgloss.Color(DarkBackground)),

		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(Gray)),

		filterStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(Gray)).
			Padding(0, 1),

		activeFilterStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(SelectionBlue)).
			Bold(true).
			Background(lipgloss.Color(DarkBackground)).
			Padding(0, 1),

		resultStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(Gray)),
	}
}

// NewHistorySearch creates a new history search component.
func NewHistorySearch() *HistorySearch {
	return &HistorySearch{
		styles:     DefaultHistorySearchStyles(),
		filterType: SearchFilterAll,
	}
}

// SetDimensions sets the terminal dimensions.
func (hs *HistorySearch) SetDimensions(width, height int) {
	hs.width = width
	hs.height = height
}

// SetStyles sets custom styles.
func (hs *HistorySearch) SetStyles(styles HistorySearchStyles) {
	hs.styles = styles
}

// Init initializes the search.
func (hs HistorySearch) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the search.
func (hs *HistorySearch) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle filter type cycling
		switch msg.Type {
		case tea.KeyEsc:
			if hs.isActive {
				hs.isActive = false
				hs.input = hs.query // Restore previous query
				return hs, nil
			}

		case tea.KeyEnter:
			if hs.isActive {
				hs.query = hs.input
				hs.isActive = false
				return hs, nil
			}

		case tea.KeyBackspace:
			if hs.isActive && len(hs.input) > 0 {
				hs.input = hs.input[:len(hs.input)-1]
			}
			return hs, nil

		case tea.KeyTab:
			hs.cycleFilterType()
			return hs, nil

		case tea.KeyRunes:
			if hs.isActive {
				hs.input += msg.String()
			}
			return hs, nil
		}
	}

	return hs, nil
}

// cycleFilterType cycles through available filter types.
func (hs *HistorySearch) cycleFilterType() {
	switch hs.filterType {
	case SearchFilterAll:
		hs.filterType = SearchFilterTask
	case SearchFilterTask:
		hs.filterType = SearchFilterID
	case SearchFilterID:
		hs.filterType = SearchFilterModel
	case SearchFilterModel:
		hs.filterType = SearchFilterDate
	case SearchFilterDate:
		hs.filterType = SearchFilterAll
	}
}

// View renders the search component.
func (hs HistorySearch) View() string {
	if !hs.isActive {
		if hs.query != "" {
			return hs.styles.resultStyle.Render(fmt.Sprintf("Filter: %s [%s]", hs.query, hs.filterType))
		}
		return ""
	}

	var content strings.Builder

	// Search prompt
	content.WriteString(hs.styles.promptStyle.Render("Search"))
	content.WriteString(" ")

	// Filter type indicator
	filterLabel := hs.getFilterLabel()
	content.WriteString(hs.styles.activeFilterStyle.Render(filterLabel))
	content.WriteString(" ")

	// Input
	content.WriteString(hs.styles.inputStyle.Render(hs.input + "▌"))

	return hs.styles.containerStyle.Render(content.String())
}

// getFilterLabel returns the display label for the current filter type.
func (hs *HistorySearch) getFilterLabel() string {
	switch hs.filterType {
	case SearchFilterAll:
		return "[all]"
	case SearchFilterTask:
		return "[task]"
	case SearchFilterID:
		return "[id]"
	case SearchFilterModel:
		return "[model]"
	case SearchFilterDate:
		return "[date]"
	default:
		return "[all]"
	}
}

// Activate enters search mode.
func (hs *HistorySearch) Activate() {
	hs.isActive = true
	hs.input = hs.query
}

// Deactivate exits search mode.
func (hs *HistorySearch) Deactivate() {
	hs.isActive = false
	hs.query = hs.input
}

// IsActive returns true if search mode is active.
func (hs *HistorySearch) IsActive() bool {
	return hs.isActive
}

// GetQuery returns the current search query.
func (hs *HistorySearch) GetQuery() string {
	return hs.query
}

// SetQuery sets the search query.
func (hs *HistorySearch) SetQuery(query string) {
	hs.query = query
	hs.input = query
}

// GetFilterType returns the current filter type.
func (hs *HistorySearch) GetFilterType() SearchFilterType {
	return hs.filterType
}

// SetFilterType sets the filter type.
func (hs *HistorySearch) SetFilterType(filterType SearchFilterType) {
	hs.filterType = filterType
}

// Reset clears the search.
func (hs *HistorySearch) Reset() {
	hs.query = ""
	hs.input = ""
	hs.isActive = false
	hs.filterType = SearchFilterAll
}

// HasQuery returns true if there's an active search query.
func (hs *HistorySearch) HasQuery() bool {
	return hs.query != ""
}

// FilterItems filters history items based on the search criteria.
func (hs *HistorySearch) FilterItems(items []HistoryItem) []HistoryItem {
	if hs.query == "" {
		return items
	}

	query := strings.ToLower(hs.query)
	filtered := make([]HistoryItem, 0)

	for _, item := range items {
		if hs.matches(item, query) {
			filtered = append(filtered, item)
		}
	}

	return filtered
}

// matches checks if a history item matches the search criteria.
func (hs *HistorySearch) matches(item HistoryItem, query string) bool {
	switch hs.filterType {
	case SearchFilterTask:
		return strings.Contains(strings.ToLower(item.Task), query)
	case SearchFilterID:
		return strings.Contains(strings.ToLower(item.ID), query)
	case SearchFilterModel:
		return strings.Contains(strings.ToLower(item.Model), query)
	case SearchFilterDate:
		// Search by date range
		return hs.matchesDate(item.Timestamp, query)
	default: // SearchFilterAll
		return strings.Contains(strings.ToLower(item.Task), query) ||
			strings.Contains(strings.ToLower(item.ID), query) ||
			strings.Contains(strings.ToLower(item.Model), query)
	}
}

// matchesDate checks if a timestamp matches date criteria.
func (hs *HistorySearch) matchesDate(t time.Time, query string) bool {
	// Try to parse query as date
	dateFormats := []string{
		"2006-01-02",
		"01-02",
		"2006-01",
		"01/02/2006",
		"01/02",
	}

	for _, format := range dateFormats {
		if searchDate, err := time.Parse(format, query); err == nil {
			// Compare year, month, day
			return t.Year() == searchDate.Year() &&
				t.Month() == searchDate.Month() &&
				t.Day() == searchDate.Day()
		}
	}

	// Try relative date keywords
	switch query {
	case "today":
		now := time.Now()
		return t.Year() == now.Year() && t.Month() == now.Month() && t.Day() == now.Day()
	case "yesterday":
		yesterday := time.Now().AddDate(0, 0, -1)
		return t.Year() == yesterday.Year() && t.Month() == yesterday.Month() && t.Day() == yesterday.Day()
	case "this week":
		now := time.Now()
		weekStart := now.AddDate(0, 0, -int(now.Weekday()))
		return t.After(weekStart) || t.Equal(weekStart)
	case "last week":
		now := time.Now()
		weekStart := now.AddDate(0, 0, -int(now.Weekday())-7)
		weekEnd := now.AddDate(0, 0, -int(now.Weekday()))
		return (t.After(weekStart) || t.Equal(weekStart)) && t.Before(weekEnd)
	case "this month":
		now := time.Now()
		return t.Year() == now.Year() && t.Month() == now.Month()
	case "last month":
		now := time.Now()
		lastMonth := now.AddDate(0, -1, 0)
		return t.Year() == lastMonth.Year() && t.Month() == lastMonth.Month()
	}

	return false
}

// Search performs a search and returns filtered results.
func (hs *HistorySearch) Search(items []HistoryItem, query string) []HistoryItem {
	hs.SetQuery(query)
	return hs.FilterItems(items)
}

// Clear clears the current search.
func (hs *HistorySearch) Clear() {
	hs.query = ""
	hs.input = ""
}

// GetHelpText returns help text for the search functionality.
func (hs *HistorySearch) GetHelpText() string {
	return "Tab: change filter • Enter: search • Esc: cancel"
}

// RenderResultsInfo renders information about search results.
func (hs *HistorySearch) RenderResultsInfo(total, filtered int) string {
	if !hs.HasQuery() {
		return fmt.Sprintf("%d tasks", total)
	}
	return fmt.Sprintf("%d of %d tasks", filtered, total)
}

// SetDateRange sets a date range filter.
func (hs *HistorySearch) SetDateRange(from, to time.Time) {
	hs.dateFrom = from
	hs.dateTo = to
	hs.filterType = SearchFilterDate
}

// ClearDateRange clears the date range filter.
func (hs *HistorySearch) ClearDateRange() {
	hs.dateFrom = time.Time{}
	hs.dateTo = time.Time{}
}

// HistorySearchMsg is sent when search is performed.
type HistorySearchMsg struct {
	Query      string
	FilterType SearchFilterType
	Results    []HistoryItem
}

// ToMsg converts the search state to a message.
func (hs *HistorySearch) ToMsg(results []HistoryItem) HistorySearchMsg {
	return HistorySearchMsg{
		Query:      hs.query,
		FilterType: hs.filterType,
		Results:    results,
	}
}
