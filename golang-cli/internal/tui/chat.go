// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// BubbleStyle defines the styling for message bubbles.
type BubbleStyle struct {
	// UserBubble is the style for user message bubbles.
	UserBubble lipgloss.Style
	// AIBubble is the style for AI message bubbles.
	AIBubble lipgloss.Style
	// ErrorBubble is the style for error message bubbles.
	ErrorBubble lipgloss.Style
	// ToolBubble is the style for tool-related message bubbles.
	ToolBubble lipgloss.Style
	// SystemBubble is the style for system message bubbles.
	SystemBubble lipgloss.Style
	// Width is the maximum width of the bubbles.
	Width int
	// Margin is the margin around bubbles.
	Margin int
}

// DefaultBubbleStyle returns the default bubble styling.
func DefaultBubbleStyle() BubbleStyle {
	return BubbleStyle{
		UserBubble: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#0088ff")).
			Background(lipgloss.Color("#004488")).
			Foreground(lipgloss.Color("#ffffff")).
			Padding(1, 2).
			MarginLeft(10).
			Width(60),

		AIBubble: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#666666")).
			Background(lipgloss.Color("#2a2a2a")).
			Foreground(lipgloss.Color("#ffffff")).
			Padding(1, 2).
			MarginLeft(0).
			Width(70),

		ErrorBubble: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#ff0000")).
			Background(lipgloss.Color("#440000")).
			Foreground(lipgloss.Color("#ffcccc")).
			Padding(1, 2).
			MarginLeft(0).
			Width(70),

		ToolBubble: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#ffaa00")).
			Background(lipgloss.Color("#332200")).
			Foreground(lipgloss.Color("#ffddaa")).
			Padding(1, 2).
			MarginLeft(5).
			Width(65),

		SystemBubble: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Italic(true).
			MarginLeft(2).
			Width(70),

		Width:  70,
		Margin: 1,
	}
}

// ChatRenderer handles rendering of chat messages with bubbles.
type ChatRenderer struct {
	style           BubbleStyle
	markdown        *MarkdownRenderer
	store           *MessageStore
	width           int
	streamingMsg    *Message
	streamingBuffer strings.Builder
	mu              sync.RWMutex
	showTimestamps  bool
	showAvatars     bool
	avatarStyle     lipgloss.Style
}

// ChatOption configures the ChatRenderer.
type ChatOption func(*ChatRenderer)

// WithStyle sets the bubble style.
func WithStyle(style BubbleStyle) ChatOption {
	return func(c *ChatRenderer) {
		c.style = style
	}
}

// WithWidth sets the chat width.
func WithWidth(width int) ChatOption {
	return func(c *ChatRenderer) {
		c.width = width
	}
}

// WithTimestamps enables/disables timestamps.
func WithTimestamps(show bool) ChatOption {
	return func(c *ChatRenderer) {
		c.showTimestamps = show
	}
}

// WithAvatars enables/disables avatars.
func WithAvatars(show bool) ChatOption {
	return func(c *ChatRenderer) {
		c.showAvatars = show
	}
}

// NewChatRenderer creates a new chat renderer.
func NewChatRenderer(opts ...ChatOption) (*ChatRenderer, error) {
	markdown, err := NewMarkdownRenderer(WithMarkdownWidth(68))
	if err != nil {
		return nil, fmt.Errorf("failed to create markdown renderer: %w", err)
	}

	cr := &ChatRenderer{
		style:          DefaultBubbleStyle(),
		markdown:       markdown,
		store:          NewMessageStore(),
		width:          80,
		showTimestamps: false,
		showAvatars:    true,
		avatarStyle: lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1),
	}

	for _, opt := range opts {
		opt(cr)
	}

	// Update markdown renderer width based on chat width
	cr.markdown.SetWidth(cr.width - 10)

	return cr, nil
}

// RenderMessage renders a single message as a bubble.
func (c *ChatRenderer) RenderMessage(msg *Message) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.renderMessageInternal(msg)
}

func (c *ChatRenderer) renderMessageInternal(msg *Message) (string, error) {
	var bubble strings.Builder

	// Add avatar/indicator
	if c.showAvatars {
		avatar := c.renderAvatar(msg)
		bubble.WriteString(avatar + "\n")
	}

	// Add timestamp if enabled
	if c.showTimestamps {
		timestamp := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Render(msg.Timestamp.Format("15:04:05"))
		bubble.WriteString(timestamp + "\n")
	}

	// Get the appropriate style and content
	content, err := c.renderContent(msg)
	if err != nil {
		return "", err
	}

	style := c.getStyleForMessage(msg)
	rendered := style.Render(content)

	bubble.WriteString(rendered)

	// Add partial indicator for streaming messages
	if msg.Partial {
		bubble.WriteString("\n" + c.renderTypingIndicator())
	}

	return bubble.String(), nil
}

