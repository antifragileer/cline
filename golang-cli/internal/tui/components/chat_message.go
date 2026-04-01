// Package components provides TUI components for the Cline CLI.
package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/glamour"
	"github.com/muesli/reflow/wordwrap"
)

// MessageType represents the type of chat message
type MessageType string

const (
	// MessageTypeUser represents a user message
	MessageTypeUser MessageType = "user"
	// MessageTypeAssistant represents an assistant message
	MessageTypeAssistant MessageType = "assistant"
	// MessageTypeSystem represents a system message
	MessageTypeSystem MessageType = "system"
	// MessageTypeTool represents a tool output message
	MessageTypeTool MessageType = "tool"
	// MessageTypeError represents an error message
	MessageTypeError MessageType = "error"
	// MessageTypeStreaming represents a streaming message
	MessageTypeStreaming MessageType = "streaming"
)

// ChatMessage represents a single message in the chat
type ChatMessage struct {
	ID        string
	Type      MessageType
	Content   string
	Timestamp time.Time
	IsStreaming bool
	IsPartial   bool
	Metadata  map[string]string
}

// NewChatMessage creates a new chat message
func NewChatMessage(msgType MessageType, content string) *ChatMessage {
	return &ChatMessage{
		Type:      msgType,
		Content:   content,
		Timestamp: time.Now(),
		IsStreaming: false,
		IsPartial:   false,
	}
}

// GetHeight returns the estimated height of the message
func (m *ChatMessage) GetHeight() int {
	// Rough estimate: one line per 80 characters plus padding
	lines := len(m.Content)/80 + 1
	return lines + 2 // Add padding
}

// MessageStyles holds styles for different message types
type MessageStyles struct {
	UserStyle      lipgloss.Style
	AssistantStyle lipgloss.Style
	SystemStyle    lipgloss.Style
	ToolStyle      lipgloss.Style
	ErrorStyle     lipgloss.Style
	StreamingStyle lipgloss.Style
	TimestampStyle lipgloss.Style
}

// DefaultMessageStyles returns default message styles
func DefaultMessageStyles() MessageStyles {
	return MessageStyles{
		UserStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0E0E0")).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#4A90D9")).
			Padding(1, 2).
			MarginLeft(2).
			Width(60),

		AssistantStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0E0E0")).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#2ECC71")).
			Padding(1, 2).
			MarginLeft(0).
			Width(70),

		SystemStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Italic(true).
			Padding(0, 2),

		ToolStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F39C12")).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#F39C12")).
			Padding(1, 2),

		ErrorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E74C3C")).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#E74C3C")).
			Padding(1, 2).
			Bold(true),

		StreamingStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9B59B6")).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#9B59B6")).
			Padding(1, 2),

		TimestampStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Italic(true),
	}
}

// MessageRenderer handles rendering of chat messages
type MessageRenderer struct {
	styles      MessageStyles
	width       int
	glamour     bool
	renderer    *glamour.TermRenderer
}

// NewMessageRenderer creates a new message renderer
func NewMessageRenderer(width int, useGlamour bool) *MessageRenderer {
	mr := &MessageRenderer{
		styles:   DefaultMessageStyles(),
		width:    width,
		glamour:  useGlamour,
	}

	if useGlamour {
		var err error
		mr.renderer, err = glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(width-10),
		)
		if err != nil {
			mr.glamour = false
		}
	}

	return mr
}

// SetWidth updates the renderer width
func (mr *MessageRenderer) SetWidth(width int) {
	mr.width = width
	if mr.glamour && mr.renderer != nil {
		var err error
		mr.renderer, err = glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(width-10),
		)
		if err != nil {
			mr.glamour = false
		}
	}
}

