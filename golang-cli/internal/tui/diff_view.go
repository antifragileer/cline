// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// DiffView displays file diffs for tool approval.
type DiffView struct {
	oldContent string
	newContent string
	filePath   string
	renderer   *DiffRenderer
}

// DiffRenderer renders diffs with syntax highlighting.
type DiffRenderer struct {
	addedStyle   lipgloss.Style
	removedStyle lipgloss.Style
	contextStyle lipgloss.Style
	headerStyle  lipgloss.Style
	width        int
	contextLines int
}

// NewDiffRenderer creates a new diff renderer.
func NewDiffRenderer() *DiffRenderer {
	return &DiffRenderer{
		addedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")).
			Background(lipgloss.Color("#003300")),

		removedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4444")).
			Background(lipgloss.Color("#330000")),

		contextStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CCCCCC")),

		headerStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true),

		width:        120,
		contextLines: 3,
	}
}

// SetWidth sets the rendering width.
func (r *DiffRenderer) SetWidth(width int) {
	r.width = width
}

// SetContextLines sets the number of context lines to show.
func (r *DiffRenderer) SetContextLines(lines int) {
	r.contextLines = lines
}

// NewDiffView creates a new diff view.
func NewDiffView(filePath, oldContent, newContent string) *DiffView {
	return &DiffView{
		filePath:   filePath,
		oldContent: oldContent,
		newContent: newContent,
		renderer:   NewDiffRenderer(),
	}
}

// Render renders the diff.
func (v *DiffView) Render() string {
	return v.renderer.Render(v.filePath, v.oldContent, v.newContent)
}

// HasChanges returns true if there are changes.
func (v *DiffView) HasChanges() bool {
	return v.oldContent != v.newContent
}

// GetStats returns diff statistics.
func (v *DiffView) GetStats() (additions, deletions int) {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(v.oldContent, v.newContent, false)

	for _, diff := range diffs {
		switch diff.Type {
		case diffmatchpatch.DiffInsert:
			additions += len(strings.Split(diff.Text, "\n"))
		case diffmatchpatch.DiffDelete:
			deletions += len(strings.Split(diff.Text, "\n"))
		}
	}

	return additions, deletions
}

// Render renders a diff between old and new content.
func (r *DiffRenderer) Render(filePath, oldContent, newContent string) string {
	if oldContent == newContent {
		return r.headerStyle.Render(fmt.Sprintf("No changes in %s", filePath))
	}

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(oldContent, newContent, false)

	var result strings.Builder

	// Header
	header := r.renderHeader(filePath, oldContent, newContent)
	result.WriteString(header)
	result.WriteString("\n")

	// Diff content
	diffContent := r.renderDiffs(diffs)
	result.WriteString(diffContent)

	return result.String()
}

// renderHeader renders the diff header.
func (r *DiffRenderer) renderHeader(filePath, oldContent, newContent string) string {
	additions, deletions := r.countChanges(oldContent, newContent)

	var parts []string
	parts = append(parts, r.headerStyle.Render(fmt.Sprintf("📄 %s", filePath)))

	if additions > 0 {
		parts = append(parts, lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")).
			Render(fmt.Sprintf("+%d", additions)))
	}

	if deletions > 0 {
		parts = append(parts, lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4444")).
			Render(fmt.Sprintf("-%d", deletions)))
	}

	return strings.Join(parts, " ")
}

// renderDiffs renders the diff hunks.
func (r *DiffRenderer) renderDiffs(diffs []diffmatchpatch.Diff) string {
	var result strings.Builder
	var oldLineNum, newLineNum int = 1, 1

	for _, diff := range diffs {
		lines := strings.Split(diff.Text, "\n")

		for i, line := range lines {
			// Skip empty last line from split
			if i == len(lines)-1 && line == "" {
				continue
			}

			switch diff.Type {
			case diffmatchpatch.DiffDelete:
				prefix := r.removedStyle.Render(fmt.Sprintf("- %3d   ", oldLineNum))
				result.WriteString(prefix)
				result.WriteString(r.removedStyle.Render(r.truncateLine(line)))
				result.WriteString("\n")
				oldLineNum++

			case diffmatchpatch.DiffInsert:
				prefix := r.addedStyle.Render(fmt.Sprintf("+   %3d ", newLineNum))
				result.WriteString(prefix)
				result.WriteString(r.addedStyle.Render(r.truncateLine(line)))
				result.WriteString("\n")
				newLineNum++

			case diffmatchpatch.DiffEqual:
				// Only show context lines around changes
				prefix := r.contextStyle.Render(fmt.Sprintf("  %3d %3d", oldLineNum, newLineNum))
				result.WriteString(prefix)
				result.WriteString(" ")
				result.WriteString(r.contextStyle.Render(r.truncateLine(line)))
				result.WriteString("\n")
				oldLineNum++
				newLineNum++
			}
		}
	}

	return result.String()
}

// truncateLine truncates a line to fit within width.
func (r *DiffRenderer) truncateLine(line string) string {
	maxLen := r.width - 12 // Account for line numbers and prefix
	if len(line) <= maxLen {
		return line
	}
	return line[:maxLen-3] + "..."
}

