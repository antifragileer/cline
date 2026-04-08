// Package grpc provides message routing for different proto types.
package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/cline/cline/golang-cli/internal/generated/cline/cline"
)

// MessageType represents the type of message being routed
type MessageType string

const (
	// Task-related message types
	MessageTypeTaskStart     MessageType = "task_start"
	MessageTypeTaskComplete  MessageType = "task_complete"
	MessageTypeTaskError     MessageType = "task_error"
	MessageTypeTaskCancelled MessageType = "task_cancelled"

	// Content message types
	MessageTypeText       MessageType = "text"
	MessageTypeCode       MessageType = "code"
	MessageTypeThinking   MessageType = "thinking"
	MessageTypeToolUse    MessageType = "tool_use"
	MessageTypeToolResult MessageType = "tool_result"

	// UI interaction types
	MessageTypeAsk        MessageType = "ask"
	MessageTypeSay        MessageType = "say"
	MessageTypeApproval   MessageType = "approval"
	MessageTypeCheckpoint MessageType = "checkpoint"

	// System message types
	MessageTypeSystem      MessageType = "system"
	MessageTypeError       MessageType = "error"
	MessageTypeInfo        MessageType = "info"
	MessageTypeStreamStart MessageType = "stream_start"
	MessageTypeStreamEnd   MessageType = "stream_end"
	MessageTypeStreamChunk MessageType = "stream_chunk"
)

// RoutedMessage represents a message that has been routed and typed
type RoutedMessage struct {
	Type      MessageType
	Subtype   string
	Payload   []byte
	Metadata  map[string]interface{}
	Timestamp time.Time
	TaskID    string
	Sequence  int64
}

// MessageHandler is a function that handles routed messages
type MessageHandler func(msg *RoutedMessage) error

// MessageRouter routes proto messages to appropriate handlers
type MessageRouter struct {
	handlers       map[MessageType][]MessageHandler
	defaultHandler MessageHandler
	mu             sync.RWMutex
	middleware     []MiddlewareFunc
}

// MiddlewareFunc is a function that wraps message handling
type MiddlewareFunc func(next MessageHandler) MessageHandler

// NewMessageRouter creates a new message router
func NewMessageRouter() *MessageRouter {
	return &MessageRouter{
		handlers:   make(map[MessageType][]MessageHandler),
		middleware: make([]MiddlewareFunc, 0),
	}
}

// RegisterHandler registers a handler for a specific message type
func (r *MessageRouter) RegisterHandler(msgType MessageType, handler MessageHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.handlers[msgType] == nil {
		r.handlers[msgType] = make([]MessageHandler, 0)
	}
	r.handlers[msgType] = append(r.handlers[msgType], handler)
}

// RegisterDefaultHandler registers a handler for unhandled message types
func (r *MessageRouter) RegisterDefaultHandler(handler MessageHandler) {
	r.defaultHandler = handler
}

// Use adds middleware to the router
func (r *MessageRouter) Use(mw MiddlewareFunc) {
	r.middleware = append(r.middleware, mw)
}

// Route routes a proto message to the appropriate handler(s)
func (r *MessageRouter) Route(msg *Message) error {
	if msg == nil {
		return fmt.Errorf("cannot route nil message")
	}

	routed, err := r.parseMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to parse message: %w", err)
	}

	r.mu.RLock()
	handlers := r.handlers[routed.Type]
	r.mu.RUnlock()

	if len(handlers) == 0 && r.defaultHandler != nil {
		handlers = []MessageHandler{r.defaultHandler}
	}

	// Apply middleware chain
	for i := len(r.middleware) - 1; i >= 0; i-- {
		for j, h := range handlers {
			handlers[j] = r.middleware[i](h)
		}
	}

	// Execute handlers
	var lastErr error
	for _, handler := range handlers {
		if err := handler(routed); err != nil {
			lastErr = err
			// Continue to next handler even if one fails
		}
	}

	return lastErr
}

// parseMessage converts a generic message to a routed message
func (r *MessageRouter) parseMessage(msg *Message) (*RoutedMessage, error) {
	routed := &RoutedMessage{
		Type:      MessageType(msg.Type),
		Payload:   msg.Payload,
		Timestamp: time.Unix(msg.Timestamp, 0),
		Sequence:  int64(msg.Sequence),
	}

	// Try to parse metadata from payload
	if len(msg.Payload) > 0 && msg.Payload[0] == '{' {
		var metadata map[string]interface{}
		if err := json.Unmarshal(msg.Payload, &metadata); err == nil {
			routed.Metadata = metadata

			// Extract common fields
			if taskID, ok := metadata["task_id"].(string); ok {
				routed.TaskID = taskID
			}
			if subtype, ok := metadata["type"].(string); ok {
				routed.Subtype = subtype
			}
		}
	}

	return routed, nil
}

