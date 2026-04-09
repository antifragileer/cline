// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ApprovalHistoryItem represents a single approval decision in history
type ApprovalHistoryItem struct {
	ID          string
	Timestamp   time.Time
	AskType     string
	ToolName    string
	Description string
	Response    ApprovalResponse
	FilePath    string
	Command     string
}

// ApprovalHistory stores and displays approval decisions
type ApprovalHistory struct {
	items        []ApprovalHistoryItem
	maxItems     int
	styles       ApprovalHistoryStyles
	width        int
	height       int
	scrollOffset int
	showDetails  bool
	selectedIdx  int
}

// ApprovalHistoryStyles holds styling for approval history
type ApprovalHistoryStyles struct {
	containerStyle lipgloss.Style
	titleStyle     lipgloss.Style
	itemStyle      lipgloss.Style
	selectedStyle  lipgloss.Style
	approvedStyle  lipgloss.Style
	rejectedStyle  lipgloss.Style
	alwaysStyle    lipgloss.Style
	timestampStyle lipgloss.Style
	toolStyle      lipgloss.Style
	helpStyle      lipgloss.Style
}

// DefaultApprovalHistoryStyles returns default styles for approval history
func DefaultApprovalHistoryStyles() ApprovalHistoryStyles {
	return ApprovalHistoryStyles{
		containerStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1),

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

		approvedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(SuccessGreen)),

		rejectedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ErrorRed)),

		alwaysStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(PlanYellow)),

		timestampStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(Gray)).
			Italic(true),

		toolStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(PrimaryBlue)),

		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(Gray)).
			MarginTop(1),
	}
}

// NewApprovalHistory creates a new approval history
func NewApprovalHistory() *ApprovalHistory {
	return &ApprovalHistory{
		items:       make([]ApprovalHistoryItem, 0),
		maxItems:    100,
		styles:      DefaultApprovalHistoryStyles(),
		showDetails: false,
		selectedIdx: 0,
	}
}

// Add adds an approval decision to history
func (ah *ApprovalHistory) Add(item ApprovalHistoryItem) {
	// Add timestamp if not set
	if item.Timestamp.IsZero() {
		item.Timestamp = time.Now()
	}

	// Generate ID if not set
	if item.ID == "" {
		item.ID = fmt.Sprintf("approval-%d", time.Now().UnixNano())
	}

	// Add to beginning (newest first)
	ah.items = append([]ApprovalHistoryItem{item}, ah.items...)

	// Trim to max items
	if len(ah.items) > ah.maxItems {
		ah.items = ah.items[:ah.maxItems]
	}
}

// AddFromApproval adds an approval from an approval model
func (ah *ApprovalHistory) AddFromApproval(reqType ApprovalType, description string, response ApprovalResponse, details map[string]string) {
	item := ApprovalHistoryItem{
		AskType:     string(reqType),
		Description: description,
		Response:    response,
		Timestamp:   time.Now(),
	}

	// Extract additional details
	if path, ok := details["filePath"]; ok {
		item.FilePath = path
	}
	if cmd, ok := details["command"]; ok {
		item.Command = cmd
	}
	if tool, ok := details["toolName"]; ok {
		item.ToolName = tool
	}

	ah.Add(item)
}

// GetRecent returns the n most recent approval items
func (ah *ApprovalHistory) GetRecent(n int) []ApprovalHistoryItem {
	if n >= len(ah.items) {
		return ah.items
	}
	return ah.items[:n]
}

// GetByType returns all approvals of a specific type
func (ah *ApprovalHistory) GetByType(askType string) []ApprovalHistoryItem {
	var result []ApprovalHistoryItem
	for _, item := range ah.items {
		if item.AskType == askType {
			result = append(result, item)
		}
	}
	return result
}

// GetStats returns statistics about approval history
func (ah *ApprovalHistory) GetStats() (total, approved, rejected, always int) {
	total = len(ah.items)
	for _, item := range ah.items {
		switch item.Response {
		case ApprovalYes:
			approved++
		case ApprovalNo:
			rejected++
		case ApprovalAlways:
			always++
		}
	}
	return
}

// Clear clears all history
func (ah *ApprovalHistory) Clear() {
	ah.items = make([]ApprovalHistoryItem, 0)
	ah.selectedIdx = 0
	ah.scrollOffset = 0
}

// SetDimensions sets the display dimensions
func (ah *ApprovalHistory) SetDimensions(width, height int) {
	ah.width = width
	ah.height = height
}

// Update handles messages for the approval history view
func (ah *ApprovalHistory) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			ah.moveUp()
		case "down", "j":
			ah.moveDown()
		case "d":
			ah.showDetails = !ah.showDetails
		case "c":
			ah.Clear()
		}
	}
	return nil
}

// moveUp moves selection up
func (ah *ApprovalHistory) moveUp() {
	if ah.selectedIdx > 0 {
		ah.selectedIdx--
		if ah.selectedIdx < ah.scrollOffset {
			ah.scrollOffset = ah.selectedIdx
		}
	}
}

// moveDown moves selection down
func (ah *ApprovalHistory) moveDown() {
	if ah.selectedIdx < len(ah.items)-1 {
		ah.selectedIdx++
		maxVisible := ah.getMaxVisibleItems()
		if ah.selectedIdx >= ah.scrollOffset+maxVisible {
			ah.scrollOffset = ah.selectedIdx - maxVisible + 1
		}
	}
}

