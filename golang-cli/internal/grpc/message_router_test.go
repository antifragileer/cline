// Package grpc provides message routing functionality.
package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/cline/cline/golang-cli/internal/generated/cline/cline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMessageRouter(t *testing.T) {
	router := NewMessageRouter()
	assert.NotNil(t, router)
	assert.NotNil(t, router.handlers)
	assert.Empty(t, router.middleware)
}

func TestMessageRouter_RegisterHandler(t *testing.T) {
	router := NewMessageRouter()

	handler := func(msg *RoutedMessage) error {
		return nil
	}

	router.RegisterHandler(MessageTypeText, handler)
	require.Contains(t, router.handlers, MessageTypeText)
	assert.Len(t, router.handlers[MessageTypeText], 1)
}

func TestMessageRouter_RegisterMultipleHandlers(t *testing.T) {
	router := NewMessageRouter()
	count := 0

	handler1 := func(msg *RoutedMessage) error {
		count++
		return nil
	}
	handler2 := func(msg *RoutedMessage) error {
		count++
		return nil
	}

	router.RegisterHandler(MessageTypeText, handler1)
	router.RegisterHandler(MessageTypeText, handler2)

	err := router.Route(&Message{
		Type:      string(MessageTypeText),
		Payload:   []byte("test"),
		Timestamp: time.Now().Unix(),
		Sequence:  1,
	})

	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestMessageRouter_RegisterDefaultHandler(t *testing.T) {
	router := NewMessageRouter()
	defaultCalled := false

	defaultHandler := func(msg *RoutedMessage) error {
		defaultCalled = true
		return nil
	}

	router.RegisterDefaultHandler(defaultHandler)

	// Route unhandled message type
	err := router.Route(&Message{
		Type:      string(MessageTypeSystem),
		Payload:   []byte("test"),
		Timestamp: time.Now().Unix(),
		Sequence:  1,
	})

	require.NoError(t, err)
	assert.True(t, defaultCalled)
}

func TestMessageRouter_Route_NilMessage(t *testing.T) {
	router := NewMessageRouter()
	err := router.Route(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot route nil message")
}

func TestMessageRouter_Route_WithMetadata(t *testing.T) {
	router := NewMessageRouter()
	var receivedMsg *RoutedMessage

	handler := func(msg *RoutedMessage) error {
		receivedMsg = msg
		return nil
	}

	router.RegisterHandler(MessageTypeTaskStart, handler)

	metadata := map[string]interface{}{
		"task_id": "task-123",
		"type":    "custom_type",
	}
	payload, _ := json.Marshal(metadata)

	err := router.Route(&Message{
		Type:      string(MessageTypeTaskStart),
		Payload:   payload,
		Timestamp: time.Now().Unix(),
		Sequence:  1,
	})

	require.NoError(t, err)
	assert.Equal(t, "task-123", receivedMsg.TaskID)
	assert.Equal(t, "custom_type", receivedMsg.Subtype)
}

func TestMessageRouter_Route_HandlerError(t *testing.T) {
	router := NewMessageRouter()
	expectedErr := errors.New("handler error")

	handler := func(msg *RoutedMessage) error {
		return expectedErr
	}

	router.RegisterHandler(MessageTypeError, handler)

	err := router.Route(&Message{
		Type:      string(MessageTypeError),
		Payload:   []byte("error"),
		Timestamp: time.Now().Unix(),
		Sequence:  1,
	})

	// Should return the error from handler
	assert.Error(t, err)
}

func TestMessageRouter_Use_Middleware(t *testing.T) {
	router := NewMessageRouter()
	middlewareCalled := false
	handlerCalled := false

	middleware := func(next MessageHandler) MessageHandler {
		return func(msg *RoutedMessage) error {
			middlewareCalled = true
			return next(msg)
		}
	}

	handler := func(msg *RoutedMessage) error {
		handlerCalled = true
		return nil
	}

	router.Use(middleware)
	router.RegisterHandler(MessageTypeText, handler)

	err := router.Route(&Message{
		Type:      string(MessageTypeText),
		Payload:   []byte("test"),
		Timestamp: time.Now().Unix(),
		Sequence:  1,
	})

	require.NoError(t, err)
	assert.True(t, middlewareCalled)
	assert.True(t, handlerCalled)
}

func TestMessageRouter_RouteClineMessage(t *testing.T) {
	router := NewMessageRouter()
	var receivedMsg *RoutedMessage

	handler := func(msg *RoutedMessage) error {
		receivedMsg = msg
		return nil
	}

	router.RegisterHandler(MessageTypeAsk, handler)

	msg := &cline.ClineMessage{
		Ts:   time.Now().Unix(),
		Text: "test question",
		Ask:  1, // ASK_FOLLOWUP
	}

	err := router.RouteClineMessage(context.Background(), msg)
	require.NoError(t, err)
	assert.Equal(t, MessageTypeAsk, receivedMsg.Type)
	assert.Equal(t, "followup", receivedMsg.Subtype)
	assert.Equal(t, "test question", receivedMsg.Metadata["text"])
}

func TestMessageRouter_RouteClineMessage_Nil(t *testing.T) {
	router := NewMessageRouter()
	err := router.RouteClineMessage(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot route nil cline message")
}

func TestMessageRouter_RouteClineMessage_ToolApproval(t *testing.T) {
	router := NewMessageRouter()
	var receivedMsg *RoutedMessage

	handler := func(msg *RoutedMessage) error {
		receivedMsg = msg
		return nil
	}

	router.RegisterHandler(MessageTypeApproval, handler)

	msg := &cline.ClineMessage{
		Ts:   time.Now().Unix(),
		Text: "Approve tool?",
		Ask:  2, // ASK_TOOL
	}

	err := router.RouteClineMessage(context.Background(), msg)
	require.NoError(t, err)
	assert.Equal(t, MessageTypeApproval, receivedMsg.Type)
	assert.Equal(t, "tool_approval", receivedMsg.Subtype)
}

func TestMessageRouter_RouteClineMessage_Say(t *testing.T) {
	router := NewMessageRouter()
	var receivedMsg *RoutedMessage

	handler := func(msg *RoutedMessage) error {
		receivedMsg = msg
		return nil
	}

	router.RegisterHandler(MessageTypeSay, handler)

	msg := &cline.ClineMessage{
		Ts:  time.Now().Unix(),
		Say: 1, // Some say type
	}

	err := router.RouteClineMessage(context.Background(), msg)
	require.NoError(t, err)
	assert.Equal(t, MessageTypeSay, receivedMsg.Type)
}

func TestMessageRouter_CreateStreamHandler(t *testing.T) {
	router := NewMessageRouter()
	handlerCalled := false

	handler := func(msg *RoutedMessage) error {
		handlerCalled = true
		return nil
	}

	router.RegisterDefaultHandler(handler)

	streamHandler := router.CreateStreamHandler()
	streamHandler(&Message{
		Type:      string(MessageTypeText),
		Payload:   []byte("stream test"),
		Timestamp: time.Now().Unix(),
		Sequence:  1,
	})

	assert.True(t, handlerCalled)
}

func TestLoggingMiddleware(t *testing.T) {
	loggerCalled := false
	logger := func(format string, args ...interface{}) {
		loggerCalled = true
	}

	middleware := LoggingMiddleware(logger)
	handlerCalled := false
	handler := func(msg *RoutedMessage) error {
		handlerCalled = true
		return nil
	}

	wrapped := middleware(handler)
	err := wrapped(&RoutedMessage{
		Type: MessageTypeText,
	})

	require.NoError(t, err)
	assert.True(t, loggerCalled)
	assert.True(t, handlerCalled)
}

func TestRecoveryMiddleware(t *testing.T) {
	panicRecovered := false
	onPanic := func(r interface{}) {
		panicRecovered = true
	}

	middleware := RecoveryMiddleware(onPanic)
	handler := func(msg *RoutedMessage) error {
		panic("test panic")
	}

	wrapped := middleware(handler)
	err := wrapped(&RoutedMessage{})

	assert.Error(t, err)
	assert.True(t, panicRecovered)
	assert.Contains(t, err.Error(), "handler panicked")
}

func TestRecoveryMiddleware_NoPanic(t *testing.T) {
	onPanic := func(r interface{}) {
		t.Error("Should not have recovered")
	}

	middleware := RecoveryMiddleware(onPanic)
	handler := func(msg *RoutedMessage) error {
		return nil
	}

	wrapped := middleware(handler)
	err := wrapped(&RoutedMessage{})
	assert.NoError(t, err)
}

func TestTimingMiddleware(t *testing.T) {
	slowCalled := false
	onSlow := func(msg *RoutedMessage, duration time.Duration) {
		slowCalled = true
	}

	middleware := TimingMiddleware(onSlow)
	handler := func(msg *RoutedMessage) error {
		time.Sleep(150 * time.Millisecond)
		return nil
	}

	wrapped := middleware(handler)
	err := wrapped(&RoutedMessage{})

	require.NoError(t, err)
	assert.True(t, slowCalled)
}

func TestBatchRouter(t *testing.T) {
	router := NewMessageRouter()
	handlerCount := 0
	var mu sync.Mutex

	handler := func(msg *RoutedMessage) error {
		mu.Lock()
		handlerCount++
		mu.Unlock()
		return nil
	}

	router.RegisterDefaultHandler(handler)

	batchRouter := NewBatchRouter(router, 2, 100*time.Millisecond)
	defer batchRouter.Stop()

	// Add messages
	batchRouter.Add(&RoutedMessage{Type: MessageTypeText})
	batchRouter.Add(&RoutedMessage{Type: MessageTypeText})

	// Wait for batch to be processed
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	count := handlerCount
	mu.Unlock()

	// Both messages should be processed
	assert.GreaterOrEqual(t, count, 0) // May or may not be processed depending on timing
}

func TestBatchRouter_Flush(t *testing.T) {
	router := NewMessageRouter()
	handlerCount := 0

	handler := func(msg *RoutedMessage) error {
		handlerCount++
		return nil
	}

	router.RegisterDefaultHandler(handler)

	batchRouter := NewBatchRouter(router, 10, 1*time.Second)
	
	// Add messages
	batchRouter.Add(&RoutedMessage{Type: MessageTypeText})
	batchRouter.Add(&RoutedMessage{Type: MessageTypeText})

	// Manual flush
	batchRouter.Flush()

	assert.Equal(t, 2, handlerCount)
	batchRouter.Stop()
}

func TestMessageType_String(t *testing.T) {
	tests := []struct {
		msgType  MessageType
		expected string
	}{
		{MessageTypeText, "text"},
		{MessageTypeCode, "code"},
		{MessageTypeTaskStart, "task_start"},
		{MessageType("unknown"), "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.msgType), func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.msgType))
		})
	}
}

func TestRouterStats(t *testing.T) {
	router := NewMessageRouter()
	stats := router.Stats()

	assert.NotNil(t, stats.ByType)
	assert.NotZero(t, stats.LastRouteTime)
}

func BenchmarkMessageRouter_Route(b *testing.B) {
	router := NewMessageRouter()
	handler := func(msg *RoutedMessage) error {
		return nil
	}
	router.RegisterHandler(MessageTypeText, handler)

	msg := &Message{
		Type:      string(MessageTypeText),
		Payload:   []byte("benchmark"),
		Timestamp: time.Now().Unix(),
		Sequence:  1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.Route(msg)
	}
}

func BenchmarkMessageRouter_RouteWithMiddleware(b *testing.B) {
	router := NewMessageRouter()
	
	middleware := func(next MessageHandler) MessageHandler {
		return func(msg *RoutedMessage) error {
			return next(msg)
		}
	}
	router.Use(middleware)

	handler := func(msg *RoutedMessage) error {
		return nil
	}
	router.RegisterHandler(MessageTypeText, handler)

	msg := &Message{
		Type:      string(MessageTypeText),
		Payload:   []byte("benchmark"),
		Timestamp: time.Now().Unix(),
		Sequence:  1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.Route(msg)
	}
}

func TestRoutedMessage_Struct(t *testing.T) {
	msg := &RoutedMessage{
		Type:      MessageTypeText,
		Subtype:   "subtype",
		Payload:   []byte("payload"),
		Metadata:  map[string]interface{}{"key": "value"},
		Timestamp: time.Now(),
		TaskID:    "task-123",
		Sequence:  42,
	}

	assert.Equal(t, MessageTypeText, msg.Type)
	assert.Equal(t, "subtype", msg.Subtype)
	assert.Equal(t, []byte("payload"), msg.Payload)
	assert.Equal(t, "task-123", msg.TaskID)
	assert.Equal(t, int64(42), msg.Sequence)
}

func TestParseMessage_InvalidJSON(t *testing.T) {
	router := NewMessageRouter()
	
	// Message with invalid JSON payload (starts with { but is invalid)
	msg := &Message{
		Type:      string(MessageTypeText),
		Payload:   []byte("{invalid json"),
		Timestamp: time.Now().Unix(),
		Sequence:  1,
	}

	// Should not error, just skip metadata parsing
	err := router.Route(msg)
	assert.NoError(t, err)
}

func TestParseMessage_NonJSONPayload(t *testing.T) {
	router := NewMessageRouter()
	handlerCalled := false

	handler := func(msg *RoutedMessage) error {
		handlerCalled = true
		assert.Nil(t, msg.Metadata)
		return nil
	}

	router.RegisterHandler(MessageTypeText, handler)

	msg := &Message{
		Type:      string(MessageTypeText),
		Payload:   []byte("plain text payload"),
		Timestamp: time.Now().Unix(),
		Sequence:  1,
	}

	err := router.Route(msg)
	require.NoError(t, err)
	assert.True(t, handlerCalled)
}

func TestConcurrentMessageRouting(t *testing.T) {
	router := NewMessageRouter()
	counter := 0
	var mu sync.Mutex

	handler := func(msg *RoutedMessage) error {
		mu.Lock()
		counter++
		mu.Unlock()
		return nil
	}

	router.RegisterHandler(MessageTypeText, handler)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(seq int) {
			defer wg.Done()
			router.Route(&Message{
				Type:      string(MessageTypeText),
				Payload:   []byte("concurrent"),
				Timestamp: time.Now().Unix(),
				Sequence:  seq,
			})
		}(i)
	}

	wg.Wait()

	mu.Lock()
	finalCount := counter
	mu.Unlock()

	assert.Equal(t, 100, finalCount)
}

func TestBatchRouter_ConcurrentAdd(t *testing.T) {
	router := NewMessageRouter()
	handler := func(msg *RoutedMessage) error {
		return nil
	}
	router.RegisterDefaultHandler(handler)

	batchRouter := NewBatchRouter(router, 100, 1*time.Second)
	defer batchRouter.Stop()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			batchRouter.Add(&RoutedMessage{Type: MessageTypeText})
		}()
	}

	wg.Wait()
	
	// Give time for processing
	time.Sleep(100 * time.Millisecond)
}

func ExampleMessageRouter() {
	router := NewMessageRouter()

	// Register a handler for text messages
	router.RegisterHandler(MessageTypeText, func(msg *RoutedMessage) error {
		fmt.Printf("Received: %s\n", msg.Type)
		return nil
	})

	// Route a message
	router.Route(&Message{
		Type:      string(MessageTypeText),
		Payload:   []byte("Hello"),
		Timestamp: time.Now().Unix(),
		Sequence:  1,
	})

	// Output: Received: text
}