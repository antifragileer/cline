// Package components provides TUI components for the Cline CLI.
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// DiffLineType represents the type of a diff line
type DiffLineType int

const (
	// DiffLineTypeContext is an unchanged line
	DiffLineTypeContext DiffLineType = iota
	// DiffLineTypeAdded is an added line
	DiffLineTypeAdded
	// DiffLineTypeRemoved is a removed line
	DiffLineTypeRemoved
	// DiffLineTypeHeader is a header line
	DiffLineTypeHeader
)

// DiffLine represents a single line in a diff
type DiffLine struct {
	Type    DiffLineType
	Content string
	OldNum  int
	NewNum  int
}

// FileDiff represents a diff for a single file
type FileDiff struct {
	Path       string
	OldContent string
	NewContent string
	Lines      []DiffLine
	Additions  int
	Deletions  int
}

// DiffView displays file diffs with syntax highlighting
type DiffView struct {
	diffs        []FileDiff
	currentIndex int
	width        int
	height       int
	contextLines int
	showLineNums bool
}

// NewDiffView creates a new diff view
func NewDiffView() *DiffView {
	return &DiffView{
		diffs:        make([]FileDiff, 0),
		currentIndex: 0,
		width:        120,
		height:       40,
		contextLines: 3,
		showLineNums: true,
	}
}

// AddDiff adds a file diff to the view
func (dv *DiffView) AddDiff(path, oldContent, newContent string) {
	diff := dv.computeDiff(path, oldContent, newContent)
	dv.diffs = append(dv.diffs, diff)
}

// computeDiff computes the diff between two contents
func (dv *DiffView) computeDiff(path, oldContent, newContent string) FileDiff {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(oldContent, newContent, false)

	fileDiff := FileDiff{
		Path:       path,
		OldContent: oldContent,
		NewContent: newContent,
		Lines:      make([]DiffLine, 0),
	}

	oldLineNum := 1
	newLineNum := 1

	for _, diff := range diffs {
		lines := strings.Split(diff.Text, "\n")
		// Remove empty trailing line from split
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}

		for _, line := range lines {
			switch diff.Type {
			case diffmatchpatch.DiffDelete:
				fileDiff.Lines = append(fileDiff.Lines, DiffLine{
					Type:    DiffLineTypeRemoved,
					Content: line,
					OldNum:  oldLineNum,
					NewNum:  0,
				})
				oldLineNum++
				fileDiff.Deletions++

			case diffmatchpatch.DiffInsert:
				fileDiff.Lines = append(fileDiff.Lines, DiffLine{
					Type:    DiffLineTypeAdded,
					Content: line,
					OldNum:  0,
					NewNum:  newLineNum,
				})
				newLineNum++
				fileDiff.Additions++

			case diffmatchpatch.DiffEqual:
				fileDiff.Lines = append(fileDiff.Lines, DiffLine{
					Type:    DiffLineTypeContext,
					Content: line,
					OldNum:  oldLineNum,
					NewNum:  newLineNum,
				})
				oldLineNum++
				newLineNum++
			}
		}
	}

	return fileDiff
}

// SetDimensions sets the view dimensions
func (dv *DiffView) SetDimensions(width, height int) {
	dv.width = width
	dv.height = height
}

// Next moves to the next diff
func (dv *DiffView) Next() {
	if dv.currentIndex < len(dv.diffs)-1 {
		dv.currentIndex++
	}
}

// Prev moves to the previous diff
func (dv *DiffView) Prev() {
	if dv.currentIndex > 0 {
		dv.currentIndex--
	}
}

// GetCurrent returns the current file diff
func (dv *DiffView) GetCurrent() *FileDiff {
	if dv.currentIndex < 0 || dv.currentIndex >= len(dv.diffs) {
		return nil
	}
	return &dv.diffs[dv.currentIndex]
}

// HasNext returns true if there are more diffs
func (dv *DiffView) HasNext() bool {
	return dv.currentIndex < len(dv.diffs)-1
}

// HasPrev returns true if there are previous diffs
func (dv *DiffView) HasPrev() bool {
	return dv.currentIndex > 0
}

