// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// SyntaxHighlighter provides syntax highlighting for code blocks
type SyntaxHighlighter struct {
	enabled bool
	theme   string
}

// NewSyntaxHighlighter creates a new syntax highlighter
func NewSyntaxHighlighter(enabled bool, theme string) *SyntaxHighlighter {
	if theme == "" {
		theme = "dark"
	}
	return &SyntaxHighlighter{
		enabled: enabled,
		theme:   theme,
	}
}

// Highlight applies syntax highlighting to code
func (sh *SyntaxHighlighter) Highlight(code, language string) string {
	if !sh.enabled {
		return code
	}

	// Simple terminal-based highlighting
	// In a full implementation, this would use a proper syntax highlighter
	// like chroma or treesitter, but for the CLI we'll use basic coloring

	lines := strings.Split(code, "\n")
	var highlighted []string

	for _, line := range lines {
		highlightedLine := sh.highlightLine(line, language)
		highlighted = append(highlighted, highlightedLine)
	}

	return strings.Join(highlighted, "\n")
}

// highlightLine applies basic highlighting to a single line
func (sh *SyntaxHighlighter) highlightLine(line, language string) string {
	if language == "" {
		return line
	}

	switch strings.ToLower(language) {
	case "go", "golang":
		return sh.highlightGo(line)
	case "typescript", "ts", "javascript", "js":
		return sh.highlightJavaScript(line)
	case "python", "py":
		return sh.highlightPython(line)
	case "json":
		return sh.highlightJSON(line)
	case "yaml", "yml":
		return sh.highlightYAML(line)
	case "bash", "sh", "shell", "zsh":
		return sh.highlightShell(line)
	case "markdown", "md":
		return sh.highlightMarkdown(line)
	default:
		return line
	}
}

// highlightGo applies Go syntax highlighting
func (sh *SyntaxHighlighter) highlightGo(line string) string {
	// Keywords
	keywords := []string{
		"package", "import", "func", "type", "struct", "interface",
		"map", "chan", "var", "const", "if", "else", "for", "range",
		"switch", "case", "default", "return", "defer", "go", "select",
		"break", "continue", "fallthrough", "goto", "nil", "true", "false",
	}

	result := line
	for _, kw := range keywords {
		// Simple word-based replacement
		result = sh.highlightKeyword(result, kw, lipgloss.Color("#ff79c6"))
	}

	// Types
	types := []string{"string", "int", "bool", "float64", "float32", "int64", "int32", "byte", "rune", "error"}
	for _, t := range types {
		result = sh.highlightKeyword(result, t, lipgloss.Color("#8be9fd"))
	}

	// Comments
	if strings.HasPrefix(strings.TrimSpace(line), "//") {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#6272a4")).Render(line)
	}

	return result
}

// highlightJavaScript applies JavaScript/TypeScript syntax highlighting
func (sh *SyntaxHighlighter) highlightJavaScript(line string) string {
	keywords := []string{
		"import", "export", "from", "const", "let", "var", "function",
		"class", "extends", "implements", "interface", "type", "enum",
		"if", "else", "for", "while", "do", "switch", "case", "default",
		"return", "break", "continue", "throw", "try", "catch", "finally",
		"async", "await", "new", "this", "super", "null", "undefined", "true", "false",
	}

	result := line
	for _, kw := range keywords {
		result = sh.highlightKeyword(result, kw, lipgloss.Color("#ff79c6"))
	}

	// Comments
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#6272a4")).Render(line)
	}

	return result
}

// highlightPython applies Python syntax highlighting
func (sh *SyntaxHighlighter) highlightPython(line string) string {
	keywords := []string{
		"import", "from", "as", "def", "class", "if", "elif", "else",
		"for", "while", "try", "except", "finally", "with", "return",
		"yield", "lambda", "and", "or", "not", "in", "is", "None", "True", "False",
	}

	result := line
	for _, kw := range keywords {
		result = sh.highlightKeyword(result, kw, lipgloss.Color("#ff79c6"))
	}

	// Comments
	if strings.Contains(line, "#") {
		parts := strings.SplitN(line, "#", 2)
		code := parts[0]
		comment := "#" + parts[1]
		return code + lipgloss.NewStyle().Foreground(lipgloss.Color("#6272a4")).Render(comment)
	}

	return result
}

// highlightJSON applies JSON syntax highlighting
func (sh *SyntaxHighlighter) highlightJSON(line string) string {
	// Strings
	result := sh.highlightStrings(line, lipgloss.Color("#f1fa8c"))

	// Numbers
	result = sh.highlightNumbers(result, lipgloss.Color("#bd93f9"))

	// Booleans and null
	booleans := []string{"true", "false", "null"}
	for _, b := range booleans {
		result = sh.highlightKeyword(result, b, lipgloss.Color("#ff79c6"))
	}

	return result
}

// highlightYAML applies YAML syntax highlighting
func (sh *SyntaxHighlighter) highlightYAML(line string) string {
	// Comments
	if strings.HasPrefix(strings.TrimSpace(line), "#") {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#6272a4")).Render(line)
	}

	// Keys (before colon)
	if strings.Contains(line, ":") {
		parts := strings.SplitN(line, ":", 2)
		key := lipgloss.NewStyle().Foreground(lipgloss.Color("#8be9fd")).Render(parts[0])
		return key + ":" + parts[1]
	}

	return line
}

// highlightShell applies Shell syntax highlighting
func (sh *SyntaxHighlighter) highlightShell(line string) string {
	// Comments
	if strings.HasPrefix(strings.TrimSpace(line), "#") {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#6272a4")).Render(line)
	}

	// Common commands
	commands := []string{"echo", "cd", "ls", "cat", "grep", "awk", "sed", "curl", "wget", "git", "docker", "kubectl"}
	result := line
	for _, cmd := range commands {
		if strings.HasPrefix(strings.TrimSpace(line), cmd) {
			result = sh.highlightKeyword(result, cmd, lipgloss.Color("#50fa7b"))
		}
	}

	return result
}