// getMaxVisibleItems returns the maximum number of items that can be displayed
func (ah *ApprovalHistory) getMaxVisibleItems() int {
	// Account for header, stats, help, and padding
	availableHeight := ah.height - 8
	if availableHeight < 3 {
		return 3
	}
	return availableHeight / 2 // Each item takes about 2 lines
}

// View renders the approval history
func (ah *ApprovalHistory) View() string {
	if len(ah.items) == 0 {
		return ah.styles.containerStyle.Render(
			ah.styles.titleStyle.Render("Approval History") + "\n\n" +
				lipgloss.NewStyle().Foreground(lipgloss.Color(Gray)).Render("No approvals yet") + "\n",
		)
	}

	var content strings.Builder

	// Title
	content.WriteString(ah.styles.titleStyle.Render("📋 Approval History"))
	content.WriteString("\n\n")

	// Stats
	total, approved, rejected, always := ah.GetStats()
	stats := fmt.Sprintf("Total: %d | ✅ %d | ❌ %d | ⭐ %d",
		total, approved, rejected, always)
	content.WriteString(ah.styles.timestampStyle.Render(stats))
	content.WriteString("\n\n")

	// Items
	maxVisible := ah.getMaxVisibleItems()
	endIdx := ah.scrollOffset + maxVisible
	if endIdx > len(ah.items) {
		endIdx = len(ah.items)
	}

	for i := ah.scrollOffset; i < endIdx; i++ {
		item := ah.items[i]
		line := ah.renderItem(item, i == ah.selectedIdx)
		content.WriteString(line)
		content.WriteString("\n")

		// Show details if selected and details mode is on
		if i == ah.selectedIdx && ah.showDetails {
			details := ah.renderItemDetails(item)
			if details != "" {
				content.WriteString(details)
				content.WriteString("\n")
			}
		}
	}

	// Scroll indicator
	if len(ah.items) > maxVisible {
		content.WriteString("\n")
		scrollInfo := fmt.Sprintf("Showing %d-%d of %d",
			ah.scrollOffset+1, endIdx, len(ah.items))
		content.WriteString(ah.styles.helpStyle.Render(scrollInfo))
		content.WriteString("\n")
	}

	// Help
	content.WriteString("\n")
	content.WriteString(ah.styles.helpStyle.Render(
		"↑↓/j/k: navigate • d: toggle details • c: clear • Esc: close"))

	return ah.styles.containerStyle.Render(content.String())
}

// renderItem renders a single approval history item
func (ah *ApprovalHistory) renderItem(item ApprovalHistoryItem, selected bool) string {
	var parts []string

	// Response indicator
	responseIndicator := ah.getResponseIndicator(item.Response)
	parts = append(parts, responseIndicator)

	// Tool/Command type
	actionType := ah.formatActionType(item)
	parts = append(parts, ah.styles.toolStyle.Render(actionType))

	// Description (truncated)
	desc := item.Description
	maxLen := ah.width - 30
	if len(desc) > maxLen {
		desc = desc[:maxLen-3] + "..."
	}
	parts = append(parts, desc)

	// Timestamp
	timeStr := ah.formatTimestamp(item.Timestamp)
	parts = append(parts, ah.styles.timestampStyle.Render(timeStr))

	line := strings.Join(parts, " ")

	if selected {
		return ah.styles.selectedStyle.Render(line)
	}
	return ah.styles.itemStyle.Render(line)
}

// renderItemDetails renders detailed information for an item
func (ah *ApprovalHistory) renderItemDetails(item ApprovalHistoryItem) string {
	var details []string

	if item.FilePath != "" {
		details = append(details, fmt.Sprintf("  File: %s", item.FilePath))
	}
	if item.Command != "" {
		details = append(details, fmt.Sprintf("  Command: %s", item.Command))
	}
	if item.ToolName != "" {
		details = append(details, fmt.Sprintf("  Tool: %s", item.ToolName))
	}

	if len(details) == 0 {
		return ""
	}

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(Gray)).
		Render(strings.Join(details, "\n"))
}

// getResponseIndicator returns the indicator for a response type
func (ah *ApprovalHistory) getResponseIndicator(response ApprovalResponse) string {
	switch response {
	case ApprovalYes:
		return ah.styles.approvedStyle.Render("✓")
	case ApprovalNo:
		return ah.styles.rejectedStyle.Render("✗")
	case ApprovalAlways:
		return ah.styles.alwaysStyle.Render("⭐")
	default:
		return "?"
	}
}

// formatActionType formats the action type for display
func (ah *ApprovalHistory) formatActionType(item ApprovalHistoryItem) string {
	switch item.AskType {
	case "tool":
		if item.ToolName != "" {
			return fmt.Sprintf("[%s]", item.ToolName)
		}
		return "[tool]"
	case "command":
		return "[cmd]"
	case "edit":
		return "[edit]"
	case "browser":
		return "[browser]"
	default:
		return fmt.Sprintf("[%s]", item.AskType)
	}
}

// formatTimestamp formats a timestamp for display
func (ah *ApprovalHistory) formatTimestamp(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	default:
		return t.Format("Jan 2")
	}
}

// GetAll returns all approval history items
func (ah *ApprovalHistory) GetAll() []ApprovalHistoryItem {
	return ah.items
}

// Len returns the number of items in history
func (ah *ApprovalHistory) Len() int {
	return len(ah.items)
}

// IsEmpty returns true if history is empty
func (ah *ApprovalHistory) IsEmpty() bool {
	return len(ah.items) == 0
}
