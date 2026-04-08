// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SearchableListItem represents an item in a searchable list.
type SearchableListItem struct {
	ID          string
	Title       string
	Description string
	Selected    bool
	Metadata    map[string]string
}

// FilterValue returns the value to filter on for list.Item interface.
func (i SearchableListItem) FilterValue() string {
	return i.Title
}

// SearchableListModel is a reusable searchable list component.
type SearchableListModel struct {
	width  int
	height int
	list   list.Model
	items  []SearchableListItem
	filter string
	styles SearchableListStyles

	// State
	ready    bool
	done     bool
	selected *SearchableListItem
	showHelp bool
}

// SearchableListStyles holds styling for the searchable list.
type SearchableListStyles struct {
	TitleStyle       lipgloss.Style
	SelectedStyle    lipgloss.Style
	FilterStyle      lipgloss.Style
	HelpStyle        lipgloss.Style
	DescriptionStyle lipgloss.Style
}

// DefaultSearchableListStyles returns default styles.
func DefaultSearchableListStyles() SearchableListStyles {
	return SearchableListStyles{
		TitleStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")),

		SelectedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00D9FF")).
			Background(lipgloss.Color("#1a1a1a")),

		FilterStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD700")),

		HelpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#606060")).
			Italic(true),

		DescriptionStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")),
	}
}

// NewSearchableListModel creates a new searchable list model.
func NewSearchableListModel(title string, items []SearchableListItem) SearchableListModel {
	// Convert items to list items
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}

	// Create list with custom delegate
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true

	l := list.New(listItems, delegate, 0, 0)
	l.Title = title
	l.SetShowHelp(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4"))

	return SearchableListModel{
		items:    items,
		list:     l,
		styles:   DefaultSearchableListStyles(),
		showHelp: true,
	}
}

// SetDimensions sets the terminal dimensions.
func (m *SearchableListModel) SetDimensions(width, height int) {
	m.width = width
	m.height = height
	m.list.SetSize(width, height-4) // Reserve space for header and help
}

// Init initializes the model.
func (m SearchableListModel) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (m SearchableListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetDimensions(msg.Width, msg.Height)
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		// Handle custom keys before passing to list
		switch msg.String() {
		case "q", "ctrl+c":
			m.done = true
			return m, tea.Quit

		case "enter":
			if item, ok := m.list.SelectedItem().(SearchableListItem); ok {
				m.selected = &item
				m.done = true
				return m, tea.Quit
			}

		case "esc":
			if m.list.FilterState() == list.Filtering {
				// Let list handle canceling filter
			} else {
				m.done = true
				return m, tea.Quit
			}

		case "?":
			m.showHelp = !m.showHelp
			return m, nil
		}
	}

	// Update list
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the searchable list.
func (m SearchableListModel) View() string {
	if !m.ready {
		return "Loading..."
	}

	var content strings.Builder

	// List
	content.WriteString(m.list.View())
	content.WriteString("\n")

	// Help
	if m.showHelp {
		help := m.styles.HelpStyle.Render(
			"↑↓: navigate • /: filter • enter: select • ?: toggle help • q: quit")
		content.WriteString(help)
	} else {
		help := m.styles.HelpStyle.Render("?: help")
		content.WriteString(help)
	}

	return content.String()
}

// IsDone returns true if selection is complete.
func (m SearchableListModel) IsDone() bool {
	return m.done
}

// GetSelected returns the selected item.
func (m SearchableListModel) GetSelected() *SearchableListItem {
	return m.selected
}

// SetItems updates the list items.
func (m *SearchableListModel) SetItems(items []SearchableListItem) {
	m.items = items
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}
	m.list.SetItems(listItems)
}

// GetFilter returns the current filter text.
func (m *SearchableListModel) GetFilter() string {
	return m.list.FilterValue()
}

// ShowSearchableList displays a searchable list and returns the selected item.
func ShowSearchableList(title string, items []SearchableListItem) (*SearchableListItem, error) {
	model := NewSearchableListModel(title, items)

	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	listModel, ok := finalModel.(SearchableListModel)
	if !ok {
		return nil, fmt.Errorf("unexpected model type")
	}

	return listModel.GetSelected(), nil
}

// SearchableListResult represents the result of a searchable list selection.
type SearchableListResult struct {
	Selected *SearchableListItem
	Canceled bool
	Error    error
}

// ShowSearchableListWithCancel displays a searchable list with cancel option.
func ShowSearchableListWithCancel(title string, items []SearchableListItem, showCancel bool) SearchableListResult {
	model := NewSearchableListModel(title, items)

	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return SearchableListResult{Error: err}
	}

	listModel, ok := finalModel.(SearchableListModel)
	if !ok {
		return SearchableListResult{Error: fmt.Errorf("unexpected model type")}
	}

	return SearchableListResult{
		Selected: listModel.GetSelected(),
		Canceled: listModel.GetSelected() == nil,
	}
}

// ConvertStringsToSearchableItems converts a slice of strings to searchable items.
func ConvertStringsToSearchableItems(items []string) []SearchableListItem {
	result := make([]SearchableListItem, len(items))
	for i, item := range items {
		result[i] = SearchableListItem{
			ID:    fmt.Sprintf("item-%d", i),
			Title: item,
		}
	}
	return result
}

// ConvertMapToSearchableItems converts a map to searchable items with titles and descriptions.
func ConvertMapToSearchableItems(items map[string]string) []SearchableListItem {
	result := make([]SearchableListItem, 0, len(items))
	for id, value := range items {
		parts := strings.SplitN(value, "|", 2)
		item := SearchableListItem{
			ID:    id,
			Title: parts[0],
		}
		if len(parts) > 1 {
			item.Description = parts[1]
		}
		result = append(result, item)
	}
	return result
}

// FilterItems filters items based on a search query.
func FilterItems(items []SearchableListItem, query string) []SearchableListItem {
	if query == "" {
		return items
	}

	query = strings.ToLower(query)
	var result []SearchableListItem

	for _, item := range items {
		titleMatch := strings.Contains(strings.ToLower(item.Title), query)
		descMatch := strings.Contains(strings.ToLower(item.Description), query)
		idMatch := strings.Contains(strings.ToLower(item.ID), query)

		if titleMatch || descMatch || idMatch {
			result = append(result, item)
		}
	}

	return result
}

// SortItemsByTitle sorts items alphabetically by title.
func SortItemsByTitle(items []SearchableListItem) []SearchableListItem {
	// Simple bubble sort for now - could be optimized
	result := make([]SearchableListItem, len(items))
	copy(result, items)

	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if strings.ToLower(result[i].Title) > strings.ToLower(result[j].Title) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}
