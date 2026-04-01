// Package components provides TUI components for the Cline CLI.
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// StatusType represents the type of status
type StatusType string

const (
	// StatusTypeIdle indicates the system is idle
	StatusTypeIdle StatusType = "idle"
	// StatusTypeConnecting indicates connecting to the server
	StatusTypeConnecting StatusType = "connecting"
	// StatusTypeRunning indicates a task is running
	StatusTypeRunning StatusType = "running"
	// StatusTypeError indicates an error state
	StatusTypeError StatusType = "error"
	// StatusTypeSuccess indicates success
	StatusTypeSuccess StatusType = "success"
)

// StatusBar displays the current application status
type StatusBar struct {
	status       StatusType
	message      string
	taskID       string
	mode         string
	connection   string
	provider     string
	model        string
	yolo         bool
	streaming    bool
	messageCount int
	pendingChanges int
	pendingApprovals int
	width        int
}

// NewStatusBar creates a new status bar
func NewStatusBar(width int) *StatusBar {
	return &StatusBar{
		status:     StatusTypeIdle,
		width:      width,
		mode:       "act",
		connection: "disconnected",
	}
}

// SetStatus updates the status
func (sb *StatusBar) SetStatus(status StatusType, message string) {
	sb.status = status
	sb.message = message
}

// SetTaskID sets the current task ID
func (sb *StatusBar) SetTaskID(taskID string) {
	sb.taskID = taskID
}

// SetMode sets the current mode
func (sb *StatusBar) SetMode(mode string) {
	sb.mode = mode
}

// SetConnection sets the connection status
func (sb *StatusBar) SetConnection(status string) {
	sb.connection = status
}

// SetPendingApprovals sets the number of pending approvals
func (sb *StatusBar) SetPendingApprovals(count int) {
	sb.pendingApprovals = count
}

// SetYolo sets yolo mode
func (sb *StatusBar) SetYolo(yolo bool) {
	sb.yolo = yolo
}

// SetProvider sets the provider
func (sb *StatusBar) SetProvider(provider string) {
	sb.provider = provider
}

// SetModel sets the model
func (sb *StatusBar) SetModel(model string) {
	sb.model = model
}

// SetStreaming sets streaming state
func (sb *StatusBar) SetStreaming(streaming bool) {
	sb.streaming = streaming
	if streaming {
		sb.status = StatusTypeRunning
		sb.message = "Streaming..."
	}
}

// SetMessageCount sets the message count
func (sb *StatusBar) SetMessageCount(count int) {
	sb.messageCount = count
}

// SetPendingChanges sets the number of pending changes
func (sb *StatusBar) SetPendingChanges(count int) {
	sb.pendingChanges = count
}

// SetWidth updates the width
func (sb *StatusBar) SetWidth(width int) {
	sb.width = width
}

// Render renders the status bar
func (sb *StatusBar) Render() string {
	// Calculate available width
	leftWidth := sb.width / 3
	centerWidth := sb.width / 3
	rightWidth := sb.width - leftWidth - centerWidth

	// Left section: Status indicator
	left := sb.renderLeftSection(leftWidth)

	// Center section: Task info
	center := sb.renderCenterSection(centerWidth)

	// Right section: Mode and connection
	right := sb.renderRightSection(rightWidth)

	// Combine sections
	leftStyle := lipgloss.NewStyle().Width(leftWidth).Align(lipgloss.Left)
	centerStyle := lipgloss.NewStyle().Width(centerWidth).Align(lipgloss.Center)
	rightStyle := lipgloss.NewStyle().Width(rightWidth).Align(lipgloss.Right)

	statusLine := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftStyle.Render(left),
		centerStyle.Render(center),
		rightStyle.Render(right),
	)

	// Apply background and border
	barStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(sb.getStatusColor()).
		BorderTop(true).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		Padding(0, 1)

	return barStyle.Render(statusLine)
}

