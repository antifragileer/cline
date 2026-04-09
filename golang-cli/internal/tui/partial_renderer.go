// Package tui provides terminal UI components for the Cline CLI.
package tui

import (
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// PartialRenderer handles rendering of partial/streaming messages
type PartialRenderer struct {
	mu           sync.RWMutex
	currentMsg   *Message
	buffer       strings.Builder
	isStreaming  bool
	lastUpdate   time.Time
	styles       PartialRendererStyles
	maxBufferLen int
	onComplete   func(Message)
}

// PartialRendererStyles holds styles for partial rendering
type PartialRendererStyles struct {
	streamingStyle lipgloss.Style
	cursorStyle    lipgloss.Style
	completeStyle  lipgloss.Style
}

// DefaultPartialRendererStyles returns default styles
func DefaultPartialRendererStyles() PartialRendererStyles {
	return PartialRendererStyles{
		streamingStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(White)),

		cursorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(PrimaryBlue)).
			Blink(true),

		completeStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(White)),
	}
}

// NewPartialRenderer creates a new partial renderer
func NewPartialRenderer() *PartialRenderer {
	return &PartialRenderer{
		styles:       DefaultPartialRendererStyles(),
		maxBufferLen: 10000, // 10KB max buffer
		lastUpdate:   time.Now(),
	}
}

// StartStreaming begins a new streaming session
func (pr *PartialRenderer) StartStreaming(msgType MessageType, metadata map[string]interface{}) *Message {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	pr.isStreaming = true
	pr.buffer.Reset()
	pr.lastUpdate = time.Now()

	msg := &Message{
		ID:        generateID(),
		Type:      msgType,
		Content:   "",
		Partial:   true,
		Timestamp: time.Now(),
		Metadata:  metadata,
	}

	pr.currentMsg = msg
	return msg
}

// AppendChunk appends a content chunk to the current stream
func (pr *PartialRenderer) AppendChunk(chunk string) bool {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if !pr.isStreaming || pr.currentMsg == nil {
		return false
	}

	// Check buffer size
	if pr.buffer.Len()+len(chunk) > pr.maxBufferLen {
		// Truncate if needed
		remaining := pr.maxBufferLen - pr.buffer.Len()
		if remaining > 0 {
			chunk = chunk[:remaining]
		} else {
			return false
		}
	}

	pr.buffer.WriteString(chunk)
	pr.currentMsg.Content = pr.buffer.String()
	pr.lastUpdate = time.Now()

	return true
}

// CompleteStream marks the current stream as complete
func (pr *PartialRenderer) CompleteStream() *Message {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if !pr.isStreaming || pr.currentMsg == nil {
		return nil
	}

	pr.isStreaming = false
	pr.currentMsg.Partial = false
	pr.currentMsg.Timestamp = time.Now()

	// Call completion callback if set
	if pr.onComplete != nil {
		pr.onComplete(*pr.currentMsg)
	}

	return pr.currentMsg
}

// CancelStream cancels the current streaming session
func (pr *PartialRenderer) CancelStream() {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	pr.isStreaming = false
	pr.buffer.Reset()
	pr.currentMsg = nil
}

// IsStreaming returns true if currently streaming
func (pr *PartialRenderer) IsStreaming() bool {
	pr.mu.RLock()
	defer pr.mu.RUnlock()
	return pr.isStreaming
}

// GetCurrentMessage returns the current streaming message
func (pr *PartialRenderer) GetCurrentMessage() *Message {
	pr.mu.RLock()
	defer pr.mu.RUnlock()
	return pr.currentMsg
}

// GetCurrentContent returns the current buffer content
func (pr *PartialRenderer) GetCurrentContent() string {
	pr.mu.RLock()
	defer pr.mu.RUnlock()
	return pr.buffer.String()
}

// GetRenderedContent returns the rendered content with cursor
func (pr *PartialRenderer) GetRenderedContent() string {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	content := pr.buffer.String()

	if pr.isStreaming {
		// Add blinking cursor indicator
		content += "▌"
	}

	return content
}

// SetOnComplete sets the completion callback
func (pr *PartialRenderer) SetOnComplete(fn func(Message)) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.onComplete = fn
}

