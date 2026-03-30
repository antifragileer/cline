// Package components provides TUI components for the Cline CLI.
package components

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SpinnerType represents different spinner styles
type SpinnerType int

const (
	// SpinnerTypeDots uses dot animation
	SpinnerTypeDots SpinnerType = iota
	// SpinnerTypeLine uses line animation
	SpinnerTypeLine
	// SpinnerTypeArrow uses arrow animation
	SpinnerTypeArrow
	// SpinnerTypeBouncingBar uses bouncing bar
	SpinnerTypeBouncingBar
)

// SpinnerFrames contains animation frames for different spinner types
var SpinnerFrames = map[SpinnerType][]string{
	SpinnerTypeDots: {
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
	},
	SpinnerTypeLine: {
		"-",
		"\\",
		"|",
		"/",
	},
	SpinnerTypeArrow: {
		"←",
		"↖",
		"↑",
		"↗",
		"→",
		"↘",
		"↓",
		"↙",
	},
	SpinnerTypeBouncingBar: {
		"[    ]",
		"[=   ]",
		"[==  ]",
		"[=== ]",
		"[====]",
		"[ ===]",
		"[  ==]",
		"[   =]",
	},
}

// Spinner is a reusable loading indicator component
type Spinner struct {
	// Configuration
	typ   SpinnerType
	text  string
	speed time.Duration

	// State
	frames     []string
	frameIndex int
	isRunning  bool
	startTime  time.Time

	// Styling
	frameStyle lipgloss.Style
	textStyle  lipgloss.Style
	timeStyle  lipgloss.Style

	// Callbacks
	onCancel func()
}

// SpinnerOption configures the spinner
type SpinnerOption func(*Spinner)

// WithSpinnerType sets the spinner type
func WithSpinnerType(typ SpinnerType) SpinnerOption {
	return func(s *Spinner) {
		s.typ = typ
		s.frames = SpinnerFrames[typ]
	}
}

// WithSpinnerText sets the spinner text
func WithSpinnerText(text string) SpinnerOption {
	return func(s *Spinner) {
		s.text = text
	}
}

// WithSpinnerSpeed sets the animation speed
func WithSpinnerSpeed(speed time.Duration) SpinnerOption {
	return func(s *Spinner) {
		s.speed = speed
	}
}

// WithSpinnerStyle sets the frame style
func WithSpinnerStyle(style lipgloss.Style) SpinnerOption {
	return func(s *Spinner) {
		s.frameStyle = style
	}
}

// NewSpinner creates a new spinner
func NewSpinner(opts ...SpinnerOption) *Spinner {
	s := &Spinner{
		typ:        SpinnerTypeDots,
		frames:     SpinnerFrames[SpinnerTypeDots],
		text:       "Loading...",
		speed:      80 * time.Millisecond,
		isRunning:  false,
		frameIndex: 0,
		startTime:  time.Now(),
		frameStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00D9FF")).
			Bold(true),
		textStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0E0E0")),
		timeStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")).
			Italic(true),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Init initializes the spinner with a tick command
func (s *Spinner) Init() tea.Cmd {
	s.isRunning = true
	s.startTime = time.Now()
	return s.tick()
}

// Update handles messages
func (s *Spinner) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tickMsg:
		if !s.isRunning {
			return nil
		}
		s.frameIndex = (s.frameIndex + 1) % len(s.frames)
		return s.tick()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			if s.onCancel != nil {
				s.onCancel()
			}
			s.Stop()
		}
	}

	return nil
}

// View renders the spinner
func (s *Spinner) View() string {
	if !s.isRunning {
		return ""
	}

	frame := s.frames[s.frameIndex]
	elapsed := time.Since(s.startTime)

	var content string
	if elapsed > time.Second {
		content = fmt.Sprintf("%s %s %s",
			s.frameStyle.Render(frame),
			s.textStyle.Render(s.text),
			s.timeStyle.Render(formatSpinnerDuration(elapsed)))
	} else {
		content = fmt.Sprintf("%s %s",
			s.frameStyle.Render(frame),
			s.textStyle.Render(s.text))
	}

	return content
}

// Start starts the spinner
func (s *Spinner) Start() {
	s.isRunning = true
	s.startTime = time.Now()
}

// Stop stops the spinner
func (s *Spinner) Stop() {
	s.isRunning = false
}

// IsRunning returns whether the spinner is running
func (s *Spinner) IsRunning() bool {
	return s.isRunning
}

// SetText updates the spinner text
func (s *Spinner) SetText(text string) {
	s.text = text
}

// SetOnCancel sets the cancel callback
func (s *Spinner) SetOnCancel(fn func()) {
	s.onCancel = fn
}

// tick returns a command that sends a tick message
func (s *Spinner) tick() tea.Cmd {
	return tea.Tick(s.speed, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// tickMsg is sent on each animation tick
type tickMsg time.Time

// formatSpinnerDuration formats a duration for display
func formatSpinnerDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("(%ds)", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("(%dm %ds)", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("(%dh %dm)", int(d.Hours()), int(d.Minutes())%60)
}

// StaticSpinner renders a static spinner frame (for use in lists)
func StaticSpinner(frameIndex int, text string, style lipgloss.Style) string {
	frames := SpinnerFrames[SpinnerTypeDots]
	frame := frames[frameIndex%len(frames)]
	return style.Render(frame) + " " + text
}

// SpinnerWithMode creates a mode-aware spinner
func SpinnerWithMode(text, mode string) *Spinner {
	modeColor := "#00D9FF" // Act mode
	if mode == "plan" {
		modeColor = "#FFB000"
	}

	return NewSpinner(
		WithSpinnerText(text),
		WithSpinnerStyle(lipgloss.NewStyle().
			Foreground(lipgloss.Color(modeColor)).
			Bold(true)),
	)
}