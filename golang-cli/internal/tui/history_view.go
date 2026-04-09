// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// HistoryView handles rendering of the history list.
type HistoryView struct {
	styles HistoryViewStyles
}

// HistoryViewStyles holds styles for the history view.
type HistoryViewStyles struct {
	containerStyle  lipgloss.Style
	titleStyle      lipgloss.Style
	itemStyle       lipgloss.Style
	selectedStyle   lipgloss.Style
	dimStyle        lipgloss.Style
	helpStyle       lipgloss.Style
	previewStyle    lipgloss.Style
	paginationStyle lipgloss.Style
	searchStyle     lipgloss.Style
	countStyle      lipgloss.Style
	separatorStyle  lipgloss.Style
}

// DefaultHistoryViewStyles returns default styles.
func DefaultHistoryViewStyles() HistoryViewStyles {
	return HistoryViewStyles{
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

		previewStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(White)).
			PaddingLeft(2).
			MarginTop(1),

		paginationStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(Gray)),

		searchStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(PrimaryBlue)).
			Bold(true),

		countStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(Gray)),

		separatorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(Gray)),
	}
}

// NewHistoryView creates a new history view.
func NewHistoryView() *HistoryView {
	return &HistoryView{
		styles: DefaultHistoryViewStyles(),
	}
}

// SetStyles sets custom styles for the view.
func (hv *HistoryView) SetStyles(styles HistoryViewStyles) {
	hv.styles = styles
}

// Render renders the history list.
func (hv *HistoryView) Render(items []HistoryItem, cursor int, page, pageSize, totalPages int, width int) string {
	var content strings.Builder

	// Title with count
	totalCount := len(items)
	title := fmt.Sprintf("Task History (%d total)", totalCount)
	content.WriteString(hv.styles.titleStyle.Render("📜 " + title))
	content.WriteString("\n\n")

	// Pagination info
	if totalPages > 1 {
		content.WriteString(hv.styles.paginationStyle.Render(fmt.Sprintf("Page %d of %d", page, totalPages)))
		content.WriteString("\n")
	}

	// Separator
	separatorWidth := width - 4
	if separatorWidth < 0 {
		separatorWidth = 0
	}
	content.WriteString(hv.styles.separatorStyle.Render(strings.Repeat("─", separatorWidth)))
	content.WriteString("\n")

	// Get current page items
	pageItems := hv.getPageItems(items, page, pageSize)

	if len(pageItems) == 0 {
		content.WriteString(hv.styles.dimStyle.Render("No task history found."))
		content.WriteString("\n")
	} else {
		// Items
		for i, item := range pageItems {
			isSelected := i == cursor
			line := hv.formatItem(item, isSelected, width)

			if isSelected {
				content.WriteString(hv.styles.selectedStyle.Render(line))
			} else {
				content.WriteString(hv.styles.itemStyle.Render(line))
			}
			content.WriteString("\n")
		}
	}

	// Help
	content.WriteString("\n")
	content.WriteString(hv.styles.helpStyle.Render("↑↓/j/k navigate • Enter select • / search • Esc/q back"))

	return hv.styles.containerStyle.Render(content.String())
}