// renderContent renders the message content based on type.
func (c *ChatRenderer) renderContent(msg *Message) (string, error) {
	switch msg.Type {
	case MessageTypeToolUse:
		return c.renderToolUse(msg)
	case MessageTypeToolResult:
		return c.renderToolResult(msg)
	case MessageTypeError:
		return c.renderError(msg)
	default:
		return c.renderMarkdown(msg.Content)
	}
}

// renderMarkdown renders markdown content.
func (c *ChatRenderer) renderMarkdown(content string) (string, error) {
	if content == "" {
		return "", nil
	}
	return c.markdown.Render(content)
}

// renderToolUse renders a tool use message.
func (c *ChatRenderer) renderToolUse(msg *Message) (string, error) {
	var result strings.Builder

	result.WriteString(lipgloss.NewStyle().Bold(true).Render("🔧 Using tool: ") + msg.ToolName + "\n")

	if msg.ToolInput != nil && len(msg.ToolInput) > 0 {
		result.WriteString("\nArguments:\n")
		for key, value := range msg.ToolInput {
			result.WriteString(fmt.Sprintf("  %s: %v\n", key, value))
		}
	}

	if msg.Content != "" {
		result.WriteString("\n" + msg.Content)
	}

	return result.String(), nil
}

// renderToolResult renders a tool result message.
func (c *ChatRenderer) renderToolResult(msg *Message) (string, error) {
	var result strings.Builder

	result.WriteString(lipgloss.NewStyle().Bold(true).Render("✓ Result from ") + msg.ToolName + "\n")

	if msg.ToolResult != "" {
		// Try to detect if it's code
		if msg.Language != "" {
			highlighted, err := c.markdown.RenderCode(msg.ToolResult, msg.Language)
			if err == nil {
				result.WriteString("\n" + highlighted)
			} else {
				result.WriteString("\n" + msg.ToolResult)
			}
		} else {
			result.WriteString("\n" + msg.ToolResult)
		}
	}

	return result.String(), nil
}

// renderError renders an error message.
func (c *ChatRenderer) renderError(msg *Message) (string, error) {
	var result strings.Builder

	result.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ff6666")).Render("✗ Error") + "\n")
	result.WriteString(msg.Content)

	return result.String(), nil
}

// renderAvatar renders the avatar/indicator for a message.
func (c *ChatRenderer) renderAvatar(msg *Message) string {
	switch msg.Type {
	case MessageTypeUser:
		return c.avatarStyle.Copy().
			Foreground(lipgloss.Color("#0088ff")).
			Render("You")
	case MessageTypeSay, MessageTypeAsk:
		return c.avatarStyle.Copy().
			Foreground(lipgloss.Color("#00ff88")).
			Render("Cline")
	case MessageTypeError:
		return c.avatarStyle.Copy().
			Foreground(lipgloss.Color("#ff0000")).
			Render("Error")
	case MessageTypeToolUse, MessageTypeToolResult:
		return c.avatarStyle.Copy().
			Foreground(lipgloss.Color("#ffaa00")).
			Render("Tool")
	default:
		return c.avatarStyle.Copy().
			Foreground(lipgloss.Color("#888888")).
			Render("System")
	}
}

// getStyleForMessage returns the appropriate style for a message type.
func (c *ChatRenderer) getStyleForMessage(msg *Message) lipgloss.Style {
	switch msg.Type {
	case MessageTypeUser:
		return c.style.UserBubble
	case MessageTypeError:
		return c.style.ErrorBubble
	case MessageTypeToolUse, MessageTypeToolResult:
		return c.style.ToolBubble
	case MessageTypeSay, MessageTypeAsk:
		return c.style.AIBubble
	default:
		return c.style.SystemBubble
	}
}

// renderTypingIndicator renders a typing indicator for streaming messages.
func (c *ChatRenderer) renderTypingIndicator() string {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frame := frames[int(time.Now().UnixMilli()/100)%len(frames)]

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00ff88")).
		Render(frame + " typing...")
}

// RenderChat renders all messages in the store.
func (c *ChatRenderer) RenderChat() (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result strings.Builder
	messages := c.store.GetAll()

	for i, msg := range messages {
		rendered, err := c.renderMessageInternal(msg)
		if err != nil {
			return "", err
		}

		result.WriteString(rendered)

		// Add spacing between messages
		if i < len(messages)-1 {
			result.WriteString(strings.Repeat("\n", c.style.Margin))
		}
	}

	return result.String(), nil
}