// Render renders a single chat message
func (mr *MessageRenderer) Render(msg ChatMessage) string {
	var style lipgloss.Style
	var prefix string

	switch msg.Type {
	case MessageTypeUser:
		style = mr.styles.UserStyle
		prefix = "You"
	case MessageTypeAssistant:
		style = mr.styles.AssistantStyle
		prefix = "Cline"
	case MessageTypeSystem:
		style = mr.styles.SystemStyle
		prefix = "System"
	case MessageTypeTool:
		style = mr.styles.ToolStyle
		prefix = "Tool"
	case MessageTypeError:
		style = mr.styles.ErrorStyle
		prefix = "Error"
	case MessageTypeStreaming:
		style = mr.styles.StreamingStyle
		prefix = "Cline"
	default:
		style = mr.styles.SystemStyle
		prefix = "Unknown"
	}

	// Render content with markdown if enabled
	content := mr.renderContent(msg.Content, msg.Type)

	// Add streaming indicator
	if msg.IsStreaming {
		content += "\n" + mr.renderStreamingIndicator()
	}

	// Build the message
	timestamp := mr.styles.TimestampStyle.Render(msg.Timestamp.Format("15:04:05"))
	header := fmt.Sprintf("%s %s", timestamp, lipgloss.NewStyle().Bold(true).Render(prefix))

	message := style.Render(content)
	
	return lipgloss.JoinVertical(lipgloss.Left, header, message)
}

// renderContent renders message content with optional markdown
func (mr *MessageRenderer) renderContent(content string, msgType MessageType) string {
	if !mr.glamour || mr.renderer == nil {
		// Plain text rendering with word wrap
		return wordwrap.String(content, mr.width-10)
	}

	// Use glamour for markdown rendering
	rendered, err := mr.renderer.Render(content)
	if err != nil {
		return wordwrap.String(content, mr.width-10)
	}

	return rendered
}

// renderStreamingIndicator renders a streaming indicator
func (mr *MessageRenderer) renderStreamingIndicator() string {
	indicators := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	// In a real implementation, this would animate
	indicator := indicators[0]
	
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9B59B6")).
		Render(indicator + " Thinking...")
}

// RenderSimple renders a simple text message without styling
func (mr *MessageRenderer) RenderSimple(content string, msgType MessageType) string {
	var prefix string
	var color string

	switch msgType {
	case MessageTypeUser:
		prefix = "You: "
		color = "#4A90D9"
	case MessageTypeAssistant:
		prefix = "Cline: "
		color = "#2ECC71"
	case MessageTypeSystem:
		prefix = "System: "
		color = "#888888"
	case MessageTypeError:
		prefix = "Error: "
		color = "#E74C3C"
	default:
		prefix = ""
		color = "#E0E0E0"
	}

	style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	wrapped := wordwrap.String(content, mr.width-10)
	
	return style.Render(prefix + wrapped)
}

// MessageList manages a list of chat messages
type MessageList struct {
	messages []ChatMessage
	renderer *MessageRenderer
	maxSize  int
}

// NewMessageList creates a new message list
func NewMessageList(width int, maxSize int) *MessageList {
	if maxSize <= 0 {
		maxSize = 1000
	}

	return &MessageList{
		messages: make([]ChatMessage, 0),
		renderer: NewMessageRenderer(width, true),
		maxSize:  maxSize,
	}
}

// AddMessage adds a message to the list
func (ml *MessageList) AddMessage(msg ChatMessage) {
	ml.messages = append(ml.messages, msg)

	// Trim if exceeding max size
	if len(ml.messages) > ml.maxSize {
		ml.messages = ml.messages[len(ml.messages)-ml.maxSize:]
	}
}

// UpdateMessage updates an existing message by ID
func (ml *MessageList) UpdateMessage(id string, content string, isStreaming bool) bool {
	for i := range ml.messages {
		if ml.messages[i].ID == id {
			ml.messages[i].Content = content
			ml.messages[i].IsStreaming = isStreaming
			return true
		}
	}
	return false
}

// GetMessages returns all messages
func (ml *MessageList) GetMessages() []ChatMessage {
	return ml.messages
}