// RenderWithSearch renders the history list with search mode active.
func (hv *HistoryView) RenderWithSearch(items []HistoryItem, cursor int, page, pageSize, totalPages int, width int, searchQuery string) string {
	var content strings.Builder

	// Title
	content.WriteString(hv.styles.titleStyle.Render("📜 Task History"))
	content.WriteString("\n")

	// Search indicator
	content.WriteString(hv.styles.searchStyle.Render("Search: " + searchQuery + "▌"))
	content.WriteString("\n\n")

	// Filtered count
	content.WriteString(hv.styles.countStyle.Render(fmt.Sprintf("%d results", len(items))))
	content.WriteString("\n")

	// Pagination
	if totalPages > 1 {
		content.WriteString(hv.styles.paginationStyle.Render(fmt.Sprintf("Page %d of %d", page, totalPages)))
		content.WriteString("\n")
	}

	// Separator
	separatorWidth := width - 4
	if separatorWidth < 0 {
		separatorWidth = 0
	}
	content.WriteString(hv.styles.separatorStyle.Render(strings.Repeat("─", separatorWidth)))
	content.WriteString("\n")

	// Get current page items
	pageItems := hv.getPageItems(items, page, pageSize)

	if len(pageItems) == 0 {
		content.WriteString(hv.styles.dimStyle.Render("No tasks match your search."))
		content.WriteString("\n")
	} else {
		// Items
		for i, item := range pageItems {
			isSelected := i == cursor
			line := hv.formatItem(item, isSelected, width)

			if isSelected {
				content.WriteString(hv.styles.selectedStyle.Render(line))
			} else {
				content.WriteString(hv.styles.itemStyle.Render(line))
			}
			content.WriteString("\n")
		}
	}

	// Help
	content.WriteString("\n")
	content.WriteString(hv.styles.helpStyle.Render("↑↓ navigate • Enter select • Esc cancel search"))

	return hv.styles.containerStyle.Render(content.String())
}

// RenderWithPreview renders the history list with a preview panel.
func (hv *HistoryView) RenderWithPreview(items []HistoryItem, cursor int, page, pageSize, totalPages int, width, height int, previewItem *HistoryItem) string {
	var content strings.Builder

	// Calculate layout
	leftWidth := width * 60 / 100
	rightWidth := width - leftWidth - 4

	// Get current page items
	pageItems := hv.getPageItems(items, page, pageSize)

	// Left panel - list
	var leftPanel strings.Builder
	leftPanel.WriteString(hv.styles.titleStyle.Render("📜 Task History"))
	leftPanel.WriteString("\n")

	if totalPages > 1 {
		leftPanel.WriteString(hv.styles.paginationStyle.Render(fmt.Sprintf("Page %d of %d", page, totalPages)))
		leftPanel.WriteString("\n")
	}

	// Separator
	sepWidth := leftWidth - 2
	if sepWidth < 0 {
		sepWidth = 0
	}
	leftPanel.WriteString(hv.styles.separatorStyle.Render(strings.Repeat("─", sepWidth)))
	leftPanel.WriteString("\n")

	// Items
	if len(pageItems) == 0 {
		leftPanel.WriteString(hv.styles.dimStyle.Render("No task history."))
		leftPanel.WriteString("\n")
	} else {
		for i, item := range pageItems {
			isSelected := i == cursor
			line := hv.formatItemCompact(item, isSelected)

			if isSelected {
				leftPanel.WriteString(hv.styles.selectedStyle.Render(line))
			} else {
				leftPanel.WriteString(hv.styles.itemStyle.Render(line))
			}
			leftPanel.WriteString("\n")
		}
	}

	// Right panel - preview
	var rightPanel strings.Builder
	if previewItem != nil {
		rightPanel.WriteString(hv.renderPreview(previewItem, rightWidth))
	} else {
		rightPanel.WriteString(hv.styles.dimStyle.Render("Select a task to preview"))
	}

	// Combine panels side by side
	leftContent := lipgloss.NewStyle().Width(leftWidth).Render(leftPanel.String())
	rightContent := lipgloss.NewStyle().Width(rightWidth).PaddingLeft(2).Render(rightPanel.String())

	content.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftContent, rightContent))

	// Help at bottom
	content.WriteString("\n\n")
	content.WriteString(hv.styles.helpStyle.Render("↑↓ navigate • Enter select • / search • Space preview • Esc/q back"))

	return hv.styles.containerStyle.Render(content.String())
}

