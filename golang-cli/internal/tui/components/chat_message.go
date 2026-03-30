// Package components provides TUI components for the Cline CLI.
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/cline/cline/golang-cli/internal/tui"
)

// MessageType represents the type of chat message
type MessageType string

const (
	// MessageTypeUser represents a user message
	MessageTypeUser MessageType = "user"
	// MessageTypeAI represents an AI message
	MessageTypeAI MessageType = "ai"
	// MessageTypeSystem represents a system message
	MessageTypeSystem MessageType = "system"
	// MessageTypeError represents an error message
	MessageTypeError MessageType = "error"
	// MessageTypeToolUse represents a tool use message
	MessageTypeToolUse MessageType = "tool_use"
	// MessageTypeToolResult represents a tool result message
	MessageTypeToolResult MessageType = "tool_result"
	// MessageTypeAsk represents an ask message
	MessageTypeAsk MessageType = "ask"
)

// ChatMessage represents a single chat message with full rendering capabilities
type ChatMessage struct {
	// Content
	Type      MessageType
	Content   string
	Partial   bool
	Timestamp int64

	// Metadata
	ToolName   string
	ToolInput  map[string]interface{}
	ToolResult string
	Language   string

	// Rendering
	width    int
	renderer *tui.MarkdownRenderer

	// Styling
	userStyle       lipgloss.Style
	aiStyle         lipgloss.Style
	systemStyle     lipgloss.Style
	errorStyle      lipgloss.Style
	toolStyle       lipgloss.Style
	toolResultStyle lipgloss.Style
	askStyle        lipgloss.Style
	timestampStyle  lipgloss.Style
}

// NewChatMessage creates a new chat message
func NewChatMessage(msgType MessageType, content string) *ChatMessage {
	cm := &ChatMessage{
		Type:    msgType,
		Content: content,
		width:   80,
		userStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00D9FF")).
			Bold(true),
		aiStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")),
		systemStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#808080")).
			Italic(true),
		errorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4444")).
			Bold(true),
		toolStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")),
		toolResultStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A0A0A0")),
		askStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFB000")).
			Bold(true),
		timestampStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#606060")).
			Italic(true),
	}

	// Initialize markdown renderer
	renderer, _ := tui.NewMarkdownRenderer(
		tui.WithMarkdownStyle("dark"),
		tui.WithMarkdownWidth(100),
	)
	cm.renderer = renderer

	return cm
}

// SetWidth sets the message width
func (cm *ChatMessage) SetWidth(width int) {
	cm.width = width
	if cm.renderer != nil {
		cm.renderer.SetWidth(width - 10)
	}
}

// SetRenderer sets a custom markdown renderer
func (cm *ChatMessage) SetRenderer(renderer *tui.MarkdownRenderer) {
	cm.renderer = renderer
}

// SetToolInfo sets tool information for tool messages
func (cm *ChatMessage) SetToolInfo(name string, input map[string]interface{}) {
	cm.ToolName = name
	cm.ToolInput = input
}

// SetToolResult sets the tool result
func (cm *ChatMessage) SetToolResult(result string) {
	cm.ToolResult = result
}

// SetLanguage sets the language for code blocks
func (cm *ChatMessage) SetLanguage(lang string) {
	cm.Language = lang
}

// SetPartial marks the message as partial/streaming
func (cm *ChatMessage) SetPartial(partial bool) {
	cm.Partial = partial
}

// Render renders the chat message
func (cm *ChatMessage) Render() string {
	switch cm.Type {
	case MessageTypeUser:
		return cm.renderUserMessage()
	case MessageTypeAI:
		return cm.renderAIMessage()
	case MessageTypeSystem:
		return cm.renderSystemMessage()
	case MessageTypeError:
		return cm.renderErrorMessage()
	case MessageTypeToolUse:
		return cm.renderToolUseMessage()
	case MessageTypeToolResult:
		return cm.renderToolResultMessage()
	case MessageTypeAsk:
		return cm.renderAskMessage()
	default:
		return cm.renderSystemMessage()
	}
}

// renderUserMessage renders a user message
func (cm *ChatMessage) renderUserMessage() string {
	prefix := cm.userStyle.Render("You: ")
	content := cm.Content

	// Wrap content
	lines := strings.Split(content, "\n")
	var wrapped []string
	for _, line := range lines {
		wrapped = append(wrapped, wrapLine(line, cm.width-6))
	}
	content = strings.Join(wrapped, "\n")

	return prefix + content
}

// renderAIMessage renders an AI message with markdown support
func (cm *ChatMessage) renderAIMessage() string {
	prefix := cm.aiStyle.Render("Cline: ")
	content := cm.Content

	// Try to render as markdown if not partial
	if cm.renderer != nil && !cm.Partial {
		if rendered, err := cm.renderer.Render(content); err == nil {
			content = rendered
		} else {
			// Fall back to basic rendering
			content = cm.renderBasicMarkdown(content)
		}
	} else {
		// For partial content, use basic rendering
		content = cm.renderBasicMarkdown(content)
	}

	return prefix + content
}

// renderSystemMessage renders a system message
func (cm *ChatMessage) renderSystemMessage() string {
	return cm.systemStyle.Render(cm.Content)
}

// renderErrorMessage renders an error message
func (cm *ChatMessage) renderErrorMessage() string {
	prefix := cm.errorStyle.Render("Error: ")
	return prefix + cm.errorStyle.Render(cm.Content)
}

