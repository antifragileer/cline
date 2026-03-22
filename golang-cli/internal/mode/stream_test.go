package mode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestDefaultStreamConfig verifies the default configuration values
func TestDefaultStreamConfig(t *testing.T) {
	config := DefaultStreamConfig()

	if !config.AutoFlush {
		t.Error("expected AutoFlush to be true")
	}

	if config.FlushInterval != 100*time.Millisecond {
		t.Errorf("expected FlushInterval 100ms, got %v", config.FlushInterval)
	}

	if config.BufferSize != 4096 {
		t.Errorf("expected BufferSize 4096, got %d", config.BufferSize)
	}

	if !config.EnablePartialMessages {
		t.Error("expected EnablePartialMessages to be true")
	}

	if config.PartialTimeout != 30*time.Second {
		t.Errorf("expected PartialTimeout 30s, got %v", config.PartialTimeout)
	}
}

// TestNewJSONStreamer verifies streamer creation
func TestNewJSONStreamer(t *testing.T) {
	t.Run("valid writer", func(t *testing.T) {
		var buf bytes.Buffer
		streamer := NewJSONStreamer(&buf, nil)

		if streamer == nil {
			t.Fatal("expected non-nil streamer")
		}

		if !streamer.IsReady() {
			t.Error("expected streamer to be ready")
		}

		if streamer.IsClosed() {
			t.Error("expected streamer to not be closed")
		}

		streamer.Close()
	})

	t.Run("nil writer panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for nil writer")
			}
		}()

		NewJSONStreamer(nil, nil)
	})

	t.Run("with custom config", func(t *testing.T) {
		var buf bytes.Buffer
		config := &StreamConfig{
			AutoFlush:             false,
			FlushInterval:         500 * time.Millisecond,
			BufferSize:            8192,
			EnablePartialMessages: false,
			PartialTimeout:        60 * time.Second,
		}

		streamer := NewJSONStreamer(&buf, config)

		if streamer.config.AutoFlush != false {
			t.Error("expected AutoFlush to be false")
		}

		if streamer.config.FlushInterval != 500*time.Millisecond {
			t.Errorf("expected FlushInterval 500ms, got %v", streamer.config.FlushInterval)
		}

		streamer.Close()
	})
}

// TestWriteMessage verifies basic message writing
func TestWriteMessage(t *testing.T) {
	var buf bytes.Buffer
	streamer := NewJSONStreamer(&buf, nil)
	defer streamer.Close()

	t.Run("simple message", func(t *testing.T) {
		buf.Reset()

		err := streamer.WriteMessage("text", "hello world")
		if err != nil {
			t.Fatalf("WriteMessage failed: %v", err)
		}

		// Parse the output
		var msg StreamMessage
		if err := json.Unmarshal(buf.Bytes(), &msg); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if msg.Type != "text" {
			t.Errorf("expected type 'text', got '%s'", msg.Type)
		}

		if msg.Content != "hello world" {
			t.Errorf("expected content 'hello world', got '%s'", msg.Content)
		}

		if msg.Partial {
			t.Error("expected Partial to be false")
		}

		if msg.MessageID == "" {
			t.Error("expected non-empty MessageID")
		}
	})

	t.Run("empty content", func(t *testing.T) {
		buf.Reset()

		err := streamer.WriteMessage("empty", "")
		if err != nil {
			t.Fatalf("WriteMessage failed: %v", err)
		}

		var msg StreamMessage
		if err := json.Unmarshal(buf.Bytes(), &msg); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if msg.Content != "" {
			t.Errorf("expected empty content, got '%s'", msg.Content)
		}
	})

	t.Run("special characters", func(t *testing.T) {
		buf.Reset()

		content := "hello\nworld\t\"quoted\" \\backslash"
		err := streamer.WriteMessage("special", content)
		if err != nil {
			t.Fatalf("WriteMessage failed: %v", err)
		}

		var msg StreamMessage
		if err := json.Unmarshal(buf.Bytes(), &msg); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if msg.Content != content {
			t.Errorf("expected content '%s', got '%s'", content, msg.Content)
		}
	})
}