// AddMessage adds a message to the chat.
func (c *ChatRenderer) AddMessage(msg *Message) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store.Add(msg)
}

// StartStreaming starts a new streaming message.
func (c *ChatRenderer) StartStreaming(msgType MessageType) *Message {
	c.mu.Lock()
	defer c.mu.Unlock()

	msg := NewMessage(msgType, "")
	msg.SetPartial(true)
	c.streamingMsg = msg
	c.streamingBuffer.Reset()
	c.store.Add(msg)

	return msg
}

// UpdateStreaming updates the current streaming message.
func (c *ChatRenderer) UpdateStreaming(content string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.streamingMsg == nil {
		return
	}

	c.streamingBuffer.WriteString(content)
	c.streamingMsg.Content = c.streamingBuffer.String()
	c.store.UpdateLast(c.streamingMsg)
}

// EndStreaming finalizes the current streaming message.
func (c *ChatRenderer) EndStreaming() *Message {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.streamingMsg == nil {
		return nil
	}

	c.streamingMsg.Partial = false
	c.store.UpdateLast(c.streamingMsg)

	msg := c.streamingMsg
	c.streamingMsg = nil
	c.streamingBuffer.Reset()

	return msg
}

// IsStreaming returns true if currently streaming a message.
func (c *ChatRenderer) IsStreaming() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.streamingMsg != nil
}

// GetStreamingContent returns the current streaming content.
func (c *ChatRenderer) GetStreamingContent() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.streamingBuffer.String()
}

// Clear clears all messages.
func (c *ChatRenderer) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store.Clear()
	c.streamingMsg = nil
	c.streamingBuffer.Reset()
}

// GetMessageCount returns the number of messages.
func (c *ChatRenderer) GetMessageCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.store.Len()
}

// GetLastMessage returns the last message.
func (c *ChatRenderer) GetLastMessage() (*Message, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.store.GetLast()
}

// SetWidth updates the chat width.
func (c *ChatRenderer) SetWidth(width int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.width = width
	c.markdown.SetWidth(width - 10)

	// Update styles with new width
	c.style.UserBubble = c.style.UserBubble.Width(width - 20)
	c.style.AIBubble = c.style.AIBubble.Width(width - 10)
	c.style.ErrorBubble = c.style.ErrorBubble.Width(width - 10)
	c.style.ToolBubble = c.style.ToolBubble.Width(width - 15)
}

// RenderAsString renders the entire chat as a string.
func (c *ChatRenderer) RenderAsString() (string, error) {
	return c.RenderChat()
}

// RenderMessageType renders all messages of a specific type.
func (c *ChatRenderer) RenderMessageType(msgType MessageType) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result strings.Builder
	messages := c.store.Filter(msgType)

	for _, msg := range messages {
		rendered, err := c.renderMessageInternal(msg)
		if err != nil {
			return "", err
		}
		result.WriteString(rendered + "\n")
	}

	return result.String(), nil
}

// StreamingRenderer provides a callback-based streaming interface.
type StreamingRenderer struct {
	renderer    *ChatRenderer
	onUpdate    func(string)
	stopChan    chan struct{}
	updateChan  chan string
	mu          sync.Mutex
	isRunning   bool
}

// NewStreamingRenderer creates a new streaming renderer.
func NewStreamingRenderer(renderer *ChatRenderer, onUpdate func(string)) *StreamingRenderer {
	return &StreamingRenderer{
		renderer:   renderer,
		onUpdate:   onUpdate,
		stopChan:   make(chan struct{}),
		updateChan: make(chan string, 100),
	}
}

// Start begins the streaming render loop.
func (s *StreamingRenderer) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return
	}

	s.isRunning = true

	go func() {
		for {
			select {
			case <-s.stopChan:
				return
			case content := <-s.updateChan:
				s.renderer.UpdateStreaming(content)
				if s.onUpdate != nil {
					chat, err := s.renderer.RenderChat()
					if err == nil {
						s.onUpdate(chat)
					}
				}
			}
		}
	}()
}

// Stop stops the streaming render loop.
func (s *StreamingRenderer) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return
	}

	close(s.stopChan)
	s.isRunning = false
}

// Write implements io.Writer for streaming content.
func (s *StreamingRenderer) Write(p []byte) (n int, err error) {
	s.updateChan <- string(p)
	return len(p), nil
}

// IsRunning returns true if the streaming renderer is running.
func (s *StreamingRenderer) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.isRunning
}