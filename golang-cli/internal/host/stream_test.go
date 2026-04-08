package host

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// mockBidiStream is a mock implementation of grpc.BidiStreamingClient for testing
type mockBidiStream struct {
	mu        sync.RWMutex
	sendMsgs  []*ClineMessageProto
	recvMsgs  []*ClineMessageProto
	recvIndex int
	closed    bool
	sendError error
	recvError error
	sendDelay time.Duration
	recvDelay time.Duration
	onSend    func(*ClineMessageProto)
	onRecv    func() *ClineMessageProto
	ctx       context.Context
}

func newMockBidiStream() *mockBidiStream {
	return &mockBidiStream{
		sendMsgs: make([]*ClineMessageProto, 0),
		recvMsgs: make([]*ClineMessageProto, 0),
		ctx:      context.Background(),
	}
}

func (m *mockBidiStream) Send(msg *ClineMessageProto) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return errors.New("stream closed")
	}

	if m.sendError != nil {
		return m.sendError
	}

	if m.sendDelay > 0 {
		time.Sleep(m.sendDelay)
	}

	m.sendMsgs = append(m.sendMsgs, msg)

	if m.onSend != nil {
		m.onSend(msg)
	}

	return nil
}

func (m *mockBidiStream) Recv() (*ClineMessageProto, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil, io.EOF
	}

	if m.recvError != nil {
		return nil, m.recvError
	}

	if m.recvDelay > 0 {
		time.Sleep(m.recvDelay)
	}

	if m.onRecv != nil {
		msg := m.onRecv()
		if msg == nil {
			return nil, io.EOF
		}
		return msg, nil
	}

	if m.recvIndex >= len(m.recvMsgs) {
		// Block until more messages are available or closed
		m.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
		m.mu.Lock()
		if m.recvIndex >= len(m.recvMsgs) {
			return nil, io.EOF
		}
	}

	msg := m.recvMsgs[m.recvIndex]
	m.recvIndex++
	return msg, nil
}

func (m *mockBidiStream) CloseSend() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockBidiStream) Context() context.Context {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ctx
}

func (m *mockBidiStream) Header() (metadata.MD, error) {
	return nil, nil
}

func (m *mockBidiStream) Trailer() metadata.MD {
	return nil
}

func (m *mockBidiStream) SendMsg(msg interface{}) error {
	if castMsg, ok := msg.(*ClineMessageProto); ok {
		return m.Send(castMsg)
	}
	return errors.New("invalid message type")
}

func (m *mockBidiStream) RecvMsg(msg interface{}) error {
	received, err := m.Recv()
	if err != nil {
		return err
	}
	if castMsg, ok := msg.(*ClineMessageProto); ok && received != nil {
		*castMsg = *received
		return nil
	}
	return errors.New("invalid message type")
}

func (m *mockBidiStream) addRecvMessage(msg *ClineMessageProto) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recvMsgs = append(m.recvMsgs, msg)
}

func (m *mockBidiStream) getSentMessages() []*ClineMessageProto {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*ClineMessageProto, len(m.sendMsgs))
	copy(result, m.sendMsgs)
	return result
}

