// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DiffViewMode represents the diff view mode
type DiffViewMode int

const (
	// DiffViewModeUnified shows unified diff
	DiffViewModeUnified DiffViewMode = iota
	// DiffViewModeSplit shows split diff
	DiffViewModeSplit
)

// DiffModel is the Bubble Tea model for diff viewing
type DiffModel struct {
	width    int
	height   int
	viewport viewport.Model
	diff     string
	filename string
	mode     DiffViewMode

	// Styling
	addedStyle      lipgloss.Style
	removedStyle    lipgloss.Style
	contextStyle    lipgloss.Style
	headerStyle     lipgloss.Style
	hunkHeaderStyle lipgloss.Style

	// State
	ready   bool
	done    bool
	approve bool
}

// DiffResult is sent when diff viewing is complete
type DiffResult struct {
	Approved bool
	Error    error
}

// NewDiffModel creates a new diff model
func NewDiffModel(filename, diff string) DiffModel {
	return DiffModel{
		filename: filename,
		diff:     diff,
		mode:     DiffViewModeUnified,

		addedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")).
			Background(lipgloss.Color("#0a2a0a")),

		removedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Background(lipgloss.Color("#2a0a0a")),

		contextStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0E0E0")),

		headerStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")),

		hunkHeaderStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD700")).
			Bold(true),
	}
}

// SetDimensions sets the terminal dimensions
func (m *DiffModel) SetDimensions(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height - 4 // Reserve space for header and footer
}

// Init initializes the model
func (m DiffModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m DiffModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetDimensions(msg.Width, msg.Height)
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-4)
			m.viewport.SetContent(m.renderDiff())
			m.ready = true
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.done = true
			m.approve = false
			return m, tea.Quit

		case "y":
			m.done = true
			m.approve = true
			return m, tea.Quit

		case "n":
			m.done = true
			m.approve = false
			return m, tea.Quit

		case "up", "k":
			m.viewport.LineUp(1)
			return m, nil

		case "down", "j":
			m.viewport.LineDown(1)
			return m, nil

		case "page_up":
			m.viewport.HalfViewUp()
			return m, nil

		case "page_down":
			m.viewport.HalfViewDown()
			return m, nil

		case "g":
			m.viewport.GotoTop()
			return m, nil

		case "G":
			m.viewport.GotoBottom()
			return m, nil

		case "m":
			// Toggle view mode
			if m.mode == DiffViewModeUnified {
				m.mode = DiffViewModeSplit
			} else {
				m.mode = DiffViewModeUnified
			}
			m.viewport.SetContent(m.renderDiff())
			return m, nil

		case "/":
			// Search functionality could be added here
			return m, nil
		}
	}

	// Update viewport
	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

// View renders the diff viewer
func (m DiffModel) View() string {
	if !m.ready {
		return "Loading diff..."
	}

	var content strings.Builder

	// Header
	header := m.headerStyle.Render(fmt.Sprintf("Diff: %s", m.filename))
	modeStr := "Unified"
	if m.mode == DiffViewModeSplit {
		modeStr = "Split"
	}
	modeInfo := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080")).
		Render(fmt.Sprintf(" [%s] ", modeStr))

	content.WriteString(header)
	content.WriteString(modeInfo)
	content.WriteString("\n")
	content.WriteString(strings.Repeat("─", m.width))
	content.WriteString("\n")

	// Diff content
	content.WriteString(m.viewport.View())
	content.WriteString("\n")

	// Footer
	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#606060")).
		Render("y: approve, n: reject, ↑↓: scroll, m: toggle mode, q: quit")
	content.WriteString(footer)

	return content.String()
}

// renderDiff renders the diff with syntax highlighting
func (m DiffModel) renderDiff() string {
	var content strings.Builder
	lines := strings.Split(m.diff, "\n")

	for _, line := range lines {
		rendered := m.renderLine(line)
		content.WriteString(rendered)
		content.WriteString("\n")
	}

	return content.String()
}

// renderLine renders a single diff line with appropriate styling
func (m DiffModel) renderLine(line string) string {
	if len(line) == 0 {
		return m.contextStyle.Render(" ")
	}

	switch line[0] {
	case '+':
		return m.addedStyle.Render(line)
	case '-':
		// Could be removal line or file header
		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") {
			return m.headerStyle.Render(line)
		}
		return m.removedStyle.Render(line)
	case '@':
		return m.hunkHeaderStyle.Render(line)
	case 'd':
		if strings.HasPrefix(line, "diff ") {
			return m.headerStyle.Render(line)
		}
		return m.contextStyle.Render(line)
	default:
		return m.contextStyle.Render(line)
	}
}

// IsDone returns true if viewing is complete
func (m DiffModel) IsDone() bool {
	return m.done
}

// IsApproved returns the approval decision
func (m DiffModel) IsApproved() bool {
	return m.approve
}

// ShowDiffViewer shows a diff viewer and returns approval decision
func ShowDiffViewer(filename, diff string) (bool, error) {
	model := NewDiffModel(filename, diff)

	p := tea.NewProgram(model, tea.WithAltScreen())

	m, err := p.Run()
	if err != nil {
		return false, err
	}

	diffModel, ok := m.(DiffModel)
	if !ok {
		return false, fmt.Errorf("unexpected model type")
	}

	return diffModel.IsApproved(), nil
}

// FormatDiff formats a unified diff string for display
func FormatDiff(diff string, maxWidth int) string {
	if maxWidth <= 0 {
		maxWidth = 80
	}

	lines := strings.Split(diff, "\n")
	var result strings.Builder

	for _, line := range lines {
		if len(line) > maxWidth {
			// Wrap long lines
			for len(line) > 0 {
				end := maxWidth
				if end > len(line) {
					end = len(line)
				}
				result.WriteString(line[:end])
				result.WriteString("\n")
				line = line[end:]
			}
		} else {
			result.WriteString(line)
			result.WriteString("\n")
		}
	}

	return result.String()
}