// RouteClineMessage routes a Cline proto message
func (r *MessageRouter) RouteClineMessage(ctx context.Context, msg *cline.ClineMessage) error {
	if msg == nil {
		return fmt.Errorf("cannot route nil cline message")
	}

	// Convert cline message to routed message
	routed := &RoutedMessage{
		Timestamp: time.Unix(msg.Ts, 0),
		Sequence:  msg.Ts,
		Metadata:  make(map[string]interface{}),
	}

	// Determine message type from ask/say fields
	// ClineAsk is an int32 enum
	switch msg.Ask {
	case 1: // ASK_FOLLOWUP
		routed.Type = MessageTypeAsk
		routed.Subtype = "followup"
	case 2: // ASK_TOOL
		routed.Type = MessageTypeApproval
		routed.Subtype = "tool_approval"
	case 3: // ASK_COMMAND
		routed.Type = MessageTypeApproval
		routed.Subtype = "command_approval"
	case 4: // ASK_BROWSER_ACTION
		routed.Type = MessageTypeApproval
		routed.Subtype = "browser_approval"
	default:
		routed.Type = MessageTypeSay
		// ClineSay is also an int32 enum
		routed.Subtype = fmt.Sprintf("say_%d", msg.Say)
	}

	// Extract text and images
	if msg.Text != "" {
		routed.Metadata["text"] = msg.Text
	}
	if len(msg.Images) > 0 {
		routed.Metadata["images"] = msg.Images
	}

	// Get handlers
	r.mu.RLock()
	handlers := r.handlers[routed.Type]
	r.mu.RUnlock()

	if len(handlers) == 0 && r.defaultHandler != nil {
		handlers = []MessageHandler{r.defaultHandler}
	}

	// Execute handlers
	var lastErr error
	for _, handler := range handlers {
		if err := handler(routed); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// CreateStreamHandler creates a handler for streaming messages from the bidirectional stream
func (r *MessageRouter) CreateStreamHandler() func(*Message) {
	return func(msg *Message) {
		if err := r.Route(msg); err != nil {
			// Log error but don't stop processing
			fmt.Printf("Error routing message: %v\n", err)
		}
	}
}

// LoggingMiddleware creates a middleware that logs all messages
func LoggingMiddleware(logger func(format string, args ...interface{})) MiddlewareFunc {
	return func(next MessageHandler) MessageHandler {
		return func(msg *RoutedMessage) error {
			logger("Routing message: type=%s subtype=%s task=%s", msg.Type, msg.Subtype, msg.TaskID)
			return next(msg)
		}
	}
}

// RecoveryMiddleware creates a middleware that recovers from panics
func RecoveryMiddleware(onPanic func(recover interface{})) MiddlewareFunc {
	return func(next MessageHandler) MessageHandler {
		return func(msg *RoutedMessage) (err error) {
			defer func() {
				if r := recover(); r != nil {
					onPanic(r)
					err = fmt.Errorf("handler panicked: %v", r)
				}
			}()
			return next(msg)
		}
	}
}

// TimingMiddleware creates a middleware that tracks handler execution time
func TimingMiddleware(onSlow func(msg *RoutedMessage, duration time.Duration)) MiddlewareFunc {
	return func(next MessageHandler) MessageHandler {
		return func(msg *RoutedMessage) error {
			start := time.Now()
			err := next(msg)
			duration := time.Since(start)

			if duration > 100*time.Millisecond {
				onSlow(msg, duration)
			}

			return err
		}
	}
}

// BatchRouter batches messages and routes them together
type BatchRouter struct {
	router        *MessageRouter
	batchSize     int
	flushInterval time.Duration
	batch         []*RoutedMessage
	mu            sync.Mutex
	flushChan     chan struct{}
	done          chan struct{}
}

// NewBatchRouter creates a new batch router
func NewBatchRouter(router *MessageRouter, batchSize int, flushInterval time.Duration) *BatchRouter {
	br := &BatchRouter{
		router:        router,
		batchSize:     batchSize,
		flushInterval: flushInterval,
		batch:         make([]*RoutedMessage, 0, batchSize),
		flushChan:     make(chan struct{}),
		done:          make(chan struct{}),
	}

	go br.flushLoop()
	return br
}

// Add adds a message to the batch
func (br *BatchRouter) Add(msg *RoutedMessage) {
	br.mu.Lock()
	br.batch = append(br.batch, msg)

	shouldFlush := len(br.batch) >= br.batchSize
	br.mu.Unlock()

	if shouldFlush {
		br.flushChan <- struct{}{}
	}
}

// flushLoop periodically flushes the batch
func (br *BatchRouter) flushLoop() {
	ticker := time.NewTicker(br.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			br.Flush()
		case <-br.flushChan:
			br.Flush()
		case <-br.done:
			return
		}
	}
}

// Flush immediately routes all batched messages
func (br *BatchRouter) Flush() {
	br.mu.Lock()
	batch := make([]*RoutedMessage, len(br.batch))
	copy(batch, br.batch)
	br.batch = br.batch[:0]
	br.mu.Unlock()

	for _, msg := range batch {
		if err := br.router.Route(&Message{
			Type:      string(msg.Type),
			Payload:   msg.Payload,
			Timestamp: msg.Timestamp.Unix(),
			Sequence:  int(msg.Sequence),
		}); err != nil {
			fmt.Printf("Error routing batched message: %v\n", err)
		}
	}
}

// Stop stops the batch router
func (br *BatchRouter) Stop() {
	close(br.done)
	br.Flush()
}

// RouterStats contains statistics about message routing
type RouterStats struct {
	TotalRouted    int64
	ByType         map[MessageType]int64
	AverageLatency time.Duration
	LastRouteTime  time.Time
}

// Stats returns routing statistics
func (r *MessageRouter) Stats() RouterStats {
	// This is a simplified implementation
	return RouterStats{
		ByType:        make(map[MessageType]int64),
		LastRouteTime: time.Now(),
	}
}