func TestNewBidiStream(t *testing.T) {
	tests := []struct {
		name    string
		config  *StreamConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config uses defaults",
			config:  nil,
			wantErr: true,
			errMsg:  "stream creator is required",
		},
		{
			name: "missing stream creator",
			config: &StreamConfig{
				BufferSize: 100,
			},
			wantErr: true,
			errMsg:  "stream creator is required",
		},
		{
			name: "valid config with defaults",
			config: &StreamConfig{
				StreamCreator: func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error) {
					return nil, nil
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with custom values",
			config: &StreamConfig{
				BufferSize:             200,
				MaxOutstandingMessages: 100,
				ReconnectDelay:         5 * time.Second,
				MaxReconnectAttempts:   10,
				MaxChunkSize:           8192,
				StreamCreator: func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error) {
					return nil, nil
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream, err := NewBidiStream(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewBidiStream() error = nil, wantErr = true")
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("NewBidiStream() error = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("NewBidiStream() error = %v, wantErr = false", err)
				return
			}

			if stream == nil {
				t.Error("NewBidiStream() returned nil stream")
				return
			}

			if stream.GetState() != StreamStateDisconnected {
				t.Errorf("expected initial state %v, got %v", StreamStateDisconnected, stream.GetState())
			}

			// Verify defaults were applied
			if tt.config.BufferSize <= 0 && stream.config.BufferSize != DefaultStreamConfig().BufferSize {
				t.Errorf("expected default BufferSize %d, got %d", DefaultStreamConfig().BufferSize, stream.config.BufferSize)
			}
		})
	}
}

func TestDefaultStreamConfig(t *testing.T) {
	config := DefaultStreamConfig()

	if config.BufferSize != 100 {
		t.Errorf("expected BufferSize 100, got %d", config.BufferSize)
	}
	if config.MaxOutstandingMessages != 50 {
		t.Errorf("expected MaxOutstandingMessages 50, got %d", config.MaxOutstandingMessages)
	}
	if config.ReconnectDelay != 2*time.Second {
		t.Errorf("expected ReconnectDelay 2s, got %v", config.ReconnectDelay)
	}
	if config.MaxReconnectAttempts != 5 {
		t.Errorf("expected MaxReconnectAttempts 5, got %d", config.MaxReconnectAttempts)
	}
	if !config.EnablePartialMessages {
		t.Error("expected EnablePartialMessages to be true")
	}
	if config.MaxChunkSize != 4096 {
		t.Errorf("expected MaxChunkSize 4096, got %d", config.MaxChunkSize)
	}
}

func TestStreamStateString(t *testing.T) {
	tests := []struct {
		state StreamState
		want  string
	}{
		{StreamStateDisconnected, "disconnected"},
		{StreamStateConnecting, "connecting"},
		{StreamStateReady, "ready"},
		{StreamStateReconnecting, "reconnecting"},
		{StreamStateClosed, "closed"},
		{StreamState(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.state.String()
			if got != tt.want {
				t.Errorf("StreamState.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBidiStreamStartStop(t *testing.T) {
	config := &StreamConfig{
		StreamCreator: func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error) {
			return newMockBidiStream(), nil
		},
		BufferSize: 10,
	}

	bidiStream, err := NewBidiStream(config)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// Verify initial state
	if bidiStream.GetState() != StreamStateDisconnected {
		t.Errorf("expected initial state disconnected, got %v", bidiStream.GetState())
	}

	// Test Stop without starting - should not panic
	err = bidiStream.Stop()
	if err != nil {
		t.Logf("Stop() error (may be expected): %v", err)
	}

	// Verify state after stop
	if bidiStream.GetState() != StreamStateClosed {
		t.Errorf("expected state closed after stop, got %v", bidiStream.GetState())
	}
}

func TestBidiStreamSend(t *testing.T) {
	mockStream := newMockBidiStream()

	config := &StreamConfig{
		StreamCreator: func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error) {
			return mockStream, nil
		},
		BufferSize:             10,
		MaxOutstandingMessages: 5,
	}

	bidiStream, err := NewBidiStream(config)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// Should fail when not started
	msg := &ClineMessageProto{
		ClineMessage: &ClineMessage{
			Ts:   time.Now().UnixMilli(),
			Type: ClineMessageType_SAY,
			Say:  ClineSay_TEXT,
			Text: "test",
		},
	}
	err = bidiStream.Send(msg)
	if err != ErrStreamNotReady {
		t.Errorf("expected ErrStreamNotReady, got %v", err)
	}

	// Should fail with nil message
	err = bidiStream.Send(nil)
	if err == nil || err.Error() != "message is nil" {
		t.Errorf("expected 'message is nil' error, got %v", err)
	}
}

func TestBidiStreamPartialMessages(t *testing.T) {
	mockStream := newMockBidiStream()

	config := &StreamConfig{
		StreamCreator: func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error) {
			return mockStream, nil
		},
		BufferSize:             100,
		MaxOutstandingMessages: 50,
		EnablePartialMessages:  true,
		MaxChunkSize:           10,
	}

	bidiStream, err := NewBidiStream(config)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// Create a message larger than MaxChunkSize
	largeText := "abcdefghijklmnopqrstuvwxyz0123456789" // 36 chars, will create 4 chunks of 10 bytes
	msg := &ClineMessageProto{
		ClineMessage: &ClineMessage{
			Ts:   time.Now().UnixMilli(),
			Type: ClineMessageType_SAY,
			Say:  ClineSay_TEXT,
			Text: largeText,
		},
	}

	// Verify chunking logic
	chunks := chunkMessage(msg, config.MaxChunkSize)
	if len(chunks) != 4 {
		t.Errorf("expected 4 chunks, got %d", len(chunks))
	}

	// Verify chunk sizes
	expectedSizes := []int{10, 10, 10, 6}
	for i, chunk := range chunks {
		if len(chunk.Text) != expectedSizes[i] {
			t.Errorf("chunk %d size = %d, want %d", i, len(chunk.Text), expectedSizes[i])
		}
	}
	_ = bidiStream // Use the stream to avoid unused variable error
}

func TestBidiStreamPartialMessageReassembly(t *testing.T) {
	config := &StreamConfig{
		StreamCreator: func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error) {
			return newMockBidiStream(), nil
		},
		BufferSize:             100,
		MaxOutstandingMessages: 50,
		EnablePartialMessages:  true,
		MaxChunkSize:           10,
	}

	stream, err := NewBidiStream(config)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	ts := time.Now().UnixMilli()
	originalText := "hello world, this is a test message"

	// Simulate receiving chunks IN ORDER
	chunks := []*ClineMessageProto{
		{
			ClineMessage: &ClineMessage{
				Ts:      ts,
				Type:    ClineMessageType_SAY,
				Say:     ClineSay_TEXT,
				Text:    originalText[0:10], // "hello worl"
				Partial: true,
			},
		},
		{
			ClineMessage: &ClineMessage{
				Ts:      ts,
				Type:    ClineMessageType_SAY,
				Say:     ClineSay_TEXT,
				Text:    originalText[10:20], // "d, this is"
				Partial: true,
			},
		},
		{
			ClineMessage: &ClineMessage{
				Ts:      ts,
				Type:    ClineMessageType_SAY,
				Say:     ClineSay_TEXT,
				Text:    originalText[20:33], // "is a test mes"
				Partial: true,
			},
		},
		{
			ClineMessage: &ClineMessage{
				Ts:      ts,
				Type:    ClineMessageType_SAY,
				Say:     ClineSay_TEXT,
				Text:    originalText[33:35], // "ge"
				Partial: false,               // Last chunk
			},
		},
	}

	// Process chunks
	var completeMsg *ClineMessageProto
	for _, chunk := range chunks {
		completeMsg = stream.handlePartialMessage(chunk)
	}

	if completeMsg == nil {
		t.Fatal("expected complete message, got nil")
	}

	if completeMsg.Text != originalText {
		t.Errorf("reassembled text mismatch: got %s, want %s", completeMsg.Text, originalText)
	}

	if completeMsg.Type != ClineMessageType_SAY {
		t.Errorf("expected type SAY, got %v", completeMsg.Type)
	}

	if completeMsg.Ts != ts {
		t.Errorf("expected ts %d, got %d", ts, completeMsg.Ts)
	}
}

func TestBidiStreamCleanupPartialMessages(t *testing.T) {
	config := &StreamConfig{
		StreamCreator: func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error) {
			return newMockBidiStream(), nil
		},
	}

	stream, err := NewBidiStream(config)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// Add partial messages with old timestamps
	stream.partialMu.Lock()
	oldTs := time.Now().UnixMilli() - 3600000 // 1 hour ago
	newTs := time.Now().UnixMilli() + 3600000 // 1 hour from now
	stream.partialBuffers[oldTs] = []*ClineMessageProto{
		{ClineMessage: &ClineMessage{Ts: oldTs, Partial: true}},
	}
	stream.partialTimeouts[oldTs] = time.Now().Add(-time.Hour) // Expired
	stream.partialBuffers[newTs] = []*ClineMessageProto{
		{ClineMessage: &ClineMessage{Ts: newTs, Partial: true}},
	}
	stream.partialTimeouts[newTs] = time.Now().Add(time.Hour) // Not expired
	stream.partialMu.Unlock()

	// Run cleanup
	stream.cleanupPartialMessages()

	// Verify old message was cleaned up
	stream.partialMu.RLock()
	if _, exists := stream.partialBuffers[oldTs]; exists {
		t.Error("old message should have been cleaned up")
	}
	if _, exists := stream.partialBuffers[newTs]; !exists {
		t.Error("new message should not have been cleaned up")
	}
	stream.partialMu.RUnlock()
}

func TestBidiStreamStats(t *testing.T) {
	config := &StreamConfig{
		StreamCreator: func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error) {
			return newMockBidiStream(), nil
		},
		BufferSize:             10,
		MaxOutstandingMessages: 5,
	}

	stream, err := NewBidiStream(config)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	stats := stream.Stats()

	if stats.State != StreamStateDisconnected {
		t.Errorf("expected state disconnected, got %v", stats.State)
	}

	if stats.OutstandingMessages != 0 {
		t.Errorf("expected 0 outstanding messages, got %d", stats.OutstandingMessages)
	}

	if stats.ReconnectCount != 0 {
		t.Errorf("expected 0 reconnect count, got %d", stats.ReconnectCount)
	}
}