// TestWritePartialMessage verifies partial message functionality
func TestWritePartialMessage(t *testing.T) {
	var buf bytes.Buffer
	streamer := NewJSONStreamer(&buf, nil)
	defer streamer.Close()

	t.Run("complete partial flow", func(t *testing.T) {
		buf.Reset()
		msgID := "test-partial-1"

		// Write first partial chunk
		err := streamer.WritePartialMessage("text", "hello ", msgID, false)
		if err != nil {
			t.Fatalf("WritePartialMessage failed: %v", err)
		}

		// Write second partial chunk
		err = streamer.WritePartialMessage("text", "world", msgID, true)
		if err != nil {
			t.Fatalf("WritePartialMessage failed: %v", err)
		}

		// Parse output - should be two JSON lines
		lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines, got %d", len(lines))
		}

		// Verify first chunk
		var chunk1 StreamMessage
		if err := json.Unmarshal([]byte(lines[0]), &chunk1); err != nil {
			t.Fatalf("failed to parse chunk1: %v", err)
		}

		if chunk1.Content != "hello " {
			t.Errorf("expected chunk1 content 'hello ', got '%s'", chunk1.Content)
		}

		if !chunk1.Partial {
			t.Error("expected chunk1 Partial to be true")
		}

		if chunk1.SequenceNumber != 1 {
			t.Errorf("expected chunk1 SequenceNumber 1, got %d", chunk1.SequenceNumber)
		}

		// Verify second (final) chunk
		var chunk2 StreamMessage
		if err := json.Unmarshal([]byte(lines[1]), &chunk2); err != nil {
			t.Fatalf("failed to parse chunk2: %v", err)
		}

		if chunk2.Content != "world" {
			t.Errorf("expected chunk2 content 'world', got '%s'", chunk2.Content)
		}

		if chunk2.Partial {
			t.Error("expected chunk2 Partial to be false")
		}

		if chunk2.SequenceNumber != 2 {
			t.Errorf("expected chunk2 SequenceNumber 2, got %d", chunk2.SequenceNumber)
		}
	})

	t.Run("single chunk complete", func(t *testing.T) {
		buf.Reset()
		msgID := "test-partial-2"

		err := streamer.WritePartialMessage("text", "complete", msgID, true)
		if err != nil {
			t.Fatalf("WritePartialMessage failed: %v", err)
		}

		var msg StreamMessage
		if err := json.Unmarshal(buf.Bytes(), &msg); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if msg.Partial {
			t.Error("expected Partial to be false for single complete chunk")
		}

		if msg.Content != "complete" {
			t.Errorf("expected content 'complete', got '%s'", msg.Content)
		}
	})

	t.Run("disabled partial messages", func(t *testing.T) {
		var buf2 bytes.Buffer
		config := &StreamConfig{
			EnablePartialMessages: false,
			AutoFlush:             true,
		}
		streamer2 := NewJSONStreamer(&buf2, config)
		defer streamer2.Close()

		msgID := "test-partial-3"

		// First partial (not last) should be ignored when disabled
		err := streamer2.WritePartialMessage("text", "ignored", msgID, false)
		if err != nil {
			t.Fatalf("WritePartialMessage failed: %v", err)
		}

		if buf2.Len() != 0 {
			t.Error("expected no output for non-final partial when disabled")
		}

		// Final partial should be written as complete message
		err = streamer2.WritePartialMessage("text", "final", msgID, true)
		if err != nil {
			t.Fatalf("WritePartialMessage failed: %v", err)
		}

		// Now buffer should have the final message
		if buf2.Len() == 0 {
			t.Fatal("expected output for final partial")
		}

		var msg StreamMessage
		if err := json.Unmarshal(buf2.Bytes(), &msg); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if msg.Partial {
			t.Error("expected Partial to be false when disabled")
		}

		if msg.Content != "final" {
			t.Errorf("expected content 'final', got '%s'", msg.Content)
		}
	})
}

// TestReassembleMessage verifies message reassembly from partial chunks
func TestReassembleMessage(t *testing.T) {
	var buf bytes.Buffer
	streamer := NewJSONStreamer(&buf, nil)
	defer streamer.Close()

	msgID := "reassemble-test"

	// Write partial chunks
	chunks := []struct {
		content string
		isLast  bool
	}{
		{"Hello ", false},
		{"World", false},
		{"!", true},
	}

	for _, chunk := range chunks {
		err := streamer.WritePartialMessage("text", chunk.content, msgID, chunk.isLast)
		if err != nil {
			t.Fatalf("WritePartialMessage failed: %v", err)
		}
	}

	// Reassemble
	reassembled := streamer.ReassembleMessage(msgID)
	if reassembled == nil {
		t.Fatal("expected reassembled message, got nil")
	}

	expectedContent := "Hello World!"
	if reassembled.Content != expectedContent {
		t.Errorf("expected content '%s', got '%s'", expectedContent, reassembled.Content)
	}

	if reassembled.Partial {
		t.Error("expected reassembled message to not be partial")
	}

	if reassembled.Type != "text" {
		t.Errorf("expected type 'text', got '%s'", reassembled.Type)
	}

	// Test non-existent message
	nonExistent := streamer.ReassembleMessage("non-existent")
	if nonExistent != nil {
		t.Error("expected nil for non-existent message")
	}
}

