// Package mode provides terminal mode detection and output streaming capabilities
// for the Cline CLI. It handles TTY detection, pipe/redirect identification,
// mode override mechanisms, and JSON streaming for long-running tasks.
package mode

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

var (
	// ErrStreamClosed is returned when attempting to write to a closed stream
	ErrStreamClosed = errors.New("stream is closed")
	// ErrStreamNotReady is returned when the stream is not ready for writing
	ErrStreamNotReady = errors.New("stream not ready")
	// ErrInvalidMessage is returned when a message fails validation
	ErrInvalidMessage = errors.New("invalid message")
)

// StreamMessage represents a JSON message in the stream.
// Each message is output as a single JSON line (JSONL format).
type StreamMessage struct {
	// Type indicates the message category (e.g., "text", "tool_use", "tool_result")
	Type string `json:"type"`

	// Content contains the message payload
	Content string `json:"content"`

	// Timestamp is when the message was created
	Timestamp time.Time `json:"timestamp"`

	// Partial indicates this is a partial/chunked message update
	Partial bool `json:"partial"`

	// SequenceNumber is used for ordering partial messages
	SequenceNumber int `json:"sequence_number,omitempty"`

	// MessageID uniquely identifies this message stream
	MessageID string `json:"message_id"`

	// Metadata contains optional additional data
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// JSONStreamer handles streaming JSON lines (JSONL) output for long-running tasks.
// It provides real-time message delivery with support for partial updates.
type JSONStreamer struct {
	// writer is the output destination
	writer io.Writer

	// bufferedWriter provides buffering for efficient output
	bufferedWriter *bufio.Writer

	// encoder handles JSON encoding
	encoder *json.Encoder

	// state management
	mu     sync.RWMutex
	closed bool
	ready  bool

	// flush management
	flushMu      sync.Mutex
	autoFlush    bool
	flushTicker  *time.Ticker
	flushStop    chan struct{}
	flushPending bool

	// partial message tracking
	partialMu      sync.RWMutex
	partialBuffers map[string]*partialBuffer

	// configuration
	config StreamConfig
}

// partialBuffer tracks partial message chunks for reassembly
type partialBuffer struct {
	chunks         []StreamMessage
	lastUpdate     time.Time
	sequenceNumber int
}

// StreamConfig contains configuration for the JSON streamer
type StreamConfig struct {
	// AutoFlush enables automatic flushing after each message
	AutoFlush bool

	// FlushInterval sets the maximum time between flushes when AutoFlush is true
	FlushInterval time.Duration

	// BufferSize sets the size of the output buffer
	BufferSize int

	// EnablePartialMessages enables support for partial/chunked messages
	EnablePartialMessages bool

	// PartialTimeout sets how long to wait for partial message completion
	PartialTimeout time.Duration

	// OnMessage is called when a message is successfully written
	OnMessage func(*StreamMessage)

	// OnError is called when an error occurs
	OnError func(error)
}

// DefaultStreamConfig returns a default stream configuration
func DefaultStreamConfig() StreamConfig {
	return StreamConfig{
		AutoFlush:             true,
		FlushInterval:         100 * time.Millisecond,
		BufferSize:            4096,
		EnablePartialMessages: true,
		PartialTimeout:        30 * time.Second,
	}
}

// NewJSONStreamer creates a new JSON streamer with the given output writer
func NewJSONStreamer(writer io.Writer, config *StreamConfig) *JSONStreamer {
	if writer == nil {
		panic("writer cannot be nil")
	}

	cfg := DefaultStreamConfig()
	if config != nil {
		if config.FlushInterval > 0 {
			cfg.FlushInterval = config.FlushInterval
		}
		if config.BufferSize > 0 {
			cfg.BufferSize = config.BufferSize
		}
		if config.PartialTimeout > 0 {
			cfg.PartialTimeout = config.PartialTimeout
		}
		cfg.AutoFlush = config.AutoFlush
		cfg.EnablePartialMessages = config.EnablePartialMessages
		cfg.OnMessage = config.OnMessage
		cfg.OnError = config.OnError
	}

	bufWriter := bufio.NewWriterSize(writer, cfg.BufferSize)
	encoder := json.NewEncoder(bufWriter)

	s := &JSONStreamer{
		writer:         writer,
		bufferedWriter: bufWriter,
		encoder:        encoder,
		autoFlush:      cfg.AutoFlush,
		flushStop:      make(chan struct{}),
		partialBuffers: make(map[string]*partialBuffer),
		config:         cfg,
		ready:          true,
	}

	// Start background flush goroutine if auto-flush is enabled
	if s.autoFlush && cfg.FlushInterval > 0 {
		s.flushTicker = time.NewTicker(cfg.FlushInterval)
		go s.flushLoop()
	}

	return s
}

// WriteMessage writes a complete message to the stream.
// The message is immediately encoded as JSON and written.
func (s *JSONStreamer) WriteMessage(msgType string, content string) error {
	msg := StreamMessage{
		Type:      msgType,
		Content:   content,
		Timestamp: time.Now().UTC(),
		Partial:   false,
		MessageID: generateMessageID(),
	}

	return s.writeMessageInternal(&msg)
}

// WritePartialMessage writes a partial message chunk to the stream.
// Partial messages are used for streaming content that arrives incrementally.
func (s *JSONStreamer) WritePartialMessage(msgType string, content string, messageID string, isLast bool) error {
	if !s.config.EnablePartialMessages {
		// If partial messages are disabled, treat as complete message
		if isLast {
			return s.WriteMessage(msgType, content)
		}
		return nil
	}

	s.partialMu.Lock()
	defer s.partialMu.Unlock()

	// Get or create partial buffer
	buf, exists := s.partialBuffers[messageID]
	if !exists {
		buf = &partialBuffer{
			chunks:     make([]StreamMessage, 0),
			lastUpdate: time.Now(),
		}
		s.partialBuffers[messageID] = buf
	}

	// Update buffer
	buf.lastUpdate = time.Now()
	buf.sequenceNumber++

	msg := StreamMessage{
		Type:           msgType,
		Content:        content,
		Timestamp:      time.Now().UTC(),
		Partial:        !isLast,
		SequenceNumber: buf.sequenceNumber,
		MessageID:      messageID,
	}

	// Store chunk for potential reassembly
	buf.chunks = append(buf.chunks, msg)

	return s.writeMessageInternal(&msg)
}

// writeMessageInternal handles the actual message writing with proper locking
func (s *JSONStreamer) writeMessageInternal(msg *StreamMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return ErrStreamClosed
	}

	if !s.ready {
		return ErrStreamNotReady
	}

	// Encode message as JSON line
	if err := s.encoder.Encode(msg); err != nil {
		s.handleError(fmt.Errorf("failed to encode message: %w", err))
		return err
	}

	// Mark flush as pending
	s.flushPending = true

	// Auto-flush if enabled
	if s.autoFlush {
		if err := s.flushUnlocked(); err != nil {
			s.handleError(fmt.Errorf("failed to flush: %w", err))
			return err
		}
	}

	// Notify callback
	if s.config.OnMessage != nil {
		s.config.OnMessage(msg)
	}

	return nil
}