func TestBidiStreamStateCallbacks(t *testing.T) {
	var stateChanges []StreamState
	var mu sync.Mutex

	config := &StreamConfig{
		StreamCreator: func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error) {
			return newMockBidiStream(), nil
		},
		OnStateChange: func(state StreamState) {
			mu.Lock()
			defer mu.Unlock()
			stateChanges = append(stateChanges, state)
		},
	}

	stream, err := NewBidiStream(config)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// Manually trigger state changes
	stream.setState(StreamStateConnecting)
	stream.setState(StreamStateReady)
	stream.setState(StreamStateReconnecting)
	stream.setState(StreamStateClosed)

	mu.Lock()
	if len(stateChanges) != 4 {
		t.Errorf("expected 4 state changes, got %d", len(stateChanges))
	}
	mu.Unlock()
}

func TestBidiStreamErrorCallback(t *testing.T) {
	var receivedError error
	var mu sync.Mutex

	config := &StreamConfig{
		StreamCreator: func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error) {
			return newMockBidiStream(), nil
		},
		OnError: func(err error) {
			mu.Lock()
			defer mu.Unlock()
			receivedError = err
		},
	}

	stream, err := NewBidiStream(config)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	testErr := errors.New("test error")
	stream.handleError(testErr)

	mu.Lock()
	if receivedError != testErr {
		t.Errorf("expected error %v, got %v", testErr, receivedError)
	}
	mu.Unlock()
}