// renderToolUseMessage renders a tool use message
func (cm *ChatMessage) renderToolUseMessage() string {
	toolName := cm.ToolName
	if toolName == "" {
		toolName = "tool"
	}

	icon := "🔧"
	var details strings.Builder

	// Build tool input summary
	if cm.ToolInput != nil && len(cm.ToolInput) > 0 {
		for key, value := range cm.ToolInput {
			details.WriteString(fmt.Sprintf("  • %s: %v\n", key, value))
		}
	}

	content := fmt.Sprintf("%s Using %s...\n%s", icon, toolName, details.String())

	// Apply tool style to the whole message
	lines := strings.Split(content, "\n")
	var styled []string
	for _, line := range lines {
		styled = append(styled, cm.toolStyle.Render(line))
	}

	return strings.Join(styled, "\n")
}

// renderToolResultMessage renders a tool result message
func (cm *ChatMessage) renderToolResultMessage() string {
	result := cm.ToolResult
	if result == "" {
		result = cm.Content
	}

	// Truncate if too long
	maxLen := 200
	if len(result) > maxLen {
		result = result[:maxLen] + "..."
	}

	return cm.toolResultStyle.Render(fmt.Sprintf("  → %s", result))
}

// renderAskMessage renders an ask message
func (cm *ChatMessage) renderAskMessage() string {
	prefix := cm.askStyle.Render("Cline: ")
	content := cm.Content

	// Wrap content
	lines := strings.Split(content, "\n")
	var wrapped []string
	for _, line := range lines {
		wrapped = append(wrapped, wrapLine(line, cm.width-10))
	}
	content = strings.Join(wrapped, "\n")

	return prefix + content
}

// renderBasicMarkdown provides basic markdown formatting
func (cm *ChatMessage) renderBasicMarkdown(content string) string {
	// Simple markdown replacements
	content = strings.ReplaceAll(content, "**", "")
	content = strings.ReplaceAll(content, "*", "")
	content = strings.ReplaceAll(content, "`", "")

	// Handle code blocks
	if strings.Contains(content, "```") {
		parts := strings.Split(content, "```")
		var result strings.Builder
		for i, part := range parts {
			if i%2 == 1 {
				// This is a code block
				lines := strings.Split(part, "\n")
				if len(lines) > 0 {
					lang := lines[0]
					code := strings.Join(lines[1:], "\n")
					result.WriteString(cm.renderCodeBlock(code, lang))
				}
			} else {
				result.WriteString(part)
			}
		}
		content = result.String()
	}

	return content
}

// renderCodeBlock renders a code block with styling
func (cm *ChatMessage) renderCodeBlock(code, lang string) string {
	codeStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#2a2a2a")).
		Foreground(lipgloss.Color("#e0e0e0")).
		Padding(1, 2).
		MarginLeft(2).
		Width(cm.width - 10)

	if lang != "" {
		header := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Italic(true).
			Render(lang)
		return header + "\n" + codeStyle.Render(code)
	}

	return codeStyle.Render(code)
}

// GetHeight returns the estimated height of the rendered message
func (cm *ChatMessage) GetHeight() int {
	lines := strings.Split(cm.Content, "\n")
	height := len(lines)
	for _, line := range lines {
		if len(line) > cm.width {
			height += len(line) / cm.width
		}
	}
	return height + 1 // +1 for spacing
}

// wrapLine wraps a line to the specified width
func wrapLine(line string, width int) string {
	if len(line) <= width {
		return line
	}

	var result strings.Builder
	currentLen := 0
	words := strings.Fields(line)

	for i, word := range words {
		wordLen := len(word)
		if currentLen+wordLen+1 > width {
			if i > 0 {
				result.WriteString("\n")
			}
			result.WriteString(word)
			currentLen = wordLen
		} else {
			if i > 0 {
				result.WriteString(" ")
				currentLen++
			}
			result.WriteString(word)
			currentLen += wordLen
		}
	}

	return result.String()
}

// ChatMessageList manages a list of chat messages
type ChatMessageList struct {
	messages []*ChatMessage
	width    int
}

// NewChatMessageList creates a new chat message list
func NewChatMessageList() *ChatMessageList {
	return &ChatMessageList{
		messages: make([]*ChatMessage, 0),
		width:    80,
	}
}

// AddMessage adds a message to the list
func (cml *ChatMessageList) AddMessage(msg *ChatMessage) {
	msg.SetWidth(cml.width)
	cml.messages = append(cml.messages, msg)
}

// UpdateLast updates the last message (for streaming)
func (cml *ChatMessageList) UpdateLast(content string, partial bool) bool {
	if len(cml.messages) == 0 {
		return false
	}
	last := cml.messages[len(cml.messages)-1]
	last.Content = content
	last.Partial = partial
	return true
}

// GetMessages returns all messages
func (cml *ChatMessageList) GetMessages() []*ChatMessage {
	return cml.messages
}

// GetLast returns the last message
func (cml *ChatMessageList) GetLast() *ChatMessage {
	if len(cml.messages) == 0 {
		return nil
	}
	return cml.messages[len(cml.messages)-1]
}

// SetWidth sets the width for all messages
func (cml *ChatMessageList) SetWidth(width int) {
	cml.width = width
	for _, msg := range cml.messages {
		msg.SetWidth(width)
	}
}

// Render renders all messages
func (cml *ChatMessageList) Render() string {
	var result strings.Builder
	for i, msg := range cml.messages {
		result.WriteString(msg.Render())
		if i < len(cml.messages)-1 {
			result.WriteString("\n\n")
		}
	}
	return result.String()
}

// GetTotalHeight returns the total height of all messages
func (cml *ChatMessageList) GetTotalHeight() int {
	height := 0
	for _, msg := range cml.messages {
		height += msg.GetHeight()
	}
	return height
}

// Clear clears all messages
func (cml *ChatMessageList) Clear() {
	cml.messages = make([]*ChatMessage, 0)
}