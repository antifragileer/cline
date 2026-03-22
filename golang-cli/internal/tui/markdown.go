// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// MarkdownRenderer handles markdown rendering with syntax highlighting.
type MarkdownRenderer struct {
	glamourRenderer *glamour.TermRenderer
	style           string
	width           int
	useGlamour      bool
}

// MarkdownOption configures the MarkdownRenderer.
type MarkdownOption func(*MarkdownRenderer)

// WithMarkdownStyle sets the glamour style.
func WithMarkdownStyle(style string) MarkdownOption {
	return func(r *MarkdownRenderer) {
		r.style = style
	}
}

// WithMarkdownWidth sets the rendering width.
func WithMarkdownWidth(width int) MarkdownOption {
	return func(r *MarkdownRenderer) {
		r.width = width
	}
}

// NewMarkdownRenderer creates a new markdown renderer.
func NewMarkdownRenderer(opts ...MarkdownOption) (*MarkdownRenderer, error) {
	r := &MarkdownRenderer{
		style:      "dark",
		width:      80,
		useGlamour: true,
	}

	for _, opt := range opts {
		opt(r)
	}

	if r.useGlamour {
		renderer, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(r.width),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create glamour renderer: %w", err)
		}
		r.glamourRenderer = renderer
	}

	return r, nil
}

// Render renders markdown content with syntax highlighting.
func (r *MarkdownRenderer) Render(content string) (string, error) {
	if !r.useGlamour || r.glamourRenderer == nil {
		return r.RenderBasic(content), nil
	}

	out, err := r.glamourRenderer.Render(content)
	if err != nil {
		return "", fmt.Errorf("failed to render markdown: %w", err)
	}

	return strings.TrimSpace(out), nil
}

// RenderStream renders partial markdown content for streaming display.
func (r *MarkdownRenderer) RenderStream(content string) (string, error) {
	// For streaming content, we use basic rendering to avoid flickering
	// and re-rendering issues with glamour
	return r.RenderBasic(content), nil
}

// RenderBasic provides basic markdown rendering without glamour.
func (r *MarkdownRenderer) RenderBasic(content string) string {
	// Simple formatting for streaming content
	content = strings.ReplaceAll(content, "\r\n", "\n")
	return content
}

// RenderCode renders a code block with syntax highlighting.
func (r *MarkdownRenderer) RenderCode(code, language string) (string, error) {
	if language == "" {
		language = "text"
	}

	// Get the lexer for the language
	lexer := lexers.Get(language)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	// Create formatter with terminal colors
	formatter := formatters.Get("terminal")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	// Tokenize the code
	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return "", fmt.Errorf("failed to tokenize code: %w", err)
	}

	// Format the tokens
	var buf strings.Builder
	style := styles.Get(r.style)
	if style == nil {
		style = styles.Fallback
	}

	err = formatter.Format(&buf, style, iterator)
	if err != nil {
		return "", fmt.Errorf("failed to format code: %w", err)
	}

	// Wrap in a styled code block
	codeStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#444444")).
		Padding(1, 2).
		Width(r.width - 4)

	return codeStyle.Render(buf.String()), nil
}

// RenderInlineCode renders inline code.
func (r *MarkdownRenderer) RenderInlineCode(code string) string {
	style := lipgloss.NewStyle().
		Background(lipgloss.Color("#333333")).
		Foreground(lipgloss.Color("#ffffff")).
		Padding(0, 1)

	return style.Render(code)
}

// RenderBlockquote renders a blockquote.
func (r *MarkdownRenderer) RenderBlockquote(content string) string {
	lines := strings.Split(content, "\n")
	var styledLines []string
	for _, line := range lines {
		styledLines = append(styledLines, "> "+line)
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.ThickBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("#666666")).
		PaddingLeft(1).
		Foreground(lipgloss.Color("#aaaaaa"))

	return style.Render(strings.Join(styledLines, "\n"))
}

// RenderList renders a list item.
func (r *MarkdownRenderer) RenderList(items []string, ordered bool) string {
	var result strings.Builder
	for i, item := range items {
		prefix := "• "
		if ordered {
			prefix = fmt.Sprintf("%d. ", i+1)
		}
		result.WriteString(prefix + item + "\n")
	}
	return result.String()
}