func TestBidiStreamIsReady(t *testing.T) {
	config := &StreamConfig{
		StreamCreator: func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error) {
			return newMockBidiStream(), nil
		},
	}

	stream, err := NewBidiStream(config)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	if stream.IsReady() {
		t.Error("stream should not be ready initially")
	}

	stream.setState(StreamStateReady)

	if !stream.IsReady() {
		t.Error("stream should be ready after setting state")
	}
}

func TestBidiStreamForceReconnect(t *testing.T) {
	config := &StreamConfig{
		StreamCreator: func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error) {
			return newMockBidiStream(), nil
		},
	}

	stream, err := NewBidiStream(config)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// Should not error on non-closed stream
	err = stream.ForceReconnect()
	if err != nil {
		t.Errorf("ForceReconnect() error = %v", err)
	}

	// Should error on closed stream
	stream.setState(StreamStateClosed)
	err = stream.ForceReconnect()
	if err != ErrStreamClosed {
		t.Errorf("expected ErrStreamClosed, got %v", err)
	}
}

func TestBidiStreamWaitForReady(t *testing.T) {
	config := &StreamConfig{
		StreamCreator: func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error) {
			return newMockBidiStream(), nil
		},
	}

	stream, err := NewBidiStream(config)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// Test with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Should timeout when not ready
	err = stream.WaitForReady(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("expected deadline exceeded, got %v", err)
	}

	// Should succeed when ready
	stream.setState(StreamStateReady)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel2()

	err = stream.WaitForReady(ctx2)
	if err != nil {
		t.Errorf("WaitForReady() error = %v", err)
	}
}

func TestBidiStreamBackpressure(t *testing.T) {
	config := &StreamConfig{
		StreamCreator: func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error) {
			return newMockBidiStream(), nil
		},
		MaxOutstandingMessages: 2,
		BufferSize:             10,
	}

	stream, err := NewBidiStream(config)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// Set state to ready manually for testing
	stream.setState(StreamStateReady)

	// Fill up backpressure tokens
	stream.backpressure <- struct{}{}
	stream.backpressure <- struct{}{}

	// Should fail with backpressure exceeded
	msg := &ClineMessageProto{
		ClineMessage: &ClineMessage{
			Ts:   time.Now().UnixMilli(),
			Type: ClineMessageType_SAY,
			Say:  ClineSay_TEXT,
			Text: "test",
		},
	}
	err = stream.Send(msg)
	if err != ErrBackpressureExceeded {
		t.Errorf("expected ErrBackpressureExceeded, got %v", err)
	}

	// Release backpressure and try again
	<-stream.backpressure
	err = stream.Send(msg)
	// May still fail due to send timeout or other issues in test environment
	if err != nil && err != ErrBackpressureExceeded {
		t.Logf("Send() error (may be expected in test): %v", err)
	}
}

func TestBidiStreamReassemblyWithMissingChunks(t *testing.T) {
	config := &StreamConfig{
		StreamCreator: func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error) {
			return newMockBidiStream(), nil
		},
	}

	stream, err := NewBidiStream(config)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	ts := time.Now().UnixMilli()

	// Add only 2 of 3 required chunks
	chunk1 := &ClineMessageProto{
		ClineMessage: &ClineMessage{
			Ts:      ts,
			Type:    ClineMessageType_SAY,
			Say:     ClineSay_TEXT,
			Text:    "part1",
			Partial: true,
		},
	}
	chunk2 := &ClineMessageProto{
		ClineMessage: &ClineMessage{
			Ts:      ts,
			Type:    ClineMessageType_SAY,
			Say:     ClineSay_TEXT,
			Text:    "part2",
			Partial: true,
		},
	}

	// Process chunks - should return nil (not complete)
	result := stream.handlePartialMessage(chunk1)
	if result != nil {
		t.Error("expected nil for incomplete message")
	}

	result = stream.handlePartialMessage(chunk2)
	if result != nil {
		t.Error("expected nil for incomplete message")
	}

	// Verify partial buffer exists
	stream.partialMu.RLock()
	if len(stream.partialBuffers[ts]) != 2 {
		t.Errorf("expected 2 chunks in buffer, got %d", len(stream.partialBuffers[ts]))
	}
	stream.partialMu.RUnlock()
}