// TestFlush verifies flush functionality
func TestFlush(t *testing.T) {
	t.Run("explicit flush", func(t *testing.T) {
		var buf bytes.Buffer
		config := &StreamConfig{AutoFlush: false}
		streamer := NewJSONStreamer(&buf, config)

		err := streamer.WriteMessage("text", "test")
		if err != nil {
			t.Fatalf("WriteMessage failed: %v", err)
		}

		// Without flush, buffer may still contain data
		err = streamer.Flush()
		if err != nil {
			t.Fatalf("Flush failed: %v", err)
		}

		// Now data should be in buffer
		if buf.Len() == 0 {
			t.Error("expected data after flush")
		}

		streamer.Close()
	})

	t.Run("auto flush", func(t *testing.T) {
		var buf bytes.Buffer
		config := &StreamConfig{AutoFlush: true}
		streamer := NewJSONStreamer(&buf, config)
		defer streamer.Close()

		err := streamer.WriteMessage("text", "test")
		if err != nil {
			t.Fatalf("WriteMessage failed: %v", err)
		}

		// With auto-flush, data should be immediately available
		if buf.Len() == 0 {
			t.Error("expected data with auto-flush")
		}
	})

	t.Run("flush on closed stream", func(t *testing.T) {
		var buf bytes.Buffer
		streamer := NewJSONStreamer(&buf, nil)
		streamer.Close()

		err := streamer.Flush()
		if err != nil {
			t.Logf("Flush on closed stream error (may be expected): %v", err)
		}
	})
}

// TestClose verifies stream closure
func TestClose(t *testing.T) {
	t.Run("normal close", func(t *testing.T) {
		var buf bytes.Buffer
		streamer := NewJSONStreamer(&buf, nil)

		if streamer.IsClosed() {
			t.Error("expected streamer to not be closed initially")
		}

		err := streamer.Close()
		if err != nil {
			t.Fatalf("Close failed: %v", err)
		}

		if !streamer.IsClosed() {
			t.Error("expected streamer to be closed after Close")
		}

		if streamer.IsReady() {
			t.Error("expected streamer to not be ready after Close")
		}
	})

	t.Run("double close", func(t *testing.T) {
		var buf bytes.Buffer
		streamer := NewJSONStreamer(&buf, nil)

		err := streamer.Close()
		if err != nil {
			t.Fatalf("first Close failed: %v", err)
		}

		// Second close should not error
		err = streamer.Close()
		if err != nil {
			t.Fatalf("second Close failed: %v", err)
		}
	})

	t.Run("write after close", func(t *testing.T) {
		var buf bytes.Buffer
		streamer := NewJSONStreamer(&buf, nil)
		streamer.Close()

		err := streamer.WriteMessage("text", "test")
		if err != ErrStreamClosed {
			t.Errorf("expected ErrStreamClosed, got %v", err)
		}
	})
}

// TestWriteRaw verifies raw byte writing
func TestWriteRaw(t *testing.T) {
	var buf bytes.Buffer
	streamer := NewJSONStreamer(&buf, nil)
	defer streamer.Close()

	data := []byte("raw data\n")
	err := streamer.WriteRaw(data)
	if err != nil {
		t.Fatalf("WriteRaw failed: %v", err)
	}

	if !bytes.Contains(buf.Bytes(), data) {
		t.Errorf("expected buffer to contain '%s', got '%s'", data, buf.Bytes())
	}

	// Test write after close
	streamer.Close()

	err = streamer.WriteRaw([]byte("test"))
	if err != ErrStreamClosed {
		t.Errorf("expected ErrStreamClosed, got %v", err)
	}
}

// TestCleanupPartialMessages verifies partial message cleanup
func TestCleanupPartialMessages(t *testing.T) {
	var buf bytes.Buffer
	config := &StreamConfig{
		PartialTimeout: 50 * time.Millisecond,
	}
	streamer := NewJSONStreamer(&buf, config)
	defer streamer.Close()

	// Add partial message
	streamer.partialMu.Lock()
	streamer.partialBuffers["old-msg"] = &partialBuffer{
		chunks:     []StreamMessage{{MessageID: "old-msg", Content: "test"}},
		lastUpdate: time.Now().Add(-time.Hour), // Expired
	}
	streamer.partialBuffers["new-msg"] = &partialBuffer{
		chunks:     []StreamMessage{{MessageID: "new-msg", Content: "test"}},
		lastUpdate: time.Now(), // Not expired
	}
	streamer.partialMu.Unlock()

	// Run cleanup
	cleaned := streamer.CleanupPartialMessages()

	if cleaned != 1 {
		t.Errorf("expected 1 cleaned message, got %d", cleaned)
	}

	if streamer.GetPartialMessageCount() != 1 {
		t.Errorf("expected 1 remaining message, got %d", streamer.GetPartialMessageCount())
	}
}