// Flush explicitly flushes the output buffer.
// This should be called when immediate output is required.
func (s *JSONStreamer) Flush() error {
	s.flushMu.Lock()
	defer s.flushMu.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.flushUnlocked()
}

// flushUnlocked performs the actual flush operation.
// Caller must hold s.mu and s.flushMu locks.
func (s *JSONStreamer) flushUnlocked() error {
	if !s.flushPending {
		return nil
	}

	if err := s.bufferedWriter.Flush(); err != nil {
		return fmt.Errorf("flush failed: %w", err)
	}

	s.flushPending = false
	return nil
}

// flushLoop runs in a background goroutine to periodically flush output
func (s *JSONStreamer) flushLoop() {
	for {
		select {
		case <-s.flushTicker.C:
			if err := s.Flush(); err != nil {
				s.handleError(fmt.Errorf("auto-flush failed: %w", err))
			}
		case <-s.flushStop:
			return
		}
	}
}

// Close gracefully closes the stream, flushing any pending output
func (s *JSONStreamer) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.ready = false
	s.mu.Unlock()

	// Stop background flush
	if s.flushTicker != nil {
		s.flushTicker.Stop()
		close(s.flushStop)
	}

	// Final flush
	if err := s.Flush(); err != nil {
		s.handleError(fmt.Errorf("final flush failed: %w", err))
		return err
	}

	// Clean up partial message buffers
	s.partialMu.Lock()
	s.partialBuffers = make(map[string]*partialBuffer)
	s.partialMu.Unlock()

	return nil
}

// IsReady returns true if the stream is ready for writing
func (s *JSONStreamer) IsReady() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ready && !s.closed
}

// IsClosed returns true if the stream has been closed
func (s *JSONStreamer) IsClosed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.closed
}