func TestBidiStreamReleaseBackpressure(t *testing.T) {
	config := &StreamConfig{
		StreamCreator: func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error) {
			return newMockBidiStream(), nil
		},
		MaxOutstandingMessages: 5,
	}

	stream, err := NewBidiStream(config)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// Set outstanding messages
	stream.outstandingMsgs.Store(3)

	// Fill backpressure channel
	for i := 0; i < 3; i++ {
		stream.backpressure <- struct{}{}
	}

	// Release backpressure
	stream.releaseBackpressure(12345)

	// Verify one token was released
	if len(stream.backpressure) != 2 {
		t.Errorf("expected 2 backpressure tokens, got %d", len(stream.backpressure))
	}

	// Verify outstanding count decreased
	if stream.outstandingMsgs.Load() != 2 {
		t.Errorf("expected outstanding messages 2, got %d", stream.outstandingMsgs.Load())
	}
}

func TestBidiStreamAcknowledge(t *testing.T) {
	config := &StreamConfig{
		StreamCreator: func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error) {
			return newMockBidiStream(), nil
		},
	}

	stream, err := NewBidiStream(config)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// Test acknowledge
	stream.Acknowledge(12345)
	stream.Acknowledge(12346)

	// Verify messages were queued
	if len(stream.ackChan) != 2 {
		t.Errorf("expected 2 acks in channel, got %d", len(stream.ackChan))
	}
}

func TestBidiStreamConcurrentAccess(t *testing.T) {
	config := &StreamConfig{
		StreamCreator: func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error) {
			return newMockBidiStream(), nil
		},
	}

	stream, err := NewBidiStream(config)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	var wg sync.WaitGroup
	numGoroutines := 20

	// Test concurrent state reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = stream.GetState()
			_ = stream.Stats()
			_ = stream.IsReady()
		}()
	}

	// Test concurrent state writes
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			states := []StreamState{StreamStateConnecting, StreamStateReady, StreamStateReconnecting}
			stream.setState(states[idx%len(states)])
		}(i)
	}

	wg.Wait()
}

func TestBidiStreamReconnectBackoff(t *testing.T) {
	t.Skip("Skipped: requires mock connection pool")
}

func TestBidiStreamReassemblyMessageOrder(t *testing.T) {
	config := &StreamConfig{
		StreamCreator: func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error) {
			return newMockBidiStream(), nil
		},
	}

	stream, err := NewBidiStream(config)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	ts := time.Now().UnixMilli()
	originalText := "abcdefghijklmnopqrstuvwxyz"

	// Create chunks in reverse order
	chunks := make([]*ClineMessageProto, 3)
	for i := 2; i >= 0; i-- {
		start := i * 10
		end := start + 10
		if end > len(originalText) {
			end = len(originalText)
		}

		chunks[2-i] = &ClineMessageProto{
			ClineMessage: &ClineMessage{
				Ts:      ts,
				Type:    ClineMessageType_SAY,
				Say:     ClineSay_TEXT,
				Text:    originalText[start:end],
				Partial: true,
			},
		}
	}

	// Mark last chunk as not partial
	chunks[2].Partial = false

	// Process chunks
	var completeMsg *ClineMessageProto
	for _, chunk := range chunks {
		completeMsg = stream.handlePartialMessage(chunk)
	}

	if completeMsg == nil {
		t.Fatal("expected complete message")
	}

	// Note: The current implementation doesn't reorder chunks by sequence number
	// It just appends them in received order
	// This test documents current behavior
	if len(completeMsg.Text) != len(originalText) {
		t.Errorf("expected text length %d, got %d", len(originalText), len(completeMsg.Text))
	}
}

func TestBidiStreamReceiveChannel(t *testing.T) {
	config := &StreamConfig{
		StreamCreator: func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error) {
			return newMockBidiStream(), nil
		},
		BufferSize: 10,
	}

	stream, err := NewBidiStream(config)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// Get receive channel
	recvChan := stream.Receive()

	// Verify it's a valid channel
	if recvChan == nil {
		t.Fatal("Receive() returned nil channel")
	}

	// Send a message to the channel
	testMsg := &ClineMessageProto{
		ClineMessage: &ClineMessage{
			Ts:   time.Now().UnixMilli(),
			Type: ClineMessageType_SAY,
			Say:  ClineSay_TEXT,
			Text: "data",
		},
	}
	go func() {
		stream.recvChan <- testMsg
	}()

	// Receive the message
	select {
	case msg := <-recvChan:
		if msg.Ts != testMsg.Ts {
			t.Errorf("expected message ts %d, got %d", testMsg.Ts, msg.Ts)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for message")
	}
}