// GetStats returns total statistics
func (dv *DiffView) GetStats() (files, additions, deletions int) {
	files = len(dv.diffs)
	for _, diff := range dv.diffs {
		additions += diff.Additions
		deletions += diff.Deletions
	}
	return
}

// Render renders the current diff
func (dv *DiffView) Render() string {
	if len(dv.diffs) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")).
			Render("No changes to display")
	}

	diff := dv.diffs[dv.currentIndex]
	return dv.renderFileDiff(diff)
}

// renderFileDiff renders a single file diff
func (dv *DiffView) renderFileDiff(diff FileDiff) string {
	var result strings.Builder

	// Header
	header := dv.renderHeader(diff)
	result.WriteString(header)
	result.WriteString("\n")

	// Hunk header
	hunkHeader := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4")).
		Render(fmt.Sprintf("@@ -%d +%d @@", diff.Additions, diff.Deletions))
	result.WriteString(hunkHeader)
	result.WriteString("\n")

	// Diff content
	content := dv.renderDiffContent(diff.Lines)
	result.WriteString(content)

	return result.String()
}

// renderHeader renders the file header
func (dv *DiffView) renderHeader(diff FileDiff) string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4"))

	addedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FF00"))

	removedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF4444"))

	var parts []string
	parts = append(parts, headerStyle.Render(fmt.Sprintf("📄 %s", diff.Path)))

	if diff.Additions > 0 {
		parts = append(parts, addedStyle.Render(fmt.Sprintf("+%d", diff.Additions)))
	}

	if diff.Deletions > 0 {
		parts = append(parts, removedStyle.Render(fmt.Sprintf("-%d", diff.Deletions)))
	}

	// Navigation indicator
	if len(dv.diffs) > 1 {
		navStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#606060"))
		parts = append(parts, navStyle.Render(fmt.Sprintf("(%d/%d)", dv.currentIndex+1, len(dv.diffs))))
	}

	return strings.Join(parts, " ")
}

// renderDiffContent renders the diff lines
func (dv *DiffView) renderDiffContent(lines []DiffLine) string {
	var result strings.Builder

	// Styles
	addedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#003300")).
		Foreground(lipgloss.Color("#00FF00"))

	removedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#330000")).
		Foreground(lipgloss.Color("#FF4444"))

	contextStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#CCCCCC"))

	lineNumStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#606060"))

	for _, line := range lines {
		var styled string

		switch line.Type {
		case DiffLineTypeAdded:
			prefix := "+"
			if dv.showLineNums {
				prefix = fmt.Sprintf("+%4d", line.NewNum)
			}
			content := dv.truncateLine(line.Content, dv.width-12)
			styled = addedStyle.Render(prefix + " " + content)

		case DiffLineTypeRemoved:
			prefix := "-"
			if dv.showLineNums {
				prefix = fmt.Sprintf("-%4d", line.OldNum)
			}
			content := dv.truncateLine(line.Content, dv.width-12)
			styled = removedStyle.Render(prefix + " " + content)

		case DiffLineTypeContext:
			prefix := " "
			if dv.showLineNums {
				prefix = fmt.Sprintf(" %4d %4d", line.OldNum, line.NewNum)
			}
			content := dv.truncateLine(line.Content, dv.width-12)
			styled = contextStyle.Render(prefix+" "+content) + lineNumStyle.Render("")
		}

		result.WriteString(styled)
		result.WriteString("\n")
	}

	return result.String()
}

// truncateLine truncates a line to fit within width
func (dv *DiffView) truncateLine(line string, maxLen int) string {
	if len(line) <= maxLen {
		return line
	}
	return line[:maxLen-3] + "..."
}

// Summary returns a summary of all changes
func (dv *DiffView) Summary() string {
	files, adds, dels := dv.GetStats()

	var parts []string
	parts = append(parts, fmt.Sprintf("%d file%s", files, pluralS(files)))

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

	return strings.Join(parts, " ")
}

// pluralS returns "s" if count != 1
func pluralS(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

// IsEmpty returns true if there are no diffs
func (dv *DiffView) IsEmpty() bool {
	return len(dv.diffs) == 0
}

// Clear clears all diffs
func (dv *DiffView) Clear() {
	dv.diffs = make([]FileDiff, 0)
	dv.currentIndex = 0
}