// WriteRaw writes raw bytes directly to the output.
// This bypasses JSON encoding and should be used with caution.
func (s *JSONStreamer) WriteRaw(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return ErrStreamClosed
	}

	if !s.ready {
		return ErrStreamNotReady
	}

	_, err := s.bufferedWriter.Write(data)
	if err != nil {
		s.handleError(fmt.Errorf("raw write failed: %w", err))
		return err
	}

	s.flushPending = true

	if s.autoFlush {
		return s.flushUnlocked()
	}

	return nil
}

// ReassembleMessage reconstructs a complete message from partial chunks.
// Returns nil if the message is incomplete or not found.
func (s *JSONStreamer) ReassembleMessage(messageID string) *StreamMessage {
	s.partialMu.Lock()
	defer s.partialMu.Unlock()

	buf, exists := s.partialBuffers[messageID]
	if !exists {
		return nil
	}

	if len(buf.chunks) == 0 {
		return nil
	}

	// Combine all chunk content
	var fullContent string
	for _, chunk := range buf.chunks {
		fullContent += chunk.Content
	}

	// Create complete message from first chunk
	first := buf.chunks[0]
	return &StreamMessage{
		Type:           first.Type,
		Content:        fullContent,
		Timestamp:      first.Timestamp,
		Partial:        false,
		MessageID:      messageID,
		Metadata:       first.Metadata,
		SequenceNumber: buf.sequenceNumber,
	}
}

// CleanupPartialMessages removes stale partial message buffers
func (s *JSONStreamer) CleanupPartialMessages() int {
	s.partialMu.Lock()
	defer s.partialMu.Unlock()

	cleanupCount := 0
	now := time.Now()
	timeout := s.config.PartialTimeout

	for id, buf := range s.partialBuffers {
		if now.Sub(buf.lastUpdate) > timeout {
			delete(s.partialBuffers, id)
			cleanupCount++
		}
	}

	return cleanupCount
}

// GetPartialMessageCount returns the number of active partial message buffers
func (s *JSONStreamer) GetPartialMessageCount() int {
	s.partialMu.RLock()
	defer s.partialMu.RUnlock()
	return len(s.partialBuffers)
}

// handleError invokes the error callback if configured
func (s *JSONStreamer) handleError(err error) {
	if s.config.OnError != nil {
		s.config.OnError(err)
	}
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	return fmt.Sprintf("msg_%d_%d", time.Now().UnixNano(), time.Now().UnixMicro())
}

// Enhanced streaming methods for Phase 4

// WriteStructuredMessage writes a message with structured metadata
func (s *JSONStreamer) WriteStructuredMessage(msgType string, content string, metadata map[string]interface{}) error {
	msg := StreamMessage{
		Type:      msgType,
		Content:   content,
		Timestamp: time.Now().UTC(),
		Partial:   false,
		MessageID: generateMessageID(),
		Metadata:  metadata,
	}

	return s.writeMessageInternal(&msg)
}

// WriteError writes an error message to the stream
func (s *JSONStreamer) WriteError(err error) error {
	if err == nil {
		return nil
	}

	metadata := map[string]interface{}{
		"error_type": fmt.Sprintf("%T", err),
	}

	msg := StreamMessage{
		Type:      "error",
		Content:   err.Error(),
		Timestamp: time.Now().UTC(),
		Partial:   false,
		MessageID: generateMessageID(),
		Metadata:  metadata,
	}

	return s.writeMessageInternal(&msg)
}

// WriteProgress writes a progress update message
func (s *JSONStreamer) WriteProgress(operation string, current, total int, message string) error {
	metadata := map[string]interface{}{
		"operation": operation,
		"current":   current,
		"total":     total,
		"percent":   float64(current) / float64(total) * 100,
	}

	msg := StreamMessage{
		Type:      "progress",
		Content:   message,
		Timestamp: time.Now().UTC(),
		Partial:   false,
		MessageID: generateMessageID(),
		Metadata:  metadata,
	}

	return s.writeMessageInternal(&msg)
}

// WriteToolUse writes a tool use message
func (s *JSONStreamer) WriteToolUse(toolName string, params map[string]interface{}, messageID string) error {
	metadata := map[string]interface{}{
		"tool":   toolName,
		"params": params,
	}

	msg := StreamMessage{
		Type:      "tool_use",
		Content:   fmt.Sprintf("Using tool: %s", toolName),
		Timestamp: time.Now().UTC(),
		Partial:   false,
		MessageID: messageID,
		Metadata:  metadata,
	}

	return s.writeMessageInternal(&msg)
}