// renderPreview renders the task preview panel.
func (hv *HistoryView) renderPreview(item *HistoryItem, width int) string {
	var content strings.Builder

	// Preview header
	content.WriteString(hv.styles.titleStyle.Render("Task Preview"))
	content.WriteString("\n\n")

	// Task details
	content.WriteString(fmt.Sprintf("ID: %s\n", item.ID))
	content.WriteString(fmt.Sprintf("Date: %s\n", item.Timestamp.Format("2006-01-02 15:04:05")))
	if item.Model != "" {
		content.WriteString(fmt.Sprintf("Model: %s\n", item.Model))
	}
	if item.Cost > 0 {
		content.WriteString(fmt.Sprintf("Cost: $%.4f\n", item.Cost))
	}
	if item.Tokens > 0 {
		content.WriteString(fmt.Sprintf("Tokens: %d\n", item.Tokens))
	}
	content.WriteString("\n")

	// Task description
	content.WriteString("Task:\n")
	taskText := item.Task
	if len(taskText) > width*3 {
		taskText = taskText[:width*3-3] + "..."
	}
	content.WriteString(hv.styles.previewStyle.Render(taskText))

	return content.String()
}

// formatItem formats a history item for display.
func (hv *HistoryView) formatItem(item HistoryItem, selected bool, width int) string {
	var parts []string

	// ID (shortened)
	shortID := item.ID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	parts = append(parts, fmt.Sprintf("[%s]", shortID))

	// Task (truncated)
	task := item.Task
	maxLen := width - 40
	if maxLen < 20 {
		maxLen = 20
	}
	if len(task) > maxLen {
		task = task[:maxLen-3] + "..."
	}
	parts = append(parts, task)

	// Model
	if item.Model != "" {
		model := item.Model
		if len(model) > 12 {
			model = model[:12]
		}
		parts = append(parts, fmt.Sprintf("(%s)", model))
	}

	return strings.Join(parts, " ")
}

// formatItemCompact formats a history item in compact mode.
func (hv *HistoryView) formatItemCompact(item HistoryItem, selected bool) string {
	var parts []string

	// ID (shortened)
	shortID := item.ID
	if len(shortID) > 6 {
		shortID = shortID[:6]
	}
	parts = append(parts, fmt.Sprintf("[%s]", shortID))

	// Task (very truncated)
	task := item.Task
	if len(task) > 30 {
		task = task[:27] + "..."
	}
	parts = append(parts, task)

	return strings.Join(parts, " ")
}

// getPageItems returns items for the current page.
func (hv *HistoryView) getPageItems(items []HistoryItem, page, pageSize int) []HistoryItem {
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(items) {
		return []HistoryItem{}
	}
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}

// RenderEmpty renders an empty state message.
func (hv *HistoryView) RenderEmpty(message string) string {
	var content strings.Builder

	content.WriteString(hv.styles.titleStyle.Render("📜 Task History"))
	content.WriteString("\n\n")
	content.WriteString(hv.styles.dimStyle.Render(message))
	content.WriteString("\n\n")
	content.WriteString(hv.styles.helpStyle.Render("Press any key to go back"))

	return hv.styles.containerStyle.Render(content.String())
}

// RenderLoading renders a loading state.
func (hv *HistoryView) RenderLoading() string {
	var content strings.Builder

	content.WriteString(hv.styles.titleStyle.Render("📜 Task History"))
	content.WriteString("\n\n")
	content.WriteString(hv.styles.dimStyle.Render("Loading..."))
	content.WriteString("\n")

	return hv.styles.containerStyle.Render(content.String())
}

// RenderError renders an error state.
func (hv *HistoryView) RenderError(err error) string {
	var content strings.Builder

	content.WriteString(hv.styles.titleStyle.Render("📜 Task History"))
	content.WriteString("\n\n")
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(ErrorRed)).Render("Error: " + err.Error()))
	content.WriteString("\n\n")
	content.WriteString(hv.styles.helpStyle.Render("Press any key to go back"))

	return hv.styles.containerStyle.Render(content.String())
}