// GetLastMessage returns the last message
func (ml *MessageList) GetLastMessage() *ChatMessage {
	if len(ml.messages) == 0 {
		return nil
	}
	return &ml.messages[len(ml.messages)-1]
}

// Render renders all messages
func (ml *MessageList) Render() string {
	var rendered []string
	
	for _, msg := range ml.messages {
		rendered = append(rendered, ml.renderer.Render(msg))
	}

	return strings.Join(rendered, "\n\n")
}

// RenderLast renders only the last N messages
func (ml *MessageList) RenderLast(n int) string {
	if n <= 0 || n > len(ml.messages) {
		n = len(ml.messages)
	}

	start := len(ml.messages) - n
	var rendered []string
	
	for i := start; i < len(ml.messages); i++ {
		rendered = append(rendered, ml.renderer.Render(ml.messages[i]))
	}

	return strings.Join(rendered, "\n\n")
}

// Clear clears all messages
func (ml *MessageList) Clear() {
	ml.messages = make([]ChatMessage, 0)
}

// SetWidth updates the renderer width
func (ml *MessageList) SetWidth(width int) {
	ml.renderer.SetWidth(width)
}

// Count returns the number of messages
func (ml *MessageList) Count() int {
	return len(ml.messages)
}

// FindMessageByType finds the most recent message of a given type
func (ml *MessageList) FindMessageByType(msgType MessageType) *ChatMessage {
	for i := len(ml.messages) - 1; i >= 0; i-- {
		if ml.messages[i].Type == msgType {
			return &ml.messages[i]
		}
	}
	return nil
}

// ChatMessageList provides a simpler message list for the Chat component
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

// SetWidth sets the width for rendering
func (cml *ChatMessageList) SetWidth(width int) {
	cml.width = width
}

// AddMessage adds a message to the list
func (cml *ChatMessageList) AddMessage(msg *ChatMessage) {
	cml.messages = append(cml.messages, msg)
}

// GetMessages returns all messages
func (cml *ChatMessageList) GetMessages() []*ChatMessage {
	return cml.messages
}

// UpdateLast updates the last message
func (cml *ChatMessageList) UpdateLast(content string, partial bool) bool {
	if len(cml.messages) == 0 {
		return false
	}
	last := cml.messages[len(cml.messages)-1]
	last.Content = content
	last.IsPartial = partial
	return true
}

// Clear clears all messages
func (cml *ChatMessageList) Clear() {
	cml.messages = make([]*ChatMessage, 0)
}

// GetTotalHeight returns the total height
func (cml *ChatMessageList) GetTotalHeight() int {
	total := 0
	for _, msg := range cml.messages {
		total += msg.GetHeight()
	}
	return total
}

// Render renders all messages
func (cml *ChatMessageList) Render() string {
	var parts []string
	for _, msg := range cml.messages {
		parts = append(parts, msg.Render())
	}
	return strings.Join(parts, "\n\n")
}

// Render renders a single chat message with simple styling
func (m *ChatMessage) Render() string {
	var prefix string
	var color string

	switch m.Type {
	case MessageTypeUser:
		prefix = "You"
		color = "#4A90D9"
	case MessageTypeAssistant:
		prefix = "Cline"
		color = "#2ECC71"
	case MessageTypeSystem:
		prefix = "System"
		color = "#888888"
	case MessageTypeError:
		prefix = "Error"
		color = "#E74C3C"
	case MessageTypeTool:
		prefix = "Tool"
		color = "#F39C12"
	default:
		prefix = "Unknown"
		color = "#E0E0E0"
	}

	timestamp := m.Timestamp.Format("15:04:05")
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Italic(true)
	prefixStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(color)).
		Bold(true)
	contentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E0E0E0"))

	header := headerStyle.Render(timestamp) + " " + prefixStyle.Render(prefix)
	content := contentStyle.Render(m.Content)

	return header + "\n" + content
}
