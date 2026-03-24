package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Styles holds the styling configuration for the TUI.
type Styles struct {
	// BorderColor is the color used for borders.
	BorderColor string

	// TitleColor is the color used for the title.
	TitleColor string

	// TextColor is the color used for regular text.
	TextColor string

	// ErrorColor is the color used for error messages.
	ErrorColor string

	// UseANSI indicates whether to use ANSI color codes.
	UseANSI bool
}

// DefaultStyles returns the default styling configuration.
func DefaultStyles() Styles {
	return Styles{
		BorderColor: "cyan",
		TitleColor:  "yellow",
		TextColor:   "white",
		ErrorColor:  "red",
		UseANSI:     true,
	}
}

// View implements the bubbletea.Model interface.
// It renders the current state of the model.
func (m Model) View() string {
	if m.mode == ModePlain {
		return m.plainView()
	}
	return m.tuiView()
}

// tuiView renders the TUI view with full styling.
func (m Model) tuiView() string {
	if !m.ready {
		return "Initializing..."
	}

	var b strings.Builder

	// Render header
	b.WriteString(m.renderHeader())

	// Render main content area
	b.WriteString(m.renderContent())

	// Render footer with help text
	b.WriteString(m.renderFooter())

	return b.String()
}

// plainView renders the plain text view without styling.
func (m Model) plainView() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}
	return m.content + "\n"
}

// renderHeader renders the header section.
func (m Model) renderHeader() string {
	if m.title == "" {
		return ""
	}

	var b strings.Builder
	styles := DefaultStyles()

	// Create top border
	width := m.dimensions.Width
	topBorder := strings.Repeat("═", width-2)

	if styles.UseANSI {
		b.WriteString(fmt.Sprintf("\033[36m╔%s╗\033[0m\n", topBorder))
		centeredTitle := centerText(m.title, width-2)
		b.WriteString(fmt.Sprintf("\033[36m║\033[33m%s\033[36m║\033[0m\n", centeredTitle))
		b.WriteString(fmt.Sprintf("\033[36m╚%s╝\033[0m\n", topBorder))
	} else {
		b.WriteString(fmt.Sprintf("+%s+\n", strings.Repeat("-", width-2)))
		centeredTitle := centerText(m.title, width-2)
		b.WriteString(fmt.Sprintf("|%s|\n", centeredTitle))
		b.WriteString(fmt.Sprintf("+%s+\n", strings.Repeat("-", width-2)))
	}

	return b.String()
}

// renderContent renders the main content area.
func (m Model) renderContent() string {
	var b strings.Builder
	styles := DefaultStyles()

	// Calculate available height for content
	headerHeight := 4 // 3 lines for header + 1 blank
	footerHeight := 3 // 2 lines for footer + 1 blank
	availableHeight := m.dimensions.Height - headerHeight - footerHeight
	if availableHeight < 1 {
		availableHeight = 1
	}

	// Render error if present
	if m.err != nil {
		errorMsg := fmt.Sprintf("Error: %v", m.err)
		if styles.UseANSI {
			b.WriteString(fmt.Sprintf("\033[31m%s\033[0m\n", errorMsg))
		} else {
			b.WriteString(errorMsg + "\n")
		}
		availableHeight--
	}

	// Render content
	contentLines := wrapText(m.content, m.dimensions.Width-4)
	for i, line := range contentLines {
		if i >= availableHeight {
			break
		}
		if styles.UseANSI {
			b.WriteString(fmt.Sprintf("\033[37m  %s\033[0m\n", line))
		} else {
			b.WriteString("  " + line + "\n")
		}
	}

	// Fill remaining space
	linesRendered := len(contentLines)
	if m.err != nil {
		linesRendered++
	}
	for i := linesRendered; i < availableHeight; i++ {
		b.WriteString("\n")
	}

	return b.String()
}