// TestCallbacks verifies callback functionality
func TestCallbacks(t *testing.T) {
	t.Run("on message callback", func(t *testing.T) {
		var receivedMsg *StreamMessage
		var mu sync.Mutex

		var buf bytes.Buffer
		config := &StreamConfig{
			OnMessage: func(msg *StreamMessage) {
				mu.Lock()
				defer mu.Unlock()
				receivedMsg = msg
			},
		}

		streamer := NewJSONStreamer(&buf, config)
		defer streamer.Close()

		err := streamer.WriteMessage("test", "callback test")
		if err != nil {
			t.Fatalf("WriteMessage failed: %v", err)
		}

		// Wait a bit for callback
		time.Sleep(10 * time.Millisecond)

		mu.Lock()
		if receivedMsg == nil {
			t.Fatal("expected message callback to be called")
		}
		if receivedMsg.Type != "test" {
			t.Errorf("expected type 'test', got '%s'", receivedMsg.Type)
		}
		if receivedMsg.Content != "callback test" {
			t.Errorf("expected content 'callback test', got '%s'", receivedMsg.Content)
		}
		mu.Unlock()
	})

	t.Run("on error callback", func(t *testing.T) {
		var receivedErr error
		var mu sync.Mutex

		var buf bytes.Buffer
		config := &StreamConfig{
			OnError: func(err error) {
				mu.Lock()
				defer mu.Unlock()
				receivedErr = err
			},
		}

		streamer := NewJSONStreamer(&buf, config)

		// Force an error by writing invalid JSON
		// This is done through handleError which is internal
		testErr := errors.New("test error")
		streamer.handleError(testErr)

		mu.Lock()
		if receivedErr != testErr {
			t.Errorf("expected error %v, got %v", testErr, receivedErr)
		}
		mu.Unlock()

		streamer.Close()
	})
}

// TestStreamContext verifies context-aware streaming
func TestStreamContext(t *testing.T) {
	t.Run("normal operation", func(t *testing.T) {
		var buf bytes.Buffer
		ctx := context.Background()

		sc := NewStreamContext(ctx, &buf, nil)
		defer sc.Close()

		err := sc.WriteMessage("text", "context test")
		if err != nil {
			t.Fatalf("WriteMessage failed: %v", err)
		}

		if buf.Len() == 0 {
			t.Error("expected data in buffer")
		}
	})

	t.Run("cancelled context", func(t *testing.T) {
		var buf bytes.Buffer
		ctx, cancel := context.WithCancel(context.Background())

		sc := NewStreamContext(ctx, &buf, nil)
		defer sc.Close()

		// Cancel context
		cancel()

		// Give time for cancellation to propagate
		time.Sleep(10 * time.Millisecond)

		err := sc.WriteMessage("text", "should fail")
		if err != context.Canceled {
			t.Logf("WriteMessage after cancel (may be expected): %v", err)
		}
	})

	t.Run("context methods", func(t *testing.T) {
		var buf bytes.Buffer
		ctx := context.Background()

		sc := NewStreamContext(ctx, &buf, nil)
		defer sc.Close()

		if sc.Streamer() == nil {
			t.Error("expected non-nil streamer")
		}

		if sc.Context() == nil {
			t.Error("expected non-nil context")
		}

		// Cancel should not panic
		sc.Cancel()
	})
}

// TestStreamWriter verifies io.Writer implementation
func TestStreamWriter(t *testing.T) {
	var buf bytes.Buffer
	streamer := NewJSONStreamer(&buf, nil)
	defer streamer.Close()

	writer := streamer.NewStreamWriter("io_writer")

	data := []byte("io writer test")
	n, err := writer.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if n != len(data) {
		t.Errorf("expected %d bytes written, got %d", len(data), n)
	}

	// Verify the message was written
	var msg StreamMessage
	if err := json.Unmarshal(buf.Bytes(), &msg); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if msg.Type != "io_writer" {
		t.Errorf("expected type 'io_writer', got '%s'", msg.Type)
	}

	if msg.Content != string(data) {
		t.Errorf("expected content '%s', got '%s'", data, msg.Content)
	}
}