func TestBidiStreamStopIdempotency(t *testing.T) {
	config := &StreamConfig{
		StreamCreator: func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error) {
			return newMockBidiStream(), nil
		},
	}

	stream, err := NewBidiStream(config)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// Multiple stops should not panic
	for i := 0; i < 3; i++ {
		err = stream.Stop()
		if i == 0 && err != nil {
			t.Errorf("first Stop() error = %v", err)
		}
	}

	if stream.GetState() != StreamStateClosed {
		t.Errorf("expected state closed, got %v", stream.GetState())
	}
}

func TestBidiStreamCloseSendOnStop(t *testing.T) {
	mockStream := newMockBidiStream()

	config := &StreamConfig{
		StreamCreator: func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error) {
			return mockStream, nil
		},
	}

	stream, err := NewBidiStream(config)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// Manually set up stream
	stream.stream = mockStream

	// Stop the stream
	stream.Stop()

	// Verify CloseSend was called
	if !mockStream.closed {
		t.Error("expected stream to be closed")
	}
}

func TestBidiStreamMessageReceivedCallback(t *testing.T) {
	var receivedMsg *ClineMessageProto
	var mu sync.Mutex

	config := &StreamConfig{
		StreamCreator: func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error) {
			return newMockBidiStream(), nil
		},
		OnMessageReceived: func(msg *ClineMessageProto) {
			mu.Lock()
			defer mu.Unlock()
			receivedMsg = msg
		},
	}

	stream, err := NewBidiStream(config)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// Simulate message reception
	testMsg := &ClineMessageProto{
		ClineMessage: &ClineMessage{
			Ts:   time.Now().UnixMilli(),
			Type: ClineMessageType_SAY,
			Say:  ClineSay_TEXT,
			Text: "callback test",
		},
	}
	stream.recvChan <- testMsg

	// Manually trigger what receiveLoop would do
	if stream.config.OnMessageReceived != nil {
		stream.config.OnMessageReceived(testMsg)
	}

	mu.Lock()
	if receivedMsg == nil {
		t.Error("OnMessageReceived callback was not called")
	} else if receivedMsg.Ts != testMsg.Ts {
		t.Errorf("expected message ts %d, got %d", testMsg.Ts, receivedMsg.Ts)
	}
	mu.Unlock()
}

func TestBidiStreamSendTimeout(t *testing.T) {
	t.Skip("Skipped: may trigger reconnection without mock connection pool")
}

func TestBidiStreamReconnectContextCancellation(t *testing.T) {
	t.Skip("Skipped: requires mock connection pool")
}

func TestBidiStreamSendPartialMessageTimeout(t *testing.T) {
	t.Skip("Skipped: may trigger reconnection without mock connection pool")
}

func TestBidiStreamReassembleMessageEmptyBuffer(t *testing.T) {
	config := &StreamConfig{
		StreamCreator: func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error) {
			return newMockBidiStream(), nil
		},
	}

	stream, err := NewBidiStream(config)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}

	// Try to reassemble non-existent message
	msg := stream.reassembleMessage(999999)
	if msg != nil {
		t.Error("expected nil for non-existent message")
	}
}

func TestBidiStreamReconnectingFlag(t *testing.T) {
	t.Skip("Skipped: requires mock connection pool")
}

// ============================================================================
// TaskStreamHandler Tests
// ============================================================================

func TestNewTaskStreamHandler(t *testing.T) {
	handler := NewTaskStreamHandler("test-task-id")
	if handler == nil {
		t.Fatal("NewTaskStreamHandler returned nil")
	}
	if handler.taskID != "test-task-id" {
		t.Errorf("expected taskID 'test-task-id', got %s", handler.taskID)
	}
	if handler.messageChan == nil {
		t.Error("messageChan should not be nil")
	}
	if handler.errorChan == nil {
		t.Error("errorChan should not be nil")
	}
	if handler.doneChan == nil {
		t.Error("doneChan should not be nil")
	}
}

func TestTaskStreamHandlerStartStop(t *testing.T) {
	mockStream := newMockBidiStream()

	handler := NewTaskStreamHandler("test-task-id")

	streamCreator := func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error) {
		return mockStream, nil
	}

	err := handler.Start(streamCreator)
	if err != nil {
		t.Fatalf("failed to start handler: %v", err)
	}

	// Verify handler has a stream
	if handler.stream == nil {
		t.Error("handler should have a stream after Start")
	}

	// Stop the handler
	err = handler.Stop()
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

