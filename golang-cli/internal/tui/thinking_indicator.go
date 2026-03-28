// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SpinnerFrames contains the animation frames for the thinking spinner
var SpinnerFrames = []string{
	"⠋",
	"⠙",
	"⠹",
	"⠸",
	"⠼",
	"⠴",
	"⠦",
	"⠧",
	"⠇",
	"⠏",
}

// ThinkingIndicator displays a spinner with status text and optional cancel button
type ThinkingIndicator struct {
	// State
	frames     []string
	frameIndex int
	text       string
	mode       string // "act" or "plan"
	startTime  time.Time

	// Dimensions
	width  int
	height int

	// Styling
	spinnerStyle    lipgloss.Style
	textStyle       lipgloss.Style
	cancelStyle     lipgloss.Style
	modeStyle       lipgloss.Style

	// Cancel callback
	onCancel func()
}

// NewThinkingIndicator creates a new thinking indicator
func NewThinkingIndicator(text, mode string) *ThinkingIndicator {
	modeColor := "#00D9FF" // Act mode blue
	if mode == "plan" {
		modeColor = "#FFB000" // Plan mode yellow
	}

	return &ThinkingIndicator{
		frames:    SpinnerFrames,
		frameIndex: 0,
		text:      text,
		mode:      mode,
		startTime: time.Now(),
		spinnerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(modeColor)).
			Bold(true),
		textStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0E0E0")),
		cancelStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")).
			Italic(true),
		modeStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(modeColor)).
			Bold(true),
	}
}

// Init initializes the model with a tick command
func (m ThinkingIndicator) Init() tea.Cmd {
	return m.tick()
}

// Update handles messages
func (m *ThinkingIndicator) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		m.frameIndex = (m.frameIndex + 1) % len(m.frames)
		return m, m.tick()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			if m.onCancel != nil {
				m.onCancel()
			}
		}
	}

	return m, nil
}

// View renders the thinking indicator
func (m ThinkingIndicator) View() string {
	var content strings.Builder

	// Spinner + text
	spinner := m.spinnerStyle.Render(m.frames[m.frameIndex])
	text := m.textStyle.Render(m.text)

	// Calculate elapsed time
	elapsed := time.Since(m.startTime)
	elapsedStr := formatDuration(elapsed)

	// Mode indicator
	modeIndicator := ""
	if m.mode == "plan" {
		modeIndicator = m.modeStyle.Render("[Plan] ")
	}

	content.WriteString(fmt.Sprintf("%s %s%s %s", spinner, modeIndicator, text, m.cancelStyle.Render(elapsedStr)))

	return content.String()
}

// tick returns a command that sends a tick message after a delay
func (m ThinkingIndicator) tick() tea.Cmd {
	return tea.Tick(time.Millisecond*80, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// SetText updates the indicator text
func (m *ThinkingIndicator) SetText(text string) {
	m.text = text
}

// SetOnCancel sets the cancel callback
func (m *ThinkingIndicator) SetOnCancel(fn func()) {
	m.onCancel = fn
}

// GetElapsedTime returns the elapsed time since the indicator started
func (m *ThinkingIndicator) GetElapsedTime() time.Duration {
	return time.Since(m.startTime)
}

// tickMsg is sent on each animation tick
type tickMsg time.Time

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("(%ds)", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("(%dm %ds)", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("(%dh %dm)", int(d.Hours()), int(d.Minutes())%60)
}

// ThinkingIndicatorMsg is sent to show/hide the thinking indicator
type ThinkingIndicatorMsg struct {
	Show      bool
	Text      string
	Mode      string
	OnCancel  func()
}

// StaticThinkingIndicator renders a static thinking indicator (no animation)
// Used when we want to show the indicator in the main view without running a separate program
func StaticThinkingIndicator(frameIndex int, text, mode string, elapsed time.Duration) string {
	modeColor := "#00D9FF" // Act mode blue
	if mode == "plan" {
		modeColor = "#FFB000" // Plan mode yellow
	}

	spinnerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(modeColor)).
		Bold(true)
	textStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E0E0E0"))
	cancelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080")).
		Italic(true)
	modeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(modeColor)).
		Bold(true)

	spinner := spinnerStyle.Render(SpinnerFrames[frameIndex%len(SpinnerFrames)])
	elapsedStr := formatDuration(elapsed)

	modeIndicator := ""
	if mode == "plan" {
		modeIndicator = modeStyle.Render("[Plan] ")
	}

	return fmt.Sprintf("%s %s%s %s", spinner, modeIndicator, textStyle.Render(text), cancelStyle.Render(elapsedStr))
}