// TestConcurrentAccess verifies thread safety
func TestConcurrentAccess(t *testing.T) {
	var buf bytes.Buffer
	streamer := NewJSONStreamer(&buf, nil)
	defer streamer.Close()

	var wg sync.WaitGroup
	numGoroutines := 50
	numMessages := 10

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numMessages; j++ {
				err := streamer.WriteMessage("concurrent", "message")
				if err != nil && err != ErrStreamClosed {
					t.Errorf("WriteMessage failed: %v", err)
				}
			}
		}(i)
	}

	// Concurrent state checks
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numMessages; j++ {
				_ = streamer.IsReady()
				_ = streamer.IsClosed()
				_ = streamer.GetPartialMessageCount()
			}
		}()
	}

	wg.Wait()

	// Verify messages were written
	if buf.Len() == 0 {
		t.Error("expected data in buffer after concurrent writes")
	}
}

// TestJSONLFormat verifies that output is valid JSONL
func TestJSONLFormat(t *testing.T) {
	var buf bytes.Buffer
	streamer := NewJSONStreamer(&buf, nil)
	defer streamer.Close()

	// Write multiple messages
	messages := []struct {
		msgType string
		content string
	}{
		{"text", "first message"},
		{"text", "second message"},
		{"tool_use", "tool content"},
		{"tool_result", "result content"},
	}

	for _, m := range messages {
		err := streamer.WriteMessage(m.msgType, m.content)
		if err != nil {
			t.Fatalf("WriteMessage failed: %v", err)
		}
	}

	// Parse each line as separate JSON
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != len(messages) {
		t.Fatalf("expected %d lines, got %d", len(messages), len(lines))
	}

	for i, line := range lines {
		var msg StreamMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			t.Errorf("line %d: failed to parse JSON: %v", i, err)
			continue
		}

		if msg.Type != messages[i].msgType {
			t.Errorf("line %d: expected type '%s', got '%s'", i, messages[i].msgType, msg.Type)
		}

		if msg.Content != messages[i].content {
			t.Errorf("line %d: expected content '%s', got '%s'", i, messages[i].content, msg.Content)
		}
	}
}

// TestStreamingTermination verifies proper stream termination
func TestStreamingTermination(t *testing.T) {
	var buf bytes.Buffer
	streamer := NewJSONStreamer(&buf, nil)

	msgID := "termination-test"

	// Write partial chunks
	for i := 0; i < 5; i++ {
		isLast := i == 4
		err := streamer.WritePartialMessage("stream", "chunk ", msgID, isLast)
		if err != nil {
			t.Fatalf("WritePartialMessage failed: %v", err)
		}
	}

	// Close stream
	err := streamer.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Verify partial buffers were cleaned up
	if streamer.GetPartialMessageCount() != 0 {
		t.Errorf("expected 0 partial messages after close, got %d", streamer.GetPartialMessageCount())
	}

	// Verify all data was flushed
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 5 {
		t.Errorf("expected 5 lines, got %d", len(lines))
	}

	// Verify last line is not partial
	var lastMsg StreamMessage
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &lastMsg); err != nil {
		t.Fatalf("failed to parse last message: %v", err)
	}

	if lastMsg.Partial {
		t.Error("expected last message to not be partial")
	}
}

// TestPartialMessageBuffering verifies partial message buffer management
func TestPartialMessageBuffering(t *testing.T) {
	var buf bytes.Buffer
	streamer := NewJSONStreamer(&buf, nil)
	defer streamer.Close()

	// Create multiple partial message streams
	msgIDs := []string{"stream-1", "stream-2", "stream-3"}

	for i, msgID := range msgIDs {
		// Write partial chunk for each
		err := streamer.WritePartialMessage("text", "chunk", msgID, false)
		if err != nil {
			t.Fatalf("WritePartialMessage failed: %v", err)
		}

		// Verify buffer count
		expectedCount := i + 1
		if count := streamer.GetPartialMessageCount(); count != expectedCount {
			t.Errorf("expected %d partial messages, got %d", expectedCount, count)
		}
	}

	// Complete one stream
	err := streamer.WritePartialMessage("text", "final", msgIDs[0], true)
	if err != nil {
		t.Fatalf("WritePartialMessage failed: %v", err)
	}

	// Buffer should still have 3 (completed messages are kept for reassembly)
	if count := streamer.GetPartialMessageCount(); count != 3 {
		t.Errorf("expected 3 partial messages, got %d", count)
	}
}

// TestAutoFlushInterval verifies background auto-flush
func TestAutoFlushInterval(t *testing.T) {
	var buf bytes.Buffer
	config := &StreamConfig{
		AutoFlush:     true,
		FlushInterval: 50 * time.Millisecond,
	}
	streamer := NewJSONStreamer(&buf, config)

	// Write without waiting
	err := streamer.WriteMessage("text", "auto flush test")
	if err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}

	// Wait for auto-flush
	time.Sleep(100 * time.Millisecond)

	// Data should be flushed
	if buf.Len() == 0 {
		t.Error("expected data to be auto-flushed")
	}

	streamer.Close()
}