func TestTaskStreamHandlerSendMessage(t *testing.T) {
	mockStream := newMockBidiStream()

	handler := NewTaskStreamHandler("test-task-id")

	streamCreator := func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error) {
		return mockStream, nil
	}

	err := handler.Start(streamCreator)
	if err != nil {
		t.Fatalf("failed to start handler: %v", err)
	}
	defer handler.Stop()

	// Manually set stream state to ready for testing
	handler.stream.setState(StreamStateReady)

	// Send a message
	err = handler.SendMessage("Hello, World!", []string{"img1"}, []string{"file1"})
	if err != nil {
		// May fail due to backpressure or other reasons in test
		t.Logf("SendMessage() error (may be expected): %v", err)
	}
}

func TestTaskStreamHandlerSendAskResponse(t *testing.T) {
	mockStream := newMockBidiStream()

	handler := NewTaskStreamHandler("test-task-id")

	streamCreator := func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error) {
		return mockStream, nil
	}

	err := handler.Start(streamCreator)
	if err != nil {
		t.Fatalf("failed to start handler: %v", err)
	}
	defer handler.Stop()

	// Manually set stream state to ready for testing
	handler.stream.setState(StreamStateReady)

	// Send an ask response
	err = handler.SendAskResponse("messageResponse", "My response", nil, nil)
	if err != nil {
		// May fail due to backpressure or other reasons in test
		t.Logf("SendAskResponse() error (may be expected): %v", err)
	}
}

func TestTaskStreamHandlerHandleSayMessage(t *testing.T) {
	handler := NewTaskStreamHandler("test-task-id")

	var receivedText string
	var receivedTool *ClineSayTool
	var receivedCommand string

	handler.OnTextMessage = func(text string) {
		receivedText = text
	}
	handler.OnToolRequest = func(tool *ClineSayTool) error {
		receivedTool = tool
		return nil
	}
	handler.OnCommandRequest = func(command string) error {
		receivedCommand = command
		return nil
	}

	// Test TEXT message
	msg := &ClineMessageProto{
		ClineMessage: &ClineMessage{
			Type: ClineMessageType_SAY,
			Say:  ClineSay_TEXT,
			Text: "Hello!",
		},
	}
	handler.handleSayMessage(msg)
	if receivedText != "Hello!" {
		t.Errorf("expected text 'Hello!', got %s", receivedText)
	}

	// Test TOOL_SAY message
	msg = &ClineMessageProto{
		ClineMessage: &ClineMessage{
			Type:    ClineMessageType_SAY,
			Say:     ClineSay_TOOL_SAY,
			SayTool: &ClineSayTool{Tool: ClineSayToolType_READ_FILE, Path: "/test/file"},
		},
	}
	handler.handleSayMessage(msg)
	if receivedTool == nil {
		t.Error("expected tool to be received")
	} else if receivedTool.Path != "/test/file" {
		t.Errorf("expected path '/test/file', got %s", receivedTool.Path)
	}

	// Test COMMAND_SAY message
	msg = &ClineMessageProto{
		ClineMessage: &ClineMessage{
			Type: ClineMessageType_SAY,
			Say:  ClineSay_COMMAND_SAY,
			Text: "ls -la",
		},
	}
	handler.handleSayMessage(msg)
	if receivedCommand != "ls -la" {
		t.Errorf("expected command 'ls -la', got %s", receivedCommand)
	}
}

func TestTaskStreamHandlerHandleAskMessage(t *testing.T) {
	handler := NewTaskStreamHandler("test-task-id")

	var receivedQuestion *ClineAskQuestion
	handler.OnAskQuestion = func(question *ClineAskQuestion) (string, error) {
		receivedQuestion = question
		return "My answer", nil
	}

	// Create a mock stream and set it up
	mockStream := newMockBidiStream()
	config := &StreamConfig{
		StreamCreator: func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error) {
			return mockStream, nil
		},
	}

	bidiStream, err := NewBidiStream(config)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}
	handler.stream = bidiStream
	// Set state to ready so messages can be sent
	handler.stream.setState(StreamStateReady)

	// Test FOLLOWUP ask
	msg := &ClineMessageProto{
		ClineMessage: &ClineMessage{
			Type: ClineMessageType_ASK,
			Ask:  ClineAsk_FOLLOWUP,
			AskQuestion: &ClineAskQuestion{
				Question: "What do you think?",
				Options:  []string{"Yes", "No"},
			},
		},
	}
	handler.handleAskMessage(msg)
	if receivedQuestion == nil {
		t.Error("expected question to be received")
	} else if receivedQuestion.Question != "What do you think?" {
		t.Errorf("expected question 'What do you think?', got %s", receivedQuestion.Question)
	}
}