// WriteToolResult writes a tool result message
func (s *JSONStreamer) WriteToolResult(toolName string, result interface{}, success bool, messageID string) error {
	metadata := map[string]interface{}{
		"tool":    toolName,
		"success": success,
		"result":  result,
	}

	msg := StreamMessage{
		Type:      "tool_result",
		Content:   fmt.Sprintf("Tool %s completed", toolName),
		Timestamp: time.Now().UTC(),
		Partial:   false,
		MessageID: messageID,
		Metadata:  metadata,
	}

	return s.writeMessageInternal(&msg)
}

// WriteCheckpoint writes a checkpoint message
func (s *JSONStreamer) WriteCheckpoint(checkpointID string, description string) error {
	metadata := map[string]interface{}{
		"checkpoint_id": checkpointID,
	}

	msg := StreamMessage{
		Type:      "checkpoint",
		Content:   description,
		Timestamp: time.Now().UTC(),
		Partial:   false,
		MessageID: generateMessageID(),
		Metadata:  metadata,
	}

	return s.writeMessageInternal(&msg)
}

// ReadJSONStream reads JSON messages from a reader
func ReadJSONStream(reader io.Reader, handler func(*StreamMessage) error) error {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var msg StreamMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			return fmt.Errorf("failed to parse JSON line: %w", err)
		}

		if err := handler(&msg); err != nil {
			return err
		}
	}

	return scanner.Err()
}

// FormatJSONOutput formats output for JSON mode
func FormatJSONOutput(msgType string, content string, isPartial bool) string {
	msg := StreamMessage{
		Type:      msgType,
		Content:   content,
		Timestamp: time.Now().UTC(),
		Partial:   isPartial,
	}

	data, _ := json.Marshal(msg)
	return string(data)
}

// NewStdioJSONStreamer creates a JSON streamer for stdio with proper buffering
func NewStdioJSONStreamer() *JSONStreamer {
	return NewJSONStreamer(os.Stdout, &StreamConfig{
		AutoFlush:             true,
		FlushInterval:         50 * time.Millisecond,
		BufferSize:            4096,
		EnablePartialMessages: true,
		PartialTimeout:        30 * time.Second,
	})
}

// StreamWriter is an io.Writer implementation that wraps JSONStreamer
type StreamWriter struct {
	streamer *JSONStreamer
	msgType  string
}

// NewStreamWriter creates a new io.Writer that writes to the JSON stream
func (s *JSONStreamer) NewStreamWriter(msgType string) *StreamWriter {
	return &StreamWriter{
		streamer: s,
		msgType:  msgType,
	}
}

// Write implements io.Writer interface
func (w *StreamWriter) Write(p []byte) (n int, err error) {
	if err := w.streamer.WriteMessage(w.msgType, string(p)); err != nil {
		return 0, err
	}
	return len(p), nil
}

// StreamContext provides a context-aware streaming interface
type StreamContext struct {
	ctx      context.Context
	streamer *JSONStreamer
	cancel   context.CancelFunc
}

// NewStreamContext creates a new streaming context
func NewStreamContext(parent context.Context, writer io.Writer, config *StreamConfig) *StreamContext {
	ctx, cancel := context.WithCancel(parent)
	return &StreamContext{
		ctx:      ctx,
		streamer: NewJSONStreamer(writer, config),
		cancel:   cancel,
	}
}

// Streamer returns the underlying JSON streamer
func (sc *StreamContext) Streamer() *JSONStreamer {
	return sc.streamer
}

// Context returns the streaming context
func (sc *StreamContext) Context() context.Context {
	return sc.ctx
}

// Cancel cancels the streaming context
func (sc *StreamContext) Cancel() {
	sc.cancel()
}

// Close closes the stream context
func (sc *StreamContext) Close() error {
	sc.cancel()
	return sc.streamer.Close()
}

// WriteMessage writes a message with context awareness
func (sc *StreamContext) WriteMessage(msgType string, content string) error {
	select {
	case <-sc.ctx.Done():
		return sc.ctx.Err()
	default:
		return sc.streamer.WriteMessage(msgType, content)
	}
}

// WritePartialMessage writes a partial message with context awareness
func (sc *StreamContext) WritePartialMessage(msgType string, content string, messageID string, isLast bool) error {
	select {
	case <-sc.ctx.Done():
		return sc.ctx.Err()
	default:
		return sc.streamer.WritePartialMessage(msgType, content, messageID, isLast)
	}
}