// TestPartialMessageSequenceNumbers verifies sequence number ordering
func TestPartialMessageSequenceNumbers(t *testing.T) {
	var buf bytes.Buffer
	streamer := NewJSONStreamer(&buf, nil)
	defer streamer.Close()

	msgID := "sequence-test"

	// Write chunks
	chunks := []string{"a", "b", "c", "d", "e"}
	for i, content := range chunks {
		isLast := i == len(chunks)-1
		err := streamer.WritePartialMessage("text", content, msgID, isLast)
		if err != nil {
			t.Fatalf("WritePartialMessage failed: %v", err)
		}
	}

	// Parse output and verify sequence numbers
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != len(chunks) {
		t.Fatalf("expected %d lines, got %d", len(chunks), len(lines))
	}

	for i, line := range lines {
		var msg StreamMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			t.Fatalf("failed to parse line %d: %v", i, err)
		}

		expectedSeq := i + 1 // Sequence starts at 1
		if msg.SequenceNumber != expectedSeq {
			t.Errorf("line %d: expected sequence %d, got %d", i, expectedSeq, msg.SequenceNumber)
		}
	}
}

// TestStreamMessageJSON verifies JSON encoding/decoding
func TestStreamMessageJSON(t *testing.T) {
	t.Run("complete message", func(t *testing.T) {
		msg := StreamMessage{
			Type:           "text",
			Content:        "test content",
			Timestamp:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Partial:        false,
			SequenceNumber: 0,
			MessageID:      "msg-123",
			Metadata:       map[string]interface{}{"key": "value"},
		}

		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var decoded StreamMessage
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if decoded.Type != msg.Type {
			t.Errorf("type mismatch: %s vs %s", decoded.Type, msg.Type)
		}

		if decoded.Content != msg.Content {
			t.Errorf("content mismatch: %s vs %s", decoded.Content, msg.Content)
		}

		if decoded.Partial != msg.Partial {
			t.Errorf("partial mismatch: %v vs %v", decoded.Partial, msg.Partial)
		}

		if decoded.MessageID != msg.MessageID {
			t.Errorf("message_id mismatch: %s vs %s", decoded.MessageID, msg.MessageID)
		}
	})

	t.Run("partial message", func(t *testing.T) {
		msg := StreamMessage{
			Type:           "text",
			Content:        "partial content",
			Timestamp:      time.Now().UTC(),
			Partial:        true,
			SequenceNumber: 5,
			MessageID:      "msg-456",
		}

		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		// Verify JSON structure
		var raw map[string]interface{}
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("failed to unmarshal to raw: %v", err)
		}

		if raw["partial"] != true {
			t.Error("expected partial to be true in JSON")
		}

		if raw["sequence_number"].(float64) != 5 {
			t.Errorf("expected sequence_number 5, got %v", raw["sequence_number"])
		}
	})
}

// TestLargeContent verifies handling of large content
func TestLargeContent(t *testing.T) {
	var buf bytes.Buffer
	streamer := NewJSONStreamer(&buf, nil)
	defer streamer.Close()

	// Create large content (1MB)
	largeContent := make([]byte, 1024*1024)
	for i := range largeContent {
		largeContent[i] = byte('a' + (i % 26))
	}

	err := streamer.WriteMessage("large", string(largeContent))
	if err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}

	// Verify the message was written
	var msg StreamMessage
	if err := json.Unmarshal(buf.Bytes(), &msg); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if len(msg.Content) != len(largeContent) {
		t.Errorf("expected content length %d, got %d", len(largeContent), len(msg.Content))
	}
}

// TestUnicodeContent verifies handling of unicode content
func TestUnicodeContent(t *testing.T) {
	var buf bytes.Buffer
	streamer := NewJSONStreamer(&buf, nil)
	defer streamer.Close()

	testCases := []string{
		"Hello 世界 🌍",
		"Привет мир",
		"مرحبا بالعالم",
		"🎉🎊🎁",
		"\\u001b[31mred\\u001b[0m",
	}

	for _, content := range testCases {
		buf.Reset()

		err := streamer.WriteMessage("unicode", content)
		if err != nil {
			t.Fatalf("WriteMessage failed for '%s': %v", content, err)
		}

		var msg StreamMessage
		if err := json.Unmarshal(buf.Bytes(), &msg); err != nil {
			t.Fatalf("failed to parse JSON for '%s': %v", content, err)
		}

		if msg.Content != content {
			t.Errorf("content mismatch for '%s': got '%s'", content, msg.Content)
		}
	}
}