// renderFooter renders the footer with help text.
func (m Model) renderFooter() string {
	var b strings.Builder
	styles := DefaultStyles()

	width := m.dimensions.Width
	helpText := "q/ctrl+c: quit"

	if styles.UseANSI {
		bottomBorder := strings.Repeat("═", width-2)
		b.WriteString(fmt.Sprintf("\033[36m╔%s╗\033[0m\n", bottomBorder))
		padding := width - len(helpText) - 4
		if padding < 0 {
			padding = 0
		}
		b.WriteString(fmt.Sprintf("\033[36m║\033[37m  %s%s\033[36m║\033[0m\n", helpText, strings.Repeat(" ", padding)))
		b.WriteString(fmt.Sprintf("\033[36m╚%s╝\033[0m\n", bottomBorder))
	} else {
		bottomBorder := strings.Repeat("-", width-2)
		b.WriteString(fmt.Sprintf("+%s+\n", bottomBorder))
		padding := width - len(helpText) - 4
		if padding < 0 {
			padding = 0
		}
		b.WriteString(fmt.Sprintf("|  %s%s|\n", helpText, strings.Repeat(" ", padding)))
		b.WriteString(fmt.Sprintf("+%s+\n", bottomBorder))
	}

	return b.String()
}

// centerText centers text within the given width.
func centerText(text string, width int) string {
	if len(text) >= width {
		return text[:width]
	}
	padding := (width - len(text)) / 2
	return strings.Repeat(" ", padding) + text + strings.Repeat(" ", width-len(text)-padding)
}

// wrapText wraps text to fit within the given width.
func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	var lines []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	currentLine := words[0]
	for _, word := range words[1:] {
		if len(currentLine)+1+len(word) <= width {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}
	lines = append(lines, currentLine)

	return lines
}

// RenderPlain renders content in plain mode without TUI styling.
func RenderPlain(content string) string {
	return content + "\n"
}

// RenderError renders an error message.
func RenderError(err error, useANSI bool) string {
	if err == nil {
		return ""
	}
	if useANSI {
		return fmt.Sprintf("\033[31mError: %v\033[0m\n", err)
	}
	return fmt.Sprintf("Error: %v\n", err)
}

// renderMessage renders a single message
func renderMessage(m Model, msg Message) string {
	return msg.Content
}

// renderChatView renders the chat view
func renderChatView(m Model) string {
	var result strings.Builder
	for _, msg := range m.messages {
		result.WriteString(renderMessage(m, msg))
		result.WriteString("\n")
	}
	return result.String()
}

// renderInputArea renders the input area
func renderInputArea(m Model) string {
	return m.textInput.View()
}

// renderWelcomeView renders the welcome view
func renderWelcomeView(m Model) string {
	return "Welcome to Cline!\nPress 'n' to start a new task, 'q' to quit."
}

// renderRecentTasks renders the recent tasks list
func renderRecentTasks(m Model) string {
	var result strings.Builder
	result.WriteString("Recent Tasks:\n")
	for _, task := range m.taskHistory {
		result.WriteString(fmt.Sprintf("  - %s: %s\n", task.ID, task.Description))
	}
	return result.String()
}

// highlightCode highlights code with syntax highlighting
func highlightCode(m Model, code, language string) string {
	return code
}

// getWelcomeStyle returns the style for the welcome screen
func getWelcomeStyle(m Model) lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7D56F4"))
}

// getLogoStyle returns the style for the logo
func getLogoStyle(m Model) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4")).
		Bold(true)
}

// getMenuStyle returns the style for menu items
func getMenuStyle(m Model) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E0E0E0"))
}

// getSelectedMenuStyle returns the style for selected menu items
func getSelectedMenuStyle(m Model) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4")).
		Bold(true).
		Background(lipgloss.Color("#2a2a2a"))
}

// getTaskItemStyle returns the style for task items
func getTaskItemStyle(m Model) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E0E0E0"))
}

// extractCodeBlocks extracts code blocks from content
func extractCodeBlocks(m Model, content string) []CodeBlock {
	return []CodeBlock{}
}
