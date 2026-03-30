// Package components provides TUI components for the Cline CLI.
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// StatusBar displays persistent status information at the bottom of the screen
type StatusBar struct {
	// Status information
	mode           string // "act" or "plan"
	yolo           bool
	taskID         string
	provider       string
	model          string
	isStreaming    bool
	isConnected    bool
	messageCount   int
	pendingChanges int

	// Dimensions
	width int

	// Styling
	containerStyle lipgloss.Style
	modeStyleAct   lipgloss.Style
	modeStylePlan  lipgloss.Style
	yoloStyle      lipgloss.Style
	taskStyle      lipgloss.Style
	providerStyle  lipgloss.Style
	connectedStyle lipgloss.Style
	disconnStyle   lipgloss.Style
	streamingStyle lipgloss.Style
	changesStyle   lipgloss.Style
}

// NewStatusBar creates a new status bar
func NewStatusBar() *StatusBar {
	return &StatusBar{
		mode:     "act",
		width:    80,
		isConnected: true,
		
		containerStyle: lipgloss.NewStyle().
			Background(lipgloss.Color("#1a1a1a")).
			Foreground(lipgloss.Color("#cccccc")).
			Padding(0, 1),
		
		modeStyleAct: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#00D9FF")),
		
		modeStylePlan: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#FFB000")),
		
		yoloStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#00FF00")),
		
		taskStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")),
		
		providerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#aaaaaa")),
		
		connectedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")),
		
		disconnStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4444")),
		
		streamingStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00D9FF")).
			Bold(true),
		
		changesStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFB000")),
	}
}

// SetWidth sets the status bar width
func (sb *StatusBar) SetWidth(width int) {
	sb.width = width
}

// SetMode sets the mode (act/plan)
func (sb *StatusBar) SetMode(mode string) {
	sb.mode = mode
}

// SetYolo sets yolo mode
func (sb *StatusBar) SetYolo(yolo bool) {
	sb.yolo = yolo
}

// SetTaskID sets the task ID
func (sb *StatusBar) SetTaskID(taskID string) {
	sb.taskID = taskID
}

// SetProvider sets the provider name
func (sb *StatusBar) SetProvider(provider string) {
	sb.provider = provider
}

// SetModel sets the model name
func (sb *StatusBar) SetModel(model string) {
	sb.model = model
}

// SetStreaming sets the streaming state
func (sb *StatusBar) SetStreaming(streaming bool) {
	sb.isStreaming = streaming
}

// SetConnected sets the connection state
func (sb *StatusBar) SetConnected(connected bool) {
	sb.isConnected = connected
}

// SetMessageCount sets the message count
func (sb *StatusBar) SetMessageCount(count int) {
	sb.messageCount = count
}

// SetPendingChanges sets the pending changes count
func (sb *StatusBar) SetPendingChanges(count int) {
	sb.pendingChanges = count
}

// Render renders the status bar
func (sb *StatusBar) Render() string {
	var leftParts []string
	var rightParts []string

	// Left side: Mode indicator
	var modeStr string
	if sb.mode == "plan" {
		modeStr = sb.modeStylePlan.Render(" PLAN ")
	} else {
		modeStr = sb.modeStyleAct.Render(" ACT ")
	}
	leftParts = append(leftParts, modeStr)

	// Yolo indicator
	if sb.yolo {
		leftParts = append(leftParts, sb.yoloStyle.Render(" YOLO "))
	}

	// Task ID
	if sb.taskID != "" {
		shortID := sb.taskID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		leftParts = append(leftParts, sb.taskStyle.Render(fmt.Sprintf(" Task:%s ", shortID)))
	}

	// Right side: Connection status
	if sb.isConnected {
		rightParts = append(rightParts, sb.connectedStyle.Render("●"))
	} else {
		rightParts = append(rightParts, sb.disconnStyle.Render("○"))
	}

	// Streaming indicator
	if sb.isStreaming {
		rightParts = append(rightParts, sb.streamingStyle.Render("Streaming..."))
	}

	// Provider/Model info
	if sb.provider != "" {
		providerInfo := sb.provider
		if sb.model != "" {
			providerInfo += "/" + sb.model
		}
		rightParts = append(rightParts, sb.providerStyle.Render(providerInfo))
	}

	// Message count
	if sb.messageCount > 0 {
		rightParts = append(rightParts, sb.providerStyle.Render(fmt.Sprintf("%d msgs", sb.messageCount)))
	}

	// Pending changes
	if sb.pendingChanges > 0 {
		rightParts = append(rightParts, sb.changesStyle.Render(fmt.Sprintf("+%d changes", sb.pendingChanges)))
	}

	// Join parts
	leftContent := strings.Join(leftParts, "")
	rightContent := strings.Join(rightParts, " ")

	// Calculate spacing
	totalContentLen := lipgloss.Width(leftContent) + lipgloss.Width(rightContent)
	spaceAvailable := sb.width - totalContentLen

	var content string
	if spaceAvailable > 0 {
		content = leftContent + strings.Repeat(" ", spaceAvailable) + rightContent
	} else {
		content = leftContent + " " + rightContent
	}

	return sb.containerStyle.Width(sb.width).Render(content)
}

// GetHeight returns the height of the status bar
func (sb *StatusBar) GetHeight() int {
	return 1
}