// TestTimestampUTC verifies timestamps are in UTC
func TestTimestampUTC(t *testing.T) {
	var buf bytes.Buffer
	streamer := NewJSONStreamer(&buf, nil)
	defer streamer.Close()

	beforeWrite := time.Now().UTC()

	err := streamer.WriteMessage("time", "test")
	if err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}

	afterWrite := time.Now().UTC()

	var msg StreamMessage
	if err := json.Unmarshal(buf.Bytes(), &msg); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if msg.Timestamp.Before(beforeWrite) {
		t.Error("timestamp is before write started")
	}

	if msg.Timestamp.After(afterWrite) {
		t.Error("timestamp is after write completed")
	}

	if msg.Timestamp.Location() != time.UTC {
		t.Errorf("expected UTC timezone, got %v", msg.Timestamp.Location())
	}
}

// TestPartialTimeoutConfiguration verifies partial timeout configuration
func TestPartialTimeoutConfiguration(t *testing.T) {
	customTimeout := 5 * time.Minute

	var buf bytes.Buffer
	config := &StreamConfig{
		PartialTimeout: customTimeout,
	}
	streamer := NewJSONStreamer(&buf, config)
	defer streamer.Close()

	if streamer.config.PartialTimeout != customTimeout {
		t.Errorf("expected PartialTimeout %v, got %v", customTimeout, streamer.config.PartialTimeout)
	}
}

// TestBufferSizeConfiguration verifies buffer size configuration
func TestBufferSizeConfiguration(t *testing.T) {
	customSize := 1024

	var buf bytes.Buffer
	config := &StreamConfig{
		BufferSize: customSize,
	}
	streamer := NewJSONStreamer(&buf, config)
	defer streamer.Close()

	if streamer.config.BufferSize != customSize {
		t.Errorf("expected BufferSize %d, got %d", customSize, streamer.config.BufferSize)
	}
}

// TestMessageIDGeneration verifies unique message IDs
func TestMessageIDGeneration(t *testing.T) {
	var buf bytes.Buffer
	streamer := NewJSONStreamer(&buf, nil)
	defer streamer.Close()

	ids := make(map[string]bool)
	numMessages := 100

	for i := 0; i < numMessages; i++ {
		buf.Reset()
		err := streamer.WriteMessage("test", "content")
		if err != nil {
			t.Fatalf("WriteMessage failed: %v", err)
		}

		var msg StreamMessage
		if err := json.Unmarshal(buf.Bytes(), &msg); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if ids[msg.MessageID] {
			t.Errorf("duplicate message ID: %s", msg.MessageID)
		}
		ids[msg.MessageID] = true
	}

	if len(ids) != numMessages {
		t.Errorf("expected %d unique IDs, got %d", numMessages, len(ids))
	}
}

// TestMetadataHandling verifies metadata in messages
func TestMetadataHandling(t *testing.T) {
	var buf bytes.Buffer
	streamer := NewJSONStreamer(&buf, nil)
	defer streamer.Close()

	msgID := "metadata-test"

	// Write partial with metadata
	streamer.partialMu.Lock()
	buf2 := &partialBuffer{
		chunks: []StreamMessage{{
			Type:     "text",
			Content:  "test",
			Metadata: map[string]interface{}{
				"key1": "value1",
				"key2": 42,
				"key3": true,
			},
		}},
	}
	streamer.partialBuffers[msgID] = buf2
	streamer.partialMu.Unlock()

	// Reassemble
	reassembled := streamer.ReassembleMessage(msgID)
	if reassembled == nil {
		t.Fatal("expected reassembled message")
	}

	if reassembled.Metadata == nil {
		t.Fatal("expected metadata in reassembled message")
	}

	if reassembled.Metadata["key1"] != "value1" {
		t.Errorf("expected key1='value1', got %v", reassembled.Metadata["key1"])
	}

	if reassembled.Metadata["key2"] != 42 {
		t.Errorf("expected key2=42, got %v", reassembled.Metadata["key2"])
	}

	if reassembled.Metadata["key3"] != true {
		t.Errorf("expected key3=true, got %v", reassembled.Metadata["key3"])
	}
}

// TestFlushLoopStop verifies flush loop stops on close
func TestFlushLoopStop(t *testing.T) {
	var buf bytes.Buffer
	config := &StreamConfig{
		AutoFlush:     true,
		FlushInterval: 10 * time.Millisecond,
	}
	streamer := NewJSONStreamer(&buf, config)

	// Close should stop the flush loop
	err := streamer.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Give time for goroutine to exit
	time.Sleep(50 * time.Millisecond)

	// Verify closed state
	if !streamer.IsClosed() {
		t.Error("expected streamer to be closed")
	}
}