// renderLeftSection renders the left section (status)
func (sb *StatusBar) renderLeftSection(width int) string {
	var indicator string
	var color string

	switch sb.status {
	case StatusTypeIdle:
		indicator = "○"
		color = "#888888"
	case StatusTypeConnecting:
		indicator = "◐"
		color = "#F39C12"
	case StatusTypeRunning:
		indicator = "◉"
		color = "#4A90D9"
	case StatusTypeError:
		indicator = "✕"
		color = "#E74C3C"
	case StatusTypeSuccess:
		indicator = "✓"
		color = "#2ECC71"
	}

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(color)).
		Bold(true)

	messageStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#CCCCCC")).
		MarginLeft(1)

	content := statusStyle.Render(indicator) + messageStyle.Render(truncate(sb.message, width-5))
	return content
}

// renderCenterSection renders the center section (task info)
func (sb *StatusBar) renderCenterSection(width int) string {
	var parts []string

	if sb.taskID != "" {
		taskStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))
		parts = append(parts, taskStyle.Render("Task: "+truncate(sb.taskID, 12)))
	}

	if sb.pendingApprovals > 0 {
		approvalStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F39C12")).
			Bold(true)
		parts = append(parts, approvalStyle.Render(fmt.Sprintf("⚠️ %d pending", sb.pendingApprovals)))
	}

	return strings.Join(parts, " | ")
}

// renderRightSection renders the right section (mode and connection)
func (sb *StatusBar) renderRightSection(width int) string {
	var parts []string

	// Mode indicator
	modeColor := "#4A90D9"
	if sb.mode == "plan" {
		modeColor = "#9B59B6"
	}
	modeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(modeColor)).
		Bold(true)
	parts = append(parts, modeStyle.Render(strings.ToUpper(sb.mode)))

	// Connection indicator
	connColor := "#E74C3C"
	if sb.connection == "connected" {
		connColor = "#2ECC71"
	} else if sb.connection == "connecting" {
		connColor = "#F39C12"
	}
	connStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(connColor))
	parts = append(parts, connStyle.Render("●"))

	return strings.Join(parts, "  ")
}

// getStatusColor returns the color for the current status
func (sb *StatusBar) getStatusColor() lipgloss.Color {
	switch sb.status {
	case StatusTypeIdle:
		return lipgloss.Color("#888888")
	case StatusTypeConnecting:
		return lipgloss.Color("#F39C12")
	case StatusTypeRunning:
		return lipgloss.Color("#4A90D9")
	case StatusTypeError:
		return lipgloss.Color("#E74C3C")
	case StatusTypeSuccess:
		return lipgloss.Color("#2ECC71")
	default:
		return lipgloss.Color("#888888")
	}
}

// truncate truncates a string to the specified length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// CompactStatusBar is a minimal status bar for smaller screens
type CompactStatusBar struct {
	status  StatusType
	message string
	width   int
}

// NewCompactStatusBar creates a new compact status bar
func NewCompactStatusBar(width int) *CompactStatusBar {
	return &CompactStatusBar{
		status: StatusTypeIdle,
		width:  width,
	}
}

// SetStatus updates the status
func (csb *CompactStatusBar) SetStatus(status StatusType, message string) {
	csb.status = status
	csb.message = message
}

// SetWidth updates the width
func (csb *CompactStatusBar) SetWidth(width int) {
	csb.width = width
}

// Render renders the compact status bar
func (csb *CompactStatusBar) Render() string {
	var indicator string
	var color string

	switch csb.status {
	case StatusTypeIdle:
		indicator = "○"
		color = "#888888"
	case StatusTypeConnecting:
		indicator = "◐"
		color = "#F39C12"
	case StatusTypeRunning:
		indicator = "◉"
		color = "#4A90D9"
	case StatusTypeError:
		indicator = "✕"
		color = "#E74C3C"
	case StatusTypeSuccess:
		indicator = "✓"
		color = "#2ECC71"
	}

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(color)).
		Bold(true)

	messageStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#CCCCCC"))

	content := statusStyle.Render(indicator) + " " + messageStyle.Render(truncate(csb.message, csb.width-5))

	barStyle := lipgloss.NewStyle().
		Width(csb.width).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(color)).
		BorderTop(true).
		Padding(0, 1)

	return barStyle.Render(content)
}