// countChanges counts additions and deletions.
func (r *DiffRenderer) countChanges(oldContent, newContent string) (additions, deletions int) {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(oldContent, newContent, false)

	for _, diff := range diffs {
		lines := strings.Split(diff.Text, "\n")
		// Don't count the empty string from trailing newline
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}

		switch diff.Type {
		case diffmatchpatch.DiffInsert:
			additions += len(lines)
		case diffmatchpatch.DiffDelete:
			deletions += len(lines)
		}
	}

	return additions, deletions
}

// DiffViewer provides an interactive diff viewer.
type DiffViewer struct {
	diffs     []FileDiff
	current   int
	width     int
	height    int
	styles    DiffViewerStyles
}

// FileDiff represents a diff for a single file.
type FileDiff struct {
	Path       string
	OldContent string
	NewContent string
	Added      int
	Deleted    int
}

// DiffViewerStyles holds styles for the diff viewer.
type DiffViewerStyles struct {
	containerStyle lipgloss.Style
	titleStyle     lipgloss.Style
	helpStyle      lipgloss.Style
}

// NewDiffViewer creates a new diff viewer.
func NewDiffViewer() *DiffViewer {
	return &DiffViewer{
		diffs:  make([]FileDiff, 0),
		current: 0,
		styles: DiffViewerStyles{
			containerStyle: lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#7D56F4")).
				Padding(1),

			titleStyle: lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7D56F4")).
				Bold(true),

			helpStyle: lipgloss.NewStyle().
				Foreground(lipgloss.Color("#606060")),
		},
	}
}

// AddFile adds a file diff.
func (v *DiffViewer) AddFile(path, oldContent, newContent string) {
	renderer := NewDiffRenderer()
	added, deleted := renderer.countChanges(oldContent, newContent)

	v.diffs = append(v.diffs, FileDiff{
		Path:       path,
		OldContent: oldContent,
		NewContent: newContent,
		Added:      added,
		Deleted:    deleted,
	})
}

// SetDimensions sets the viewer dimensions.
func (v *DiffViewer) SetDimensions(width, height int) {
	v.width = width
	v.height = height
}

// Next moves to the next diff.
func (v *DiffViewer) Next() {
	if v.current < len(v.diffs)-1 {
		v.current++
	}
}

// Prev moves to the previous diff.
func (v *DiffViewer) Prev() {
	if v.current > 0 {
		v.current--
	}
}

// Current returns the current file diff.
func (v *DiffViewer) Current() *FileDiff {
	if v.current < 0 || v.current >= len(v.diffs) {
		return nil
	}
	return &v.diffs[v.current]
}

// HasNext returns true if there are more diffs.
func (v *DiffViewer) HasNext() bool {
	return v.current < len(v.diffs)-1
}

// HasPrev returns true if there are previous diffs.
func (v *DiffViewer) HasPrev() bool {
	return v.current > 0
}

// Render renders the current diff.
func (v *DiffViewer) Render() string {
	if len(v.diffs) == 0 {
		return v.styles.helpStyle.Render("No changes to display")
	}

	diff := v.diffs[v.current]
	renderer := NewDiffRenderer()
	renderer.SetWidth(v.width - 4)

	var content strings.Builder

	// Title with navigation
	title := v.renderTitle()
	content.WriteString(title)
	content.WriteString("\n")

	// Diff content
	diffView := NewDiffView(diff.Path, diff.OldContent, diff.NewContent)
	content.WriteString(diffView.Render())
	content.WriteString("\n")

	// Help
	help := v.renderHelp()
	content.WriteString(help)

	return v.styles.containerStyle.Render(content.String())
}

// renderTitle renders the title with navigation.
func (v *DiffViewer) renderTitle() string {
	var parts []string

	parts = append(parts, v.styles.titleStyle.Render(
		fmt.Sprintf("(%d/%d)", v.current+1, len(v.diffs))))

	if len(v.diffs) > 0 {
		diff := v.diffs[v.current]
		parts = append(parts, v.styles.titleStyle.Render(diff.Path))
	}

	return strings.Join(parts, " ")
}

// renderHelp renders the help text.
func (v *DiffViewer) renderHelp() string {
	var hints []string

	if v.HasPrev() {
		hints = append(hints, "↑/k prev")
	}
	if v.HasNext() {
		hints = append(hints, "↓/j next")
	}
	hints = append(hints, "y approve", "n reject")

	return v.styles.helpStyle.Render(strings.Join(hints, " • "))
}

// GetTotalStats returns total statistics for all diffs.
func (v *DiffViewer) GetTotalStats() (totalFiles, totalAdditions, totalDeletions int) {
	totalFiles = len(v.diffs)
	for _, diff := range v.diffs {
		totalAdditions += diff.Added
		totalDeletions += diff.Deleted
	}
	return
}

// IsEmpty returns true if there are no diffs.
func (v *DiffViewer) IsEmpty() bool {
	return len(v.diffs) == 0
}

// Summary returns a summary of all changes.
func (v *DiffViewer) Summary() string {
	files, adds, dels := v.GetTotalStats()

	var parts []string
	parts = append(parts, fmt.Sprintf("%d file%s", files, pluralize(files)))

	if adds > 0 {
		parts = append(parts, lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")).
			Render(fmt.Sprintf("+%d", adds)))
	}

	if dels > 0 {
		parts = append(parts, lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4444")).
			Render(fmt.Sprintf("-%d", dels)))
	}

	return strings.Join(parts, ", ")
}

// pluralize returns "s" if count != 1.
func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}