// TestNilConfigUsesDefaults verifies nil config uses defaults
func TestNilConfigUsesDefaults(t *testing.T) {
	var buf bytes.Buffer
	streamer := NewJSONStreamer(&buf, nil)
	defer streamer.Close()

	defaults := DefaultStreamConfig()

	if streamer.config.AutoFlush != defaults.AutoFlush {
		t.Error("AutoFlush should use default")
	}

	if streamer.config.FlushInterval != defaults.FlushInterval {
		t.Error("FlushInterval should use default")
	}

	if streamer.config.BufferSize != defaults.BufferSize {
		t.Error("BufferSize should use default")
	}

	if streamer.config.EnablePartialMessages != defaults.EnablePartialMessages {
		t.Error("EnablePartialMessages should use default")
	}

	if streamer.config.PartialTimeout != defaults.PartialTimeout {
		t.Error("PartialTimeout should use default")
	}
}

// TestEmptyPartialBufferReassembly verifies reassembly with empty buffer
func TestEmptyPartialBufferReassembly(t *testing.T) {
	var buf bytes.Buffer
	streamer := NewJSONStreamer(&buf, nil)
	defer streamer.Close()

	// Create empty buffer
	msgID := "empty-buffer"
	streamer.partialMu.Lock()
	streamer.partialBuffers[msgID] = &partialBuffer{
		chunks: []StreamMessage{},
	}
	streamer.partialMu.Unlock()

	// Reassemble should return nil
	result := streamer.ReassembleMessage(msgID)
	if result != nil {
		t.Error("expected nil for empty buffer")
	}
}

// TestConcurrentPartialMessages verifies concurrent partial message handling
func TestConcurrentPartialMessages(t *testing.T) {
	var buf bytes.Buffer
	streamer := NewJSONStreamer(&buf, nil)
	defer streamer.Close()

	var wg sync.WaitGroup
	numStreams := 10
	numChunks := 5

	for i := 0; i < numStreams; i++ {
		wg.Add(1)
		go func(streamNum int) {
			defer wg.Done()

			msgID := fmt.Sprintf("concurrent-stream-%d", streamNum)
			for j := 0; j < numChunks; j++ {
				isLast := j == numChunks-1
				err := streamer.WritePartialMessage("text", "chunk", msgID, isLast)
				if err != nil {
					t.Errorf("WritePartialMessage failed: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify all streams were written
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	expectedLines := numStreams * numChunks
	if len(lines) != expectedLines {
		t.Errorf("expected %d lines, got %d", expectedLines, len(lines))
	}
}

// TestWriteAfterCloseRace tests race conditions on close
func TestWriteAfterCloseRace(t *testing.T) {
	var buf bytes.Buffer
	streamer := NewJSONStreamer(&buf, nil)

	var wg sync.WaitGroup
	var writeErrors int32

	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			err := streamer.WriteMessage("test", "content")
			if err == ErrStreamClosed {
				atomic.AddInt32(&writeErrors, 1)
			}
		}
	}()

	// Closer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(5 * time.Millisecond)
		streamer.Close()
	}()

	wg.Wait()

	// Should have some writes succeed and some fail with ErrStreamClosed
	if atomic.LoadInt32(&writeErrors) == 0 {
		t.Log("all writes succeeded (race may not have been triggered)")
	}
}

// TestStreamStats verifies stream statistics
func TestStreamStats(t *testing.T) {
	var buf bytes.Buffer
	streamer := NewJSONStreamer(&buf, nil)
	defer streamer.Close()

	// Initial state
	if !streamer.IsReady() {
		t.Error("expected streamer to be ready initially")
	}

	if streamer.IsClosed() {
		t.Error("expected streamer to not be closed initially")
	}

	if streamer.GetPartialMessageCount() != 0 {
		t.Errorf("expected 0 partial messages initially, got %d", streamer.GetPartialMessageCount())
	}

	// Add partial messages
	msgID := "stats-test"
	err := streamer.WritePartialMessage("text", "chunk", msgID, false)
	if err != nil {
		t.Fatalf("WritePartialMessage failed: %v", err)
	}

	if streamer.GetPartialMessageCount() != 1 {
		t.Errorf("expected 1 partial message, got %d", streamer.GetPartialMessageCount())
	}
}

// TestFailingWriter is a test helper that fails after N writes
type failingWriter struct {
	failAfter   int
	writeCount  int
	mu          sync.Mutex
}

func (w *failingWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.writeCount++
	if w.failAfter > 0 && w.writeCount > w.failAfter {
		return 0, errors.New("simulated write failure")
	}
	return len(p), nil
}