// highlightMarkdown applies Markdown syntax highlighting
func (sh *SyntaxHighlighter) highlightMarkdown(line string) string {
	// Headers
	if strings.HasPrefix(line, "#") {
		level := 0
		for _, c := range line {
			if c == '#' {
				level++
			} else {
				// Color based on header level
				colors := []string{"#ff79c6", "#bd93f9", "#8be9fd", "#50fa7b", "#f1fa8c", "#ffb86c"}
				color := colors[level-1]
				if level > len(colors) {
					color = colors[len(colors)-1]
				}
				return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Bold(true).Render(line)
			}
		}
	}

	// Bold
	if strings.Contains(line, "**") {
		parts := strings.Split(line, "**")
		for i := 1; i < len(parts); i += 2 {
			parts[i] = lipgloss.NewStyle().Bold(true).Render(parts[i])
		}
		return strings.Join(parts, "")
	}

	// Italic
	if strings.Contains(line, "*") {
		parts := strings.Split(line, "*")
		for i := 1; i < len(parts); i += 2 {
			parts[i] = lipgloss.NewStyle().Italic(true).Render(parts[i])
		}
		return strings.Join(parts, "")
	}

	// Code inline
	if strings.Contains(line, "`") {
		parts := strings.Split(line, "`")
		for i := 1; i < len(parts); i += 2 {
			parts[i] = lipgloss.NewStyle().Background(lipgloss.Color("#44475a")).Render(parts[i])
		}
		return strings.Join(parts, "")
	}

	return line
}

// highlightKeyword highlights a keyword in the line
func (sh *SyntaxHighlighter) highlightKeyword(line, keyword string, color lipgloss.Color) string {
	// Simple string replacement - in production, use regex for word boundaries
	styled := lipgloss.NewStyle().Foreground(color).Render(keyword)
	return strings.ReplaceAll(line, keyword, styled)
}

// highlightStrings highlights string literals
func (sh *SyntaxHighlighter) highlightStrings(line string, color lipgloss.Color) string {
	// Simple string highlighting - find quoted strings
	inString := false
	var result strings.Builder
	var currentString strings.Builder

	for _, char := range line {
		if char == '"' || char == '\'' {
			if inString {
				// End of string
				styled := lipgloss.NewStyle().Foreground(color).Render(currentString.String())
				result.WriteString(styled)
				result.WriteRune(char)
				currentString.Reset()
				inString = false
			} else {
				// Start of string
				if currentString.Len() > 0 {
					result.WriteString(currentString.String())
					currentString.Reset()
				}
				result.WriteRune(char)
				inString = true
			}
		} else {
			if inString {
				currentString.WriteRune(char)
			} else {
				result.WriteRune(char)
			}
		}
	}

	// Handle unterminated string
	if inString && currentString.Len() > 0 {
		result.WriteString(currentString.String())
	}

	return result.String()
}

// highlightNumbers highlights numeric literals
func (sh *SyntaxHighlighter) highlightNumbers(line string, color lipgloss.Color) string {
	// Simple number highlighting
	var result strings.Builder
	var currentNumber strings.Builder
	inNumber := false

	for i, char := range line {
		isDigit := (char >= '0' && char <= '9') || char == '.' || char == 'e' || char == 'E' || char == '+' || char == '-'

		// Handle negative sign (only at start of number)
		if char == '-' && i > 0 {
			prevChar := line[i-1]
			if prevChar != 'e' && prevChar != 'E' {
				isDigit = false
			}
		}

		if isDigit {
			if !inNumber {
				inNumber = true
			}
			currentNumber.WriteRune(char)
		} else {
			if inNumber && currentNumber.Len() > 0 {
				styled := lipgloss.NewStyle().Foreground(color).Render(currentNumber.String())
				result.WriteString(styled)
				currentNumber.Reset()
				inNumber = false
			}
			result.WriteRune(char)
		}
	}

	// Handle number at end of line
	if inNumber && currentNumber.Len() > 0 {
		styled := lipgloss.NewStyle().Foreground(color).Render(currentNumber.String())
		result.WriteString(styled)
	}

	return result.String()
}

// CodeBlock represents a code block with syntax highlighting
type CodeBlock struct {
	Language string
	Code     string
}

// Render renders a code block with syntax highlighting
func (cb *CodeBlock) Render(highlighter *SyntaxHighlighter) string {
	if highlighter == nil {
		return cb.Code
	}

	highlighted := highlighter.Highlight(cb.Code, cb.Language)

	// Add code block styling
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6272a4")).
		Padding(1).
		Background(lipgloss.Color("#282a36"))

	return style.Render(highlighted)
}

// ExtractCodeBlocks extracts code blocks from markdown text
func ExtractCodeBlocks(text string) []CodeBlock {
	var blocks []CodeBlock
	lines := strings.Split(text, "\n")

	inBlock := false
	var currentBlock strings.Builder
	var currentLang string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			if inBlock {
				// End of block
				blocks = append(blocks, CodeBlock{
					Language: currentLang,
					Code:     strings.TrimSuffix(currentBlock.String(), "\n"),
				})
				currentBlock.Reset()
				inBlock = false
				currentLang = ""
			} else {
				// Start of block
				inBlock = true
				currentLang = strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
			}
			continue
		}

		if inBlock {
			currentBlock.WriteString(line)
			currentBlock.WriteString("\n")
		}
	}

	return blocks
}