// GetLastUpdateTime returns when the last chunk was received
func (pr *PartialRenderer) GetLastUpdateTime() time.Time {
	pr.mu.RLock()
	defer pr.mu.RUnlock()
	return pr.lastUpdate
}

// IsStale returns true if no updates received recently
func (pr *PartialRenderer) IsStale(timeout time.Duration) bool {
	pr.mu.RLock()
	defer pr.mu.RUnlock()
	return time.Since(pr.lastUpdate) > timeout
}

// ProcessMessage handles a message, updating state for partial messages
func (pr *PartialRenderer) ProcessMessage(msg Message) Message {
	// If this is a partial message and we're not streaming, start streaming
	if msg.Partial && !pr.IsStreaming() {
		pr.StartStreaming(msg.Type, msg.Metadata)
		if msg.Content != "" {
			pr.AppendChunk(msg.Content)
		}
		return *pr.GetCurrentMessage()
	}

	// If this is a partial message and we are streaming, update
	if msg.Partial && pr.IsStreaming() {
		pr.AppendChunk(msg.Content)
		return *pr.GetCurrentMessage()
	}

	// If this is a complete message and we're streaming, complete
	if !msg.Partial && pr.IsStreaming() {
		pr.AppendChunk(msg.Content)
		return *pr.CompleteStream()
	}

	// Not a streaming message, return as-is
	return msg
}

// StreamingMessageView returns a view suitable for displaying in the TUI
func (pr *PartialRenderer) StreamingMessageView(msg Message, width int) string {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	content := msg.Content

	// Apply word wrapping if needed
	if width > 0 {
		content = wrapText(content, width)
	}

	// Add streaming indicator if partial
	if msg.Partial {
		content += "\n" + pr.styles.cursorStyle.Render("▌ streaming...")
	}

	return pr.styles.streamingStyle.Render(content)
}

// CompleteMessageView returns a view for a complete message
func (pr *PartialRenderer) CompleteMessageView(msg Message, width int) string {
	content := msg.Content

	// Apply word wrapping if needed
	if width > 0 {
		content = wrapText(content, width)
	}

	return pr.styles.completeStyle.Render(content)
}

// wrapText wraps text to fit within a given width
func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	lines := strings.Split(text, "\n")
	var result strings.Builder

	for _, line := range lines {
		for len(line) > width {
			// Find break point
			breakPoint := width
			for breakPoint > 0 && line[breakPoint] != ' ' {
				breakPoint--
			}
			if breakPoint == 0 {
				breakPoint = width
			}

			result.WriteString(line[:breakPoint])
			result.WriteString("\n")
			line = strings.TrimLeft(line[breakPoint:], " ")
		}
		result.WriteString(line)
		result.WriteString("\n")
	}

	return strings.TrimSuffix(result.String(), "\n")
}

// PartialMessageStore stores partial messages for resumption
type PartialMessageStore struct {
	mu       sync.RWMutex
	messages map[string]PartialMessageState
}

// PartialMessageState stores the state of a partial message
type PartialMessageState struct {
	ID          string
	Content     string
	Type        MessageType
	Metadata    map[string]interface{}
	StartedAt   time.Time
	LastUpdated time.Time
}

// NewPartialMessageStore creates a new partial message store
func NewPartialMessageStore() *PartialMessageStore {
	return &PartialMessageStore{
		messages: make(map[string]PartialMessageState),
	}
}

// Save saves a partial message state
func (s *PartialMessageStore) Save(state PartialMessageState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages[state.ID] = state
}

// Get retrieves a partial message state
func (s *PartialMessageStore) Get(id string) (PartialMessageState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.messages[id]
	return state, ok
}

// Delete removes a partial message state
func (s *PartialMessageStore) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.messages, id)
}

// GetAll returns all partial message states
func (s *PartialMessageStore) GetAll() []PartialMessageState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]PartialMessageState, 0, len(s.messages))
	for _, state := range s.messages {
		result = append(result, state)
	}
	return result
}

// CleanupStale removes stale partial messages
func (s *PartialMessageStore) CleanupStale(maxAge time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	removed := 0

	for id, state := range s.messages {
		if now.Sub(state.LastUpdated) > maxAge {
			delete(s.messages, id)
			removed++
		}
	}

	return removed
}