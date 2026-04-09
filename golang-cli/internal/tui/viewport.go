// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Viewport manages a scrollable viewport for content display
type Viewport struct {
	width        int
	height       int
	content      []string
	yOffset      int
	xOffset      int
	styles       ViewportStyles
	wrapContent  bool
	showLineNum  bool
	totalLines   int
	selectedLine int
}

// ViewportStyles holds styles for the viewport
type ViewportStyles struct {
	borderStyle    lipgloss.Style
	contentStyle   lipgloss.Style
	lineNumStyle   lipgloss.Style
	selectedStyle  lipgloss.Style
	scrollBarStyle lipgloss.Style
}

// DefaultViewportStyles returns default viewport styles
func DefaultViewportStyles() ViewportStyles {
	return ViewportStyles{
		borderStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")),

		contentStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(White)),

		lineNumStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(Gray)).
			Width(4).
			Align(lipgloss.Right),

		selectedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(SelectionBlue)).
			Background(lipgloss.Color(DarkBackground)),

		scrollBarStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(PrimaryBlue)).
			Background(lipgloss.Color(DarkBackground)),
	}
}

// NewViewport creates a new viewport
func NewViewport(width, height int) *Viewport {
	return &Viewport{
		width:       width,
		height:      height,
		content:     make([]string, 0),
		styles:      DefaultViewportStyles(),
		wrapContent: true,
		showLineNum: false,
		totalLines:  0,
	}
}

// SetContent sets the content to display
func (v *Viewport) SetContent(content string) {
	v.content = strings.Split(content, "\n")
	v.totalLines = len(v.content)
	v.yOffset = 0
}

// AppendContent appends content to the viewport
func (v *Viewport) AppendContent(content string) {
	lines := strings.Split(content, "\n")
	v.content = append(v.content, lines...)
	v.totalLines = len(v.content)
}

// SetLines sets the content as pre-split lines
func (v *Viewport) SetLines(lines []string) {
	v.content = lines
	v.totalLines = len(lines)
}

// GetContent returns the current content
func (v *Viewport) GetContent() string {
	return strings.Join(v.content, "\n")
}

// GetLines returns the current lines
func (v *Viewport) GetLines() []string {
	return v.content
}

// Init initializes the viewport
func (v *Viewport) Init() tea.Cmd {
	return nil
}

// Update handles messages for the viewport
func (v *Viewport) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			v.ScrollUp(1)
		case "down", "j":
			v.ScrollDown(1)
		case "pgup":
			v.ScrollUp(v.height - 1)
		case "pgdown", " ":
			v.ScrollDown(v.height - 1)
		case "home", "g":
			v.GotoTop()
		case "end", "G":
			v.GotoBottom()
		case "left", "h":
			v.ScrollLeft(1)
		case "right", "l":
			v.ScrollRight(1)
		}
	}

	return v, nil
}

// ScrollUp scrolls up by n lines
func (v *Viewport) ScrollUp(n int) {
	v.yOffset -= n
	if v.yOffset < 0 {
		v.yOffset = 0
	}
}

// ScrollDown scrolls down by n lines
func (v *Viewport) ScrollDown(n int) {
	v.yOffset += n
	maxOffset := v.getMaxYOffset()
	if v.yOffset > maxOffset {
		v.yOffset = maxOffset
	}
}

// ScrollLeft scrolls left by n columns
func (v *Viewport) ScrollLeft(n int) {
	v.xOffset -= n
	if v.xOffset < 0 {
		v.xOffset = 0
	}
}

// ScrollRight scrolls right by n columns
func (v *Viewport) ScrollRight(n int) {
	v.xOffset += n
}

// GotoTop scrolls to the top
func (v *Viewport) GotoTop() {
	v.yOffset = 0
}

// GotoBottom scrolls to the bottom
func (v *Viewport) GotoBottom() {
	v.yOffset = v.getMaxYOffset()
}

// getMaxYOffset returns the maximum y offset
func (v *Viewport) getMaxYOffset() int {
	if v.totalLines <= v.height {
		return 0
	}
	return v.totalLines - v.height
}

// GetYOffset returns the current y offset
func (v *Viewport) GetYOffset() int {
	return v.yOffset
}

// SetYOffset sets the y offset
func (v *Viewport) SetYOffset(offset int) {
	v.yOffset = offset
	maxOffset := v.getMaxYOffset()
	if v.yOffset > maxOffset {
		v.yOffset = maxOffset
	}
	if v.yOffset < 0 {
		v.yOffset = 0
	}
}

