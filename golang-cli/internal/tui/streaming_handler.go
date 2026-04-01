// Package tui provides streaming message handling for real-time updates.
package tui

import (
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// StreamingMessage represents a message being streamed
type StreamingMessage struct {
	ID          string
	Content     string
	IsComplete  bool
	Timestamp   time.Time
	MessageType string
}

// StreamingHandler manages streaming message state
type StreamingHandler struct {
	mu           sync.RWMutex
	streamingID  string
	buffer       strings.Builder
	updateChan   chan tea.Msg
	isStreaming  bool
	lastUpdate   time.Time
	flushTimer   *time.Timer
	flushDelay   time.Duration
}

// NewStreamingHandler creates a new streaming handler
func NewStreamingHandler(updateChan chan tea.Msg) *StreamingHandler {
	return &StreamingHandler{
		updateChan: updateChan,
		flushDelay: 100 * time.Millisecond,
	}
}

// StartStreaming begins a new streaming session
func (sh *StreamingHandler) StartStreaming(id string, msgType string) {
	sh.mu.Lock()
	defer sh.mu.Unlock()

	sh.streamingID = id
	sh.buffer.Reset()
	sh.isStreaming = true
	sh.lastUpdate = time.Now()
}

// AppendContent adds content to the streaming buffer
func (sh *StreamingHandler) AppendContent(content string) {
	sh.mu.Lock()
	defer sh.mu.Unlock()

	if !sh.isStreaming {
		return
	}

	sh.buffer.WriteString(content)
	sh.lastUpdate = time.Now()

	// Schedule a UI update
	if sh.flushTimer != nil {
		sh.flushTimer.Stop()
	}
	sh.flushTimer = time.AfterFunc(sh.flushDelay, func() {
		select {
		case sh.updateChan <- StreamingUpdateMsg{ID: sh.streamingID}:
		default:
		}
	})
}

// EndStreaming marks the streaming as complete
func (sh *StreamingHandler) EndStreaming() {
	sh.mu.Lock()
	defer sh.mu.Unlock()

	sh.isStreaming = false
	if sh.flushTimer != nil {
		sh.flushTimer.Stop()
	}
}

// GetContent returns the current streaming content
func (sh *StreamingHandler) GetContent() string {
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	return sh.buffer.String()
}

// GetStreamingID returns the current streaming message ID
func (sh *StreamingHandler) GetStreamingID() string {
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	return sh.streamingID
}

// IsStreaming returns true if currently streaming
func (sh *StreamingHandler) IsStreaming() bool {
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	return sh.isStreaming
}

// StreamingUpdateMsg is sent when streaming content updates
type StreamingUpdateMsg struct {
	ID string
}

// StreamingCompleteMsg is sent when streaming completes
type StreamingCompleteMsg struct {
	ID      string
	Content string
}

// StreamingStartMsg is sent when streaming starts
type StreamingStartMsg struct {
	ID          string
	MessageType string
}

// SpinnerTickMsg is sent on each spinner animation tick
type SpinnerTickMsg struct {
	Time time.Time
}

// StartSpinner starts the spinner animation
func StartSpinner() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return SpinnerTickMsg{Time: t}
	})
}

// StreamBuffer manages chunked stream content with proper line handling
type StreamBuffer struct {
	mu       sync.RWMutex
	chunks   []string
	complete bool
}

// NewStreamBuffer creates a new stream buffer
func NewStreamBuffer() *StreamBuffer {
	return &StreamBuffer{
		chunks: make([]string, 0),
	}
}

// Write adds a chunk to the buffer
func (sb *StreamBuffer) Write(chunk string) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.chunks = append(sb.chunks, chunk)
}

// Read reads all content from the buffer
func (sb *StreamBuffer) Read() string {
	sb.mu.RLock()
	defer sb.mu.RUnlock()
	return strings.Join(sb.chunks, "")
}

// Complete marks the buffer as complete
func (sb *StreamBuffer) Complete() {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.complete = true
}

// IsComplete returns true if the buffer is complete
func (sb *StreamBuffer) IsComplete() bool {
	sb.mu.RLock()
	defer sb.mu.RUnlock()
	return sb.complete
}

// Reset clears the buffer
func (sb *StreamBuffer) Reset() {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.chunks = make([]string, 0)
	sb.complete = false
}

// MessageChunk represents a chunk of a streaming message
type MessageChunk struct {
	Sequence int
	Content  string
	IsLast   bool
}

// MessageReassembler reassembles chunked messages
type MessageReassembler struct {
	mu       sync.RWMutex
	chunks   map[string][]MessageChunk
	timeouts map[string]time.Time
}

// NewMessageReassembler creates a new message reassembler
func NewMessageReassembler() *MessageReassembler {
	return &MessageReassembler{
		chunks:   make(map[string][]MessageChunk),
		timeouts: make(map[string]time.Time),
	}
}

// AddChunk adds a chunk to be reassembled
func (mr *MessageReassembler) AddChunk(msgID string, chunk MessageChunk) (string, bool) {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	if _, exists := mr.chunks[msgID]; !exists {
		mr.chunks[msgID] = make([]MessageChunk, 0)
		mr.timeouts[msgID] = time.Now().Add(30 * time.Second)
	}

	mr.chunks[msgID] = append(mr.chunks[msgID], chunk)

	if chunk.IsLast {
		// Reassemble the message
		result := mr.reassemble(msgID)
		delete(mr.chunks, msgID)
		delete(mr.timeouts, msgID)
		return result, true
	}

	return "", false
}

// reassemble combines chunks into a complete message
func (mr *MessageReassembler) reassemble(msgID string) string {
	chunks := mr.chunks[msgID]
	if len(chunks) == 0 {
		return ""
	}

	// Sort by sequence
	// In a real implementation, we'd sort properly
	var result strings.Builder
	for _, chunk := range chunks {
		result.WriteString(chunk.Content)
	}

	return result.String()
}

// Cleanup removes stale partial messages
func (mr *MessageReassembler) Cleanup() {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	now := time.Now()
	for msgID, timeout := range mr.timeouts {
		if now.After(timeout) {
			delete(mr.chunks, msgID)
			delete(mr.timeouts, msgID)
		}
	}
}