// RenderTable renders a simple markdown table.
func (r *MarkdownRenderer) RenderTable(headers []string, rows [][]string) string {
	if len(headers) == 0 || len(rows) == 0 {
		return ""
	}

	var result strings.Builder

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Write header
	result.WriteString("| ")
	for i, h := range headers {
		result.WriteString(padRight(h, widths[i]) + " | ")
	}
	result.WriteString("\n")

	// Write separator
	result.WriteString("|")
	for _, w := range widths {
		result.WriteString(strings.Repeat("-", w+2) + "|")
	}
	result.WriteString("\n")

	// Write rows
	for _, row := range rows {
		result.WriteString("| ")
		for i, cell := range row {
			if i < len(widths) {
				result.WriteString(padRight(cell, widths[i]) + " | ")
			}
		}
		result.WriteString("\n")
	}

	return result.String()
}

// RenderLink renders a hyperlink.
func (r *MarkdownRenderer) RenderLink(text, url string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#0088ff")).
		Underline(true).
		Render(text) + fmt.Sprintf(" (%s)", url)
}

// RenderBold renders bold text.
func (r *MarkdownRenderer) RenderBold(text string) string {
	return lipgloss.NewStyle().
		Bold(true).
		Render(text)
}

// RenderItalic renders italic text.
func (r *MarkdownRenderer) RenderItalic(text string) string {
	return lipgloss.NewStyle().
		Italic(true).
		Render(text)
}

// RenderHeading renders a heading.
func (r *MarkdownRenderer) RenderHeading(level int, text string) string {
	colors := []string{
		"#ffffff",
		"#eeeeee",
		"#dddddd",
		"#cccccc",
		"#bbbbbb",
		"#aaaaaa",
	}

	color := "#aaaaaa"
	if level > 0 && level <= len(colors) {
		color = colors[level-1]
	}

	sizes := []int{2, 2, 1, 1, 1, 1}
	margin := 1
	if level > 0 && level <= len(sizes) {
		margin = sizes[level-1]
	}

	prefix := strings.Repeat("#", level) + " "

	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color(color)).
		Bold(true).
		MarginTop(margin).
		MarginBottom(margin)

	return style.Render(prefix + text)
}

// HighlightSyntax applies syntax highlighting to code using chroma.
func HighlightSyntax(code, language, styleName string) (string, error) {
	if styleName == "" {
		styleName = "monokai"
	}

	lexer := lexers.Get(language)
	if lexer == nil {
		lexer = lexers.Analyse(code)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	formatter := formatters.Get("terminal")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	style := styles.Get(styleName)
	if style == nil {
		style = styles.Fallback
	}

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	err = formatter.Format(&buf, style, iterator)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// RenderWithChroma renders code with specific chroma style and formatter.
func RenderWithChroma(code, language string, style *chroma.Style, formatter chroma.Formatter) (string, error) {
	lexer := lexers.Get(language)
	if lexer == nil {
		lexer = lexers.Analyse(code)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	err = formatter.Format(&buf, style, iterator)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// padRight pads a string to the right to reach the specified width.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// WriteTo writes the rendered content to a writer.
func (r *MarkdownRenderer) WriteTo(w io.Writer, content string) (int64, error) {
	rendered, err := r.Render(content)
	if err != nil {
		return 0, err
	}
	n, err := w.Write([]byte(rendered))
	return int64(n), err
}

// SetWidth updates the rendering width.
func (r *MarkdownRenderer) SetWidth(width int) {
	r.width = width
	if r.glamourRenderer != nil {
		// Recreate renderer with new width
		newRenderer, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(width),
		)
		if err == nil {
			r.glamourRenderer = newRenderer
		}
	}
}

// AvailableStyles returns a list of available glamour styles.
func AvailableStyles() []string {
	return []string{
		"ascii",
		"dark",
		"dracula",
		"light",
		"notty",
		"pink",
		"tokyo-night",
	}
}