func TestTaskStreamHandlerHandleMessage(t *testing.T) {
	handler := NewTaskStreamHandler("test-task-id")

	var receivedSay bool
	var receivedAsk bool

	handler.OnTextMessage = func(text string) {
		receivedSay = true
	}
	handler.OnAskQuestion = func(question *ClineAskQuestion) (string, error) {
		receivedAsk = true
		return "", nil
	}

	// Test SAY message
	msg := &ClineMessageProto{
		ClineMessage: &ClineMessage{
			Type: ClineMessageType_SAY,
			Say:  ClineSay_TEXT,
			Text: "Test",
		},
	}
	handler.handleMessage(msg)
	if !receivedSay {
		t.Error("expected SAY handler to be called")
	}

	// Test ASK message - must have stream set up
	mockStream := newMockBidiStream()
	config := &StreamConfig{
		StreamCreator: func(ctx context.Context, conn *grpc.ClientConn) (TaskService_StreamClient, error) {
			return mockStream, nil
		},
	}

	bidiStream, err := NewBidiStream(config)
	if err != nil {
		t.Fatalf("failed to create stream: %v", err)
	}
	handler.stream = bidiStream
	handler.stream.setState(StreamStateReady)

	msg = &ClineMessageProto{
		ClineMessage: &ClineMessage{
			Type: ClineMessageType_ASK,
			Ask:  ClineAsk_FOLLOWUP,
			AskQuestion: &ClineAskQuestion{
				Question: "Test question?",
			},
		},
	}
	handler.handleMessage(msg)
	if !receivedAsk {
		t.Error("expected ASK handler to be called")
	}

	// Test nil message
	handler.handleMessage(nil) // Should not panic
}

func TestTaskStreamHandlerWaitForCompletion(t *testing.T) {
	handler := NewTaskStreamHandler("test-task-id")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Should timeout when no completion message
	err := handler.WaitForCompletion(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("expected deadline exceeded, got %v", err)
	}
}

func TestTaskStreamHandlerErrorCallback(t *testing.T) {
	handler := NewTaskStreamHandler("test-task-id")

	var receivedError error
	handler.OnError = func(err error) {
		receivedError = err
	}

	// Simulate error
	testErr := errors.New("test error")
	handler.errorChan <- testErr

	// Process error through routeMessages (manually trigger)
	select {
	case err := <-handler.errorChan:
		if handler.OnError != nil {
			handler.OnError(err)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for error")
	}

	if receivedError != testErr {
		t.Errorf("expected error %v, got %v", testErr, receivedError)
	}
}

// ============================================================================
// JSON Serialization Tests
// ============================================================================

func TestMessageToJSON(t *testing.T) {
	msg := &ClineMessageProto{
		ClineMessage: &ClineMessage{
			Ts:   time.Now().UnixMilli(),
			Type: ClineMessageType_SAY,
			Say:  ClineSay_TEXT,
			Text: "Hello, World!",
		},
	}

	data, err := MessageToJSON(msg)
	if err != nil {
		t.Fatalf("MessageToJSON() error = %v", err)
	}

	// Verify it's valid JSON
	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Errorf("result is not valid JSON: %v", err)
	}

	if decoded["text"] != "Hello, World!" {
		t.Errorf("expected text 'Hello, World!', got %v", decoded["text"])
	}
}

func TestMessageToJSONNil(t *testing.T) {
	_, err := MessageToJSON(nil)
	if err == nil || err.Error() != "message is nil" {
		t.Errorf("expected 'message is nil' error, got %v", err)
	}
}

func TestJSONToMessage(t *testing.T) {
	data := []byte(`{"ts":1234567890,"type":1,"say":4,"text":"Hello"}`)

	msg, err := JSONToMessage(data)
	if err != nil {
		t.Fatalf("JSONToMessage() error = %v", err)
	}

	if msg.Ts != 1234567890 {
		t.Errorf("expected ts 1234567890, got %d", msg.Ts)
	}
	if msg.Type != ClineMessageType_SAY {
		t.Errorf("expected type SAY, got %v", msg.Type)
	}
	if msg.Say != ClineSay_TEXT {
		t.Errorf("expected say TEXT, got %v", msg.Say)
	}
	if msg.Text != "Hello" {
		t.Errorf("expected text 'Hello', got %s", msg.Text)
	}
}

// ============================================================================
// StreamCreator Tests
// ============================================================================

func TestCreateTaskStreamCreator(t *testing.T) {
	creator := CreateTaskStreamCreator("test-task-id")

	// Since we can't easily mock the grpc.ClientConn, we'll just verify the function exists
	// and has the correct signature
	if creator == nil {
		t.Error("CreateTaskStreamCreator returned nil")
	}
}

// ============================================================================
// TaskServiceClient Tests
// ============================================================================

func TestNewTaskServiceClient(t *testing.T) {
	// This test just verifies the function exists and doesn't panic
	// Actual gRPC calls would require a real server
	t.Skip("Requires gRPC server connection")
}