// SetDimensions sets the viewport dimensions
func (v *Viewport) SetDimensions(width, height int) {
	v.width = width
	v.height = height
}

// SetWrap sets whether to wrap content
func (v *Viewport) SetWrap(wrap bool) {
	v.wrapContent = wrap
}

// SetShowLineNumbers sets whether to show line numbers
func (v *Viewport) SetShowLineNumbers(show bool) {
	v.showLineNum = show
}

// AtTop returns true if at the top of the content
func (v *Viewport) AtTop() bool {
	return v.yOffset == 0
}

// AtBottom returns true if at the bottom of the content
func (v *Viewport) AtBottom() bool {
	return v.yOffset >= v.getMaxYOffset()
}

// View renders the viewport
func (v *Viewport) View() string {
	if v.totalLines == 0 {
		return v.styles.contentStyle.Render("")
	}

	visibleLines := v.getVisibleLines()

	var content strings.Builder
	for i, line := range visibleLines {
		lineNum := v.yOffset + i + 1
		renderedLine := v.renderLine(line, lineNum)
		content.WriteString(renderedLine)
		if i < len(visibleLines)-1 {
			content.WriteString("\n")
		}
	}

	// Fill remaining space if needed
	for i := len(visibleLines); i < v.height; i++ {
		content.WriteString("\n")
	}

	mainContent := content.String()

	// Add scrollbar if needed
	if v.totalLines > v.height {
		scrollBar := v.renderScrollBar()
		mainContent = lipgloss.JoinHorizontal(lipgloss.Top, mainContent, scrollBar)
	}

	return mainContent
}

// getVisibleLines returns the lines currently visible in the viewport
func (v *Viewport) getVisibleLines() []string {
	end := v.yOffset + v.height
	if end > v.totalLines {
		end = v.totalLines
	}
	if v.yOffset >= len(v.content) {
		return []string{}
	}
	return v.content[v.yOffset:end]
}

// renderLine renders a single line
func (v *Viewport) renderLine(line string, lineNum int) string {
	// Apply horizontal scroll
	if v.xOffset > 0 && len(line) > v.xOffset {
		line = line[v.xOffset:]
	}

	// Apply width limit
	contentWidth := v.width
	if v.showLineNum {
		contentWidth -= 5 // Account for line number
	}
	if v.totalLines > v.height {
		contentWidth -= 1 // Account for scrollbar
	}

	if len(line) > contentWidth {
		line = line[:contentWidth]
	}

	// Apply line number if enabled
	if v.showLineNum {
		lineNumStr := v.styles.lineNumStyle.Render(fmt.Sprintf("%d", lineNum))
		line = lineNumStr + " " + line
	}

	// Apply selection style if needed
	if lineNum == v.selectedLine {
		line = v.styles.selectedStyle.Render(line)
	}

	return line
}

// renderScrollBar renders the scrollbar
func (v *Viewport) renderScrollBar() string {
	if v.totalLines <= v.height {
		return ""
	}

	// Calculate scroll thumb position and size
	thumbSize := v.height * v.height / v.totalLines
	if thumbSize < 1 {
		thumbSize = 1
	}

	thumbPos := v.yOffset * (v.height - thumbSize) / (v.totalLines - v.height)
	if thumbPos < 0 {
		thumbPos = 0
	}
	if thumbPos > v.height-thumbSize {
		thumbPos = v.height - thumbSize
	}

	// Build scrollbar
	var sb strings.Builder
	for i := 0; i < v.height; i++ {
		if i >= thumbPos && i < thumbPos+thumbSize {
			sb.WriteString("█")
		} else {
			sb.WriteString("░")
		}
		sb.WriteString("\n")
	}

	return v.styles.scrollBarStyle.Render(sb.String())
}

// SetSelectedLine sets the selected line
func (v *Viewport) SetSelectedLine(lineNum int) {
	v.selectedLine = lineNum
}

// GetTotalLines returns the total number of lines
func (v *Viewport) GetTotalLines() int {
	return v.totalLines
}

// GetVisibleHeight returns the height of the visible area
func (v *Viewport) GetVisibleHeight() int {
	return v.height
}

// GetVisibleWidth returns the width of the visible area
func (v *Viewport) GetVisibleWidth() int {
	return v.width
}

// ScrollPercentage returns the scroll position as a percentage (0-100)
func (v *Viewport) ScrollPercentage() int {
	if v.totalLines <= v.height {
		return 100
	}
	return v.yOffset * 100 / (v.totalLines - v.height)
}
